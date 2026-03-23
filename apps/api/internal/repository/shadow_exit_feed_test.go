package repository

import (
	"context"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type fakeShadowExitFeedLoader struct {
	page   domain.ShadowExitFeedPage
	err    error
	called bool
}

func (f *fakeShadowExitFeedLoader) LoadShadowExitFeed(context.Context, db.ShadowExitFeedQuery) (domain.ShadowExitFeedPage, error) {
	f.called = true
	return f.page, f.err
}

func TestQueryBackedShadowExitFeedRepositoryFindShadowExitFeed(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 20, 3, 4, 5, 0, time.UTC)
	cursor := db.EncodeShadowExitFeedCursor(latest, "wallet_1")
	repo := NewQueryBackedShadowExitFeedRepository(&fakeShadowExitFeedLoader{
		page: domain.ShadowExitFeedPage{
			Items: []domain.ShadowExitFeedItem{
				{
					WalletID:       "wallet_1",
					Chain:          domain.ChainSolana,
					Address:        "So11111111111111111111111111111111111111112",
					Label:          "Seed Whale",
					WalletRoute:    "/wallets/solana/So11111111111111111111111111111111111111112",
					Recommendation: "Potential exit-like reshuffling; review recent counterparties and bridge activity.",
					ObservedAt:     latest.Format(time.RFC3339),
				},
			},
			NextCursor: &cursor,
			HasMore:    true,
		},
	})

	page, err := repo.FindShadowExitFeed(context.Background(), "", 20)
	if err != nil {
		t.Fatalf("expected page, got error: %v", err)
	}
	if !repo.loader.(*fakeShadowExitFeedLoader).called {
		t.Fatal("expected loader to be called")
	}
	if len(page.Items) != 1 || page.Items[0].WalletID != "wallet_1" {
		t.Fatalf("unexpected page %#v", page)
	}
}
