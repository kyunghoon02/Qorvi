package config

import "testing"

var baseEnv = map[string]string{
	"APP_BASE_URL":                      "http://localhost:3000",
	"API_HOST":                          "0.0.0.0",
	"API_PORT":                          "4000",
	"AUTH_PROVIDER":                     "clerk",
	"AUTH_SECRET":                       "supersecret",
	"CLERK_SECRET_KEY":                  "clerk_secret",
	"CLERK_AUDIENCE":                    "flowintel",
	"CLERK_CLOCK_SKEW_SECONDS":          "60",
	"CLERK_ISSUER_URL":                  "https://example.clerk.accounts.dev",
	"CLERK_JWKS_URL":                    "https://example.clerk.accounts.dev/.well-known/jwks.json",
	"LOG_LEVEL":                         "info",
	"NEO4J_PASSWORD":                    "neo4jpassword",
	"NEO4J_URL":                         "bolt://localhost:7687",
	"NEO4J_USERNAME":                    "neo4j",
	"NEXT_PUBLIC_APP_BASE_URL":          "http://localhost:3000",
	"NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY": "clerk_publishable",
	"NODE_ENV":                          "development",
	"POSTGRES_URL":                      "postgres://postgres:postgres@localhost:5432/flowintel",
	"REDIS_URL":                         "redis://localhost:6379",
}

func cloneEnv() map[string]string {
	clone := make(map[string]string, len(baseEnv))
	for key, value := range baseEnv {
		clone[key] = value
	}

	return clone
}

func TestParseAPIEnv(t *testing.T) {
	t.Parallel()

	env, err := ParseAPIEnv(cloneEnv())
	if err != nil {
		t.Fatalf("expected valid api env, got %v", err)
	}

	if env.APIPort != 4000 {
		t.Fatalf("expected API port 4000, got %d", env.APIPort)
	}
	if env.ClerkVerification.IssuerURL != "https://example.clerk.accounts.dev" {
		t.Fatalf("unexpected issuer url %q", env.ClerkVerification.IssuerURL)
	}
	if env.ClerkVerification.ClockSkewSeconds != 60 {
		t.Fatalf("unexpected clock skew %d", env.ClerkVerification.ClockSkewSeconds)
	}
}

func TestParseWebEnv(t *testing.T) {
	t.Parallel()

	env, err := ParseWebEnv(cloneEnv())
	if err != nil {
		t.Fatalf("expected valid web env, got %v", err)
	}

	if env.NextPublicAppBaseURL != "http://localhost:3000" {
		t.Fatalf("unexpected base url %q", env.NextPublicAppBaseURL)
	}
}

func TestParseClerkVerificationConfig(t *testing.T) {
	t.Parallel()

	cfg, err := ParseClerkVerificationConfig(cloneEnv())
	if err != nil {
		t.Fatalf("expected clerk verification config, got %v", err)
	}

	if cfg.JWKSURL != "https://example.clerk.accounts.dev/.well-known/jwks.json" {
		t.Fatalf("unexpected jwks url %q", cfg.JWKSURL)
	}
	if cfg.Audience != "flowintel" {
		t.Fatalf("unexpected audience %q", cfg.Audience)
	}
}

func TestParseClerkVerificationConfigDerivesURLsFromPublishableKey(t *testing.T) {
	t.Parallel()

	env := cloneEnv()
	env["CLERK_ISSUER_URL"] = ""
	env["CLERK_JWKS_URL"] = ""
	env["NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY"] = "pk_test_YWJzb2x1dGUtaW1wYWxhLTY0LmNsZXJrLmFjY291bnRzLmRldiQ"

	cfg, err := ParseClerkVerificationConfig(env)
	if err != nil {
		t.Fatalf("expected derived clerk verification config, got %v", err)
	}

	if cfg.IssuerURL != "https://absolute-impala-64.clerk.accounts.dev" {
		t.Fatalf("unexpected derived issuer url %q", cfg.IssuerURL)
	}
	if cfg.JWKSURL != "https://absolute-impala-64.clerk.accounts.dev/.well-known/jwks.json" {
		t.Fatalf("unexpected derived jwks url %q", cfg.JWKSURL)
	}
}

func TestParseAPIEnvFailsWhenPostgresURLIsMissing(t *testing.T) {
	t.Parallel()

	env := cloneEnv()
	delete(env, "POSTGRES_URL")

	_, err := ParseAPIEnv(env)
	if err == nil {
		t.Fatal("expected missing POSTGRES_URL to fail")
	}
}

func TestParseClerkVerificationConfigRejectsNonHttpsIssuer(t *testing.T) {
	t.Parallel()

	env := cloneEnv()
	env["CLERK_ISSUER_URL"] = "http://example.clerk.accounts.dev"

	_, err := ParseClerkVerificationConfig(env)
	if err == nil {
		t.Fatal("expected non-https issuer url to fail")
	}
}
