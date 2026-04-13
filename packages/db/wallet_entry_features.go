package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

const upsertWalletEntryFeaturesDailySQL = `
INSERT INTO wallet_entry_features_daily (
  wallet_id,
  observed_day,
  window_start_at,
  window_end_at,
  quality_wallet_overlap_count,
  sustained_overlap_counterparty_count,
  strong_lead_counterparty_count,
  first_entry_before_crowding_count,
  best_lead_hours_before_peers,
  persistence_after_entry_proxy_count,
  repeat_early_entry_success,
  latest_counterparty_chain,
  latest_counterparty_address,
  metadata,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
)
ON CONFLICT (wallet_id, observed_day) DO UPDATE SET
  window_start_at = EXCLUDED.window_start_at,
  window_end_at = EXCLUDED.window_end_at,
  quality_wallet_overlap_count = EXCLUDED.quality_wallet_overlap_count,
  sustained_overlap_counterparty_count = EXCLUDED.sustained_overlap_counterparty_count,
  strong_lead_counterparty_count = EXCLUDED.strong_lead_counterparty_count,
  first_entry_before_crowding_count = EXCLUDED.first_entry_before_crowding_count,
  best_lead_hours_before_peers = EXCLUDED.best_lead_hours_before_peers,
  persistence_after_entry_proxy_count = EXCLUDED.persistence_after_entry_proxy_count,
  repeat_early_entry_success = EXCLUDED.repeat_early_entry_success,
  latest_counterparty_chain = EXCLUDED.latest_counterparty_chain,
  latest_counterparty_address = EXCLUDED.latest_counterparty_address,
  metadata = EXCLUDED.metadata,
  updated_at = EXCLUDED.updated_at
`

const latestWalletEntryFeaturesByRefSQL = `
SELECT
  we.wallet_id,
  w.chain,
  w.address,
  we.window_start_at,
  we.window_end_at,
  we.quality_wallet_overlap_count,
  we.sustained_overlap_counterparty_count,
  we.strong_lead_counterparty_count,
  we.first_entry_before_crowding_count,
  we.best_lead_hours_before_peers,
  we.persistence_after_entry_proxy_count,
  we.repeat_early_entry_success,
  we.latest_counterparty_chain,
  we.latest_counterparty_address,
  we.metadata
FROM wallet_entry_features_daily we
JOIN wallets w ON w.id = we.wallet_id
WHERE w.chain = $1 AND w.address = $2
ORDER BY we.observed_day DESC, we.updated_at DESC
LIMIT 1
`

const latestWalletEntryFeaturesByRefBeforeSQL = `
SELECT
  we.wallet_id,
  w.chain,
  w.address,
  we.window_start_at,
  we.window_end_at,
  we.quality_wallet_overlap_count,
  we.sustained_overlap_counterparty_count,
  we.strong_lead_counterparty_count,
  we.first_entry_before_crowding_count,
  we.best_lead_hours_before_peers,
  we.persistence_after_entry_proxy_count,
  we.repeat_early_entry_success,
  we.latest_counterparty_chain,
  we.latest_counterparty_address,
  we.metadata
FROM wallet_entry_features_daily we
JOIN wallets w ON w.id = we.wallet_id
WHERE w.chain = $1 AND w.address = $2 AND we.window_end_at < $3
ORDER BY we.window_end_at DESC, we.updated_at DESC
LIMIT 1
`

const walletEntryFeatureFollowThroughSQL = `
WITH cp AS (
  SELECT *
  FROM UNNEST($2::text[], $3::text[]) AS cp(counterparty_chain, counterparty_address)
),
agg AS (
  SELECT
    cp.counterparty_chain,
    cp.counterparty_address,
    COUNT(t.*) AS follow_through_count,
    COALESCE(
      MAX(
        GREATEST(EXTRACT(EPOCH FROM (t.observed_at - $4::timestamptz)) / 3600, 0)
      ),
      0
    )::int AS max_persistence_hours
  FROM cp
  LEFT JOIN transactions t
    ON t.wallet_id = $1::uuid
   AND COALESCE(NULLIF(t.counterparty_chain, ''), $5) = cp.counterparty_chain
   AND NULLIF(t.counterparty_address, '') = cp.counterparty_address
   AND t.observed_at > $4
   AND t.observed_at <= $6
  GROUP BY cp.counterparty_chain, cp.counterparty_address
)
SELECT
  COUNT(*) FILTER (WHERE follow_through_count > 0) AS post_window_follow_through_count,
  COALESCE(MAX(max_persistence_hours), 0) AS max_post_window_persistence_hours,
  COUNT(*) FILTER (WHERE follow_through_count = 0) AS short_lived_overlap_count
FROM agg
`

const historicalSustainedEntryOutcomeCountSQL = `
SELECT COUNT(*)
FROM wallet_entry_features_daily we
JOIN wallets w ON w.id = we.wallet_id
WHERE w.chain = $1
  AND w.address = $2
  AND we.window_end_at < $3
  AND COALESCE(we.metadata->>'holding_persistence_state', '') = 'sustained'
`

type WalletEntryFeatureCounterparty struct {
	Chain                domain.Chain `json:"chain"`
	Address              string       `json:"address"`
	InteractionCount     int64        `json:"interaction_count"`
	PeerWalletCount      int64        `json:"peer_wallet_count"`
	PeerTxCount          int64        `json:"peer_tx_count"`
	FirstActivityAt      string       `json:"first_activity_at,omitempty"`
	LatestActivityAt     string       `json:"latest_activity_at,omitempty"`
	LeadHoursBeforePeers int64        `json:"lead_hours_before_peers"`
}

type WalletEntryFeaturesUpsert struct {
	WalletID                          string
	WindowStartAt                     time.Time
	WindowEndAt                       time.Time
	QualityWalletOverlapCount         int
	SustainedOverlapCounterpartyCount int
	StrongLeadCounterpartyCount       int
	FirstEntryBeforeCrowdingCount     int
	BestLeadHoursBeforePeers          int
	PersistenceAfterEntryProxyCount   int
	RepeatEarlyEntrySuccess           bool
	HistoricalSustainedOutcomeCount   int
	PostWindowFollowThroughCount      int
	MaxPostWindowPersistenceHours     int
	ShortLivedOverlapCount            int
	HoldingPersistenceState           string
	OutcomeResolvedAt                 *time.Time
	TopCounterparties                 []WalletEntryFeatureCounterparty
}

type WalletEntryFeaturesStore interface {
	UpsertWalletEntryFeatures(context.Context, WalletEntryFeaturesUpsert) error
}

type WalletEntryFeaturesMaturityReader interface {
	ReadLatestWalletEntryFeaturesBefore(context.Context, WalletRef, time.Time) (WalletEntryFeaturesSnapshot, error)
	ReadWalletEntryFeatureFollowThrough(context.Context, WalletEntryFeatureFollowThroughQuery) (WalletEntryFeatureFollowThrough, error)
	ReadHistoricalSustainedEntryOutcomeCount(context.Context, WalletRef, time.Time) (int, error)
}

type WalletEntryFeaturesSnapshot struct {
	WalletID                          string
	Chain                             domain.Chain
	Address                           string
	WindowStartAt                     time.Time
	WindowEndAt                       time.Time
	QualityWalletOverlapCount         int
	SustainedOverlapCounterpartyCount int
	StrongLeadCounterpartyCount       int
	FirstEntryBeforeCrowdingCount     int
	BestLeadHoursBeforePeers          int
	PersistenceAfterEntryProxyCount   int
	RepeatEarlyEntrySuccess           bool
	HistoricalSustainedOutcomeCount   int
	PostWindowFollowThroughCount      int
	MaxPostWindowPersistenceHours     int
	ShortLivedOverlapCount            int
	HoldingPersistenceState           string
	OutcomeResolvedAt                 *time.Time
	LatestCounterpartyChain           domain.Chain
	LatestCounterpartyAddress         string
	TopCounterparties                 []WalletEntryFeatureCounterparty
}

type WalletEntryFeaturesReader interface {
	ReadLatestWalletEntryFeatures(context.Context, WalletRef) (WalletEntryFeaturesSnapshot, error)
}

type WalletEntryFeatureFollowThroughQuery struct {
	WalletID          string
	WalletChain       domain.Chain
	TopCounterparties []WalletEntryFeatureCounterparty
	WindowStartAt     time.Time
	WindowEndAt       time.Time
}

type WalletEntryFeatureFollowThrough struct {
	PostWindowFollowThroughCount  int
	MaxPostWindowPersistenceHours int
	ShortLivedOverlapCount        int
}

type PostgresWalletEntryFeaturesStore struct {
	Querier postgresQuerier
	Execer  postgresFindingExecer
	Now     func() time.Time
}

func NewPostgresWalletEntryFeaturesStore(
	querier postgresQuerier,
	execer postgresFindingExecer,
) *PostgresWalletEntryFeaturesStore {
	return &PostgresWalletEntryFeaturesStore{
		Querier: querier,
		Execer:  execer,
		Now:     time.Now,
	}
}

func NewPostgresWalletEntryFeaturesStoreFromPool(pool interface {
	postgresQuerier
	postgresFindingExecer
}) *PostgresWalletEntryFeaturesStore {
	return NewPostgresWalletEntryFeaturesStore(pool, pool)
}

func (s *PostgresWalletEntryFeaturesStore) UpsertWalletEntryFeatures(
	ctx context.Context,
	entry WalletEntryFeaturesUpsert,
) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("wallet entry features store is nil")
	}
	walletID := strings.TrimSpace(entry.WalletID)
	if walletID == "" {
		return fmt.Errorf("wallet id is required")
	}
	windowEnd := entry.WindowEndAt.UTC()
	observedDay := windowEnd.Format("2006-01-02")

	latestChain := ""
	latestAddress := ""
	if len(entry.TopCounterparties) > 0 {
		latestChain = strings.TrimSpace(string(entry.TopCounterparties[0].Chain))
		latestAddress = strings.TrimSpace(entry.TopCounterparties[0].Address)
	}

	metadata, err := json.Marshal(map[string]any{
		"wallet_id":                            walletID,
		"quality_wallet_overlap_count":         entry.QualityWalletOverlapCount,
		"sustained_overlap_counterparty_count": entry.SustainedOverlapCounterpartyCount,
		"strong_lead_counterparty_count":       entry.StrongLeadCounterpartyCount,
		"first_entry_before_crowding_count":    entry.FirstEntryBeforeCrowdingCount,
		"best_lead_hours_before_peers":         entry.BestLeadHoursBeforePeers,
		"persistence_after_entry_proxy_count":  entry.PersistenceAfterEntryProxyCount,
		"repeat_early_entry_success":           entry.RepeatEarlyEntrySuccess,
		"historical_sustained_outcome_count":   entry.HistoricalSustainedOutcomeCount,
		"post_window_follow_through_count":     entry.PostWindowFollowThroughCount,
		"max_post_window_persistence_hours":    entry.MaxPostWindowPersistenceHours,
		"short_lived_overlap_count":            entry.ShortLivedOverlapCount,
		"holding_persistence_state":            strings.TrimSpace(entry.HoldingPersistenceState),
		"outcome_resolved_at":                  formatWalletEntryOptionalTime(entry.OutcomeResolvedAt),
		"top_counterparties":                   entry.TopCounterparties,
		"feature_version":                      "wg052-qualified-v2",
	})
	if err != nil {
		return fmt.Errorf("marshal wallet entry features metadata: %w", err)
	}

	_, err = s.Execer.Exec(
		ctx,
		upsertWalletEntryFeaturesDailySQL,
		walletID,
		observedDay,
		entry.WindowStartAt.UTC(),
		windowEnd,
		entry.QualityWalletOverlapCount,
		entry.SustainedOverlapCounterpartyCount,
		entry.StrongLeadCounterpartyCount,
		entry.FirstEntryBeforeCrowdingCount,
		entry.BestLeadHoursBeforePeers,
		entry.PersistenceAfterEntryProxyCount,
		entry.RepeatEarlyEntrySuccess,
		latestChain,
		latestAddress,
		metadata,
		s.now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("upsert wallet entry features: %w", err)
	}
	return nil
}

func (s *PostgresWalletEntryFeaturesStore) ReadLatestWalletEntryFeatures(
	ctx context.Context,
	ref WalletRef,
) (WalletEntryFeaturesSnapshot, error) {
	if s == nil || s.Querier == nil {
		return WalletEntryFeaturesSnapshot{}, fmt.Errorf("wallet entry features store is nil")
	}
	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return WalletEntryFeaturesSnapshot{}, err
	}
	return s.readLatestWalletEntryFeatures(ctx, latestWalletEntryFeaturesByRefSQL, string(normalized.Chain), normalized.Address)
}

func (s *PostgresWalletEntryFeaturesStore) ReadLatestWalletEntryFeaturesBefore(
	ctx context.Context,
	ref WalletRef,
	before time.Time,
) (WalletEntryFeaturesSnapshot, error) {
	if s == nil || s.Querier == nil {
		return WalletEntryFeaturesSnapshot{}, fmt.Errorf("wallet entry features store is nil")
	}
	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return WalletEntryFeaturesSnapshot{}, err
	}
	return s.readLatestWalletEntryFeatures(ctx, latestWalletEntryFeaturesByRefBeforeSQL, string(normalized.Chain), normalized.Address, before.UTC())
}

func (s *PostgresWalletEntryFeaturesStore) ReadWalletEntryFeatureFollowThrough(
	ctx context.Context,
	query WalletEntryFeatureFollowThroughQuery,
) (WalletEntryFeatureFollowThrough, error) {
	if s == nil || s.Querier == nil {
		return WalletEntryFeatureFollowThrough{}, fmt.Errorf("wallet entry features store is nil")
	}
	walletID := strings.TrimSpace(query.WalletID)
	if walletID == "" {
		return WalletEntryFeatureFollowThrough{}, fmt.Errorf("wallet id is required")
	}
	if len(query.TopCounterparties) == 0 {
		return WalletEntryFeatureFollowThrough{}, nil
	}
	chains := make([]string, 0, len(query.TopCounterparties))
	addresses := make([]string, 0, len(query.TopCounterparties))
	for _, item := range query.TopCounterparties {
		address := strings.TrimSpace(item.Address)
		if address == "" {
			continue
		}
		chains = append(chains, strings.TrimSpace(string(item.Chain)))
		addresses = append(addresses, address)
	}
	if len(chains) == 0 {
		return WalletEntryFeatureFollowThrough{}, nil
	}

	var followThrough WalletEntryFeatureFollowThrough
	err := s.Querier.QueryRow(
		ctx,
		walletEntryFeatureFollowThroughSQL,
		walletID,
		chains,
		addresses,
		query.WindowStartAt.UTC(),
		string(query.WalletChain),
		query.WindowEndAt.UTC(),
	).Scan(
		&followThrough.PostWindowFollowThroughCount,
		&followThrough.MaxPostWindowPersistenceHours,
		&followThrough.ShortLivedOverlapCount,
	)
	if err != nil {
		return WalletEntryFeatureFollowThrough{}, err
	}
	return followThrough, nil
}

func (s *PostgresWalletEntryFeaturesStore) ReadHistoricalSustainedEntryOutcomeCount(
	ctx context.Context,
	ref WalletRef,
	before time.Time,
) (int, error) {
	if s == nil || s.Querier == nil {
		return 0, fmt.Errorf("wallet entry features store is nil")
	}
	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return 0, err
	}
	var count int
	if err := s.Querier.QueryRow(
		ctx,
		historicalSustainedEntryOutcomeCountSQL,
		string(normalized.Chain),
		normalized.Address,
		before.UTC(),
	).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *PostgresWalletEntryFeaturesStore) readLatestWalletEntryFeatures(
	ctx context.Context,
	query string,
	args ...any,
) (WalletEntryFeaturesSnapshot, error) {
	var (
		snapshot                WalletEntryFeaturesSnapshot
		latestCounterpartyChain string
		metadataRaw             []byte
	)
	err := s.Querier.QueryRow(ctx, query, args...).Scan(
		&snapshot.WalletID,
		&snapshot.Chain,
		&snapshot.Address,
		&snapshot.WindowStartAt,
		&snapshot.WindowEndAt,
		&snapshot.QualityWalletOverlapCount,
		&snapshot.SustainedOverlapCounterpartyCount,
		&snapshot.StrongLeadCounterpartyCount,
		&snapshot.FirstEntryBeforeCrowdingCount,
		&snapshot.BestLeadHoursBeforePeers,
		&snapshot.PersistenceAfterEntryProxyCount,
		&snapshot.RepeatEarlyEntrySuccess,
		&latestCounterpartyChain,
		&snapshot.LatestCounterpartyAddress,
		&metadataRaw,
	)
	if err != nil {
		return WalletEntryFeaturesSnapshot{}, err
	}
	snapshot.LatestCounterpartyChain = domain.Chain(strings.TrimSpace(latestCounterpartyChain))
	applyWalletEntryFeatureMetadata(&snapshot, metadataRaw)
	return snapshot, nil
}

func applyWalletEntryFeatureMetadata(snapshot *WalletEntryFeaturesSnapshot, metadataRaw []byte) {
	if snapshot == nil || len(metadataRaw) == 0 {
		return
	}
	var metadata struct {
		PostWindowFollowThroughCount    int                              `json:"post_window_follow_through_count"`
		MaxPostWindowPersistenceHours   int                              `json:"max_post_window_persistence_hours"`
		ShortLivedOverlapCount          int                              `json:"short_lived_overlap_count"`
		HistoricalSustainedOutcomeCount int                              `json:"historical_sustained_outcome_count"`
		HoldingPersistenceState         string                           `json:"holding_persistence_state"`
		OutcomeResolvedAt               string                           `json:"outcome_resolved_at"`
		TopCounterparties               []WalletEntryFeatureCounterparty `json:"top_counterparties"`
	}
	if err := json.Unmarshal(metadataRaw, &metadata); err != nil {
		return
	}
	snapshot.PostWindowFollowThroughCount = metadata.PostWindowFollowThroughCount
	snapshot.MaxPostWindowPersistenceHours = metadata.MaxPostWindowPersistenceHours
	snapshot.ShortLivedOverlapCount = metadata.ShortLivedOverlapCount
	snapshot.HistoricalSustainedOutcomeCount = metadata.HistoricalSustainedOutcomeCount
	snapshot.HoldingPersistenceState = strings.TrimSpace(metadata.HoldingPersistenceState)
	if resolvedAt := strings.TrimSpace(metadata.OutcomeResolvedAt); resolvedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, resolvedAt); err == nil {
			parsedUTC := parsed.UTC()
			snapshot.OutcomeResolvedAt = &parsedUTC
		}
	}
	snapshot.TopCounterparties = metadata.TopCounterparties
}

func formatWalletEntryOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (s *PostgresWalletEntryFeaturesStore) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
