package intelligence

import (
	"testing"

	"github.com/flowintel/flowintel/packages/domain"
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
