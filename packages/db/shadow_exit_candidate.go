package db

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/qorvi/qorvi/packages/domain"
)

const shadowExitCandidateIdentitySQL = `
SELECT
  w.id,
  w.chain,
  w.address,
  w.display_name,
  COALESCE(w.entity_key, '') AS entity_key,
  COALESCE(e.entity_type, '') AS entity_type
FROM wallets w
LEFT JOIN entities e ON e.entity_key = w.entity_key
WHERE w.chain = $1 AND w.address = $2
LIMIT 1
`

const shadowExitCandidateTransactionsSQL = `
SELECT
  t.direction,
  t.observed_at,
  COALESCE(t.counterparty_chain, '') AS counterparty_chain,
  COALESCE(t.counterparty_address, '') AS counterparty_address,
  COALESCE(cp.display_name, '') AS counterparty_display_name,
  COALESCE(cp.entity_key, '') AS counterparty_entity_key,
  COALESCE(cp_entity.entity_type, '') AS counterparty_entity_type
FROM transactions t
LEFT JOIN wallets cp
  ON cp.chain = t.counterparty_chain
 AND cp.address = t.counterparty_address
LEFT JOIN entities cp_entity
  ON cp_entity.entity_key = cp.entity_key
WHERE t.wallet_id = $1
  AND t.observed_at >= $2
ORDER BY t.observed_at DESC, t.id DESC
`

type ShadowExitCandidateReader interface {
	ReadShadowExitCandidateMetrics(ctx context.Context, ref WalletRef, window time.Duration) (ShadowExitCandidateMetrics, error)
}

type PostgresShadowExitCandidateReader struct {
	Querier postgresQuerier
	Now     func() time.Time
}

type ShadowExitCandidateMetrics struct {
	WalletID    string
	Chain       domain.Chain
	Address     string
	DisplayName string
	EntityKey   string
	EntityType  string

	WindowStart time.Time
	WindowEnd   time.Time

	TotalTxCount                       int64
	InboundTxCount                     int64
	OutboundTxCount                    int64
	UniqueCounterpartyCount            int64
	FanOutCounterpartyCount            int64
	OutflowRatio                       float64
	BridgeRelatedCount                 int64
	CEXProximityCount                  int64
	TreasuryCounterpartyCount          int64
	InternalRebalanceCounterpartyCount int64

	DiscountInputs    ShadowExitCandidateDiscountInputs
	TopCounterparties []ShadowExitCandidateCounterparty
}

type ShadowExitCandidateCounterparty struct {
	Chain            domain.Chain
	Address          string
	Label            string
	EntityKey        string
	EntityType       string
	InteractionCount int64
	LatestActivityAt time.Time
	IsBridge         bool
	IsCEX            bool
	IsTreasury       bool
	IsInternal       bool
}

type ShadowExitCandidateDiscountInputs struct {
	RootWhitelist         bool
	RootTreasury          bool
	RootInternalRebalance bool
	Reasons               []string
}

type shadowExitCandidateWalletIdentity struct {
	WalletID    string
	Chain       domain.Chain
	Address     string
	DisplayName string
	EntityKey   string
	EntityType  string
}

type shadowExitCandidateTransactionRow struct {
	Direction               string
	ObservedAt              time.Time
	CounterpartyChain       string
	CounterpartyAddress     string
	CounterpartyDisplayName string
	CounterpartyEntityKey   string
	CounterpartyEntityType  string
}

type shadowExitCandidateCounterpartyAggregate struct {
	Chain            domain.Chain
	Address          string
	Label            string
	EntityKey        string
	EntityType       string
	InteractionCount int64
	LatestActivityAt time.Time
	IsBridge         bool
	IsCEX            bool
	IsTreasury       bool
	IsInternal       bool
}

func NewPostgresShadowExitCandidateReader(querier postgresQuerier) *PostgresShadowExitCandidateReader {
	return &PostgresShadowExitCandidateReader{
		Querier: querier,
		Now:     time.Now,
	}
}

func NewPostgresShadowExitCandidateReaderFromPool(pool postgresQuerier) *PostgresShadowExitCandidateReader {
	return NewPostgresShadowExitCandidateReader(pool)
}

func (r *PostgresShadowExitCandidateReader) ReadShadowExitCandidateMetrics(
	ctx context.Context,
	ref WalletRef,
	window time.Duration,
) (ShadowExitCandidateMetrics, error) {
	if r == nil || r.Querier == nil {
		return ShadowExitCandidateMetrics{}, fmt.Errorf("shadow exit candidate reader is nil")
	}

	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return ShadowExitCandidateMetrics{}, err
	}

	now := r.now().UTC()
	if window <= 0 {
		window = 24 * time.Hour
	}
	windowStart := now.Add(-window)

	identity, err := r.readIdentity(ctx, normalized)
	if err != nil {
		return ShadowExitCandidateMetrics{}, err
	}

	rows, err := r.Querier.Query(ctx, shadowExitCandidateTransactionsSQL, identity.WalletID, windowStart)
	if err != nil {
		return ShadowExitCandidateMetrics{}, fmt.Errorf("list shadow exit candidate transactions: %w", err)
	}
	defer rows.Close()

	aggregates := map[string]*shadowExitCandidateCounterpartyAggregate{}
	uniqueCounterparties := map[string]struct{}{}
	outboundCounterparties := map[string]struct{}{}
	bridgeCounterparties := map[string]struct{}{}
	cexCounterparties := map[string]struct{}{}
	treasuryCounterparties := map[string]struct{}{}
	internalCounterparties := map[string]struct{}{}

	var (
		totalTxCount    int64
		inboundTxCount  int64
		outboundTxCount int64
	)

	for rows.Next() {
		var record shadowExitCandidateTransactionRow
		if err := rows.Scan(
			&record.Direction,
			&record.ObservedAt,
			&record.CounterpartyChain,
			&record.CounterpartyAddress,
			&record.CounterpartyDisplayName,
			&record.CounterpartyEntityKey,
			&record.CounterpartyEntityType,
		); err != nil {
			return ShadowExitCandidateMetrics{}, fmt.Errorf("scan shadow exit candidate transaction: %w", err)
		}

		totalTxCount++
		direction := strings.ToLower(strings.TrimSpace(record.Direction))
		switch direction {
		case string(domain.TransactionDirectionInbound):
			inboundTxCount++
		case string(domain.TransactionDirectionOutbound):
			outboundTxCount++
		}

		counterpartyAddress := strings.TrimSpace(record.CounterpartyAddress)
		if counterpartyAddress == "" {
			continue
		}

		counterpartyChain := normalizeCandidateChain(record.CounterpartyChain, normalized.Chain)
		counterpartyKey := shadowExitCandidateCanonicalKey(counterpartyChain, counterpartyAddress)
		aggregate := aggregates[counterpartyKey]
		if aggregate == nil {
			aggregate = &shadowExitCandidateCounterpartyAggregate{
				Chain:      counterpartyChain,
				Address:    counterpartyAddress,
				Label:      firstNonEmpty(strings.TrimSpace(record.CounterpartyDisplayName), strings.TrimSpace(record.CounterpartyEntityType), counterpartyAddress),
				EntityKey:  strings.TrimSpace(record.CounterpartyEntityKey),
				EntityType: strings.TrimSpace(record.CounterpartyEntityType),
			}
			aggregates[counterpartyKey] = aggregate
		}

		aggregate.InteractionCount++
		if record.ObservedAt.After(aggregate.LatestActivityAt) {
			aggregate.LatestActivityAt = record.ObservedAt.UTC()
		}

		joinedText := strings.Join([]string{
			strings.TrimSpace(record.CounterpartyDisplayName),
			strings.TrimSpace(record.CounterpartyEntityType),
			strings.TrimSpace(record.CounterpartyEntityKey),
		}, " ")
		if classifyShadowExitBridge(joinedText) {
			aggregate.IsBridge = true
			bridgeCounterparties[counterpartyKey] = struct{}{}
		}
		if classifyShadowExitCEX(joinedText) {
			aggregate.IsCEX = true
			cexCounterparties[counterpartyKey] = struct{}{}
		}
		if classifyShadowExitTreasury(joinedText) {
			aggregate.IsTreasury = true
			treasuryCounterparties[counterpartyKey] = struct{}{}
		}
		if classifyShadowExitInternal(joinedText) {
			aggregate.IsInternal = true
			internalCounterparties[counterpartyKey] = struct{}{}
		}

		uniqueCounterparties[counterpartyKey] = struct{}{}
		if direction == string(domain.TransactionDirectionOutbound) {
			outboundCounterparties[counterpartyKey] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return ShadowExitCandidateMetrics{}, fmt.Errorf("iterate shadow exit candidate transactions: %w", err)
	}

	candidate := ShadowExitCandidateMetrics{
		WalletID:                           identity.WalletID,
		Chain:                              identity.Chain,
		Address:                            identity.Address,
		DisplayName:                        identity.DisplayName,
		EntityKey:                          identity.EntityKey,
		EntityType:                         identity.EntityType,
		WindowStart:                        now.Add(-window),
		WindowEnd:                          now,
		TotalTxCount:                       totalTxCount,
		InboundTxCount:                     inboundTxCount,
		OutboundTxCount:                    outboundTxCount,
		UniqueCounterpartyCount:            int64(len(uniqueCounterparties)),
		FanOutCounterpartyCount:            int64(len(outboundCounterparties)),
		BridgeRelatedCount:                 int64(len(bridgeCounterparties)),
		CEXProximityCount:                  int64(len(cexCounterparties)),
		TreasuryCounterpartyCount:          int64(len(treasuryCounterparties)),
		InternalRebalanceCounterpartyCount: int64(len(internalCounterparties)),
		DiscountInputs:                     buildShadowExitCandidateDiscountInputs(identity, int64(len(bridgeCounterparties)), int64(len(cexCounterparties)), int64(len(treasuryCounterparties)), int64(len(internalCounterparties))),
	}
	if candidate.TotalTxCount > 0 {
		candidate.OutflowRatio = float64(candidate.OutboundTxCount) / float64(candidate.TotalTxCount)
	}

	candidate.TopCounterparties = buildShadowExitCandidateCounterparties(aggregates)
	candidate.DiscountInputs.Reasons = append(
		candidate.DiscountInputs.Reasons,
		buildShadowExitCandidateCounterpartyReasons(candidate.BridgeRelatedCount, candidate.CEXProximityCount, candidate.TreasuryCounterpartyCount, candidate.InternalRebalanceCounterpartyCount)...,
	)

	return candidate, nil
}

func (r *PostgresShadowExitCandidateReader) readIdentity(ctx context.Context, ref WalletRef) (shadowExitCandidateWalletIdentity, error) {
	var identity shadowExitCandidateWalletIdentity
	if err := r.Querier.QueryRow(
		ctx,
		shadowExitCandidateIdentitySQL,
		string(ref.Chain),
		ref.Address,
	).Scan(
		&identity.WalletID,
		&identity.Chain,
		&identity.Address,
		&identity.DisplayName,
		&identity.EntityKey,
		&identity.EntityType,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return shadowExitCandidateWalletIdentity{}, ErrWalletSummaryNotFound
		}

		return shadowExitCandidateWalletIdentity{}, fmt.Errorf("scan shadow exit candidate identity: %w", err)
	}

	return identity, nil
}

func (r *PostgresShadowExitCandidateReader) now() time.Time {
	if r != nil && r.Now != nil {
		return r.Now()
	}

	return time.Now()
}

func buildShadowExitCandidateCounterparties(
	aggregates map[string]*shadowExitCandidateCounterpartyAggregate,
) []ShadowExitCandidateCounterparty {
	counterparties := make([]ShadowExitCandidateCounterparty, 0, len(aggregates))
	for _, aggregate := range aggregates {
		if aggregate == nil {
			continue
		}

		counterparties = append(counterparties, ShadowExitCandidateCounterparty{
			Chain:            aggregate.Chain,
			Address:          aggregate.Address,
			Label:            aggregate.Label,
			EntityKey:        aggregate.EntityKey,
			EntityType:       aggregate.EntityType,
			InteractionCount: aggregate.InteractionCount,
			LatestActivityAt: aggregate.LatestActivityAt,
			IsBridge:         aggregate.IsBridge,
			IsCEX:            aggregate.IsCEX,
			IsTreasury:       aggregate.IsTreasury,
			IsInternal:       aggregate.IsInternal,
		})
	}

	sort.Slice(counterparties, func(i, j int) bool {
		if counterparties[i].InteractionCount != counterparties[j].InteractionCount {
			return counterparties[i].InteractionCount > counterparties[j].InteractionCount
		}
		if !counterparties[i].LatestActivityAt.Equal(counterparties[j].LatestActivityAt) {
			return counterparties[i].LatestActivityAt.After(counterparties[j].LatestActivityAt)
		}
		if counterparties[i].Chain != counterparties[j].Chain {
			return counterparties[i].Chain < counterparties[j].Chain
		}
		return counterparties[i].Address < counterparties[j].Address
	})

	if len(counterparties) > 5 {
		counterparties = counterparties[:5]
	}

	return counterparties
}

func buildShadowExitCandidateDiscountInputs(
	identity shadowExitCandidateWalletIdentity,
	bridgeCounterparties int64,
	cexCounterparties int64,
	treasuryCounterparties int64,
	internalCounterparties int64,
) ShadowExitCandidateDiscountInputs {
	joined := strings.Join([]string{
		identity.DisplayName,
		identity.EntityType,
		identity.EntityKey,
	}, " ")

	inputs := ShadowExitCandidateDiscountInputs{
		RootWhitelist:         classifyShadowExitWhitelist(joined),
		RootTreasury:          classifyShadowExitTreasury(joined),
		RootInternalRebalance: classifyShadowExitInternal(joined),
	}

	if inputs.RootWhitelist {
		inputs.Reasons = append(inputs.Reasons, "root wallet appears whitelist/allowlist managed")
	}
	if inputs.RootTreasury {
		inputs.Reasons = append(inputs.Reasons, "root wallet appears treasury managed")
	}
	if inputs.RootInternalRebalance {
		inputs.Reasons = append(inputs.Reasons, "root wallet appears internal rebalance managed")
	}
	if bridgeCounterparties > 0 {
		inputs.Reasons = append(inputs.Reasons, fmt.Sprintf("%d bridge counterparties in window", bridgeCounterparties))
	}
	if cexCounterparties > 0 {
		inputs.Reasons = append(inputs.Reasons, fmt.Sprintf("%d cex-like counterparties in window", cexCounterparties))
	}
	if treasuryCounterparties > 0 {
		inputs.Reasons = append(inputs.Reasons, fmt.Sprintf("%d treasury-like counterparties in window", treasuryCounterparties))
	}
	if internalCounterparties > 0 {
		inputs.Reasons = append(inputs.Reasons, fmt.Sprintf("%d internal-rebalance-like counterparties in window", internalCounterparties))
	}

	return inputs
}

func buildShadowExitCandidateCounterpartyReasons(
	bridgeCounterparties int64,
	cexCounterparties int64,
	treasuryCounterparties int64,
	internalCounterparties int64,
) []string {
	reasons := make([]string, 0, 4)
	if bridgeCounterparties > 0 {
		reasons = append(reasons, fmt.Sprintf("bridge-related counterparties: %d", bridgeCounterparties))
	}
	if cexCounterparties > 0 {
		reasons = append(reasons, fmt.Sprintf("cex proximity counterparties: %d", cexCounterparties))
	}
	if treasuryCounterparties > 0 {
		reasons = append(reasons, fmt.Sprintf("treasury-like counterparties: %d", treasuryCounterparties))
	}
	if internalCounterparties > 0 {
		reasons = append(reasons, fmt.Sprintf("internal-rebalance-like counterparties: %d", internalCounterparties))
	}

	return reasons
}

func normalizeCandidateChain(chain string, fallback domain.Chain) domain.Chain {
	normalized := domain.Chain(strings.ToLower(strings.TrimSpace(chain)))
	if normalized == "" {
		return fallback
	}

	return normalized
}

func shadowExitCandidateCanonicalKey(chain domain.Chain, address string) string {
	return fmt.Sprintf("%s:%s", strings.ToLower(strings.TrimSpace(string(chain))), strings.TrimSpace(address))
}

func classifyShadowExitBridge(text string) bool {
	return containsAny(strings.ToLower(text), []string{
		"bridge",
		"wormhole",
		"stargate",
		"layerzero",
		"axelar",
		"synapse",
	})
}

func classifyShadowExitCEX(text string) bool {
	return containsAny(strings.ToLower(text), []string{
		"cex",
		"exchange",
		"binance",
		"coinbase",
		"kraken",
		"okx",
		"bybit",
		"gate",
		"huobi",
		"kucoin",
		"bitstamp",
		"upbit",
		"bitget",
	})
}

func classifyShadowExitTreasury(text string) bool {
	return containsAny(strings.ToLower(text), []string{
		"treasury",
		"treas",
		"multisig",
		"safe",
	})
}

func classifyShadowExitInternal(text string) bool {
	return containsAny(strings.ToLower(text), []string{
		"internal",
		"rebalance",
		"rebal",
		"allowlist",
		"whitelist",
		"whitelisted",
	})
}

func classifyShadowExitWhitelist(text string) bool {
	return containsAny(strings.ToLower(text), []string{
		"allowlist",
		"whitelist",
		"whitelisted",
	})
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	return false
}
