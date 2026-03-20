package providers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
)

type AlchemyClient struct {
	baseURL string
	apiKey  string
	http    jsonHTTPClient
}

func NewAlchemyClient(credentials ProviderCredentials, client *http.Client) *AlchemyClient {
	return &AlchemyClient{
		baseURL: strings.TrimRight(credentials.BaseURL, "/"),
		apiKey:  strings.TrimSpace(credentials.APIKey),
		http:    newJSONHTTPClient(client),
	}
}

func (c *AlchemyClient) FetchHistoricalWalletActivity(batch HistoricalBackfillBatch) ([]ProviderWalletActivity, error) {
	if c == nil {
		return nil, fmt.Errorf("alchemy client is nil")
	}
	if err := batch.Validate(); err != nil {
		return nil, err
	}
	endpoint, err := c.endpoint()
	if err != nil {
		return nil, err
	}

	activities := make([]ProviderWalletActivity, 0, batch.Limit)
	pageKey := ""

	for len(activities) < batch.Limit {
		requestBody := alchemyAssetTransfersRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "alchemy_getAssetTransfers",
			Params: []alchemyAssetTransfersParams{{
				FromBlock:        "0x0",
				ToBlock:          "latest",
				ToAddress:        batch.Request.WalletAddress,
				Category:         []string{"external", "erc20", "erc721", "erc1155"},
				WithMetadata:     true,
				ExcludeZeroValue: true,
				MaxCount:         minInt(batch.Limit-len(activities), 1000),
				PageKey:          pageKey,
				Order:            "desc",
			}},
		}

		req, err := newJSONRequest(http.MethodPost, endpoint, requestBody)
		if err != nil {
			return nil, err
		}

		response := alchemyAssetTransfersResponse{}
		rawBody, err := c.http.doJSONRequestWithRaw(req, &response)
		if err != nil {
			return nil, err
		}
		if response.Error != nil {
			return nil, fmt.Errorf("alchemy transfers api error: %s", response.Error.Message)
		}
		pageMetadata := capturePagePayloadMetadata(
			ProviderAlchemy,
			"alchemy_getAssetTransfers",
			batch.WindowEnd,
			response.Result.PageKey,
			rawBody,
			map[string]any{
				"response_page_key": response.Result.PageKey,
				"response_count":    len(response.Result.Transfers),
			},
		)

		for _, transfer := range response.Result.Transfers {
			activities = append(activities, alchemyTransferToActivity(batch, transfer, len(activities), pageMetadata))
			if len(activities) >= batch.Limit {
				break
			}
		}

		pageKey = response.Result.PageKey
		if pageKey == "" {
			break
		}
	}

	return activities, nil
}

func (c *AlchemyClient) endpoint() (string, error) {
	parsed, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse alchemy base url: %w", err)
	}
	trimmedPath := strings.TrimRight(parsed.Path, "/")
	if strings.Contains(trimmedPath, "/v2/") && !strings.HasSuffix(trimmedPath, "/v2") {
		return parsed.String(), nil
	}
	parsed.Path = trimmedPath + "/v2/" + url.PathEscape(c.apiKey)
	parsed.RawQuery = ""
	return parsed.String(), nil
}

type alchemyAssetTransfersRequest struct {
	JSONRPC string                        `json:"jsonrpc"`
	ID      int                           `json:"id"`
	Method  string                        `json:"method"`
	Params  []alchemyAssetTransfersParams `json:"params"`
}

type alchemyAssetTransfersParams struct {
	FromBlock        string   `json:"fromBlock"`
	ToBlock          string   `json:"toBlock"`
	ToAddress        string   `json:"toAddress,omitempty"`
	Category         []string `json:"category,omitempty"`
	WithMetadata     bool     `json:"withMetadata"`
	ExcludeZeroValue bool     `json:"excludeZeroValue"`
	MaxCount         int      `json:"maxCount"`
	PageKey          string   `json:"pageKey,omitempty"`
	Order            string   `json:"order,omitempty"`
}

type alchemyAssetTransfersResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Transfers []alchemyAssetTransfer `json:"transfers"`
		PageKey   string                 `json:"pageKey"`
	} `json:"result"`
	Error *alchemyRPCError `json:"error,omitempty"`
}

type alchemyRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type alchemyAssetTransfer struct {
	BlockNum    string `json:"blockNum"`
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       any    `json:"value"`
	Asset       string `json:"asset"`
	Category    string `json:"category"`
	RawContract struct {
		Value   string `json:"value"`
		Address string `json:"address"`
		Decimal string `json:"decimal"`
	} `json:"rawContract"`
}

func alchemyTransferToActivity(batch HistoricalBackfillBatch, transfer alchemyAssetTransfer, index int, pageMetadata map[string]any) ProviderWalletActivity {
	metadata := mergeMetadata(pageMetadata, map[string]any{
		"tx_hash":              transfer.Hash,
		"raw_payload_path":     fmt.Sprintf("alchemy://transfers/%s", transfer.Hash),
		"direction":            alchemyTransferDirection(batch.Request.WalletAddress, transfer),
		"amount":               fmt.Sprint(transfer.Value),
		"block_number":         parseHexInt64(transfer.BlockNum),
		"transaction_index":    index,
		"kind":                 transfer.Category,
		"token_symbol":         transfer.Asset,
		"token_address":        transfer.RawContract.Address,
		"token_decimals":       parseHexInt(transfer.RawContract.Decimal),
		"counterparty_address": alchemyCounterparty(batch.Request.WalletAddress, transfer),
	})

	return CreateProviderActivityFixture(ProviderActivityFixtureInput{
		Provider:      ProviderAlchemy,
		Chain:         batch.Request.Chain,
		WalletAddress: batch.Request.WalletAddress,
		SourceID:      "alchemy_getAssetTransfers",
		Kind:          "transfer",
		Confidence:    0.91,
		ObservedAt:    batch.WindowEnd.Add(-time.Duration(index) * time.Minute),
		Metadata:      metadata,
	})
}

func alchemyTransferDirection(walletAddress string, transfer alchemyAssetTransfer) string {
	switch {
	case strings.EqualFold(transfer.From, walletAddress) && strings.EqualFold(transfer.To, walletAddress):
		return string(domain.TransactionDirectionSelf)
	case strings.EqualFold(transfer.From, walletAddress):
		return string(domain.TransactionDirectionOutbound)
	case strings.EqualFold(transfer.To, walletAddress):
		return string(domain.TransactionDirectionInbound)
	default:
		return string(domain.TransactionDirectionUnknown)
	}
}

func alchemyCounterparty(walletAddress string, transfer alchemyAssetTransfer) string {
	switch {
	case strings.EqualFold(transfer.From, walletAddress):
		return transfer.To
	case strings.EqualFold(transfer.To, walletAddress):
		return transfer.From
	default:
		return ""
	}
}

func parseHexInt64(raw string) int64 {
	value, _ := strconv.ParseInt(strings.TrimPrefix(strings.TrimSpace(raw), "0x"), 16, 64)
	return value
}

func parseHexInt(raw string) int {
	value, _ := strconv.ParseInt(strings.TrimPrefix(strings.TrimSpace(raw), "0x"), 16, 64)
	return int(value)
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
