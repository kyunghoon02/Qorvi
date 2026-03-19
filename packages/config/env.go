package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type ClerkVerificationConfig struct {
	IssuerURL        string
	JWKSURL          string
	Audience         string
	ClockSkewSeconds int
}

type APIEnv struct {
	NodeEnv                       string
	LogLevel                      string
	AppBaseURL                    string
	APIHost                       string
	APIPort                       int
	PostgresURL                   string
	Neo4jURL                      string
	Neo4jUsername                 string
	Neo4jPassword                 string
	RedisURL                      string
	AuthProvider                  string
	AuthSecret                    string
	ClerkSecretKey                string
	ClerkVerification             ClerkVerificationConfig
	NextPublicClerkPublishableKey string
}

type WorkerEnv struct {
	NodeEnv                       string
	LogLevel                      string
	AppBaseURL                    string
	PostgresURL                   string
	Neo4jURL                      string
	Neo4jUsername                 string
	Neo4jPassword                 string
	RedisURL                      string
	AuthProvider                  string
	AuthSecret                    string
	ClerkSecretKey                string
	ClerkVerification             ClerkVerificationConfig
	NextPublicClerkPublishableKey string
}

type WebEnv struct {
	NodeEnv                       string
	NextPublicAppBaseURL          string
	NextPublicClerkPublishableKey string
}

func ParseAPIEnvFromOS() (APIEnv, error) {
	return ParseAPIEnv(envMapFromOS())
}

func ParseWorkerEnvFromOS() (WorkerEnv, error) {
	return ParseWorkerEnv(envMapFromOS())
}

func ParseWebEnvFromOS() (WebEnv, error) {
	return ParseWebEnv(envMapFromOS())
}

func ParseAPIEnv(source map[string]string) (APIEnv, error) {
	appBaseURL, err := required(source, "APP_BASE_URL")
	if err != nil {
		return APIEnv{}, err
	}
	apiHost, err := required(source, "API_HOST")
	if err != nil {
		return APIEnv{}, err
	}
	apiPort, err := requiredInt(source, "API_PORT")
	if err != nil {
		return APIEnv{}, err
	}
	postgresURL, err := required(source, "POSTGRES_URL")
	if err != nil {
		return APIEnv{}, err
	}
	neo4jURL, err := required(source, "NEO4J_URL")
	if err != nil {
		return APIEnv{}, err
	}
	neo4jUsername, err := required(source, "NEO4J_USERNAME")
	if err != nil {
		return APIEnv{}, err
	}
	neo4jPassword, err := required(source, "NEO4J_PASSWORD")
	if err != nil {
		return APIEnv{}, err
	}
	redisURL, err := required(source, "REDIS_URL")
	if err != nil {
		return APIEnv{}, err
	}
	authProvider, err := required(source, "AUTH_PROVIDER")
	if err != nil {
		return APIEnv{}, err
	}
	authSecret, err := required(source, "AUTH_SECRET")
	if err != nil {
		return APIEnv{}, err
	}
	clerkSecretKey, err := required(source, "CLERK_SECRET_KEY")
	if err != nil {
		return APIEnv{}, err
	}
	clerkVerification, err := ParseClerkVerificationConfig(source)
	if err != nil {
		return APIEnv{}, err
	}
	publishableKey, err := required(source, "NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY")
	if err != nil {
		return APIEnv{}, err
	}

	env := APIEnv{
		NodeEnv:                       requiredWithDefault(source, "NODE_ENV", "development"),
		LogLevel:                      requiredWithDefault(source, "LOG_LEVEL", "info"),
		AppBaseURL:                    appBaseURL,
		APIHost:                       apiHost,
		APIPort:                       apiPort,
		PostgresURL:                   postgresURL,
		Neo4jURL:                      neo4jURL,
		Neo4jUsername:                 neo4jUsername,
		Neo4jPassword:                 neo4jPassword,
		RedisURL:                      redisURL,
		AuthProvider:                  authProvider,
		AuthSecret:                    authSecret,
		ClerkSecretKey:                clerkSecretKey,
		ClerkVerification:             clerkVerification,
		NextPublicClerkPublishableKey: publishableKey,
	}

	return env, validateBackendURLs(env.AppBaseURL, env.PostgresURL, env.Neo4jURL, env.RedisURL)
}

func ParseWorkerEnv(source map[string]string) (WorkerEnv, error) {
	appBaseURL, err := required(source, "APP_BASE_URL")
	if err != nil {
		return WorkerEnv{}, err
	}
	postgresURL, err := required(source, "POSTGRES_URL")
	if err != nil {
		return WorkerEnv{}, err
	}
	neo4jURL, err := required(source, "NEO4J_URL")
	if err != nil {
		return WorkerEnv{}, err
	}
	neo4jUsername, err := required(source, "NEO4J_USERNAME")
	if err != nil {
		return WorkerEnv{}, err
	}
	neo4jPassword, err := required(source, "NEO4J_PASSWORD")
	if err != nil {
		return WorkerEnv{}, err
	}
	redisURL, err := required(source, "REDIS_URL")
	if err != nil {
		return WorkerEnv{}, err
	}
	authProvider, err := required(source, "AUTH_PROVIDER")
	if err != nil {
		return WorkerEnv{}, err
	}
	authSecret, err := required(source, "AUTH_SECRET")
	if err != nil {
		return WorkerEnv{}, err
	}
	clerkSecretKey, err := required(source, "CLERK_SECRET_KEY")
	if err != nil {
		return WorkerEnv{}, err
	}
	clerkVerification, err := ParseClerkVerificationConfig(source)
	if err != nil {
		return WorkerEnv{}, err
	}
	publishableKey, err := required(source, "NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY")
	if err != nil {
		return WorkerEnv{}, err
	}

	env := WorkerEnv{
		NodeEnv:                       requiredWithDefault(source, "NODE_ENV", "development"),
		LogLevel:                      requiredWithDefault(source, "LOG_LEVEL", "info"),
		AppBaseURL:                    appBaseURL,
		PostgresURL:                   postgresURL,
		Neo4jURL:                      neo4jURL,
		Neo4jUsername:                 neo4jUsername,
		Neo4jPassword:                 neo4jPassword,
		RedisURL:                      redisURL,
		AuthProvider:                  authProvider,
		AuthSecret:                    authSecret,
		ClerkSecretKey:                clerkSecretKey,
		ClerkVerification:             clerkVerification,
		NextPublicClerkPublishableKey: publishableKey,
	}

	return env, validateBackendURLs(env.AppBaseURL, env.PostgresURL, env.Neo4jURL, env.RedisURL)
}

func ParseWebEnv(source map[string]string) (WebEnv, error) {
	baseURL, err := required(source, "NEXT_PUBLIC_APP_BASE_URL")
	if err != nil {
		return WebEnv{}, err
	}
	publishableKey, err := required(source, "NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY")
	if err != nil {
		return WebEnv{}, err
	}

	env := WebEnv{
		NodeEnv:                       requiredWithDefault(source, "NODE_ENV", "development"),
		NextPublicAppBaseURL:          baseURL,
		NextPublicClerkPublishableKey: publishableKey,
	}

	if err := validateURL(env.NextPublicAppBaseURL, "NEXT_PUBLIC_APP_BASE_URL"); err != nil {
		return WebEnv{}, err
	}

	return env, nil
}

func envMapFromOS() map[string]string {
	keys := []string{
		"APP_BASE_URL",
		"API_HOST",
		"API_PORT",
		"AUTH_PROVIDER",
		"AUTH_SECRET",
		"CLERK_SECRET_KEY",
		"CLERK_AUDIENCE",
		"CLERK_CLOCK_SKEW_SECONDS",
		"CLERK_ISSUER_URL",
		"CLERK_JWKS_URL",
		"LOG_LEVEL",
		"NEO4J_PASSWORD",
		"NEO4J_URL",
		"NEO4J_USERNAME",
		"NEXT_PUBLIC_APP_BASE_URL",
		"NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY",
		"NODE_ENV",
		"POSTGRES_URL",
		"REDIS_URL",
	}

	source := make(map[string]string, len(keys))
	for _, key := range keys {
		source[key] = os.Getenv(key)
	}

	return source
}

func ParseClerkVerificationConfig(source map[string]string) (ClerkVerificationConfig, error) {
	issuerURL, err := requiredHTTPSURL(source, "CLERK_ISSUER_URL")
	if err != nil {
		return ClerkVerificationConfig{}, err
	}
	jwksURL, err := requiredHTTPSURL(source, "CLERK_JWKS_URL")
	if err != nil {
		return ClerkVerificationConfig{}, err
	}
	audience := strings.TrimSpace(source["CLERK_AUDIENCE"])
	clockSkewSeconds, err := optionalInt(source, "CLERK_CLOCK_SKEW_SECONDS", 60)
	if err != nil {
		return ClerkVerificationConfig{}, err
	}

	return ClerkVerificationConfig{
		IssuerURL:        issuerURL,
		JWKSURL:          jwksURL,
		Audience:         audience,
		ClockSkewSeconds: clockSkewSeconds,
	}, nil
}

func required(source map[string]string, key string) (string, error) {
	value := source[key]
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}

	return value, nil
}

func requiredWithDefault(source map[string]string, key string, fallback string) string {
	value := source[key]
	if value == "" {
		return fallback
	}

	return value
}

func requiredInt(source map[string]string, key string) (int, error) {
	value := source[key]
	if value == "" {
		return 0, fmt.Errorf("%s is required", key)
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}

	return parsed, nil
}

func optionalInt(source map[string]string, key string, fallback int) (int, error) {
	value := strings.TrimSpace(source[key])
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}

	if parsed < 0 {
		return 0, fmt.Errorf("%s must be non-negative", key)
	}

	return parsed, nil
}

func validateBackendURLs(values ...string) error {
	for index, value := range values {
		key := "backend_url"
		switch index {
		case 0:
			key = "APP_BASE_URL"
		case 1:
			key = "POSTGRES_URL"
		case 2:
			key = "NEO4J_URL"
		case 3:
			key = "REDIS_URL"
		}

		if err := validateURL(value, key); err != nil {
			return err
		}
	}

	return nil
}

func validateURL(value string, key string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("%s must be a valid connection string", key)
	}

	return nil
}

func requiredHTTPSURL(source map[string]string, key string) (string, error) {
	value, err := required(source, key)
	if err != nil {
		return "", err
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("%s must be a valid URL: %w", key, err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("%s must use https", key)
	}

	return value, nil
}
