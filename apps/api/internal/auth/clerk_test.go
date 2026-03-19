package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sharedconfig "github.com/whalegraph/whalegraph/packages/config"
)

func TestHeaderClerkVerifierRequiresIdentityHeaders(t *testing.T) {
	t.Parallel()

	verifier := NewHeaderClerkVerifier()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)

	_, err := verifier.Verify(req)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected unauthenticated error, got %v", err)
	}
}

func TestHeaderClerkVerifierParsesRole(t *testing.T) {
	t.Parallel()

	verifier := NewHeaderClerkVerifier()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "Admin")

	principal, err := verifier.Verify(req)
	if err != nil {
		t.Fatalf("expected principal, got %v", err)
	}
	if principal.Role != "admin" {
		t.Fatalf("expected admin role, got %s", principal.Role)
	}
}

func TestHeaderClerkVerifierVerifiesBearerToken(t *testing.T) {
	t.Parallel()

	privateKey := mustRSAKey(t)
	jwksServer, jwksURL := newJWKS(t, privateKey, "clerk-test-key")
	defer jwksServer.Close()

	verifier := NewHeaderClerkVerifierWithConfig(sharedconfig.ClerkVerificationConfig{
		IssuerURL:        "https://example.clerk.accounts.dev",
		JWKSURL:          jwksURL,
		Audience:         "whalegraph",
		ClockSkewSeconds: 60,
	})

	token := signClerkToken(t, privateKey, map[string]any{
		"sub":   "user_123",
		"sid":   "session_123",
		"role":  "admin",
		"iss":   "https://example.clerk.accounts.dev",
		"aud":   "whalegraph",
		"exp":   time.Now().UTC().Add(time.Minute).Unix(),
		"nbf":   time.Now().UTC().Add(-time.Minute).Unix(),
		"email": "user@example.com",
	}, "clerk-test-key")

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	principal, err := verifier.Verify(req)
	if err != nil {
		t.Fatalf("expected bearer token to verify, got %v", err)
	}
	if principal.UserID != "user_123" {
		t.Fatalf("unexpected user id %q", principal.UserID)
	}
	if principal.SessionID != "session_123" {
		t.Fatalf("unexpected session id %q", principal.SessionID)
	}
	if principal.Role != "admin" {
		t.Fatalf("unexpected role %q", principal.Role)
	}
	if principal.Email != "user@example.com" {
		t.Fatalf("unexpected email %q", principal.Email)
	}
}

func TestHeaderClerkVerifierPrefersBearerTokenOverLegacyHeaders(t *testing.T) {
	t.Parallel()

	privateKey := mustRSAKey(t)
	jwksServer, jwksURL := newJWKS(t, privateKey, "clerk-test-key")
	defer jwksServer.Close()

	verifier := NewHeaderClerkVerifierWithConfig(sharedconfig.ClerkVerificationConfig{
		IssuerURL:        "https://example.clerk.accounts.dev",
		JWKSURL:          jwksURL,
		Audience:         "whalegraph",
		ClockSkewSeconds: 60,
	})

	token := signClerkToken(t, privateKey, map[string]any{
		"sub":  "user_123",
		"sid":  "session_123",
		"role": "user",
		"iss":  "https://example.clerk.accounts.dev",
		"aud":  "whalegraph",
		"exp":  time.Now().UTC().Add(time.Minute).Unix(),
		"nbf":  time.Now().UTC().Add(-time.Minute).Unix(),
	}, "clerk-test-key")

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Clerk-User-Id", "legacy-user")
	req.Header.Set("X-Clerk-Session-Id", "legacy-session")
	req.Header.Set("X-Clerk-Role", "admin")

	principal, err := verifier.Verify(req)
	if err != nil {
		t.Fatalf("expected bearer token to win, got %v", err)
	}
	if principal.UserID != "user_123" || principal.SessionID != "session_123" || principal.Role != "user" {
		t.Fatalf("unexpected principal from bearer token: %+v", principal)
	}
}

func TestHeaderClerkVerifierRejectsInvalidBearerEvenWithLegacyHeaders(t *testing.T) {
	t.Parallel()

	privateKey := mustRSAKey(t)
	jwksServer, jwksURL := newJWKS(t, privateKey, "clerk-test-key")
	defer jwksServer.Close()

	verifier := NewHeaderClerkVerifierWithConfig(sharedconfig.ClerkVerificationConfig{
		IssuerURL:        "https://example.clerk.accounts.dev",
		JWKSURL:          jwksURL,
		Audience:         "whalegraph",
		ClockSkewSeconds: 60,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "admin")

	_, err := verifier.Verify(req)
	if err == nil {
		t.Fatal("expected invalid bearer token to fail")
	}
}

func TestRequireClerkRoleBlocksUnauthorizedRequests(t *testing.T) {
	t.Parallel()

	responder := testResponder{}
	handler := RequireClerkRole(NewHeaderClerkVerifier(), responder, "admin")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be reached")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

type testResponder struct{}

func (testResponder) Unauthorized(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
}

func (testResponder) Forbidden(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusForbidden)
}

func mustRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	return key
}

func newJWKS(t *testing.T, privateKey *rsa.PrivateKey, kid string) (*httptest.Server, string) {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
				},
			},
		})
	})

	server := httptest.NewServer(handler)
	return server, server.URL
}

func signClerkToken(t *testing.T, privateKey *rsa.PrivateKey, claims map[string]any, kid string) string {
	t.Helper()

	headerJSON, err := json.Marshal(map[string]any{
		"alg": "RS256",
		"kid": kid,
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := header + "." + payload
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}
