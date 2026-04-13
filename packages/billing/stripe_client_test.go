package billing

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestHTTPStripeClientCreateCheckoutSession(t *testing.T) {
	t.Parallel()

	client := NewHTTPStripeClient(&http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/checkout/sessions" {
				t.Fatalf("unexpected path %s", request.URL.Path)
			}
			if got := request.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
				t.Fatalf("unexpected content type %q", got)
			}

			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			payload := string(body)
			if !strings.Contains(payload, "line_items%5B0%5D%5Bprice%5D=price_pro_live") {
				t.Fatalf("expected stripe price in payload, got %s", payload)
			}
			if !strings.Contains(payload, "metadata%5Bowner_user_id%5D=user_123") {
				t.Fatalf("expected metadata in payload, got %s", payload)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id":"cs_live_123",
					"url":"https://checkout.stripe.com/c/pay/cs_live_123",
					"customer":"cus_live_123",
					"subscription":"sub_live_123",
					"created":1774224000,
					"metadata":{"owner_user_id":"user_123","target_tier":"pro"}
				}`)),
				Header: make(http.Header),
			}, nil
		}),
	})

	record, err := client.CreateCheckoutSession(context.Background(), StripeConfig{
		BaseURL:   "https://stripe.test",
		SecretKey: "sk_live_test",
	}, StripeCheckoutSessionCreateRequest{
		OwnerUserID:   "user_123",
		Tier:          domain.PlanPro,
		CustomerEmail: "ops@qorvi.test",
		SuccessURL:    "https://qorvi.test/account?checkout=success",
		CancelURL:     "https://qorvi.test/account?checkout=cancel",
		StripePriceID: "price_pro_live",
	})
	if err != nil {
		t.Fatalf("CreateCheckoutSession returned error: %v", err)
	}

	if record.SessionID != "cs_live_123" {
		t.Fatalf("unexpected session id %q", record.SessionID)
	}
	if record.CustomerID != "cus_live_123" {
		t.Fatalf("unexpected customer id %q", record.CustomerID)
	}
	if record.SubscriptionID != "sub_live_123" {
		t.Fatalf("unexpected subscription id %q", record.SubscriptionID)
	}
}

func TestHTTPStripeClientGetSubscription(t *testing.T) {
	t.Parallel()

	client := NewHTTPStripeClient(&http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/subscriptions/sub_live_123" {
				t.Fatalf("unexpected path %s", request.URL.Path)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id":"sub_live_123",
					"customer":"cus_live_123",
					"status":"active",
					"current_period_start":1774224000,
					"current_period_end":1776816000,
					"items":{"data":[{"price":{"id":"price_pro_placeholder"}}]},
					"metadata":{"source":"stripe"}
				}`)),
				Header: make(http.Header),
			}, nil
		}),
	})

	record, err := client.GetSubscription(context.Background(), StripeConfig{
		BaseURL:   "https://stripe.test",
		SecretKey: "sk_live_test",
	}, "sub_live_123")
	if err != nil {
		t.Fatalf("GetSubscription returned error: %v", err)
	}

	if record.SubscriptionID != "sub_live_123" {
		t.Fatalf("unexpected subscription id %q", record.SubscriptionID)
	}
	if record.Tier != domain.PlanPro {
		t.Fatalf("expected pro tier, got %q", record.Tier)
	}
	if record.Status != StripeSubscriptionStatusActive {
		t.Fatalf("unexpected status %q", record.Status)
	}
	if record.CurrentPeriodEnd.Before(time.Unix(1776816000, 0).UTC()) {
		t.Fatalf("unexpected period end %#v", record.CurrentPeriodEnd)
	}
}
