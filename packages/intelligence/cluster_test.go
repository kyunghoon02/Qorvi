package intelligence

import (
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestBuildClusterScore(t *testing.T) {
	t.Parallel()

	score := BuildClusterScore(ClusterSignal{
		Chain:                domain.ChainEVM,
		ObservedAt:           "2026-03-19T00:00:00Z",
		OverlappingWallets:   4,
		SharedCounterparties: 3,
		MutualTransferCount:  2,
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
