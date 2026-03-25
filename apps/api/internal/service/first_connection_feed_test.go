package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

type fakeFirstConnectionFeedRepository struct {
	page   domain.FirstConnectionFeedPage
	err    error
	called bool
	sort   string
}

func (f *fakeFirstConnectionFeedRepository) FindFirstConnectionFeed(_ context.Context, _ string, _ int, sort string) (domain.FirstConnectionFeedPage, error) {
	f.called = true
	f.sort = sort
	return f.page, f.err
}

func TestFirstConnectionFeedServiceConvertsRepositoryPage(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
	cursor := "2026-03-20T09:10:11Z|wallet_1"
	svc := NewFirstConnectionFeedService(&fakeFirstConnectionFeedRepository{
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
					Score: domain.Score{
						Name:   domain.ScoreAlpha,
						Value:  72,
						Rating: domain.RatingHigh,
						Evidence: []domain.Evidence{
							{
								Kind:       domain.EvidenceTransfer,
								Label:      "first connection discovery signal",
								Source:     "first-connection-snapshot",
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

	response, err := svc.ListFirstConnectionFeed(context.Background(), "", 20)
	if err != nil {
		t.Fatalf("expected feed, got %v", err)
	}
	if !svc.repo.(*fakeFirstConnectionFeedRepository).called {
		t.Fatal("expected repository to be called")
	}
	if svc.repo.(*fakeFirstConnectionFeedRepository).sort != "" {
		t.Fatalf("expected default sort to be empty, got %q", svc.repo.(*fakeFirstConnectionFeedRepository).sort)
	}
	if len(response.Items) != 1 {
		t.Fatalf("unexpected response %#v", response)
	}
	if response.WindowLabel != "Hot feed baseline" {
		t.Fatalf("unexpected window label %q", response.WindowLabel)
	}
	if response.GeneratedAt != latest.Format(time.RFC3339) {
		t.Fatalf("unexpected generated_at %q", response.GeneratedAt)
	}
	if response.Items[0].Score != 72 {
		t.Fatalf("unexpected score %d", response.Items[0].Score)
	}
	if response.Items[0].Rating != "high" {
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

func TestFirstConnectionFeedServiceReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	svc := NewFirstConnectionFeedService(&fakeFirstConnectionFeedRepository{err: errors.New("boom")})
	_, err := svc.ListFirstConnectionFeed(context.Background(), "", 20)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFirstConnectionFeedServicePropagatesSort(t *testing.T) {
	t.Parallel()

	svc := NewFirstConnectionFeedService(&fakeFirstConnectionFeedRepository{
		page: domain.FirstConnectionFeedPage{},
	})

	_, err := svc.ListFirstConnectionFeedSorted(context.Background(), "", 20, "score")
	if err != nil {
		t.Fatalf("expected feed, got %v", err)
	}
	if svc.repo.(*fakeFirstConnectionFeedRepository).sort != "score" {
		t.Fatalf("unexpected propagated sort %q", svc.repo.(*fakeFirstConnectionFeedRepository).sort)
	}
}
