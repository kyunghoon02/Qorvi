package main

import (
	"context"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
	"github.com/whalegraph/whalegraph/packages/providers"
)

type fakeSeedDiscoveryWatchlistStore struct {
	watchlists map[string][]domain.Watchlist
	items      map[string][]domain.WatchlistItem
}

func (s *fakeSeedDiscoveryWatchlistStore) ListWatchlists(_ context.Context, ownerUserID string) ([]domain.Watchlist, error) {
	return append([]domain.Watchlist(nil), s.watchlists[ownerUserID]...), nil
}

func (s *fakeSeedDiscoveryWatchlistStore) CreateWatchlist(
	_ context.Context,
	ownerUserID string,
	name string,
	notes string,
	tags []string,
) (domain.Watchlist, error) {
	if s.watchlists == nil {
		s.watchlists = map[string][]domain.Watchlist{}
	}
	watchlist := domain.Watchlist{
		ID:          "seed_watchlist_1",
		OwnerUserID: ownerUserID,
		Name:        name,
		Notes:       notes,
		Tags:        append([]string(nil), tags...),
		Items:       []domain.WatchlistItem{},
		CreatedAt:   time.Date(2026, time.March, 22, 9, 10, 11, 0, time.UTC),
		UpdatedAt:   time.Date(2026, time.March, 22, 9, 10, 11, 0, time.UTC),
	}
	s.watchlists[ownerUserID] = append(s.watchlists[ownerUserID], watchlist)
	return watchlist, nil
}

func (s *fakeSeedDiscoveryWatchlistStore) ListWatchlistItems(_ context.Context, _ string, watchlistID string) ([]domain.WatchlistItem, error) {
	return append([]domain.WatchlistItem(nil), s.items[watchlistID]...), nil
}

func (s *fakeSeedDiscoveryWatchlistStore) AddWatchlistWalletItem(
	_ context.Context,
	_ string,
	watchlistID string,
	ref db.WalletRef,
	tags []string,
	notes string,
) (domain.WatchlistItem, error) {
	if s.items == nil {
		s.items = map[string][]domain.WatchlistItem{}
	}
	itemKey, err := db.BuildWatchlistWalletItemKey(ref)
	if err != nil {
		return domain.WatchlistItem{}, err
	}
	item := domain.WatchlistItem{
		ID:          "item_" + itemKey,
		WatchlistID: watchlistID,
		ItemType:    domain.WatchlistItemTypeWallet,
		ItemKey:     itemKey,
		Tags:        append([]string(nil), tags...),
		Notes:       notes,
		CreatedAt:   time.Date(2026, time.March, 22, 9, 10, 11, 0, time.UTC),
		UpdatedAt:   time.Date(2026, time.March, 22, 9, 10, 11, 0, time.UTC),
	}
	s.items[watchlistID] = append(s.items[watchlistID], item)
	return item, nil
}

func TestSeedDiscoveryJobRunnerRunEnqueue(t *testing.T) {
	t.Parallel()

	queue := &fakeWalletBackfillQueueStore{}
	dedup := &fakeIngestDedupStore{}
	jobRuns := &fakeJobRunStore{}
	runner := NewSeedDiscoveryJobRunner(providers.DefaultRegistry())
	runner.Queue = queue
	runner.Dedup = dedup
	runner.JobRuns = jobRuns
	runner.Now = func() time.Time {
		return time.Date(2026, time.March, 22, 9, 10, 11, 0, time.UTC)
	}

	report, err := runner.RunEnqueue(context.Background())
	if err != nil {
		t.Fatalf("RunEnqueue returned error: %v", err)
	}
	if report.BatchesWritten != 1 || report.CandidatesSeen != 1 {
		t.Fatalf("unexpected report %#v", report)
	}
	if report.CandidatesEnqueued != 1 {
		t.Fatalf("expected 1 candidate enqueued, got %#v", report)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("expected 1 queued job, got %d", len(queue.jobs))
	}
	if queue.jobs[0].Source != "seed_discovery" {
		t.Fatalf("unexpected job source %q", queue.jobs[0].Source)
	}
	if queue.jobs[0].Metadata["seed_label"] != "dune_fixture_whale" {
		t.Fatalf("expected seed label metadata, got %#v", queue.jobs[0].Metadata["seed_label"])
	}
	if queue.jobs[0].Metadata["backfill_window_days"] != 90 {
		t.Fatalf("expected 90-day seed-discovery window, got %#v", queue.jobs[0].Metadata["backfill_window_days"])
	}
	if queue.jobs[0].Metadata["backfill_expansion_depth"] != 2 {
		t.Fatalf("expected 2-hop seed-discovery expansion depth, got %#v", queue.jobs[0].Metadata["backfill_expansion_depth"])
	}
	if len(jobRuns.entries) != 1 {
		t.Fatalf("expected 1 job run entry, got %d", len(jobRuns.entries))
	}
	if jobRuns.entries[0].Status != db.JobRunStatusSucceeded {
		t.Fatalf("unexpected job run status %q", jobRuns.entries[0].Status)
	}
}

func TestSeedDiscoveryJobRunnerRunEnqueueDedupsRepeatedCandidates(t *testing.T) {
	t.Parallel()

	queue := &fakeWalletBackfillQueueStore{}
	dedup := &fakeIngestDedupStore{}
	runner := NewSeedDiscoveryJobRunner(providers.DefaultRegistry())
	runner.Queue = queue
	runner.Dedup = dedup

	first, err := runner.RunEnqueue(context.Background())
	if err != nil {
		t.Fatalf("first RunEnqueue returned error: %v", err)
	}
	second, err := runner.RunEnqueue(context.Background())
	if err != nil {
		t.Fatalf("second RunEnqueue returned error: %v", err)
	}

	if first.CandidatesEnqueued != 1 {
		t.Fatalf("expected first run to enqueue 1 candidate, got %#v", first)
	}
	if second.CandidatesDeduped != 1 {
		t.Fatalf("expected second run to dedup candidate, got %#v", second)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("expected queue to contain 1 job, got %d", len(queue.jobs))
	}
}

func TestBuildSeedDiscoveryFixtureBatchesUsesDuneEVMFixture(t *testing.T) {
	t.Parallel()

	batches := buildSeedDiscoveryFixtureBatches()
	if len(batches) != 1 {
		t.Fatalf("expected 1 fixture batch, got %d", len(batches))
	}
	if batches[0].Provider != providers.ProviderDune {
		t.Fatalf("unexpected provider %q", batches[0].Provider)
	}
	if batches[0].Request.Chain != domain.ChainEVM {
		t.Fatalf("unexpected chain %q", batches[0].Request.Chain)
	}
}

func TestSeedDiscoveryJobRunnerRunSeedWatchlist(t *testing.T) {
	t.Parallel()

	watchlists := &fakeSeedDiscoveryWatchlistStore{}
	jobRuns := &fakeJobRunStore{}
	runner := NewSeedDiscoveryJobRunner(providers.NewConfiguredRegistry(providers.ProviderEnv{
		DuneAPIKey: "dune_secret",
		DuneSeedExportRows: []providers.DuneSeedExportRow{
			{
				Chain:         "evm",
				WalletAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				SeedLabel:     "high-signal",
				Confidence:    0.95,
			},
			{
				Chain:         "evm",
				WalletAddress: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				SeedLabel:     "mid-signal",
				Confidence:    0.75,
			},
			{
				Chain:         "evm",
				WalletAddress: "0xcccccccccccccccccccccccccccccccccccccccc",
				SeedLabel:     "low-signal",
				Confidence:    0.4,
			},
		},
		AlchemyAPIKey: "alchemy_secret",
		HeliusAPIKey:  "helius_secret",
		MoralisAPIKey: "moralis_secret",
	}))
	runner.Watchlists = watchlists
	runner.JobRuns = jobRuns
	runner.Now = func() time.Time {
		return time.Date(2026, time.March, 22, 9, 10, 11, 0, time.UTC)
	}

	report, err := runner.RunSeedWatchlist(context.Background(), 2, 0.7)
	if err != nil {
		t.Fatalf("RunSeedWatchlist returned error: %v", err)
	}
	if report.CandidatesSeen != 3 || report.CandidatesSelected != 2 {
		t.Fatalf("unexpected report %#v", report)
	}
	if !report.WatchlistCreated {
		t.Fatalf("expected watchlist to be created, got %#v", report)
	}
	if report.WatchlistItemsAdded != 2 {
		t.Fatalf("expected 2 watchlist items added, got %#v", report)
	}
	if len(watchlists.items[report.WatchlistID]) != 2 {
		t.Fatalf("expected 2 stored watchlist items, got %d", len(watchlists.items[report.WatchlistID]))
	}
	if len(jobRuns.entries) != 1 || jobRuns.entries[0].Status != db.JobRunStatusSucceeded {
		t.Fatalf("expected one succeeded job run, got %#v", jobRuns.entries)
	}
}

func TestSeedDiscoveryJobRunnerRunSeedWatchlistKeepsExistingItems(t *testing.T) {
	t.Parallel()

	watchlists := &fakeSeedDiscoveryWatchlistStore{
		watchlists: map[string][]domain.Watchlist{
			seedDiscoveryOwnerUserID: {
				{
					ID:          "seed_watchlist_1",
					OwnerUserID: seedDiscoveryOwnerUserID,
					Name:        seedDiscoveryWatchlistName,
				},
			},
		},
		items: map[string][]domain.WatchlistItem{
			"seed_watchlist_1": {
				{
					ID:          "item_existing",
					WatchlistID: "seed_watchlist_1",
					ItemType:    domain.WatchlistItemTypeWallet,
					ItemKey:     "evm:0x1234567890abcdef1234567890abcdef12345678",
				},
			},
		},
	}
	runner := NewSeedDiscoveryJobRunner(providers.DefaultRegistry())
	runner.Watchlists = watchlists

	report, err := runner.RunSeedWatchlist(context.Background(), 10, 0.7)
	if err != nil {
		t.Fatalf("RunSeedWatchlist returned error: %v", err)
	}
	if report.WatchlistCreated {
		t.Fatalf("expected existing watchlist to be reused, got %#v", report)
	}
	if report.WatchlistItemsKept != 1 {
		t.Fatalf("expected existing item to be kept, got %#v", report)
	}
	if report.WatchlistItemsAdded != 0 {
		t.Fatalf("expected no new item, got %#v", report)
	}
}
