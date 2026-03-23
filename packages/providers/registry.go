package providers

type Registry map[ProviderName]ProviderAdapter

func DefaultRegistry() Registry {
	return Registry{
		ProviderDune:    NewDuneAdapter(nil),
		ProviderAlchemy: AlchemyAdapter{},
		ProviderHelius:  HeliusAdapter{},
		ProviderMoralis: MoralisAdapter{},
	}
}

func NewConfiguredRegistry(env ProviderEnv) Registry {
	return Registry{
		ProviderDune: NewDuneAdapter(env.DuneSeedExportRows),
		ProviderAlchemy: NewAlchemyAdapter(ProviderCredentials{
			Provider:      ProviderAlchemy,
			APIKey:        env.AlchemyAPIKey,
			BaseURL:       env.AlchemyBaseURL,
			SolanaBaseURL: env.AlchemySolanaBaseURL,
		}),
		ProviderHelius: NewHeliusAdapter(ProviderCredentials{
			Provider:        ProviderHelius,
			APIKey:          env.HeliusAPIKey,
			BaseURL:         env.HeliusBaseURL,
			DataAPIBaseURL:  env.HeliusDataAPIBaseURL,
			FallbackAPIKey:  env.AlchemyAPIKey,
			FallbackBaseURL: env.AlchemySolanaBaseURL,
		}),
		ProviderMoralis: NewMoralisAdapter(ProviderCredentials{Provider: ProviderMoralis, APIKey: env.MoralisAPIKey, BaseURL: env.MoralisBaseURL}),
	}
}

func NewConfiguredRegistryFromOS() (Registry, error) {
	env, err := ParseProviderEnvFromOS()
	if err != nil {
		return nil, err
	}

	return NewConfiguredRegistry(env), nil
}
