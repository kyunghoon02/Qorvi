package service

import (
	"strings"
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestBuildDeterministicWalletBriefPrefersMaterializedFindingSummary(t *testing.T) {
	t.Parallel()

	summary := domain.WalletSummary{
		DisplayName: "Seed Whale",
		Scores: []domain.Score{
			{
				Name:   domain.ScoreCluster,
				Value:  88,
				Rating: domain.RatingHigh,
				Evidence: []domain.Evidence{
					{
						Metadata: map[string]any{
							"wallet_peer_overlap":      6,
							"shared_entity_neighbors":  3,
							"bidirectional_flow_peers": 2,
						},
					},
				},
			},
		},
	}
	findings := []domain.Finding{
		{Summary: "Materialized cluster finding summary."},
	}

	got := buildDeterministicWalletBrief(summary, findings, nil)
	if got != "Materialized cluster finding summary." {
		t.Fatalf("expected finding summary, got %q", got)
	}
}

func TestBuildDeterministicWalletBriefBuildsClusterFallbackFromPeerEntityFlowSignals(t *testing.T) {
	t.Parallel()

	summary := domain.WalletSummary{
		DisplayName: "Seed Whale",
		Scores: []domain.Score{
			{
				Name:   domain.ScoreCluster,
				Value:  88,
				Rating: domain.RatingHigh,
				Evidence: []domain.Evidence{
					{
						Metadata: map[string]any{
							"wallet_peer_overlap":      6,
							"shared_entity_neighbors":  4,
							"bidirectional_flow_peers": 2,
						},
					},
				},
			},
		},
	}

	got := buildDeterministicWalletBrief(summary, nil, nil)
	want := "Seed Whale is moving with a coordinated wallet cohort: 6 peer overlaps, 4 shared entity links, and 2 bidirectional peer flows are active inside the indexed coverage window."
	if got != want {
		t.Fatalf("expected cluster fallback %q, got %q", want, got)
	}
}

func TestBuildDeterministicWalletBriefAddsClusterCautionWhenBidirectionalFlowIsLimited(t *testing.T) {
	t.Parallel()

	summary := domain.WalletSummary{
		DisplayName: "Seed Whale",
		Scores: []domain.Score{
			{
				Name:   domain.ScoreCluster,
				Value:  73,
				Rating: domain.RatingHigh,
				Evidence: []domain.Evidence{
					{
						Metadata: map[string]any{
							"wallet_peer_overlap":     5,
							"shared_entity_neighbors": 2,
						},
					},
				},
			},
		},
	}

	got := buildDeterministicWalletBrief(summary, nil, nil)
	want := "Seed Whale is showing coordinated cohort overlap through 5 peer wallets and 2 shared entity links. Direct two-way flow is still limited, so conviction should be read with some caution."
	if got != want {
		t.Fatalf("expected cluster caution fallback %q, got %q", want, got)
	}
}

func TestBuildDeterministicWalletBriefPrefersEntryFeatureNarrativeOverClusterFallback(t *testing.T) {
	t.Parallel()

	summary := domain.WalletSummary{
		DisplayName: "Seed Whale",
		Scores: []domain.Score{
			{
				Name:   domain.ScoreCluster,
				Value:  73,
				Rating: domain.RatingHigh,
				Evidence: []domain.Evidence{
					{
						Metadata: map[string]any{
							"wallet_peer_overlap":     5,
							"shared_entity_neighbors": 2,
						},
					},
				},
			},
		},
	}

	got := buildDeterministicWalletBrief(summary, nil, &WalletEntryFeatures{
		QualityWalletOverlapCount:     2,
		FirstEntryBeforeCrowdingCount: 1,
		BestLeadHoursBeforePeers:      10,
		HoldingPersistenceState:       "short_lived",
		TopCounterparties: []WalletEntryFeatureCounterparty{
			{Chain: "evm", Address: "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed"},
		},
	})
	if !strings.Contains(got, "short-lived lead") {
		t.Fatalf("expected entry-feature summary, got %q", got)
	}
}
