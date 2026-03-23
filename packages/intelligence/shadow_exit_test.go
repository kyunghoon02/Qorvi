package intelligence

import (
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
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
	if len(score.Evidence) != 2 {
		t.Fatalf("expected 2 evidence entries, got %d", len(score.Evidence))
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

	if err := validateScore(score); err != nil {
		t.Fatalf("expected valid score, got %v", err)
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
