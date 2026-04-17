package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExchangeListingClientFetchUpbitListings(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/market/all" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("is_details"); got != "true" {
			t.Fatalf("unexpected upbit is_details query %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"market":"KRW-BTC","korean_name":"비트코인","english_name":"Bitcoin","market_warning":"NONE"},
			{"market":"BTC-XRP","korean_name":"리플","english_name":"XRP","market_warning":"CAUTION"}
		]`))
	}))
	defer server.Close()

	client := NewUpbitExchangeListingClient(server.URL, server.Client())
	listings, err := client.FetchUpbitListings(context.Background())
	if err != nil {
		t.Fatalf("FetchUpbitListings returned error: %v", err)
	}

	if len(listings) != 2 {
		t.Fatalf("expected 2 listings, got %d", len(listings))
	}
	if listings[0].Exchange != ExchangeUpbit {
		t.Fatalf("unexpected exchange %q", listings[0].Exchange)
	}
	if listings[0].BaseSymbol != "BTC" || listings[0].QuoteSymbol != "KRW" {
		t.Fatalf("unexpected market split %#v", listings[0])
	}
	if listings[0].NormalizedAssetKey != "btc" {
		t.Fatalf("unexpected normalized asset key %q", listings[0].NormalizedAssetKey)
	}
	if listings[1].MarketWarning != "CAUTION" {
		t.Fatalf("unexpected market warning %q", listings[1].MarketWarning)
	}
}

func TestExchangeListingClientFetchBithumbListings(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/market/all" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("isDetails"); got != "true" {
			t.Fatalf("unexpected bithumb isDetails query %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"market":"KRW-ETH","korean_name":"이더리움","english_name":"Ethereum","market_warning":"NONE"}
		]`))
	}))
	defer server.Close()

	client := NewBithumbExchangeListingClient(server.URL, server.Client())
	listings, err := client.FetchBithumbListings(context.Background())
	if err != nil {
		t.Fatalf("FetchBithumbListings returned error: %v", err)
	}

	if len(listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(listings))
	}
	if listings[0].Exchange != ExchangeBithumb {
		t.Fatalf("unexpected exchange %q", listings[0].Exchange)
	}
	if listings[0].DisplayName != "Ethereum" {
		t.Fatalf("unexpected display name %q", listings[0].DisplayName)
	}
}

func TestNormalizeExchangeListingSkipsInvalidMarket(t *testing.T) {
	t.Parallel()

	if _, ok := normalizeExchangeListing(ExchangeUpbit, "BTC", "Bitcoin", "비트코인", "", nil); ok {
		t.Fatal("expected invalid market to be skipped")
	}
}
