package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

const DefaultStripeAPIBaseURL = "https://api.stripe.com"

type StripeClient interface {
	CreateCheckoutSession(context.Context, StripeConfig, StripeCheckoutSessionCreateRequest) (StripeCheckoutSessionRecord, error)
	GetSubscription(context.Context, StripeConfig, string) (StripeSubscriptionRecord, error)
}

type StripeCheckoutSessionCreateRequest struct {
	OwnerUserID   string
	Tier          domain.PlanTier
	CustomerEmail string
	SuccessURL    string
	CancelURL     string
	StripePriceID string
}

type HTTPStripeClient struct {
	HTTPClient *http.Client
}

func NewHTTPStripeClient(client *http.Client) *HTTPStripeClient {
	return &HTTPStripeClient{HTTPClient: client}
}

func (c *HTTPStripeClient) CreateCheckoutSession(
	ctx context.Context,
	config StripeConfig,
	req StripeCheckoutSessionCreateRequest,
) (StripeCheckoutSessionRecord, error) {
	form := url.Values{}
	form.Set("mode", "subscription")
	form.Set("line_items[0][price]", strings.TrimSpace(req.StripePriceID))
	form.Set("line_items[0][quantity]", "1")
	form.Set("success_url", strings.TrimSpace(req.SuccessURL))
	form.Set("cancel_url", strings.TrimSpace(req.CancelURL))
	if email := strings.TrimSpace(req.CustomerEmail); email != "" {
		form.Set("customer_email", email)
	}
	if ownerUserID := strings.TrimSpace(req.OwnerUserID); ownerUserID != "" {
		form.Set("metadata[owner_user_id]", ownerUserID)
	}
	form.Set("metadata[target_tier]", string(NormalizePlanTier(string(req.Tier))))

	responseBody, err := c.doStripeRequest(
		ctx,
		config,
		http.MethodPost,
		"/v1/checkout/sessions",
		strings.NewReader(form.Encode()),
		"application/x-www-form-urlencoded",
	)
	if err != nil {
		return StripeCheckoutSessionRecord{}, err
	}

	var payload struct {
		ID           string            `json:"id"`
		URL          string            `json:"url"`
		CustomerID   string            `json:"customer"`
		Subscription string            `json:"subscription"`
		Created      int64             `json:"created"`
		Metadata     map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return StripeCheckoutSessionRecord{}, fmt.Errorf("decode stripe checkout session: %w", err)
	}

	createdAt := time.Now().UTC()
	if payload.Created > 0 {
		createdAt = time.Unix(payload.Created, 0).UTC()
	}

	record := NormalizeStripeCheckoutSessionRecord(StripeCheckoutSessionRecord{
		SessionID:      payload.ID,
		CustomerID:     payload.CustomerID,
		CustomerEmail:  strings.TrimSpace(req.CustomerEmail),
		SubscriptionID: payload.Subscription,
		Tier:           NormalizePlanTier(string(req.Tier)),
		StripePriceID:  strings.TrimSpace(req.StripePriceID),
		Status:         StripeSessionStatusOpen,
		SuccessURL:     strings.TrimSpace(req.SuccessURL),
		CancelURL:      strings.TrimSpace(req.CancelURL),
		Metadata:       payload.Metadata,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	})
	if err := ValidateStripeCheckoutSessionRecord(record); err != nil {
		return StripeCheckoutSessionRecord{}, err
	}

	return record, nil
}

func (c *HTTPStripeClient) GetSubscription(
	ctx context.Context,
	config StripeConfig,
	subscriptionID string,
) (StripeSubscriptionRecord, error) {
	responseBody, err := c.doStripeRequest(
		ctx,
		config,
		http.MethodGet,
		"/v1/subscriptions/"+url.PathEscape(strings.TrimSpace(subscriptionID)),
		nil,
		"",
	)
	if err != nil {
		return StripeSubscriptionRecord{}, err
	}

	var payload struct {
		ID                 string `json:"id"`
		CustomerID         string `json:"customer"`
		Status             string `json:"status"`
		CurrentPeriodStart int64  `json:"current_period_start"`
		CurrentPeriodEnd   int64  `json:"current_period_end"`
		CancelAt           int64  `json:"cancel_at"`
		CanceledAt         int64  `json:"canceled_at"`
		Metadata           map[string]string `json:"metadata"`
		Items              struct {
			Data []struct {
				Price struct {
					ID string `json:"id"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return StripeSubscriptionRecord{}, fmt.Errorf("decode stripe subscription: %w", err)
	}

	priceID := ""
	if len(payload.Items.Data) > 0 {
		priceID = strings.TrimSpace(payload.Items.Data[0].Price.ID)
	}

	plan, _ := FindPlanByPriceID(priceID)
	record := NormalizeStripeSubscriptionRecord(StripeSubscriptionRecord{
		SubscriptionID:     payload.ID,
		CustomerID:         payload.CustomerID,
		StripePriceID:      priceID,
		Tier:               plan.Tier,
		Status:             StripeSubscriptionStatus(strings.ToLower(strings.TrimSpace(payload.Status))),
		CurrentPeriodStart: unixOrNow(payload.CurrentPeriodStart),
		CurrentPeriodEnd:   unixOrNow(payload.CurrentPeriodEnd),
		CancelAt:           unixPointer(payload.CancelAt),
		CanceledAt:         unixPointer(payload.CanceledAt),
		Metadata:           payload.Metadata,
		SyncedAt:           time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	})
	if err := ValidateStripeSubscriptionRecord(record); err != nil {
		return StripeSubscriptionRecord{}, err
	}

	return record, nil
}

func FindPlanByPriceID(priceID string) (Plan, error) {
	normalizedPriceID := strings.TrimSpace(priceID)
	for _, plan := range DefaultPlans() {
		if strings.TrimSpace(plan.StripePriceID) == normalizedPriceID {
			return plan, nil
		}
	}

	return Plan{}, fmt.Errorf("unknown stripe price id: %s", normalizedPriceID)
}

func (c *HTTPStripeClient) doStripeRequest(
	ctx context.Context,
	config StripeConfig,
	method string,
	path string,
	body io.Reader,
	contentType string,
) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("stripe client is required")
	}

	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		baseURL = DefaultStripeAPIBaseURL
	}

	request, err := http.NewRequestWithContext(
		ctx,
		method,
		strings.TrimRight(baseURL, "/")+path,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("build stripe request: %w", err)
	}
	request.SetBasicAuth(strings.TrimSpace(config.SecretKey), "")
	request.Header.Set("Accept", "application/json")
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("perform stripe request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read stripe response: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("stripe api status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return responseBody, nil
}

func unixOrNow(value int64) time.Time {
	if value <= 0 {
		return time.Now().UTC()
	}
	return time.Unix(value, 0).UTC()
}

func unixPointer(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	parsed := time.Unix(value, 0).UTC()
	return &parsed
}

func BuildCheckoutSessionPlaceholderRecord(
	request CheckoutRequest,
	priceID string,
	metadata map[string]string,
) StripeCheckoutSessionRecord {
	session := CheckoutSessionPlaceholder(request, priceID)
	return NormalizeStripeCheckoutSessionRecord(StripeCheckoutSessionRecord{
		SessionID:      session.SessionID,
		CustomerID:     "cus_placeholder",
		CustomerEmail:  request.CustomerEmail,
		SubscriptionID: "",
		Tier:           NormalizePlanTier(string(request.Tier)),
		StripePriceID:  priceID,
		Status:         StripeSessionStatusOpen,
		SuccessURL:     request.SuccessURL,
		CancelURL:      request.CancelURL,
		Metadata:       cloneStringMap(metadata),
		CreatedAt:      session.CreatedAt.UTC(),
		UpdatedAt:      session.CreatedAt.UTC(),
	})
}

func ParseStripeUnixString(raw string) int64 {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}
