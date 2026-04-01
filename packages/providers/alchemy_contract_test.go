package providers

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestAlchemyHistoricalContractFixture(t *testing.T) {
	t.Parallel()

	outboundFixture := readProviderFixture(t, "alchemy", "historical_outbound.json")
	inboundFixture := readProviderFixture(t, "alchemy", "historical_inbound.json")

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			defer req.Body.Close()

			var request alchemyAssetTransfersRequest
			if err := json.Unmarshal(body, &request); err != nil {
				t.Fatalf("unmarshal request: %v", err)
			}
			if len(request.Params) != 1 {
				t.Fatalf("expected 1 params entry, got %d", len(request.Params))
			}

			if request.Params[0].FromAddress != "" {
				return jsonHTTPResponse(200, outboundFixture), nil
			}

			return jsonHTTPResponse(200, inboundFixture), nil
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

	transactions, err := NormalizeProviderActivities(activities)
	if err != nil {
		t.Fatalf("NormalizeProviderActivities returned error: %v", err)
	}

	if len(transactions) != 3 {
		t.Fatalf("expected 3 normalized transactions, got %d", len(transactions))
	}
	if transactions[0].RawPayloadPath == "" {
		t.Fatal("expected raw payload path to be populated")
	}
	directions := map[domain.TransactionDirection]int{}
	for _, tx := range transactions {
		directions[tx.Direction]++
	}
	if directions[domain.TransactionDirectionOutbound] != 1 {
		t.Fatalf("expected 1 outbound transaction, got %#v", directions)
	}
	if directions[domain.TransactionDirectionSelf] != 1 {
		t.Fatalf("expected 1 self transaction, got %#v", directions)
	}
	if directions[domain.TransactionDirectionInbound] != 1 {
		t.Fatalf("expected 1 inbound transaction, got %#v", directions)
	}
}
