package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

func TestHeliusClientFetchHistoricalWalletActivity(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("unexpected method %s", req.Method)
			}
			if got := req.URL.Query().Get("api-key"); got != "test-helius-key" {
				t.Fatalf("unexpected api key query %q", got)
			}

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
						{"signature":"helius_sig_1","slot":1054,"blockTime":1641038400}
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
							"signatures":["helius_sig_1"],
							"message":{
								"accountKeys":[
									{"pubkey":"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq"},
									{"pubkey":"Counterparty111111111111111111111111111111111"}
								]
							}
						},
						"meta":{
							"preBalances":[100,900],
							"postBalances":[50,950]
						}
					}
				}`), nil
			default:
				t.Fatalf("unexpected rpc method %s", request.Method)
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
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if activities[0].Provider != ProviderHelius {
		t.Fatalf("unexpected provider %q", activities[0].Provider)
	}
	if activities[0].Metadata["tx_hash"] != "helius_sig_1" {
		t.Fatalf("unexpected tx hash %#v", activities[0].Metadata["tx_hash"])
	}
	if activities[0].Metadata["direction"] != string(domain.TransactionDirectionOutbound) {
		t.Fatalf("unexpected direction %#v", activities[0].Metadata["direction"])
	}
	if activities[0].Metadata["counterparty_address"] != "Counterparty111111111111111111111111111111111" {
		t.Fatalf("unexpected counterparty %#v", activities[0].Metadata["counterparty_address"])
	}
	if activities[0].Metadata["amount"] != "50" {
		t.Fatalf("unexpected amount %#v", activities[0].Metadata["amount"])
	}
	if activities[0].ObservedAt.Unix() != 1641038400 {
		t.Fatalf("unexpected observed_at %v", activities[0].ObservedAt)
	}
}

func TestBuildHeliusHistoricalTransferMetadataHandlesInboundNativeFunding(t *testing.T) {
	t.Parallel()

	metadata := buildHeliusHistoricalTransferMetadata(
		"TargetWallet1111111111111111111111111111111111",
		heliusEnhancedTransaction{
			Source:   "SYSTEM_PROGRAM",
			FeePayer: "TargetWallet1111111111111111111111111111111111",
			NativeTransfers: []heliusEnhancedNativeTransfer{
				{
					FromUserAccount: "FundingWallet11111111111111111111111111111111",
					ToUserAccount:   "TargetWallet1111111111111111111111111111111111",
					Amount:          2500000,
				},
			},
		},
	)

	if metadata["direction"] != string(domain.TransactionDirectionInbound) {
		t.Fatalf("unexpected direction %#v", metadata["direction"])
	}
	if metadata["counterparty_address"] != "FundingWallet11111111111111111111111111111111" {
		t.Fatalf("unexpected counterparty %#v", metadata["counterparty_address"])
	}
	if metadata["funder_address"] != "FundingWallet11111111111111111111111111111111" {
		t.Fatalf("unexpected funder %#v", metadata["funder_address"])
	}
	if metadata["amount"] != "2500000" {
		t.Fatalf("unexpected amount %#v", metadata["amount"])
	}
}

func TestShouldSkipHeliusEnrichment(t *testing.T) {
	t.Parallel()

	if !shouldSkipHeliusEnrichment(fmt.Errorf("unexpected status 403: {\"message\":\"This feature is only available for paid plans. Please upgrade if you would like to gain access.\"}")) {
		t.Fatal("expected paid-plan 403 to be skipped")
	}
	if shouldSkipHeliusEnrichment(fmt.Errorf("unexpected status 500: boom")) {
		t.Fatal("expected non-403 errors not to be skipped")
	}
}

func TestShouldFallbackHeliusHistorical(t *testing.T) {
	t.Parallel()

	if !shouldFallbackHeliusHistorical(fmt.Errorf("unexpected status 403: {\"message\":\"This feature is only available for paid plans. Please upgrade if you would like to gain access.\"}")) {
		t.Fatal("expected paid-plan 403 to trigger fallback")
	}
	if shouldFallbackHeliusHistorical(fmt.Errorf("unexpected status 500: boom")) {
		t.Fatal("expected non-403 errors not to trigger fallback")
	}
}

func TestHeliusClientFetchHistoricalWalletActivityReturnsRPCFailure(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusForbidden, `{"jsonrpc":"2.0","error":{"code":-32403,"message":"rpc denied"}}`), nil
		}),
	}

	client := NewHeliusClient(ProviderCredentials{
		Provider:       ProviderHelius,
		APIKey:         "helius-key",
		BaseURL:        "https://helius.invalid",
		DataAPIBaseURL: "https://helius.invalid/v0",
	}, httpClient)

	batch := HistoricalBackfillBatch{
		Provider: ProviderHelius,
		Request: ProviderRequestContext{
			Chain:         domain.ChainSolana,
			WalletAddress: "TargetWallet1111111111111111111111111111111111",
			Access: domain.AccessContext{
				Role: domain.RoleOperator,
				Plan: domain.PlanPro,
			},
		},
		WindowStart: time.Unix(1641030000, 0).UTC(),
		WindowEnd:   time.Unix(1641040000, 0).UTC(),
		Limit:       25,
	}

	_, err := client.FetchHistoricalWalletActivity(batch)
	if err == nil {
		t.Fatal("expected rpc failure")
	}
	if got := err.Error(); !strings.Contains(got, "unexpected status 403") {
		t.Fatalf("expected wrapped rpc failure, got %q", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}
