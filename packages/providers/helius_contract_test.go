package providers

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestHeliusHistoricalContractFixture(t *testing.T) {
	t.Parallel()

	signaturesFixture := readProviderFixture(t, "helius", "historical_signatures.json")
	parsedFixture := readProviderFixture(t, "helius", "parsed_transactions.json")

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			defer req.Body.Close()

			switch req.URL.Path {
			case "", "/":
				var request heliusTransactionsRequest
				if err := json.Unmarshal(body, &request); err != nil {
					t.Fatalf("unmarshal request: %v", err)
				}
				return jsonHTTPResponse(200, signaturesFixture), nil
			case "/v0/transactions":
				var request heliusTransactionsParseRequest
				if err := json.Unmarshal(body, &request); err != nil {
					t.Fatalf("unmarshal parse request: %v", err)
				}
				return jsonHTTPResponse(200, parsedFixture), nil
			default:
				t.Fatalf("unexpected path %s", req.URL.Path)
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

	batch := CreateHistoricalBackfillBatchFixture(ProviderHelius, domain.ChainSolana, "7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq")
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
