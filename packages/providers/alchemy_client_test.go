package providers

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/flowintel/flowintel/packages/domain"
)

func TestAlchemyClientFetchHistoricalWalletActivity(t *testing.T) {
	t.Parallel()

	requestBodies := make([]alchemyAssetTransfersRequest, 0, 2)
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("unexpected method %s", req.Method)
			}
			if got := req.URL.Path; got != "/v2/test-alchemy-key" {
				t.Fatalf("unexpected path %s", got)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			defer req.Body.Close()

			var request alchemyAssetTransfersRequest
			if err := json.Unmarshal(body, &request); err != nil {
				t.Fatalf("unmarshal request: %v", err)
			}
			requestBodies = append(requestBodies, request)
			if request.Method != "alchemy_getAssetTransfers" {
				t.Fatalf("unexpected method %s", request.Method)
			}
			if len(request.Params) != 1 {
				t.Fatalf("expected 1 params entry, got %d", len(request.Params))
			}
			if request.Params[0].MaxCount != "0x7d" {
				t.Fatalf("unexpected max count %s", request.Params[0].MaxCount)
			}
			if request.Params[0].ToAddress == "" && request.Params[0].FromAddress == "" {
				t.Fatalf("expected either to or from address to be set")
			}

			if request.Params[0].ToAddress != "" {
				return jsonHTTPResponse(200, `{
				"jsonrpc":"2.0",
				"id":1,
				"result":{
					"transfers":[
						{
							"blockNum":"0xb0eadc",
							"hash":"0xinbound123",
							"from":"0xef4396d9ff8107086d215a1c9f8866c54795d7c7",
							"to":"0x1234567890abcdef1234567890abcdef12345678",
							"value":0.5,
							"asset":"ETH",
							"category":"external",
							"rawContract":{"value":"0x6f05b59d3b20000","address":null,"decimal":"0x12"}
						}
					],
					"pageKey":""
				}
			}`), nil
			}

			return jsonHTTPResponse(200, `{
			"jsonrpc":"2.0",
			"id":1,
			"result":{
				"transfers":[
					{
						"blockNum":"0xb0eadc",
						"hash":"0xoutbound123",
						"from":"0x1234567890abcdef1234567890abcdef12345678",
						"to":"0xef4396d9ff8107086d215a1c9f8866c54795d7c7",
						"value":0.5,
						"asset":"ETH",
						"category":"external",
						"rawContract":{"value":"0x6f05b59d3b20000","address":null,"decimal":"0x12"}
					},
					{
						"blockNum":"0xb0eadc",
						"hash":"0xself123",
						"from":"0x1234567890abcdef1234567890abcdef12345678",
						"to":"0x1234567890abcdef1234567890abcdef12345678",
						"value":0.1,
						"asset":"ETH",
						"category":"external",
						"rawContract":{"value":"0x16345785d8a0000","address":null,"decimal":"0x12"}
					}
				],
				"pageKey":""
			}
		}`), nil
		}),
	}

	client := NewAlchemyClient(ProviderCredentials{
		Provider: ProviderAlchemy,
		APIKey:   "test-alchemy-key",
		BaseURL:  "https://alchemy.example",
	}, httpClient)

	batch := CreateHistoricalBackfillBatchFixture(ProviderAlchemy, domain.ChainEVM, "0x1234567890abcdef1234567890abcdef12345678")
	activities, err := client.FetchHistoricalWalletActivity(batch)
	if err != nil {
		t.Fatalf("FetchHistoricalWalletActivity returned error: %v", err)
	}
	if len(requestBodies) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(requestBodies))
	}
	fromMatches := 0
	toMatches := 0
	for _, request := range requestBodies {
		if request.Params[0].FromAddress == "0x1234567890abcdef1234567890abcdef12345678" {
			fromMatches++
		}
		if request.Params[0].ToAddress == "0x1234567890abcdef1234567890abcdef12345678" {
			toMatches++
		}
	}
	if fromMatches != 1 || toMatches != 1 {
		t.Fatalf("expected one from-address and one to-address request, got from=%d to=%d bodies=%#v", fromMatches, toMatches, requestBodies)
	}
	if len(activities) != 3 {
		t.Fatalf("expected 3 activities, got %d", len(activities))
	}
	if activities[0].Provider != ProviderAlchemy {
		t.Fatalf("unexpected provider %q", activities[0].Provider)
	}
	txHashes := map[string]bool{}
	directions := map[string]int{}
	for _, activity := range activities {
		if txHash, ok := activity.Metadata["tx_hash"].(string); ok {
			txHashes[txHash] = true
		}
		if direction, ok := activity.Metadata["direction"].(string); ok {
			directions[direction]++
		}
	}
	if !txHashes["0xoutbound123"] || !txHashes["0xself123"] || !txHashes["0xinbound123"] {
		t.Fatalf("unexpected tx hashes %#v", txHashes)
	}
	if directions[string(domain.TransactionDirectionOutbound)] != 1 {
		t.Fatalf("unexpected outbound direction counts %#v", directions)
	}
	if directions[string(domain.TransactionDirectionSelf)] != 1 {
		t.Fatalf("unexpected self direction counts %#v", directions)
	}
	if directions[string(domain.TransactionDirectionInbound)] != 1 {
		t.Fatalf("unexpected inbound direction counts %#v", directions)
	}
}

func TestAlchemyTransferToActivityNormalizesNilAmount(t *testing.T) {
	t.Parallel()

	batch := CreateHistoricalBackfillBatchFixture(
		ProviderAlchemy,
		domain.ChainEVM,
		"0x1234567890abcdef1234567890abcdef12345678",
	)

	activity := alchemyTransferToActivity(batch, alchemyAssetTransfer{
		Hash:     "0xnilvalue",
		BlockNum: "0xb0eadc",
		From:     "0x1234567890abcdef1234567890abcdef12345678",
		To:       "0xef4396d9ff8107086d215a1c9f8866c54795d7c7",
		Value:    nil,
		Asset:    "ETH",
		Category: "external",
	}, 0, nil)

	if got := activity.Metadata["amount"]; got != "" {
		t.Fatalf("expected nil amount to normalize to empty string, got %#v", got)
	}
}
