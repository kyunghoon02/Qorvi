package providers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	sharedconfig "github.com/qorvi/qorvi/packages/config"
)

const (
	defaultAlchemyBaseURL       = "https://eth-mainnet.g.alchemy.com"
	defaultAlchemySolanaBaseURL = "https://solana-mainnet.g.alchemy.com"
	defaultHeliusBaseURL        = "https://mainnet.helius-rpc.com"
	defaultHeliusDataAPIBaseURL = "https://api-mainnet.helius-rpc.com/v0"
	defaultMobulaBaseURL        = "https://api.mobula.io"
	defaultMoralisBaseURL       = "https://deep-index.moralis.io/api/v2.2"
)

type ProviderEnv struct {
	Worker                    sharedconfig.WorkerEnv
	DuneAPIKey                string
	DuneSeedExportJSON        string
	DuneSeedExportPath        string
	DuneSeedExportRows        []DuneSeedExportRow
	AlchemyAPIKey             string
	AlchemyBaseURL            string
	AlchemySolanaBaseURL      string
	HeliusAPIKey              string
	HeliusBaseURL             string
	HeliusDataAPIBaseURL      string
	MobulaAPIKey              string
	MobulaBaseURL             string
	MobulaSmartMoneySeedsJSON string
	MobulaSmartMoneySeeds     []MobulaSmartMoneySeed
	MoralisAPIKey             string
	MoralisBaseURL            string
}

func ParseProviderEnvFromOS() (ProviderEnv, error) {
	return ParseProviderEnv(envMapFromOS())
}

func ParseProviderEnv(source map[string]string) (ProviderEnv, error) {
	worker, err := sharedconfig.ParseWorkerEnv(source)
	if err != nil {
		return ProviderEnv{}, err
	}

	duneAPIKey, err := required(source, "DUNE_API_KEY")
	if err != nil {
		return ProviderEnv{}, err
	}
	duneSeedExportJSON := strings.TrimSpace(source["DUNE_SEED_EXPORT_JSON"])
	duneSeedExportPath := strings.TrimSpace(source["DUNE_SEED_EXPORT_PATH"])
	duneSeedExportRows, err := parseDuneSeedExportRows(duneSeedExportJSON, duneSeedExportPath)
	if err != nil {
		return ProviderEnv{}, err
	}
	alchemyAPIKey, err := required(source, "ALCHEMY_API_KEY")
	if err != nil {
		return ProviderEnv{}, err
	}
	alchemyBaseURL := optional(source, "ALCHEMY_BASE_URL", defaultAlchemyBaseURL)
	alchemySolanaBaseURL := optional(source, "ALCHEMY_SOLANA_BASE_URL", defaultAlchemySolanaBaseURL)
	heliusAPIKey, err := required(source, "HELIUS_API_KEY")
	if err != nil {
		return ProviderEnv{}, err
	}
	heliusBaseURL := normalizeHeliusBaseURL(optional(source, "HELIUS_BASE_URL", defaultHeliusBaseURL))
	heliusDataAPIBaseURL := normalizeHeliusDataAPIBaseURL(optional(source, "HELIUS_DATA_API_BASE_URL", defaultHeliusDataAPIBaseURL))
	mobulaAPIKey := strings.TrimSpace(source["MOBULA_API_KEY"])
	mobulaBaseURL := optional(source, "MOBULA_BASE_URL", defaultMobulaBaseURL)
	mobulaSmartMoneySeedsJSON := strings.TrimSpace(source["QORVI_MOBULA_SMART_MONEY_SEEDS_JSON"])
	if mobulaSmartMoneySeedsJSON == "" {
		mobulaSmartMoneySeedsJSON = strings.TrimSpace(source["MOBULA_SMART_MONEY_SEEDS_JSON"])
	}
	mobulaSmartMoneySeeds, err := parseMobulaSmartMoneySeeds(mobulaSmartMoneySeedsJSON)
	if err != nil {
		return ProviderEnv{}, err
	}
	moralisAPIKey, err := required(source, "MORALIS_API_KEY")
	if err != nil {
		return ProviderEnv{}, err
	}
	moralisBaseURL := optional(source, "MORALIS_BASE_URL", defaultMoralisBaseURL)

	env := ProviderEnv{
		Worker:                    worker,
		DuneAPIKey:                duneAPIKey,
		DuneSeedExportJSON:        duneSeedExportJSON,
		DuneSeedExportPath:        duneSeedExportPath,
		DuneSeedExportRows:        duneSeedExportRows,
		AlchemyAPIKey:             alchemyAPIKey,
		AlchemyBaseURL:            alchemyBaseURL,
		AlchemySolanaBaseURL:      alchemySolanaBaseURL,
		HeliusAPIKey:              heliusAPIKey,
		HeliusBaseURL:             heliusBaseURL,
		HeliusDataAPIBaseURL:      heliusDataAPIBaseURL,
		MobulaAPIKey:              mobulaAPIKey,
		MobulaBaseURL:             mobulaBaseURL,
		MobulaSmartMoneySeedsJSON: mobulaSmartMoneySeedsJSON,
		MobulaSmartMoneySeeds:     mobulaSmartMoneySeeds,
		MoralisAPIKey:             moralisAPIKey,
		MoralisBaseURL:            moralisBaseURL,
	}

	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "DUNE_API_KEY", value: env.DuneAPIKey},
		{name: "ALCHEMY_API_KEY", value: env.AlchemyAPIKey},
		{name: "HELIUS_API_KEY", value: env.HeliusAPIKey},
		{name: "MORALIS_API_KEY", value: env.MoralisAPIKey},
	} {
		if len(field.value) < 8 {
			return ProviderEnv{}, fmt.Errorf("%s must be at least 8 characters", field.name)
		}
	}
	if env.MobulaAPIKey != "" && len(env.MobulaAPIKey) < 8 {
		return ProviderEnv{}, fmt.Errorf("MOBULA_API_KEY must be at least 8 characters")
	}
	if len(env.MobulaSmartMoneySeeds) > 0 && len(env.MobulaAPIKey) < 8 {
		return ProviderEnv{}, fmt.Errorf("MOBULA_API_KEY is required when QORVI_MOBULA_SMART_MONEY_SEEDS_JSON is set")
	}

	return env, nil
}

func required(source map[string]string, key string) (string, error) {
	value := source[key]
	if len(value) < 8 {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func optional(source map[string]string, key, fallback string) string {
	value := strings.TrimSpace(source[key])
	if value == "" {
		return fallback
	}

	return value
}

func envMapFromOS() map[string]string {
	result := make(map[string]string)
	for _, pair := range os.Environ() {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		result[key] = value
	}
	return result
}

func parseDuneSeedExportRows(rawJSON string, filePath string) ([]DuneSeedExportRow, error) {
	source := strings.TrimSpace(rawJSON)
	if source == "" && strings.TrimSpace(filePath) != "" {
		content, err := os.ReadFile(strings.TrimSpace(filePath))
		if err != nil {
			return nil, fmt.Errorf("read DUNE_SEED_EXPORT_PATH: %w", err)
		}
		source = strings.TrimSpace(string(content))
	}
	if source == "" {
		return nil, nil
	}

	var rows []DuneSeedExportRow
	if err := json.Unmarshal([]byte(source), &rows); err != nil {
		return nil, fmt.Errorf("parse dune seed export rows: %w", err)
	}

	return append([]DuneSeedExportRow(nil), rows...), nil
}

func parseMobulaSmartMoneySeeds(rawJSON string) ([]MobulaSmartMoneySeed, error) {
	source := strings.TrimSpace(rawJSON)
	if source == "" {
		return nil, nil
	}

	var seeds []MobulaSmartMoneySeed
	if err := json.Unmarshal([]byte(source), &seeds); err != nil {
		return nil, fmt.Errorf("parse Mobula smart money seeds: %w", err)
	}

	normalized := make([]MobulaSmartMoneySeed, 0, len(seeds))
	for index, seed := range seeds {
		cleaned, err := seed.Normalize()
		if err != nil {
			return nil, fmt.Errorf("normalize Mobula smart money seed %d: %w", index, err)
		}
		normalized = append(normalized, cleaned)
	}

	return normalized, nil
}

func normalizeHeliusBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultHeliusBaseURL
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return trimmed
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	if parsed.Path == "/" {
		parsed.Path = ""
	}

	return strings.TrimRight(parsed.String(), "/")
}

func normalizeHeliusDataAPIBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultHeliusDataAPIBaseURL
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return trimmed
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""

	host := parsed.Hostname()
	switch host {
	case "mainnet.helius-rpc.com":
		parsed.Host = strings.Replace(parsed.Host, "mainnet.helius-rpc.com", "api-mainnet.helius-rpc.com", 1)
	case "api.helius.xyz":
		parsed.Host = strings.Replace(parsed.Host, "api.helius.xyz", "api-mainnet.helius-rpc.com", 1)
	}

	trimmedPath := strings.TrimRight(parsed.Path, "/")
	if trimmedPath == "" || trimmedPath == "/" {
		parsed.Path = "/v0"
	} else if !strings.HasSuffix(trimmedPath, "/v0") && !strings.Contains(trimmedPath, "/v0/") {
		parsed.Path = "/v0"
	}

	return strings.TrimRight(parsed.String(), "/")
}
