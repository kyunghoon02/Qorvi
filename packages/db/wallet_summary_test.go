package db

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
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

type fakeWalletCache struct {
	getInputs WalletSummaryInputs
	getOK     bool
	getKey    string
	setKey    string
	setInputs WalletSummaryInputs
	setTTL    time.Duration
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
	if !contains(plan.StatsSQL, "LEFT JOIN transactions") {
		t.Fatalf("expected stats SQL to aggregate transactions")
	}
	if !contains(plan.StatsSQL, "counterparty_address") {
		t.Fatalf("expected stats SQL to use counterparty aggregation")
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
				AsOfDate:          latest,
				TransactionCount:  42,
				CounterpartyCount: 18,
				LatestActivityAt:  &latest,
				IncomingTxCount:   13,
				OutgoingTxCount:   29,
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

func contains(value, fragment string) bool {
	return strings.Contains(value, fragment)
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
