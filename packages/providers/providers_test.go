package providers

import (
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestProviderAdaptersEmitSchemaCompliantFixtures(t *testing.T) {
	t.Parallel()

	ctx := ProviderRequestContext{
		Chain:         domain.ChainEVM,
		WalletAddress: "0x1234567890abcdef1234567890abcdef12345678",
		Access: domain.AccessContext{
			Role: domain.RoleUser,
			Plan: domain.PlanPro,
		},
	}

	adapters := []ProviderAdapter{
		DuneAdapter{},
		AlchemyAdapter{},
		HeliusAdapter{},
		MoralisAdapter{},
	}

	for _, adapter := range adapters {
		adapter := adapter
		t.Run(string(adapter.Name()), func(t *testing.T) {
			t.Parallel()

			activities, err := adapter.FetchWalletActivity(ctx)
			if err != nil {
				t.Fatalf("FetchWalletActivity returned error: %v", err)
			}
			if len(activities) != 1 {
				t.Fatalf("expected 1 activity, got %d", len(activities))
			}
			if activities[0].Provider != adapter.Name() {
				t.Fatalf("expected provider %q, got %q", adapter.Name(), activities[0].Provider)
			}
			if activities[0].WalletAddress != ctx.WalletAddress {
				t.Fatalf("expected wallet %q, got %q", ctx.WalletAddress, activities[0].WalletAddress)
			}
		})
	}
}

func TestCreateProviderWalletSummaryReusesSharedDomainContract(t *testing.T) {
	t.Parallel()

	summary := CreateProviderWalletSummary(
		domain.ChainSolana,
		"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
	)

	if summary.Chain != domain.ChainSolana {
		t.Fatalf("expected chain %q, got %q", domain.ChainSolana, summary.Chain)
	}
	if len(summary.Scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(summary.Scores))
	}
	if summary.Scores[0].Name != "cluster_score" {
		t.Fatalf("expected first score cluster_score, got %q", summary.Scores[0].Name)
	}
}

func TestParseProviderEnvLayersOnTopOfSharedWorkerEnvValidation(t *testing.T) {
	t.Parallel()

	env, err := ParseProviderEnv(map[string]string{
		"APP_BASE_URL":                      "http://localhost:3000",
		"ALCHEMY_API_KEY":                   "alchemy_secret",
		"ALCHEMY_BASE_URL":                  "https://eth-mainnet.g.alchemy.com",
		"AUTH_PROVIDER":                     "clerk",
		"AUTH_SECRET":                       "supersecret",
		"CLERK_AUDIENCE":                    "whalegraph",
		"CLERK_CLOCK_SKEW_SECONDS":          "60",
		"CLERK_ISSUER_URL":                  "https://example.clerk.accounts.dev",
		"CLERK_JWKS_URL":                    "https://example.clerk.accounts.dev/.well-known/jwks.json",
		"CLERK_SECRET_KEY":                  "clerk_secret",
		"DUNE_API_KEY":                      "dune_secret",
		"HELIUS_API_KEY":                    "helius_secret",
		"HELIUS_BASE_URL":                   "https://mainnet.helius-rpc.com",
		"HELIUS_DATA_API_BASE_URL":          "https://api-mainnet.helius-rpc.com/v0",
		"LOG_LEVEL":                         "info",
		"MORALIS_API_KEY":                   "moralis_secret",
		"NEO4J_PASSWORD":                    "neo4jpassword",
		"NEO4J_URL":                         "bolt://localhost:7687",
		"NEO4J_USERNAME":                    "neo4j",
		"NODE_ENV":                          "development",
		"POSTGRES_URL":                      "postgres://postgres:postgres@localhost:5432/whalegraph",
		"REDIS_URL":                         "redis://localhost:6379",
		"NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY": "clerk_publishable",
	})
	if err != nil {
		t.Fatalf("ParseProviderEnv returned error: %v", err)
	}

	if env.Worker.NodeEnv != "development" {
		t.Fatalf("expected worker NODE_ENV development, got %q", env.Worker.NodeEnv)
	}
	if env.DuneAPIKey != "dune_secret" {
		t.Fatalf("expected Dune API key to be loaded")
	}
	if env.Worker.RedisURL != "redis://localhost:6379" {
		t.Fatalf("expected Redis URL to be loaded")
	}
	if env.AlchemyBaseURL != "https://eth-mainnet.g.alchemy.com" {
		t.Fatalf("expected Alchemy base URL to be loaded")
	}
	if env.HeliusBaseURL != "https://mainnet.helius-rpc.com" {
		t.Fatalf("expected Helius base URL to be loaded")
	}
	if env.HeliusDataAPIBaseURL != "https://api-mainnet.helius-rpc.com/v0" {
		t.Fatalf("expected Helius data API base URL to be loaded")
	}
}
