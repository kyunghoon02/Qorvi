package intelligence

import (
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestBuildShadowExitRiskScore(t *testing.T) {
	t.Parallel()

	score := BuildShadowExitRiskScore(ShadowExitSignal{
		WalletID:          "wallet_seed",
		Chain:             domain.ChainSolana,
		Address:           "So11111111111111111111111111111111111111112",
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

func TestBuildShadowExitSignalFromInputs(t *testing.T) {
	t.Parallel()

	signal := BuildShadowExitSignalFromInputs(ShadowExitDetectorInputs{
		WalletID:                       "wallet_seed",
		Chain:                          domain.ChainSolana,
		Address:                        "So11111111111111111111111111111111111111112",
		ObservedAt:                     "2026-03-19T00:00:00Z",
		BridgeTransfers:                2,
		CEXProximityCount:              1,
		FanOutCount:                    1,
		FanOutCandidateCount24h:        3,
		OutboundTransferCount24h:       6,
		InboundTransferCount24h:        4,
		BridgeEscapeCount:              2,
		TreasuryWhitelistEvidenceCount: 1,
		InternalRebalanceEvidenceCount: 0,
	})

	if signal.FanOut24hCount != 3 {
		t.Fatalf("expected fan-out candidate count 3, got %d", signal.FanOut24hCount)
	}
	if signal.OutflowRatio != 0.6 {
		t.Fatalf("expected outflow ratio 0.6, got %.2f", signal.OutflowRatio)
	}
	if signal.BridgeEscapeCount != 2 {
		t.Fatalf("expected bridge escape count 2, got %d", signal.BridgeEscapeCount)
	}
	if !signal.TreasuryWhitelistDiscount {
		t.Fatal("expected treasury whitelist discount to be true")
	}
	if signal.InternalRebalanceDiscount {
		t.Fatal("expected internal rebalance discount to be false")
	}
}

func TestBuildShadowExitRiskScoreWithDetectorInputs(t *testing.T) {
	t.Parallel()

	score := BuildShadowExitRiskScore(ShadowExitSignal{
		WalletID:                  "wallet_seed",
		Chain:                     domain.ChainSolana,
		Address:                   "So11111111111111111111111111111111111111112",
		ObservedAt:                "2026-03-19T00:00:00Z",
		BridgeTransfers:           1,
		CEXProximityCount:         1,
		FanOutCount:               1,
		FanOut24hCount:            2,
		OutflowRatio:              0.4,
		BridgeEscapeCount:         1,
		TreasuryWhitelistDiscount: true,
		InternalRebalanceDiscount: true,
	})

	if score.Name != domain.ScoreShadowExit {
		t.Fatalf("expected score name %q, got %q", domain.ScoreShadowExit, score.Name)
	}
	if score.Value != 58 {
		t.Fatalf("expected score value 58, got %d", score.Value)
	}
	if score.Rating != domain.RatingMedium {
		t.Fatalf("expected medium rating, got %q", score.Rating)
	}
	if len(score.Evidence) != 3 {
		t.Fatalf("expected 3 evidence entries, got %d", len(score.Evidence))
	}
	if got := score.Evidence[0].Metadata["fan_out_candidate_count_24h"]; got != 2 {
		t.Fatalf("expected fan_out_candidate_count_24h metadata 2, got %v", got)
	}
	if got := score.Evidence[0].Metadata["outflow_ratio"]; got != 0.4 {
		t.Fatalf("expected outflow_ratio metadata 0.4, got %v", got)
	}
	if got := score.Evidence[0].Metadata["bridge_escape_count"]; got != 1 {
		t.Fatalf("expected bridge_escape_count metadata 1, got %v", got)
	}
	if got := score.Evidence[0].Metadata["discount_points"]; got != 32 {
		t.Fatalf("expected discount_points metadata 32, got %v", got)
	}
	if got := score.Evidence[len(score.Evidence)-1].Metadata["suppression_reasons"]; len(got.([]string)) != 2 {
		t.Fatalf("expected suppression reasons metadata, got %#v", score.Evidence[len(score.Evidence)-1].Metadata)
	}

	if err := validateScore(score); err != nil {
		t.Fatalf("expected valid score, got %v", err)
	}
}

func TestBuildShadowExitRiskScoreCapsHighRatingWithoutEnoughCriticalEvidence(t *testing.T) {
	t.Parallel()

	score := BuildShadowExitRiskScore(ShadowExitSignal{
		WalletID:        "wallet_seed",
		Chain:           domain.ChainSolana,
		Address:         "So11111111111111111111111111111111111111112",
		ObservedAt:      "2026-03-19T00:00:00Z",
		BridgeTransfers: 3,
	})

	if score.Value != 69 {
		t.Fatalf("expected capped score value 69, got %d", score.Value)
	}
	if score.Rating != domain.RatingMedium {
		t.Fatalf("expected medium rating after cap, got %q", score.Rating)
	}
	if score.Evidence[len(score.Evidence)-1].Metadata["rating_block_reason"] != "insufficient_critical_evidence_for_high" {
		t.Fatalf("expected rating block metadata, got %#v", score.Evidence[len(score.Evidence)-1].Metadata)
	}
}

func TestBuildShadowExitRiskScoreIncludesContradictionReasons(t *testing.T) {
	t.Parallel()

	score := BuildShadowExitRiskScore(ShadowExitSignal{
		WalletID:          "wallet_seed",
		Chain:             domain.ChainSolana,
		Address:           "So11111111111111111111111111111111111111112",
		ObservedAt:        "2026-03-19T00:00:00Z",
		BridgeTransfers:   2,
		CEXProximityCount: 1,
		FanOutCount:       1,
		OutflowRatio:      0.4,
		BridgeEscapeCount: 0,
	})

	got := score.Evidence[len(score.Evidence)-1].Metadata["contradiction_reasons"]
	reasons, ok := got.([]string)
	if !ok {
		t.Fatalf("expected contradiction reasons slice, got %#v", score.Evidence[len(score.Evidence)-1].Metadata)
	}
	if len(reasons) != 3 {
		t.Fatalf("expected three contradiction reasons, got %#v", reasons)
	}
}

func TestBuildShadowExitRiskScoreAppliesRouteDerivedAdjustments(t *testing.T) {
	t.Parallel()

	score := BuildShadowExitRiskScore(ShadowExitSignal{
		WalletID:                   "wallet_seed",
		Chain:                      domain.ChainSolana,
		Address:                    "So11111111111111111111111111111111111111112",
		ObservedAt:                 "2026-03-19T00:00:00Z",
		BridgeTransfers:            1,
		CEXProximityCount:          1,
		FanOutCount:                1,
		OutflowRatio:               0.4,
		AggregatorRoutingCount:     1,
		TreasuryRebalanceRoutes:    1,
		BridgeReturnCandidateCount: 1,
	})

	if score.Value != 36 {
		t.Fatalf("expected route-adjusted score value 36, got %d", score.Value)
	}
	if score.Rating != domain.RatingMedium {
		t.Fatalf("expected medium rating after route adjustments, got %q", score.Rating)
	}
	metadata := score.Evidence[len(score.Evidence)-1].Metadata
	if got := metadata["suppression_discount"]; got != 8 {
		t.Fatalf("expected suppression discount 8, got %#v", metadata)
	}
	if got := metadata["contradiction_penalty"]; got != 14 {
		t.Fatalf("expected contradiction penalty 14, got %#v", metadata)
	}
	if !containsString(metadata["suppression_reasons"], "treasury_rebalance_route") {
		t.Fatalf("expected treasury rebalance suppressor, got %#v", metadata)
	}
	if !containsString(metadata["contradiction_reasons"], "aggregator_routing_dominates_path") {
		t.Fatalf("expected aggregator contradiction, got %#v", metadata)
	}
	if !containsString(metadata["contradiction_reasons"], "bridge_return_candidate") {
		t.Fatalf("expected bridge return contradiction, got %#v", metadata)
	}
}

func TestValidateShadowExitSignal(t *testing.T) {
	t.Parallel()

	if err := ValidateShadowExitSignal(ShadowExitSignal{
		WalletID: "wallet_seed",
		Chain:    domain.ChainEVM,
		Address:  "0x1234567890abcdef1234567890abcdef12345678",
	}); err != nil {
		t.Fatalf("expected valid signal, got %v", err)
	}

	if err := ValidateShadowExitSignal(ShadowExitSignal{
		WalletID: "wallet_seed",
		Chain:    domain.Chain("unsupported"),
		Address:  "0x1234567890abcdef1234567890abcdef12345678",
	}); err == nil {
		t.Fatal("expected unsupported chain to fail validation")
	}
}

func containsString(value any, needle string) bool {
	items, ok := value.([]string)
	if !ok {
		return false
	}
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}
