package providers

type DuneAdapter struct{}

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
