package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAlchemyAddressActivityWebhookRouteAcceptsValidPayload(t *testing.T) {
	t.Parallel()

	srv := New()
	payload := AlchemyAddressActivityWebhook{
		WebhookID: "wh_k63lg72rxda78gce",
		ID:        "whevt_vq499kv7elmlbp2v",
		CreatedAt: "2026-03-20T00:00:00Z",
		Type:      "ADDRESS_ACTIVITY",
		Event: AlchemyAddressActivityWebhookEvent{
			Network: "ETH_MAINNET",
			Activity: []AlchemyAddressActivityRecord{
				{
					BlockNum:    "0xdf34a3",
					Hash:        "0x7a4a39da2a3fa1fc2ef88fd1eaea070286ed2aba21e0419dcfb6d5c5d9f02a72",
					FromAddress: "0x503828976d22510aad0201ac7ec88293211d23da",
					ToAddress:   "0xbe3f4b43db5eb49d1f48f53443b9abce45da3b79",
					Category:    "token",
					Asset:       "USDC",
				},
			},
		},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/providers/alchemy/address-activity", bytes.NewReader(raw))
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
	if body.Data.Provider != "alchemy" {
		t.Fatalf("unexpected provider %q", body.Data.Provider)
	}
	if body.Data.EventKind != "address_activity" {
		t.Fatalf("unexpected event kind %q", body.Data.EventKind)
	}
	if body.Data.AcceptedCount != 1 {
		t.Fatalf("expected accepted count 1, got %d", body.Data.AcceptedCount)
	}
}

func TestAlchemyAddressActivityWebhookRouteRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	srv := New()
	raw := []byte(`{"webhookId":"wh_test","id":"evt_test","type":"ADDRESS_ACTIVITY","event":{"network":"ETH_MAINNET","activity":[]}}`)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/providers/alchemy/address-activity", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var body Envelope[any]
	decode(t, rr.Body.Bytes(), &body)
	if body.Error == nil || body.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("expected invalid argument error, got %#v", body.Error)
	}
}
