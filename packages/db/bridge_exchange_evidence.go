package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

const walletBridgeExchangeIdentitySQL = `
SELECT
  w.id,
  w.chain,
  w.address,
  w.display_name
FROM wallets w
WHERE w.chain = $1 AND w.address = $2
LIMIT 1
`

const walletBridgeExchangeTransactionsSQL = `
SELECT
  t.tx_hash,
  t.direction,
  t.observed_at,
  COALESCE(t.counterparty_chain, '') AS counterparty_chain,
  COALESCE(t.counterparty_address, '') AS counterparty_address,
  COALESCE(cp.display_name, '') AS counterparty_display_name,
  COALESCE(cp.entity_key, '') AS counterparty_entity_key,
  COALESCE(cp_entity.entity_type, '') AS counterparty_entity_type,
  COALESCE(t.amount_numeric, 0::numeric)::text AS amount_numeric,
  COALESCE(t.token_symbol, '') AS token_symbol
FROM transactions t
LEFT JOIN wallets cp
  ON cp.chain = COALESCE(NULLIF(t.counterparty_chain, ''), $2)
 AND cp.address = t.counterparty_address
LEFT JOIN entities cp_entity
  ON cp_entity.entity_key = cp.entity_key
WHERE t.wallet_id = $1
  AND t.direction = 'outbound'
  AND t.observed_at >= $3
ORDER BY t.observed_at DESC, t.id DESC
`

const walletBridgeExchangeDownstreamSQL = `
SELECT
  t.tx_hash,
  t.observed_at,
  COALESCE(t.counterparty_chain, '') AS counterparty_chain,
  COALESCE(t.counterparty_address, '') AS counterparty_address,
  COALESCE(cp.display_name, '') AS counterparty_display_name,
  COALESCE(cp.entity_key, '') AS counterparty_entity_key,
  COALESCE(cp_entity.entity_type, '') AS counterparty_entity_type
FROM wallets w
JOIN transactions t
  ON t.wallet_id = w.id
LEFT JOIN wallets cp
  ON cp.chain = COALESCE(NULLIF(t.counterparty_chain, ''), w.chain)
 AND cp.address = t.counterparty_address
LEFT JOIN entities cp_entity
  ON cp_entity.entity_key = cp.entity_key
WHERE w.chain = $1
  AND w.address = $2
  AND t.direction = 'outbound'
  AND t.observed_at >= $3
  AND t.observed_at <= $4
ORDER BY t.observed_at ASC, t.id ASC
LIMIT 10
`

const walletAddressPriorActivityExistsSQL = `
SELECT EXISTS (
  SELECT 1
  FROM wallets w
  JOIN transactions t ON t.wallet_id = w.id
  WHERE w.chain = $1
    AND w.address = $2
    AND t.observed_at < $3
)
`

const deleteWalletBridgeLinksForDaySQL = `
DELETE FROM wallet_bridge_links
WHERE wallet_id = $1
  AND observed_day = $2
`

const deleteWalletExchangePathsForDaySQL = `
DELETE FROM wallet_exchange_paths
WHERE wallet_id = $1
  AND observed_day = $2
`

const upsertWalletBridgeFeaturesDailySQL = `
INSERT INTO wallet_bridge_features_daily (
  wallet_id,
  observed_day,
  window_start_at,
  window_end_at,
  bridge_outbound_count,
  distinct_bridge_counterparties,
  distinct_bridge_protocols,
  confirmed_destination_count,
  post_bridge_fresh_wallet_count,
  post_bridge_exchange_touch_count,
  post_bridge_protocol_entry_count,
  bridge_outflow_amount,
  bridge_outflow_share,
  bridge_recurrence_days,
  latest_bridge_tx_hash,
  metadata,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
  $12, $13, $14, $15, $16, $17
)
ON CONFLICT (wallet_id, observed_day) DO UPDATE SET
  window_start_at = EXCLUDED.window_start_at,
  window_end_at = EXCLUDED.window_end_at,
  bridge_outbound_count = EXCLUDED.bridge_outbound_count,
  distinct_bridge_counterparties = EXCLUDED.distinct_bridge_counterparties,
  distinct_bridge_protocols = EXCLUDED.distinct_bridge_protocols,
  confirmed_destination_count = EXCLUDED.confirmed_destination_count,
  post_bridge_fresh_wallet_count = EXCLUDED.post_bridge_fresh_wallet_count,
  post_bridge_exchange_touch_count = EXCLUDED.post_bridge_exchange_touch_count,
  post_bridge_protocol_entry_count = EXCLUDED.post_bridge_protocol_entry_count,
  bridge_outflow_amount = EXCLUDED.bridge_outflow_amount,
  bridge_outflow_share = EXCLUDED.bridge_outflow_share,
  bridge_recurrence_days = EXCLUDED.bridge_recurrence_days,
  latest_bridge_tx_hash = EXCLUDED.latest_bridge_tx_hash,
  metadata = EXCLUDED.metadata,
  updated_at = EXCLUDED.updated_at
`

const upsertWalletExchangeFeaturesDailySQL = `
INSERT INTO wallet_exchange_flow_features_daily (
  wallet_id,
  observed_day,
  window_start_at,
  window_end_at,
  exchange_outbound_count,
  distinct_exchange_counterparties,
  deposit_like_path_count,
  exchange_fanout_count,
  fresh_recipient_count,
  exchange_outflow_amount,
  exchange_outflow_share,
  exchange_recurrence_days,
  latest_exchange_tx_hash,
  metadata,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9,
  $10, $11, $12, $13, $14, $15
)
ON CONFLICT (wallet_id, observed_day) DO UPDATE SET
  window_start_at = EXCLUDED.window_start_at,
  window_end_at = EXCLUDED.window_end_at,
  exchange_outbound_count = EXCLUDED.exchange_outbound_count,
  distinct_exchange_counterparties = EXCLUDED.distinct_exchange_counterparties,
  deposit_like_path_count = EXCLUDED.deposit_like_path_count,
  exchange_fanout_count = EXCLUDED.exchange_fanout_count,
  fresh_recipient_count = EXCLUDED.fresh_recipient_count,
  exchange_outflow_amount = EXCLUDED.exchange_outflow_amount,
  exchange_outflow_share = EXCLUDED.exchange_outflow_share,
  exchange_recurrence_days = EXCLUDED.exchange_recurrence_days,
  latest_exchange_tx_hash = EXCLUDED.latest_exchange_tx_hash,
  metadata = EXCLUDED.metadata,
  updated_at = EXCLUDED.updated_at
`

const insertWalletBridgeLinkSQL = `
INSERT INTO wallet_bridge_links (
  wallet_id,
  observed_day,
  tx_hash,
  observed_at,
  bridge_chain,
  bridge_address,
  bridge_label,
  bridge_entity_key,
  bridge_entity_type,
  amount_numeric,
  token_symbol,
  destination_chain,
  destination_address,
  destination_label,
  destination_entity_key,
  destination_entity_type,
  destination_tx_hash,
  destination_observed_at,
  confidence,
  metadata,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, '')::numeric,
  $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
)
ON CONFLICT (wallet_id, tx_hash, bridge_chain, bridge_address, destination_tx_hash, observed_at) DO UPDATE SET
  bridge_label = EXCLUDED.bridge_label,
  bridge_entity_key = EXCLUDED.bridge_entity_key,
  bridge_entity_type = EXCLUDED.bridge_entity_type,
  amount_numeric = EXCLUDED.amount_numeric,
  token_symbol = EXCLUDED.token_symbol,
  destination_chain = EXCLUDED.destination_chain,
  destination_address = EXCLUDED.destination_address,
  destination_label = EXCLUDED.destination_label,
  destination_entity_key = EXCLUDED.destination_entity_key,
  destination_entity_type = EXCLUDED.destination_entity_type,
  destination_observed_at = EXCLUDED.destination_observed_at,
  confidence = EXCLUDED.confidence,
  metadata = EXCLUDED.metadata,
  updated_at = EXCLUDED.updated_at
`

const insertWalletExchangePathSQL = `
INSERT INTO wallet_exchange_paths (
  wallet_id,
  observed_day,
  tx_hash,
  observed_at,
  path_kind,
  intermediary_chain,
  intermediary_address,
  intermediary_label,
  intermediary_entity_key,
  intermediary_entity_type,
  exchange_chain,
  exchange_address,
  exchange_label,
  exchange_entity_key,
  exchange_entity_type,
  exchange_tx_hash,
  exchange_observed_at,
  amount_numeric,
  token_symbol,
  confidence,
  metadata,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
  $11, $12, $13, $14, $15, $16, $17, NULLIF($18, '')::numeric,
  $19, $20, $21, $22
)
ON CONFLICT (wallet_id, tx_hash, path_kind, exchange_chain, exchange_address, exchange_tx_hash, observed_at) DO UPDATE SET
  intermediary_chain = EXCLUDED.intermediary_chain,
  intermediary_address = EXCLUDED.intermediary_address,
  intermediary_label = EXCLUDED.intermediary_label,
  intermediary_entity_key = EXCLUDED.intermediary_entity_key,
  intermediary_entity_type = EXCLUDED.intermediary_entity_type,
  exchange_label = EXCLUDED.exchange_label,
  exchange_entity_key = EXCLUDED.exchange_entity_key,
  exchange_entity_type = EXCLUDED.exchange_entity_type,
  exchange_observed_at = EXCLUDED.exchange_observed_at,
  amount_numeric = EXCLUDED.amount_numeric,
  token_symbol = EXCLUDED.token_symbol,
  confidence = EXCLUDED.confidence,
  metadata = EXCLUDED.metadata,
  updated_at = EXCLUDED.updated_at
`

type WalletBridgeExchangeEvidenceReader interface {
	ReadWalletBridgeExchangeEvidence(context.Context, WalletRef, time.Duration) (WalletBridgeExchangeEvidenceReport, error)
}

type WalletBridgeExchangeEvidenceStore interface {
	ReplaceWalletBridgeExchangeEvidence(context.Context, WalletBridgeExchangeEvidenceReport) error
}

type WalletBridgeExchangeEvidenceReadWriter interface {
	WalletBridgeExchangeEvidenceReader
	WalletBridgeExchangeEvidenceStore
}

type WalletBridgeLinkObservation struct {
	TxHash                string
	ObservedAt            time.Time
	BridgeChain           domain.Chain
	BridgeAddress         string
	BridgeLabel           string
	BridgeEntityKey       string
	BridgeEntityType      string
	Amount                string
	TokenSymbol           string
	DestinationChain      domain.Chain
	DestinationAddress    string
	DestinationLabel      string
	DestinationEntityKey  string
	DestinationEntityType string
	DestinationTxHash     string
	DestinationObservedAt *time.Time
	FreshDestination      bool
	Confidence            float64
}

type WalletExchangePathObservation struct {
	TxHash                 string
	ObservedAt             time.Time
	PathKind               string
	IntermediaryChain      domain.Chain
	IntermediaryAddress    string
	IntermediaryLabel      string
	IntermediaryEntityKey  string
	IntermediaryEntityType string
	ExchangeChain          domain.Chain
	ExchangeAddress        string
	ExchangeLabel          string
	ExchangeEntityKey      string
	ExchangeEntityType     string
	ExchangeTxHash         string
	ExchangeObservedAt     *time.Time
	Amount                 string
	TokenSymbol            string
	FreshRecipient         bool
	Confidence             float64
}

type WalletBridgeFeatures struct {
	BridgeOutboundCount          int
	DistinctBridgeCounterparties int
	DistinctBridgeProtocols      int
	ConfirmedDestinationCount    int
	PostBridgeFreshWalletCount   int
	PostBridgeExchangeTouchCount int
	PostBridgeProtocolEntryCount int
	BridgeOutflowAmount          string
	BridgeOutflowShare           float64
	BridgeRecurrenceDays         int
	LatestBridgeTxHash           string
}

type WalletExchangeFlowFeatures struct {
	ExchangeOutboundCount          int
	DistinctExchangeCounterparties int
	DepositLikePathCount           int
	ExchangeFanoutCount            int
	FreshRecipientCount            int
	ExchangeOutflowAmount          string
	ExchangeOutflowShare           float64
	ExchangeRecurrenceDays         int
	LatestExchangeTxHash           string
}

type WalletBridgeExchangeEvidenceReport struct {
	WalletID         string
	Chain            domain.Chain
	Address          string
	DisplayName      string
	WindowStartAt    time.Time
	WindowEndAt      time.Time
	BridgeLinks      []WalletBridgeLinkObservation
	ExchangePaths    []WalletExchangePathObservation
	BridgeFeatures   WalletBridgeFeatures
	ExchangeFeatures WalletExchangeFlowFeatures
}

type walletBridgeExchangeIdentity struct {
	WalletID    string
	Chain       domain.Chain
	Address     string
	DisplayName string
}

type walletBridgeExchangeTransactionRow struct {
	TxHash                  string
	Direction               string
	ObservedAt              time.Time
	CounterpartyChain       string
	CounterpartyAddress     string
	CounterpartyDisplayName string
	CounterpartyEntityKey   string
	CounterpartyEntityType  string
	Amount                  string
	TokenSymbol             string
}

type PostgresWalletBridgeExchangeEvidenceStore struct {
	Querier postgresQuerier
	Execer  postgresFindingExecer
	Now     func() time.Time
}

func NewPostgresWalletBridgeExchangeEvidenceStore(
	querier postgresQuerier,
	execer postgresFindingExecer,
) *PostgresWalletBridgeExchangeEvidenceStore {
	return &PostgresWalletBridgeExchangeEvidenceStore{
		Querier: querier,
		Execer:  execer,
		Now:     time.Now,
	}
}

func NewPostgresWalletBridgeExchangeEvidenceStoreFromPool(pool interface {
	postgresQuerier
	postgresFindingExecer
}) *PostgresWalletBridgeExchangeEvidenceStore {
	return NewPostgresWalletBridgeExchangeEvidenceStore(pool, pool)
}

func (s *PostgresWalletBridgeExchangeEvidenceStore) ReadWalletBridgeExchangeEvidence(
	ctx context.Context,
	ref WalletRef,
	window time.Duration,
) (WalletBridgeExchangeEvidenceReport, error) {
	if s == nil || s.Querier == nil {
		return WalletBridgeExchangeEvidenceReport{}, fmt.Errorf("wallet bridge/exchange evidence store is nil")
	}
	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return WalletBridgeExchangeEvidenceReport{}, err
	}
	if window <= 0 {
		window = 24 * time.Hour
	}
	windowEnd := s.now().UTC()
	windowStart := windowEnd.Add(-window)
	identity, err := s.readIdentity(ctx, normalized)
	if err != nil {
		return WalletBridgeExchangeEvidenceReport{}, err
	}
	rows, err := s.Querier.Query(
		ctx,
		walletBridgeExchangeTransactionsSQL,
		identity.WalletID,
		string(normalized.Chain),
		windowStart,
	)
	if err != nil {
		return WalletBridgeExchangeEvidenceReport{}, fmt.Errorf("list wallet bridge/exchange transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]walletBridgeExchangeTransactionRow, 0, 32)
	totalOutboundAmount := 0.0
	for rows.Next() {
		var item walletBridgeExchangeTransactionRow
		if err := rows.Scan(
			&item.TxHash,
			&item.Direction,
			&item.ObservedAt,
			&item.CounterpartyChain,
			&item.CounterpartyAddress,
			&item.CounterpartyDisplayName,
			&item.CounterpartyEntityKey,
			&item.CounterpartyEntityType,
			&item.Amount,
			&item.TokenSymbol,
		); err != nil {
			return WalletBridgeExchangeEvidenceReport{}, fmt.Errorf("scan wallet bridge/exchange transaction: %w", err)
		}
		transactions = append(transactions, item)
		totalOutboundAmount += numericStringToFloat(item.Amount)
	}
	if err := rows.Err(); err != nil {
		return WalletBridgeExchangeEvidenceReport{}, fmt.Errorf("iterate wallet bridge/exchange transactions: %w", err)
	}

	report := WalletBridgeExchangeEvidenceReport{
		WalletID:      identity.WalletID,
		Chain:         identity.Chain,
		Address:       identity.Address,
		DisplayName:   identity.DisplayName,
		WindowStartAt: windowStart,
		WindowEndAt:   windowEnd,
	}

	bridgeProtocolKeys := map[string]struct{}{}
	bridgeCounterparties := map[string]struct{}{}
	bridgeDays := map[string]struct{}{}
	exchangeCounterparties := map[string]struct{}{}
	exchangeDays := map[string]struct{}{}
	exchangeFanout := map[string]struct{}{}
	bridgeOutflowAmount := 0.0
	exchangeOutflowAmount := 0.0

	for _, tx := range transactions {
		counterpartyAddress := strings.TrimSpace(tx.CounterpartyAddress)
		if counterpartyAddress == "" {
			continue
		}
		counterpartyChain := normalizeCandidateChain(tx.CounterpartyChain, normalized.Chain)
		joinedText := strings.Join([]string{
			strings.TrimSpace(tx.CounterpartyDisplayName),
			strings.TrimSpace(tx.CounterpartyEntityType),
			strings.TrimSpace(tx.CounterpartyEntityKey),
		}, " ")

		if classifyShadowExitBridge(joinedText) {
			bridgeProtocolKeys[firstNonEmpty(strings.TrimSpace(tx.CounterpartyEntityKey), strings.ToLower(counterpartyAddress))] = struct{}{}
			bridgeCounterparties[shadowExitCandidateCanonicalKey(counterpartyChain, counterpartyAddress)] = struct{}{}
			bridgeDays[tx.ObservedAt.UTC().Format("2006-01-02")] = struct{}{}
			bridgeOutflowAmount += numericStringToFloat(tx.Amount)

			observation, found, err := s.buildBridgeObservation(
				ctx,
				normalized.Chain,
				tx,
			)
			if err != nil {
				return WalletBridgeExchangeEvidenceReport{}, err
			}
			if found {
				report.BridgeLinks = append(report.BridgeLinks, observation)
			}
			continue
		}

		if classifyShadowExitCEX(joinedText) {
			exchangeCounterparties[shadowExitCandidateCanonicalKey(counterpartyChain, counterpartyAddress)] = struct{}{}
			exchangeDays[tx.ObservedAt.UTC().Format("2006-01-02")] = struct{}{}
			exchangeFanout[shadowExitCandidateCanonicalKey(counterpartyChain, counterpartyAddress)] = struct{}{}
			exchangeOutflowAmount += numericStringToFloat(tx.Amount)
			report.ExchangePaths = append(report.ExchangePaths, WalletExchangePathObservation{
				TxHash:             strings.TrimSpace(tx.TxHash),
				ObservedAt:         tx.ObservedAt.UTC(),
				PathKind:           "direct_exchange_outflow",
				ExchangeChain:      counterpartyChain,
				ExchangeAddress:    counterpartyAddress,
				ExchangeLabel:      firstNonEmpty(strings.TrimSpace(tx.CounterpartyDisplayName), counterpartyAddress),
				ExchangeEntityKey:  strings.TrimSpace(tx.CounterpartyEntityKey),
				ExchangeEntityType: strings.TrimSpace(tx.CounterpartyEntityType),
				Amount:             strings.TrimSpace(tx.Amount),
				TokenSymbol:        strings.TrimSpace(tx.TokenSymbol),
				Confidence:         0.82,
			})
			continue
		}

		observation, found, err := s.buildExchangeObservation(ctx, normalized.Chain, tx)
		if err != nil {
			return WalletBridgeExchangeEvidenceReport{}, err
		}
		if found {
			exchangeCounterparties[shadowExitCandidateCanonicalKey(observation.ExchangeChain, observation.ExchangeAddress)] = struct{}{}
			exchangeDays[tx.ObservedAt.UTC().Format("2006-01-02")] = struct{}{}
			exchangeFanout[shadowExitCandidateCanonicalKey(counterpartyChain, counterpartyAddress)] = struct{}{}
			exchangeOutflowAmount += numericStringToFloat(tx.Amount)
			report.ExchangePaths = append(report.ExchangePaths, observation)
		}
	}

	report.BridgeFeatures = WalletBridgeFeatures{
		BridgeOutboundCount:          len(report.BridgeLinks),
		DistinctBridgeCounterparties: len(bridgeCounterparties),
		DistinctBridgeProtocols:      len(bridgeProtocolKeys),
		ConfirmedDestinationCount:    countBridgeConfirmedDestinations(report.BridgeLinks),
		PostBridgeFreshWalletCount:   countBridgeFreshDestinations(report.BridgeLinks),
		PostBridgeExchangeTouchCount: countBridgeExchangeTouchDestinations(report.BridgeLinks),
		PostBridgeProtocolEntryCount: countBridgeProtocolEntryDestinations(report.BridgeLinks),
		BridgeOutflowAmount:          formatFloatAmount(bridgeOutflowAmount),
		BridgeOutflowShare:           ratio(bridgeOutflowAmount, totalOutboundAmount),
		BridgeRecurrenceDays:         len(bridgeDays),
		LatestBridgeTxHash:           latestBridgeTxHash(report.BridgeLinks),
	}
	report.ExchangeFeatures = WalletExchangeFlowFeatures{
		ExchangeOutboundCount:          len(report.ExchangePaths),
		DistinctExchangeCounterparties: len(exchangeCounterparties),
		DepositLikePathCount:           countDepositLikePaths(report.ExchangePaths),
		ExchangeFanoutCount:            len(exchangeFanout),
		FreshRecipientCount:            countFreshExchangeRecipients(report.ExchangePaths),
		ExchangeOutflowAmount:          formatFloatAmount(exchangeOutflowAmount),
		ExchangeOutflowShare:           ratio(exchangeOutflowAmount, totalOutboundAmount),
		ExchangeRecurrenceDays:         len(exchangeDays),
		LatestExchangeTxHash:           latestExchangeTxHash(report.ExchangePaths),
	}

	sort.SliceStable(report.BridgeLinks, func(i, j int) bool {
		return report.BridgeLinks[i].ObservedAt.After(report.BridgeLinks[j].ObservedAt)
	})
	sort.SliceStable(report.ExchangePaths, func(i, j int) bool {
		return report.ExchangePaths[i].ObservedAt.After(report.ExchangePaths[j].ObservedAt)
	})

	return report, nil
}

func (s *PostgresWalletBridgeExchangeEvidenceStore) ReplaceWalletBridgeExchangeEvidence(
	ctx context.Context,
	report WalletBridgeExchangeEvidenceReport,
) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("wallet bridge/exchange evidence store is nil")
	}
	walletID := strings.TrimSpace(report.WalletID)
	if walletID == "" {
		return fmt.Errorf("wallet id is required")
	}
	observedDay := report.WindowEndAt.UTC().Format("2006-01-02")
	now := s.now().UTC()
	if _, err := s.Execer.Exec(ctx, deleteWalletBridgeLinksForDaySQL, walletID, observedDay); err != nil {
		return fmt.Errorf("delete wallet bridge links: %w", err)
	}
	if _, err := s.Execer.Exec(ctx, deleteWalletExchangePathsForDaySQL, walletID, observedDay); err != nil {
		return fmt.Errorf("delete wallet exchange paths: %w", err)
	}

	bridgeFeatureMetadata, err := json.Marshal(map[string]any{
		"wallet_id":                        walletID,
		"confirmed_destination_count":      report.BridgeFeatures.ConfirmedDestinationCount,
		"post_bridge_exchange_touch_count": report.BridgeFeatures.PostBridgeExchangeTouchCount,
		"post_bridge_protocol_entry_count": report.BridgeFeatures.PostBridgeProtocolEntryCount,
	})
	if err != nil {
		return fmt.Errorf("marshal bridge features metadata: %w", err)
	}
	if _, err := s.Execer.Exec(
		ctx,
		upsertWalletBridgeFeaturesDailySQL,
		walletID,
		observedDay,
		report.WindowStartAt.UTC(),
		report.WindowEndAt.UTC(),
		report.BridgeFeatures.BridgeOutboundCount,
		report.BridgeFeatures.DistinctBridgeCounterparties,
		report.BridgeFeatures.DistinctBridgeProtocols,
		report.BridgeFeatures.ConfirmedDestinationCount,
		report.BridgeFeatures.PostBridgeFreshWalletCount,
		report.BridgeFeatures.PostBridgeExchangeTouchCount,
		report.BridgeFeatures.PostBridgeProtocolEntryCount,
		report.BridgeFeatures.BridgeOutflowAmount,
		report.BridgeFeatures.BridgeOutflowShare,
		report.BridgeFeatures.BridgeRecurrenceDays,
		report.BridgeFeatures.LatestBridgeTxHash,
		bridgeFeatureMetadata,
		now,
	); err != nil {
		return fmt.Errorf("upsert wallet bridge features: %w", err)
	}

	exchangeFeatureMetadata, err := json.Marshal(map[string]any{
		"wallet_id":               walletID,
		"deposit_like_path_count": report.ExchangeFeatures.DepositLikePathCount,
		"exchange_fanout_count":   report.ExchangeFeatures.ExchangeFanoutCount,
	})
	if err != nil {
		return fmt.Errorf("marshal exchange features metadata: %w", err)
	}
	if _, err := s.Execer.Exec(
		ctx,
		upsertWalletExchangeFeaturesDailySQL,
		walletID,
		observedDay,
		report.WindowStartAt.UTC(),
		report.WindowEndAt.UTC(),
		report.ExchangeFeatures.ExchangeOutboundCount,
		report.ExchangeFeatures.DistinctExchangeCounterparties,
		report.ExchangeFeatures.DepositLikePathCount,
		report.ExchangeFeatures.ExchangeFanoutCount,
		report.ExchangeFeatures.FreshRecipientCount,
		report.ExchangeFeatures.ExchangeOutflowAmount,
		report.ExchangeFeatures.ExchangeOutflowShare,
		report.ExchangeFeatures.ExchangeRecurrenceDays,
		report.ExchangeFeatures.LatestExchangeTxHash,
		exchangeFeatureMetadata,
		now,
	); err != nil {
		return fmt.Errorf("upsert wallet exchange features: %w", err)
	}

	for _, item := range report.BridgeLinks {
		metadata, err := json.Marshal(map[string]any{
			"fresh_destination": item.FreshDestination,
			"txRef": map[string]any{
				"chain":      string(report.Chain),
				"address":    report.Address,
				"txHash":     item.TxHash,
				"observedAt": item.ObservedAt.UTC().Format(time.RFC3339),
			},
			"pathRef": map[string]any{
				"kind":               "bridge_link_confirmation",
				"bridgeChain":        string(item.BridgeChain),
				"bridgeAddress":      item.BridgeAddress,
				"destinationChain":   string(item.DestinationChain),
				"destinationAddress": item.DestinationAddress,
				"destinationTxHash":  item.DestinationTxHash,
			},
		})
		if err != nil {
			return fmt.Errorf("marshal wallet bridge link metadata: %w", err)
		}
		if _, err := s.Execer.Exec(
			ctx,
			insertWalletBridgeLinkSQL,
			walletID,
			observedDay,
			item.TxHash,
			item.ObservedAt.UTC(),
			string(item.BridgeChain),
			item.BridgeAddress,
			item.BridgeLabel,
			item.BridgeEntityKey,
			item.BridgeEntityType,
			item.Amount,
			item.TokenSymbol,
			string(item.DestinationChain),
			item.DestinationAddress,
			item.DestinationLabel,
			item.DestinationEntityKey,
			item.DestinationEntityType,
			item.DestinationTxHash,
			item.DestinationObservedAt,
			item.Confidence,
			metadata,
			now,
		); err != nil {
			return fmt.Errorf("insert wallet bridge link: %w", err)
		}
	}

	for _, item := range report.ExchangePaths {
		metadata, err := json.Marshal(map[string]any{
			"fresh_recipient": item.FreshRecipient,
			"txRef": map[string]any{
				"chain":      string(report.Chain),
				"address":    report.Address,
				"txHash":     item.TxHash,
				"observedAt": item.ObservedAt.UTC().Format(time.RFC3339),
			},
			"pathRef": map[string]any{
				"kind":                item.PathKind,
				"intermediaryChain":   string(item.IntermediaryChain),
				"intermediaryAddress": item.IntermediaryAddress,
				"exchangeChain":       string(item.ExchangeChain),
				"exchangeAddress":     item.ExchangeAddress,
				"exchangeTxHash":      item.ExchangeTxHash,
			},
		})
		if err != nil {
			return fmt.Errorf("marshal wallet exchange path metadata: %w", err)
		}
		if _, err := s.Execer.Exec(
			ctx,
			insertWalletExchangePathSQL,
			walletID,
			observedDay,
			item.TxHash,
			item.ObservedAt.UTC(),
			item.PathKind,
			string(item.IntermediaryChain),
			item.IntermediaryAddress,
			item.IntermediaryLabel,
			item.IntermediaryEntityKey,
			item.IntermediaryEntityType,
			string(item.ExchangeChain),
			item.ExchangeAddress,
			item.ExchangeLabel,
			item.ExchangeEntityKey,
			item.ExchangeEntityType,
			item.ExchangeTxHash,
			item.ExchangeObservedAt,
			item.Amount,
			item.TokenSymbol,
			item.Confidence,
			metadata,
			now,
		); err != nil {
			return fmt.Errorf("insert wallet exchange path: %w", err)
		}
	}

	return nil
}

func (s *PostgresWalletBridgeExchangeEvidenceStore) readIdentity(
	ctx context.Context,
	ref WalletRef,
) (walletBridgeExchangeIdentity, error) {
	var identity walletBridgeExchangeIdentity
	if err := s.Querier.QueryRow(
		ctx,
		walletBridgeExchangeIdentitySQL,
		string(ref.Chain),
		ref.Address,
	).Scan(
		&identity.WalletID,
		&identity.Chain,
		&identity.Address,
		&identity.DisplayName,
	); err != nil {
		return walletBridgeExchangeIdentity{}, fmt.Errorf("read wallet bridge/exchange identity: %w", err)
	}
	return identity, nil
}

func (s *PostgresWalletBridgeExchangeEvidenceStore) buildBridgeObservation(
	ctx context.Context,
	rootChain domain.Chain,
	tx walletBridgeExchangeTransactionRow,
) (WalletBridgeLinkObservation, bool, error) {
	counterpartyAddress := strings.TrimSpace(tx.CounterpartyAddress)
	if counterpartyAddress == "" {
		return WalletBridgeLinkObservation{}, false, nil
	}
	counterpartyChain := normalizeCandidateChain(tx.CounterpartyChain, rootChain)
	downstream, err := s.lookupDownstreamTransactions(
		ctx,
		counterpartyChain,
		counterpartyAddress,
		tx.ObservedAt.UTC(),
		tx.ObservedAt.UTC().Add(6*time.Hour),
	)
	if err != nil {
		return WalletBridgeLinkObservation{}, false, err
	}
	observation := WalletBridgeLinkObservation{
		TxHash:           strings.TrimSpace(tx.TxHash),
		ObservedAt:       tx.ObservedAt.UTC(),
		BridgeChain:      counterpartyChain,
		BridgeAddress:    counterpartyAddress,
		BridgeLabel:      firstNonEmpty(strings.TrimSpace(tx.CounterpartyDisplayName), counterpartyAddress),
		BridgeEntityKey:  strings.TrimSpace(tx.CounterpartyEntityKey),
		BridgeEntityType: strings.TrimSpace(tx.CounterpartyEntityType),
		Amount:           strings.TrimSpace(tx.Amount),
		TokenSymbol:      strings.TrimSpace(tx.TokenSymbol),
		Confidence:       0.72,
	}

	if len(downstream) == 0 {
		return observation, true, nil
	}

	next := downstream[0]
	observation.DestinationChain = normalizeCandidateChain(next.CounterpartyChain, counterpartyChain)
	observation.DestinationAddress = strings.TrimSpace(next.CounterpartyAddress)
	observation.DestinationLabel = firstNonEmpty(strings.TrimSpace(next.CounterpartyDisplayName), observation.DestinationAddress)
	observation.DestinationEntityKey = strings.TrimSpace(next.CounterpartyEntityKey)
	observation.DestinationEntityType = strings.TrimSpace(next.CounterpartyEntityType)
	observation.DestinationTxHash = strings.TrimSpace(next.TxHash)
	destinationObservedAt := next.ObservedAt.UTC()
	observation.DestinationObservedAt = &destinationObservedAt
	observation.Confidence = 0.84
	if observation.DestinationAddress != "" {
		fresh, err := s.isFreshAddress(ctx, observation.DestinationChain, observation.DestinationAddress, tx.ObservedAt.UTC())
		if err != nil {
			return WalletBridgeLinkObservation{}, false, err
		}
		observation.FreshDestination = fresh
	}
	return observation, true, nil
}

func (s *PostgresWalletBridgeExchangeEvidenceStore) buildExchangeObservation(
	ctx context.Context,
	rootChain domain.Chain,
	tx walletBridgeExchangeTransactionRow,
) (WalletExchangePathObservation, bool, error) {
	counterpartyAddress := strings.TrimSpace(tx.CounterpartyAddress)
	if counterpartyAddress == "" {
		return WalletExchangePathObservation{}, false, nil
	}
	counterpartyChain := normalizeCandidateChain(tx.CounterpartyChain, rootChain)
	downstream, err := s.lookupDownstreamTransactions(
		ctx,
		counterpartyChain,
		counterpartyAddress,
		tx.ObservedAt.UTC(),
		tx.ObservedAt.UTC().Add(2*time.Hour),
	)
	if err != nil {
		return WalletExchangePathObservation{}, false, err
	}
	for _, next := range downstream {
		joinedText := strings.Join([]string{
			strings.TrimSpace(next.CounterpartyDisplayName),
			strings.TrimSpace(next.CounterpartyEntityType),
			strings.TrimSpace(next.CounterpartyEntityKey),
		}, " ")
		if !classifyShadowExitCEX(joinedText) {
			continue
		}
		observation := WalletExchangePathObservation{
			TxHash:                 strings.TrimSpace(tx.TxHash),
			ObservedAt:             tx.ObservedAt.UTC(),
			PathKind:               "intermediary_exchange_path",
			IntermediaryChain:      counterpartyChain,
			IntermediaryAddress:    counterpartyAddress,
			IntermediaryLabel:      firstNonEmpty(strings.TrimSpace(tx.CounterpartyDisplayName), counterpartyAddress),
			IntermediaryEntityKey:  strings.TrimSpace(tx.CounterpartyEntityKey),
			IntermediaryEntityType: strings.TrimSpace(tx.CounterpartyEntityType),
			ExchangeChain:          normalizeCandidateChain(next.CounterpartyChain, counterpartyChain),
			ExchangeAddress:        strings.TrimSpace(next.CounterpartyAddress),
			ExchangeLabel:          firstNonEmpty(strings.TrimSpace(next.CounterpartyDisplayName), strings.TrimSpace(next.CounterpartyAddress)),
			ExchangeEntityKey:      strings.TrimSpace(next.CounterpartyEntityKey),
			ExchangeEntityType:     strings.TrimSpace(next.CounterpartyEntityType),
			ExchangeTxHash:         strings.TrimSpace(next.TxHash),
			Amount:                 strings.TrimSpace(tx.Amount),
			TokenSymbol:            strings.TrimSpace(tx.TokenSymbol),
			Confidence:             0.81,
		}
		exchangeObservedAt := next.ObservedAt.UTC()
		observation.ExchangeObservedAt = &exchangeObservedAt
		if observation.IntermediaryAddress != "" {
			fresh, err := s.isFreshAddress(ctx, observation.IntermediaryChain, observation.IntermediaryAddress, tx.ObservedAt.UTC())
			if err != nil {
				return WalletExchangePathObservation{}, false, err
			}
			observation.FreshRecipient = fresh
		}
		return observation, true, nil
	}
	return WalletExchangePathObservation{}, false, nil
}

func (s *PostgresWalletBridgeExchangeEvidenceStore) lookupDownstreamTransactions(
	ctx context.Context,
	chain domain.Chain,
	address string,
	start time.Time,
	end time.Time,
) ([]walletBridgeExchangeTransactionRow, error) {
	rows, err := s.Querier.Query(
		ctx,
		walletBridgeExchangeDownstreamSQL,
		string(chain),
		strings.TrimSpace(address),
		start,
		end,
	)
	if err != nil {
		return nil, fmt.Errorf("query downstream transactions: %w", err)
	}
	defer rows.Close()

	out := make([]walletBridgeExchangeTransactionRow, 0, 8)
	for rows.Next() {
		var item walletBridgeExchangeTransactionRow
		if err := rows.Scan(
			&item.TxHash,
			&item.ObservedAt,
			&item.CounterpartyChain,
			&item.CounterpartyAddress,
			&item.CounterpartyDisplayName,
			&item.CounterpartyEntityKey,
			&item.CounterpartyEntityType,
		); err != nil {
			return nil, fmt.Errorf("scan downstream transaction: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate downstream transactions: %w", err)
	}
	return out, nil
}

func (s *PostgresWalletBridgeExchangeEvidenceStore) isFreshAddress(
	ctx context.Context,
	chain domain.Chain,
	address string,
	before time.Time,
) (bool, error) {
	var exists bool
	if err := s.Querier.QueryRow(
		ctx,
		walletAddressPriorActivityExistsSQL,
		string(chain),
		strings.TrimSpace(address),
		before,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("lookup prior address activity: %w", err)
	}
	return !exists, nil
}

func countBridgeConfirmedDestinations(items []WalletBridgeLinkObservation) int {
	count := 0
	for _, item := range items {
		if strings.TrimSpace(item.DestinationAddress) != "" {
			count++
		}
	}
	return count
}

func countBridgeFreshDestinations(items []WalletBridgeLinkObservation) int {
	count := 0
	for _, item := range items {
		if item.FreshDestination {
			count++
		}
	}
	return count
}

func countBridgeExchangeTouchDestinations(items []WalletBridgeLinkObservation) int {
	count := 0
	for _, item := range items {
		if classifyShadowExitCEX(strings.Join([]string{item.DestinationLabel, item.DestinationEntityKey, item.DestinationEntityType}, " ")) {
			count++
		}
	}
	return count
}

func countBridgeProtocolEntryDestinations(items []WalletBridgeLinkObservation) int {
	count := 0
	for _, item := range items {
		if classifyProtocolLike(item.DestinationLabel, item.DestinationEntityType, item.DestinationEntityKey) {
			count++
		}
	}
	return count
}

func latestBridgeTxHash(items []WalletBridgeLinkObservation) string {
	for _, item := range items {
		if strings.TrimSpace(item.TxHash) != "" {
			return strings.TrimSpace(item.TxHash)
		}
	}
	return ""
}

func countDepositLikePaths(items []WalletExchangePathObservation) int {
	count := 0
	for _, item := range items {
		if strings.TrimSpace(item.PathKind) != "" {
			count++
		}
	}
	return count
}

func countFreshExchangeRecipients(items []WalletExchangePathObservation) int {
	count := 0
	for _, item := range items {
		if item.FreshRecipient {
			count++
		}
	}
	return count
}

func latestExchangeTxHash(items []WalletExchangePathObservation) string {
	for _, item := range items {
		if strings.TrimSpace(item.TxHash) != "" {
			return strings.TrimSpace(item.TxHash)
		}
	}
	return ""
}

func formatFloatAmount(value float64) string {
	if value == 0 {
		return "0"
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func numericStringToFloat(raw string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0
	}
	return parsed
}

func ratio(part float64, total float64) float64 {
	if part <= 0 || total <= 0 {
		return 0
	}
	return part / total
}

func classifyProtocolLike(parts ...string) bool {
	joined := strings.ToLower(strings.Join(parts, " "))
	return strings.Contains(joined, "protocol") ||
		strings.Contains(joined, "dex") ||
		strings.Contains(joined, "swap") ||
		strings.Contains(joined, "pool")
}

func (s *PostgresWalletBridgeExchangeEvidenceStore) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
