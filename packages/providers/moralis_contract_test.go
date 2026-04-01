package providers

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestMoralisEnrichmentContractFixture(t *testing.T) {
	t.Parallel()

	netWorthFixture := readProviderFixture(t, "moralis", "net_worth.json")
	chainsFixture := readProviderFixture(t, "moralis", "chains.json")
	tokensFixture := readProviderFixture(t, "moralis", "tokens.json")

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.HasSuffix(req.URL.Path, "/net-worth"):
				return jsonHTTPResponse(200, netWorthFixture), nil
			case strings.HasSuffix(req.URL.Path, "/chains"):
				return jsonHTTPResponse(200, chainsFixture), nil
			case strings.HasSuffix(req.URL.Path, "/tokens"):
				return jsonHTTPResponse(200, tokensFixture), nil
			default:
				t.Fatalf("unexpected request path %s", req.URL.Path)
				return nil, nil
			}
		}),
	}

	client := NewMoralisClient(ProviderCredentials{
		Provider: ProviderMoralis,
		APIKey:   "test-moralis-key",
		BaseURL:  "https://moralis.example/api/v2.2",
	}, httpClient)

	enrichment, err := client.FetchWalletEnrichment(context.Background(), ProviderRequestContext{
		Chain:         domain.ChainEVM,
		WalletAddress: "0x1234567890abcdef1234567890abcdef12345678",
	})
	if err != nil {
		t.Fatalf("FetchWalletEnrichment returned error: %v", err)
	}

	if enrichment.NetWorthUSD != "157.00" {
		t.Fatalf("unexpected net worth %#v", enrichment.NetWorthUSD)
	}
	if len(enrichment.Holdings) != 2 {
		t.Fatalf("unexpected holdings %#v", enrichment.Holdings)
	}
	if len(enrichment.ActiveChains) != 3 {
		t.Fatalf("unexpected active chains %#v", enrichment.ActiveChains)
	}
}
