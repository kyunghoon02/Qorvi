package providers

import (
	"fmt"
	"os"
	"strings"

	sharedconfig "github.com/whalegraph/whalegraph/packages/config"
)

const (
	defaultAlchemyBaseURL       = "https://eth-mainnet.g.alchemy.com"
	defaultHeliusBaseURL        = "https://mainnet.helius-rpc.com"
	defaultHeliusDataAPIBaseURL = "https://api-mainnet.helius-rpc.com/v0"
)

type ProviderEnv struct {
	Worker               sharedconfig.WorkerEnv
	DuneAPIKey           string
	AlchemyAPIKey        string
	AlchemyBaseURL       string
	HeliusAPIKey         string
	HeliusBaseURL        string
	HeliusDataAPIBaseURL string
	MoralisAPIKey        string
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
	alchemyAPIKey, err := required(source, "ALCHEMY_API_KEY")
	if err != nil {
		return ProviderEnv{}, err
	}
	alchemyBaseURL := optional(source, "ALCHEMY_BASE_URL", defaultAlchemyBaseURL)
	heliusAPIKey, err := required(source, "HELIUS_API_KEY")
	if err != nil {
		return ProviderEnv{}, err
	}
	heliusBaseURL := optional(source, "HELIUS_BASE_URL", defaultHeliusBaseURL)
	heliusDataAPIBaseURL := optional(source, "HELIUS_DATA_API_BASE_URL", defaultHeliusDataAPIBaseURL)
	moralisAPIKey, err := required(source, "MORALIS_API_KEY")
	if err != nil {
		return ProviderEnv{}, err
	}

	env := ProviderEnv{
		Worker:               worker,
		DuneAPIKey:           duneAPIKey,
		AlchemyAPIKey:        alchemyAPIKey,
		AlchemyBaseURL:       alchemyBaseURL,
		HeliusAPIKey:         heliusAPIKey,
		HeliusBaseURL:        heliusBaseURL,
		HeliusDataAPIBaseURL: heliusDataAPIBaseURL,
		MoralisAPIKey:        moralisAPIKey,
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
