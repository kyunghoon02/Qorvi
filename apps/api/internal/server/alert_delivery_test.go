package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flowintel/flowintel/apps/api/internal/auth"
	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/apps/api/internal/service"
	"github.com/flowintel/flowintel/packages/domain"
)

func TestAlertDeliveryRoutesRequireAuthAndPlan(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		AlertDelivery: service.NewAlertDeliveryService(repository.NewInMemoryAlertDeliveryRepository()),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}

	forbidden := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/alerts", nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	srv.Handler().ServeHTTP(forbidden, req)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", forbidden.Code)
	}
}

func TestAlertDeliveryRoutesSupportInboxAndChannelCrudForProOwner(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryAlertDeliveryRepository()
	repo.SeedAlertEvent(domain.AlertEvent{
		ID:          "evt_1",
		AlertRuleID: "rule_1",
		OwnerUserID: "user_123",
		EventKey:    "shadow_exit:evm:0x123",
		DedupKey:    "dedup_1",
		SignalType:  "shadow_exit",
		Severity:    domain.AlertSeverityHigh,
		Payload:     map[string]any{"score_value": 88},
		ObservedAt:  time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC),
		CreatedAt:   time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC),
	})

	srv := NewWithDependencies(Dependencies{
		AlertDelivery: service.NewAlertDeliveryService(repo),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	inbox := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/alerts?signalType=shadow_exit&limit=10", nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(inbox, req)
	if inbox.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", inbox.Code)
	}

	var inboxBody Envelope[service.AlertInboxCollection]
	decode(t, inbox.Body.Bytes(), &inboxBody)
	if len(inboxBody.Data.Items) != 1 {
		t.Fatalf("expected one inbox item, got %d", len(inboxBody.Data.Items))
	}

	create := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/alert-delivery-channels", bytes.NewBufferString(`{
	  "label":"Ops Email",
	  "channelType":"email",
	  "target":"ops@example.com",
	  "metadata":{"format":"compact"},
	  "isEnabled":true
	}`))
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(create, req)
	if create.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", create.Code)
	}

	var created Envelope[service.AlertDeliveryChannelSummary]
	decode(t, create.Body.Bytes(), &created)
	if created.Data.ChannelType != "email" {
		t.Fatalf("unexpected channel type %q", created.Data.ChannelType)
	}

	list := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/alert-delivery-channels", nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(list, req)
	if list.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", list.Code)
	}

	update := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/v1/alert-delivery-channels/"+created.Data.ID, bytes.NewBufferString(`{
	  "label":"Ops Email Updated",
	  "target":"ops+alerts@example.com",
	  "metadata":{"format":"long"},
	  "isEnabled":false
	}`))
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(update, req)
	if update.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", update.Code)
	}

	deleteChannel := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/v1/alert-delivery-channels/"+created.Data.ID, nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(deleteChannel, req)
	if deleteChannel.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", deleteChannel.Code)
	}
}
