package intelligence

import (
	"fmt"
	"math"
	"strings"

	"github.com/whalegraph/whalegraph/packages/domain"
)

type ShadowExitDetectorMetrics struct {
	FanOutCandidateCount24h int
	OutflowRatioPoints      int
	BridgeEscapePoints      int
	DiscountPoints          int
	GrossValue              int
}

func BuildShadowExitSignalFromInputs(inputs ShadowExitDetectorInputs) ShadowExitSignal {
	return ShadowExitSignal{
		WalletID:                  inputs.WalletID,
		Chain:                     inputs.Chain,
		Address:                   inputs.Address,
		ObservedAt:                inputs.ObservedAt,
		BridgeTransfers:           inputs.BridgeTransfers,
		CEXProximityCount:         inputs.CEXProximityCount,
		FanOutCount:               inputs.FanOutCount,
		FanOut24hCount:            clampScore(inputs.FanOutCandidateCount24h),
		OutflowRatio:              deriveShadowExitOutflowRatio(inputs.OutboundTransferCount24h, inputs.InboundTransferCount24h),
		BridgeEscapeCount:         inputs.BridgeEscapeCount,
		TreasuryWhitelistDiscount: inputs.TreasuryWhitelistEvidenceCount > 0,
		InternalRebalanceDiscount: inputs.InternalRebalanceEvidenceCount > 0,
	}
}

func BuildShadowExitDetectorMetrics(signal ShadowExitSignal) ShadowExitDetectorMetrics {
	fanOutCandidateCount24h := signal.FanOut24hCount
	outflowRatioPoints := clampScore(int(math.Round(signal.OutflowRatio * 30)))
	bridgeEscapePoints := signal.BridgeEscapeCount * 16
	discountPoints := 0
	if signal.TreasuryWhitelistDiscount {
		discountPoints += 18
	}
	if signal.InternalRebalanceDiscount {
		discountPoints += 14
	}

	grossValue := signal.BridgeTransfers*24 +
		signal.CEXProximityCount*12 +
		signal.FanOutCount*10 +
		fanOutCandidateCount24h*8 +
		bridgeEscapePoints +
		outflowRatioPoints

	return ShadowExitDetectorMetrics{
		FanOutCandidateCount24h: fanOutCandidateCount24h,
		OutflowRatioPoints:      outflowRatioPoints,
		BridgeEscapePoints:      bridgeEscapePoints,
		DiscountPoints:          discountPoints,
		GrossValue:              grossValue,
	}
}

func BuildShadowExitRiskScore(signal ShadowExitSignal) domain.Score {
	metrics := BuildShadowExitDetectorMetrics(signal)
	value := clampScore(metrics.GrossValue - metrics.DiscountPoints)

	evidence := []domain.Evidence{
		buildEvidence(
			domain.EvidenceBridge,
			"shadow exit risk signal",
			"shadow-exit-engine",
			signal.ObservedAt,
			0.77,
			map[string]any{
				"chain":                       signal.Chain,
				"bridge_transfers":            signal.BridgeTransfers,
				"cex_proximity_count":         signal.CEXProximityCount,
				"fan_out_count":               signal.FanOutCount,
				"fan_out_candidate_count_24h": metrics.FanOutCandidateCount24h,
				"outflow_ratio":               signal.OutflowRatio,
				"bridge_escape_count":         signal.BridgeEscapeCount,
				"outflow_ratio_points":        metrics.OutflowRatioPoints,
				"bridge_escape_points":        metrics.BridgeEscapePoints,
				"discount_points":             metrics.DiscountPoints,
				"treasury_whitelist_discount": signal.TreasuryWhitelistDiscount,
				"internal_rebalance_discount": signal.InternalRebalanceDiscount,
				"gross_value":                 metrics.GrossValue,
			},
		),
	}
	if metrics.DiscountPoints > 0 {
		evidence = append(evidence, buildEvidence(
			domain.EvidenceLabel,
			"treasury or whitelist discount applied",
			"shadow-exit-engine",
			signal.ObservedAt,
			0.64,
			map[string]any{
				"discount_points":             metrics.DiscountPoints,
				"treasury_whitelist_discount": signal.TreasuryWhitelistDiscount,
				"internal_rebalance_discount": signal.InternalRebalanceDiscount,
			},
		))
	}

	score := domain.Score{
		Name:     domain.ScoreShadowExit,
		Value:    value,
		Rating:   rateScore(value),
		Evidence: evidence,
	}

	return score
}

func deriveShadowExitOutflowRatio(outboundCount24h, inboundCount24h int) float64 {
	if outboundCount24h < 0 || inboundCount24h < 0 {
		return 0
	}

	totalCount := outboundCount24h + inboundCount24h
	if totalCount == 0 {
		return 0
	}

	return float64(outboundCount24h) / float64(totalCount)
}

func ValidateShadowExitSignal(signal ShadowExitSignal) error {
	if strings.TrimSpace(signal.WalletID) == "" {
		return fmt.Errorf("wallet_id is required")
	}
	if !domain.IsSupportedChain(signal.Chain) {
		return fmt.Errorf("unsupported chain %q", signal.Chain)
	}
	if strings.TrimSpace(signal.Address) == "" {
		return fmt.Errorf("address is required")
	}
	if signal.BridgeTransfers < 0 {
		return fmt.Errorf("bridge_transfers must be non-negative")
	}
	if signal.CEXProximityCount < 0 {
		return fmt.Errorf("cex_proximity_count must be non-negative")
	}
	if signal.FanOutCount < 0 {
		return fmt.Errorf("fan_out_count must be non-negative")
	}
	if signal.FanOut24hCount < 0 {
		return fmt.Errorf("fan_out_24h_count must be non-negative")
	}
	if signal.OutflowRatio < 0 {
		return fmt.Errorf("outflow_ratio must be non-negative")
	}
	if signal.BridgeEscapeCount < 0 {
		return fmt.Errorf("bridge_escape_count must be non-negative")
	}

	return nil
}
