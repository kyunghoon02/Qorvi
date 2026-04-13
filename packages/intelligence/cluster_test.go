package intelligence

import (
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestBuildClusterScore(t *testing.T) {
	t.Parallel()

	score := BuildClusterScore(ClusterSignal{
		Chain:                          domain.ChainEVM,
		ObservedAt:                     "2026-03-19T00:00:00Z",
		OverlappingWallets:             4,
		SharedCounterparties:           3,
		MutualTransferCount:            2,
		SharedCounterpartiesStrength:   0,
		InteractionPersistenceStrength: 0,
	})

	if score.Name != domain.ScoreCluster {
		t.Fatalf("expected score name %q, got %q", domain.ScoreCluster, score.Name)
	}

	if score.Value != 56 {
		t.Fatalf("expected score value 56, got %d", score.Value)
	}

	if err := validateScore(score); err != nil {
		t.Fatalf("expected valid score, got %v", err)
	}
}

func TestBuildClusterScoreIncludesRelationStrengths(t *testing.T) {
	t.Parallel()

	score := BuildClusterScore(ClusterSignal{
		Chain:                          domain.ChainEVM,
		ObservedAt:                     "2026-03-19T00:00:00Z",
		OverlappingWallets:             2,
		SharedCounterparties:           2,
		MutualTransferCount:            2,
		SharedCounterpartiesStrength:   36,
		InteractionPersistenceStrength: 56,
	})

	if score.Value != 62 {
		t.Fatalf("expected score value 62, got %d", score.Value)
	}
	if score.Evidence[0].Metadata["shared_counterparties_strength"] != 36 {
		t.Fatalf("expected relation strength evidence metadata, got %#v", score.Evidence[0].Metadata)
	}
}

func TestBuildClusterScoreCapsHighRatingWithoutEnoughCriticalEvidence(t *testing.T) {
	t.Parallel()

	score := BuildClusterScore(ClusterSignal{
		Chain:              domain.ChainEVM,
		ObservedAt:         "2026-03-19T00:00:00Z",
		OverlappingWallets: 10,
	})

	if score.Value != 69 {
		t.Fatalf("expected capped score value 69, got %d", score.Value)
	}
	if score.Rating != domain.RatingMedium {
		t.Fatalf("expected medium rating after cap, got %q", score.Rating)
	}
	if score.Evidence[len(score.Evidence)-1].Metadata["rating_block_reason"] != "insufficient_critical_evidence_for_high" {
		t.Fatalf("expected high-rating block metadata, got %#v", score.Evidence[len(score.Evidence)-1].Metadata)
	}
}

func TestBuildClusterScoreIncludesContradictionReasons(t *testing.T) {
	t.Parallel()

	score := BuildClusterScore(ClusterSignal{
		Chain:                          domain.ChainEVM,
		ObservedAt:                     "2026-03-19T00:00:00Z",
		OverlappingWallets:             2,
		SharedCounterparties:           3,
		SharedCounterpartiesStrength:   24,
		InteractionPersistenceStrength: 8,
	})

	got := score.Evidence[len(score.Evidence)-1].Metadata["contradiction_reasons"]
	reasons, ok := got.([]string)
	if !ok {
		t.Fatalf("expected contradiction reasons slice, got %#v", score.Evidence[len(score.Evidence)-1].Metadata)
	}
	if len(reasons) != 2 {
		t.Fatalf("expected two contradiction reasons, got %#v", reasons)
	}
}

func TestBuildClusterScoreAppliesRouteAwareGating(t *testing.T) {
	t.Parallel()

	score := BuildClusterScore(ClusterSignal{
		Chain:                           domain.ChainEVM,
		ObservedAt:                      "2026-03-19T00:00:00Z",
		OverlappingWallets:              4,
		SharedCounterparties:            4,
		MutualTransferCount:             2,
		SharedCounterpartiesStrength:    36,
		InteractionPersistenceStrength:  40,
		AggregatorRoutingCounterparties: 2,
		ExchangeHubCounterparties:       1,
		BridgeInfraCounterparties:       1,
		TreasuryAdjacencyCounterparties: 1,
	})

	if score.Value != 50 {
		t.Fatalf("expected route-aware discounted score value 50, got %d", score.Value)
	}
	metadata := score.Evidence[len(score.Evidence)-1].Metadata
	if metadata["contradiction_penalty"] != 24 {
		t.Fatalf("expected contradiction penalty 24, got %#v", metadata)
	}
	if metadata["suppression_discount"] != 8 {
		t.Fatalf("expected suppression discount 8, got %#v", metadata)
	}
	if metadata["source_quality"] != 72 {
		t.Fatalf("expected source quality 72, got %#v", metadata)
	}
	reasons, ok := metadata["contradiction_reasons"].([]string)
	if !ok || len(reasons) < 3 {
		t.Fatalf("expected route contradiction reasons, got %#v", metadata)
	}
	suppressionReasons, ok := metadata["suppression_reasons"].([]string)
	if !ok || len(suppressionReasons) != 1 || suppressionReasons[0] != "treasury_adjacency_hub" {
		t.Fatalf("expected treasury suppression reason, got %#v", metadata)
	}
}
