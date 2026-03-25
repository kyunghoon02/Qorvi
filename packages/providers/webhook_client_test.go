package providers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestAlchemyWebhookClientListsAndReplacesAddresses(t *testing.T) {
	var (
		gotToken          string
		gotReplacePayload string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Alchemy-Token")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/webhook-addresses":
			switch r.URL.Query().Get("after") {
			case "":
				_, _ = w.Write([]byte(`{"data":["0xbbb","0xaaa"],"pagination":{"cursors":{"after":"cursor_1"},"total_count":3}}`))
			case "cursor_1":
				_, _ = w.Write([]byte(`{"data":["0xccc"],"pagination":{"cursors":{"after":""},"total_count":3}}`))
			default:
				http.Error(w, "unexpected cursor", http.StatusBadRequest)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/api/update-webhook-addresses":
			body, _ := io.ReadAll(r.Body)
			gotReplacePayload = string(body)
			_, _ = w.Write([]byte(`{}`))
		default:
			http.Error(w, "unexpected route", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewAlchemyWebhookClient(server.URL, "notify_token", server.Client())
	addresses, err := client.ListWebhookAddresses(t.Context(), "wh_123", 2)
	if err != nil {
		t.Fatalf("ListWebhookAddresses returned error: %v", err)
	}
	if !slices.Equal(addresses, []string{"0xaaa", "0xbbb", "0xccc"}) {
		t.Fatalf("unexpected addresses %#v", addresses)
	}
	if gotToken != "notify_token" {
		t.Fatalf("unexpected auth token %q", gotToken)
	}

	if err := client.ReplaceWebhookAddresses(t.Context(), "wh_123", []string{"0xbbb", "0xaaa"}); err != nil {
		t.Fatalf("ReplaceWebhookAddresses returned error: %v", err)
	}
	if gotReplacePayload != `{"addresses":["0xaaa","0xbbb"],"webhook_id":"wh_123"}` {
		t.Fatalf("unexpected replace payload %q", gotReplacePayload)
	}
}

func TestHeliusWebhookClientGetsAndReplacesAddresses(t *testing.T) {
	var gotUpdatePayload string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api-key") != "helius_key" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{
				"webhookID":"wh_live",
				"webhookURL":"https://flowintel.test/webhooks/helius",
				"transactionTypes":["ANY"],
				"accountAddresses":["So222","So111"],
				"webhookType":"enhanced",
				"authHeader":"Bearer test",
				"encoding":"jsonParsed",
				"txnStatus":"all",
				"active":true
			}`))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			gotUpdatePayload = string(body)
			_, _ = w.Write([]byte(`{}`))
		default:
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := NewHeliusWebhookClient(server.URL+"/v0", "helius_key", server.Client())
	record, err := client.GetWebhook(t.Context(), "wh_live")
	if err != nil {
		t.Fatalf("GetWebhook returned error: %v", err)
	}
	if !slices.Equal(record.AccountAddresses, []string{"So111", "So222"}) {
		t.Fatalf("unexpected webhook record %#v", record)
	}

	if err := client.ReplaceWebhookAddresses(t.Context(), "wh_live", []string{"So333", "So111"}); err != nil {
		t.Fatalf("ReplaceWebhookAddresses returned error: %v", err)
	}
	expected := `{"accountAddresses":["So111","So333"],"authHeader":"Bearer test","encoding":"jsonParsed","transactionTypes":["ANY"],"txnStatus":"all","webhookType":"enhanced","webhookURL":"https://flowintel.test/webhooks/helius"}`
	if gotUpdatePayload != expected {
		t.Fatalf("unexpected update payload %q", gotUpdatePayload)
	}
}
