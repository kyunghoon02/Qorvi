package providers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
)

type HeliusClient struct {
	baseURL        string
	dataAPIBaseURL string
	apiKey         string
	http           jsonHTTPClient
}

func NewHeliusClient(credentials ProviderCredentials, client *http.Client) *HeliusClient {
	return &HeliusClient{
		baseURL:        strings.TrimRight(credentials.BaseURL, "/"),
		dataAPIBaseURL: strings.TrimRight(credentials.DataAPIBaseURL, "/"),
		apiKey:         strings.TrimSpace(credentials.APIKey),
		http:           newJSONHTTPClient(client),
	}
}

func (c *HeliusClient) FetchHistoricalWalletActivity(batch HistoricalBackfillBatch) ([]ProviderWalletActivity, error) {
	if c == nil {
		return nil, fmt.Errorf("helius client is nil")
	}
	if err := batch.Validate(); err != nil {
		return nil, err
	}
	endpoint, err := c.endpoint()
	if err != nil {
		return nil, err
	}

	activities := make([]ProviderWalletActivity, 0, batch.Limit)
	paginationToken := ""

	for len(activities) < batch.Limit {
		requestBody := heliusTransactionsRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "getTransactionsForAddress",
			Params: []any{
				batch.Request.WalletAddress,
				heliusTransactionsOptions{
					TransactionDetails: "signatures",
					SortOrder:          "desc",
					Limit:              minInt(batch.Limit-len(activities), 1000),
					PaginationToken:    paginationToken,
					Filters: heliusTransactionsFilters{
						BlockTime: heliusRangeFilter{
							GTE: batch.WindowStart.Unix(),
							LTE: batch.WindowEnd.Unix(),
						},
						Status: "succeeded",
					},
				},
			},
		}

		req, err := newJSONRequest(http.MethodPost, endpoint, requestBody)
		if err != nil {
			return nil, err
		}

		response := heliusTransactionsResponse{}
		rawBody, err := c.http.doJSONRequestWithRaw(req, &response)
		if err != nil {
			return nil, err
		}
		if response.Error != nil {
			return nil, fmt.Errorf("helius transactions api error: %s", response.Error.Message)
		}
		enrichmentBySignature, err := c.fetchTransactionEnrichment(batch, response.Result.Data)
		if err != nil {
			return nil, err
		}
		pageMetadata := capturePagePayloadMetadata(
			ProviderHelius,
			"getTransactionsForAddress",
			batch.WindowEnd,
			response.Result.PaginationToken,
			rawBody,
			map[string]any{
				"response_pagination_token": response.Result.PaginationToken,
				"response_count":            len(response.Result.Data),
			},
		)

		for _, tx := range response.Result.Data {
			activities = append(activities, heliusTransactionToActivity(batch, tx, len(activities), pageMetadata, enrichmentBySignature[tx.Signature]))
			if len(activities) >= batch.Limit {
				break
			}
		}

		paginationToken = response.Result.PaginationToken
		if paginationToken == "" {
			break
		}
	}

	return activities, nil
}

func (c *HeliusClient) fetchTransactionEnrichment(batch HistoricalBackfillBatch, transactions []heliusTransaction) (map[string]map[string]any, error) {
	if c == nil || strings.TrimSpace(c.dataAPIBaseURL) == "" || len(transactions) == 0 {
		return map[string]map[string]any{}, nil
	}

	signatures := make([]string, 0, len(transactions))
	for _, tx := range transactions {
		if strings.TrimSpace(tx.Signature) != "" {
			signatures = append(signatures, tx.Signature)
		}
	}
	if len(signatures) == 0 {
		return map[string]map[string]any{}, nil
	}

	req, err := newJSONRequest(http.MethodPost, c.dataEndpoint(), heliusTransactionsParseRequest{
		Transactions: signatures,
	})
	if err != nil {
		return nil, err
	}

	var response []heliusEnhancedTransaction
	rawBody, err := c.http.doJSONRequestWithRaw(req, &response)
	if err != nil {
		return nil, err
	}
	payloadMetadata := capturePagePayloadMetadata(
		ProviderHelius,
		"parseTransactions",
		batch.WindowEnd,
		strings.Join(signatures, ","),
		rawBody,
		map[string]any{"response_count": len(response)},
	)
	prefixedPayloadMetadata := prefixMetadataKeys(payloadMetadata, "helius_data_api_")

	enrichmentBySignature := make(map[string]map[string]any, len(response))
	for _, item := range response {
		if strings.TrimSpace(item.Signature) == "" {
			continue
		}

		enrichmentBySignature[item.Signature] = mergeMetadata(prefixedPayloadMetadata, map[string]any{
			"helius_description":                strings.TrimSpace(item.Description),
			"helius_type":                       strings.TrimSpace(item.Type),
			"helius_source":                     strings.TrimSpace(item.Source),
			"helius_fee_lamports":               item.Fee,
			"helius_fee_payer":                  strings.TrimSpace(item.FeePayer),
			"helius_timestamp":                  item.Timestamp,
			"helius_native_transfer_count":      len(item.NativeTransfers),
			"helius_token_transfer_count":       len(item.TokenTransfers),
			"helius_account_data_count":         len(item.AccountData),
			"helius_has_transaction_error":      item.TransactionError != nil,
			"helius_enrichment_source_endpoint": c.dataAPIBaseURL,
		})
	}

	return enrichmentBySignature, nil
}

func (c *HeliusClient) dataEndpoint() string {
	parsed, err := url.Parse(c.dataAPIBaseURL)
	if err != nil {
		return c.dataAPIBaseURL
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/transactions"
	query := parsed.Query()
	query.Set("api-key", c.apiKey)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func (c *HeliusClient) endpoint() (string, error) {
	parsed, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse helius base url: %w", err)
	}
	query := parsed.Query()
	query.Set("api-key", c.apiKey)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

type heliusTransactionsRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type heliusTransactionsOptions struct {
	TransactionDetails string                    `json:"transactionDetails"`
	SortOrder          string                    `json:"sortOrder"`
	Limit              int                       `json:"limit"`
	PaginationToken    string                    `json:"paginationToken,omitempty"`
	Filters            heliusTransactionsFilters `json:"filters"`
}

type heliusTransactionsFilters struct {
	BlockTime heliusRangeFilter `json:"blockTime"`
	Status    string            `json:"status"`
}

type heliusRangeFilter struct {
	GTE int64 `json:"gte,omitempty"`
	LTE int64 `json:"lte,omitempty"`
}

type heliusTransactionsResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Data            []heliusTransaction `json:"data"`
		PaginationToken string              `json:"paginationToken"`
	} `json:"result"`
	Error *alchemyRPCError `json:"error,omitempty"`
}

type heliusTransaction struct {
	Signature        string `json:"signature"`
	Slot             int64  `json:"slot"`
	TransactionIndex int64  `json:"transactionIndex"`
	BlockTime        int64  `json:"blockTime"`
}

type heliusTransactionsParseRequest struct {
	Transactions []string `json:"transactions"`
}

type heliusEnhancedTransaction struct {
	Description      string           `json:"description"`
	Type             string           `json:"type"`
	Source           string           `json:"source"`
	Fee              int64            `json:"fee"`
	FeePayer         string           `json:"feePayer"`
	Signature        string           `json:"signature"`
	Slot             int64            `json:"slot"`
	Timestamp        int64            `json:"timestamp"`
	NativeTransfers  []map[string]any `json:"nativeTransfers"`
	TokenTransfers   []map[string]any `json:"tokenTransfers"`
	AccountData      []map[string]any `json:"accountData"`
	TransactionError map[string]any   `json:"transactionError"`
	Instructions     []map[string]any `json:"instructions"`
	Events           map[string]any   `json:"events"`
}

func heliusTransactionToActivity(batch HistoricalBackfillBatch, tx heliusTransaction, index int, pageMetadata map[string]any, enrichment map[string]any) ProviderWalletActivity {
	observedAt := batch.WindowEnd.Add(-time.Duration(index) * time.Minute)
	if tx.BlockTime > 0 {
		observedAt = time.Unix(tx.BlockTime, 0).UTC()
	}

	metadata := mergeMetadata(pageMetadata, map[string]any{
		"tx_hash":           tx.Signature,
		"raw_payload_path":  fmt.Sprintf("helius://transactions/%s", tx.Signature),
		"block_number":      tx.Slot,
		"transaction_index": tx.TransactionIndex,
		"direction":         string(domain.TransactionDirectionUnknown),
	})
	metadata = mergeMetadata(metadata, enrichment)

	return CreateProviderActivityFixture(ProviderActivityFixtureInput{
		Provider:      ProviderHelius,
		Chain:         batch.Request.Chain,
		WalletAddress: batch.Request.WalletAddress,
		SourceID:      "getTransactionsForAddress",
		Kind:          "transaction",
		Confidence:    0.87,
		ObservedAt:    observedAt,
		Metadata:      metadata,
	})
}
