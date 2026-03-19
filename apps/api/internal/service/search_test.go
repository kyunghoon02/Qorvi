package service

import (
	"context"
	"errors"
	"testing"
)

type fakeWalletSummaryLookup struct {
	summary WalletSummary
	err     error
	called  bool
	chain   string
	address string
}

func (f *fakeWalletSummaryLookup) GetWalletSummary(_ context.Context, chain, address string) (WalletSummary, error) {
	f.called = true
	f.chain = chain
	f.address = address
	return f.summary, f.err
}

func TestSearchServiceClassifiesEVMAddress(t *testing.T) {
	t.Parallel()

	svc := NewSearchService(&fakeWalletSummaryLookup{err: errors.New("wallet summary not found")})
	result := svc.Search(context.Background(), " 0x1234567890abcdef1234567890abcdef12345678 ")

	if result.InputKind != searchKindEVMAddress {
		t.Fatalf("expected evm input kind, got %s", result.InputKind)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(result.Results))
	}

	got := result.Results[0]
	if got.Type != searchTypeWallet {
		t.Fatalf("expected wallet type, got %s", got.Type)
	}
	if got.KindLabel != "EVM wallet address" {
		t.Fatalf("expected evm kind label, got %s", got.KindLabel)
	}
	if got.Chain != "evm" {
		t.Fatalf("expected evm chain, got %s", got.Chain)
	}
	if got.ChainLabel != "EVM" {
		t.Fatalf("expected evm chain label, got %s", got.ChainLabel)
	}
	if got.WalletRoute != "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/summary" {
		t.Fatalf("unexpected wallet route: %s", got.WalletRoute)
	}
	if !got.Navigation {
		t.Fatal("expected navigation to be enabled")
	}
}

func TestSearchServiceEnrichesWalletAddressFromSummaryLookup(t *testing.T) {
	t.Parallel()

	lookup := &fakeWalletSummaryLookup{
		summary: WalletSummary{
			Chain:       "solana",
			Address:     "So11111111111111111111111111111111111111112",
			DisplayName: "Seed Whale",
		},
	}

	svc := NewSearchService(lookup)
	result := svc.Search(context.Background(), "So11111111111111111111111111111111111111112")

	if !lookup.called {
		t.Fatal("expected wallet summary lookup to be called")
	}
	if lookup.chain != "solana" {
		t.Fatalf("expected solana lookup chain, got %s", lookup.chain)
	}

	got := result.Results[0]
	if got.Label != "Seed Whale" {
		t.Fatalf("expected display name label, got %s", got.Label)
	}
	if got.Chain != "solana" {
		t.Fatalf("expected enriched chain, got %s", got.Chain)
	}
	if got.ChainLabel != "Solana" {
		t.Fatalf("expected enriched chain label, got %s", got.ChainLabel)
	}
	if got.WalletRoute != "/v1/wallets/solana/So11111111111111111111111111111111111111112/summary" {
		t.Fatalf("unexpected wallet route: %s", got.WalletRoute)
	}
	if result.Explanation != "Found wallet summary for Seed Whale." {
		t.Fatalf("unexpected explanation: %s", result.Explanation)
	}
}

func TestSearchServiceFallsBackWhenWalletLookupMisses(t *testing.T) {
	t.Parallel()

	lookup := &fakeWalletSummaryLookup{err: errors.New("wallet summary not found")}
	svc := NewSearchService(lookup)
	result := svc.Search(context.Background(), "0x1234567890abcdef1234567890abcdef12345678")

	if !lookup.called {
		t.Fatal("expected wallet summary lookup to be called")
	}

	got := result.Results[0]
	if got.Label != "EVM wallet 0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("expected fallback label, got %s", got.Label)
	}
	if got.Chain != "evm" {
		t.Fatalf("expected fallback chain, got %s", got.Chain)
	}
	if got.Explanation != "Recognized as an EVM wallet address." {
		t.Fatalf("expected fallback explanation, got %s", got.Explanation)
	}
}

func TestSearchServiceClassifiesSolanaAddress(t *testing.T) {
	t.Parallel()

	svc := NewSearchService(nil)
	result := svc.Search(context.Background(), "So11111111111111111111111111111111111111112")

	if result.InputKind != searchKindSolanaAddress {
		t.Fatalf("expected solana input kind, got %s", result.InputKind)
	}

	got := result.Results[0]
	if got.Type != searchTypeWallet {
		t.Fatalf("expected wallet type, got %s", got.Type)
	}
	if got.KindLabel != "Solana wallet address" {
		t.Fatalf("expected solana kind label, got %s", got.KindLabel)
	}
	if got.Chain != "solana" {
		t.Fatalf("expected solana chain, got %s", got.Chain)
	}
	if got.ChainLabel != "Solana" {
		t.Fatalf("expected solana chain label, got %s", got.ChainLabel)
	}
	if got.WalletRoute != "/v1/wallets/solana/So11111111111111111111111111111111111111112/summary" {
		t.Fatalf("unexpected wallet route: %s", got.WalletRoute)
	}
}

func TestSearchServiceClassifiesENSLikeName(t *testing.T) {
	t.Parallel()

	svc := NewSearchService(nil)
	result := svc.Search(context.Background(), "vitalik.eth")

	if result.InputKind != searchKindENSName {
		t.Fatalf("expected ens input kind, got %s", result.InputKind)
	}

	got := result.Results[0]
	if got.Type != searchTypeIdentity {
		t.Fatalf("expected identity type, got %s", got.Type)
	}
	if got.KindLabel != "ENS-like name" {
		t.Fatalf("expected ens kind label, got %s", got.KindLabel)
	}
	if got.WalletRoute != "" {
		t.Fatalf("expected no wallet route for ENS-like input, got %s", got.WalletRoute)
	}
	if got.Navigation {
		t.Fatal("expected navigation to be disabled for ENS-like input")
	}
}

func TestSearchServiceExplainsUnknownQuery(t *testing.T) {
	t.Parallel()

	svc := NewSearchService(nil)
	result := svc.Search(context.Background(), "whalegraph")

	if result.InputKind != searchKindUnknown {
		t.Fatalf("expected unknown input kind, got %s", result.InputKind)
	}

	got := result.Results[0]
	if got.Type != searchTypeUnknown {
		t.Fatalf("expected unknown result type, got %s", got.Type)
	}
	if got.KindLabel != "Unknown input" {
		t.Fatalf("expected unknown kind label, got %s", got.KindLabel)
	}
	if got.WalletRoute != "" {
		t.Fatalf("expected no wallet route, got %s", got.WalletRoute)
	}
	if got.Explanation == "" || result.Explanation == "" {
		t.Fatal("expected explanation for unknown query")
	}
}
