package main

import (
	"context"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

type fakeWatchlistSeedSource struct {
	refs []db.WalletRef
}

func (s fakeWatchlistSeedSource) ListWalletRefs(_ context.Context) ([]db.WalletRef, error) {
	return append([]db.WalletRef(nil), s.refs...), nil
}

func TestWatchlistBootstrapServiceRunEnqueue(t *testing.T) {
	t.Parallel()

	queue := &fakeWalletBackfillQueueStore{}
	service := WatchlistBootstrapService{
		Watchlists: fakeWatchlistSeedSource{
			refs: []db.WalletRef{
				{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
				{Chain: domain.ChainSolana, Address: "So11111111111111111111111111111111111111112"},
			},
		},
		Queue:   queue,
		Dedup:   &fakeIngestDedupStore{},
		JobRuns: &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 8, 9, 10, 0, time.UTC)
		},
	}

	report, err := service.RunEnqueue(context.Background())
	if err != nil {
		t.Fatalf("RunEnqueue returned error: %v", err)
	}
	if report.WalletsSeen != 2 || report.WalletsEnqueued != 2 {
		t.Fatalf("unexpected report %#v", report)
	}
	if len(queue.jobs) != 2 {
		t.Fatalf("expected 2 enqueued jobs, got %d", len(queue.jobs))
	}
	if queue.jobs[0].Source != "watchlist_bootstrap" {
		t.Fatalf("unexpected source %q", queue.jobs[0].Source)
	}
	if queue.jobs[0].Metadata["backfill_window_days"] != 365 {
		t.Fatalf("expected 365-day watchlist backfill policy, got %#v", queue.jobs[0].Metadata["backfill_window_days"])
	}
	if queue.jobs[0].Metadata["backfill_expansion_depth"] != 2 {
		t.Fatalf("expected 2-hop watchlist expansion depth, got %#v", queue.jobs[0].Metadata["backfill_expansion_depth"])
	}
}

func TestWatchlistBootstrapServiceRunEnqueueDedupsRepeatWallets(t *testing.T) {
	t.Parallel()

	queue := &fakeWalletBackfillQueueStore{}
	dedup := &fakeIngestDedupStore{}
	service := WatchlistBootstrapService{
		Watchlists: fakeWatchlistSeedSource{
			refs: []db.WalletRef{
				{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
			},
		},
		Queue: queue,
		Dedup: dedup,
	}

	first, err := service.RunEnqueue(context.Background())
	if err != nil {
		t.Fatalf("first RunEnqueue returned error: %v", err)
	}
	second, err := service.RunEnqueue(context.Background())
	if err != nil {
		t.Fatalf("second RunEnqueue returned error: %v", err)
	}

	if first.WalletsEnqueued != 1 {
		t.Fatalf("expected first enqueue to queue 1 wallet, got %#v", first)
	}
	if second.WalletsDeduped != 1 {
		t.Fatalf("expected second enqueue to dedup wallet, got %#v", second)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("expected queue to contain 1 job, got %d", len(queue.jobs))
	}
}
