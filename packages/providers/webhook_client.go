package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

const defaultAlchemyNotifyBaseURL = "https://dashboard.alchemy.com"
const defaultAlchemyAddressActivityNetwork = "ETH_MAINNET"

type WebhookEnsureResult struct {
	WebhookID  string
	Created    bool
	Discovered bool
}

type AlchemyWebhookClient struct {
	baseURL                string
	authToken              string
	addressActivityNetwork string
	http                   jsonHTTPClient
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

type AlchemyWebhookRecord struct {
	ID          string   `json:"id"`
	Network     string   `json:"network"`
	WebhookType string   `json:"webhook_type"`
	WebhookURL  string   `json:"webhook_url"`
	IsActive    bool     `json:"is_active"`
	Addresses   []string `json:"addresses"`
}

type alchemyWebhookListPage struct {
	Data []AlchemyWebhookRecord `json:"data"`
}

func NewAlchemyWebhookClient(baseURL, authToken, addressActivityNetwork string, client *http.Client) *AlchemyWebhookClient {
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmedBase == "" {
		trimmedBase = defaultAlchemyNotifyBaseURL
	}
	trimmedNetwork := strings.ToUpper(strings.TrimSpace(addressActivityNetwork))
	if trimmedNetwork == "" {
		trimmedNetwork = defaultAlchemyAddressActivityNetwork
	}
	return &AlchemyWebhookClient{
		baseURL:                trimmedBase,
		authToken:              strings.TrimSpace(authToken),
		addressActivityNetwork: trimmedNetwork,
		http:                   newJSONHTTPClient(client),
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

func (c *AlchemyWebhookClient) ListWebhooks(ctx context.Context) ([]AlchemyWebhookRecord, error) {
	if c == nil || c.authToken == "" {
		return nil, fmt.Errorf("alchemy webhook client is not configured")
	}
	req, err := newJSONRequest(http.MethodGet, c.baseURL+"/api/team-webhooks", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("X-Alchemy-Token", c.authToken)

	var pages []alchemyWebhookListPage
	if err := c.http.doJSONRequest(req, &pages); err != nil {
		return nil, err
	}

	records := make([]AlchemyWebhookRecord, 0)
	for _, page := range pages {
		for _, record := range page.Data {
			record.Addresses = uniqueSortedStrings(record.Addresses)
			records = append(records, record)
		}
	}

	return records, nil
}

func (c *AlchemyWebhookClient) CreateAddressActivityWebhook(
	ctx context.Context,
	callbackURL string,
	addresses []string,
) (AlchemyWebhookRecord, error) {
	if c == nil || c.authToken == "" {
		return AlchemyWebhookRecord{}, fmt.Errorf("alchemy webhook client is not configured")
	}
	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL == "" {
		return AlchemyWebhookRecord{}, fmt.Errorf("alchemy webhook callback url is required")
	}

	req, err := newJSONRequest(http.MethodPost, c.baseURL+"/api/create-webhook", map[string]any{
		"network":      c.addressActivityNetwork,
		"webhook_type": "ADDRESS_ACTIVITY",
		"webhook_url":  callbackURL,
		"addresses":    uniqueSortedStrings(addresses),
	})
	if err != nil {
		return AlchemyWebhookRecord{}, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("X-Alchemy-Token", c.authToken)

	raw, err := c.http.doJSONRequestWithRaw(req, nil)
	if err != nil {
		return AlchemyWebhookRecord{}, err
	}

	record, err := decodeAlchemyWebhookRecord(raw)
	if err != nil {
		return AlchemyWebhookRecord{}, err
	}
	record.Addresses = uniqueSortedStrings(record.Addresses)
	return record, nil
}

func (c *AlchemyWebhookClient) EnsureWebhookAddresses(
	ctx context.Context,
	configuredWebhookID string,
	callbackURL string,
	addresses []string,
) (WebhookEnsureResult, error) {
	webhookID := strings.TrimSpace(configuredWebhookID)
	if webhookID != "" {
		if err := c.ReplaceWebhookAddresses(ctx, webhookID, addresses); err != nil {
			return WebhookEnsureResult{}, err
		}
		return WebhookEnsureResult{WebhookID: webhookID}, nil
	}

	callbackURL = normalizeWebhookURL(callbackURL)
	if callbackURL == "" {
		return WebhookEnsureResult{}, fmt.Errorf("alchemy webhook callback url is required")
	}

	records, err := c.ListWebhooks(ctx)
	if err != nil {
		return WebhookEnsureResult{}, err
	}
	for _, record := range records {
		if !strings.EqualFold(strings.TrimSpace(record.WebhookType), "ADDRESS_ACTIVITY") {
			continue
		}
		if !sameNormalizedWebhookURL(record.WebhookURL, callbackURL) {
			continue
		}
		if strings.TrimSpace(record.ID) == "" {
			continue
		}
		if err := c.ReplaceWebhookAddresses(ctx, record.ID, addresses); err != nil {
			return WebhookEnsureResult{}, err
		}
		return WebhookEnsureResult{
			WebhookID:  strings.TrimSpace(record.ID),
			Discovered: true,
		}, nil
	}

	record, err := c.CreateAddressActivityWebhook(ctx, callbackURL, addresses)
	if err != nil {
		return WebhookEnsureResult{}, err
	}
	return WebhookEnsureResult{
		WebhookID: strings.TrimSpace(record.ID),
		Created:   true,
	}, nil
}

type HeliusWebhookClient struct {
	dataAPIBaseURL string
	apiKey         string
	authHeader     string
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

func NewHeliusWebhookClient(dataAPIBaseURL, apiKey, authHeader string, client *http.Client) *HeliusWebhookClient {
	return &HeliusWebhookClient{
		dataAPIBaseURL: strings.TrimRight(strings.TrimSpace(dataAPIBaseURL), "/"),
		apiKey:         strings.TrimSpace(apiKey),
		authHeader:     strings.TrimSpace(authHeader),
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

func (c *HeliusWebhookClient) ListWebhooks(ctx context.Context) ([]HeliusWebhookRecord, error) {
	if c == nil || c.apiKey == "" || c.dataAPIBaseURL == "" {
		return nil, fmt.Errorf("helius webhook client is not configured")
	}
	reqURL := c.dataAPIBaseURL + "/webhooks?api-key=" + url.QueryEscape(c.apiKey)
	req, err := newJSONRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	var records []HeliusWebhookRecord
	if err := c.http.doJSONRequest(req, &records); err != nil {
		return nil, err
	}
	for index := range records {
		records[index].AccountAddresses = uniqueSortedStrings(records[index].AccountAddresses)
		records[index].TransactionTypes = uniqueSortedStrings(records[index].TransactionTypes)
	}
	return records, nil
}

func (c *HeliusWebhookClient) CreateWebhook(
	ctx context.Context,
	callbackURL string,
	addresses []string,
) (HeliusWebhookRecord, error) {
	if c == nil || c.apiKey == "" || c.dataAPIBaseURL == "" {
		return HeliusWebhookRecord{}, fmt.Errorf("helius webhook client is not configured")
	}
	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL == "" {
		return HeliusWebhookRecord{}, fmt.Errorf("helius webhook callback url is required")
	}

	payload := map[string]any{
		"webhookURL":       callbackURL,
		"transactionTypes": []string{"ANY"},
		"accountAddresses": uniqueSortedStrings(addresses),
		"webhookType":      "enhanced",
	}
	if c.authHeader != "" {
		payload["authHeader"] = c.authHeader
	}

	reqURL := c.dataAPIBaseURL + "/webhooks?api-key=" + url.QueryEscape(c.apiKey)
	req, err := newJSONRequest(http.MethodPost, reqURL, payload)
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

func (c *HeliusWebhookClient) EnsureWebhookAddresses(
	ctx context.Context,
	configuredWebhookID string,
	callbackURL string,
	addresses []string,
) (WebhookEnsureResult, error) {
	webhookID := strings.TrimSpace(configuredWebhookID)
	if webhookID != "" {
		if err := c.ReplaceWebhookAddresses(ctx, webhookID, addresses); err != nil {
			return WebhookEnsureResult{}, err
		}
		return WebhookEnsureResult{WebhookID: webhookID}, nil
	}

	callbackURL = normalizeWebhookURL(callbackURL)
	if callbackURL == "" {
		return WebhookEnsureResult{}, fmt.Errorf("helius webhook callback url is required")
	}

	records, err := c.ListWebhooks(ctx)
	if err != nil {
		return WebhookEnsureResult{}, err
	}
	for _, record := range records {
		if !sameNormalizedWebhookURL(record.WebhookURL, callbackURL) {
			continue
		}
		if strings.TrimSpace(record.WebhookID) == "" {
			continue
		}
		if err := c.ReplaceWebhookAddresses(ctx, record.WebhookID, addresses); err != nil {
			return WebhookEnsureResult{}, err
		}
		return WebhookEnsureResult{
			WebhookID:  strings.TrimSpace(record.WebhookID),
			Discovered: true,
		}, nil
	}

	record, err := c.CreateWebhook(ctx, callbackURL, addresses)
	if err != nil {
		return WebhookEnsureResult{}, err
	}
	return WebhookEnsureResult{
		WebhookID: strings.TrimSpace(record.WebhookID),
		Created:   true,
	}, nil
}

func decodeAlchemyWebhookRecord(raw []byte) (AlchemyWebhookRecord, error) {
	var direct AlchemyWebhookRecord
	if err := json.Unmarshal(raw, &direct); err == nil && strings.TrimSpace(direct.ID) != "" {
		return direct, nil
	}

	var wrapped struct {
		Data AlchemyWebhookRecord `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && strings.TrimSpace(wrapped.Data.ID) != "" {
		return wrapped.Data, nil
	}

	return AlchemyWebhookRecord{}, fmt.Errorf("decode created alchemy webhook: missing id in response")
}

func normalizeWebhookURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func sameNormalizedWebhookURL(left string, right string) bool {
	return normalizeWebhookURL(left) == normalizeWebhookURL(right)
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
