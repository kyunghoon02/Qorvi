package intelligence

import (
	"fmt"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func clampScore(value int) int {
	switch {
	case value < 0:
		return 0
	case value > 100:
		return 100
	default:
		return value
	}
}

func rateScore(value int) domain.ScoreRating {
	switch {
	case value >= 70:
		return domain.RatingHigh
	case value >= 35:
		return domain.RatingMedium
	default:
		return domain.RatingLow
	}
}

func buildEvidence(kind domain.EvidenceKind, label, source, observedAt string, confidence float64, metadata map[string]any) domain.Evidence {
	return domain.Evidence{
		Kind:       kind,
		Label:      label,
		Source:     source,
		Confidence: confidence,
		ObservedAt: observedAt,
		Metadata:   metadata,
	}
}

func validateScore(score domain.Score) error {
	if score.Value < 0 || score.Value > 100 {
		return fmt.Errorf("score value must be between 0 and 100")
	}

	if len(score.Evidence) == 0 {
		return fmt.Errorf("score %s must include evidence", score.Name)
	}

	expectedRating := rateScore(score.Value)
	if score.Rating != expectedRating {
		return fmt.Errorf("score %s rating mismatch: got %s want %s", score.Name, score.Rating, expectedRating)
	}

	return nil
}
