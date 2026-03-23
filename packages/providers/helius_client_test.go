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

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestHeliusClientFetchHistoricalWalletActivity(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
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

			switch req.URL.Path {
			case "", "/":
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

				return jsonHTTPResponse(200, `{
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
			}`), nil
			case "/v0/transactions":
				var request heliusTransactionsParseRequest
				if err := json.Unmarshal(body, &request); err != nil {
					t.Fatalf("unmarshal parse request: %v", err)
				}
				if len(request.Transactions) != 1 {
					t.Fatalf("expected 1 transaction, got %d", len(request.Transactions))
				}

				return jsonHTTPResponse(200, `[
				{
					"signature":"5h6xBEauJ3PK6SWCZ1PGjBvj8vDdWG3KpwATGy1ARAXFSDwt8GFXM7W5Ncn16wmqokgpiKRLuS83KUxyZyv2sUYv",
					"fee":5000,
					"feePayer":"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
					"type":"TRANSFER",
					"source":"TRANSFER",
					"timestamp":1641038400,
					"nativeTransfers":[
						{
							"fromUserAccount":"FundingWallet11111111111111111111111111111111",
							"toUserAccount":"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
							"amount":1
						}
					],
					"tokenTransfers":[
						{
							"fromUserAccount":"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
							"toUserAccount":"Counterparty111111111111111111111111111111111",
							"mint":"So11111111111111111111111111111111111111112",
							"tokenAmount":"12.5",
							"symbol":"SOL",
							"decimals":9
						}
					],
					"accountData":[{"account":"foo"}],
					"transactionError":null,
					"description":"transfer"
				}
			]`), nil
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
	if activities[0].Metadata["direction"] != string(domain.TransactionDirectionOutbound) {
		t.Fatalf("unexpected direction %#v", activities[0].Metadata["direction"])
	}
	if activities[0].Metadata["counterparty_address"] != "Counterparty111111111111111111111111111111111" {
		t.Fatalf("unexpected counterparty %#v", activities[0].Metadata["counterparty_address"])
	}
	if activities[0].Metadata["amount"] != "12.5" {
		t.Fatalf("unexpected amount %#v", activities[0].Metadata["amount"])
	}
	if activities[0].Metadata["token_address"] != "So11111111111111111111111111111111111111112" {
		t.Fatalf("unexpected token address %#v", activities[0].Metadata["token_address"])
	}
	if activities[0].Metadata["token_symbol"] != "SOL" {
		t.Fatalf("unexpected token symbol %#v", activities[0].Metadata["token_symbol"])
	}
	if activities[0].Metadata["funder_address"] != "FundingWallet11111111111111111111111111111111" {
		t.Fatalf("unexpected funder %#v", activities[0].Metadata["funder_address"])
	}
	if activities[0].Metadata["schema_version"] != 2 {
		t.Fatalf("unexpected schema version %#v", activities[0].Metadata["schema_version"])
	}
	if activities[0].Metadata["helius_data_api_raw_payload_sha256"] == "" {
		t.Fatal("expected prefixed data api raw payload metadata")
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

func TestHeliusClientFetchHistoricalWalletActivitySkipsPaidPlanEnrichment403(t *testing.T) {
	t.Parallel()

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
				return jsonHTTPResponse(200, `{
				"jsonrpc":"2.0",
				"id":1,
				"result":{
					"data":[
						{
							"signature":"sig_paid_plan_fallback",
							"slot":1054,
							"transactionIndex":42,
							"blockTime":1641038400
						}
					],
					"paginationToken":""
				}
			}`), nil
			case "/v0/transactions":
				return jsonHTTPResponse(http.StatusForbidden, `{"jsonrpc":"2.0","error":{"code":-32403,"message":"This feature is only available for paid plans. Please upgrade if you would like to gain access."}}`), nil
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
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if got := activities[0].Metadata["direction"]; got != string(domain.TransactionDirectionUnknown) {
		t.Fatalf("expected fallback unknown direction without enrichment, got %#v", got)
	}
	if _, ok := activities[0].Metadata["helius_fee_lamports"]; ok {
		t.Fatalf("expected no enrichment metadata when paid-plan endpoint is unavailable, got %#v", activities[0].Metadata["helius_fee_lamports"])
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

func TestHeliusClientFetchHistoricalWalletActivityFallsBackToAlchemySolanaRPCOnPaidPlan403(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.Host == "helius.invalid":
				return jsonResponse(http.StatusForbidden, `{"jsonrpc":"2.0","error":{"code":-32403,"message":"This feature is only available for paid plans. Please upgrade if you would like to gain access."}}`), nil
			case req.URL.Host == "alchemy.invalid" && req.URL.Path == "/v2/alchemy-solana-key":
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("read request body: %v", err)
				}
				defer req.Body.Close()

				var rpcRequest solanaRPCRequest
				if err := json.Unmarshal(body, &rpcRequest); err != nil {
					t.Fatalf("unmarshal request: %v", err)
				}

				switch rpcRequest.Method {
				case "getSignaturesForAddress":
					if len(rpcRequest.Params) == 2 {
						if options, ok := rpcRequest.Params[1].(map[string]any); ok && options["before"] != nil {
							return jsonResponse(http.StatusOK, `{"jsonrpc":"2.0","id":1,"result":[]}`), nil
						}
					}
					return jsonResponse(http.StatusOK, `{"jsonrpc":"2.0","id":1,"result":[{"signature":"solana_fallback_sig","slot":111,"blockTime":1641038400}]}`), nil
				case "getTransaction":
					return jsonResponse(http.StatusOK, `{"jsonrpc":"2.0","id":1,"result":{"slot":111,"blockTime":1641038400,"transaction":{"signatures":["solana_fallback_sig"],"message":{"accountKeys":[{"pubkey":"TargetWallet1111111111111111111111111111111111"},{"pubkey":"Counterparty1111111111111111111111111111111"}]}},"meta":{"preBalances":[100,900],"postBalances":[50,950]}}}`), nil
				default:
					t.Fatalf("unexpected rpc method %s", rpcRequest.Method)
				}
			}

			t.Fatalf("unexpected request url %s", req.URL.String())
			return nil, nil
		}),
	}

	client := NewHeliusClient(ProviderCredentials{
		Provider:        ProviderHelius,
		APIKey:          "helius-key",
		BaseURL:         "https://helius.invalid",
		DataAPIBaseURL:  "https://helius.invalid/v0",
		FallbackAPIKey:  "alchemy-solana-key",
		FallbackBaseURL: "https://alchemy.invalid",
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

	activities, err := client.FetchHistoricalWalletActivity(batch)
	if err != nil {
		t.Fatalf("FetchHistoricalWalletActivity returned error: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 fallback activity, got %d", len(activities))
	}
	if activities[0].Provider != ProviderAlchemy {
		t.Fatalf("expected fallback provider %q, got %q", ProviderAlchemy, activities[0].Provider)
	}
	if activities[0].Metadata["counterparty_address"] != "Counterparty1111111111111111111111111111111" {
		t.Fatalf("unexpected fallback counterparty %#v", activities[0].Metadata["counterparty_address"])
	}
	if activities[0].Metadata["amount"] != "50" {
		t.Fatalf("unexpected fallback amount %#v", activities[0].Metadata["amount"])
	}
}

func TestHeliusClientFetchHistoricalWalletActivityReturnsFallbackFailure(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Host {
			case "helius.invalid":
				return jsonResponse(http.StatusForbidden, `{"jsonrpc":"2.0","error":{"code":-32403,"message":"This feature is only available for paid plans. Please upgrade if you would like to gain access."}}`), nil
			case "alchemy.invalid":
				return jsonResponse(http.StatusForbidden, `{"error":"alchemy fallback denied"}`), nil
			default:
				t.Fatalf("unexpected request url %s", req.URL.String())
				return nil, nil
			}
		}),
	}

	client := NewHeliusClient(ProviderCredentials{
		Provider:        ProviderHelius,
		APIKey:          "helius-key",
		BaseURL:         "https://helius.invalid",
		DataAPIBaseURL:  "https://helius.invalid/v0",
		FallbackAPIKey:  "alchemy-solana-key",
		FallbackBaseURL: "https://alchemy.invalid",
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
		t.Fatal("expected fallback failure")
	}
	if got := err.Error(); !strings.Contains(got, "fallback to alchemy failed") || !strings.Contains(got, "unexpected status 403") {
		t.Fatalf("expected wrapped fallback failure, got %q", got)
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
