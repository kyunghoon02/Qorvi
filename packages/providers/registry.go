package providers

type Registry map[ProviderName]ProviderAdapter

func DefaultRegistry() Registry {
	return Registry{
		ProviderDune:    DuneAdapter{},
		ProviderAlchemy: AlchemyAdapter{},
		ProviderHelius:  HeliusAdapter{},
		ProviderMoralis: MoralisAdapter{},
	}
}

func NewConfiguredRegistry(env ProviderEnv) Registry {
	return Registry{
		ProviderDune:    DuneAdapter{},
		ProviderAlchemy: NewAlchemyAdapter(ProviderCredentials{Provider: ProviderAlchemy, APIKey: env.AlchemyAPIKey, BaseURL: env.AlchemyBaseURL}),
		ProviderHelius:  NewHeliusAdapter(ProviderCredentials{Provider: ProviderHelius, APIKey: env.HeliusAPIKey, BaseURL: env.HeliusBaseURL, DataAPIBaseURL: env.HeliusDataAPIBaseURL}),
		ProviderMoralis: MoralisAdapter{},
	}
}

func NewConfiguredRegistryFromOS() (Registry, error) {
	env, err := ParseProviderEnvFromOS()
	if err != nil {
		return nil, err
	}

	return NewConfiguredRegistry(env), nil
}
