package repository

import (
	"context"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

type fakeFirstConnectionFeedLoader struct {
	page   domain.FirstConnectionFeedPage
	err    error
	called bool
	query  db.FirstConnectionFeedQuery
}

func (f *fakeFirstConnectionFeedLoader) LoadFirstConnectionFeed(_ context.Context, query db.FirstConnectionFeedQuery) (domain.FirstConnectionFeedPage, error) {
	f.called = true
	f.query = query
	return f.page, f.err
}

func TestQueryBackedFirstConnectionFeedRepositoryFindFirstConnectionFeed(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
	cursor := db.EncodeFirstConnectionFeedCursor(latest, "wallet_1")
	repo := NewQueryBackedFirstConnectionFeedRepository(&fakeFirstConnectionFeedLoader{
		page: domain.FirstConnectionFeedPage{
			Items: []domain.FirstConnectionFeedItem{
				{
					WalletID:       "wallet_1",
					Chain:          domain.ChainEVM,
					Address:        "0x1234567890abcdef1234567890abcdef12345678",
					Label:          "Seed Whale",
					WalletRoute:    "/wallets/evm/0x1234567890abcdef1234567890abcdef12345678",
					Recommendation: "Elevated first-connection activity; review recent counterparties and activity.",
					ObservedAt:     latest.Format(time.RFC3339),
				},
			},
			NextCursor: &cursor,
			HasMore:    true,
		},
	})

	page, err := repo.FindFirstConnectionFeed(context.Background(), "", 20, "latest")
	if err != nil {
		t.Fatalf("expected page, got error: %v", err)
	}
	if !repo.loader.(*fakeFirstConnectionFeedLoader).called {
		t.Fatal("expected loader to be called")
	}
	if repo.loader.(*fakeFirstConnectionFeedLoader).query.Sort != db.FirstConnectionFeedSortLatest {
		t.Fatalf("unexpected sort %#v", repo.loader.(*fakeFirstConnectionFeedLoader).query.Sort)
	}
	if len(page.Items) != 1 || page.Items[0].WalletID != "wallet_1" {
		t.Fatalf("unexpected page %#v", page)
	}
}

func TestQueryBackedFirstConnectionFeedRepositoryFindFirstConnectionFeedSortScore(t *testing.T) {
	t.Parallel()

	repo := NewQueryBackedFirstConnectionFeedRepository(&fakeFirstConnectionFeedLoader{
		page: domain.FirstConnectionFeedPage{},
	})

	_, err := repo.FindFirstConnectionFeed(context.Background(), "", 20, "score")
	if err != nil {
		t.Fatalf("expected page, got error: %v", err)
	}
	if repo.loader.(*fakeFirstConnectionFeedLoader).query.Sort != db.FirstConnectionFeedSortScore {
		t.Fatalf("unexpected sort %#v", repo.loader.(*fakeFirstConnectionFeedLoader).query.Sort)
	}
}
