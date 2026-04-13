package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/packages/domain"
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

type fakeWalletSummaryEnricher struct {
	summary domain.WalletSummary
	err     error
	called  bool
}

func (f *fakeWalletSummaryEnricher) EnrichWalletSummary(_ context.Context, summary domain.WalletSummary) (domain.WalletSummary, error) {
	f.called = true
	if f.err != nil {
		return domain.WalletSummary{}, f.err
	}
	if f.summary.Address == "" {
		return summary, nil
	}
	return f.summary, nil
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
			TopCounterparties: []domain.WalletCounterparty{
				{
					Chain:            domain.ChainEVM,
					Address:          "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
					EntityKey:        "heuristic:evm:opensea",
					EntityType:       "marketplace",
					EntityLabel:      "OpenSea",
					InteractionCount: 9,
					InboundCount:     2,
					OutboundCount:    7,
					InboundAmount:    "24.100000",
					OutboundAmount:   "214.550000",
					PrimaryToken:     "WETH",
					TokenBreakdowns: []domain.WalletCounterpartyTokenSummary{
						{
							Symbol:         "WETH",
							InboundAmount:  "24.100000",
							OutboundAmount: "214.550000",
						},
					},
					DirectionLabel:   "outbound",
					FirstSeenAt:      "2026-03-10T00:00:00Z",
					LatestActivityAt: latest.Format(time.RFC3339),
				},
			},
			RecentFlow: domain.WalletRecentFlow{
				IncomingTxCount7d:  4,
				OutgoingTxCount7d:  9,
				IncomingTxCount30d: 13,
				OutgoingTxCount30d: 29,
				NetDirection7d:     "outbound",
				NetDirection30d:    "outbound",
			},
			Tags: []string{"wallet-summary", "evm"},
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
			LatestSignals: []domain.WalletLatestSignal{
				{
					Name:       domain.ScoreCluster,
					Value:      82,
					Rating:     domain.RatingHigh,
					Label:      "cluster overlap",
					Source:     "cluster-engine",
					ObservedAt: latest.Format(time.RFC3339),
				},
				{
					Name:       domain.ScoreShadowExit,
					Value:      34,
					Rating:     domain.RatingMedium,
					Label:      "bridge movement",
					Source:     "shadow-exit-engine",
					ObservedAt: latest.Format(time.RFC3339),
				},
				{
					Name:       domain.ScoreAlpha,
					Value:      21,
					Rating:     domain.RatingLow,
					Label:      "first connection",
					Source:     "alpha-engine",
					ObservedAt: latest.Format(time.RFC3339),
				},
			},
		},
	}

	enricher := &fakeWalletSummaryEnricher{
		summary: repo.summary,
	}
	enricher.summary.Enrichment = &domain.WalletEnrichment{
		Provider:               "moralis",
		NetWorthUSD:            "157.00",
		NativeBalance:          "0.00402",
		NativeBalanceFormatted: "0.00402 ETH",
		ActiveChains:           []string{"Ethereum", "Base"},
		ActiveChainCount:       2,
		Holdings: []domain.WalletHolding{
			{
				Symbol:              "USDC",
				TokenAddress:        "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Balance:             "149.20",
				BalanceFormatted:    "149.20",
				ValueUSD:            "149.20",
				PortfolioPercentage: 74.1,
				IsNative:            false,
			},
			{
				Symbol:              "WETH",
				TokenAddress:        "0xC02aaA39b223FE8D0A0E5C4F27eAD9083C756Cc2",
				Balance:             "0.00402",
				BalanceFormatted:    "0.00402",
				ValueUSD:            "8.14",
				PortfolioPercentage: 25.9,
				IsNative:            false,
			},
		},
		HoldingCount: 2,
		Source:       "live",
		UpdatedAt:    latest.Format(time.RFC3339),
	}
	enricher.summary.Indexing = domain.WalletIndexingState{
		Status:             "ready",
		LastIndexedAt:      latest.Format(time.RFC3339),
		CoverageStartAt:    "2026-03-10T00:00:00Z",
		CoverageEndAt:      latest.Format(time.RFC3339),
		CoverageWindowDays: 10,
	}

	svc := NewWalletSummaryService(repo, enricher)
	summary, err := svc.GetWalletSummary(context.Background(), "evm", "0x1234567890abcdef1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("expected summary, got %v", err)
	}

	if !repo.called {
		t.Fatal("expected repository to be called")
	}
	if !enricher.called {
		t.Fatal("expected enricher to be called")
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
	if len(summary.TopCounterparties) != 1 || summary.TopCounterparties[0].InteractionCount != 9 {
		t.Fatalf("unexpected top counterparties %#v", summary.TopCounterparties)
	}
	if summary.TopCounterparties[0].DirectionLabel != "outbound" {
		t.Fatalf("unexpected counterparty direction %#v", summary.TopCounterparties[0])
	}
	if summary.TopCounterparties[0].OutboundAmount != "214.550000" {
		t.Fatalf("unexpected counterparty amounts %#v", summary.TopCounterparties[0])
	}
	if summary.TopCounterparties[0].PrimaryToken != "WETH" {
		t.Fatalf("unexpected counterparty token %#v", summary.TopCounterparties[0])
	}
	if summary.TopCounterparties[0].EntityLabel != "OpenSea" {
		t.Fatalf("unexpected counterparty entity label %#v", summary.TopCounterparties[0])
	}
	if summary.RecentFlow.NetDirection7d != "outbound" {
		t.Fatalf("unexpected recent flow %#v", summary.RecentFlow)
	}
	if summary.Enrichment == nil {
		t.Fatal("expected enrichment to be present")
	}
	if summary.Enrichment.NetWorthUSD != "157.00" {
		t.Fatalf("unexpected enrichment %#v", summary.Enrichment)
	}
	if summary.Enrichment.HoldingCount != 2 || len(summary.Enrichment.Holdings) != 2 {
		t.Fatalf("expected holdings to be converted %#v", summary.Enrichment)
	}
	if summary.Enrichment.Holdings[0].Symbol != "USDC" {
		t.Fatalf("unexpected holdings %#v", summary.Enrichment.Holdings)
	}
	if len(summary.LatestSignals) != 3 {
		t.Fatalf("expected latest signals to be converted %#v", summary.LatestSignals)
	}
	if summary.LatestSignals[0].Name != "cluster_score" {
		t.Fatalf("unexpected latest signals %#v", summary.LatestSignals)
	}
	if summary.Indexing.Status != "ready" || summary.Indexing.CoverageWindowDays != 10 {
		t.Fatalf("unexpected indexing state %#v", summary.Indexing)
	}
}

func TestWalletSummaryServiceReturnsNotFound(t *testing.T) {
	t.Parallel()

	svc := NewWalletSummaryService(&fakeWalletSummaryRepository{err: repository.ErrWalletSummaryNotFound}, nil)
	_, err := svc.GetWalletSummary(context.Background(), "evm", "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if !errors.Is(err, ErrWalletSummaryNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestWalletSummaryServiceIgnoresEnrichmentFailure(t *testing.T) {
	t.Parallel()

	repo := &fakeWalletSummaryRepository{
		summary: domain.CreateWalletSummaryFixture(
			domain.ChainEVM,
			"0x1234567890abcdef1234567890abcdef12345678",
		),
	}
	svc := NewWalletSummaryService(repo, &fakeWalletSummaryEnricher{
		err: errors.New("moralis unavailable"),
	})

	summary, err := svc.GetWalletSummary(context.Background(), "evm", "0x1234567890abcdef1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("expected summary, got %v", err)
	}
	if summary.DisplayName == "" {
		t.Fatalf("expected base summary to survive enrichment failure: %#v", summary)
	}
}
