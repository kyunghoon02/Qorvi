package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeliusAddressActivityWebhookRouteAcceptsValidPayload(t *testing.T) {
	t.Parallel()

	srv := New()
	raw := []byte(`[
		{
			"description":"swap",
			"type":"SWAP",
			"source":"JUPITER",
			"fee":5000,
			"feePayer":"So11111111111111111111111111111111111111112",
			"signature":"5N7fjLzsJ8T5hEj6jKn8kDcQa1hUNY7bH6mvYzu3PV6P9ToYeR3dStv8smh3QvYzSxC25f6GCb79iVup1QdP4n2g",
			"slot":123,
			"timestamp":1774828800,
			"nativeTransfers":[{"fromUserAccount":"So11111111111111111111111111111111111111112","toUserAccount":"So22222222222222222222222222222222222222222","amount":1000}]
		}
	]`)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/providers/helius/address-activity", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}

	var body Envelope[ProviderWebhookAcceptancePayload]
	decode(t, rr.Body.Bytes(), &body)
	if !body.Success {
		t.Fatal("expected success response")
	}
	if body.Data.Provider != "helius" {
		t.Fatalf("unexpected provider %q", body.Data.Provider)
	}
	if body.Data.EventKind != "webhook_batch" {
		t.Fatalf("unexpected event kind %q", body.Data.EventKind)
	}
}

func TestHeliusAddressActivityWebhookRouteRequiresConfiguredAuthHeader(t *testing.T) {
	t.Setenv("QORVI_PROVIDER_WEBHOOK_AUTH_HEADER", "Bearer qorvi-webhook-secret")
	srv := New()
	raw := []byte(`[
		{
			"signature":"5N7fjLzsJ8T5hEj6jKn8kDcQa1hUNY7bH6mvYzu3PV6P9ToYeR3dStv8smh3QvYzSxC25f6GCb79iVup1QdP4n2g",
			"feePayer":"So11111111111111111111111111111111111111112"
		}
	]`)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/providers/helius/address-activity", bytes.NewReader(raw))

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}
