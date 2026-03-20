package providers

import (
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestCreateHistoricalBackfillBatchFixture(t *testing.T) {
	t.Parallel()

	batch := CreateHistoricalBackfillBatchFixture(
		ProviderAlchemy,
		domain.ChainEVM,
		"0x1234567890abcdef1234567890abcdef12345678",
	)

	if batch.Provider != ProviderAlchemy {
		t.Fatalf("expected provider %q, got %q", ProviderAlchemy, batch.Provider)
	}
	if batch.Request.Chain != domain.ChainEVM {
		t.Fatalf("expected chain %q, got %q", domain.ChainEVM, batch.Request.Chain)
	}
	if batch.Limit != 250 {
		t.Fatalf("expected limit 250, got %d", batch.Limit)
	}
	if err := batch.Validate(); err != nil {
		t.Fatalf("expected fixture batch to validate, got %v", err)
	}
}

func TestHistoricalBackfillRunnerRoutesAlchemyBatchThroughAdapter(t *testing.T) {
	t.Parallel()

	runner := NewHistoricalBackfillRunner(DefaultRegistry())
	batch := CreateHistoricalBackfillBatchFixture(
		ProviderAlchemy,
		domain.ChainEVM,
		"0x1234567890abcdef1234567890abcdef12345678",
	)

	result, err := runner.Run(batch)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Batch.Provider != ProviderAlchemy {
		t.Fatalf("expected provider %q, got %q", ProviderAlchemy, result.Batch.Provider)
	}
	if len(result.Activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(result.Activities))
	}

	activity := result.Activities[0]
	if activity.Provider != ProviderAlchemy {
		t.Fatalf("expected activity provider %q, got %q", ProviderAlchemy, activity.Provider)
	}
	if activity.Metadata["backfill_limit"] != 250 {
		t.Fatalf("expected backfill limit metadata 250, got %#v", activity.Metadata["backfill_limit"])
	}
	if activity.Metadata["backfill_window_start"] == "" || activity.Metadata["backfill_window_end"] == "" {
		t.Fatalf("expected backfill window metadata, got %#v", activity.Metadata)
	}
}

func TestHistoricalBackfillRunnerRoutesHeliusBatchThroughAdapter(t *testing.T) {
	t.Parallel()

	runner := NewHistoricalBackfillRunner(DefaultRegistry())
	batch := CreateHistoricalBackfillBatchFixture(
		ProviderHelius,
		domain.ChainSolana,
		"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
	)

	result, err := runner.Run(batch)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(result.Activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(result.Activities))
	}
	if result.Activities[0].Provider != ProviderHelius {
		t.Fatalf("expected activity provider %q, got %q", ProviderHelius, result.Activities[0].Provider)
	}
}

func TestHistoricalBackfillRunnerRejectsProvidersWithoutHistoricalContract(t *testing.T) {
	t.Parallel()

	runner := NewHistoricalBackfillRunner(DefaultRegistry())
	batch := CreateHistoricalBackfillBatchFixture(
		ProviderDune,
		domain.ChainEVM,
		"0x1234567890abcdef1234567890abcdef12345678",
	)

	if _, err := runner.Run(batch); err == nil {
		t.Fatal("expected runner to reject provider without historical backfill contract")
	}
}

func TestNormalizeProviderActivityBuildsNormalizedTransaction(t *testing.T) {
	t.Parallel()

	activity := CreateProviderActivityFixture(ProviderActivityFixtureInput{
		Provider:      ProviderAlchemy,
		Chain:         domain.ChainEVM,
		WalletAddress: "0x1234567890abcdef1234567890abcdef12345678",
		SourceID:      "alchemy_transfers_v0",
		Kind:          "transfer",
		Confidence:    0.91,
		Metadata: map[string]any{
			"direction":            "outbound",
			"counterparty_address": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
			"token_address":        "0x0000000000000000000000000000000000000001",
			"token_symbol":         "USDC",
			"token_decimals":       6,
			"amount":               "12.500000",
			"block_number":         12345678,
			"transaction_index":    3,
		},
	})

	tx, err := NormalizeProviderActivity(activity)
	if err != nil {
		t.Fatalf("NormalizeProviderActivity returned error: %v", err)
	}
	if tx.TxHash == "" {
		t.Fatal("expected tx hash to be populated")
	}
	if tx.RawPayloadPath == "" {
		t.Fatal("expected raw payload path to be populated")
	}
	if tx.Direction != domain.TransactionDirectionOutbound {
		t.Fatalf("unexpected direction %q", tx.Direction)
	}
	if tx.Counterparty == nil || tx.Counterparty.Address == "" {
		t.Fatalf("expected counterparty to be populated, got %#v", tx.Counterparty)
	}
	if tx.Token == nil || tx.Token.Address == "" {
		t.Fatalf("expected token to be populated, got %#v", tx.Token)
	}
}

func TestNormalizeProviderActivitiesUsesFixtureDefaults(t *testing.T) {
	t.Parallel()

	transactions, err := NormalizeProviderActivities([]ProviderWalletActivity{
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      ProviderHelius,
			Chain:         domain.ChainSolana,
			WalletAddress: "7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
			SourceID:      "helius_webhook_v0",
			Kind:          "contract_interaction",
			Confidence:    0.87,
		}),
	})
	if err != nil {
		t.Fatalf("NormalizeProviderActivities returned error: %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(transactions))
	}
	if transactions[0].Provider != "helius" {
		t.Fatalf("unexpected provider %q", transactions[0].Provider)
	}
	if transactions[0].RawPayloadPath == "" {
		t.Fatal("expected raw payload path to use fixture default")
	}
}
