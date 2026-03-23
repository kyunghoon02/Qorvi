package providers

import (
	"net/http"
	"time"
)

type HeliusAdapter struct {
	client *HeliusClient
}

func NewHeliusAdapter(credentials ProviderCredentials) HeliusAdapter {
	return HeliusAdapter{
		client: NewHeliusClient(credentials, nil),
	}
}

func (a HeliusAdapter) Name() ProviderName { return ProviderHelius }
func (a HeliusAdapter) Kind() AdapterKind  { return AdapterHybrid }

func (a HeliusAdapter) FetchWalletActivity(ctx ProviderRequestContext) ([]ProviderWalletActivity, error) {
	batch := CreateHistoricalBackfillBatchFixture(a.Name(), ctx.Chain, ctx.WalletAddress)
	batch.Request.Access = ctx.Access

	return a.FetchHistoricalWalletActivity(batch)
}

func (a HeliusAdapter) FetchHistoricalWalletActivity(batch HistoricalBackfillBatch) ([]ProviderWalletActivity, error) {
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
			SourceID:      "helius_webhook_v0",
			Kind:          "contract_interaction",
			Confidence:    0.87,
			ObservedAt:    batch.WindowEnd.Add(-2 * time.Minute),
		}),
	}, nil
}

func (a HeliusAdapter) WithHTTPClient(client *http.Client) HeliusAdapter {
	if a.client == nil {
		return a
	}

	copy := a
	copy.client = NewHeliusClient(ProviderCredentials{
		Provider:        ProviderHelius,
		APIKey:          a.client.apiKey,
		BaseURL:         a.client.baseURL,
		DataAPIBaseURL:  a.client.dataAPIBaseURL,
		FallbackAPIKey:  a.client.fallbackAPIKey,
		FallbackBaseURL: a.client.fallbackBaseURL,
	}, client)
	return copy
}
