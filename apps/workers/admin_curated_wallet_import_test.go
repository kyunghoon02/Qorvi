package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/config"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeAdminCuratedImportWatchlistStore struct {
	watchlists []domain.Watchlist
	created    []domain.Watchlist
	added      []domain.WatchlistItem
}

func (s *fakeAdminCuratedImportWatchlistStore) ListWatchlists(_ context.Context, ownerUserID string) ([]domain.Watchlist, error) {
	if ownerUserID != db.AdminCuratedOwnerUserID {
		return nil, nil
	}
	return append([]domain.Watchlist(nil), s.watchlists...), nil
}

func (s *fakeAdminCuratedImportWatchlistStore) CreateWatchlist(_ context.Context, ownerUserID string, name string, notes string, tags []string) (domain.Watchlist, error) {
	item := domain.Watchlist{
		ID:          "list_" + strings.ReplaceAll(strings.ToLower(name), " ", "_"),
		OwnerUserID: ownerUserID,
		Name:        name,
		Notes:       notes,
		Tags:        append([]string(nil), tags...),
		Items:       []domain.WatchlistItem{},
	}
	s.created = append(s.created, item)
	s.watchlists = append(s.watchlists, item)
	return item, nil
}

func (s *fakeAdminCuratedImportWatchlistStore) ListWatchlistItems(_ context.Context, ownerUserID string, watchlistID string) ([]domain.WatchlistItem, error) {
	for _, watchlist := range s.watchlists {
		if watchlist.OwnerUserID == ownerUserID && watchlist.ID == watchlistID {
			return append([]domain.WatchlistItem(nil), watchlist.Items...), nil
		}
	}
	return []domain.WatchlistItem{}, nil
}

func (s *fakeAdminCuratedImportWatchlistStore) AddWatchlistWalletItem(_ context.Context, ownerUserID string, watchlistID string, ref db.WalletRef, tags []string, notes string) (domain.WatchlistItem, error) {
	itemKey, err := db.BuildWatchlistWalletItemKey(ref)
	if err != nil {
		return domain.WatchlistItem{}, err
	}
	item := domain.WatchlistItem{
		ID:          "item_" + strings.ToLower(ref.Address),
		WatchlistID: watchlistID,
		ItemType:    domain.WatchlistItemTypeWallet,
		ItemKey:     itemKey,
		Tags:        append([]string(nil), tags...),
		Notes:       notes,
	}
	s.added = append(s.added, item)
	for index := range s.watchlists {
		if s.watchlists[index].ID != watchlistID || s.watchlists[index].OwnerUserID != ownerUserID {
			continue
		}
		s.watchlists[index].Items = append(s.watchlists[index].Items, item)
		break
	}
	return item, nil
}

type fakeAdminCuratedEntityIndex struct {
	syncedOwners []string
}

func (s *fakeAdminCuratedEntityIndex) SyncAdminCuratedEntityIndex(_ context.Context, ownerUserID string) error {
	s.syncedOwners = append(s.syncedOwners, ownerUserID)
	return nil
}

func TestAdminCuratedWalletImportServiceRunImport(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "curated-wallet-seeds.json")
	payload := `[
		{
			"chain": "evm",
			"address": "0x28C6c06298d514Db089934071355E5743bf21d60",
			"displayName": "Binance Hot Wallet",
			"description": "Curated exchange wallet",
			"category": "exchange",
			"tags": ["featured", "exchange"]
		},
		{
			"chain": "solana",
			"address": "5Q544fKrFoe6tsEbD7S8EmxGTJYAKtTVhAW5Q5pge4j1",
			"displayName": "Smart Money Seed",
			"description": "Curated smart money wallet",
			"category": "smart-money",
			"tags": ["featured", "smart-money"]
		}
	]`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	watchlists := &fakeAdminCuratedImportWatchlistStore{}
	entityIndex := &fakeAdminCuratedEntityIndex{}
	service := AdminCuratedWalletImportService{
		Watchlists: watchlists,
		EntityIndex: entityIndex,
		JobRuns: &fakeJobRunStore{},
		SeedPath: path,
		Now: func() time.Time {
			return time.Date(2026, time.March, 31, 10, 11, 12, 0, time.UTC)
		},
	}

	report, err := service.RunImport(context.Background())
	if err != nil {
		t.Fatalf("RunImport returned error: %v", err)
	}
	if report.ItemsAdded != 2 || report.ListsCreated != 2 {
		t.Fatalf("unexpected import report %#v", report)
	}
	if len(watchlists.added) != 2 {
		t.Fatalf("expected 2 added items, got %d", len(watchlists.added))
	}
	if len(entityIndex.syncedOwners) != 1 || entityIndex.syncedOwners[0] != db.AdminCuratedOwnerUserID {
		t.Fatalf("expected entity index sync for admin curated owner, got %#v", entityIndex.syncedOwners)
	}
}

func TestBuildWorkerOutputRunsAdminCuratedWalletImportFlow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "curated-wallet-seeds.json")
	payload := `[
		{
			"chain": "evm",
			"address": "0x28C6c06298d514Db089934071355E5743bf21d60",
			"displayName": "Binance Hot Wallet",
			"description": "Curated exchange wallet",
			"category": "exchange",
			"tags": ["featured", "exchange"]
		}
	]`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	t.Setenv("QORVI_CURATED_WALLET_SEEDS_PATH", path)

	seedDiscovery := NewSeedDiscoveryJobRunner(nil)
	seedDiscovery.Watchlists = &fakeAdminCuratedImportWatchlistStore{}
	seedDiscovery.EntityIndex = &fakeAdminCuratedEntityIndex{}
	seedDiscovery.JobRuns = &fakeJobRunStore{}
	seedDiscovery.Now = func() time.Time {
		return time.Date(2026, time.March, 31, 10, 11, 12, 0, time.UTC)
	}

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeAdminCuratedWalletImport,
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
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}
	if !strings.Contains(output, "Admin curated wallet import complete") {
		t.Fatalf("unexpected output %q", output)
	}
}
