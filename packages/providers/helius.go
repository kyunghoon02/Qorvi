package providers

import "github.com/whalegraph/whalegraph/packages/domain"

type HeliusAdapter struct{}

func (a HeliusAdapter) Name() ProviderName { return ProviderHelius }
func (a HeliusAdapter) Kind() AdapterKind  { return AdapterRealtime }

func (a HeliusAdapter) FetchWalletActivity(ctx ProviderRequestContext) ([]ProviderWalletActivity, error) {
	return []ProviderWalletActivity{
		CreateProviderActivityFixture(struct {
			Provider      ProviderName
			Chain         domain.Chain
			WalletAddress string
			SourceID      string
			Kind          string
			Confidence    float64
		}{
			Provider:      a.Name(),
			Chain:         ctx.Chain,
			WalletAddress: ctx.WalletAddress,
			SourceID:      "helius_webhook_v0",
			Kind:          "contract_interaction",
			Confidence:    0.87,
		}),
	}, nil
}
