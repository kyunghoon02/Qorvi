package server

import (
	"context"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flowintel/flowintel/apps/api/internal/auth"
	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/apps/api/internal/service"
	"github.com/flowintel/flowintel/packages/billing"
	"github.com/flowintel/flowintel/packages/domain"
)

func TestAlertRuleRoutesRequireAuthAndPlan(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		AlertRules:    service.NewAlertRuleService(repository.NewInMemoryAlertRuleRepository()),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/alert-rules", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}

	forbidden := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/alert-rules", nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	srv.Handler().ServeHTTP(forbidden, req)

	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", forbidden.Code)
	}
}

func TestAlertRuleRoutesSupportCRUDAndEventsForProOwner(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		AlertRules:    service.NewAlertRuleService(repository.NewInMemoryAlertRuleRepository()),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	create := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/alert-rules", bytes.NewBufferString(`{
	  "name":"Shadow Exit Rule",
	  "ruleType":"watchlist_signal",
	  "isEnabled":true,
	  "cooldownSeconds":1800,
	  "definition":{"watchlistId":"watch_1","signalTypes":["shadow_exit"],"minimumSeverity":"high","renotifyOnSeverityIncrease":true},
	  "tags":["ops","ops"],
	  "notes":"watch closely"
	}`))
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(create, req)

	if create.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", create.Code)
	}

	var created Envelope[service.AlertRuleDetail]
	decode(t, create.Body.Bytes(), &created)
	if created.Data.Name != "Shadow Exit Rule" {
		t.Fatalf("unexpected alert rule name %s", created.Data.Name)
	}
	if len(created.Data.Tags) != 1 {
		t.Fatalf("expected normalized tags, got %v", created.Data.Tags)
	}

	list := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/alert-rules", nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(list, req)

	if list.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", list.Code)
	}

	var collection Envelope[service.AlertRuleCollection]
	decode(t, list.Body.Bytes(), &collection)
	if len(collection.Data.Items) != 1 {
		t.Fatalf("expected one alert rule, got %d", len(collection.Data.Items))
	}

	update := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/v1/alert-rules/"+created.Data.ID, bytes.NewBufferString(`{
	  "name":"Updated Rule",
	  "ruleType":"watchlist_signal",
	  "isEnabled":false,
	  "cooldownSeconds":900,
	  "definition":{"watchlistId":"watch_1","signalTypes":["first_connection"],"minimumSeverity":"medium","renotifyOnSeverityIncrease":false},
	  "tags":["updated"],
	  "notes":"updated note"
	}`))
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(update, req)

	if update.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", update.Code)
	}

	var updated Envelope[service.AlertRuleDetail]
	decode(t, update.Body.Bytes(), &updated)
	if updated.Data.IsEnabled {
		t.Fatal("expected updated alert rule to be disabled")
	}
	if updated.Data.Definition.SignalTypes[0] != "first_connection" {
		t.Fatalf("unexpected signal type %v", updated.Data.Definition.SignalTypes)
	}

	events := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/alert-rules/"+created.Data.ID+"/events", nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(events, req)

	if events.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", events.Code)
	}

	deleteRule := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/v1/alert-rules/"+created.Data.ID, nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(deleteRule, req)

	if deleteRule.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", deleteRule.Code)
	}

	var deleted Envelope[service.AlertRuleMutationResult]
	decode(t, deleteRule.Body.Bytes(), &deleted)
	if !deleted.Data.Deleted {
		t.Fatal("expected deleted true")
	}
}

func TestAlertRuleRoutesResolvePersistedBillingPlan(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryBillingRepository()
	_, err := repo.UpsertBillingAccount(context.Background(), repository.BillingAccount{
		OwnerUserID: "user_123",
		CurrentTier: domain.PlanPro,
		Status:      "active",
	})
	if err != nil {
		t.Fatalf("upsert billing account: %v", err)
	}

	srv := NewWithDependencies(Dependencies{
		AlertRules:    service.NewAlertRuleService(repository.NewInMemoryAlertRuleRepository()),
		Billing:       service.NewBillingService(repo, billing.StripeConfig{}),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	create := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/alert-rules", bytes.NewBufferString(`{
	  "name":"Persisted plan rule",
	  "ruleType":"watchlist_signal",
	  "isEnabled":true,
	  "cooldownSeconds":1800,
	  "definition":{"watchlistId":"watch_1","signalTypes":["shadow_exit"],"minimumSeverity":"high","renotifyOnSeverityIncrease":true},
	  "tags":["ops"],
	  "notes":"watch closely"
	}`))
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	srv.Handler().ServeHTTP(create, req)

	if create.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", create.Code)
	}
}
