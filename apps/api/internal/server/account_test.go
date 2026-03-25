package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flowintel/flowintel/apps/api/internal/auth"
	"github.com/flowintel/flowintel/apps/api/internal/service"
)

func TestAccountRouteRequiresAuth(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{ClerkVerifier: auth.NewHeaderClerkVerifier()})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/account", nil)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestAccountRouteReturnsEntitlements(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{ClerkVerifier: auth.NewHeaderClerkVerifier()})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/account", nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[service.AccountResponse]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatal("expected success response")
	}

	if body.Data.Plan.Tier != "pro" {
		t.Fatalf("expected pro plan, got %s", body.Data.Plan.Tier)
	}
	if len(body.Data.Entitlements) == 0 {
		t.Fatal("expected entitlements")
	}
}

func TestAccountEntitlementsAliasRouteReturnsSamePayload(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{ClerkVerifier: auth.NewHeaderClerkVerifier()})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/account/entitlements", nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "team")

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[service.AccountResponse]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatal("expected success response")
	}
	if body.Data.Plan.EnabledFeatureCount == 0 {
		t.Fatal("expected enabled feature count in plan summary")
	}
}
