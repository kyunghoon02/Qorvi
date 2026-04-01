package providers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestMobulaAdapterSeedDiscoveryBatches(t *testing.T) {
	t.Parallel()

	adapter := NewMobulaAdapter(ProviderCredentials{Provider: ProviderMobula}, []MobulaSmartMoneySeed{
		{
			Chain:        "evm",
			TokenAddress: "0x6982508145454Ce325dDbE47a25d4ec3d2311933",
			TokenSymbol:  "PEPE",
		},
	}, nil)

	batches := adapter.SeedDiscoveryBatches(time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC))
	if len(batches) != 2 {
		t.Fatalf("expected default labels to produce 2 batches, got %d", len(batches))
	}
	if batches[0].Provider != ProviderMobula {
		t.Fatalf("unexpected provider %q", batches[0].Provider)
	}
	if batches[0].Request.Chain != domain.ChainEVM {
		t.Fatalf("unexpected chain %q", batches[0].Request.Chain)
	}
	if batches[0].Request.WalletAddress != "0x6982508145454Ce325dDbE47a25d4ec3d2311933" {
		t.Fatalf("unexpected token address %q", batches[0].Request.WalletAddress)
	}
}

func TestMobulaAdapterFetchSeedDiscoveryCandidates(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "mobula_secret" {
			t.Fatalf("unexpected Authorization header %q", got)
		}
		if got := r.URL.Query().Get("label"); got != "smartTrader" {
			t.Fatalf("unexpected label query %q", got)
		}
		if got := r.URL.Query().Get("address"); got != "0x6982508145454Ce325dDbE47a25d4ec3d2311933" {
			t.Fatalf("unexpected address query %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"walletAddress": "0x1111111111111111111111111111111111111111",
					"tokenAddress": "0x6982508145454Ce325dDbE47a25d4ec3d2311933",
					"lastActivityAt": "2026-03-29T12:00:00Z",
					"totalPnlUSD": "125000.5",
					"volumeBuyUSD": "750000",
					"labels": ["smartTrader"]
				},
				{
					"walletAddress": "0x2222222222222222222222222222222222222222",
					"tokenAddress": "0x6982508145454Ce325dDbE47a25d4ec3d2311933",
					"lastActivityAt": "2026-01-01T12:00:00Z",
					"labels": ["smartTrader"]
				}
			],
			"totalCount": 2
		}`))
	}))
	defer server.Close()

	adapter := NewMobulaAdapter(ProviderCredentials{
		Provider: ProviderMobula,
		APIKey:   "mobula_secret",
		BaseURL:  server.URL,
	}, nil, server.Client())

	batch := SeedDiscoveryBatch{
		Provider: ProviderMobula,
		Request: ProviderRequestContext{
			Chain:         domain.ChainEVM,
			WalletAddress: "0x6982508145454Ce325dDbE47a25d4ec3d2311933",
		},
		WindowStart: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC),
		WindowEnd:   time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC),
		Limit:       25,
		Metadata: map[string]any{
			mobulaQueryLabelKey:       "smartTrader",
			mobulaQuerySeedKeyKey:     "ethereum:pepe",
			mobulaQueryTokenSymbolKey: "PEPE",
		},
	}

	candidates, err := adapter.FetchSeedDiscoveryCandidates(batch)
	if err != nil {
		t.Fatalf("FetchSeedDiscoveryCandidates returned error: %v", err)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected old candidates to be filtered, got %d", len(candidates))
	}
	if candidates[0].WalletAddress != "0x1111111111111111111111111111111111111111" {
		t.Fatalf("unexpected wallet address %q", candidates[0].WalletAddress)
	}
	if candidates[0].Confidence < 0.9 {
		t.Fatalf("expected smartTrader confidence boost, got %f", candidates[0].Confidence)
	}
	if candidates[0].Metadata["seed_label"] != "smartTrader" {
		t.Fatalf("expected seed label metadata, got %#v", candidates[0].Metadata["seed_label"])
	}
}
