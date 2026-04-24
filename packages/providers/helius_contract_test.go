package providers

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestHeliusHistoricalContractFixture(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			defer req.Body.Close()

			var request solanaRPCRequest
			if err := json.Unmarshal(body, &request); err != nil {
				t.Fatalf("unmarshal request: %v", err)
			}

			switch request.Method {
			case "getSignaturesForAddress":
				if len(request.Params) == 2 {
					if options, ok := request.Params[1].(map[string]any); ok && options["before"] != nil {
						return jsonHTTPResponse(200, `{"jsonrpc":"2.0","id":1,"result":[]}`), nil
					}
				}
				return jsonHTTPResponse(200, `{
					"jsonrpc":"2.0",
					"id":1,
					"result":[
						{"signature":"contract_sig_1","slot":1054,"blockTime":1641038400}
					]
				}`), nil
			case "getTransaction":
				return jsonHTTPResponse(200, `{
					"jsonrpc":"2.0",
					"id":1,
					"result":{
						"slot":1054,
						"blockTime":1641038400,
						"transaction":{
							"signatures":["contract_sig_1"],
							"message":{
								"accountKeys":[
									{"pubkey":"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq"},
									{"pubkey":"Counterparty111111111111111111111111111111111"}
								]
							}
						},
						"meta":{
							"preTokenBalances":[
								{"owner":"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq","mint":"So11111111111111111111111111111111111111112","uiTokenAmount":{"amount":"125","decimals":1}}
							],
							"postTokenBalances":[
								{"owner":"Counterparty111111111111111111111111111111111","mint":"So11111111111111111111111111111111111111112","uiTokenAmount":{"amount":"125","decimals":1}}
							]
						}
					}
				}`), nil
			default:
				t.Fatalf("unexpected method %s", request.Method)
				return nil, nil
			}
		}),
	}

	client := NewHeliusClient(ProviderCredentials{
		Provider:       ProviderHelius,
		APIKey:         "test-helius-key",
		BaseURL:        "https://helius.example",
		DataAPIBaseURL: "https://helius.example/v0",
	}, httpClient)

	batch := HistoricalBackfillBatch{
		Provider: ProviderHelius,
		Request: ProviderRequestContext{
			Chain:         domain.ChainSolana,
			WalletAddress: "7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
			Access: domain.AccessContext{
				Role: domain.RoleOperator,
				Plan: domain.PlanPro,
			},
		},
		WindowStart: time.Unix(1641030000, 0).UTC(),
		WindowEnd:   time.Unix(1641040000, 0).UTC(),
		Limit:       25,
	}
	activities, err := client.FetchHistoricalWalletActivity(batch)
	if err != nil {
		t.Fatalf("FetchHistoricalWalletActivity returned error: %v", err)
	}

	transactions, err := NormalizeProviderActivities(activities)
	if err != nil {
		t.Fatalf("NormalizeProviderActivities returned error: %v", err)
	}

	if len(transactions) != 1 {
		t.Fatalf("expected 1 normalized transaction, got %d", len(transactions))
	}
	if transactions[0].Direction != domain.TransactionDirectionOutbound {
		t.Fatalf("unexpected direction %q", transactions[0].Direction)
	}
	if transactions[0].Counterparty == nil || transactions[0].Counterparty.Address != "Counterparty111111111111111111111111111111111" {
		t.Fatalf("unexpected counterparty %#v", transactions[0].Counterparty)
	}
	if transactions[0].Token == nil || transactions[0].Token.Address != "So11111111111111111111111111111111111111112" {
		t.Fatalf("unexpected token %#v", transactions[0].Token)
	}
}
