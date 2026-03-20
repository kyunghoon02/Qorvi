package providers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestHeliusClientFetchHistoricalWalletActivity(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if got := r.URL.Query().Get("api-key"); got != "test-helius-key" {
			t.Fatalf("unexpected api key query %q", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		defer r.Body.Close()

		switch r.URL.Path {
		case "/":
			var request heliusTransactionsRequest
			if err := json.Unmarshal(body, &request); err != nil {
				t.Fatalf("unmarshal request: %v", err)
			}
			if request.Method != "getTransactionsForAddress" {
				t.Fatalf("unexpected method %s", request.Method)
			}
			if len(request.Params) != 2 {
				t.Fatalf("expected 2 params entries, got %d", len(request.Params))
			}

			_, _ = io.WriteString(w, `{
				"jsonrpc":"2.0",
				"id":1,
				"result":{
					"data":[
						{
							"signature":"5h6xBEauJ3PK6SWCZ1PGjBvj8vDdWG3KpwATGy1ARAXFSDwt8GFXM7W5Ncn16wmqokgpiKRLuS83KUxyZyv2sUYv",
							"slot":1054,
							"transactionIndex":42,
							"blockTime":1641038400
						}
					],
					"paginationToken":""
				}
			}`)
		case "/v0/transactions":
			var request heliusTransactionsParseRequest
			if err := json.Unmarshal(body, &request); err != nil {
				t.Fatalf("unmarshal parse request: %v", err)
			}
			if len(request.Transactions) != 1 {
				t.Fatalf("expected 1 transaction, got %d", len(request.Transactions))
			}

			_, _ = io.WriteString(w, `[
				{
					"signature":"5h6xBEauJ3PK6SWCZ1PGjBvj8vDdWG3KpwATGy1ARAXFSDwt8GFXM7W5Ncn16wmqokgpiKRLuS83KUxyZyv2sUYv",
					"fee":5000,
					"feePayer":"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
					"type":"TRANSFER",
					"source":"TRANSFER",
					"timestamp":1641038400,
					"nativeTransfers":[{"amount":1}],
					"tokenTransfers":[{"mint":"USDC"}],
					"accountData":[{"account":"foo"}],
					"transactionError":null,
					"description":"transfer"
				}
			]`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewHeliusClient(ProviderCredentials{
		Provider:       ProviderHelius,
		APIKey:         "test-helius-key",
		BaseURL:        server.URL,
		DataAPIBaseURL: server.URL + "/v0",
	}, server.Client())

	batch := CreateHistoricalBackfillBatchFixture(ProviderHelius, domain.ChainSolana, "7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq")
	activities, err := client.FetchHistoricalWalletActivity(batch)
	if err != nil {
		t.Fatalf("FetchHistoricalWalletActivity returned error: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if activities[0].Provider != ProviderHelius {
		t.Fatalf("unexpected provider %q", activities[0].Provider)
	}
	if activities[0].Metadata["tx_hash"] == "" {
		t.Fatal("expected tx hash metadata")
	}
	if activities[0].Metadata["helius_fee_lamports"] != int64(5000) {
		t.Fatalf("unexpected enrichment fee %#v", activities[0].Metadata["helius_fee_lamports"])
	}
	if activities[0].Metadata["helius_native_transfer_count"] != 1 {
		t.Fatalf("unexpected native transfer count %#v", activities[0].Metadata["helius_native_transfer_count"])
	}
	if activities[0].Metadata["helius_data_api_raw_payload_sha256"] == "" {
		t.Fatal("expected prefixed data api raw payload metadata")
	}
	if activities[0].ObservedAt.Unix() != 1641038400 {
		t.Fatalf("unexpected observed_at %v", activities[0].ObservedAt)
	}
}
