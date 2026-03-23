package providers

import (
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestDeriveHeuristicEntityAssignmentsUsesMeaningfulProviderMetadata(t *testing.T) {
	t.Parallel()

	assignments := DeriveHeuristicEntityAssignments([]ProviderWalletActivity{
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      ProviderHelius,
			Chain:         domain.ChainSolana,
			WalletAddress: "RootWallet111111111111111111111111111111111",
			SourceID:      "tx-1",
			Confidence:    0.91,
			Metadata: map[string]any{
				"counterparty_address":      "Counterparty1111111111111111111111111111111",
				"funder_address":            "FundingWallet11111111111111111111111111111111",
				"helius_identity_source":    "JUPITER",
				"helius_identity_fee_payer": "Counterparty1111111111111111111111111111111",
			},
		}),
	})

	if len(assignments) != 1 {
		t.Fatalf("expected 1 unique assignment, got %d", len(assignments))
	}
	if assignments[0].EntityKey != "heuristic:solana:jupiter" {
		t.Fatalf("unexpected entity key %#v", assignments[0])
	}
	if assignments[0].EntityLabel != "Jupiter" || assignments[0].EntityType != "protocol" {
		t.Fatalf("unexpected entity metadata %#v", assignments[0])
	}
	if assignments[0].Source != "provider-heuristic" {
		t.Fatalf("unexpected source %#v", assignments[0])
	}
}

func TestDeriveHeuristicEntityAssignmentsSkipsGenericSources(t *testing.T) {
	t.Parallel()

	assignments := DeriveHeuristicEntityAssignments([]ProviderWalletActivity{
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      ProviderHelius,
			Chain:         domain.ChainSolana,
			WalletAddress: "RootWallet111111111111111111111111111111111",
			SourceID:      "tx-1",
			Metadata: map[string]any{
				"counterparty_address":   "Counterparty1111111111111111111111111111111",
				"helius_identity_source": "TRANSFER",
			},
		}),
	})

	if len(assignments) != 0 {
		t.Fatalf("expected generic source to be ignored, got %#v", assignments)
	}
}

func TestDeriveHeuristicEntityAssignmentsUsesKnownAddressCatalogWhenProviderMetadataIsMissing(t *testing.T) {
	t.Parallel()

	assignments := DeriveHeuristicEntityAssignments([]ProviderWalletActivity{
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      ProviderAlchemy,
			Chain:         domain.ChainEVM,
			WalletAddress: "0x9ba2456137053d33ac556b569defb3f05b324811",
			SourceID:      "tx-2",
			Confidence:    0.82,
			Metadata: map[string]any{
				"counterparty_address": "0x0000000000000068f116a894984e2db1123eb395",
			},
		}),
	})

	if len(assignments) != 1 {
		t.Fatalf("expected 1 known-address assignment, got %#v", assignments)
	}
	if assignments[0].EntityKey != "heuristic:evm:seaport" {
		t.Fatalf("unexpected entity key %#v", assignments[0])
	}
	if assignments[0].EntityLabel != "Seaport 1.6" || assignments[0].EntityType != "marketplace" {
		t.Fatalf("unexpected entity metadata %#v", assignments[0])
	}
	if assignments[0].Address != "0x0000000000000068f116a894984e2db1123eb395" {
		t.Fatalf("unexpected assigned address %#v", assignments[0])
	}
}

func TestDeriveHeuristicEntityAssignmentsLabelsKnownWalletAddressItself(t *testing.T) {
	t.Parallel()

	assignments := DeriveHeuristicEntityAssignments([]ProviderWalletActivity{
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      ProviderAlchemy,
			Chain:         domain.ChainEVM,
			WalletAddress: "0xf5042e6ffac5a625d4e7848e0b01373d8eb9e222",
			SourceID:      "tx-3",
			Confidence:    0.77,
			Metadata:      map[string]any{},
		}),
	})

	if len(assignments) != 1 {
		t.Fatalf("expected self-address assignment, got %#v", assignments)
	}
	if assignments[0].EntityKey != "heuristic:evm:relay-link" {
		t.Fatalf("unexpected entity key %#v", assignments[0])
	}
	if assignments[0].EntityLabel != "Relay.link" || assignments[0].EntityType != "router" {
		t.Fatalf("unexpected entity metadata %#v", assignments[0])
	}
	if assignments[0].Address != "0xf5042e6ffac5a625d4e7848e0b01373d8eb9e222" {
		t.Fatalf("unexpected self assignment %#v", assignments[0])
	}
}

func TestDeriveHeuristicEntityAssignmentsNormalizesSourceAliases(t *testing.T) {
	t.Parallel()

	assignments := DeriveHeuristicEntityAssignments([]ProviderWalletActivity{
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      ProviderHelius,
			Chain:         domain.ChainSolana,
			WalletAddress: "RootWallet111111111111111111111111111111111",
			SourceID:      "tx-4",
			Confidence:    0.91,
			Metadata: map[string]any{
				"counterparty_address":   "Counterparty1111111111111111111111111111111",
				"helius_identity_source": "JUP-AG",
			},
		}),
	})

	if len(assignments) != 1 {
		t.Fatalf("expected aliased source assignment, got %#v", assignments)
	}
	if assignments[0].EntityKey != "heuristic:solana:jupiter" {
		t.Fatalf("unexpected entity key %#v", assignments[0])
	}
	if assignments[0].EntityLabel != "Jupiter" || assignments[0].EntityType != "protocol" {
		t.Fatalf("unexpected entity metadata %#v", assignments[0])
	}
}

func TestDeriveHeuristicEntityAssignmentsUsesMetadataLabelsWhenExplicitSourceIsMissing(t *testing.T) {
	t.Parallel()

	assignments := DeriveHeuristicEntityAssignments([]ProviderWalletActivity{
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      ProviderAlchemy,
			Chain:         domain.ChainEVM,
			WalletAddress: "0x9ba2456137053d33ac556b569defb3f05b324811",
			SourceID:      "tx-5",
			Confidence:    0.84,
			Metadata: map[string]any{
				"counterparty_address": "0x0000a26b00c1f0df003000390027140000faa719",
				"counterparty_label":   "OpenSea: Fees 3",
			},
		}),
	})

	if len(assignments) != 1 {
		t.Fatalf("expected one label-based assignment, got %#v", assignments)
	}
	if assignments[0].EntityKey != "heuristic:evm:opensea" {
		t.Fatalf("unexpected entity key %#v", assignments[0])
	}
	if assignments[0].EntityLabel != "OpenSea" || assignments[0].EntityType != "marketplace" {
		t.Fatalf("unexpected entity metadata %#v", assignments[0])
	}
}

func TestDeriveHeuristicEntityAssignmentsUsesFunderServiceLabelForExchangeClassification(t *testing.T) {
	t.Parallel()

	assignments := DeriveHeuristicEntityAssignments([]ProviderWalletActivity{
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      ProviderHelius,
			Chain:         domain.ChainSolana,
			WalletAddress: "RootWallet111111111111111111111111111111111",
			SourceID:      "tx-6",
			Confidence:    0.78,
			Metadata: map[string]any{
				"funder_address": "FundingWallet11111111111111111111111111111111",
				"funder_label":   "Binance Hot Wallet",
			},
		}),
	})

	if len(assignments) != 1 {
		t.Fatalf("expected one funder-label assignment, got %#v", assignments)
	}
	if assignments[0].EntityKey != "heuristic:solana:binance" {
		t.Fatalf("unexpected entity key %#v", assignments[0])
	}
	if assignments[0].EntityType != "exchange" || assignments[0].EntityLabel != "Binance" {
		t.Fatalf("unexpected entity metadata %#v", assignments[0])
	}
}
