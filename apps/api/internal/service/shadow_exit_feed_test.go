package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

type fakeShadowExitFeedRepository struct {
	page   domain.ShadowExitFeedPage
	err    error
	called bool
}

func (f *fakeShadowExitFeedRepository) FindShadowExitFeed(context.Context, string, int) (domain.ShadowExitFeedPage, error) {
	f.called = true
	return f.page, f.err
}

func TestShadowExitFeedServiceConvertsRepositoryPage(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 20, 3, 4, 5, 0, time.UTC)
	cursor := "2026-03-20T03:04:05Z|wallet_1"
	svc := NewShadowExitFeedService(&fakeShadowExitFeedRepository{
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
					Score: domain.Score{
						Name:   domain.ScoreShadowExit,
						Value:  34,
						Rating: domain.RatingMedium,
						Evidence: []domain.Evidence{
							{
								Kind:       domain.EvidenceBridge,
								Label:      "bridge movement",
								Source:     "shadow-exit-snapshot",
								Confidence: 1,
								ObservedAt: latest.Format(time.RFC3339),
							},
						},
					},
				},
			},
			NextCursor: &cursor,
			HasMore:    true,
		},
	})

	response, err := svc.ListShadowExitFeed(context.Background(), "", 20)
	if err != nil {
		t.Fatalf("expected feed, got %v", err)
	}
	if !svc.repo.(*fakeShadowExitFeedRepository).called {
		t.Fatal("expected repository to be called")
	}
	if len(response.Items) != 1 {
		t.Fatalf("unexpected response %#v", response)
	}
	if response.WindowLabel != "Last 24 hours" {
		t.Fatalf("unexpected window label %q", response.WindowLabel)
	}
	if response.GeneratedAt != latest.Format(time.RFC3339) {
		t.Fatalf("unexpected generated_at %q", response.GeneratedAt)
	}
	if response.Items[0].Score != 34 {
		t.Fatalf("unexpected score %d", response.Items[0].Score)
	}
	if response.Items[0].Rating != "medium" {
		t.Fatalf("unexpected rating %q", response.Items[0].Rating)
	}
	if response.Items[0].Explanation == "" {
		t.Fatal("expected explanation")
	}
	if len(response.Items[0].Evidence) != 1 {
		t.Fatalf("expected evidence, got %#v", response.Items[0].Evidence)
	}
	if response.NextCursor != cursor || !response.HasMore {
		t.Fatalf("unexpected pagination %#v", response)
	}
}

func TestShadowExitFeedServiceReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	svc := NewShadowExitFeedService(&fakeShadowExitFeedRepository{err: errors.New("boom")})
	_, err := svc.ListShadowExitFeed(context.Background(), "", 20)
	if err == nil {
		t.Fatal("expected error")
	}
}
