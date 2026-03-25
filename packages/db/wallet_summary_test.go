package db

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

type fakeWalletIdentityReader struct {
	called bool
}

func (f *fakeWalletIdentityReader) ReadWalletIdentity(context.Context, WalletSummaryQueryPlan) (WalletSummaryIdentity, error) {
	f.called = true
	return WalletSummaryIdentity{}, errors.New("unexpected identity lookup")
}

type fakeWalletStatsReader struct {
	called bool
}

func (f *fakeWalletStatsReader) ReadWalletStats(context.Context, WalletSummaryQueryPlan) (WalletSummaryStats, error) {
	f.called = true
	return WalletSummaryStats{}, errors.New("unexpected stats lookup")
}

type fakeWalletSignalReader struct {
	called bool
}

func (f *fakeWalletSignalReader) ReadWalletGraphSignals(context.Context, WalletSummaryQueryPlan) (WalletGraphSignals, error) {
	f.called = true
	return WalletGraphSignals{}, errors.New("unexpected signal lookup")
}

type fakeWalletEnrichmentReader struct {
	called     bool
	enrichment *domain.WalletEnrichment
}

func (f *fakeWalletEnrichmentReader) ReadWalletEnrichmentSnapshot(context.Context, WalletRef) (*domain.WalletEnrichment, error) {
	f.called = true
	return f.enrichment, nil
}

type fakeClusterScoreSnapshotReader struct {
	called   bool
	snapshot *ClusterScoreSnapshot
}

func (f *fakeClusterScoreSnapshotReader) ReadLatestClusterScoreSnapshot(context.Context, string) (*ClusterScoreSnapshot, error) {
	f.called = true
	return f.snapshot, nil
}

type fakeShadowExitSnapshotReader struct {
	called   bool
	snapshot *ShadowExitSnapshot
}

func (f *fakeShadowExitSnapshotReader) ReadLatestShadowExitSnapshot(context.Context, string) (*ShadowExitSnapshot, error) {
	f.called = true
	return f.snapshot, nil
}

type fakeFirstConnectionSnapshotReader struct {
	called   bool
	snapshot *FirstConnectionSnapshot
}

func (f *fakeFirstConnectionSnapshotReader) ReadLatestFirstConnectionSnapshot(context.Context, string) (*FirstConnectionSnapshot, error) {
	f.called = true
	return f.snapshot, nil
}

type fakeWalletLatestSignalsReader struct {
	called  bool
	signals []domain.WalletLatestSignal
}

func (f *fakeWalletLatestSignalsReader) ReadLatestWalletSignals(context.Context, string) ([]domain.WalletLatestSignal, error) {
	f.called = true
	items := make([]domain.WalletLatestSignal, len(f.signals))
	copy(items, f.signals)
	return items, nil
}

type fakeWalletCache struct {
	getInputs WalletSummaryInputs
	getOK     bool
	getKey    string
	setKey    string
	setInputs WalletSummaryInputs
	setTTL    time.Duration
	deleteKey string
}

func (f *fakeWalletCache) GetWalletSummaryInputs(_ context.Context, key string) (WalletSummaryInputs, bool, error) {
	f.getKey = key
	return f.getInputs, f.getOK, nil
}

func (f *fakeWalletCache) SetWalletSummaryInputs(_ context.Context, key string, inputs WalletSummaryInputs, ttl time.Duration) error {
	f.setKey = key
	f.setInputs = inputs
	f.setTTL = ttl
	return nil
}

func (f *fakeWalletCache) DeleteWalletSummaryInputs(_ context.Context, key string) error {
	f.deleteKey = key
	return nil
}

func TestBuildWalletSummaryQueryPlan(t *testing.T) {
	t.Parallel()

	plan, err := BuildWalletSummaryQueryPlan(WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, 5*time.Minute)
	if err != nil {
		t.Fatalf("expected plan, got error: %v", err)
	}

	if plan.CacheKey != "wallet-summary:evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected cache key %q", plan.CacheKey)
	}

	if len(plan.IdentityArgs) != 2 || plan.IdentityArgs[0] != "evm" {
		t.Fatalf("unexpected identity args %#v", plan.IdentityArgs)
	}

	if plan.SignalsParams["address"] != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected graph params %#v", plan.SignalsParams)
	}
	if !contains(plan.StatsSQL, "FROM transactions") {
		t.Fatalf("expected stats SQL to aggregate transactions")
	}
	if !contains(plan.StatsSQL, "jsonb_agg") {
		t.Fatalf("expected stats SQL to aggregate top counterparties")
	}
	if !contains(plan.StatsSQL, "interval '7 days'") {
		t.Fatalf("expected stats SQL to aggregate recent flow windows")
	}
	if !contains(plan.StatsSQL, "counterparty_address") {
		t.Fatalf("expected stats SQL to use counterparty aggregation")
	}
	if !contains(plan.StatsSQL, "FROM wallet_daily_stats") {
		t.Fatalf("expected stats SQL to use wallet_daily_stats aggregate")
	}
	if !contains(plan.SignalsCypher, "INTERACTED_WITH") {
		t.Fatalf("expected graph cypher to reference graph signals")
	}
}

func TestLoadWalletSummaryInputsUsesCacheHitFirst(t *testing.T) {
	t.Parallel()

	cache := &fakeWalletCache{
		getOK: true,
		getInputs: WalletSummaryInputs{
			Ref: WalletRef{
				Chain:   domain.ChainEVM,
				Address: "0x1234567890abcdef1234567890abcdef12345678",
			},
		},
	}
	identityReader := &fakeWalletIdentityReader{}
	statsReader := &fakeWalletStatsReader{}
	signalReader := &fakeWalletSignalReader{}
		repo := NewWalletSummaryRepository(
			identityReader,
			statsReader,
			signalReader,
			nil,
			nil,
			nil,
			nil,
		nil,
		nil,
		cache,
		5*time.Minute,
	)

	inputs, err := repo.LoadWalletSummaryInputs(context.Background(), WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	})
	if err != nil {
		t.Fatalf("expected cache hit to succeed, got %v", err)
	}

	if !inputs.Cache.Hit || inputs.Cache.Source != "cache" {
		t.Fatalf("expected cache hit metadata, got %#v", inputs.Cache)
	}
	if identityReader.called || statsReader.called || signalReader.called {
		t.Fatalf("expected cache hit to bypass readers")
	}

	if cache.getKey != "wallet-summary:evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected cache key %q", cache.getKey)
	}
}

func TestLoadWalletSummaryInputsAggregatesAndCaches(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)
	cache := &fakeWalletCache{}
		repo := NewWalletSummaryRepository(
			&stubWalletIdentityReader{
			identity: WalletSummaryIdentity{
				WalletID:    "wallet_1",
				Chain:       domain.ChainSolana,
				Address:     "7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
				DisplayName: "Seed Whale",
				EntityKey:   "entity_seed",
				CreatedAt:   latest.Add(-time.Hour),
				UpdatedAt:   latest.Add(-time.Minute),
			},
		},
		&stubWalletStatsReader{
			stats: WalletSummaryStats{
				AsOfDate:           latest,
				TransactionCount:   42,
				CounterpartyCount:  18,
				LatestActivityAt:   &latest,
				IncomingTxCount:    13,
				OutgoingTxCount:    29,
				IncomingTxCount7d:  4,
				OutgoingTxCount7d:  9,
				IncomingTxCount30d: 13,
				OutgoingTxCount30d: 29,
				TopCounterparties: []WalletSummaryCounterparty{
					{
						Chain:            domain.ChainSolana,
						Address:          "8h7z6y5x4w3v2u1t9s8r7q6p5o4n3m2l1k0j9h8g7f6d",
						InteractionCount: 9,
						InboundCount:     3,
						OutboundCount:    6,
						DirectionLabel:   "outbound",
						FirstSeenAt:      ptrWalletSummaryTime(latest.Add(-7 * 24 * time.Hour)),
						LatestActivityAt: &latest,
					},
				},
			},
		},
			&stubWalletSignalReader{
			signals: WalletGraphSignals{
				ClusterKey:            "cluster_seed_whales",
				ClusterType:           "whale",
				ClusterScore:          82,
				ClusterMemberCount:    7,
				InteractedWalletCount: 11,
				BridgeTransferCount:   1,
				CEXProximityCount:     2,
			},
			},
			nil,
			&fakeWalletEnrichmentReader{
			enrichment: &domain.WalletEnrichment{
				Provider:               "moralis",
				NetWorthUSD:            "157.00",
				NativeBalanceFormatted: "0.00402 ETH",
				ActiveChains:           []string{"Ethereum", "Base"},
				ActiveChainCount:       2,
				HoldingCount:           1,
				Source:                 "snapshot",
				UpdatedAt:              latest.Format(time.RFC3339),
			},
		},
		nil,
		nil,
		nil,
		&fakeWalletLatestSignalsReader{
			signals: []domain.WalletLatestSignal{
				{
					Name:       domain.ScoreCluster,
					Value:      82,
					Rating:     domain.RatingHigh,
					Label:      "latest cluster score snapshot",
					Source:     "cluster-score-snapshot",
					ObservedAt: latest.Format(time.RFC3339),
				},
			},
		},
		cache,
		10*time.Minute,
	)
	repo.Now = func() time.Time { return latest }

	inputs, err := repo.LoadWalletSummaryInputs(context.Background(), WalletRef{
		Chain:   domain.ChainSolana,
		Address: "7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
	})
	if err != nil {
		t.Fatalf("expected aggregate load to succeed, got %v", err)
	}

	if inputs.Identity.DisplayName != "Seed Whale" {
		t.Fatalf("unexpected identity %#v", inputs.Identity)
	}
	if inputs.Stats.TransactionCount != 42 {
		t.Fatalf("unexpected stats %#v", inputs.Stats)
	}
	if inputs.Signals.ClusterKey != "cluster_seed_whales" {
		t.Fatalf("unexpected signals %#v", inputs.Signals)
	}
	if len(inputs.LatestSignals) != 1 || inputs.LatestSignals[0].Source != "cluster-score-snapshot" {
		t.Fatalf("unexpected latest signals %#v", inputs.LatestSignals)
	}
	if inputs.Enrichment == nil || inputs.Enrichment.Source != "snapshot" {
		t.Fatalf("expected enrichment snapshot, got %#v", inputs.Enrichment)
	}
	if len(inputs.Stats.TopCounterparties) != 1 {
		t.Fatalf("unexpected top counterparties %#v", inputs.Stats.TopCounterparties)
	}
	if inputs.Cache.Hit {
		t.Fatalf("expected aggregate path, got cache hit %#v", inputs.Cache)
	}
	if cache.setKey != "wallet-summary:solana:7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq" {
		t.Fatalf("unexpected cache set key %q", cache.setKey)
	}
	if cache.setTTL != 10*time.Minute {
		t.Fatalf("unexpected cache ttl %s", cache.setTTL)
	}
}

func TestLoadWalletSummaryInputsIncludesClusterScoreSnapshot(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)
		repo := NewWalletSummaryRepository(
			&stubWalletIdentityReader{
			identity: WalletSummaryIdentity{
				WalletID:    "wallet_1",
				Chain:       domain.ChainEVM,
				Address:     "0x1234567890abcdef1234567890abcdef12345678",
				DisplayName: "Seed Whale",
				CreatedAt:   latest.Add(-time.Hour),
				UpdatedAt:   latest,
			},
		},
		&stubWalletStatsReader{
			stats: WalletSummaryStats{
				AsOfDate: latest,
			},
		},
			&stubWalletSignalReader{},
			nil,
			nil,
			&fakeClusterScoreSnapshotReader{
			snapshot: &ClusterScoreSnapshot{
				SignalType:  "cluster_score_snapshot",
				ScoreValue:  82,
				ScoreRating: domain.RatingHigh,
				ObservedAt:  latest,
			},
		},
		&fakeShadowExitSnapshotReader{
			snapshot: &ShadowExitSnapshot{
				SignalType:  "shadow_exit_snapshot",
				ScoreValue:  34,
				ScoreRating: domain.RatingMedium,
				ObservedAt:  latest,
			},
		},
		&fakeFirstConnectionSnapshotReader{
			snapshot: &FirstConnectionSnapshot{
				SignalType:  "first_connection_snapshot",
				ScoreValue:  61,
				ScoreRating: domain.RatingHigh,
				ObservedAt:  latest,
			},
		},
		nil,
		nil,
		5*time.Minute,
	)

	inputs, err := repo.LoadWalletSummaryInputs(context.Background(), WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	})
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if inputs.ClusterScoreSnapshot == nil {
		t.Fatal("expected cluster score snapshot")
	}
	if inputs.ClusterScoreSnapshot.ScoreValue != 82 {
		t.Fatalf("unexpected snapshot %#v", inputs.ClusterScoreSnapshot)
	}
	if inputs.ShadowExitSnapshot == nil {
		t.Fatal("expected shadow exit snapshot")
	}
	if inputs.ShadowExitSnapshot.ScoreValue != 34 {
		t.Fatalf("unexpected shadow exit snapshot %#v", inputs.ShadowExitSnapshot)
	}
	if inputs.FirstConnectionSnapshot == nil {
		t.Fatal("expected first connection snapshot")
	}
	if inputs.FirstConnectionSnapshot.ScoreValue != 61 {
		t.Fatalf("unexpected first connection snapshot %#v", inputs.FirstConnectionSnapshot)
	}
}

func contains(value, fragment string) bool {
	return strings.Contains(value, fragment)
}

func ptrWalletSummaryTime(value time.Time) *time.Time {
	return &value
}

type stubWalletIdentityReader struct {
	identity WalletSummaryIdentity
}

func (s *stubWalletIdentityReader) ReadWalletIdentity(context.Context, WalletSummaryQueryPlan) (WalletSummaryIdentity, error) {
	return s.identity, nil
}

type stubWalletStatsReader struct {
	stats WalletSummaryStats
}

func (s *stubWalletStatsReader) ReadWalletStats(context.Context, WalletSummaryQueryPlan) (WalletSummaryStats, error) {
	return s.stats, nil
}

type stubWalletSignalReader struct {
	signals WalletGraphSignals
}

func (s *stubWalletSignalReader) ReadWalletGraphSignals(context.Context, WalletSummaryQueryPlan) (WalletGraphSignals, error) {
	return s.signals, nil
}
