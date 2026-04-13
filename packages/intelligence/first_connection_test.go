package intelligence

import (
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestBuildFirstConnectionScore(t *testing.T) {
	t.Parallel()

	score := BuildFirstConnectionScore(FirstConnectionSignal{
		WalletID:                "wallet_seed",
		Chain:                   domain.ChainEVM,
		Address:                 "0x1234567890abcdef1234567890abcdef12345678",
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

func TestBuildFirstConnectionSignalFromInputs(t *testing.T) {
	t.Parallel()

	signal := BuildFirstConnectionSignalFromInputs(FirstConnectionDetectorInputs{
		WalletID:                "wallet_seed",
		Chain:                   domain.ChainSolana,
		Address:                 "So11111111111111111111111111111111111111112",
		ObservedAt:              "2026-03-19T00:00:00Z",
		NewCommonEntries:        2,
		FirstSeenCounterparties: 3,
		HotFeedMentions:         1,
	})

	if signal.WalletID != "wallet_seed" {
		t.Fatalf("unexpected wallet id %q", signal.WalletID)
	}
	if signal.Address == "" {
		t.Fatal("expected address to be populated")
	}
	if signal.NewCommonEntries != 2 {
		t.Fatalf("unexpected new common entries %d", signal.NewCommonEntries)
	}
}

func TestValidateFirstConnectionSignal(t *testing.T) {
	t.Parallel()

	if err := ValidateFirstConnectionSignal(FirstConnectionSignal{
		WalletID:                "wallet_seed",
		Chain:                   domain.ChainEVM,
		Address:                 "0x1234567890abcdef1234567890abcdef12345678",
		ObservedAt:              "2026-03-19T00:00:00Z",
		NewCommonEntries:        1,
		FirstSeenCounterparties: 2,
		HotFeedMentions:         1,
	}); err != nil {
		t.Fatalf("expected valid signal, got %v", err)
	}

	if err := ValidateFirstConnectionSignal(FirstConnectionSignal{
		Chain:            domain.ChainEVM,
		ObservedAt:       "2026-03-19T00:00:00Z",
		NewCommonEntries: 1,
	}); err == nil {
		t.Fatal("expected missing wallet id to fail")
	}
}

func TestBuildFirstConnectionScoreCapsHighRatingWithoutEnoughCriticalEvidence(t *testing.T) {
	t.Parallel()

	score := BuildFirstConnectionScore(FirstConnectionSignal{
		WalletID:         "wallet_seed",
		Chain:            domain.ChainEVM,
		Address:          "0x1234567890abcdef1234567890abcdef12345678",
		ObservedAt:       "2026-03-19T00:00:00Z",
		NewCommonEntries: 4,
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

func TestBuildFirstConnectionScoreIncludesContradictionReasons(t *testing.T) {
	t.Parallel()

	score := BuildFirstConnectionScore(FirstConnectionSignal{
		WalletID:         "wallet_seed",
		Chain:            domain.ChainEVM,
		Address:          "0x1234567890abcdef1234567890abcdef12345678",
		ObservedAt:       "2026-03-19T00:00:00Z",
		NewCommonEntries: 2,
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

func TestBuildFirstConnectionScoreAppliesRouteDerivedContradictions(t *testing.T) {
	t.Parallel()

	score := BuildFirstConnectionScore(FirstConnectionSignal{
		WalletID:                        "wallet_seed",
		Chain:                           domain.ChainEVM,
		Address:                         "0x1234567890abcdef1234567890abcdef12345678",
		ObservedAt:                      "2026-03-19T00:00:00Z",
		NewCommonEntries:                3,
		FirstSeenCounterparties:         2,
		AggregatorCounterparties:        1,
		DeployerCollectorCounterparties: 1,
	})

	if score.Value != 56 {
		t.Fatalf("expected route-adjusted score value 56, got %d", score.Value)
	}
	if score.Rating != domain.RatingMedium {
		t.Fatalf("expected medium rating after route-derived contradictions, got %q", score.Rating)
	}
	metadata := score.Evidence[len(score.Evidence)-1].Metadata
	if got := metadata["contradiction_penalty"]; got != 18 {
		t.Fatalf("expected contradiction penalty 18, got %#v", metadata)
	}
	reasons, ok := metadata["contradiction_reasons"].([]string)
	if !ok {
		t.Fatalf("expected contradiction reasons slice, got %#v", metadata)
	}
	if len(reasons) != 3 {
		t.Fatalf("expected three contradiction reasons with route-derived adjustments, got %#v", reasons)
	}
}
