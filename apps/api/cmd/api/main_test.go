package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qorvi/qorvi/apps/api/internal/config"
)

func TestLoadRuntimeConfigFallsBackWhenEnvIsMissing(t *testing.T) {
	t.Setenv("APP_BASE_URL", "")
	t.Setenv("API_HOST", "")
	t.Setenv("API_PORT", "")
	t.Setenv("POSTGRES_URL", "")
	t.Setenv("NEO4J_URL", "")
	t.Setenv("NEO4J_USERNAME", "")
	t.Setenv("NEO4J_PASSWORD", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("AUTH_PROVIDER", "")
	t.Setenv("AUTH_SECRET", "")
	t.Setenv("CLERK_SECRET_KEY", "")
	t.Setenv("CLERK_ISSUER_URL", "")
	t.Setenv("CLERK_JWKS_URL", "")
	t.Setenv("NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY", "")

	cfg, minimal := loadRuntimeConfig()
	if !minimal {
		t.Fatal("expected minimal fallback mode")
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != "3000" {
		t.Fatalf("unexpected fallback config: %#v", cfg)
	}
}

func TestBuildClerkVerifierOrFallbackUsesHeaderVerifierInMinimalMode(t *testing.T) {
	t.Setenv("APP_BASE_URL", "")
	t.Setenv("CLERK_ISSUER_URL", "")
	t.Setenv("CLERK_JWKS_URL", "")

	verifier := buildClerkVerifierOrFallback(config.Config{}, true)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)
	req.Header.Set("X-Clerk-User-Id", "admin_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")

	principal, err := verifier.Verify(req)
	if err != nil {
		t.Fatalf("expected header verifier to accept legacy headers: %v", err)
	}
	if principal.UserID != "admin_1" || principal.Role != "admin" {
		t.Fatalf("unexpected principal %#v", principal)
	}
}
