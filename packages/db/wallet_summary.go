package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
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
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WalletSummaryStats struct {
	AsOfDate          time.Time
	TransactionCount  int64
	CounterpartyCount int64
	LatestActivityAt  *time.Time
	IncomingTxCount   int64
	OutgoingTxCount   int64
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

type WalletSummaryCacheState struct {
	Key         string
	Hit         bool
	Source      string
	CollectedAt time.Time
	ExpiresAt   time.Time
}

type WalletSummaryInputs struct {
	Ref      WalletRef
	Identity WalletSummaryIdentity
	Stats    WalletSummaryStats
	Signals  WalletGraphSignals
	Cache    WalletSummaryCacheState
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

type WalletSummaryCache interface {
	GetWalletSummaryInputs(ctx context.Context, key string) (WalletSummaryInputs, bool, error)
	SetWalletSummaryInputs(ctx context.Context, key string, inputs WalletSummaryInputs, ttl time.Duration) error
}

type WalletSummaryRepository struct {
	IdentityReader    WalletIdentityReader
	StatsReader       WalletStatsReader
	GraphSignalReader WalletGraphSignalReader
	Cache             WalletSummaryCache
	CacheTTL          time.Duration
	Now               func() time.Time
}

var ErrWalletSummaryNotFound = errors.New("wallet summary not found")

func NewWalletSummaryRepository(
	identityReader WalletIdentityReader,
	statsReader WalletStatsReader,
	graphSignalReader WalletGraphSignalReader,
	cache WalletSummaryCache,
	cacheTTL time.Duration,
) *WalletSummaryRepository {
	return &WalletSummaryRepository{
		IdentityReader:    identityReader,
		StatsReader:       statsReader,
		GraphSignalReader: graphSignalReader,
		Cache:             cache,
		CacheTTL:          cacheTTL,
		Now:               time.Now,
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
SELECT
  w.id AS wallet_id,
  s.as_of_date,
  s.transaction_count,
  s.counterparty_count,
  s.latest_activity_at,
  s.incoming_tx_count,
  s.outgoing_tx_count
FROM wallets w
JOIN wallet_daily_stats s ON s.wallet_id = w.id
WHERE w.chain = $1 AND w.address = $2
ORDER BY s.as_of_date DESC
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

	signals, err := r.readSignals(ctx, plan)
	if err != nil {
		return WalletSummaryInputs{}, err
	}

	inputs := WalletSummaryInputs{
		Ref:      plan.Ref,
		Identity: identity,
		Stats:    stats,
		Signals:  signals,
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
		return WalletGraphSignals{}, fmt.Errorf("read wallet graph signals: %w", err)
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
