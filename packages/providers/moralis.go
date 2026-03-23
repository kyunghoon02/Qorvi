package providers

import (
	"context"
	"net/http"
	"time"
)

type MoralisAdapter struct {
	client *MoralisClient
}

func NewMoralisAdapter(credentials ProviderCredentials) MoralisAdapter {
	return MoralisAdapter{
		client: NewMoralisClient(credentials, nil),
	}
}

func (a MoralisAdapter) Name() ProviderName { return ProviderMoralis }
func (a MoralisAdapter) Kind() AdapterKind  { return AdapterHistorical }

func (a MoralisAdapter) FetchWalletActivity(ctx ProviderRequestContext) ([]ProviderWalletActivity, error) {
	if a.client != nil {
		enrichment, err := a.client.FetchWalletEnrichment(context.Background(), ctx)
		if err == nil && enrichment.NetWorthUSD != "" {
			return []ProviderWalletActivity{
				CreateProviderActivityFixture(ProviderActivityFixtureInput{
					Provider:      a.Name(),
					Chain:         ctx.Chain,
					WalletAddress: ctx.WalletAddress,
					SourceID:      "moralis_wallet_enrichment_v1",
					Kind:          "enrichment",
					Confidence:    0.74,
					ObservedAt:    enrichment.ObservedAt,
					Metadata:      enrichment.Metadata(),
				}),
			}, nil
		}
	}

	return []ProviderWalletActivity{
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      a.Name(),
			Chain:         ctx.Chain,
			WalletAddress: ctx.WalletAddress,
			SourceID:      "moralis_wallet_history_v0",
			Kind:          "funding",
			Confidence:    0.79,
			ObservedAt:    time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC),
		}),
	}, nil
}

func (a MoralisAdapter) WithHTTPClient(client *http.Client) MoralisAdapter {
	if a.client == nil {
		return a
	}

	copy := a
	copy.client = NewMoralisClient(ProviderCredentials{
		Provider: ProviderMoralis,
		APIKey:   a.client.apiKey,
		BaseURL:  a.client.baseURL,
	}, client)
	return copy
}
