package providers

import (
	"fmt"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

type DuneSeedExportRow struct {
	Chain             string
	WalletAddress     string
	Kind              string
	Confidence        float64
	ObservedAt        string
	SourceID          string
	SeedLabel         string
	SeedLabelReason   string
	SeedLabelSourceID string
	Metadata          map[string]any
}

type DuneAdapter struct {
	SeedDiscoveryRows []DuneSeedExportRow
}

func NewDuneAdapter(rows []DuneSeedExportRow) DuneAdapter {
	cloned := append([]DuneSeedExportRow(nil), rows...)
	return DuneAdapter{SeedDiscoveryRows: cloned}
}

func (a DuneAdapter) Name() ProviderName { return ProviderDune }
func (a DuneAdapter) Kind() AdapterKind  { return AdapterHistorical }

func (a DuneAdapter) FetchWalletActivity(ctx ProviderRequestContext) ([]ProviderWalletActivity, error) {
	return []ProviderWalletActivity{
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      a.Name(),
			Chain:         ctx.Chain,
			WalletAddress: ctx.WalletAddress,
			SourceID:      "dune_seed_export_v0",
			Kind:          "label",
			Confidence:    0.84,
		}),
	}, nil
}

func (a DuneAdapter) FetchSeedDiscoveryCandidates(batch SeedDiscoveryBatch) ([]SeedDiscoveryCandidate, error) {
	if err := batch.Validate(); err != nil {
		return nil, err
	}

	return a.buildSeedDiscoveryCandidates(batch), nil
}

func (a DuneAdapter) buildSeedDiscoveryCandidates(batch SeedDiscoveryBatch) []SeedDiscoveryCandidate {
	rows := a.SeedDiscoveryRows
	if len(rows) == 0 {
		rows = defaultDuneSeedExportRows(batch)
	}

	candidates := make([]SeedDiscoveryCandidate, 0, len(rows))
	for _, row := range rows {
		candidate, ok := buildDuneSeedDiscoveryCandidate(batch, row)
		if !ok {
			continue
		}
		candidates = append(candidates, candidate)
	}

	return candidates
}

func buildDuneSeedDiscoveryCandidate(batch SeedDiscoveryBatch, row DuneSeedExportRow) (SeedDiscoveryCandidate, bool) {
	chain, ok := normalizeDuneSeedExportChain(row.Chain, batch.Request.Chain)
	if !ok {
		return SeedDiscoveryCandidate{}, false
	}

	walletAddress := strings.TrimSpace(row.WalletAddress)
	if walletAddress == "" {
		return SeedDiscoveryCandidate{}, false
	}

	observedAt := batch.WindowEnd
	if parsedObservedAt, ok := parseDuneSeedObservedAt(row.ObservedAt); ok {
		observedAt = parsedObservedAt
	}

	kind := strings.TrimSpace(row.Kind)
	if kind == "" {
		kind = "seed_label"
	}

	sourceID := strings.TrimSpace(row.SourceID)
	if sourceID == "" {
		sourceID = strings.TrimSpace(row.SeedLabelSourceID)
	}
	if sourceID == "" {
		sourceID = "dune_seed_export_v0"
	}

	metadata := map[string]any{
		"seed_label":           strings.TrimSpace(row.SeedLabel),
		"seed_label_reason":    strings.TrimSpace(row.SeedLabelReason),
		"seed_label_source_id": strings.TrimSpace(row.SeedLabelSourceID),
		"dune_export_source":   sourceID,
	}
	for key, value := range row.Metadata {
		metadata[key] = value
	}

	return CreateSeedDiscoveryCandidateFixture(batch, SeedDiscoveryCandidateInput{
		Provider:      ProviderDune,
		Chain:         chain,
		WalletAddress: walletAddress,
		SourceID:      sourceID,
		Kind:          kind,
		Confidence:    clampDuneSeedConfidence(row.Confidence),
		ObservedAt:    observedAt,
		Metadata:      metadata,
	}), true
}

func defaultDuneSeedExportRows(batch SeedDiscoveryBatch) []DuneSeedExportRow {
	return []DuneSeedExportRow{
		{
			Chain:             string(batch.Request.Chain),
			WalletAddress:     batch.Request.WalletAddress,
			Kind:              "seed_label",
			Confidence:        0.84,
			ObservedAt:        fixedObservedAt.Format(time.RFC3339),
			SourceID:          "dune_seed_export_v0",
			SeedLabel:         "dune_fixture_whale",
			SeedLabelReason:   "curated seed export",
			SeedLabelSourceID: "dune_seed_export_v0",
			Metadata: map[string]any{
				"query_slug": "seed-wallet-export",
			},
		},
	}
}

func normalizeDuneSeedExportChain(raw string, fallback domain.Chain) (domain.Chain, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "", string(fallback):
		if domain.IsSupportedChain(fallback) {
			return fallback, true
		}
	case "evm", "ethereum", "eth":
		return domain.ChainEVM, true
	case "solana", "sol":
		return domain.ChainSolana, true
	}

	return "", false
}

func parseDuneSeedObservedAt(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}

	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC(), true
		}
	}

	return time.Time{}, false
}

func clampDuneSeedConfidence(value float64) float64 {
	switch {
	case value <= 0:
		return 0.5
	case value > 1:
		return 1
	default:
		return value
	}
}

func (r DuneSeedExportRow) String() string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(r.Chain), strings.TrimSpace(r.WalletAddress))
}
