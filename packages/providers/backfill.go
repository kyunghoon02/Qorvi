package providers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

type ProviderActivityFixtureInput struct {
	Provider      ProviderName
	Chain         domain.Chain
	WalletAddress string
	SourceID      string
	Kind          string
	Confidence    float64
	ObservedAt    time.Time
	Metadata      map[string]any
}

type HistoricalBackfillBatch struct {
	Provider    ProviderName
	Request     ProviderRequestContext
	WindowStart time.Time
	WindowEnd   time.Time
	Limit       int
}

type HistoricalBackfillAdapter interface {
	ProviderAdapter
	FetchHistoricalWalletActivity(batch HistoricalBackfillBatch) ([]ProviderWalletActivity, error)
}

type HistoricalBackfillResult struct {
	Batch      HistoricalBackfillBatch
	Activities []ProviderWalletActivity
}

type HistoricalBackfillRunner struct {
	Registry Registry
}

func NewHistoricalBackfillRunner(registry Registry) HistoricalBackfillRunner {
	return HistoricalBackfillRunner{Registry: registry}
}

func (b HistoricalBackfillBatch) Validate() error {
	if b.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if !domain.IsSupportedChain(b.Request.Chain) {
		return fmt.Errorf("unsupported chain %q", b.Request.Chain)
	}
	if b.Request.WalletAddress == "" {
		return fmt.Errorf("wallet address is required")
	}
	if b.WindowStart.IsZero() || b.WindowEnd.IsZero() {
		return fmt.Errorf("window start and end are required")
	}
	if b.WindowEnd.Before(b.WindowStart) {
		return fmt.Errorf("window end must be after window start")
	}
	if b.Limit <= 0 {
		return fmt.Errorf("limit must be positive")
	}

	return nil
}

func (r HistoricalBackfillRunner) Run(batch HistoricalBackfillBatch) (HistoricalBackfillResult, error) {
	if err := batch.Validate(); err != nil {
		return HistoricalBackfillResult{}, err
	}

	adapter, ok := r.Registry[batch.Provider]
	if !ok {
		return HistoricalBackfillResult{}, fmt.Errorf("provider %q is not registered", batch.Provider)
	}

	historicalAdapter, ok := adapter.(HistoricalBackfillAdapter)
	if !ok {
		return HistoricalBackfillResult{}, fmt.Errorf("provider %q does not support historical backfill", batch.Provider)
	}

	activities, err := historicalAdapter.FetchHistoricalWalletActivity(batch)
	if err != nil {
		if fallbackResult, handled, fallbackErr := r.runHeliusHistoricalFallback(batch, err); handled {
			if fallbackErr != nil {
				return HistoricalBackfillResult{}, fallbackErr
			}
			return fallbackResult, nil
		}
		return HistoricalBackfillResult{}, err
	}

	return HistoricalBackfillResult{
		Batch:      batch,
		Activities: append([]ProviderWalletActivity(nil), activities...),
	}, nil
}

func (r HistoricalBackfillRunner) runHeliusHistoricalFallback(
	batch HistoricalBackfillBatch,
	cause error,
) (HistoricalBackfillResult, bool, error) {
	if batch.Provider != ProviderHelius || batch.Request.Chain != domain.ChainSolana {
		return HistoricalBackfillResult{}, false, nil
	}
	if !shouldFallbackHeliusHistorical(cause) {
		return HistoricalBackfillResult{}, false, nil
	}

	adapter, ok := r.Registry[ProviderAlchemy]
	if !ok {
		return HistoricalBackfillResult{}, true, fmt.Errorf("helius paid-plan historical runner fallback unavailable: alchemy adapter not registered after %w", cause)
	}

	historicalAdapter, ok := adapter.(HistoricalBackfillAdapter)
	if !ok {
		return HistoricalBackfillResult{}, true, fmt.Errorf("helius paid-plan historical runner fallback unavailable: alchemy adapter has no historical contract after %w", cause)
	}

	fallbackBatch := batch
	fallbackBatch.Provider = ProviderAlchemy

	activities, err := historicalAdapter.FetchHistoricalWalletActivity(fallbackBatch)
	if err != nil {
		return HistoricalBackfillResult{}, true, fmt.Errorf("helius paid-plan historical runner fallback to alchemy failed: %w", err)
	}

	return HistoricalBackfillResult{
		Batch:      fallbackBatch,
		Activities: append([]ProviderWalletActivity(nil), activities...),
	}, true, nil
}

func CreateHistoricalBackfillBatchFixture(provider ProviderName, chain domain.Chain, walletAddress string) HistoricalBackfillBatch {
	return HistoricalBackfillBatch{
		Provider: provider,
		Request: ProviderRequestContext{
			Chain:         chain,
			WalletAddress: walletAddress,
			Access: domain.AccessContext{
				Role: domain.RoleOperator,
				Plan: domain.PlanPro,
			},
		},
		WindowStart: fixedObservedAt.Add(-24 * time.Hour),
		WindowEnd:   fixedObservedAt,
		Limit:       250,
	}
}

func createHistoricalBackfillActivityFixture(batch HistoricalBackfillBatch, input ProviderActivityFixtureInput) ProviderWalletActivity {
	metadata := map[string]any{}
	for key, value := range input.Metadata {
		metadata[key] = value
	}
	metadata["backfill_window_start"] = batch.WindowStart.Format(time.RFC3339)
	metadata["backfill_window_end"] = batch.WindowEnd.Format(time.RFC3339)
	metadata["backfill_limit"] = batch.Limit

	input.Metadata = metadata
	if input.ObservedAt.IsZero() {
		input.ObservedAt = fixedObservedAt
	}

	return CreateProviderActivityFixture(input)
}

func NormalizeProviderActivity(activity ProviderWalletActivity) (domain.NormalizedTransaction, error) {
	tx := domain.NormalizeNormalizedTransaction(domain.NormalizedTransaction{
		Chain: activity.Chain,
		TxHash: metadataStringOrDefault(
			activity.Metadata,
			"tx_hash",
			fmt.Sprintf(
				"%s:%s:%s:%s",
				activity.Provider,
				activity.SourceID,
				strings.TrimSpace(activity.WalletAddress),
				activity.ObservedAt.UTC().Format(time.RFC3339Nano),
			),
		),
		Wallet: domain.WalletRef{
			Chain:   activity.Chain,
			Address: activity.WalletAddress,
		},
		Direction:        normalizeTransactionDirection(metadataStringOrDefault(activity.Metadata, "direction", string(domain.TransactionDirectionUnknown))),
		Amount:           metadataStringOrDefault(activity.Metadata, "amount", ""),
		ObservedAt:       activity.ObservedAt,
		BlockNumber:      metadataInt64OrDefault(activity.Metadata, "block_number", 0),
		TransactionIndex: metadataInt64OrDefault(activity.Metadata, "transaction_index", 0),
		SchemaVersion:    metadataIntOrDefault(activity.Metadata, "schema_version", 1),
		RawPayloadPath: metadataStringOrDefault(
			activity.Metadata,
			"raw_payload_path",
			fmt.Sprintf("s3://qorvi/raw/%s/%s/%s.json", activity.Provider, activity.Chain, sanitizeSourceID(activity.SourceID)),
		),
		Provider: string(activity.Provider),
	})

	if counterparty := metadataStringOrDefault(activity.Metadata, "counterparty_address", ""); counterparty != "" {
		tx.Counterparty = &domain.WalletRef{
			Chain:   domain.Chain(metadataStringOrDefault(activity.Metadata, "counterparty_chain", string(activity.Chain))),
			Address: counterparty,
		}
	}

	if tokenAddress := metadataStringOrDefault(activity.Metadata, "token_address", ""); tokenAddress != "" {
		tx.Token = &domain.TokenRef{
			Chain:    domain.Chain(metadataStringOrDefault(activity.Metadata, "token_chain", string(activity.Chain))),
			Address:  tokenAddress,
			Symbol:   metadataStringOrDefault(activity.Metadata, "token_symbol", ""),
			Decimals: metadataIntOrDefault(activity.Metadata, "token_decimals", 0),
		}
	}

	if err := domain.ValidateNormalizedTransaction(tx); err != nil {
		return domain.NormalizedTransaction{}, err
	}

	return tx, nil
}

func NormalizeProviderActivities(activities []ProviderWalletActivity) ([]domain.NormalizedTransaction, error) {
	transactions := make([]domain.NormalizedTransaction, 0, len(activities))
	for _, activity := range activities {
		tx, err := NormalizeProviderActivity(activity)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func metadataStringOrDefault(metadata map[string]any, key string, fallback string) string {
	if metadata == nil {
		return fallback
	}

	value, ok := metadata[key]
	if !ok {
		return fallback
	}

	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return fallback
		}
		return trimmed
	case fmt.Stringer:
		trimmed := strings.TrimSpace(typed.String())
		if trimmed == "" {
			return fallback
		}
		return trimmed
	default:
		return fallback
	}
}

func metadataIntOrDefault(metadata map[string]any, key string, fallback int) int {
	if metadata == nil {
		return fallback
	}

	value, ok := metadata[key]
	if !ok {
		return fallback
	}

	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}

func metadataInt64OrDefault(metadata map[string]any, key string, fallback int64) int64 {
	if metadata == nil {
		return fallback
	}

	value, ok := metadata[key]
	if !ok {
		return fallback
	}

	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}

func normalizeTransactionDirection(value string) domain.TransactionDirection {
	switch domain.TransactionDirection(strings.ToLower(strings.TrimSpace(value))) {
	case domain.TransactionDirectionInbound:
		return domain.TransactionDirectionInbound
	case domain.TransactionDirectionOutbound:
		return domain.TransactionDirectionOutbound
	case domain.TransactionDirectionSelf:
		return domain.TransactionDirectionSelf
	default:
		return domain.TransactionDirectionUnknown
	}
}
