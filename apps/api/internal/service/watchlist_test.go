package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/packages/domain"
)

func TestWatchlistServiceAllowsFreePlan(t *testing.T) {
	t.Parallel()

	svc := NewWatchlistService(repository.NewInMemoryWatchlistRepository())
	page, err := svc.ListWatchlists(context.Background(), "user_123", domain.PlanFree)
	if err != nil {
		t.Fatalf("expected free plan to list watchlists, got %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("expected empty watchlist collection, got %#v", page)
	}
}

func TestWatchlistServiceEnforcesOwnerLimitsAndNormalizesItemTags(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryWatchlistRepository()
	svc := NewWatchlistService(repo)
	svc.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC)
	}

	created := make([]WatchlistDetail, 0, 25)
	for index := 0; index < 25; index++ {
		detail, err := svc.CreateWatchlist(context.Background(), "user_123", domain.PlanFree, CreateWatchlistRequest{Name: fmt.Sprintf("List %d", index)})
		if err != nil {
			t.Fatalf("CreateWatchlist %d failed: %v", index, err)
		}
		created = append(created, detail)
	}

	if _, err := svc.CreateWatchlist(context.Background(), "user_123", domain.PlanFree, CreateWatchlistRequest{Name: "Overflow"}); !errors.Is(err, ErrWatchlistLimitExceeded) {
		t.Fatalf("expected list limit exceeded, got %v", err)
	}

	detail, err := svc.AddWatchlistItem(context.Background(), "user_123", domain.PlanFree, created[0].ID, CreateWatchlistItemRequest{
		Chain:   "evm",
		Address: "0x1234567890abcdef1234567890abcdef12345678",
		Tags:    []string{" Seed ", "seed", "Hot"},
		Note:    "  funding source  ",
	})
	if err != nil {
		t.Fatalf("AddWatchlistItem failed: %v", err)
	}
	if len(detail.Items) != 1 {
		t.Fatalf("expected one item, got %d", len(detail.Items))
	}
	if detail.Items[0].ItemType != "wallet" {
		t.Fatalf("unexpected item type %s", detail.Items[0].ItemType)
	}
	if len(detail.Items[0].Tags) != 2 || detail.Items[0].Tags[0] != "hot" || detail.Items[0].Tags[1] != "seed" {
		t.Fatalf("unexpected normalized tags %#v", detail.Items[0].Tags)
	}
	if detail.Items[0].Note != "funding source" {
		t.Fatalf("unexpected note %q", detail.Items[0].Note)
	}

	for index := 1; index < 1000; index++ {
		address := fmt.Sprintf("0x%040x", index+1)
		_, err := svc.AddWatchlistItem(context.Background(), "user_123", domain.PlanFree, created[0].ID, CreateWatchlistItemRequest{
			Chain:   "evm",
			Address: address,
		})
		if err != nil {
			t.Fatalf("AddWatchlistItem %d failed: %v", index, err)
		}
	}

	if _, err := svc.AddWatchlistItem(context.Background(), "user_123", domain.PlanFree, created[0].ID, CreateWatchlistItemRequest{
		Chain:   "evm",
		Address: "0x9999999999999999999999999999999999999999",
	}); !errors.Is(err, ErrWatchlistLimitExceeded) {
		t.Fatalf("expected item limit exceeded, got %v", err)
	}
}

func TestWatchlistServiceOwnerScope(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryWatchlistRepository()
	svc := NewWatchlistService(repo)
	svc.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC)
	}

	detail, err := svc.CreateWatchlist(context.Background(), "user_123", domain.PlanTeam, CreateWatchlistRequest{Name: "Primary"})
	if err != nil {
		t.Fatalf("CreateWatchlist failed: %v", err)
	}

	if _, err := svc.GetWatchlist(context.Background(), "user_999", domain.PlanTeam, detail.ID); !errors.Is(err, ErrWatchlistNotFound) {
		t.Fatalf("expected not found for foreign owner, got %v", err)
	}
}
