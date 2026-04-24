package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/config"
)

func TestAutoIndexLoopRunCycleRunsAllConfiguredSteps(t *testing.T) {
	t.Setenv("DUNE_SEED_EXPORT_PATH", "/tmp/qorvi-seeds.json")
	t.Setenv("FLOWINTEL_SEED_DISCOVERY_TOP_N", "12")
	t.Setenv("FLOWINTEL_SEED_DISCOVERY_MIN_CONFIDENCE", "0.85")
	t.Setenv("FLOWINTEL_WALLET_BACKFILL_DRAIN_LIMIT", "4")

	now := time.Date(2026, time.April, 13, 12, 0, 0, 0, time.UTC)
	service := AutoIndexLoopService{
		SeedWatchlist: func(_ context.Context, topN int, minConfidence float64) (SeedDiscoveryWatchlistReport, error) {
			if topN != 12 {
				t.Fatalf("unexpected topN %d", topN)
			}
			if minConfidence != 0.85 {
				t.Fatalf("unexpected minConfidence %f", minConfidence)
			}
			return SeedDiscoveryWatchlistReport{
				CandidatesSelected:  3,
				WatchlistItemsAdded: 2,
			}, nil
		},
		SeedEnqueue: func(_ context.Context) (SeedDiscoveryIngestReport, error) {
			return SeedDiscoveryIngestReport{
				CandidatesEnqueued: 2,
				CandidatesDeduped:  1,
			}, nil
		},
		WatchlistBootstrap: func(_ context.Context) (WatchlistBootstrapReport, error) {
			return WatchlistBootstrapReport{
				WalletsSeen:     5,
				WalletsEnqueued: 3,
				WalletsDeduped:  2,
			}, nil
		},
		DrainBatch: func(_ context.Context, limit int) (QueuedWalletBackfillBatchReport, error) {
			if limit != 4 {
				t.Fatalf("unexpected backfill drain limit %d", limit)
			}
			return QueuedWalletBackfillBatchReport{
				JobsProcessed:       4,
				ActivitiesFetched:   18,
				TransactionsWritten: 11,
			}, nil
		},
		Now: func() time.Time { return now },
	}

	report, err := service.RunCycle(context.Background(), &autoIndexLoopState{})
	if err != nil {
		t.Fatalf("RunCycle returned error: %v", err)
	}
	if !report.SeedDiscoveryEnabled {
		t.Fatalf("expected seed discovery to be enabled")
	}
	if !report.SeedWatchlistRan || !report.SeedEnqueueRan {
		t.Fatalf("expected seed discovery steps to run, got %#v", report)
	}
	if !report.WatchlistBootstrapRan || !report.BackfillDrainRan {
		t.Fatalf("expected watchlist/bootstrap steps to run, got %#v", report)
	}
}

func TestAutoIndexLoopRunCycleSkipsSeedDiscoveryWithoutSource(t *testing.T) {
	service := AutoIndexLoopService{
		SeedWatchlist: func(_ context.Context, _ int, _ float64) (SeedDiscoveryWatchlistReport, error) {
			t.Fatalf("seed watchlist should not run without a configured source")
			return SeedDiscoveryWatchlistReport{}, nil
		},
		SeedEnqueue: func(_ context.Context) (SeedDiscoveryIngestReport, error) {
			t.Fatalf("seed enqueue should not run without a configured source")
			return SeedDiscoveryIngestReport{}, nil
		},
		WatchlistBootstrap: func(_ context.Context) (WatchlistBootstrapReport, error) {
			return WatchlistBootstrapReport{WalletsSeen: 1}, nil
		},
		DrainBatch: func(_ context.Context, _ int) (QueuedWalletBackfillBatchReport, error) {
			return QueuedWalletBackfillBatchReport{JobsProcessed: 1}, nil
		},
		Now: func() time.Time {
			return time.Date(2026, time.April, 13, 12, 0, 0, 0, time.UTC)
		},
	}

	report, err := service.RunCycle(context.Background(), &autoIndexLoopState{})
	if err != nil {
		t.Fatalf("RunCycle returned error: %v", err)
	}
	if report.SeedDiscoveryEnabled {
		t.Fatalf("expected seed discovery to be disabled without a configured source")
	}
	if report.SeedWatchlistRan || report.SeedEnqueueRan {
		t.Fatalf("expected seed steps to be skipped, got %#v", report)
	}
}

func TestAutoIndexLoopRunCycleContinuesWhenBackfillFails(t *testing.T) {
	service := AutoIndexLoopService{
		WatchlistBootstrap: func(_ context.Context) (WatchlistBootstrapReport, error) {
			return WatchlistBootstrapReport{WalletsSeen: 1, WalletsEnqueued: 1}, nil
		},
		DrainBatch: func(_ context.Context, _ int) (QueuedWalletBackfillBatchReport, error) {
			return QueuedWalletBackfillBatchReport{}, context.DeadlineExceeded
		},
		Now: func() time.Time {
			return time.Date(2026, time.April, 13, 12, 0, 0, 0, time.UTC)
		},
	}

	report, err := service.RunCycle(context.Background(), &autoIndexLoopState{})
	if err != nil {
		t.Fatalf("RunCycle returned error: %v", err)
	}
	if report.BackfillDrainError == "" {
		t.Fatalf("expected backfill error to be captured, got %#v", report)
	}
	if !report.WatchlistBootstrapRan {
		t.Fatalf("expected watchlist bootstrap to still run, got %#v", report)
	}
}

func TestBuildWorkerOutputWithAutoIndexRunsLoopMode(t *testing.T) {
	t.Setenv("FLOWINTEL_AUTO_INDEX_RUN_ONCE", "true")

	output, err := buildWorkerOutputWithAutoIndex(
		t.Context(),
		workerModeAutoIndexLoop,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		HistoricalBackfillJobRunner{},
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
		BillingSubscriptionSyncService{},
		AutoIndexLoopService{
			WatchlistBootstrap: func(_ context.Context) (WatchlistBootstrapReport, error) {
				return WatchlistBootstrapReport{WalletsSeen: 2, WalletsEnqueued: 2}, nil
			},
			DrainBatch: func(_ context.Context, _ int) (QueuedWalletBackfillBatchReport, error) {
				return QueuedWalletBackfillBatchReport{JobsProcessed: 2, TransactionsWritten: 7}, nil
			},
			Now: func() time.Time {
				return time.Date(2026, time.April, 13, 12, 0, 0, 0, time.UTC)
			},
		},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutputWithAutoIndex returned error: %v", err)
	}
	if !strings.Contains(output, "Auto index cycle complete") {
		t.Fatalf("unexpected auto index output %q", output)
	}
}
