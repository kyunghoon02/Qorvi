package providers

import (
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

func TestDeriveWalletLabelingBuildsInferredAndBehavioralLabels(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 25, 1, 2, 3, 0, time.UTC)
	activity := CreateProviderActivityFixture(ProviderActivityFixtureInput{
		Provider:      ProviderAlchemy,
		Chain:         domain.ChainEVM,
		WalletAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SourceID:      "tx_exchange",
		Confidence:    0.91,
		ObservedAt:    observedAt,
		Metadata: map[string]any{
			"direction":            "outbound",
			"counterparty_address": "0x0000000000000068f116a894984e2db1123eb395",
			"counterparty_label":   "Binance Hot Wallet 14",
		},
	})

	derived := DeriveWalletLabeling([]ProviderWalletActivity{activity})
	if len(derived.Definitions) == 0 {
		t.Fatal("expected label definitions")
	}
	if len(derived.Evidences) == 0 {
		t.Fatal("expected wallet evidences")
	}
	if len(derived.Memberships) == 0 {
		t.Fatal("expected wallet label memberships")
	}

	foundCounterpartyExchange := false
	foundRootBehavior := false
	for _, membership := range derived.Memberships {
		if membership.Address == "0x0000000000000068f116a894984e2db1123eb395" &&
			membership.LabelKey == "inferred:exchange:binance" {
			foundCounterpartyExchange = true
		}
		if membership.Address == "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" &&
			membership.LabelKey == "behavioral:exchange_distribution_pattern" {
			foundRootBehavior = true
		}
	}

	if !foundCounterpartyExchange {
		t.Fatalf("expected inferred exchange label membership, got %#v", derived.Memberships)
	}
	if !foundRootBehavior {
		t.Fatalf("expected behavioral exchange distribution label membership, got %#v", derived.Memberships)
	}
}

func TestDeriveWalletLabelingBuildsTreasuryAndMarketMakerLabels(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 25, 4, 5, 6, 0, time.UTC)
	activities := []ProviderWalletActivity{
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      ProviderAlchemy,
			Chain:         domain.ChainEVM,
			WalletAddress: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			SourceID:      "tx_treasury",
			Confidence:    0.82,
			ObservedAt:    observedAt,
			Metadata: map[string]any{
				"direction":            "inbound",
				"counterparty_address": "0xcccccccccccccccccccccccccccccccccccccccc",
				"counterparty_label":   "Foundation Treasury Multisig",
			},
		}),
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      ProviderAlchemy,
			Chain:         domain.ChainEVM,
			WalletAddress: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			SourceID:      "tx_mm",
			Confidence:    0.84,
			ObservedAt:    observedAt.Add(time.Minute),
			Metadata: map[string]any{
				"direction":            "outbound",
				"counterparty_address": "0xdddddddddddddddddddddddddddddddddddddddd",
				"counterparty_label":   "Wintermute OTC 5",
			},
		}),
	}

	derived := DeriveWalletLabeling(activities)
	foundTreasury := false
	foundMarketMaker := false
	for _, membership := range derived.Memberships {
		if membership.Address == "0xcccccccccccccccccccccccccccccccccccccccc" &&
			membership.LabelKey == "inferred:treasury:treasury" {
			foundTreasury = true
		}
		if membership.Address == "0xdddddddddddddddddddddddddddddddddddddddd" &&
			membership.LabelKey == "inferred:market_maker:wintermute" {
			foundMarketMaker = true
		}
	}

	if !foundTreasury {
		t.Fatalf("expected treasury label membership, got %#v", derived.Memberships)
	}
	if !foundMarketMaker {
		t.Fatalf("expected market maker label membership, got %#v", derived.Memberships)
	}
}
