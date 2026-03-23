package providers

import (
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestCreateSeedDiscoveryBatchFixture(t *testing.T) {
	t.Parallel()

	batch := CreateSeedDiscoveryBatchFixture(
		ProviderDune,
		domain.ChainEVM,
		"0x1234567890abcdef1234567890abcdef12345678",
	)

	if batch.Provider != ProviderDune {
		t.Fatalf("expected provider %q, got %q", ProviderDune, batch.Provider)
	}
	if batch.Request.Chain != domain.ChainEVM {
		t.Fatalf("expected chain %q, got %q", domain.ChainEVM, batch.Request.Chain)
	}
	if batch.Limit != 100 {
		t.Fatalf("expected limit 100, got %d", batch.Limit)
	}
	if err := batch.Validate(); err != nil {
		t.Fatalf("expected fixture batch to validate, got %v", err)
	}
}

func TestDuneAdapterFetchSeedDiscoveryCandidatesReturnsFixtureCandidate(t *testing.T) {
	t.Parallel()

	batch := CreateSeedDiscoveryBatchFixture(
		ProviderDune,
		domain.ChainEVM,
		"0x1234567890abcdef1234567890abcdef12345678",
	)

	adapter := DuneAdapter{}
	candidates, err := adapter.FetchSeedDiscoveryCandidates(batch)
	if err != nil {
		t.Fatalf("FetchSeedDiscoveryCandidates returned error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	candidate := candidates[0]
	if candidate.Provider != ProviderDune {
		t.Fatalf("expected provider %q, got %q", ProviderDune, candidate.Provider)
	}
	if candidate.Chain != domain.ChainEVM {
		t.Fatalf("expected chain %q, got %q", domain.ChainEVM, candidate.Chain)
	}
	if candidate.WalletAddress != batch.Request.WalletAddress {
		t.Fatalf("expected wallet %q, got %q", batch.Request.WalletAddress, candidate.WalletAddress)
	}
	if candidate.SourceID != "dune_seed_export_v0" {
		t.Fatalf("unexpected source id %q", candidate.SourceID)
	}
	if candidate.Kind != "seed_label" {
		t.Fatalf("unexpected kind %q", candidate.Kind)
	}
	if candidate.Metadata["seed_discovery_provider"] != string(ProviderDune) {
		t.Fatalf("expected seed discovery provider metadata, got %#v", candidate.Metadata["seed_discovery_provider"])
	}
	if candidate.Metadata["seed_label"] != "dune_fixture_whale" {
		t.Fatalf("expected seed label metadata, got %#v", candidate.Metadata["seed_label"])
	}
}

func TestDuneAdapterFetchSeedDiscoveryCandidatesParsesExportRows(t *testing.T) {
	t.Parallel()

	batch := CreateSeedDiscoveryBatchFixture(
		ProviderDune,
		domain.ChainEVM,
		"0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
	)

	adapter := DuneAdapter{
		SeedDiscoveryRows: []DuneSeedExportRow{
			{
				Chain:             "ethereum",
				WalletAddress:     " 0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa ",
				Confidence:        1.2,
				ObservedAt:        "2026-03-20T01:02:03Z",
				SeedLabel:         "whale cohort",
				SeedLabelReason:   "ranked by inflow",
				SeedLabelSourceID: "query-42",
				Metadata: map[string]any{
					"query_id": 42,
				},
			},
			{
				Chain:         "unsupported",
				WalletAddress: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
		},
	}

	candidates, err := adapter.FetchSeedDiscoveryCandidates(batch)
	if err != nil {
		t.Fatalf("FetchSeedDiscoveryCandidates returned error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 parsed candidate, got %d", len(candidates))
	}

	candidate := candidates[0]
	if candidate.WalletAddress != "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("unexpected wallet address %q", candidate.WalletAddress)
	}
	if candidate.Chain != domain.ChainEVM {
		t.Fatalf("unexpected chain %q", candidate.Chain)
	}
	if candidate.Confidence != 1 {
		t.Fatalf("expected confidence to clamp to 1, got %v", candidate.Confidence)
	}
	if candidate.SourceID != "query-42" {
		t.Fatalf("unexpected source id %q", candidate.SourceID)
	}
	if candidate.Metadata["seed_label"] != "whale cohort" {
		t.Fatalf("unexpected seed label %#v", candidate.Metadata["seed_label"])
	}
	if candidate.Metadata["query_id"] != 42 {
		t.Fatalf("expected query id metadata, got %#v", candidate.Metadata["query_id"])
	}
}

func TestSeedDiscoveryRunnerRejectsProvidersWithoutSeedDiscoveryContract(t *testing.T) {
	t.Parallel()

	runner := NewSeedDiscoveryRunner(DefaultRegistry())
	batch := CreateSeedDiscoveryBatchFixture(
		ProviderAlchemy,
		domain.ChainEVM,
		"0x1234567890abcdef1234567890abcdef12345678",
	)

	if _, err := runner.Run(batch); err == nil {
		t.Fatal("expected runner to reject provider without seed discovery contract")
	}
}
