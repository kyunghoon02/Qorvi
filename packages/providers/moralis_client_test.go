package providers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type fakeMoralisWalletEnrichmentCache struct {
	value   domain.WalletEnrichment
	has     bool
	getKeys []string
	setKeys []string
}

type fakeMoralisWalletEnrichmentSnapshotStore struct {
	calls []struct {
		chain      domain.Chain
		address    string
		enrichment domain.WalletEnrichment
	}
}

type fakeMoralisWalletSummaryCache struct {
	deleteKeys []string
}

func (f *fakeMoralisWalletSummaryCache) GetWalletSummaryInputs(context.Context, string) (db.WalletSummaryInputs, bool, error) {
	return db.WalletSummaryInputs{}, false, nil
}

func (f *fakeMoralisWalletSummaryCache) SetWalletSummaryInputs(context.Context, string, db.WalletSummaryInputs, time.Duration) error {
	return nil
}

func (f *fakeMoralisWalletSummaryCache) DeleteWalletSummaryInputs(_ context.Context, key string) error {
	f.deleteKeys = append(f.deleteKeys, key)
	return nil
}

func (f *fakeMoralisWalletEnrichmentSnapshotStore) UpsertWalletEnrichmentSnapshot(
	_ context.Context,
	chain domain.Chain,
	address string,
	enrichment domain.WalletEnrichment,
) error {
	f.calls = append(f.calls, struct {
		chain      domain.Chain
		address    string
		enrichment domain.WalletEnrichment
	}{
		chain:      chain,
		address:    address,
		enrichment: enrichment,
	})
	return nil
}

func (f *fakeMoralisWalletEnrichmentCache) GetWalletEnrichment(
	_ context.Context,
	key string,
) (domain.WalletEnrichment, bool, error) {
	f.getKeys = append(f.getKeys, key)
	return f.value, f.has, nil
}

func (f *fakeMoralisWalletEnrichmentCache) SetWalletEnrichment(
	_ context.Context,
	key string,
	enrichment domain.WalletEnrichment,
	_ time.Duration,
) error {
	f.setKeys = append(f.setKeys, key)
	f.value = enrichment
	f.has = true
	return nil
}

func TestMoralisClientFetchWalletEnrichment(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("X-API-Key"); got != "test-moralis-key" {
				t.Fatalf("unexpected api key header %q", got)
			}

			switch {
			case strings.HasSuffix(req.URL.Path, "/net-worth"):
				if req.URL.Query().Get("exclude_spam") != "true" {
					t.Fatalf("expected exclude_spam query")
				}
				return jsonHTTPResponse(200, `{
					"total_networth_usd":"157.00",
					"chains":[
						{
							"chain":"eth",
							"chain_name":"Ethereum",
							"native_balance":"0.00402",
							"native_balance_formatted":"0.00402 ETH"
						}
					]
				}`), nil
			case strings.HasSuffix(req.URL.Path, "/chains"):
				return jsonHTTPResponse(200, `{
					"result":[
						{"chain":"eth","chain_name":"Ethereum"},
						{"chain":"base","chain_name":"Base"},
						{"chain":"arb","chain_name":"Arbitrum"}
					]
				}`), nil
			case strings.HasSuffix(req.URL.Path, "/tokens"):
				if req.URL.Query().Get("limit") != "5" {
					t.Fatalf("expected limit query")
				}
				return jsonHTTPResponse(200, `{
					"result":[
						{
							"token_address":"0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							"symbol":"USDC",
							"balance":"149.20",
							"balance_formatted":"149.20",
							"usd_value":"149.20",
							"portfolio_percentage":94.8
						},
						{
							"token_address":"0xC02aaA39b223FE8D0A0E5C4F27eAD9083C756Cc2",
							"symbol":"WETH",
							"balance":"0.00402",
							"balance_formatted":"0.00402",
							"usd_value":"8.14",
							"portfolio_percentage":5.2
						}
					]
				}`), nil
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
		t.Fatalf("unexpected net worth %#v", enrichment)
	}
	if enrichment.NativeBalanceFormatted != "0.00402 ETH" {
		t.Fatalf("unexpected native balance %#v", enrichment)
	}
	if len(enrichment.ActiveChains) != 3 {
		t.Fatalf("unexpected active chains %#v", enrichment.ActiveChains)
	}
	if len(enrichment.Holdings) != 2 {
		t.Fatalf("unexpected holdings %#v", enrichment.Holdings)
	}
	if enrichment.Holdings[0].Symbol != "USDC" || enrichment.Holdings[0].ValueUSD != "149.20" {
		t.Fatalf("unexpected holdings %#v", enrichment.Holdings)
	}
	if enrichment.ActiveChains[0] != "Ethereum" {
		t.Fatalf("unexpected first active chain %#v", enrichment.ActiveChains)
	}
	if enrichment.Metadata()["moralis_net_worth_usd"] != "157.00" {
		t.Fatalf("unexpected metadata %#v", enrichment.Metadata())
	}
}

func TestMoralisClientFetchWalletEnrichmentKeepsNetWorthWhenChainsFail(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.HasSuffix(req.URL.Path, "/net-worth"):
				return jsonHTTPResponse(200, `{
					"total_networth_usd":"157.00",
					"native_balance":"0.00402",
					"native_balance_formatted":"0.00402 ETH"
				}`), nil
			case strings.HasSuffix(req.URL.Path, "/chains"):
				return jsonHTTPResponse(500, `{"message":"chains unavailable"}`), nil
			case strings.HasSuffix(req.URL.Path, "/tokens"):
				return jsonHTTPResponse(200, `{"result":[{"symbol":"USDC","balance_formatted":"149.20","usd_value":"149.20"}]}`), nil
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
		t.Fatalf("expected partial net worth enrichment, got %#v", enrichment)
	}
	if len(enrichment.ActiveChains) != 0 {
		t.Fatalf("expected chains to be omitted on partial failure, got %#v", enrichment.ActiveChains)
	}
	if len(enrichment.Holdings) != 1 {
		t.Fatalf("expected holdings to survive partial failure, got %#v", enrichment.Holdings)
	}
	if enrichment.Metadata()["moralis_partial_failure"] != true {
		t.Fatalf("expected partial failure metadata, got %#v", enrichment.Metadata())
	}
}

func TestMoralisClientFetchWalletEnrichmentKeepsChainsWhenNetWorthFails(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.HasSuffix(req.URL.Path, "/net-worth"):
				return jsonHTTPResponse(500, `{"message":"net worth unavailable"}`), nil
			case strings.HasSuffix(req.URL.Path, "/chains"):
				return jsonHTTPResponse(200, `{
					"result":[
						{"chain":"eth","chain_name":"Ethereum"},
						{"chain":"base","chain_name":"Base"}
					]
				}`), nil
			case strings.HasSuffix(req.URL.Path, "/tokens"):
				return jsonHTTPResponse(200, `{"result":[{"symbol":"USDC","balance_formatted":"149.20","usd_value":"149.20"}]}`), nil
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

	if enrichment.NetWorthUSD != "" {
		t.Fatalf("expected net worth to be omitted on partial failure, got %#v", enrichment)
	}
	if len(enrichment.ActiveChains) != 2 {
		t.Fatalf("expected chains to survive partial failure, got %#v", enrichment.ActiveChains)
	}
	if len(enrichment.Holdings) != 1 {
		t.Fatalf("expected holdings to survive partial failure, got %#v", enrichment.Holdings)
	}
	if enrichment.Metadata()["moralis_partial_failure"] != true {
		t.Fatalf("expected partial failure metadata, got %#v", enrichment.Metadata())
	}
}

func TestMoralisWalletSummaryEnricherUsesCacheHit(t *testing.T) {
	t.Parallel()

	cache := &fakeMoralisWalletEnrichmentCache{
		has: true,
		value: domain.WalletEnrichment{
			Provider:               "moralis",
			NetWorthUSD:            "88.50",
			NativeBalance:          "0.021",
			NativeBalanceFormatted: "0.021 ETH",
			ActiveChains:           []string{"Ethereum", "Base"},
			ActiveChainCount:       2,
			Source:                 "live",
			UpdatedAt:              "2026-03-22T00:00:00Z",
		},
	}

	enricher := NewMoralisWalletSummaryEnricher(nil, cache, nil, nil, 15*time.Minute)
	summary, err := enricher.EnrichWalletSummary(context.Background(), domain.CreateWalletSummaryFixture(
		domain.ChainEVM,
		"0x1234567890abcdef1234567890abcdef12345678",
	))
	if err != nil {
		t.Fatalf("EnrichWalletSummary returned error: %v", err)
	}

	if summary.Enrichment == nil {
		t.Fatal("expected cached enrichment")
	}
	if summary.Enrichment.Source != "cache" {
		t.Fatalf("expected cache source, got %#v", summary.Enrichment)
	}
	if len(cache.getKeys) != 1 {
		t.Fatalf("expected one cache lookup, got %#v", cache.getKeys)
	}
}

func TestMoralisWalletSummaryEnricherFetchesAndCaches(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.HasSuffix(req.URL.Path, "/net-worth"):
				return jsonHTTPResponse(200, `{
					"total_networth_usd":"201.30",
					"native_balance":"0.120",
					"native_balance_formatted":"0.120 ETH"
				}`), nil
			case strings.HasSuffix(req.URL.Path, "/chains"):
				return jsonHTTPResponse(200, `[{"chain":"eth"},{"chain":"base"}]`), nil
			case strings.HasSuffix(req.URL.Path, "/tokens"):
				return jsonHTTPResponse(200, `{
					"result":[
						{"symbol":"USDC","balance_formatted":"149.20","usd_value":"149.20","portfolio_percentage":74.1},
						{"symbol":"WETH","balance_formatted":"0.120","usd_value":"52.10","portfolio_percentage":25.9}
					]
				}`), nil
			default:
				t.Fatalf("unexpected request path %s", req.URL.Path)
				return nil, nil
			}
		}),
	}

	cache := &fakeMoralisWalletEnrichmentCache{}
	snapshots := &fakeMoralisWalletEnrichmentSnapshotStore{}
	summaryCache := &fakeMoralisWalletSummaryCache{}
	enricher := NewMoralisWalletSummaryEnricher(
		NewMoralisClient(ProviderCredentials{
			Provider: ProviderMoralis,
			APIKey:   "test-moralis-key",
			BaseURL:  "https://moralis.example/api/v2.2",
		}, httpClient),
		cache,
		snapshots,
		summaryCache,
		15*time.Minute,
	)

	summary, err := enricher.EnrichWalletSummary(context.Background(), domain.WalletSummary{
		Chain:       domain.ChainEVM,
		Address:     "0x1234567890abcdef1234567890abcdef12345678",
		DisplayName: "Live Whale",
	})
	if err != nil {
		t.Fatalf("EnrichWalletSummary returned error: %v", err)
	}

	if summary.Enrichment == nil {
		t.Fatal("expected enrichment")
	}
	if summary.Enrichment.NetWorthUSD != "201.30" {
		t.Fatalf("unexpected enrichment %#v", summary.Enrichment)
	}
	if summary.Enrichment.Source != "live" {
		t.Fatalf("expected live enrichment source %#v", summary.Enrichment)
	}
	if summary.Enrichment.HoldingCount != 2 || len(summary.Enrichment.Holdings) != 2 {
		t.Fatalf("expected holdings to be cached with enrichment %#v", summary.Enrichment)
	}
	if len(cache.setKeys) != 1 {
		t.Fatalf("expected cache set, got %#v", cache.setKeys)
	}
	if len(snapshots.calls) != 1 {
		t.Fatalf("expected enrichment snapshot write, got %#v", snapshots.calls)
	}
	if len(summaryCache.deleteKeys) != 1 || summaryCache.deleteKeys[0] != "wallet-summary:evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("expected wallet summary cache invalidation, got %#v", summaryCache.deleteKeys)
	}
}

func TestMoralisWalletSummaryEnricherKeepsSummarySnapshotWithoutLiveFetch(t *testing.T) {
	t.Parallel()

	cache := &fakeMoralisWalletEnrichmentCache{}
	enricher := NewMoralisWalletSummaryEnricher(nil, cache, nil, nil, 15*time.Minute)

	summary, err := enricher.EnrichWalletSummary(context.Background(), domain.WalletSummary{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
		Enrichment: &domain.WalletEnrichment{
			Provider:    "moralis",
			NetWorthUSD: "157.00",
			Source:      "snapshot",
			UpdatedAt:   "2026-03-22T00:00:00Z",
		},
	})
	if err != nil {
		t.Fatalf("EnrichWalletSummary returned error: %v", err)
	}
	if summary.Enrichment == nil || summary.Enrichment.Source != "snapshot" {
		t.Fatalf("expected existing snapshot enrichment to survive, got %#v", summary.Enrichment)
	}
}

func jsonHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}
