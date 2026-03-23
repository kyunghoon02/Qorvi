package domain

import (
	"testing"
	"time"
)

func TestCreateNormalizedTransactionFixture(t *testing.T) {
	t.Parallel()

	tx := CreateNormalizedTransactionFixture(
		ChainEVM,
		"0x1234567890abcdef1234567890abcdef12345678",
		"0xdeadbeef",
	)

	if err := ValidateNormalizedTransaction(tx); err != nil {
		t.Fatalf("expected valid normalized transaction, got %v", err)
	}

	if got := BuildWalletCanonicalKey(tx.Wallet.Chain, tx.Wallet.Address); got != "evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected wallet key %q", got)
	}
	if got := BuildTokenCanonicalKey(tx.Token.Chain, tx.Token.Address); got != "evm:0x0000000000000000000000000000000000000001" {
		t.Fatalf("unexpected token key %q", got)
	}
	if got := BuildTransactionCanonicalKey(tx); got != "evm:0xdeadbeef:evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected transaction key %q", got)
	}
}

func TestValidateNormalizedTransactionRejectsMissingFields(t *testing.T) {
	t.Parallel()

	tx := NormalizeNormalizedTransaction(NormalizedTransaction{
		Chain:      ChainSolana,
		TxHash:     "  ",
		Wallet:     WalletRef{Chain: ChainSolana, Address: "  "},
		ObservedAt: time.Time{},
	})

	if err := ValidateNormalizedTransaction(tx); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestNormalizeNormalizedTransactionSanitizesNilLikeAmount(t *testing.T) {
	t.Parallel()

	tx := NormalizeNormalizedTransaction(NormalizedTransaction{
		Chain:      ChainEVM,
		TxHash:     "0xdeadbeef",
		Wallet:     WalletRef{Chain: ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
		ObservedAt: time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC),
		Amount:     " <nil> ",
	})

	if tx.Amount != "" {
		t.Fatalf("expected nil-like amount to be cleared, got %q", tx.Amount)
	}
}
