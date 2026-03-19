package domain

import "testing"

func TestCreateWalletSummaryFixture(t *testing.T) {
	t.Parallel()

	summary := CreateWalletSummaryFixture(ChainEVM, "0x1234567890abcdef1234567890abcdef12345678")

	if summary.Chain != ChainEVM {
		t.Fatalf("expected chain %q, got %q", ChainEVM, summary.Chain)
	}

	if len(summary.Scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(summary.Scores))
	}

	if err := ValidateWalletSummary(summary); err != nil {
		t.Fatalf("expected valid summary, got %v", err)
	}
}

func TestHasAnyRole(t *testing.T) {
	t.Parallel()

	allowed := HasAnyRole(
		AccessContext{
			Role: RoleAdmin,
			Plan: PlanPro,
		},
		RoleAdmin,
		RoleOperator,
	)

	if !allowed {
		t.Fatal("expected admin role to be allowed")
	}
}

func TestIsSupportedChain(t *testing.T) {
	t.Parallel()

	if !IsSupportedChain(ChainEVM) {
		t.Fatal("expected evm to be supported")
	}
	if !IsSupportedChain(ChainSolana) {
		t.Fatal("expected solana to be supported")
	}
	if IsSupportedChain(Chain("bitcoin")) {
		t.Fatal("expected bitcoin to be unsupported")
	}
}
