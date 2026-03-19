package intelligence

import (
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestBuildShadowExitRiskScore(t *testing.T) {
	t.Parallel()

	score := BuildShadowExitRiskScore(ShadowExitSignal{
		Chain:             domain.ChainSolana,
		ObservedAt:        "2026-03-19T00:00:00Z",
		BridgeTransfers:   2,
		CEXProximityCount: 1,
		FanOutCount:       1,
	})

	if score.Name != domain.ScoreShadowExit {
		t.Fatalf("expected score name %q, got %q", domain.ScoreShadowExit, score.Name)
	}

	if score.Rating != domain.RatingHigh {
		t.Fatalf("expected high rating, got %q", score.Rating)
	}

	if err := validateScore(score); err != nil {
		t.Fatalf("expected valid score, got %v", err)
	}
}
