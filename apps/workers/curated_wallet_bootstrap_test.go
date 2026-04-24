package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/config"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeCuratedWalletSeedReader struct {
	items []db.CuratedWalletSeed
	err   error
}

func (r fakeCuratedWalletSeedReader) ListAdminCuratedWalletSeeds(context.Context) ([]db.CuratedWalletSeed, error) {
	return r.items, r.err
}

func TestCuratedWalletSeedBootstrapServiceRunEnqueue(t *testing.T) {
	t.Parallel()

	queue := &fakeWalletBackfillQueueStore{}
	tracking := &fakeWalletTrackingStateStore{}
	service := CuratedWalletSeedBootstrapService{
		Reader: fakeCuratedWalletSeedReader{
			items: []db.CuratedWalletSeed{
				{
					ListID:    "list_exchange",
					ListName:  "Featured wallets",
					ListTags:  []string{"featured"},
					ItemID:    "item_exchange",
					Chain:     domain.ChainEVM,
					Address:   "0x28C6c06298d514Db089934071355E5743bf21d60",
					ItemTags:  []string{"exchange"},
					ItemNotes: "Large exchange wallet",
					UpdatedAt: time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		Tracking: tracking,
		Queue:    queue,
		Dedup:    &fakeIngestDedupStore{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 31, 8, 9, 10, 0, time.UTC)
		},
	}

	report, err := service.RunEnqueue(context.Background())
	if err != nil {
		t.Fatalf("RunEnqueue returned error: %v", err)
	}
	if report.Source != "admin_curated" {
		t.Fatalf("unexpected source %q", report.Source)
	}
	if report.SeedsEnqueued != 1 {
		t.Fatalf("unexpected enqueue count %#v", report)
	}
	if len(tracking.candidates) != 1 {
		t.Fatalf("expected 1 tracking candidate, got %d", len(tracking.candidates))
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("expected 1 enqueued backfill job, got %d", len(queue.jobs))
	}
	if queue.jobs[0].Source != "curated_wallet_seed" {
		t.Fatalf("unexpected queue source %q", queue.jobs[0].Source)
	}
	if queue.jobs[0].Metadata["source_type"] != db.WalletTrackingSourceTypeSeedList {
		t.Fatalf("expected seed_list source_type, got %#v", queue.jobs[0].Metadata["source_type"])
	}
}

func TestBuildWorkerOutputRunsCuratedWalletSeedEnqueueFlow(t *testing.T) {
	t.Parallel()

	seedDiscovery := NewSeedDiscoveryJobRunner(nil)
	seedDiscovery.CuratedSeeds = fakeCuratedWalletSeedReader{
		items: []db.CuratedWalletSeed{
			{
				ListID:    "list_exchange",
				ListName:  "Featured wallets",
				ListTags:  []string{"featured"},
				ItemID:    "item_exchange",
				Chain:     domain.ChainEVM,
				Address:   "0x28C6c06298d514Db089934071355E5743bf21d60",
				ItemTags:  []string{"exchange"},
				ItemNotes: "Large exchange wallet",
				UpdatedAt: time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	seedDiscovery.Tracking = &fakeWalletTrackingStateStore{}
	seedDiscovery.Queue = &fakeWalletBackfillQueueStore{}
	seedDiscovery.Dedup = &fakeIngestDedupStore{}
	seedDiscovery.JobRuns = &fakeJobRunStore{}

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeCuratedWalletSeedEnqueue,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(nil),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		seedDiscovery,
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
		ExchangeListingRegistrySyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}
	if !strings.Contains(output, "Curated wallet seed enqueue complete") {
		t.Fatalf("unexpected output %q", output)
	}
	if !strings.Contains(output, "source=admin_curated") {
		t.Fatalf("expected admin_curated source in output, got %q", output)
	}
}
