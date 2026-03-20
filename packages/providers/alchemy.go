package providers

import (
	"net/http"
	"time"
)

type AlchemyAdapter struct {
	client *AlchemyClient
}

func NewAlchemyAdapter(credentials ProviderCredentials) AlchemyAdapter {
	return AlchemyAdapter{
		client: NewAlchemyClient(credentials, nil),
	}
}

func (a AlchemyAdapter) Name() ProviderName { return ProviderAlchemy }
func (a AlchemyAdapter) Kind() AdapterKind  { return AdapterHybrid }

func (a AlchemyAdapter) FetchWalletActivity(ctx ProviderRequestContext) ([]ProviderWalletActivity, error) {
	batch := CreateHistoricalBackfillBatchFixture(a.Name(), ctx.Chain, ctx.WalletAddress)
	batch.Request.Access = ctx.Access

	return a.FetchHistoricalWalletActivity(batch)
}

func (a AlchemyAdapter) FetchHistoricalWalletActivity(batch HistoricalBackfillBatch) ([]ProviderWalletActivity, error) {
	if a.client != nil {
		return a.client.FetchHistoricalWalletActivity(batch)
	}
	if err := batch.Validate(); err != nil {
		return nil, err
	}

	return []ProviderWalletActivity{
		createHistoricalBackfillActivityFixture(batch, ProviderActivityFixtureInput{
			Provider:      a.Name(),
			Chain:         batch.Request.Chain,
			WalletAddress: batch.Request.WalletAddress,
			SourceID:      "alchemy_transfers_v0",
			Kind:          "transfer",
			Confidence:    0.91,
			ObservedAt:    batch.WindowEnd.Add(-time.Minute),
		}),
	}, nil
}

func (a AlchemyAdapter) WithHTTPClient(client *http.Client) AlchemyAdapter {
	if a.client == nil {
		return a
	}

	copy := a
	copy.client = NewAlchemyClient(ProviderCredentials{
		Provider: ProviderAlchemy,
		APIKey:   a.client.apiKey,
		BaseURL:  a.client.baseURL,
	}, client)
	return copy
}
