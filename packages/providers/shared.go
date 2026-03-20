package providers

import (
	"fmt"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
)

var fixedObservedAt = time.Date(2026, time.March, 19, 0, 0, 0, 0, time.UTC)

func CreateProviderWalletSummary(chain domain.Chain, address string) domain.WalletSummary {
	summary := domain.CreateWalletSummaryFixture(chain, address)
	summary.LatestActivityAt = fixedObservedAt.Format(time.RFC3339)

	return summary
}

func CreateProviderActivityFixture(input ProviderActivityFixtureInput) ProviderWalletActivity {
	metadata := map[string]any{}
	for key, value := range input.Metadata {
		metadata[key] = value
	}
	if input.ObservedAt.IsZero() {
		input.ObservedAt = fixedObservedAt
	}
	applyProviderActivityFixtureDefaults(input, metadata)

	return ProviderWalletActivity{
		Provider:      input.Provider,
		Chain:         input.Chain,
		WalletAddress: input.WalletAddress,
		SourceID:      input.SourceID,
		ObservedAt:    input.ObservedAt,
		Kind:          input.Kind,
		Confidence:    input.Confidence,
		Metadata:      metadata,
	}
}

func applyProviderActivityFixtureDefaults(input ProviderActivityFixtureInput, metadata map[string]any) {
	if _, ok := metadata["tx_hash"]; !ok {
		metadata["tx_hash"] = fmt.Sprintf(
			"%s:%s:%s:%s",
			input.Provider,
			input.SourceID,
			strings.TrimSpace(input.WalletAddress),
			input.ObservedAt.UTC().Format("20060102T150405"),
		)
	}
	if _, ok := metadata["raw_payload_path"]; !ok {
		metadata["raw_payload_path"] = fmt.Sprintf(
			"s3://whalegraph/raw/%s/%s/%s/%s.json",
			input.Provider,
			strings.ToLower(strings.TrimSpace(string(input.Chain))),
			sanitizeWalletAddress(input.WalletAddress),
			sanitizeSourceID(input.SourceID),
		)
	}
	if _, ok := metadata["direction"]; !ok {
		metadata["direction"] = string(domain.TransactionDirectionUnknown)
	}
}

func sanitizeWalletAddress(address string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", " ", "")
	return replacer.Replace(strings.TrimSpace(address))
}

func sanitizeSourceID(sourceID string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", " ", "-")
	return replacer.Replace(strings.TrimSpace(sourceID))
}
