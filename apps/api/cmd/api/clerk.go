package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/auth"
	"github.com/whalegraph/whalegraph/apps/api/internal/config"
)

const clerkSessionCookieName = "__session"

type clerkJWTVerifier struct {
	issuerURL     string
	jwksURL       string
	audience      string
	allowedOrigin string
	clockSkew     time.Duration
	client        *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

type clerkJWTHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Type      string `json:"typ"`
}

type clerkJWTClaims struct {
	Issuer    string         `json:"iss"`
	Subject   string         `json:"sub"`
	Session   string         `json:"sid"`
	Role      string         `json:"rol"`
	OrgRole   string         `json:"org_role"`
	Email     string         `json:"email"`
	AZP       string         `json:"azp"`
	Expiry    int64          `json:"exp"`
	NotBefore int64          `json:"nbf"`
	Audience  clerkAudiences `json:"aud"`
}

type clerkAudiences []string

func (a *clerkAudiences) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*a = nil
		return nil
	}

	if len(trimmed) > 0 && trimmed[0] == '"' {
		var single string
		if err := json.Unmarshal(data, &single); err != nil {
			return err
		}
		*a = []string{single}
		return nil
	}

	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*a = append((*a)[:0], many...)
	return nil
}

func buildClerkVerifier(cfg config.Config) (auth.ClerkVerifier, error) {
	if shouldUseLegacyClerkVerifier(cfg) {
		return auth.NewHeaderClerkVerifier(), nil
	}

	verifier := &clerkJWTVerifier{
		issuerURL:     strings.TrimSpace(cfg.API.ClerkVerification.IssuerURL),
		jwksURL:       strings.TrimSpace(cfg.API.ClerkVerification.JWKSURL),
		audience:      strings.TrimSpace(cfg.API.ClerkVerification.Audience),
		allowedOrigin: strings.TrimSpace(cfg.API.AppBaseURL),
		clockSkew:     time.Duration(cfg.API.ClerkVerification.ClockSkewSeconds) * time.Second,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		keys: make(map[string]*rsa.PublicKey),
	}

	if err := verifier.refreshJWKS(); err != nil {
		if strings.EqualFold(strings.TrimSpace(cfg.API.NodeEnv), "development") {
			return auth.NewHeaderClerkVerifier(), nil
		}
		return nil, err
	}

	return verifier, nil
}

func shouldUseLegacyClerkVerifier(cfg config.Config) bool {
	if !strings.EqualFold(strings.TrimSpace(cfg.API.NodeEnv), "development") {
		return false
	}

	issuerURL := strings.TrimSpace(cfg.API.ClerkVerification.IssuerURL)
	jwksURL := strings.TrimSpace(cfg.API.ClerkVerification.JWKSURL)

	return strings.Contains(issuerURL, "example.clerk.accounts.dev") ||
		strings.Contains(jwksURL, "example.clerk.accounts.dev")
}

func (v *clerkJWTVerifier) Verify(r *http.Request) (auth.ClerkPrincipal, error) {
	token, err := extractClerkSessionToken(r)
	if err != nil {
		return auth.ClerkPrincipal{}, auth.ErrUnauthenticated
	}

	header, signingInput, signature, claims, err := parseClerkJWT(token)
	if err != nil {
		return auth.ClerkPrincipal{}, auth.ErrUnauthenticated
	}

	if header.Algorithm != "RS256" || header.KeyID == "" {
		return auth.ClerkPrincipal{}, auth.ErrUnauthenticated
	}

	key, err := v.publicKey(header.KeyID)
	if err != nil {
		return auth.ClerkPrincipal{}, auth.ErrUnauthenticated
	}

	if err := verifyClerkSignature(key, signingInput, signature); err != nil {
		return auth.ClerkPrincipal{}, auth.ErrUnauthenticated
	}

	principal, err := claims.toPrincipal(v, time.Now().UTC())
	if err != nil {
		return auth.ClerkPrincipal{}, auth.ErrUnauthenticated
	}

	return principal, nil
}

func extractClerkSessionToken(r *http.Request) (string, error) {
	if cookie, err := r.Cookie(clerkSessionCookieName); err == nil {
		if token := strings.TrimSpace(cookie.Value); token != "" {
			return token, nil
		}
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", errors.New("missing authorization")
	}

	if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
		authHeader = strings.TrimSpace(authHeader[7:])
	}

	if authHeader == "" {
		return "", errors.New("missing bearer token")
	}

	return authHeader, nil
}

func parseClerkJWT(token string) (clerkJWTHeader, string, []byte, clerkJWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return clerkJWTHeader{}, "", nil, clerkJWTClaims{}, fmt.Errorf("invalid jwt format")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return clerkJWTHeader{}, "", nil, clerkJWTClaims{}, err
	}

	var header clerkJWTHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return clerkJWTHeader{}, "", nil, clerkJWTClaims{}, err
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return clerkJWTHeader{}, "", nil, clerkJWTClaims{}, err
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return clerkJWTHeader{}, "", nil, clerkJWTClaims{}, err
	}

	var claims clerkJWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return clerkJWTHeader{}, "", nil, clerkJWTClaims{}, err
	}

	return header, parts[0] + "." + parts[1], signature, claims, nil
}

func verifyClerkSignature(key *rsa.PublicKey, signingInput string, signature []byte) error {
	if key == nil {
		return errors.New("missing public key")
	}

	sum := sha256.Sum256([]byte(signingInput))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], signature)
}

func (v *clerkJWTVerifier) publicKey(keyID string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[keyID]
	fetchedAt := v.fetchedAt
	v.mu.RUnlock()

	if ok && time.Since(fetchedAt) < 10*time.Minute {
		return key, nil
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	key, ok = v.keys[keyID]
	if ok && time.Since(v.fetchedAt) < 10*time.Minute {
		return key, nil
	}

	if err := v.refreshJWKSLocked(); err != nil {
		return nil, err
	}

	key, ok = v.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("clerk jwks missing key %q", keyID)
	}

	return key, nil
}

func (v *clerkJWTVerifier) refreshJWKS() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.refreshJWKSLocked()
}

func (v *clerkJWTVerifier) refreshJWKSLocked() error {
	req, err := http.NewRequest(http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("clerk jwks fetch failed: %s", resp.Status)
	}

	var payload struct {
		Keys []struct {
			KeyID string `json:"kid"`
			Type  string `json:"kty"`
			Use   string `json:"use"`
			Alg   string `json:"alg"`
			N     string `json:"n"`
			E     string `json:"e"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}

	keys := make(map[string]*rsa.PublicKey, len(payload.Keys))
	for _, jwk := range payload.Keys {
		if jwk.KeyID == "" || jwk.Type != "RSA" {
			continue
		}
		if jwk.Use != "" && jwk.Use != "sig" {
			continue
		}

		modulus, err := base64.RawURLEncoding.DecodeString(jwk.N)
		if err != nil {
			return err
		}
		exponentBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
		if err != nil {
			return err
		}
		exponent := 0
		for _, b := range exponentBytes {
			exponent = exponent<<8 | int(b)
		}
		if exponent == 0 {
			return fmt.Errorf("invalid jwks exponent for key %q", jwk.KeyID)
		}

		keys[jwk.KeyID] = &rsa.PublicKey{
			N: new(big.Int).SetBytes(modulus),
			E: exponent,
		}
	}

	if len(keys) == 0 {
		return errors.New("clerk jwks did not include any usable rsa keys")
	}

	v.keys = keys
	v.fetchedAt = time.Now().UTC()
	return nil
}

func (c clerkJWTClaims) toPrincipal(v *clerkJWTVerifier, now time.Time) (auth.ClerkPrincipal, error) {
	if c.Issuer != v.issuerURL {
		return auth.ClerkPrincipal{}, errors.New("invalid issuer")
	}
	if strings.TrimSpace(c.Subject) == "" || strings.TrimSpace(c.Session) == "" {
		return auth.ClerkPrincipal{}, errors.New("missing clerk identity")
	}
	if c.Expiry == 0 {
		return auth.ClerkPrincipal{}, errors.New("missing exp")
	}

	skewSeconds := int64(v.clockSkew / time.Second)
	nowUnix := now.Unix()
	if nowUnix > c.Expiry+skewSeconds {
		return auth.ClerkPrincipal{}, errors.New("token expired")
	}
	if c.NotBefore != 0 && nowUnix+skewSeconds < c.NotBefore {
		return auth.ClerkPrincipal{}, errors.New("token not active")
	}

	if v.audience != "" && !c.Audience.contains(v.audience) {
		return auth.ClerkPrincipal{}, errors.New("invalid audience")
	}

	if c.AZP != "" && v.allowedOrigin != "" && c.AZP != v.allowedOrigin {
		return auth.ClerkPrincipal{}, errors.New("invalid authorized party")
	}

	role := normalizeClerkRole(c.Role)
	if role == "" {
		role = normalizeClerkRole(c.OrgRole)
	}
	if role == "" {
		return auth.ClerkPrincipal{}, errors.New("missing role")
	}

	return auth.ClerkPrincipal{
		UserID:    strings.TrimSpace(c.Subject),
		SessionID: strings.TrimSpace(c.Session),
		Role:      role,
		Email:     strings.TrimSpace(c.Email),
	}, nil
}

func (a clerkAudiences) contains(expected string) bool {
	expected = strings.TrimSpace(expected)
	for _, candidate := range a {
		if strings.TrimSpace(candidate) == expected {
			return true
		}
	}

	return false
}

func normalizeClerkRole(role string) string {
	normalized := strings.ToLower(strings.TrimSpace(role))
	return strings.TrimPrefix(normalized, "org:")
}
