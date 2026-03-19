package providers

import (
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
)

var fixedObservedAt = time.Date(2026, time.March, 19, 0, 0, 0, 0, time.UTC)

func CreateProviderWalletSummary(chain domain.Chain, address string) domain.WalletSummary {
	summary := domain.CreateWalletSummaryFixture(chain, address)
	summary.LatestActivityAt = fixedObservedAt.Format(time.RFC3339)

	return summary
}

func CreateProviderActivityFixture(input struct {
	Provider      ProviderName
	Chain         domain.Chain
	WalletAddress string
	SourceID      string
	Kind          string
	Confidence    float64
}) ProviderWalletActivity {
	return ProviderWalletActivity{
		Provider:      input.Provider,
		Chain:         input.Chain,
		WalletAddress: input.WalletAddress,
		SourceID:      input.SourceID,
		ObservedAt:    fixedObservedAt,
		Kind:          input.Kind,
		Confidence:    input.Confidence,
		Metadata:      map[string]any{},
	}
}
