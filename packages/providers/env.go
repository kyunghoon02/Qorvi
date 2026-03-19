package providers

import (
	"fmt"

	sharedconfig "github.com/whalegraph/whalegraph/packages/config"
)

type ProviderEnv struct {
	Worker        sharedconfig.WorkerEnv
	DuneAPIKey    string
	AlchemyAPIKey string
	HeliusAPIKey  string
	MoralisAPIKey string
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
	heliusAPIKey, err := required(source, "HELIUS_API_KEY")
	if err != nil {
		return ProviderEnv{}, err
	}
	moralisAPIKey, err := required(source, "MORALIS_API_KEY")
	if err != nil {
		return ProviderEnv{}, err
	}

	env := ProviderEnv{
		Worker:        worker,
		DuneAPIKey:    duneAPIKey,
		AlchemyAPIKey: alchemyAPIKey,
		HeliusAPIKey:  heliusAPIKey,
		MoralisAPIKey: moralisAPIKey,
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
