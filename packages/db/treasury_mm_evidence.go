package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

const walletTreasuryMMTransactionsSQL = `
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
  AND t.observed_at >= $3
ORDER BY t.observed_at DESC, t.id DESC
`

const deleteWalletTreasuryPathsForDaySQL = `
DELETE FROM wallet_treasury_paths
WHERE wallet_id = $1
  AND observed_day = $2
`

const deleteWalletMMPathsForDaySQL = `
DELETE FROM wallet_mm_paths
WHERE wallet_id = $1
  AND observed_day = $2
`

const upsertWalletTreasuryFeaturesDailySQL = `
INSERT INTO wallet_treasury_features_daily (
  wallet_id,
  observed_day,
  window_start_at,
  window_end_at,
  anchor_match_count,
  fanout_signature_count,
  operational_distribution_count,
  rebalance_discount_count,
  treasury_to_market_path_count,
  treasury_to_exchange_path_count,
  treasury_to_bridge_path_count,
  treasury_to_mm_path_count,
  distinct_market_counterparty_count,
  operational_only_distribution_count,
  internal_ops_distribution_count,
  external_ops_distribution_count,
  external_market_adjacent_distribution_count,
  external_non_market_distribution_count,
  latest_treasury_tx_hash,
  metadata,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
)
ON CONFLICT (wallet_id, observed_day) DO UPDATE SET
  window_start_at = EXCLUDED.window_start_at,
  window_end_at = EXCLUDED.window_end_at,
  anchor_match_count = EXCLUDED.anchor_match_count,
  fanout_signature_count = EXCLUDED.fanout_signature_count,
  operational_distribution_count = EXCLUDED.operational_distribution_count,
  rebalance_discount_count = EXCLUDED.rebalance_discount_count,
  treasury_to_market_path_count = EXCLUDED.treasury_to_market_path_count,
  treasury_to_exchange_path_count = EXCLUDED.treasury_to_exchange_path_count,
  treasury_to_bridge_path_count = EXCLUDED.treasury_to_bridge_path_count,
  treasury_to_mm_path_count = EXCLUDED.treasury_to_mm_path_count,
  distinct_market_counterparty_count = EXCLUDED.distinct_market_counterparty_count,
  operational_only_distribution_count = EXCLUDED.operational_only_distribution_count,
  internal_ops_distribution_count = EXCLUDED.internal_ops_distribution_count,
  external_ops_distribution_count = EXCLUDED.external_ops_distribution_count,
  external_market_adjacent_distribution_count = EXCLUDED.external_market_adjacent_distribution_count,
  external_non_market_distribution_count = EXCLUDED.external_non_market_distribution_count,
  latest_treasury_tx_hash = EXCLUDED.latest_treasury_tx_hash,
  metadata = EXCLUDED.metadata,
  updated_at = EXCLUDED.updated_at
`

const upsertWalletMMFeaturesDailySQL = `
INSERT INTO wallet_mm_features_daily (
  wallet_id,
  observed_day,
  window_start_at,
  window_end_at,
  mm_anchor_match_count,
  inventory_rotation_count,
  project_to_mm_path_count,
  post_handoff_distribution_count,
  post_handoff_exchange_touch_count,
  post_handoff_bridge_touch_count,
  project_to_mm_contact_count,
  project_to_mm_routed_candidate_count,
  project_to_mm_adjacency_count,
  repeat_mm_counterparty_count,
  latest_mm_tx_hash,
  metadata,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
)
ON CONFLICT (wallet_id, observed_day) DO UPDATE SET
  window_start_at = EXCLUDED.window_start_at,
  window_end_at = EXCLUDED.window_end_at,
  mm_anchor_match_count = EXCLUDED.mm_anchor_match_count,
  inventory_rotation_count = EXCLUDED.inventory_rotation_count,
  project_to_mm_path_count = EXCLUDED.project_to_mm_path_count,
  post_handoff_distribution_count = EXCLUDED.post_handoff_distribution_count,
  post_handoff_exchange_touch_count = EXCLUDED.post_handoff_exchange_touch_count,
  post_handoff_bridge_touch_count = EXCLUDED.post_handoff_bridge_touch_count,
  project_to_mm_contact_count = EXCLUDED.project_to_mm_contact_count,
  project_to_mm_routed_candidate_count = EXCLUDED.project_to_mm_routed_candidate_count,
  project_to_mm_adjacency_count = EXCLUDED.project_to_mm_adjacency_count,
  repeat_mm_counterparty_count = EXCLUDED.repeat_mm_counterparty_count,
  latest_mm_tx_hash = EXCLUDED.latest_mm_tx_hash,
  metadata = EXCLUDED.metadata,
  updated_at = EXCLUDED.updated_at
`

const insertWalletTreasuryPathSQL = `
INSERT INTO wallet_treasury_paths (
  wallet_id,
  observed_day,
  tx_hash,
  observed_at,
  path_kind,
  counterparty_chain,
  counterparty_address,
  counterparty_label,
  counterparty_entity_key,
  counterparty_entity_type,
  downstream_chain,
  downstream_address,
  downstream_label,
  downstream_entity_key,
  downstream_entity_type,
  downstream_tx_hash,
  downstream_observed_at,
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
ON CONFLICT (wallet_id, tx_hash, path_kind, counterparty_chain, counterparty_address, downstream_tx_hash, observed_at) DO UPDATE SET
  counterparty_label = EXCLUDED.counterparty_label,
  counterparty_entity_key = EXCLUDED.counterparty_entity_key,
  counterparty_entity_type = EXCLUDED.counterparty_entity_type,
  downstream_chain = EXCLUDED.downstream_chain,
  downstream_address = EXCLUDED.downstream_address,
  downstream_label = EXCLUDED.downstream_label,
  downstream_entity_key = EXCLUDED.downstream_entity_key,
  downstream_entity_type = EXCLUDED.downstream_entity_type,
  downstream_observed_at = EXCLUDED.downstream_observed_at,
  amount_numeric = EXCLUDED.amount_numeric,
  token_symbol = EXCLUDED.token_symbol,
  confidence = EXCLUDED.confidence,
  metadata = EXCLUDED.metadata,
  updated_at = EXCLUDED.updated_at
`

const insertWalletMMPathSQL = `
INSERT INTO wallet_mm_paths (
  wallet_id,
  observed_day,
  tx_hash,
  observed_at,
  path_kind,
  counterparty_chain,
  counterparty_address,
  counterparty_label,
  counterparty_entity_key,
  counterparty_entity_type,
  downstream_chain,
  downstream_address,
  downstream_label,
  downstream_entity_key,
  downstream_entity_type,
  downstream_tx_hash,
  downstream_observed_at,
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
ON CONFLICT (wallet_id, tx_hash, path_kind, counterparty_chain, counterparty_address, downstream_tx_hash, observed_at) DO UPDATE SET
  counterparty_label = EXCLUDED.counterparty_label,
  counterparty_entity_key = EXCLUDED.counterparty_entity_key,
  counterparty_entity_type = EXCLUDED.counterparty_entity_type,
  downstream_chain = EXCLUDED.downstream_chain,
  downstream_address = EXCLUDED.downstream_address,
  downstream_label = EXCLUDED.downstream_label,
  downstream_entity_key = EXCLUDED.downstream_entity_key,
  downstream_entity_type = EXCLUDED.downstream_entity_type,
  downstream_observed_at = EXCLUDED.downstream_observed_at,
  amount_numeric = EXCLUDED.amount_numeric,
  token_symbol = EXCLUDED.token_symbol,
  confidence = EXCLUDED.confidence,
  metadata = EXCLUDED.metadata,
  updated_at = EXCLUDED.updated_at
`

type WalletTreasuryMMEvidenceReader interface {
	ReadWalletTreasuryMMEvidence(context.Context, WalletRef, time.Duration) (WalletTreasuryMMEvidenceReport, error)
}

type WalletTreasuryMMEvidenceStore interface {
	ReplaceWalletTreasuryMMEvidence(context.Context, WalletTreasuryMMEvidenceReport) error
}

type WalletTreasuryMMEvidenceReadWriter interface {
	WalletTreasuryMMEvidenceReader
	WalletTreasuryMMEvidenceStore
}

type WalletTreasuryPathObservation struct {
	TxHash                 string
	ObservedAt             time.Time
	PathKind               string
	CounterpartyChain      domain.Chain
	CounterpartyAddress    string
	CounterpartyLabel      string
	CounterpartyEntityKey  string
	CounterpartyEntityType string
	DownstreamChain        domain.Chain
	DownstreamAddress      string
	DownstreamLabel        string
	DownstreamEntityKey    string
	DownstreamEntityType   string
	DownstreamTxHash       string
	DownstreamObservedAt   *time.Time
	Amount                 string
	TokenSymbol            string
	Confidence             float64
}

type WalletMMPathObservation struct {
	TxHash                 string
	ObservedAt             time.Time
	PathKind               string
	CounterpartyChain      domain.Chain
	CounterpartyAddress    string
	CounterpartyLabel      string
	CounterpartyEntityKey  string
	CounterpartyEntityType string
	DownstreamChain        domain.Chain
	DownstreamAddress      string
	DownstreamLabel        string
	DownstreamEntityKey    string
	DownstreamEntityType   string
	DownstreamTxHash       string
	DownstreamObservedAt   *time.Time
	Amount                 string
	TokenSymbol            string
	Confidence             float64
}

type WalletTreasuryFeatures struct {
	AnchorMatchCount                 int
	FanoutSignatureCount             int
	OperationalDistributionCount     int
	RebalanceDiscountCount           int
	TreasuryToMarketPathCount        int
	TreasuryToExchangePathCount      int
	TreasuryToBridgePathCount        int
	TreasuryToMMPathCount            int
	DistinctMarketCounterpartyCount  int
	OperationalOnlyDistributionCount int
	InternalOpsDistributionCount     int
	ExternalOpsDistributionCount     int
	ExternalMarketAdjacentCount      int
	ExternalNonMarketCount           int
	LatestTreasuryTxHash             string
}

type WalletMMFeatures struct {
	MMAnchorMatchCount              int
	InventoryRotationCount          int
	ProjectToMMPathCount            int
	PostHandoffDistributionCount    int
	PostHandoffExchangeTouchCount   int
	PostHandoffBridgeTouchCount     int
	ProjectToMMContactCount         int
	ProjectToMMRoutedCandidateCount int
	ProjectToMMAdjacencyCount       int
	RepeatMMCounterpartyCount       int
	LatestMMTxHash                  string
}

type WalletTreasuryMMEvidenceReport struct {
	WalletID         string
	Chain            domain.Chain
	Address          string
	DisplayName      string
	WindowStartAt    time.Time
	WindowEndAt      time.Time
	HasTreasuryLabel bool
	HasFundLabel     bool
	TreasuryPaths    []WalletTreasuryPathObservation
	MMPaths          []WalletMMPathObservation
	TreasuryFeatures WalletTreasuryFeatures
	MMFeatures       WalletMMFeatures
}

type treasuryMMTransactionRow struct {
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

type PostgresWalletTreasuryMMEvidenceStore struct {
	Querier postgresQuerier
	Execer  postgresFindingExecer
	Labels  WalletLabelReader
	Now     func() time.Time
}

func NewPostgresWalletTreasuryMMEvidenceStore(
	querier postgresQuerier,
	execer postgresFindingExecer,
	labels WalletLabelReader,
) *PostgresWalletTreasuryMMEvidenceStore {
	return &PostgresWalletTreasuryMMEvidenceStore{
		Querier: querier,
		Execer:  execer,
		Labels:  labels,
		Now:     time.Now,
	}
}

func NewPostgresWalletTreasuryMMEvidenceStoreFromPool(pool interface {
	postgresQuerier
	postgresFindingExecer
}, labels WalletLabelReader) *PostgresWalletTreasuryMMEvidenceStore {
	return NewPostgresWalletTreasuryMMEvidenceStore(pool, pool, labels)
}

func (s *PostgresWalletTreasuryMMEvidenceStore) ReadWalletTreasuryMMEvidence(
	ctx context.Context,
	ref WalletRef,
	window time.Duration,
) (WalletTreasuryMMEvidenceReport, error) {
	if s == nil || s.Querier == nil {
		return WalletTreasuryMMEvidenceReport{}, fmt.Errorf("wallet treasury/mm evidence store is nil")
	}
	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return WalletTreasuryMMEvidenceReport{}, err
	}
	if window <= 0 {
		window = 24 * time.Hour
	}
	windowEnd := s.now().UTC()
	windowStart := windowEnd.Add(-window)
	identity, err := (&PostgresWalletBridgeExchangeEvidenceStore{Querier: s.Querier, Now: s.now}).readIdentity(ctx, normalized)
	if err != nil {
		return WalletTreasuryMMEvidenceReport{}, err
	}

	labels, err := readTreasuryMMRootLabels(ctx, s.Labels, normalized)
	if err != nil {
		return WalletTreasuryMMEvidenceReport{}, err
	}

	rows, err := s.Querier.Query(ctx, walletTreasuryMMTransactionsSQL, identity.WalletID, string(normalized.Chain), windowStart)
	if err != nil {
		return WalletTreasuryMMEvidenceReport{}, fmt.Errorf("list wallet treasury/mm transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]treasuryMMTransactionRow, 0, 32)
	for rows.Next() {
		var item treasuryMMTransactionRow
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
			return WalletTreasuryMMEvidenceReport{}, fmt.Errorf("scan wallet treasury/mm transaction: %w", err)
		}
		transactions = append(transactions, item)
	}
	if err := rows.Err(); err != nil {
		return WalletTreasuryMMEvidenceReport{}, fmt.Errorf("iterate wallet treasury/mm transactions: %w", err)
	}

	report := WalletTreasuryMMEvidenceReport{
		WalletID:         identity.WalletID,
		Chain:            identity.Chain,
		Address:          identity.Address,
		DisplayName:      identity.DisplayName,
		WindowStartAt:    windowStart,
		WindowEndAt:      windowEnd,
		HasTreasuryLabel: labels.hasTreasury,
		HasFundLabel:     labels.hasFund,
	}

	type directionalCount struct {
		inboundDays  map[string]struct{}
		outboundDays map[string]struct{}
	}
	mmDirection := map[string]*directionalCount{}
	outboundCounterparties := map[string]struct{}{}
	outboundByDay := map[string]int{}
	treasuryAnchorCounterparties := map[string]struct{}{}
	treasuryMarketCounterparties := map[string]struct{}{}
	mmCounterparties := map[string]map[string]struct{}{}
	treasuryMarketPathCount := 0
	treasuryToExchangePathCount := 0
	treasuryToBridgePathCount := 0
	treasuryToMMPathCount := 0
	operationalOnlyDistributionCount := 0
	internalOpsDistributionCount := 0
	externalOpsDistributionCount := 0
	externalMarketAdjacentCount := 0
	externalNonMarketCount := 0
	rebalanceDiscountCount := 0

	for _, tx := range transactions {
		counterpartyAddress := strings.TrimSpace(tx.CounterpartyAddress)
		if counterpartyAddress == "" {
			continue
		}
		counterpartyChain := normalizeCandidateChain(tx.CounterpartyChain, normalized.Chain)
		dayKey := tx.ObservedAt.UTC().Format("2006-01-02")
		counterpartyKey := shadowExitCandidateCanonicalKey(counterpartyChain, counterpartyAddress)
		joinedText := strings.Join([]string{
			strings.TrimSpace(tx.CounterpartyDisplayName),
			strings.TrimSpace(tx.CounterpartyEntityType),
			strings.TrimSpace(tx.CounterpartyEntityKey),
		}, " ")

		if classifyTreasuryMMMarketMaker(joinedText) {
			dir := mmDirection[counterpartyKey]
			if dir == nil {
				dir = &directionalCount{inboundDays: map[string]struct{}{}, outboundDays: map[string]struct{}{}}
				mmDirection[counterpartyKey] = dir
			}
			if strings.EqualFold(strings.TrimSpace(tx.Direction), string(domain.TransactionDirectionOutbound)) {
				dir.outboundDays[dayKey] = struct{}{}
			} else if strings.EqualFold(strings.TrimSpace(tx.Direction), string(domain.TransactionDirectionInbound)) {
				dir.inboundDays[dayKey] = struct{}{}
			}
		}

		if !strings.EqualFold(strings.TrimSpace(tx.Direction), string(domain.TransactionDirectionOutbound)) {
			continue
		}

		outboundCounterparties[counterpartyKey] = struct{}{}
		outboundByDay[dayKey]++

		if classifyShadowExitTreasury(joinedText) {
			treasuryAnchorCounterparties[counterpartyKey] = struct{}{}
			report.TreasuryPaths = append(report.TreasuryPaths, WalletTreasuryPathObservation{
				TxHash:                 strings.TrimSpace(tx.TxHash),
				ObservedAt:             tx.ObservedAt.UTC(),
				PathKind:               "treasury_anchor_match",
				CounterpartyChain:      counterpartyChain,
				CounterpartyAddress:    counterpartyAddress,
				CounterpartyLabel:      firstNonEmpty(strings.TrimSpace(tx.CounterpartyDisplayName), counterpartyAddress),
				CounterpartyEntityKey:  strings.TrimSpace(tx.CounterpartyEntityKey),
				CounterpartyEntityType: strings.TrimSpace(tx.CounterpartyEntityType),
				Amount:                 strings.TrimSpace(tx.Amount),
				TokenSymbol:            strings.TrimSpace(tx.TokenSymbol),
				Confidence:             0.84,
			})
		}
		if classifyShadowExitTreasury(joinedText) || classifyShadowExitInternal(joinedText) {
			rebalanceDiscountCount++
		}
		marketPathKind, marketPathConfidence, ok := classifyTreasuryMarketPath(joinedText)
		if ok {
			treasuryMarketPathCount++
			treasuryMarketCounterparties[counterpartyKey] = struct{}{}
			switch marketPathKind {
			case "treasury_to_exchange_path":
				treasuryToExchangePathCount++
			case "treasury_to_bridge_path":
				treasuryToBridgePathCount++
			case "treasury_to_mm_path":
				treasuryToMMPathCount++
			}
			report.TreasuryPaths = append(report.TreasuryPaths, WalletTreasuryPathObservation{
				TxHash:                 strings.TrimSpace(tx.TxHash),
				ObservedAt:             tx.ObservedAt.UTC(),
				PathKind:               marketPathKind,
				CounterpartyChain:      counterpartyChain,
				CounterpartyAddress:    counterpartyAddress,
				CounterpartyLabel:      firstNonEmpty(strings.TrimSpace(tx.CounterpartyDisplayName), counterpartyAddress),
				CounterpartyEntityKey:  strings.TrimSpace(tx.CounterpartyEntityKey),
				CounterpartyEntityType: strings.TrimSpace(tx.CounterpartyEntityType),
				Amount:                 strings.TrimSpace(tx.Amount),
				TokenSymbol:            strings.TrimSpace(tx.TokenSymbol),
				Confidence:             marketPathConfidence,
			})
		} else if !classifyShadowExitTreasury(joinedText) {
			operationalOnlyDistributionCount++
			pathKind := "treasury_external_non_market_ops"
			confidence := 0.58
			if classifyShadowExitInternal(joinedText) {
				pathKind = "treasury_internal_ops_distribution"
				confidence = 0.64
				internalOpsDistributionCount++
			} else {
				externalOpsDistributionCount++
				if subtype := classifyTreasuryExternalMarketAdjacentType(joinedText); subtype != "" {
					pathKind = "treasury_external_market_adjacent_direct_" + subtype
					confidence = treasuryExternalMarketAdjacentConfidence(subtype, false)
					externalMarketAdjacentCount++
				} else if observation, found, err := s.buildTreasuryExternalOpsObservation(ctx, normalized.Chain, tx); err != nil {
					return WalletTreasuryMMEvidenceReport{}, err
				} else if found {
					report.TreasuryPaths = append(report.TreasuryPaths, observation)
					externalMarketAdjacentCount++
					continue
				} else {
					externalNonMarketCount++
				}
			}
			report.TreasuryPaths = append(report.TreasuryPaths, WalletTreasuryPathObservation{
				TxHash:                 strings.TrimSpace(tx.TxHash),
				ObservedAt:             tx.ObservedAt.UTC(),
				PathKind:               pathKind,
				CounterpartyChain:      counterpartyChain,
				CounterpartyAddress:    counterpartyAddress,
				CounterpartyLabel:      firstNonEmpty(strings.TrimSpace(tx.CounterpartyDisplayName), counterpartyAddress),
				CounterpartyEntityKey:  strings.TrimSpace(tx.CounterpartyEntityKey),
				CounterpartyEntityType: strings.TrimSpace(tx.CounterpartyEntityType),
				Amount:                 strings.TrimSpace(tx.Amount),
				TokenSymbol:            strings.TrimSpace(tx.TokenSymbol),
				Confidence:             confidence,
			})
		}
		if classifyTreasuryMMMarketMaker(joinedText) {
			days := mmCounterparties[counterpartyKey]
			if days == nil {
				days = map[string]struct{}{}
				mmCounterparties[counterpartyKey] = days
			}
			days[dayKey] = struct{}{}
			observation, found, err := s.buildMMObservation(ctx, normalized.Chain, tx)
			if err != nil {
				return WalletTreasuryMMEvidenceReport{}, err
			}
			if found {
				report.MMPaths = append(report.MMPaths, observation)
			} else {
				pathKind := "project_to_mm_adjacency"
				confidence := 0.58
				if subtype := classifyMMRoutedCandidateType(joinedText); subtype != "" {
					pathKind = "project_to_mm_adjacency_" + subtype
					confidence = mmAdjacencyConfidence(subtype)
				} else {
					pathKind = "project_to_mm_adjacency_generic"
				}
				report.MMPaths = append(report.MMPaths, WalletMMPathObservation{
					TxHash:                 strings.TrimSpace(tx.TxHash),
					ObservedAt:             tx.ObservedAt.UTC(),
					PathKind:               pathKind,
					CounterpartyChain:      counterpartyChain,
					CounterpartyAddress:    counterpartyAddress,
					CounterpartyLabel:      firstNonEmpty(strings.TrimSpace(tx.CounterpartyDisplayName), counterpartyAddress),
					CounterpartyEntityKey:  strings.TrimSpace(tx.CounterpartyEntityKey),
					CounterpartyEntityType: strings.TrimSpace(tx.CounterpartyEntityType),
					Amount:                 strings.TrimSpace(tx.Amount),
					TokenSymbol:            strings.TrimSpace(tx.TokenSymbol),
					Confidence:             confidence,
				})
			}
		}
	}

	fanoutSignatureCount := 0
	operationalDistributionCount := 0
	for _, count := range outboundByDay {
		if count >= 3 {
			fanoutSignatureCount = treasuryMMMaxInt(fanoutSignatureCount, count)
			operationalDistributionCount++
		}
	}

	inventoryRotationCount := 0
	repeatMMCounterpartyCount := 0
	for key, item := range mmDirection {
		if len(item.inboundDays) > 0 && len(item.outboundDays) > 0 {
			inventoryRotationCount++
		}
		if len(mmCounterparties[key]) >= 2 {
			repeatMMCounterpartyCount++
		}
	}

	report.TreasuryFeatures = WalletTreasuryFeatures{
		AnchorMatchCount:                 treasuryMMMaxInt(treasuryMMBoolToInt(labels.hasTreasury), len(treasuryAnchorCounterparties)),
		FanoutSignatureCount:             fanoutSignatureCount,
		OperationalDistributionCount:     operationalDistributionCount,
		RebalanceDiscountCount:           rebalanceDiscountCount,
		TreasuryToMarketPathCount:        treasuryMarketPathCount,
		TreasuryToExchangePathCount:      treasuryToExchangePathCount,
		TreasuryToBridgePathCount:        treasuryToBridgePathCount,
		TreasuryToMMPathCount:            treasuryToMMPathCount,
		DistinctMarketCounterpartyCount:  len(treasuryMarketCounterparties),
		OperationalOnlyDistributionCount: operationalOnlyDistributionCount,
		InternalOpsDistributionCount:     internalOpsDistributionCount,
		ExternalOpsDistributionCount:     externalOpsDistributionCount,
		ExternalMarketAdjacentCount:      externalMarketAdjacentCount,
		ExternalNonMarketCount:           externalNonMarketCount,
		LatestTreasuryTxHash:             latestTreasuryTxHash(report.TreasuryPaths),
	}
	report.MMFeatures = WalletMMFeatures{
		MMAnchorMatchCount:              len(mmCounterparties),
		InventoryRotationCount:          inventoryRotationCount,
		ProjectToMMPathCount:            countMMPathsByKind(report.MMPaths, "project_to_mm_path"),
		PostHandoffDistributionCount:    countMMPathsByPrefix(report.MMPaths, "post_handoff_"),
		PostHandoffExchangeTouchCount:   countMMPathsByKind(report.MMPaths, "post_handoff_exchange_distribution"),
		PostHandoffBridgeTouchCount:     countMMPathsByKind(report.MMPaths, "post_handoff_bridge_distribution"),
		ProjectToMMRoutedCandidateCount: countMMPathsByPrefix(report.MMPaths, "project_to_mm_routed_candidate_"),
		ProjectToMMAdjacencyCount:       countMMPathsByPrefix(report.MMPaths, "project_to_mm_adjacency_"),
		ProjectToMMContactCount: countMMPathsByPrefix(report.MMPaths, "project_to_mm_routed_candidate_") +
			countMMPathsByPrefix(report.MMPaths, "project_to_mm_adjacency_"),
		RepeatMMCounterpartyCount: repeatMMCounterpartyCount,
		LatestMMTxHash:            latestMMTxHash(report.MMPaths),
	}

	sort.SliceStable(report.TreasuryPaths, func(i, j int) bool {
		return report.TreasuryPaths[i].ObservedAt.After(report.TreasuryPaths[j].ObservedAt)
	})
	sort.SliceStable(report.MMPaths, func(i, j int) bool {
		return report.MMPaths[i].ObservedAt.After(report.MMPaths[j].ObservedAt)
	})

	return report, nil
}

func (s *PostgresWalletTreasuryMMEvidenceStore) ReplaceWalletTreasuryMMEvidence(
	ctx context.Context,
	report WalletTreasuryMMEvidenceReport,
) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("wallet treasury/mm evidence store is nil")
	}
	walletID := strings.TrimSpace(report.WalletID)
	if walletID == "" {
		return fmt.Errorf("wallet id is required")
	}
	observedDay := report.WindowEndAt.UTC().Format("2006-01-02")
	now := s.now().UTC()
	if _, err := s.Execer.Exec(ctx, deleteWalletTreasuryPathsForDaySQL, walletID, observedDay); err != nil {
		return fmt.Errorf("delete wallet treasury paths: %w", err)
	}
	if _, err := s.Execer.Exec(ctx, deleteWalletMMPathsForDaySQL, walletID, observedDay); err != nil {
		return fmt.Errorf("delete wallet mm paths: %w", err)
	}

	treasuryMetadata, err := json.Marshal(map[string]any{
		"wallet_id":                           walletID,
		"has_treasury_label":                  report.HasTreasuryLabel,
		"fanout_signature_count":              report.TreasuryFeatures.FanoutSignatureCount,
		"treasury_to_market_path_count":       report.TreasuryFeatures.TreasuryToMarketPathCount,
		"treasury_to_exchange_path_count":     report.TreasuryFeatures.TreasuryToExchangePathCount,
		"treasury_to_bridge_path_count":       report.TreasuryFeatures.TreasuryToBridgePathCount,
		"treasury_to_mm_path_count":           report.TreasuryFeatures.TreasuryToMMPathCount,
		"distinct_market_counterparty_count":  report.TreasuryFeatures.DistinctMarketCounterpartyCount,
		"operational_only_distribution_count": report.TreasuryFeatures.OperationalOnlyDistributionCount,
		"internal_ops_distribution_count":     report.TreasuryFeatures.InternalOpsDistributionCount,
		"external_ops_distribution_count":     report.TreasuryFeatures.ExternalOpsDistributionCount,
		"external_market_adjacent_count":      report.TreasuryFeatures.ExternalMarketAdjacentCount,
		"external_non_market_count":           report.TreasuryFeatures.ExternalNonMarketCount,
		"rebalance_discount_count":            report.TreasuryFeatures.RebalanceDiscountCount,
	})
	if err != nil {
		return fmt.Errorf("marshal wallet treasury features metadata: %w", err)
	}
	if _, err := s.Execer.Exec(
		ctx,
		upsertWalletTreasuryFeaturesDailySQL,
		walletID,
		observedDay,
		report.WindowStartAt.UTC(),
		report.WindowEndAt.UTC(),
		report.TreasuryFeatures.AnchorMatchCount,
		report.TreasuryFeatures.FanoutSignatureCount,
		report.TreasuryFeatures.OperationalDistributionCount,
		report.TreasuryFeatures.RebalanceDiscountCount,
		report.TreasuryFeatures.TreasuryToMarketPathCount,
		report.TreasuryFeatures.TreasuryToExchangePathCount,
		report.TreasuryFeatures.TreasuryToBridgePathCount,
		report.TreasuryFeatures.TreasuryToMMPathCount,
		report.TreasuryFeatures.DistinctMarketCounterpartyCount,
		report.TreasuryFeatures.OperationalOnlyDistributionCount,
		report.TreasuryFeatures.InternalOpsDistributionCount,
		report.TreasuryFeatures.ExternalOpsDistributionCount,
		report.TreasuryFeatures.ExternalMarketAdjacentCount,
		report.TreasuryFeatures.ExternalNonMarketCount,
		report.TreasuryFeatures.LatestTreasuryTxHash,
		treasuryMetadata,
		now,
	); err != nil {
		return fmt.Errorf("upsert wallet treasury features: %w", err)
	}

	mmMetadata, err := json.Marshal(map[string]any{
		"wallet_id":                            walletID,
		"has_fund_label":                       report.HasFundLabel,
		"inventory_rotation_count":             report.MMFeatures.InventoryRotationCount,
		"post_handoff_distribution_count":      report.MMFeatures.PostHandoffDistributionCount,
		"post_handoff_exchange_touch_count":    report.MMFeatures.PostHandoffExchangeTouchCount,
		"post_handoff_bridge_touch_count":      report.MMFeatures.PostHandoffBridgeTouchCount,
		"project_to_mm_contact_count":          report.MMFeatures.ProjectToMMContactCount,
		"project_to_mm_routed_candidate_count": report.MMFeatures.ProjectToMMRoutedCandidateCount,
		"project_to_mm_adjacency_count":        report.MMFeatures.ProjectToMMAdjacencyCount,
		"repeat_mm_counterparty_count":         report.MMFeatures.RepeatMMCounterpartyCount,
	})
	if err != nil {
		return fmt.Errorf("marshal wallet mm features metadata: %w", err)
	}
	if _, err := s.Execer.Exec(
		ctx,
		upsertWalletMMFeaturesDailySQL,
		walletID,
		observedDay,
		report.WindowStartAt.UTC(),
		report.WindowEndAt.UTC(),
		report.MMFeatures.MMAnchorMatchCount,
		report.MMFeatures.InventoryRotationCount,
		report.MMFeatures.ProjectToMMPathCount,
		report.MMFeatures.PostHandoffDistributionCount,
		report.MMFeatures.PostHandoffExchangeTouchCount,
		report.MMFeatures.PostHandoffBridgeTouchCount,
		report.MMFeatures.ProjectToMMContactCount,
		report.MMFeatures.ProjectToMMRoutedCandidateCount,
		report.MMFeatures.ProjectToMMAdjacencyCount,
		report.MMFeatures.RepeatMMCounterpartyCount,
		report.MMFeatures.LatestMMTxHash,
		mmMetadata,
		now,
	); err != nil {
		return fmt.Errorf("upsert wallet mm features: %w", err)
	}

	for _, item := range report.TreasuryPaths {
		metadata, err := json.Marshal(treasuryMMPathMetadata(
			report.Chain,
			report.Address,
			item.TxHash,
			item.ObservedAt,
			item.PathKind,
			item.CounterpartyChain,
			item.CounterpartyAddress,
			item.CounterpartyEntityKey,
			item.DownstreamChain,
			item.DownstreamAddress,
			item.DownstreamTxHash,
		))
		if err != nil {
			return fmt.Errorf("marshal treasury path metadata: %w", err)
		}
		if _, err := s.Execer.Exec(
			ctx,
			insertWalletTreasuryPathSQL,
			walletID,
			observedDay,
			item.TxHash,
			item.ObservedAt.UTC(),
			item.PathKind,
			string(item.CounterpartyChain),
			item.CounterpartyAddress,
			item.CounterpartyLabel,
			item.CounterpartyEntityKey,
			item.CounterpartyEntityType,
			string(item.DownstreamChain),
			item.DownstreamAddress,
			item.DownstreamLabel,
			item.DownstreamEntityKey,
			item.DownstreamEntityType,
			item.DownstreamTxHash,
			item.DownstreamObservedAt,
			item.Amount,
			item.TokenSymbol,
			item.Confidence,
			metadata,
			now,
		); err != nil {
			return fmt.Errorf("insert wallet treasury path: %w", err)
		}
	}

	for _, item := range report.MMPaths {
		metadata, err := json.Marshal(treasuryMMPathMetadata(
			report.Chain,
			report.Address,
			item.TxHash,
			item.ObservedAt,
			item.PathKind,
			item.CounterpartyChain,
			item.CounterpartyAddress,
			item.CounterpartyEntityKey,
			item.DownstreamChain,
			item.DownstreamAddress,
			item.DownstreamTxHash,
		))
		if err != nil {
			return fmt.Errorf("marshal mm path metadata: %w", err)
		}
		if _, err := s.Execer.Exec(
			ctx,
			insertWalletMMPathSQL,
			walletID,
			observedDay,
			item.TxHash,
			item.ObservedAt.UTC(),
			item.PathKind,
			string(item.CounterpartyChain),
			item.CounterpartyAddress,
			item.CounterpartyLabel,
			item.CounterpartyEntityKey,
			item.CounterpartyEntityType,
			string(item.DownstreamChain),
			item.DownstreamAddress,
			item.DownstreamLabel,
			item.DownstreamEntityKey,
			item.DownstreamEntityType,
			item.DownstreamTxHash,
			item.DownstreamObservedAt,
			item.Amount,
			item.TokenSymbol,
			item.Confidence,
			metadata,
			now,
		); err != nil {
			return fmt.Errorf("insert wallet mm path: %w", err)
		}
	}

	return nil
}

type treasuryMMRootLabels struct {
	hasTreasury bool
	hasFund     bool
}

func readTreasuryMMRootLabels(
	ctx context.Context,
	reader WalletLabelReader,
	ref WalletRef,
) (treasuryMMRootLabels, error) {
	if reader == nil {
		return treasuryMMRootLabels{}, nil
	}
	labelsByWallet, err := reader.ReadWalletLabels(ctx, []WalletRef{ref})
	if err != nil {
		return treasuryMMRootLabels{}, err
	}
	labels := labelsByWallet[strings.ToLower(strings.TrimSpace(string(ref.Chain)))+"|"+strings.ToLower(strings.TrimSpace(ref.Address))]
	return treasuryMMRootLabels{
		hasTreasury: hasTreasuryMMLabel(labels, "treasury"),
		hasFund:     hasTreasuryMMLabel(labels, "fund"),
	}, nil
}

func hasTreasuryMMLabel(labels domain.WalletLabelSet, entityType string) bool {
	check := func(items []domain.WalletLabel) bool {
		for _, item := range items {
			if strings.EqualFold(strings.TrimSpace(item.EntityType), entityType) {
				return true
			}
		}
		return false
	}
	return check(labels.Verified) || check(labels.Inferred)
}

func (s *PostgresWalletTreasuryMMEvidenceStore) buildMMObservation(
	ctx context.Context,
	rootChain domain.Chain,
	tx treasuryMMTransactionRow,
) (WalletMMPathObservation, bool, error) {
	windowStart := tx.ObservedAt.UTC()
	windowEnd := windowStart.Add(2 * time.Hour)
	rows, err := s.Querier.Query(
		ctx,
		walletBridgeExchangeDownstreamSQL,
		string(normalizeCandidateChain(tx.CounterpartyChain, rootChain)),
		strings.TrimSpace(tx.CounterpartyAddress),
		windowStart,
		windowEnd,
	)
	if err != nil {
		return WalletMMPathObservation{}, false, fmt.Errorf("list downstream mm handoff paths: %w", err)
	}
	defer rows.Close()

	base := WalletMMPathObservation{
		TxHash:                 strings.TrimSpace(tx.TxHash),
		ObservedAt:             tx.ObservedAt.UTC(),
		PathKind:               "project_to_mm_path",
		CounterpartyChain:      normalizeCandidateChain(tx.CounterpartyChain, rootChain),
		CounterpartyAddress:    strings.TrimSpace(tx.CounterpartyAddress),
		CounterpartyLabel:      firstNonEmpty(strings.TrimSpace(tx.CounterpartyDisplayName), strings.TrimSpace(tx.CounterpartyAddress)),
		CounterpartyEntityKey:  strings.TrimSpace(tx.CounterpartyEntityKey),
		CounterpartyEntityType: strings.TrimSpace(tx.CounterpartyEntityType),
		Amount:                 strings.TrimSpace(tx.Amount),
		TokenSymbol:            strings.TrimSpace(tx.TokenSymbol),
		Confidence:             0.8,
	}
	var firstDownstream *walletBridgeExchangeTransactionRow

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
			return WalletMMPathObservation{}, false, fmt.Errorf("scan downstream mm handoff path: %w", err)
		}
		if firstDownstream == nil {
			copy := item
			firstDownstream = &copy
		}
		joinedText := strings.Join([]string{
			strings.TrimSpace(item.CounterpartyDisplayName),
			strings.TrimSpace(item.CounterpartyEntityType),
			strings.TrimSpace(item.CounterpartyEntityKey),
		}, " ")
		pathKind, confidence, ok := classifyMMDownstreamPath(joinedText)
		if !ok {
			continue
		}
		observedAt := item.ObservedAt.UTC()
		base.PathKind = pathKind
		base.DownstreamChain = normalizeCandidateChain(item.CounterpartyChain, rootChain)
		base.DownstreamAddress = strings.TrimSpace(item.CounterpartyAddress)
		base.DownstreamLabel = firstNonEmpty(strings.TrimSpace(item.CounterpartyDisplayName), strings.TrimSpace(item.CounterpartyAddress))
		base.DownstreamEntityKey = strings.TrimSpace(item.CounterpartyEntityKey)
		base.DownstreamEntityType = strings.TrimSpace(item.CounterpartyEntityType)
		base.DownstreamTxHash = strings.TrimSpace(item.TxHash)
		base.DownstreamObservedAt = &observedAt
		base.Confidence = confidence
		return base, true, nil
	}
	if err := rows.Err(); err != nil {
		return WalletMMPathObservation{}, false, fmt.Errorf("iterate downstream mm handoff paths: %w", err)
	}
	if firstDownstream != nil {
		subtype := classifyMMRoutedCandidateType(strings.Join([]string{
			strings.TrimSpace(tx.CounterpartyDisplayName),
			strings.TrimSpace(tx.CounterpartyEntityType),
			strings.TrimSpace(tx.CounterpartyEntityKey),
		}, " "))
		if subtype == "" {
			return base, false, nil
		}
		observedAt := firstDownstream.ObservedAt.UTC()
		base.PathKind = "project_to_mm_routed_candidate_" + subtype
		base.DownstreamChain = normalizeCandidateChain(firstDownstream.CounterpartyChain, rootChain)
		base.DownstreamAddress = strings.TrimSpace(firstDownstream.CounterpartyAddress)
		base.DownstreamLabel = firstNonEmpty(strings.TrimSpace(firstDownstream.CounterpartyDisplayName), strings.TrimSpace(firstDownstream.CounterpartyAddress))
		base.DownstreamEntityKey = strings.TrimSpace(firstDownstream.CounterpartyEntityKey)
		base.DownstreamEntityType = strings.TrimSpace(firstDownstream.CounterpartyEntityType)
		base.DownstreamTxHash = strings.TrimSpace(firstDownstream.TxHash)
		base.DownstreamObservedAt = &observedAt
		base.Confidence = mmRoutedCandidateConfidence(subtype)
		return base, true, nil
	}
	return base, false, nil
}

func (s *PostgresWalletTreasuryMMEvidenceStore) buildTreasuryExternalOpsObservation(
	ctx context.Context,
	rootChain domain.Chain,
	tx treasuryMMTransactionRow,
) (WalletTreasuryPathObservation, bool, error) {
	windowStart := tx.ObservedAt.UTC()
	windowEnd := windowStart.Add(2 * time.Hour)
	rows, err := s.Querier.Query(
		ctx,
		walletBridgeExchangeDownstreamSQL,
		string(normalizeCandidateChain(tx.CounterpartyChain, rootChain)),
		strings.TrimSpace(tx.CounterpartyAddress),
		windowStart,
		windowEnd,
	)
	if err != nil {
		return WalletTreasuryPathObservation{}, false, fmt.Errorf("list downstream treasury external ops paths: %w", err)
	}
	defer rows.Close()

	base := WalletTreasuryPathObservation{
		TxHash:                 strings.TrimSpace(tx.TxHash),
		ObservedAt:             tx.ObservedAt.UTC(),
		CounterpartyChain:      normalizeCandidateChain(tx.CounterpartyChain, rootChain),
		CounterpartyAddress:    strings.TrimSpace(tx.CounterpartyAddress),
		CounterpartyLabel:      firstNonEmpty(strings.TrimSpace(tx.CounterpartyDisplayName), strings.TrimSpace(tx.CounterpartyAddress)),
		CounterpartyEntityKey:  strings.TrimSpace(tx.CounterpartyEntityKey),
		CounterpartyEntityType: strings.TrimSpace(tx.CounterpartyEntityType),
		Amount:                 strings.TrimSpace(tx.Amount),
		TokenSymbol:            strings.TrimSpace(tx.TokenSymbol),
		Confidence:             0.68,
	}
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
			return WalletTreasuryPathObservation{}, false, fmt.Errorf("scan downstream treasury external ops path: %w", err)
		}
		joinedText := strings.Join([]string{
			strings.TrimSpace(item.CounterpartyDisplayName),
			strings.TrimSpace(item.CounterpartyEntityType),
			strings.TrimSpace(item.CounterpartyEntityKey),
		}, " ")
		subtype := classifyTreasuryRoutedMarketSubtype(joinedText)
		if subtype == "" {
			continue
		}
		observedAt := item.ObservedAt.UTC()
		base.PathKind = "treasury_external_market_adjacent_routed_" + subtype
		base.DownstreamChain = normalizeCandidateChain(item.CounterpartyChain, rootChain)
		base.DownstreamAddress = strings.TrimSpace(item.CounterpartyAddress)
		base.DownstreamLabel = firstNonEmpty(strings.TrimSpace(item.CounterpartyDisplayName), strings.TrimSpace(item.CounterpartyAddress))
		base.DownstreamEntityKey = strings.TrimSpace(item.CounterpartyEntityKey)
		base.DownstreamEntityType = strings.TrimSpace(item.CounterpartyEntityType)
		base.DownstreamTxHash = strings.TrimSpace(item.TxHash)
		base.DownstreamObservedAt = &observedAt
		base.Confidence = treasuryExternalMarketAdjacentConfidence(subtype, true)
		return base, true, nil
	}
	if err := rows.Err(); err != nil {
		return WalletTreasuryPathObservation{}, false, fmt.Errorf("iterate downstream treasury external ops paths: %w", err)
	}
	return base, false, nil
}

func classifyMMRoutedCandidateType(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return ""
	}
	patternMap := []struct {
		subtype string
		terms   []string
	}{
		{subtype: "desk", terms: []string{"desk", "otc"}},
		{subtype: "liquidity", terms: []string{"liquidity", "liquidity provider", "inventory"}},
		{subtype: "router", terms: []string{"routing", "router"}},
	}
	for _, item := range patternMap {
		for _, term := range item.terms {
			if strings.Contains(normalized, term) {
				return item.subtype
			}
		}
	}
	return ""
}

func classifyTreasuryExternalMarketAdjacentType(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return ""
	}
	patternMap := []struct {
		subtype string
		terms   []string
	}{
		{subtype: "router", terms: []string{"router", "aggregator"}},
		{subtype: "dex", terms: []string{"dex", "swap", "amm", "pool"}},
		{subtype: "marketplace", terms: []string{"marketplace"}},
	}
	for _, item := range patternMap {
		for _, term := range item.terms {
			if strings.Contains(normalized, term) {
				return item.subtype
			}
		}
	}
	return ""
}

func classifyTreasuryRoutedMarketSubtype(text string) string {
	switch {
	case classifyShadowExitCEX(text):
		return "exchange"
	case classifyShadowExitBridge(text):
		return "bridge"
	case classifyTreasuryMMMarketMaker(text):
		return "mm"
	default:
		return ""
	}
}

func mmRoutedCandidateConfidence(subtype string) float64 {
	switch strings.TrimSpace(subtype) {
	case "desk":
		return 0.76
	case "liquidity":
		return 0.74
	case "router":
		return 0.7
	default:
		return 0.68
	}
}

func mmAdjacencyConfidence(subtype string) float64 {
	switch strings.TrimSpace(subtype) {
	case "desk":
		return 0.66
	case "liquidity":
		return 0.64
	case "router":
		return 0.62
	default:
		return 0.58
	}
}

func treasuryExternalMarketAdjacentConfidence(subtype string, routed bool) float64 {
	switch strings.TrimSpace(subtype) {
	case "exchange":
		if routed {
			return 0.78
		}
	case "mm":
		if routed {
			return 0.76
		}
	case "bridge":
		if routed {
			return 0.72
		}
	case "router":
		return 0.64
	case "dex":
		return 0.66
	case "marketplace":
		return 0.6
	}
	if routed {
		return 0.7
	}
	return 0.62
}

func classifyTreasuryMMMarketMaker(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	patterns := []string{
		"market maker",
		"market_maker",
		"market-maker",
		"wintermute",
		"gsr",
		"jump trading",
		"cumberland",
		"flow trader",
	}
	for _, pattern := range patterns {
		if strings.Contains(normalized, pattern) {
			return true
		}
	}
	return false
}

func classifyTreasuryMarketPath(text string) (string, float64, bool) {
	switch {
	case classifyShadowExitCEX(text):
		return "treasury_to_exchange_path", 0.82, true
	case classifyTreasuryMMMarketMaker(text):
		return "treasury_to_mm_path", 0.8, true
	case classifyShadowExitBridge(text):
		return "treasury_to_bridge_path", 0.72, true
	default:
		return "", 0, false
	}
}

func classifyMMDownstreamPath(text string) (string, float64, bool) {
	switch {
	case classifyShadowExitCEX(text):
		return "post_handoff_exchange_distribution", 0.9, true
	case classifyShadowExitBridge(text):
		return "post_handoff_bridge_distribution", 0.82, true
	default:
		return "", 0, false
	}
}

func classifyTreasuryExternalMarketAdjacent(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	patterns := []string{
		"router",
		"dex",
		"swap",
		"aggregator",
		"marketplace",
		"pool",
		"amm",
	}
	for _, pattern := range patterns {
		if strings.Contains(normalized, pattern) {
			return true
		}
	}
	return false
}

func treasuryMMPathMetadata(
	rootChain domain.Chain,
	rootAddress string,
	txHash string,
	observedAt time.Time,
	pathKind string,
	counterpartyChain domain.Chain,
	counterpartyAddress string,
	counterpartyEntityKey string,
	downstreamChain domain.Chain,
	downstreamAddress string,
	downstreamTxHash string,
) map[string]any {
	sourceSubtype, downstreamSubtype, pathStrength, confidenceTier := treasuryMMPathHints(pathKind)
	metadata := map[string]any{
		"txRef": map[string]any{
			"chain":      string(rootChain),
			"address":    strings.TrimSpace(rootAddress),
			"txHash":     strings.TrimSpace(txHash),
			"observedAt": observedAt.UTC().Format(time.RFC3339),
		},
		"pathRef": map[string]any{
			"kind":                strings.TrimSpace(pathKind),
			"pathStrength":        pathStrength,
			"confidenceTier":      confidenceTier,
			"sourceSubtype":       sourceSubtype,
			"downstreamSubtype":   downstreamSubtype,
			"counterpartyChain":   string(counterpartyChain),
			"counterpartyAddress": strings.TrimSpace(counterpartyAddress),
			"downstreamChain":     string(downstreamChain),
			"downstreamAddress":   strings.TrimSpace(downstreamAddress),
			"downstreamTxHash":    strings.TrimSpace(downstreamTxHash),
		},
	}
	if strings.TrimSpace(counterpartyEntityKey) != "" {
		metadata["entityRef"] = map[string]any{
			"entityKey": strings.TrimSpace(counterpartyEntityKey),
		}
	}
	if strings.TrimSpace(counterpartyAddress) != "" {
		metadata["counterpartyRef"] = map[string]any{
			"chain":   string(counterpartyChain),
			"address": strings.TrimSpace(counterpartyAddress),
		}
	}
	return metadata
}

func treasuryMMPathHints(pathKind string) (string, string, string, string) {
	trimmed := strings.TrimSpace(pathKind)
	switch {
	case strings.HasPrefix(trimmed, "project_to_mm_routed_candidate_"):
		return strings.TrimPrefix(trimmed, "project_to_mm_routed_candidate_"), "", "routed_candidate", "medium_high"
	case strings.HasPrefix(trimmed, "project_to_mm_adjacency_"):
		return strings.TrimPrefix(trimmed, "project_to_mm_adjacency_"), "", "adjacency", "medium"
	case strings.HasPrefix(trimmed, "treasury_external_market_adjacent_direct_"):
		return strings.TrimPrefix(trimmed, "treasury_external_market_adjacent_direct_"), "", "direct_market_adjacent", "medium"
	case strings.HasPrefix(trimmed, "treasury_external_market_adjacent_routed_"):
		return "", strings.TrimPrefix(trimmed, "treasury_external_market_adjacent_routed_"), "routed_market_adjacent", "medium_high"
	case trimmed == "treasury_to_exchange_path":
		return "", "exchange", "confirmed_market_path", "high"
	case trimmed == "treasury_to_bridge_path":
		return "", "bridge", "confirmed_market_path", "medium_high"
	case trimmed == "treasury_to_mm_path":
		return "", "mm", "confirmed_market_path", "high"
	case trimmed == "post_handoff_exchange_distribution":
		return "", "exchange", "confirmed_distribution", "high"
	case trimmed == "post_handoff_bridge_distribution":
		return "", "bridge", "confirmed_distribution", "medium_high"
	case trimmed == "treasury_internal_ops_distribution":
		return "internal", "", "internal_ops", "medium"
	case trimmed == "treasury_external_non_market_ops":
		return "external", "", "non_market_ops", "low_medium"
	default:
		return "", "", "generic", "medium"
	}
}

func latestTreasuryTxHash(items []WalletTreasuryPathObservation) string {
	for _, item := range items {
		if strings.TrimSpace(item.TxHash) != "" {
			return strings.TrimSpace(item.TxHash)
		}
	}
	return ""
}

func latestMMTxHash(items []WalletMMPathObservation) string {
	for _, item := range items {
		if strings.TrimSpace(item.TxHash) != "" {
			return strings.TrimSpace(item.TxHash)
		}
	}
	return ""
}

func countMMPathsByPrefix(items []WalletMMPathObservation, prefix string) int {
	normalizedPrefix := strings.TrimSpace(prefix)
	if normalizedPrefix == "" {
		return 0
	}
	count := 0
	for _, item := range items {
		if strings.HasPrefix(strings.TrimSpace(item.PathKind), normalizedPrefix) {
			count++
		}
	}
	return count
}

func countMMPathsByKind(items []WalletMMPathObservation, kind string) int {
	total := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.PathKind), strings.TrimSpace(kind)) {
			total++
		}
	}
	return total
}

func (s *PostgresWalletTreasuryMMEvidenceStore) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func treasuryMMMaxInt(values ...int) int {
	max := 0
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}

func treasuryMMBoolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
