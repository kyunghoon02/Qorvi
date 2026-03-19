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
