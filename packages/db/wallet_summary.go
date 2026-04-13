package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

type WalletRef struct {
	Chain   domain.Chain
	Address string
}

type WalletSummaryIdentity struct {
	WalletID    string
	Chain       domain.Chain
	Address     string
	DisplayName string
	EntityKey   string
	Labels      domain.WalletLabelSet
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WalletSummaryStats struct {
	AsOfDate           time.Time
	TransactionCount   int64
	CounterpartyCount  int64
	EarliestActivityAt *time.Time
	LatestActivityAt   *time.Time
	IncomingTxCount    int64
	OutgoingTxCount    int64
	IncomingTxCount7d  int64
	OutgoingTxCount7d  int64
	IncomingTxCount30d int64
	OutgoingTxCount30d int64
	TopCounterparties  []WalletSummaryCounterparty
}

type WalletSummaryCounterparty struct {
	Chain            domain.Chain
	Address          string
	EntityKey        string
	EntityType       string
	EntityLabel      string
	Labels           domain.WalletLabelSet
	InteractionCount int64
	InboundCount     int64
	OutboundCount    int64
	InboundAmount    string
	OutboundAmount   string
	PrimaryToken     string
	TokenBreakdowns  []WalletSummaryCounterpartyTokenSummary
	DirectionLabel   string
	FirstSeenAt      *time.Time
	LatestActivityAt *time.Time
}

type WalletSummaryCounterpartyTokenSummary struct {
	Symbol         string
	InboundAmount  string
	OutboundAmount string
}

type WalletGraphSignals struct {
	ClusterKey            string
	ClusterType           string
	ClusterScore          int
	ClusterMemberCount    int64
	InteractedWalletCount int64
	BridgeTransferCount   int64
	CEXProximityCount     int64
}

type ClusterScoreSnapshot struct {
	SignalType  string
	ScoreValue  int
	ScoreRating domain.ScoreRating
	ObservedAt  time.Time
}

type ShadowExitSnapshot struct {
	SignalType  string
	ScoreValue  int
	ScoreRating domain.ScoreRating
	ObservedAt  time.Time
}

type FirstConnectionSnapshot struct {
	SignalType  string
	ScoreValue  int
	ScoreRating domain.ScoreRating
	ObservedAt  time.Time
}

type WalletLatestSignalsReader interface {
	ReadLatestWalletSignals(context.Context, string) ([]domain.WalletLatestSignal, error)
}

type WalletSummaryCacheState struct {
	Key         string
	Hit         bool
	Source      string
	CollectedAt time.Time
	ExpiresAt   time.Time
}

type WalletSummaryInputs struct {
	Ref                     WalletRef
	Identity                WalletSummaryIdentity
	Stats                   WalletSummaryStats
	Signals                 WalletGraphSignals
	Enrichment              *domain.WalletEnrichment
	ClusterScoreSnapshot    *ClusterScoreSnapshot
	ShadowExitSnapshot      *ShadowExitSnapshot
	FirstConnectionSnapshot *FirstConnectionSnapshot
	LatestSignals           []domain.WalletLatestSignal
	Cache                   WalletSummaryCacheState
}

type WalletSummaryQueryPlan struct {
	Ref           WalletRef
	CacheKey      string
	IdentitySQL   string
	IdentityArgs  []any
	StatsSQL      string
	StatsArgs     []any
	SignalsCypher string
	SignalsParams map[string]any
	CacheTTL      time.Duration
}

type WalletIdentityReader interface {
	ReadWalletIdentity(ctx context.Context, plan WalletSummaryQueryPlan) (WalletSummaryIdentity, error)
}

type WalletStatsReader interface {
	ReadWalletStats(ctx context.Context, plan WalletSummaryQueryPlan) (WalletSummaryStats, error)
}

type WalletGraphSignalReader interface {
	ReadWalletGraphSignals(ctx context.Context, plan WalletSummaryQueryPlan) (WalletGraphSignals, error)
}

type WalletEnrichmentReader interface {
	ReadWalletEnrichmentSnapshot(context.Context, WalletRef) (*domain.WalletEnrichment, error)
}

type ClusterScoreSnapshotReader interface {
	ReadLatestClusterScoreSnapshot(ctx context.Context, walletID string) (*ClusterScoreSnapshot, error)
}

type ShadowExitSnapshotReader interface {
	ReadLatestShadowExitSnapshot(ctx context.Context, walletID string) (*ShadowExitSnapshot, error)
}

type FirstConnectionSnapshotReader interface {
	ReadLatestFirstConnectionSnapshot(ctx context.Context, walletID string) (*FirstConnectionSnapshot, error)
}

type WalletSummaryCache interface {
	GetWalletSummaryInputs(ctx context.Context, key string) (WalletSummaryInputs, bool, error)
	SetWalletSummaryInputs(ctx context.Context, key string, inputs WalletSummaryInputs, ttl time.Duration) error
	DeleteWalletSummaryInputs(ctx context.Context, key string) error
}

type WalletSummaryRepository struct {
	IdentityReader                WalletIdentityReader
	StatsReader                   WalletStatsReader
	GraphSignalReader             WalletGraphSignalReader
	LabelReader                   WalletLabelReader
	EnrichmentReader              WalletEnrichmentReader
	ClusterScoreSnapshotReader    ClusterScoreSnapshotReader
	ShadowExitSnapshotReader      ShadowExitSnapshotReader
	FirstConnectionSnapshotReader FirstConnectionSnapshotReader
	LatestSignalsReader           WalletLatestSignalsReader
	Cache                         WalletSummaryCache
	CacheTTL                      time.Duration
	Now                           func() time.Time
}

var ErrWalletSummaryNotFound = errors.New("wallet summary not found")

func NewWalletSummaryRepository(
	identityReader WalletIdentityReader,
	statsReader WalletStatsReader,
	graphSignalReader WalletGraphSignalReader,
	labelReader WalletLabelReader,
	enrichmentReader WalletEnrichmentReader,
	clusterScoreSnapshotReader ClusterScoreSnapshotReader,
	shadowExitSnapshotReader ShadowExitSnapshotReader,
	firstConnectionSnapshotReader FirstConnectionSnapshotReader,
	latestSignalsReader WalletLatestSignalsReader,
	cache WalletSummaryCache,
	cacheTTL time.Duration,
) *WalletSummaryRepository {
	return &WalletSummaryRepository{
		IdentityReader:                identityReader,
		StatsReader:                   statsReader,
		GraphSignalReader:             graphSignalReader,
		LabelReader:                   labelReader,
		EnrichmentReader:              enrichmentReader,
		ClusterScoreSnapshotReader:    clusterScoreSnapshotReader,
		ShadowExitSnapshotReader:      shadowExitSnapshotReader,
		FirstConnectionSnapshotReader: firstConnectionSnapshotReader,
		LatestSignalsReader:           latestSignalsReader,
		Cache:                         cache,
		CacheTTL:                      cacheTTL,
		Now:                           time.Now,
	}
}

func BuildWalletSummaryQueryPlan(ref WalletRef, cacheTTL time.Duration) (WalletSummaryQueryPlan, error) {
	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return WalletSummaryQueryPlan{}, err
	}

	return WalletSummaryQueryPlan{
		Ref:      normalized,
		CacheKey: BuildWalletSummaryCacheKey(normalized),
		IdentitySQL: `
SELECT
  w.id,
  w.chain,
  w.address,
  w.display_name,
  COALESCE(w.entity_key, '') AS entity_key,
  w.created_at,
  w.updated_at
FROM wallets w
WHERE w.chain = $1 AND w.address = $2
LIMIT 1
`,
		IdentityArgs: []any{string(normalized.Chain), normalized.Address},
		StatsSQL: `
WITH root AS (
  SELECT w.id, w.updated_at
  FROM wallets w
  WHERE w.chain = $1 AND w.address = $2
  LIMIT 1
),
daily_snapshots AS (
  SELECT
    wds.wallet_id,
    wds.as_of_date,
    wds.transaction_count,
    wds.counterparty_count,
    wds.latest_activity_at,
    wds.incoming_tx_count,
    wds.outgoing_tx_count
  FROM wallet_daily_stats wds
  JOIN root r ON r.id = wds.wallet_id
),
latest_snapshot AS (
  SELECT
    wallet_id,
    as_of_date,
    transaction_count,
    counterparty_count,
    latest_activity_at,
    incoming_tx_count,
    outgoing_tx_count
  FROM daily_snapshots
  ORDER BY as_of_date DESC
  LIMIT 1
),
baseline_7d AS (
  SELECT
    incoming_tx_count,
    outgoing_tx_count
  FROM daily_snapshots
  WHERE as_of_date < (SELECT as_of_date FROM latest_snapshot) - 6
  ORDER BY as_of_date DESC
  LIMIT 1
),
baseline_30d AS (
  SELECT
    incoming_tx_count,
    outgoing_tx_count
  FROM daily_snapshots
  WHERE as_of_date < (SELECT as_of_date FROM latest_snapshot) - 29
  ORDER BY as_of_date DESC
  LIMIT 1
),
tx_base AS (
  SELECT
    t.id,
    t.wallet_id,
    t.direction,
    t.counterparty_chain,
    nullif(t.counterparty_address, '') AS counterparty_address,
    COALESCE(t.amount_numeric, 0::numeric) AS amount_numeric,
    COALESCE(NULLIF(t.token_symbol, ''), '') AS token_symbol,
    t.observed_at
  FROM transactions t
  JOIN root r ON r.id = t.wallet_id
),
counterparty_token_rollup AS (
  SELECT
    coalesce(counterparty_chain, '') AS chain,
    counterparty_address AS address,
    coalesce(nullif(token_symbol, ''), 'token') AS token_symbol,
    COALESCE(sum(CASE WHEN direction = 'inbound' THEN amount_numeric ELSE 0 END), 0::numeric) AS inbound_amount,
    COALESCE(sum(CASE WHEN direction = 'outbound' THEN amount_numeric ELSE 0 END), 0::numeric) AS outbound_amount,
    COALESCE(sum(amount_numeric), 0::numeric) AS total_amount
  FROM tx_base
  WHERE counterparty_address IS NOT NULL
  GROUP BY coalesce(counterparty_chain, ''), counterparty_address, coalesce(nullif(token_symbol, ''), 'token')
),
counterparty_primary_token AS (
  SELECT DISTINCT ON (ctr.chain, ctr.address)
    ctr.chain,
    ctr.address,
    ctr.token_symbol AS primary_token
  FROM counterparty_token_rollup ctr
  ORDER BY ctr.chain ASC, ctr.address ASC, ctr.total_amount DESC, ctr.token_symbol ASC
),
counterparty_token_json AS (
  SELECT
    ctr.chain,
    ctr.address,
    jsonb_agg(jsonb_build_object(
      'symbol', ctr.token_symbol,
      'inbound_amount', ctr.inbound_amount::text,
      'outbound_amount', ctr.outbound_amount::text
    ) ORDER BY ctr.total_amount DESC, ctr.token_symbol ASC) AS token_breakdowns
  FROM counterparty_token_rollup ctr
  GROUP BY ctr.chain, ctr.address
),
activity_bounds AS (
  SELECT
    min(observed_at) AS earliest_activity_at,
    max(observed_at) AS latest_activity_at
  FROM tx_base
),
counterparty_rollup_base AS (
  SELECT
    coalesce(tx.counterparty_chain, '') AS chain,
    tx.counterparty_address AS address,
    COALESCE(w.entity_key, '') AS entity_key,
    COALESCE(e.entity_type, '') AS entity_type,
    COALESCE(NULLIF(e.display_name, ''), '') AS entity_label,
    count(*) AS interaction_count,
    count(*) FILTER (WHERE tx.direction = 'inbound') AS inbound_count,
    count(*) FILTER (WHERE tx.direction = 'outbound') AS outbound_count,
    COALESCE(sum(CASE WHEN tx.direction = 'inbound' THEN tx.amount_numeric ELSE 0 END), 0::numeric)::text AS inbound_amount,
    COALESCE(sum(CASE WHEN tx.direction = 'outbound' THEN tx.amount_numeric ELSE 0 END), 0::numeric)::text AS outbound_amount,
    min(tx.observed_at) AS first_seen_at,
    max(tx.observed_at) AS latest_activity_at
  FROM tx_base tx
  LEFT JOIN wallets w
    ON w.chain = coalesce(tx.counterparty_chain, '')
   AND w.address = tx.counterparty_address
  LEFT JOIN entities e
    ON e.entity_key = w.entity_key
  WHERE tx.counterparty_address IS NOT NULL
  GROUP BY coalesce(tx.counterparty_chain, ''), tx.counterparty_address, w.entity_key, e.entity_type, e.display_name
),
counterparty_rollup AS (
  SELECT
    base.chain,
    base.address,
    base.entity_key,
    base.entity_type,
    base.entity_label,
    base.interaction_count,
    base.inbound_count,
    base.outbound_count,
    base.inbound_amount,
    base.outbound_amount,
    COALESCE(primary_token.primary_token, '') AS primary_token,
    COALESCE(token_json.token_breakdowns, '[]'::jsonb) AS token_breakdowns,
    base.first_seen_at,
    base.latest_activity_at
  FROM counterparty_rollup_base base
  LEFT JOIN counterparty_primary_token primary_token
    ON primary_token.chain = base.chain
   AND primary_token.address = base.address
  LEFT JOIN counterparty_token_json token_json
    ON token_json.chain = base.chain
   AND token_json.address = base.address
  ORDER BY interaction_count DESC, latest_activity_at DESC, address ASC
  LIMIT 5
)
SELECT
  r.id AS wallet_id,
  COALESCE(ls.as_of_date, date(ab.latest_activity_at), r.updated_at::date) AS as_of_date,
  COALESCE(ls.transaction_count, count(t.id)::integer, 0) AS transaction_count,
  COALESCE(ls.counterparty_count, count(DISTINCT nullif(t.counterparty_address, ''))::integer, 0) AS counterparty_count,
  ab.earliest_activity_at,
  COALESCE(ls.latest_activity_at, ab.latest_activity_at) AS latest_activity_at,
  COALESCE(ls.incoming_tx_count, count(*) FILTER (WHERE t.direction = 'inbound')::integer, 0) AS incoming_tx_count,
  COALESCE(ls.outgoing_tx_count, count(*) FILTER (WHERE t.direction = 'outbound')::integer, 0) AS outgoing_tx_count,
  COALESCE(
    ls.incoming_tx_count - COALESCE(b7.incoming_tx_count, 0),
    count(*) FILTER (WHERE t.direction = 'inbound' AND t.observed_at >= now() - interval '7 days')::integer,
    0
  ) AS incoming_tx_count_7d,
  COALESCE(
    ls.outgoing_tx_count - COALESCE(b7.outgoing_tx_count, 0),
    count(*) FILTER (WHERE t.direction = 'outbound' AND t.observed_at >= now() - interval '7 days')::integer,
    0
  ) AS outgoing_tx_count_7d,
  COALESCE(
    ls.incoming_tx_count - COALESCE(b30.incoming_tx_count, 0),
    count(*) FILTER (WHERE t.direction = 'inbound' AND t.observed_at >= now() - interval '30 days')::integer,
    0
  ) AS incoming_tx_count_30d,
  COALESCE(
    ls.outgoing_tx_count - COALESCE(b30.outgoing_tx_count, 0),
    count(*) FILTER (WHERE t.direction = 'outbound' AND t.observed_at >= now() - interval '30 days')::integer,
    0
  ) AS outgoing_tx_count_30d,
  COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
      'chain', chain,
      'address', address,
      'entity_key', entity_key,
      'entity_type', entity_type,
      'entity_label', entity_label,
      'interaction_count', interaction_count,
      'inbound_count', inbound_count,
      'outbound_count', outbound_count,
      'inbound_amount', inbound_amount,
      'outbound_amount', outbound_amount,
      'primary_token', primary_token,
      'token_breakdowns', token_breakdowns,
      'direction_label', CASE
        WHEN outbound_count > inbound_count THEN 'outbound'
        WHEN inbound_count > outbound_count THEN 'inbound'
        ELSE 'mixed'
      END,
      'first_seen_at', to_char(first_seen_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
      'latest_activity_at', to_char(latest_activity_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
    ))
    FROM counterparty_rollup
  ), '[]'::jsonb) AS top_counterparties
FROM root r
LEFT JOIN latest_snapshot ls ON ls.wallet_id = r.id
LEFT JOIN baseline_7d b7 ON true
LEFT JOIN baseline_30d b30 ON true
LEFT JOIN tx_base t ON true
LEFT JOIN activity_bounds ab ON true
GROUP BY
  r.id,
  r.updated_at,
  ls.as_of_date,
  ls.transaction_count,
  ls.counterparty_count,
  ls.latest_activity_at,
  ls.incoming_tx_count,
  ls.outgoing_tx_count,
  b7.incoming_tx_count,
  b7.outgoing_tx_count,
  b30.incoming_tx_count,
  b30.outgoing_tx_count,
  ab.earliest_activity_at,
  ab.latest_activity_at
LIMIT 1
`,
		StatsArgs: []any{string(normalized.Chain), normalized.Address},
		SignalsCypher: `
MATCH (w:Wallet {chain: $chain, address: $address})
OPTIONAL MATCH (w)-[:MEMBER_OF]->(c:Cluster)
OPTIONAL MATCH (c)<-[:MEMBER_OF]-(member:Wallet)
OPTIONAL MATCH (w)-[interacted:INTERACTED_WITH]->(:Wallet)
OPTIONAL MATCH (w)-[bridged:BRIDGED_TO]->(:Wallet)
OPTIONAL MATCH (w)-[cex:CEX_PROXIMITY]->(:Entity)
RETURN
  coalesce(c.clusterKey, '') AS clusterKey,
  coalesce(c.clusterType, '') AS clusterType,
  coalesce(c.clusterScore, 0) AS clusterScore,
  count(DISTINCT member) AS clusterMemberCount,
  count(DISTINCT interacted) AS interactedWalletCount,
  count(DISTINCT bridged) AS bridgeTransferCount,
  count(DISTINCT cex) AS cexProximityCount
LIMIT 1
`,
		SignalsParams: map[string]any{
			"chain":   string(normalized.Chain),
			"address": normalized.Address,
		},
		CacheTTL: cacheTTL,
	}, nil
}

func (r *WalletSummaryRepository) LoadWalletSummaryInputs(ctx context.Context, ref WalletRef) (WalletSummaryInputs, error) {
	if r == nil {
		return WalletSummaryInputs{}, fmt.Errorf("wallet summary repository is nil")
	}

	ttl := r.CacheTTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	now := r.now().UTC()
	plan, err := BuildWalletSummaryQueryPlan(ref, ttl)
	if err != nil {
		return WalletSummaryInputs{}, err
	}

	if r.Cache != nil {
		if cached, ok, err := r.Cache.GetWalletSummaryInputs(ctx, plan.CacheKey); err != nil {
			return WalletSummaryInputs{}, fmt.Errorf("load wallet summary cache: %w", err)
		} else if ok {
			cached.Ref = plan.Ref
			cached.Cache.Key = plan.CacheKey
			cached.Cache.Hit = true
			cached.Cache.Source = "cache"
			if cached.Cache.CollectedAt.IsZero() {
				cached.Cache.CollectedAt = now
			}
			return cached, nil
		}
	}

	identity, err := r.readIdentity(ctx, plan)
	if err != nil {
		return WalletSummaryInputs{}, err
	}

	stats, err := r.readStats(ctx, plan)
	if err != nil {
		return WalletSummaryInputs{}, err
	}

	if err := r.readLabels(ctx, &identity, &stats); err != nil {
		return WalletSummaryInputs{}, err
	}

	signals, err := r.readSignals(ctx, plan)
	if err != nil {
		return WalletSummaryInputs{}, err
	}
	enrichment, err := r.readEnrichment(ctx, plan.Ref)
	if err != nil {
		return WalletSummaryInputs{}, err
	}

	clusterScoreSnapshot, err := r.readClusterScoreSnapshot(ctx, identity.WalletID)
	if err != nil {
		return WalletSummaryInputs{}, err
	}

	shadowExitSnapshot, err := r.readShadowExitSnapshot(ctx, identity.WalletID)
	if err != nil {
		return WalletSummaryInputs{}, err
	}

	firstConnectionSnapshot, err := r.readFirstConnectionSnapshot(ctx, identity.WalletID)
	if err != nil {
		return WalletSummaryInputs{}, err
	}

	latestSignals, err := r.readLatestSignals(ctx, identity.WalletID)
	if err != nil {
		return WalletSummaryInputs{}, err
	}

	inputs := WalletSummaryInputs{
		Ref:                     plan.Ref,
		Identity:                identity,
		Stats:                   stats,
		Signals:                 signals,
		Enrichment:              enrichment,
		ClusterScoreSnapshot:    clusterScoreSnapshot,
		ShadowExitSnapshot:      shadowExitSnapshot,
		FirstConnectionSnapshot: firstConnectionSnapshot,
		LatestSignals:           latestSignals,
		Cache: WalletSummaryCacheState{
			Key:         plan.CacheKey,
			Hit:         false,
			Source:      "aggregate",
			CollectedAt: now,
			ExpiresAt:   now.Add(ttl),
		},
	}

	if r.Cache != nil {
		if err := r.Cache.SetWalletSummaryInputs(ctx, plan.CacheKey, inputs, ttl); err != nil {
			return WalletSummaryInputs{}, fmt.Errorf("store wallet summary cache: %w", err)
		}
	}

	return inputs, nil
}

func (r *WalletSummaryRepository) readIdentity(ctx context.Context, plan WalletSummaryQueryPlan) (WalletSummaryIdentity, error) {
	if r.IdentityReader == nil {
		return WalletSummaryIdentity{}, fmt.Errorf("wallet identity reader is nil")
	}

	identity, err := r.IdentityReader.ReadWalletIdentity(ctx, plan)
	if err != nil {
		return WalletSummaryIdentity{}, fmt.Errorf("read wallet identity: %w", err)
	}

	return identity, nil
}

func (r *WalletSummaryRepository) readStats(ctx context.Context, plan WalletSummaryQueryPlan) (WalletSummaryStats, error) {
	if r.StatsReader == nil {
		return WalletSummaryStats{}, fmt.Errorf("wallet stats reader is nil")
	}

	stats, err := r.StatsReader.ReadWalletStats(ctx, plan)
	if err != nil {
		return WalletSummaryStats{}, fmt.Errorf("read wallet stats: %w", err)
	}

	return stats, nil
}

func (r *WalletSummaryRepository) readSignals(ctx context.Context, plan WalletSummaryQueryPlan) (WalletGraphSignals, error) {
	if r.GraphSignalReader == nil {
		return WalletGraphSignals{}, fmt.Errorf("wallet graph signal reader is nil")
	}

	signals, err := r.GraphSignalReader.ReadWalletGraphSignals(ctx, plan)
	if err != nil {
		if errors.Is(err, ErrWalletSummaryNotFound) {
			return WalletGraphSignals{}, nil
		}
		return WalletGraphSignals{}, fmt.Errorf("read wallet graph signals: %w", err)
	}

	return signals, nil
}

func (r *WalletSummaryRepository) readLabels(
	ctx context.Context,
	identity *WalletSummaryIdentity,
	stats *WalletSummaryStats,
) error {
	if r.LabelReader == nil || identity == nil || stats == nil {
		return nil
	}

	refs := make([]WalletRef, 0, len(stats.TopCounterparties)+1)
	refs = append(refs, WalletRef{
		Chain:   identity.Chain,
		Address: identity.Address,
	})
	for _, counterparty := range stats.TopCounterparties {
		refs = append(refs, WalletRef{
			Chain:   counterparty.Chain,
			Address: counterparty.Address,
		})
	}

	labelsByRef, err := r.LabelReader.ReadWalletLabels(ctx, refs)
	if err != nil {
		return fmt.Errorf("read wallet labels: %w", err)
	}

	identity.Labels = labelsByRef[walletRefLabelKey(WalletRef{
		Chain:   identity.Chain,
		Address: identity.Address,
	})]
	for index := range stats.TopCounterparties {
		key := walletRefLabelKey(WalletRef{
			Chain:   stats.TopCounterparties[index].Chain,
			Address: stats.TopCounterparties[index].Address,
		})
		stats.TopCounterparties[index].Labels = labelsByRef[key]
	}

	return nil
}

func (r *WalletSummaryRepository) readEnrichment(
	ctx context.Context,
	ref WalletRef,
) (*domain.WalletEnrichment, error) {
	if r.EnrichmentReader == nil {
		return nil, nil
	}

	enrichment, err := r.EnrichmentReader.ReadWalletEnrichmentSnapshot(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("read wallet enrichment snapshot: %w", err)
	}

	return enrichment, nil
}

func (r *WalletSummaryRepository) readClusterScoreSnapshot(
	ctx context.Context,
	walletID string,
) (*ClusterScoreSnapshot, error) {
	if r.ClusterScoreSnapshotReader == nil {
		return nil, nil
	}

	snapshot, err := r.ClusterScoreSnapshotReader.ReadLatestClusterScoreSnapshot(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("read cluster score snapshot: %w", err)
	}

	return snapshot, nil
}

func (r *WalletSummaryRepository) readShadowExitSnapshot(
	ctx context.Context,
	walletID string,
) (*ShadowExitSnapshot, error) {
	if r.ShadowExitSnapshotReader == nil {
		return nil, nil
	}

	snapshot, err := r.ShadowExitSnapshotReader.ReadLatestShadowExitSnapshot(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("read shadow exit snapshot: %w", err)
	}

	return snapshot, nil
}

func (r *WalletSummaryRepository) readFirstConnectionSnapshot(
	ctx context.Context,
	walletID string,
) (*FirstConnectionSnapshot, error) {
	if r.FirstConnectionSnapshotReader == nil {
		return nil, nil
	}

	snapshot, err := r.FirstConnectionSnapshotReader.ReadLatestFirstConnectionSnapshot(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("read first connection snapshot: %w", err)
	}

	return snapshot, nil
}

func (r *WalletSummaryRepository) readLatestSignals(
	ctx context.Context,
	walletID string,
) ([]domain.WalletLatestSignal, error) {
	if r.LatestSignalsReader == nil {
		return nil, nil
	}

	signals, err := r.LatestSignalsReader.ReadLatestWalletSignals(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("read latest signals: %w", err)
	}

	return signals, nil
}

func (r *WalletSummaryRepository) now() time.Time {
	if r != nil && r.Now != nil {
		return r.Now()
	}

	return time.Now()
}

func NormalizeWalletRef(ref WalletRef) (WalletRef, error) {
	chain := domain.Chain(strings.ToLower(strings.TrimSpace(string(ref.Chain))))
	switch chain {
	case domain.ChainEVM, domain.ChainSolana:
	default:
		return WalletRef{}, fmt.Errorf("unsupported chain %q", ref.Chain)
	}

	address := strings.TrimSpace(ref.Address)
	if address == "" {
		return WalletRef{}, fmt.Errorf("wallet address is required")
	}

	return WalletRef{
		Chain:   chain,
		Address: address,
	}, nil
}

func BuildWalletSummaryCacheKey(ref WalletRef) string {
	return fmt.Sprintf("wallet-summary:%s:%s", strings.ToLower(string(ref.Chain)), strings.TrimSpace(ref.Address))
}

func InvalidateWalletSummaryCache(
	ctx context.Context,
	cache WalletSummaryCache,
	ref WalletRef,
) error {
	if cache == nil {
		return nil
	}

	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return err
	}

	return cache.DeleteWalletSummaryInputs(ctx, BuildWalletSummaryCacheKey(normalized))
}
