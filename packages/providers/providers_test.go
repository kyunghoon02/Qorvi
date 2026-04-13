package providers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
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
		NewMobulaAdapter(ProviderCredentials{Provider: ProviderMobula}, nil, nil),
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
		"CLERK_AUDIENCE":                    "qorvi",
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
		"POSTGRES_URL":                      "postgres://postgres:postgres@localhost:5432/qorvi",
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
	if env.DuneSeedExportRows != nil {
		t.Fatalf("expected no Dune seed export rows by default, got %#v", env.DuneSeedExportRows)
	}
	if env.Worker.RedisURL != "redis://localhost:6379" {
		t.Fatalf("expected Redis URL to be loaded")
	}
	if env.AlchemyBaseURL != "https://eth-mainnet.g.alchemy.com" {
		t.Fatalf("expected Alchemy base URL to be loaded")
	}
	if env.AlchemySolanaBaseURL != "https://solana-mainnet.g.alchemy.com" {
		t.Fatalf("expected Alchemy Solana base URL to be loaded")
	}
	if env.HeliusBaseURL != "https://mainnet.helius-rpc.com" {
		t.Fatalf("expected Helius base URL to be loaded")
	}
	if env.HeliusDataAPIBaseURL != "https://api-mainnet.helius-rpc.com/v0" {
		t.Fatalf("expected Helius data API base URL to be loaded")
	}
	if env.MoralisBaseURL != "https://deep-index.moralis.io/api/v2.2" {
		t.Fatalf("expected Moralis base URL default to be loaded, got %q", env.MoralisBaseURL)
	}
}

func TestParseProviderEnvNormalizesHeliusURLs(t *testing.T) {
	t.Parallel()

	env, err := ParseProviderEnv(map[string]string{
		"APP_BASE_URL":                      "http://localhost:3000",
		"ALCHEMY_API_KEY":                   "alchemy_secret",
		"AUTH_PROVIDER":                     "clerk",
		"AUTH_SECRET":                       "supersecret",
		"CLERK_AUDIENCE":                    "qorvi",
		"CLERK_CLOCK_SKEW_SECONDS":          "60",
		"CLERK_ISSUER_URL":                  "https://example.clerk.accounts.dev",
		"CLERK_JWKS_URL":                    "https://example.clerk.accounts.dev/.well-known/jwks.json",
		"CLERK_SECRET_KEY":                  "clerk_secret",
		"DUNE_API_KEY":                      "dune_secret",
		"HELIUS_API_KEY":                    "helius_secret",
		"HELIUS_BASE_URL":                   "https://mainnet.helius-rpc.com/?api-key=test-helius-key",
		"HELIUS_DATA_API_BASE_URL":          "https://mainnet.helius-rpc.com/?api-key=test-helius-key",
		"LOG_LEVEL":                         "info",
		"MORALIS_API_KEY":                   "moralis_secret",
		"NEO4J_PASSWORD":                    "neo4jpassword",
		"NEO4J_URL":                         "bolt://localhost:7687",
		"NEO4J_USERNAME":                    "neo4j",
		"NODE_ENV":                          "development",
		"POSTGRES_URL":                      "postgres://postgres:postgres@localhost:5432/qorvi",
		"REDIS_URL":                         "redis://localhost:6379",
		"NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY": "clerk_publishable",
	})
	if err != nil {
		t.Fatalf("ParseProviderEnv returned error: %v", err)
	}

	if env.HeliusBaseURL != "https://mainnet.helius-rpc.com" {
		t.Fatalf("expected normalized Helius base URL, got %q", env.HeliusBaseURL)
	}
	if env.HeliusDataAPIBaseURL != "https://api-mainnet.helius-rpc.com/v0" {
		t.Fatalf("expected normalized Helius data API base URL, got %q", env.HeliusDataAPIBaseURL)
	}
}

func TestParseProviderEnvLoadsDuneSeedExportRowsFromJSON(t *testing.T) {
	t.Parallel()

	env, err := ParseProviderEnv(map[string]string{
		"APP_BASE_URL":                      "http://localhost:3000",
		"ALCHEMY_API_KEY":                   "alchemy_secret",
		"AUTH_PROVIDER":                     "clerk",
		"AUTH_SECRET":                       "supersecret",
		"CLERK_AUDIENCE":                    "qorvi",
		"CLERK_CLOCK_SKEW_SECONDS":          "60",
		"CLERK_ISSUER_URL":                  "https://example.clerk.accounts.dev",
		"CLERK_JWKS_URL":                    "https://example.clerk.accounts.dev/.well-known/jwks.json",
		"CLERK_SECRET_KEY":                  "clerk_secret",
		"DUNE_API_KEY":                      "dune_secret",
		"DUNE_SEED_EXPORT_JSON":             `[{"chain":"evm","walletAddress":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","seedLabel":"seed-whale","confidence":0.92}]`,
		"HELIUS_API_KEY":                    "helius_secret",
		"LOG_LEVEL":                         "info",
		"MORALIS_API_KEY":                   "moralis_secret",
		"NEO4J_PASSWORD":                    "neo4jpassword",
		"NEO4J_URL":                         "bolt://localhost:7687",
		"NEO4J_USERNAME":                    "neo4j",
		"NODE_ENV":                          "development",
		"POSTGRES_URL":                      "postgres://postgres:postgres@localhost:5432/qorvi",
		"REDIS_URL":                         "redis://localhost:6379",
		"NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY": "clerk_publishable",
	})
	if err != nil {
		t.Fatalf("ParseProviderEnv returned error: %v", err)
	}

	if len(env.DuneSeedExportRows) != 1 {
		t.Fatalf("expected 1 Dune seed row, got %d", len(env.DuneSeedExportRows))
	}
	if env.DuneSeedExportRows[0].SeedLabel != "seed-whale" {
		t.Fatalf("unexpected Dune seed label %q", env.DuneSeedExportRows[0].SeedLabel)
	}
}

func TestParseProviderEnvLoadsDuneSeedExportRowsFromFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	exportPath := filepath.Join(tempDir, "dune-seeds.json")
	if err := os.WriteFile(exportPath, []byte(`[{"chain":"solana","walletAddress":"So11111111111111111111111111111111111111112","seedLabel":"sol-seed","confidence":0.88}]`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	env, err := ParseProviderEnv(map[string]string{
		"APP_BASE_URL":                      "http://localhost:3000",
		"ALCHEMY_API_KEY":                   "alchemy_secret",
		"AUTH_PROVIDER":                     "clerk",
		"AUTH_SECRET":                       "supersecret",
		"CLERK_AUDIENCE":                    "qorvi",
		"CLERK_CLOCK_SKEW_SECONDS":          "60",
		"CLERK_ISSUER_URL":                  "https://example.clerk.accounts.dev",
		"CLERK_JWKS_URL":                    "https://example.clerk.accounts.dev/.well-known/jwks.json",
		"CLERK_SECRET_KEY":                  "clerk_secret",
		"DUNE_API_KEY":                      "dune_secret",
		"DUNE_SEED_EXPORT_PATH":             exportPath,
		"HELIUS_API_KEY":                    "helius_secret",
		"LOG_LEVEL":                         "info",
		"MORALIS_API_KEY":                   "moralis_secret",
		"NEO4J_PASSWORD":                    "neo4jpassword",
		"NEO4J_URL":                         "bolt://localhost:7687",
		"NEO4J_USERNAME":                    "neo4j",
		"NODE_ENV":                          "development",
		"POSTGRES_URL":                      "postgres://postgres:postgres@localhost:5432/qorvi",
		"REDIS_URL":                         "redis://localhost:6379",
		"NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY": "clerk_publishable",
	})
	if err != nil {
		t.Fatalf("ParseProviderEnv returned error: %v", err)
	}

	if len(env.DuneSeedExportRows) != 1 {
		t.Fatalf("expected 1 Dune seed row, got %d", len(env.DuneSeedExportRows))
	}
	if env.DuneSeedExportRows[0].Chain != "solana" {
		t.Fatalf("unexpected Dune seed row chain %q", env.DuneSeedExportRows[0].Chain)
	}
}

func TestNewConfiguredRegistryInjectsDuneSeedExportRows(t *testing.T) {
	t.Parallel()

	registry := NewConfiguredRegistry(ProviderEnv{
		DuneAPIKey: "dune_secret",
		DuneSeedExportRows: []DuneSeedExportRow{
			{
				Chain:         "evm",
				WalletAddress: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				SeedLabel:     "seed-whale",
				Confidence:    0.9,
			},
		},
		AlchemyAPIKey: "alchemy_secret",
		HeliusAPIKey:  "helius_secret",
		MoralisAPIKey: "moralis_secret",
	})

	adapter, ok := registry[ProviderDune].(DuneAdapter)
	if !ok {
		t.Fatalf("expected DuneAdapter in configured registry")
	}
	if len(adapter.SeedDiscoveryRows) != 1 {
		t.Fatalf("expected configured Dune rows to be injected, got %d", len(adapter.SeedDiscoveryRows))
	}
}

func TestParseProviderEnvLoadsMobulaSmartMoneySeeds(t *testing.T) {
	t.Parallel()

	env, err := ParseProviderEnv(map[string]string{
		"APP_BASE_URL":                        "http://localhost:3000",
		"ALCHEMY_API_KEY":                     "alchemy_secret",
		"AUTH_PROVIDER":                       "clerk",
		"AUTH_SECRET":                         "supersecret",
		"CLERK_AUDIENCE":                      "qorvi",
		"CLERK_CLOCK_SKEW_SECONDS":            "60",
		"CLERK_ISSUER_URL":                    "https://example.clerk.accounts.dev",
		"CLERK_JWKS_URL":                      "https://example.clerk.accounts.dev/.well-known/jwks.json",
		"CLERK_SECRET_KEY":                    "clerk_secret",
		"DUNE_API_KEY":                        "dune_secret",
		"HELIUS_API_KEY":                      "helius_secret",
		"LOG_LEVEL":                           "info",
		"MOBULA_API_KEY":                      "mobula_secret",
		"QORVI_MOBULA_SMART_MONEY_SEEDS_JSON": `[{"chain":"evm","tokenAddress":"0x6982508145454Ce325dDbE47a25d4ec3d2311933","tokenSymbol":"PEPE"}]`,
		"MORALIS_API_KEY":                     "moralis_secret",
		"NEO4J_PASSWORD":                      "neo4jpassword",
		"NEO4J_URL":                           "bolt://localhost:7687",
		"NEO4J_USERNAME":                      "neo4j",
		"NODE_ENV":                            "development",
		"POSTGRES_URL":                        "postgres://postgres:postgres@localhost:5432/qorvi",
		"REDIS_URL":                           "redis://localhost:6379",
		"NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY":   "clerk_publishable",
	})
	if err != nil {
		t.Fatalf("ParseProviderEnv returned error: %v", err)
	}

	if env.MobulaBaseURL != "https://api.mobula.io" {
		t.Fatalf("expected Mobula base URL default, got %q", env.MobulaBaseURL)
	}
	if len(env.MobulaSmartMoneySeeds) != 1 {
		t.Fatalf("expected 1 Mobula seed, got %d", len(env.MobulaSmartMoneySeeds))
	}
	if env.MobulaSmartMoneySeeds[0].Blockchain != "ethereum" {
		t.Fatalf("expected normalized blockchain, got %q", env.MobulaSmartMoneySeeds[0].Blockchain)
	}
	if env.MobulaSmartMoneySeeds[0].Address != "0x6982508145454Ce325dDbE47a25d4ec3d2311933" {
		t.Fatalf("unexpected seed address %q", env.MobulaSmartMoneySeeds[0].Address)
	}
	if len(env.MobulaSmartMoneySeeds[0].Labels) != 2 {
		t.Fatalf("expected default Mobula labels, got %#v", env.MobulaSmartMoneySeeds[0].Labels)
	}
}

func TestParseProviderEnvRequiresMobulaKeyWhenSeedsConfigured(t *testing.T) {
	t.Parallel()

	_, err := ParseProviderEnv(map[string]string{
		"APP_BASE_URL":                        "http://localhost:3000",
		"ALCHEMY_API_KEY":                     "alchemy_secret",
		"AUTH_PROVIDER":                       "clerk",
		"AUTH_SECRET":                         "supersecret",
		"CLERK_AUDIENCE":                      "qorvi",
		"CLERK_CLOCK_SKEW_SECONDS":            "60",
		"CLERK_ISSUER_URL":                    "https://example.clerk.accounts.dev",
		"CLERK_JWKS_URL":                      "https://example.clerk.accounts.dev/.well-known/jwks.json",
		"CLERK_SECRET_KEY":                    "clerk_secret",
		"DUNE_API_KEY":                        "dune_secret",
		"HELIUS_API_KEY":                      "helius_secret",
		"LOG_LEVEL":                           "info",
		"QORVI_MOBULA_SMART_MONEY_SEEDS_JSON": `[{"blockchain":"ethereum","address":"0x6982508145454Ce325dDbE47a25d4ec3d2311933","labels":["smartTrader"]}]`,
		"MORALIS_API_KEY":                     "moralis_secret",
		"NEO4J_PASSWORD":                      "neo4jpassword",
		"NEO4J_URL":                           "bolt://localhost:7687",
		"NEO4J_USERNAME":                      "neo4j",
		"NODE_ENV":                            "development",
		"POSTGRES_URL":                        "postgres://postgres:postgres@localhost:5432/qorvi",
		"REDIS_URL":                           "redis://localhost:6379",
		"NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY":   "clerk_publishable",
	})
	if err == nil {
		t.Fatal("expected ParseProviderEnv to require MOBULA_API_KEY")
	}
}
