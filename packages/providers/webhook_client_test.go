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
		gotCreatePayload  string
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
		case r.Method == http.MethodGet && r.URL.Path == "/api/team-webhooks":
			_, _ = w.Write([]byte(`[{"data":[{"id":"wh_existing","network":"ETH_MAINNET","webhook_type":"ADDRESS_ACTIVITY","webhook_url":"https://qorvi.test/v1/webhooks/providers/alchemy/address-activity","is_active":true,"addresses":["0xaaa"]}]}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/create-webhook":
			body, _ := io.ReadAll(r.Body)
			gotCreatePayload = string(body)
			_, _ = w.Write([]byte(`{"id":"wh_created","network":"ETH_MAINNET","webhook_type":"ADDRESS_ACTIVITY","webhook_url":"https://qorvi.test/v1/webhooks/providers/alchemy/address-activity","is_active":true,"addresses":["0xabc"]}`))
		default:
			http.Error(w, "unexpected route", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewAlchemyWebhookClient(server.URL, "notify_token", "ETH_MAINNET", server.Client())
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

	webhooks, err := client.ListWebhooks(t.Context())
	if err != nil {
		t.Fatalf("ListWebhooks returned error: %v", err)
	}
	if len(webhooks) != 1 || webhooks[0].ID != "wh_existing" {
		t.Fatalf("unexpected webhooks %#v", webhooks)
	}

	record, err := client.CreateAddressActivityWebhook(
		t.Context(),
		"https://qorvi.test/v1/webhooks/providers/alchemy/address-activity",
		[]string{"0xabc"},
	)
	if err != nil {
		t.Fatalf("CreateAddressActivityWebhook returned error: %v", err)
	}
	if record.ID != "wh_created" {
		t.Fatalf("unexpected created record %#v", record)
	}
	expectedCreate := `{"addresses":["0xabc"],"network":"ETH_MAINNET","webhook_type":"ADDRESS_ACTIVITY","webhook_url":"https://qorvi.test/v1/webhooks/providers/alchemy/address-activity"}`
	if gotCreatePayload != expectedCreate {
		t.Fatalf("unexpected create payload %q", gotCreatePayload)
	}

	ensure, err := client.EnsureWebhookAddresses(
		t.Context(),
		"",
		"https://qorvi.test/v1/webhooks/providers/alchemy/address-activity",
		[]string{"0xbbb", "0xaaa"},
	)
	if err != nil {
		t.Fatalf("EnsureWebhookAddresses returned error: %v", err)
	}
	if ensure.WebhookID != "wh_existing" || !ensure.Discovered || ensure.Created {
		t.Fatalf("unexpected ensure result %#v", ensure)
	}
}

func TestHeliusWebhookClientGetsAndReplacesAddresses(t *testing.T) {
	var (
		gotUpdatePayload string
		gotCreatePayload string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api-key") != "helius_key" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && (r.URL.Path == "/v0/webhooks/wh_live" || r.URL.Path == "/v0/webhooks/wh_existing"):
			_, _ = w.Write([]byte(`{
				"webhookID":"wh_live",
				"webhookURL":"https://qorvi.test/webhooks/helius",
				"transactionTypes":["ANY"],
				"accountAddresses":["So222","So111"],
				"webhookType":"enhanced",
				"authHeader":"Bearer test",
				"encoding":"jsonParsed",
				"txnStatus":"all",
				"active":true
			}`))
		case r.Method == http.MethodPut && (r.URL.Path == "/v0/webhooks/wh_live" || r.URL.Path == "/v0/webhooks/wh_existing"):
			body, _ := io.ReadAll(r.Body)
			gotUpdatePayload = string(body)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v0/webhooks":
			_, _ = w.Write([]byte(`[
				{
					"webhookID":"wh_existing",
					"webhookURL":"https://qorvi.test/webhooks/helius",
					"transactionTypes":["ANY"],
					"accountAddresses":["So222","So111"],
					"webhookType":"enhanced",
					"authHeader":"Bearer test",
					"active":true
				}
			]`))
		case r.Method == http.MethodPost && r.URL.Path == "/v0/webhooks":
			body, _ := io.ReadAll(r.Body)
			gotCreatePayload = string(body)
			_, _ = w.Write([]byte(`{
				"webhookID":"wh_created",
				"webhookURL":"https://qorvi.test/webhooks/helius",
				"transactionTypes":["ANY"],
				"accountAddresses":["So111"],
				"webhookType":"enhanced",
				"authHeader":"Bearer test",
				"active":true
			}`))
		default:
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := NewHeliusWebhookClient(server.URL+"/v0", "helius_key", "Bearer test", server.Client())
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
	expected := `{"accountAddresses":["So111","So333"],"authHeader":"Bearer test","encoding":"jsonParsed","transactionTypes":["ANY"],"txnStatus":"all","webhookType":"enhanced","webhookURL":"https://qorvi.test/webhooks/helius"}`
	if gotUpdatePayload != expected {
		t.Fatalf("unexpected update payload %q", gotUpdatePayload)
	}

	webhooks, err := client.ListWebhooks(t.Context())
	if err != nil {
		t.Fatalf("ListWebhooks returned error: %v", err)
	}
	if len(webhooks) != 1 || webhooks[0].WebhookID != "wh_existing" {
		t.Fatalf("unexpected webhooks %#v", webhooks)
	}

	created, err := client.CreateWebhook(t.Context(), "https://qorvi.test/webhooks/helius", []string{"So111"})
	if err != nil {
		t.Fatalf("CreateWebhook returned error: %v", err)
	}
	if created.WebhookID != "wh_created" {
		t.Fatalf("unexpected created webhook %#v", created)
	}
	expectedCreate := `{"accountAddresses":["So111"],"authHeader":"Bearer test","transactionTypes":["ANY"],"webhookType":"enhanced","webhookURL":"https://qorvi.test/webhooks/helius"}`
	if gotCreatePayload != expectedCreate {
		t.Fatalf("unexpected create payload %q", gotCreatePayload)
	}

	ensure, err := client.EnsureWebhookAddresses(t.Context(), "", "https://qorvi.test/webhooks/helius", []string{"So111", "So333"})
	if err != nil {
		t.Fatalf("EnsureWebhookAddresses returned error: %v", err)
	}
	if ensure.WebhookID != "wh_existing" || !ensure.Discovered || ensure.Created {
		t.Fatalf("unexpected ensure result %#v", ensure)
	}
}
