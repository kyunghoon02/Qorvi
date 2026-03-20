package providers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestAlchemyClientFetchHistoricalWalletActivity(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if got := r.URL.Path; got != "/v2/test-alchemy-key" {
			t.Fatalf("unexpected path %s", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		defer r.Body.Close()

		var request alchemyAssetTransfersRequest
		if err := json.Unmarshal(body, &request); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if request.Method != "alchemy_getAssetTransfers" {
			t.Fatalf("unexpected method %s", request.Method)
		}
		if len(request.Params) != 1 {
			t.Fatalf("expected 1 params entry, got %d", len(request.Params))
		}
		if request.Params[0].ToAddress != "0x1234567890abcdef1234567890abcdef12345678" {
			t.Fatalf("unexpected to address %s", request.Params[0].ToAddress)
		}

		_, _ = io.WriteString(w, `{
			"jsonrpc":"2.0",
			"id":1,
			"result":{
				"transfers":[
					{
						"blockNum":"0xb0eadc",
						"hash":"0xabc123",
						"from":"0x1234567890abcdef1234567890abcdef12345678",
						"to":"0xef4396d9ff8107086d215a1c9f8866c54795d7c7",
						"value":0.5,
						"asset":"ETH",
						"category":"external",
						"rawContract":{"value":"0x6f05b59d3b20000","address":null,"decimal":"0x12"}
					}
				],
				"pageKey":""
			}
		}`)
	}))
	defer server.Close()

	client := NewAlchemyClient(ProviderCredentials{
		Provider: ProviderAlchemy,
		APIKey:   "test-alchemy-key",
		BaseURL:  server.URL,
	}, server.Client())

	batch := CreateHistoricalBackfillBatchFixture(ProviderAlchemy, domain.ChainEVM, "0x1234567890abcdef1234567890abcdef12345678")
	activities, err := client.FetchHistoricalWalletActivity(batch)
	if err != nil {
		t.Fatalf("FetchHistoricalWalletActivity returned error: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if activities[0].Provider != ProviderAlchemy {
		t.Fatalf("unexpected provider %q", activities[0].Provider)
	}
	if activities[0].Metadata["tx_hash"] != "0xabc123" {
		t.Fatalf("unexpected tx hash %#v", activities[0].Metadata["tx_hash"])
	}
	if activities[0].Metadata["direction"] != string(domain.TransactionDirectionOutbound) {
		t.Fatalf("unexpected direction %#v", activities[0].Metadata["direction"])
	}
}
