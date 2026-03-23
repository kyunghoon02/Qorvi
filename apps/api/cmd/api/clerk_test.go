package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/config"
	sharedconfig "github.com/whalegraph/whalegraph/packages/config"
)

func TestBuildClerkVerifierVerifiesSignedSessionToken(t *testing.T) {
	t.Parallel()

	privateKey := mustRSAKey(t)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				jwkFromPublicKey("test-key", &privateKey.PublicKey),
			},
		})
	}))
	defer jwksServer.Close()

	verifier, err := buildClerkVerifier(config.Config{
		API: sharedconfig.APIEnv{
			AppBaseURL: "https://app.whalegraph.com",
			ClerkVerification: sharedconfig.ClerkVerificationConfig{
				IssuerURL:        "https://clerk.whalegraph.com",
				JWKSURL:          jwksServer.URL,
				Audience:         "whalegraph",
				ClockSkewSeconds: 60,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected verifier, got %v", err)
	}

	token := mustSignedJWT(t, privateKey, "test-key", map[string]any{
		"iss":   "https://clerk.whalegraph.com",
		"sub":   "user_123",
		"sid":   "sess_123",
		"rol":   "org:admin",
		"email": "admin@whalegraph.com",
		"azp":   "https://app.whalegraph.com",
		"aud":   "whalegraph",
		"exp":   time.Now().Add(10 * time.Minute).Unix(),
		"nbf":   time.Now().Add(-1 * time.Minute).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	principal, err := verifier.Verify(req)
	if err != nil {
		t.Fatalf("expected principal, got %v", err)
	}

	if principal.UserID != "user_123" {
		t.Fatalf("unexpected user id %q", principal.UserID)
	}
	if principal.SessionID != "sess_123" {
		t.Fatalf("unexpected session id %q", principal.SessionID)
	}
	if principal.Role != "admin" {
		t.Fatalf("unexpected role %q", principal.Role)
	}
	if principal.Email != "admin@whalegraph.com" {
		t.Fatalf("unexpected email %q", principal.Email)
	}
}

func TestBuildClerkVerifierRejectsWrongAuthorizedParty(t *testing.T) {
	t.Parallel()

	privateKey := mustRSAKey(t)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				jwkFromPublicKey("test-key", &privateKey.PublicKey),
			},
		})
	}))
	defer jwksServer.Close()

	verifier, err := buildClerkVerifier(config.Config{
		API: sharedconfig.APIEnv{
			AppBaseURL: "https://app.whalegraph.com",
			ClerkVerification: sharedconfig.ClerkVerificationConfig{
				IssuerURL: "https://clerk.whalegraph.com",
				JWKSURL:   jwksServer.URL,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected verifier, got %v", err)
	}

	token := mustSignedJWT(t, privateKey, "test-key", map[string]any{
		"iss": "https://clerk.whalegraph.com",
		"sub": "user_123",
		"sid": "sess_123",
		"rol": "admin",
		"azp": "https://malicious.example.com",
		"exp": time.Now().Add(10 * time.Minute).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	if _, err := verifier.Verify(req); err == nil {
		t.Fatal("expected verification to fail")
	}
}

func TestBuildClerkVerifierFallsBackToLegacyHeadersInDevelopment(t *testing.T) {
	t.Parallel()

	verifier, err := buildClerkVerifier(config.Config{
		API: sharedconfig.APIEnv{
			NodeEnv:    "development",
			AppBaseURL: "http://localhost:3000",
			ClerkVerification: sharedconfig.ClerkVerificationConfig{
				IssuerURL: "https://example.clerk.accounts.dev",
				JWKSURL:   "https://example.clerk.accounts.dev/.well-known/jwks.json",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected legacy verifier fallback, got %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)
	req.Header.Set("X-Clerk-User-Id", "admin_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")

	principal, err := verifier.Verify(req)
	if err != nil {
		t.Fatalf("expected legacy header verification, got %v", err)
	}
	if principal.UserID != "admin_1" || principal.Role != "admin" {
		t.Fatalf("unexpected principal %#v", principal)
	}
}

func mustRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	return key
}

func mustSignedJWT(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()

	header := map[string]any{
		"alg": "RS256",
		"kid": kid,
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}

	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
	sum := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func jwkFromPublicKey(kid string, pub *rsa.PublicKey) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"kid": kid,
		"use": "sig",
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   encodeExponent(pub.E),
	}
}

func encodeExponent(value int) string {
	if value <= 0 {
		return ""
	}

	buf := big.NewInt(int64(value)).Bytes()
	return base64.RawURLEncoding.EncodeToString(buf)
}
