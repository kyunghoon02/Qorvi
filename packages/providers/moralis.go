package providers

import "github.com/whalegraph/whalegraph/packages/domain"

type MoralisAdapter struct{}

func (a MoralisAdapter) Name() ProviderName { return ProviderMoralis }
func (a MoralisAdapter) Kind() AdapterKind  { return AdapterHistorical }

func (a MoralisAdapter) FetchWalletActivity(ctx ProviderRequestContext) ([]ProviderWalletActivity, error) {
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
			SourceID:      "moralis_wallet_history_v0",
			Kind:          "funding",
			Confidence:    0.79,
		}),
	}, nil
}
