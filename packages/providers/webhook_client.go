package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

const defaultAlchemyNotifyBaseURL = "https://dashboard.alchemy.com"

type AlchemyWebhookClient struct {
	baseURL   string
	authToken string
	http      jsonHTTPClient
}

type alchemyWebhookAddressesResponse struct {
	Data       []string `json:"data"`
	Pagination struct {
		Cursors struct {
			After string `json:"after"`
		} `json:"cursors"`
		TotalCount int `json:"total_count"`
	} `json:"pagination"`
}

func NewAlchemyWebhookClient(baseURL, authToken string, client *http.Client) *AlchemyWebhookClient {
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmedBase == "" {
		trimmedBase = defaultAlchemyNotifyBaseURL
	}
	return &AlchemyWebhookClient{
		baseURL:   trimmedBase,
		authToken: strings.TrimSpace(authToken),
		http:      newJSONHTTPClient(client),
	}
}

func (c *AlchemyWebhookClient) ListWebhookAddresses(ctx context.Context, webhookID string, limit int) ([]string, error) {
	if c == nil || c.authToken == "" {
		return nil, fmt.Errorf("alchemy webhook client is not configured")
	}
	if strings.TrimSpace(webhookID) == "" {
		return nil, fmt.Errorf("alchemy webhook id is required")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	addresses := make([]string, 0, limit)
	after := ""
	for {
		reqURL, err := url.Parse(c.baseURL + "/api/webhook-addresses")
		if err != nil {
			return nil, err
		}
		query := reqURL.Query()
		query.Set("webhook_id", strings.TrimSpace(webhookID))
		query.Set("limit", fmt.Sprintf("%d", limit))
		if after != "" {
			query.Set("after", after)
		}
		reqURL.RawQuery = query.Encode()

		req, err := newJSONRequest(http.MethodGet, reqURL.String(), nil)
		if err != nil {
			return nil, err
		}
		req = req.WithContext(ctx)
		req.Header.Set("X-Alchemy-Token", c.authToken)

		var response alchemyWebhookAddressesResponse
		if err := c.http.doJSONRequest(req, &response); err != nil {
			return nil, err
		}
		addresses = append(addresses, response.Data...)
		if strings.TrimSpace(response.Pagination.Cursors.After) == "" {
			break
		}
		after = strings.TrimSpace(response.Pagination.Cursors.After)
	}

	return uniqueSortedStrings(addresses), nil
}

func (c *AlchemyWebhookClient) ReplaceWebhookAddresses(ctx context.Context, webhookID string, addresses []string) error {
	if c == nil || c.authToken == "" {
		return fmt.Errorf("alchemy webhook client is not configured")
	}
	if strings.TrimSpace(webhookID) == "" {
		return fmt.Errorf("alchemy webhook id is required")
	}

	req, err := newJSONRequest(http.MethodPut, c.baseURL+"/api/update-webhook-addresses", map[string]any{
		"webhook_id": strings.TrimSpace(webhookID),
		"addresses":  uniqueSortedStrings(addresses),
	})
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header.Set("X-Alchemy-Token", c.authToken)
	_, err = c.http.doJSONRequestWithRaw(req, nil)
	return err
}

type HeliusWebhookClient struct {
	dataAPIBaseURL string
	apiKey         string
	http           jsonHTTPClient
}

type HeliusWebhookRecord struct {
	WebhookID        string   `json:"webhookID"`
	WebhookURL       string   `json:"webhookURL"`
	TransactionTypes []string `json:"transactionTypes"`
	AccountAddresses []string `json:"accountAddresses"`
	WebhookType      string   `json:"webhookType"`
	AuthHeader       string   `json:"authHeader"`
	Encoding         string   `json:"encoding"`
	TxnStatus        string   `json:"txnStatus"`
	Active           bool     `json:"active"`
}

func NewHeliusWebhookClient(dataAPIBaseURL, apiKey string, client *http.Client) *HeliusWebhookClient {
	return &HeliusWebhookClient{
		dataAPIBaseURL: strings.TrimRight(strings.TrimSpace(dataAPIBaseURL), "/"),
		apiKey:         strings.TrimSpace(apiKey),
		http:           newJSONHTTPClient(client),
	}
}

func (c *HeliusWebhookClient) GetWebhook(ctx context.Context, webhookID string) (HeliusWebhookRecord, error) {
	if c == nil || c.apiKey == "" || c.dataAPIBaseURL == "" {
		return HeliusWebhookRecord{}, fmt.Errorf("helius webhook client is not configured")
	}
	if strings.TrimSpace(webhookID) == "" {
		return HeliusWebhookRecord{}, fmt.Errorf("helius webhook id is required")
	}
	reqURL := c.dataAPIBaseURL + "/webhooks/" + url.PathEscape(strings.TrimSpace(webhookID)) + "?api-key=" + url.QueryEscape(c.apiKey)
	req, err := newJSONRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return HeliusWebhookRecord{}, err
	}
	req = req.WithContext(ctx)
	var record HeliusWebhookRecord
	if err := c.http.doJSONRequest(req, &record); err != nil {
		return HeliusWebhookRecord{}, err
	}
	record.AccountAddresses = uniqueSortedStrings(record.AccountAddresses)
	record.TransactionTypes = uniqueSortedStrings(record.TransactionTypes)
	return record, nil
}

func (c *HeliusWebhookClient) ReplaceWebhookAddresses(ctx context.Context, webhookID string, addresses []string) error {
	record, err := c.GetWebhook(ctx, webhookID)
	if err != nil {
		return err
	}
	reqURL := c.dataAPIBaseURL + "/webhooks/" + url.PathEscape(strings.TrimSpace(webhookID)) + "?api-key=" + url.QueryEscape(c.apiKey)
	req, err := newJSONRequest(http.MethodPut, reqURL, map[string]any{
		"webhookURL":       record.WebhookURL,
		"transactionTypes": record.TransactionTypes,
		"accountAddresses": uniqueSortedStrings(addresses),
		"webhookType":      record.WebhookType,
		"authHeader":       record.AuthHeader,
		"encoding":         record.Encoding,
		"txnStatus":        record.TxnStatus,
	})
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	_, err = c.http.doJSONRequestWithRaw(req, nil)
	return err
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	slices.Sort(result)
	return result
}
