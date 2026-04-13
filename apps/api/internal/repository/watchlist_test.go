package repository

import (
	"context"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestInMemoryWatchlistRepositoryOwnerScopedAndSorted(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryWatchlistRepository()
	now := time.Date(2026, time.March, 20, 10, 0, 0, 0, time.UTC)

	older := domain.Watchlist{
		ID:          "watchlist_old",
		OwnerUserID: "user_123",
		Name:        "Older",
		Notes:       "older notes",
		Tags:        []string{"seed", "watchlist"},
		ItemCount:   1,
		CreatedAt:   now.Add(-time.Hour),
		UpdatedAt:   now.Add(-time.Hour),
		Items: []domain.WatchlistItem{
			{
				ID:          "item_1",
				WatchlistID: "watchlist_old",
				ItemType:    domain.WatchlistItemTypeWallet,
				ItemKey:     "evm:0x1234567890abcdef1234567890abcdef12345678",
				Tags:        []string{"seed"},
				Notes:       "primary",
				CreatedAt:   now.Add(-time.Hour),
				UpdatedAt:   now.Add(-time.Hour),
			},
		},
	}
	newer := domain.Watchlist{
		ID:          "watchlist_new",
		OwnerUserID: "user_123",
		Name:        "Newer",
		Notes:       "newer notes",
		Tags:        []string{"hot"},
		ItemCount:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	otherOwner := domain.Watchlist{
		ID:          "watchlist_other",
		OwnerUserID: "user_999",
		Name:        "Other",
		Notes:       "other notes",
		Tags:        []string{"watchlist"},
		ItemCount:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if _, err := repo.CreateWatchlist(context.Background(), older); err != nil {
		t.Fatalf("CreateWatchlist older failed: %v", err)
	}
	if _, err := repo.CreateWatchlist(context.Background(), newer); err != nil {
		t.Fatalf("CreateWatchlist newer failed: %v", err)
	}
	if _, err := repo.CreateWatchlist(context.Background(), otherOwner); err != nil {
		t.Fatalf("CreateWatchlist other owner failed: %v", err)
	}

	items, err := repo.ListWatchlists(context.Background(), "user_123")
	if err != nil {
		t.Fatalf("ListWatchlists failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 watchlists, got %d", len(items))
	}
	if items[0].ID != "watchlist_new" {
		t.Fatalf("expected newest watchlist first, got %s", items[0].ID)
	}

	found, err := repo.FindWatchlist(context.Background(), "user_123", "watchlist_old")
	if err != nil {
		t.Fatalf("FindWatchlist failed: %v", err)
	}
	found.Name = "mutated"
	if found.Items[0].ItemKey != "evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected item key %s", found.Items[0].ItemKey)
	}

	other, err := repo.FindWatchlist(context.Background(), "user_999", "watchlist_other")
	if err != nil {
		t.Fatalf("FindWatchlist other failed: %v", err)
	}
	if other.Name != "Other" {
		t.Fatalf("unexpected other owner watchlist %s", other.Name)
	}

	if err := repo.DeleteWatchlist(context.Background(), "user_123", "watchlist_old"); err != nil {
		t.Fatalf("DeleteWatchlist failed: %v", err)
	}
	if _, err := repo.FindWatchlist(context.Background(), "user_123", "watchlist_old"); err == nil {
		t.Fatal("expected deleted watchlist to be missing")
	}
}
