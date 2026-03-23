package intelligence

import (
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
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
