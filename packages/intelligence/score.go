package intelligence

import (
	"fmt"
	"strings"

	"github.com/qorvi/qorvi/packages/domain"
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

type scoreCalibration struct {
	SignalStrength                      int
	EvidenceSufficiency                 int
	SourceQuality                       int
	Freshness                           int
	ContradictionPenalty                int
	ContradictionReasons                []string
	SuppressionDiscount                 int
	SuppressionReasons                  []string
	CriticalEvidenceCount               int
	RequiredCriticalEvidenceForHigh     int
	MinimumEvidenceSufficiencyForMedium int
}

func applyScoreCalibration(score domain.Score, observedAt string, calibration scoreCalibration) domain.Score {
	value := clampScore(score.Value)

	blockedReason := ""
	if value >= 70 &&
		calibration.RequiredCriticalEvidenceForHigh > 0 &&
		calibration.CriticalEvidenceCount < calibration.RequiredCriticalEvidenceForHigh {
		value = minInt(value, 69)
		blockedReason = "insufficient_critical_evidence_for_high"
	}

	if value >= 35 &&
		blockedReason == "" &&
		calibration.MinimumEvidenceSufficiencyForMedium > 0 &&
		calibration.EvidenceSufficiency < calibration.MinimumEvidenceSufficiencyForMedium {
		value = minInt(value, 34)
		if blockedReason == "" {
			blockedReason = "insufficient_evidence_for_medium"
		}
	}

	score.Value = value
	score.Rating = rateScore(value)
	score.Evidence = append(score.Evidence, buildEvidence(
		domain.EvidenceLabel,
		"score calibration",
		"score-calibration",
		normalizeScoreObservedAt(observedAt, score.Evidence),
		calibrationEvidenceConfidence(calibration),
		buildScoreCalibrationMetadata(score, calibration, blockedReason),
	))

	return score
}

func normalizeScoreObservedAt(observedAt string, evidence []domain.Evidence) string {
	trimmed := strings.TrimSpace(observedAt)
	if trimmed != "" {
		return trimmed
	}
	for _, item := range evidence {
		if candidate := strings.TrimSpace(item.ObservedAt); candidate != "" {
			return candidate
		}
	}
	return ""
}

func calibrationEvidenceConfidence(calibration scoreCalibration) float64 {
	confidence := 0.55
	confidence += float64(clampScore(calibration.EvidenceSufficiency)-50) / 200
	confidence += float64(clampScore(calibration.SourceQuality)-50) / 250
	confidence -= float64(clampScore(calibration.ContradictionPenalty+calibration.SuppressionDiscount)) / 300
	switch {
	case confidence < 0.2:
		return 0.2
	case confidence > 0.95:
		return 0.95
	default:
		return confidence
	}
}

func buildScoreCalibrationMetadata(
	score domain.Score,
	calibration scoreCalibration,
	blockedReason string,
) map[string]any {
	metadata := map[string]any{
		"score_name":                              string(score.Name),
		"score_value":                             score.Value,
		"score_rating":                            string(score.Rating),
		"signal_strength":                         clampScore(calibration.SignalStrength),
		"evidence_sufficiency":                    clampScore(calibration.EvidenceSufficiency),
		"source_quality":                          clampScore(calibration.SourceQuality),
		"freshness":                               clampScore(calibration.Freshness),
		"contradiction_penalty":                   clampScore(calibration.ContradictionPenalty),
		"suppression_discount":                    clampScore(calibration.SuppressionDiscount),
		"critical_evidence_count":                 calibration.CriticalEvidenceCount,
		"required_critical_evidence_for_high":     calibration.RequiredCriticalEvidenceForHigh,
		"minimum_evidence_sufficiency_for_medium": calibration.MinimumEvidenceSufficiencyForMedium,
	}
	if blockedReason != "" {
		metadata["rating_blocked"] = true
		metadata["rating_block_reason"] = blockedReason
	}
	if reasons := compactReasonList(calibration.ContradictionReasons); len(reasons) > 0 {
		metadata["contradiction_reasons"] = reasons
	}
	if reasons := compactReasonList(calibration.SuppressionReasons); len(reasons) > 0 {
		metadata["suppression_reasons"] = reasons
	}
	return metadata
}

func boolCount(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

func compactReasonList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	reasons := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		reasons = append(reasons, trimmed)
	}
	if len(reasons) == 0 {
		return nil
	}
	return reasons
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
