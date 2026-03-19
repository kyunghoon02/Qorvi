package intelligence

import (
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestBuildFirstConnectionScore(t *testing.T) {
	t.Parallel()

	score := BuildFirstConnectionScore(FirstConnectionSignal{
		Chain:                   domain.ChainEVM,
		ObservedAt:              "2026-03-19T00:00:00Z",
		NewCommonEntries:        2,
		FirstSeenCounterparties: 3,
		HotFeedMentions:         1,
	})

	if score.Name != domain.ScoreAlpha {
		t.Fatalf("expected score name %q, got %q", domain.ScoreAlpha, score.Name)
	}

	if score.Value != 72 {
		t.Fatalf("expected score value 72, got %d", score.Value)
	}

	if score.Rating != domain.RatingHigh {
		t.Fatalf("expected high rating, got %q", score.Rating)
	}

	if err := validateScore(score); err != nil {
		t.Fatalf("expected valid score, got %v", err)
	}
}
