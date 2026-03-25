package auth

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	sharedconfig "github.com/flowintel/flowintel/packages/config"
)

type ClerkPrincipal struct {
	UserID    string
	SessionID string
	Role      string
	Email     string
}

type ClerkVerifier interface {
	Verify(*http.Request) (ClerkPrincipal, error)
}

type ClerkErrorResponder interface {
	Unauthorized(http.ResponseWriter, *http.Request)
	Forbidden(http.ResponseWriter, *http.Request)
}

var ErrUnauthenticated = errors.New("clerk identity is missing")

type HeaderClerkVerifier struct {
	jwtVerifier *clerkJWTVerifier
}

func NewHeaderClerkVerifier() HeaderClerkVerifier {
	cfg, err := loadClerkVerificationConfigFromOS()
	if err != nil {
		return HeaderClerkVerifier{}
	}

	return NewHeaderClerkVerifierWithConfig(cfg)
}

func NewHeaderClerkVerifierWithConfig(cfg sharedconfig.ClerkVerificationConfig) HeaderClerkVerifier {
	return HeaderClerkVerifier{
		jwtVerifier: newClerkJWTVerifier(cfg, http.DefaultClient, time.Now),
	}
}

func (v HeaderClerkVerifier) Verify(r *http.Request) (ClerkPrincipal, error) {
	token, bearerPresent, err := bearerTokenFromRequest(r)
	if err != nil {
		return ClerkPrincipal{}, err
	}

	if bearerPresent {
		if v.jwtVerifier == nil {
			return ClerkPrincipal{}, ErrUnauthenticated
		}

		principal, err := v.jwtVerifier.Verify(token)
		if err != nil {
			return ClerkPrincipal{}, err
		}

		return principal, nil
	}

	return verifyLegacyHeaders(r)
}

func RequireClerkRole(verifier ClerkVerifier, responder ClerkErrorResponder, allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowed[normalizeRole(role)] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := verifier.Verify(r)
			if err != nil {
				responder.Unauthorized(w, r)
				return
			}

			if _, ok := allowed[principal.Role]; !ok {
				responder.Forbidden(w, r)
				return
			}

			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal)))
		})
	}
}

func PrincipalFromContext(ctx context.Context) (ClerkPrincipal, bool) {
	principal, ok := ctx.Value(clerkPrincipalKey{}).(ClerkPrincipal)
	return principal, ok
}

type clerkPrincipalKey struct{}

func withPrincipal(ctx context.Context, principal ClerkPrincipal) context.Context {
	return context.WithValue(ctx, clerkPrincipalKey{}, principal)
}

func verifyLegacyHeaders(r *http.Request) (ClerkPrincipal, error) {
	principal := ClerkPrincipal{
		UserID:    strings.TrimSpace(r.Header.Get("X-Clerk-User-Id")),
		SessionID: strings.TrimSpace(r.Header.Get("X-Clerk-Session-Id")),
		Role:      normalizeRole(r.Header.Get("X-Clerk-Role")),
		Email:     strings.TrimSpace(r.Header.Get("X-Clerk-Email")),
	}

	if principal.UserID == "" || principal.SessionID == "" || principal.Role == "" {
		return ClerkPrincipal{}, ErrUnauthenticated
	}

	return principal, nil
}

func bearerTokenFromRequest(r *http.Request) (string, bool, error) {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		return "", false, nil
	}

	scheme, token, ok := strings.Cut(authorization, " ")
	if !ok || !strings.EqualFold(strings.TrimSpace(scheme), "Bearer") {
		return "", false, nil
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", true, ErrUnauthenticated
	}

	return token, true, nil
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin", "operator", "user":
		return strings.ToLower(strings.TrimSpace(role))
	default:
		return ""
	}
}

func loadClerkVerificationConfigFromOS() (sharedconfig.ClerkVerificationConfig, error) {
	source := map[string]string{
		"CLERK_ISSUER_URL":         os.Getenv("CLERK_ISSUER_URL"),
		"CLERK_JWKS_URL":           os.Getenv("CLERK_JWKS_URL"),
		"CLERK_AUDIENCE":           os.Getenv("CLERK_AUDIENCE"),
		"CLERK_CLOCK_SKEW_SECONDS": os.Getenv("CLERK_CLOCK_SKEW_SECONDS"),
	}

	return sharedconfig.ParseClerkVerificationConfig(source)
}

type clerkJWTVerifier struct {
	cfg    sharedconfig.ClerkVerificationConfig
	client *http.Client
	now    func() time.Time

	mu   sync.RWMutex
	keys map[string]*rsa.PublicKey
}

func newClerkJWTVerifier(cfg sharedconfig.ClerkVerificationConfig, client *http.Client, now func() time.Time) *clerkJWTVerifier {
	if client == nil {
		client = http.DefaultClient
	}
	if now == nil {
		now = time.Now
	}

	return &clerkJWTVerifier{
		cfg:    cfg,
		client: client,
		now:    now,
		keys:   make(map[string]*rsa.PublicKey),
	}
}

func (v *clerkJWTVerifier) Verify(token string) (ClerkPrincipal, error) {
	header, payload, signature, signingInput, err := parseJWT(token)
	if err != nil {
		return ClerkPrincipal{}, err
	}

	if !strings.EqualFold(header.Alg, "RS256") {
		return ClerkPrincipal{}, fmt.Errorf("unsupported clerk token algorithm %q", header.Alg)
	}

	publicKey, err := v.publicKeyForKid(header.Kid)
	if err != nil {
		return ClerkPrincipal{}, err
	}

	if err := verifyRS256(signingInput, signature, publicKey); err != nil {
		return ClerkPrincipal{}, err
	}

	claims, err := decodeClaims(payload)
	if err != nil {
		return ClerkPrincipal{}, err
	}

	principal, err := v.principalFromClaims(claims)
	if err != nil {
		return ClerkPrincipal{}, err
	}

	return principal, nil
}

func (v *clerkJWTVerifier) principalFromClaims(claims map[string]any) (ClerkPrincipal, error) {
	issuer, _ := claimString(claims["iss"])
	if strings.TrimSpace(issuer) != strings.TrimSpace(v.cfg.IssuerURL) {
		return ClerkPrincipal{}, ErrUnauthenticated
	}

	if v.cfg.Audience != "" && !claimMatchesAudience(claims["aud"], v.cfg.Audience) {
		return ClerkPrincipal{}, ErrUnauthenticated
	}

	now := v.now().UTC().Unix()
	skew := int64(v.cfg.ClockSkewSeconds)

	if exp, ok := claimInt64(claims["exp"]); !ok || now > exp+skew {
		return ClerkPrincipal{}, ErrUnauthenticated
	}

	if nbf, ok := claimInt64(claims["nbf"]); ok && now+skew < nbf {
		return ClerkPrincipal{}, ErrUnauthenticated
	}

	userID, _ := claimString(claims["sub"])
	sessionID, _ := claimString(claims["sid"])
	role := normalizeRole(claimRole(claims))
	email := claimEmail(claims)

	if userID == "" || sessionID == "" || role == "" {
		return ClerkPrincipal{}, ErrUnauthenticated
	}

	return ClerkPrincipal{
		UserID:    userID,
		SessionID: sessionID,
		Role:      role,
		Email:     email,
	}, nil
}

func (v *clerkJWTVerifier) publicKeyForKid(kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	if key, ok := v.keys[kid]; ok {
		v.mu.RUnlock()
		return key, nil
	}
	v.mu.RUnlock()

	keys, err := v.fetchJWKS()
	if err != nil {
		return nil, err
	}

	v.mu.Lock()
	v.keys = keys
	key := v.keys[kid]
	v.mu.Unlock()

	if key == nil && len(keys) == 1 && kid == "" {
		for _, candidate := range keys {
			return candidate, nil
		}
	}

	if key == nil {
		return nil, ErrUnauthenticated
	}

	return key, nil
}

func (v *clerkJWTVerifier) fetchJWKS() (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequest(http.MethodGet, v.cfg.JWKSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build clerk jwks request: %w", err)
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch clerk jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("fetch clerk jwks: unexpected status %d", resp.StatusCode)
	}

	var document struct {
		Keys []clerkJWK `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		return nil, fmt.Errorf("decode clerk jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, jwk := range document.Keys {
		if jwk.Kid == "" {
			continue
		}

		publicKey, err := jwk.rsaPublicKey()
		if err != nil {
			return nil, err
		}

		keys[jwk.Kid] = publicKey
	}

	if len(keys) == 0 {
		return nil, ErrUnauthenticated
	}

	return keys, nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

type clerkJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (j clerkJWK) rsaPublicKey() (*rsa.PublicKey, error) {
	if !strings.EqualFold(j.Kty, "RSA") {
		return nil, fmt.Errorf("unsupported clerk jwk key type %q", j.Kty)
	}
	if j.N == "" || j.E == "" {
		return nil, ErrUnauthenticated
	}

	modulus, err := base64.RawURLEncoding.DecodeString(j.N)
	if err != nil {
		return nil, fmt.Errorf("decode clerk jwk modulus: %w", err)
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(j.E)
	if err != nil {
		return nil, fmt.Errorf("decode clerk jwk exponent: %w", err)
	}

	exponent := 0
	for _, b := range exponentBytes {
		exponent = (exponent << 8) | int(b)
	}
	if exponent <= 0 {
		return nil, ErrUnauthenticated
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(modulus),
		E: exponent,
	}, nil
}

func parseJWT(token string) (jwtHeader, []byte, []byte, []byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtHeader{}, nil, nil, nil, ErrUnauthenticated
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtHeader{}, nil, nil, nil, ErrUnauthenticated
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtHeader{}, nil, nil, nil, ErrUnauthenticated
	}
	signatureBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtHeader{}, nil, nil, nil, ErrUnauthenticated
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return jwtHeader{}, nil, nil, nil, ErrUnauthenticated
	}

	signingInput := []byte(parts[0] + "." + parts[1])
	return header, payloadBytes, signatureBytes, signingInput, nil
}

func verifyRS256(signingInput []byte, signature []byte, publicKey *rsa.PublicKey) error {
	digest := sha256.Sum256(signingInput)
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return ErrUnauthenticated
	}

	return nil
}

func decodeClaims(payload []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()

	var claims map[string]any
	if err := decoder.Decode(&claims); err != nil {
		return nil, ErrUnauthenticated
	}

	return claims, nil
}

func claimString(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), true
	case json.Number:
		return typed.String(), true
	default:
		return "", false
	}
}

func claimInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	case float64:
		return int64(typed), true
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	default:
		return 0, false
	}
}

func claimMatchesAudience(value any, expected string) bool {
	switch typed := value.(type) {
	case string:
		return typed == expected
	case []any:
		for _, candidate := range typed {
			if s, ok := candidate.(string); ok && s == expected {
				return true
			}
		}
		return false
	case []string:
		for _, candidate := range typed {
			if candidate == expected {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func claimRole(claims map[string]any) string {
	for _, key := range []string{"role", "rol", "org_role"} {
		if value, ok := claimString(claims[key]); ok {
			role := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "org:")
			if normalized := normalizeRole(role); normalized != "" {
				return normalized
			}
		}
	}

	return ""
}

func claimEmail(claims map[string]any) string {
	for _, key := range []string{"email", "email_address"} {
		if value, ok := claimString(claims[key]); ok && value != "" {
			return value
		}
	}

	return ""
}
