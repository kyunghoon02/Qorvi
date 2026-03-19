package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/repository"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type fakeWalletSummaryRepository struct {
	summary domain.WalletSummary
	err     error
	called  bool
}

func (f *fakeWalletSummaryRepository) FindWalletSummary(context.Context, string, string) (domain.WalletSummary, error) {
	f.called = true
	return f.summary, f.err
}

func TestWalletSummaryServiceConvertsRepositoryRecord(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)
	clusterID := "cluster_seed_whales"
	repo := &fakeWalletSummaryRepository{
		summary: domain.WalletSummary{
			Chain:            domain.ChainEVM,
			Address:          "0x1234567890abcdef1234567890abcdef12345678",
			DisplayName:      "Seed Whale",
			ClusterID:        &clusterID,
			Counterparties:   18,
			LatestActivityAt: latest.Format(time.RFC3339),
			Tags:             []string{"wallet-summary", "evm"},
			Scores: []domain.Score{
				{
					Name:   domain.ScoreCluster,
					Value:  82,
					Rating: domain.RatingHigh,
					Evidence: []domain.Evidence{
						{
							Kind:       domain.EvidenceClusterOverlap,
							Label:      "cluster overlap",
							Source:     "cluster-engine",
							Confidence: 0.9,
							ObservedAt: latest.Format(time.RFC3339),
							Metadata:   map[string]any{"overlapping_wallets": 6},
						},
					},
				},
				{
					Name:   domain.ScoreShadowExit,
					Value:  34,
					Rating: domain.RatingMedium,
					Evidence: []domain.Evidence{
						{
							Kind:       domain.EvidenceBridge,
							Label:      "bridge movement",
							Source:     "shadow-exit-engine",
							Confidence: 0.58,
							ObservedAt: latest.Format(time.RFC3339),
							Metadata:   map[string]any{"bridge_transfers": 1},
						},
					},
				},
				{
					Name:   domain.ScoreAlpha,
					Value:  21,
					Rating: domain.RatingLow,
					Evidence: []domain.Evidence{
						{
							Kind:       domain.EvidenceLabel,
							Label:      "first connection",
							Source:     "alpha-engine",
							Confidence: 0.42,
							ObservedAt: latest.Format(time.RFC3339),
							Metadata:   map[string]any{"hot_feed_mentions": 1},
						},
					},
				},
			},
		},
	}

	svc := NewWalletSummaryService(repo)
	summary, err := svc.GetWalletSummary(context.Background(), "evm", "0x1234567890abcdef1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("expected summary, got %v", err)
	}

	if !repo.called {
		t.Fatal("expected repository to be called")
	}
	if summary.Chain != "evm" {
		t.Fatalf("unexpected chain %q", summary.Chain)
	}
	if len(summary.Scores) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(summary.Scores))
	}
	if summary.Scores[0].Name != "cluster_score" {
		t.Fatalf("unexpected score name %q", summary.Scores[0].Name)
	}
	if summary.ClusterID != clusterID {
		t.Fatalf("unexpected cluster id %q", summary.ClusterID)
	}
}

func TestWalletSummaryServiceReturnsNotFound(t *testing.T) {
	t.Parallel()

	svc := NewWalletSummaryService(&fakeWalletSummaryRepository{err: repository.ErrWalletSummaryNotFound})
	_, err := svc.GetWalletSummary(context.Background(), "evm", "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if !errors.Is(err, ErrWalletSummaryNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}
