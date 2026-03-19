package providers

import "github.com/whalegraph/whalegraph/packages/domain"

type AlchemyAdapter struct{}

func (a AlchemyAdapter) Name() ProviderName { return ProviderAlchemy }
func (a AlchemyAdapter) Kind() AdapterKind  { return AdapterHybrid }

func (a AlchemyAdapter) FetchWalletActivity(ctx ProviderRequestContext) ([]ProviderWalletActivity, error) {
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
			SourceID:      "alchemy_transfers_v0",
			Kind:          "transfer",
			Confidence:    0.91,
		}),
	}, nil
}
