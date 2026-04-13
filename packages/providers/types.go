package providers

import (
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

type ProviderName string

const (
	ProviderDune    ProviderName = "dune"
	ProviderAlchemy ProviderName = "alchemy"
	ProviderHelius  ProviderName = "helius"
	ProviderMobula  ProviderName = "mobula"
	ProviderMoralis ProviderName = "moralis"
)

type AdapterKind string

const (
	AdapterHistorical AdapterKind = "historical"
	AdapterRealtime   AdapterKind = "realtime"
	AdapterHybrid     AdapterKind = "hybrid"
)

type ProviderCredentials struct {
	Provider        ProviderName
	APIKey          string
	BaseURL         string
	DataAPIBaseURL  string
	SolanaBaseURL   string
	FallbackAPIKey  string
	FallbackBaseURL string
}

type ProviderRequestContext struct {
	Chain         domain.Chain
	WalletAddress string
	Access        domain.AccessContext
}

type ProviderWalletActivity struct {
	Provider      ProviderName
	Chain         domain.Chain
	WalletAddress string
	SourceID      string
	ObservedAt    time.Time
	Kind          string
	Confidence    float64
	Metadata      map[string]any
}

type ProviderAdapter interface {
	Name() ProviderName
	Kind() AdapterKind
	FetchWalletActivity(ctx ProviderRequestContext) ([]ProviderWalletActivity, error)
}
