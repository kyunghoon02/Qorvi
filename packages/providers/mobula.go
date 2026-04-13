package providers

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

const (
	defaultMobulaSeedLimit    = 25
	defaultMobulaSeedWindow   = 30 * 24 * time.Hour
	defaultMobulaHTTPTimeout  = 30 * time.Second
	mobulaQueryLabelKey       = "mobula_label"
	mobulaQuerySeedKeyKey     = "mobula_seed_key"
	mobulaQueryTokenSymbolKey = "mobula_token_symbol"
)

var defaultMobulaSeedLabels = []string{"smartTrader", "proTrader"}

type MobulaSmartMoneySeed struct {
	Blockchain   string   `json:"blockchain"`
	Chain        string   `json:"chain"`
	Address      string   `json:"address"`
	TokenAddress string   `json:"tokenAddress"`
	TokenSymbol  string   `json:"tokenSymbol"`
	Labels       []string `json:"labels"`
	Limit        int      `json:"limit"`
}

type MobulaAdapter struct {
	Credentials ProviderCredentials
	Seeds       []MobulaSmartMoneySeed
	http        jsonHTTPClient
}

type mobulaTraderPositionsResponse struct {
	Data       []mobulaTraderPosition `json:"data"`
	TotalCount int                    `json:"totalCount"`
}

type mobulaTraderPosition struct {
	ChainID          string         `json:"chainId"`
	WalletAddress    string         `json:"walletAddress"`
	TokenAddress     string         `json:"tokenAddress"`
	TotalPnlUSD      string         `json:"totalPnlUSD"`
	TokenAmountUSD   string         `json:"tokenAmountUSD"`
	VolumeBuyUSD     string         `json:"volumeBuyUSD"`
	VolumeSellUSD    string         `json:"volumeSellUSD"`
	WalletFundAt     string         `json:"walletFundAt"`
	LastActivityAt   string         `json:"lastActivityAt"`
	FirstTradeAt     string         `json:"firstTradeAt"`
	LastTradeAt      string         `json:"lastTradeAt"`
	TotalFeesPaidUSD string         `json:"totalFeesPaidUSD"`
	Labels           []string       `json:"labels"`
	WalletMetadata   map[string]any `json:"walletMetadata"`
	FundingInfo      map[string]any `json:"fundingInfo"`
	Platform         map[string]any `json:"platform"`
}

type SeedDiscoveryBatchSource interface {
	SeedDiscoveryBatches(reference time.Time) []SeedDiscoveryBatch
}

func NewMobulaAdapter(credentials ProviderCredentials, seeds []MobulaSmartMoneySeed, client *http.Client) MobulaAdapter {
	baseURL := strings.TrimSpace(credentials.BaseURL)
	if baseURL == "" {
		baseURL = defaultMobulaBaseURL
	}
	credentials.BaseURL = strings.TrimRight(baseURL, "/")
	if client == nil {
		client = &http.Client{Timeout: defaultMobulaHTTPTimeout}
	}
	return MobulaAdapter{
		Credentials: credentials,
		Seeds:       append([]MobulaSmartMoneySeed(nil), seeds...),
		http:        newJSONHTTPClient(client),
	}
}

func (a MobulaAdapter) Name() ProviderName { return ProviderMobula }
func (a MobulaAdapter) Kind() AdapterKind  { return AdapterHistorical }

func (a MobulaAdapter) FetchWalletActivity(ctx ProviderRequestContext) ([]ProviderWalletActivity, error) {
	return []ProviderWalletActivity{
		CreateProviderActivityFixture(ProviderActivityFixtureInput{
			Provider:      a.Name(),
			Chain:         ctx.Chain,
			WalletAddress: ctx.WalletAddress,
			SourceID:      "mobula_wallet_labels_v0",
			Kind:          "label",
			Confidence:    0.9,
			Metadata: map[string]any{
				"label": "smartTrader",
			},
		}),
	}, nil
}

func (a MobulaAdapter) SeedDiscoveryBatches(reference time.Time) []SeedDiscoveryBatch {
	if len(a.Seeds) == 0 {
		return nil
	}

	windowEnd := reference.UTC()
	windowStart := windowEnd.Add(-defaultMobulaSeedWindow)
	batches := make([]SeedDiscoveryBatch, 0, len(a.Seeds)*2)

	for _, seed := range a.Seeds {
		chain, ok := seed.DomainChain()
		if !ok {
			continue
		}
		for _, label := range seed.NormalizedLabels() {
			batches = append(batches, SeedDiscoveryBatch{
				Provider: ProviderMobula,
				Request: ProviderRequestContext{
					Chain:         chain,
					WalletAddress: seed.NormalizedAddress(),
					Access: domain.AccessContext{
						Role: domain.RoleOperator,
						Plan: domain.PlanPro,
					},
				},
				WindowStart: windowStart,
				WindowEnd:   windowEnd,
				Limit:       seed.NormalizedLimit(),
				Metadata: map[string]any{
					mobulaQueryLabelKey:       label,
					mobulaQuerySeedKeyKey:     seed.Key(),
					mobulaQueryTokenSymbolKey: strings.TrimSpace(seed.TokenSymbol),
					"mobula_blockchain":       seed.NormalizedBlockchain(),
				},
			})
		}
	}

	return batches
}

func (a MobulaAdapter) FetchSeedDiscoveryCandidates(batch SeedDiscoveryBatch) ([]SeedDiscoveryCandidate, error) {
	if err := batch.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Credentials.APIKey) == "" {
		return nil, fmt.Errorf("mobula API key is required")
	}

	queryURL, err := a.buildTraderPositionsURL(batch)
	if err != nil {
		return nil, err
	}

	req, err := newJSONRequest(http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", a.Credentials.APIKey)

	var response mobulaTraderPositionsResponse
	if err := a.http.doJSONRequest(req, &response); err != nil {
		return nil, err
	}

	candidates := make([]SeedDiscoveryCandidate, 0, len(response.Data))
	for _, entry := range response.Data {
		candidate, ok := buildMobulaSeedDiscoveryCandidate(batch, entry)
		if !ok {
			continue
		}
		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

func (a MobulaAdapter) buildTraderPositionsURL(batch SeedDiscoveryBatch) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(a.Credentials.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultMobulaBaseURL
	}

	parsed, err := url.Parse(baseURL + "/api/2/token/trader-positions")
	if err != nil {
		return "", fmt.Errorf("parse Mobula base URL: %w", err)
	}

	values := parsed.Query()
	values.Set("blockchain", mobulaBlockchainForChain(batch.Request.Chain))
	values.Set("address", strings.TrimSpace(batch.Request.WalletAddress))
	values.Set("limit", strconv.Itoa(batch.Limit))
	if label := strings.TrimSpace(stringMetadata(batch.Metadata, mobulaQueryLabelKey)); label != "" {
		values.Set("label", label)
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func buildMobulaSeedDiscoveryCandidate(batch SeedDiscoveryBatch, entry mobulaTraderPosition) (SeedDiscoveryCandidate, bool) {
	walletAddress := strings.TrimSpace(entry.WalletAddress)
	if walletAddress == "" {
		return SeedDiscoveryCandidate{}, false
	}

	observedAt := parseMobulaObservedAt(batch.WindowEnd, entry.LastActivityAt, entry.LastTradeAt, entry.WalletFundAt)
	if observedAt.Before(batch.WindowStart) {
		return SeedDiscoveryCandidate{}, false
	}

	label := strings.TrimSpace(stringMetadata(batch.Metadata, mobulaQueryLabelKey))
	seedKey := strings.TrimSpace(stringMetadata(batch.Metadata, mobulaQuerySeedKeyKey))
	sourceID := fmt.Sprintf("mobula:%s:%s", batch.Request.Chain, strings.ToLower(walletAddress))

	metadata := map[string]any{
		"seed_label":              label,
		"seed_label_reason":       "mobula_token_trader_positions",
		"seed_label_source_id":    seedKey,
		"mobula_chain_id":         strings.TrimSpace(entry.ChainID),
		"mobula_token_address":    strings.TrimSpace(entry.TokenAddress),
		"mobula_seed_key":         seedKey,
		"mobula_seed_label":       label,
		"mobula_labels":           append([]string(nil), entry.Labels...),
		"mobula_total_pnl_usd":    strings.TrimSpace(entry.TotalPnlUSD),
		"mobula_token_amount_usd": strings.TrimSpace(entry.TokenAmountUSD),
		"mobula_volume_buy_usd":   strings.TrimSpace(entry.VolumeBuyUSD),
		"mobula_volume_sell_usd":  strings.TrimSpace(entry.VolumeSellUSD),
		"mobula_total_fees_usd":   strings.TrimSpace(entry.TotalFeesPaidUSD),
		"mobula_wallet_metadata":  cloneAnyMap(entry.WalletMetadata),
		"mobula_platform":         cloneAnyMap(entry.Platform),
		"mobula_funding_info":     cloneAnyMap(entry.FundingInfo),
		"mobula_wallet_funded_at": strings.TrimSpace(entry.WalletFundAt),
		"mobula_first_trade_at":   strings.TrimSpace(entry.FirstTradeAt),
		"mobula_last_trade_at":    strings.TrimSpace(entry.LastTradeAt),
		"mobula_token_symbol":     strings.TrimSpace(stringMetadata(batch.Metadata, mobulaQueryTokenSymbolKey)),
		"mobula_query_blockchain": mobulaBlockchainForChain(batch.Request.Chain),
		"mobula_query_token":      strings.TrimSpace(batch.Request.WalletAddress),
		"mobula_total_count_hint": 1,
	}

	return CreateSeedDiscoveryCandidateFixture(batch, SeedDiscoveryCandidateInput{
		Provider:      ProviderMobula,
		Chain:         batch.Request.Chain,
		WalletAddress: walletAddress,
		SourceID:      sourceID,
		Kind:          "smart_money_label",
		Confidence:    mobulaLabelConfidence(label, entry.Labels),
		ObservedAt:    observedAt,
		Metadata:      metadata,
	}), true
}

func (s MobulaSmartMoneySeed) Normalize() (MobulaSmartMoneySeed, error) {
	blockchain := s.NormalizedBlockchain()
	if blockchain == "" {
		return MobulaSmartMoneySeed{}, fmt.Errorf("blockchain is required")
	}
	address := s.NormalizedAddress()
	if address == "" {
		return MobulaSmartMoneySeed{}, fmt.Errorf("address is required")
	}

	labels := s.NormalizedLabels()
	return MobulaSmartMoneySeed{
		Blockchain:  blockchain,
		Address:     address,
		TokenSymbol: strings.TrimSpace(s.TokenSymbol),
		Labels:      labels,
		Limit:       s.NormalizedLimit(),
	}, nil
}

func (s MobulaSmartMoneySeed) NormalizedBlockchain() string {
	for _, candidate := range []string{s.Blockchain, s.Chain} {
		switch strings.ToLower(strings.TrimSpace(candidate)) {
		case "ethereum", "eth", "evm", "evm:1":
			return "ethereum"
		case "solana", "sol":
			return "solana"
		}
	}
	return ""
}

func (s MobulaSmartMoneySeed) DomainChain() (domain.Chain, bool) {
	switch s.NormalizedBlockchain() {
	case "ethereum":
		return domain.ChainEVM, true
	case "solana":
		return domain.ChainSolana, true
	default:
		return "", false
	}
}

func (s MobulaSmartMoneySeed) NormalizedAddress() string {
	for _, candidate := range []string{s.Address, s.TokenAddress} {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (s MobulaSmartMoneySeed) NormalizedLabels() []string {
	rawLabels := s.Labels
	if len(rawLabels) == 0 {
		rawLabels = defaultMobulaSeedLabels
	}
	labels := make([]string, 0, len(rawLabels))
	seen := map[string]struct{}{}
	for _, raw := range rawLabels {
		label := normalizeMobulaLabel(raw)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}
	if len(labels) == 0 {
		labels = append(labels, defaultMobulaSeedLabels...)
	}
	slices.Sort(labels)
	return labels
}

func (s MobulaSmartMoneySeed) NormalizedLimit() int {
	if s.Limit <= 0 {
		return defaultMobulaSeedLimit
	}
	if s.Limit > 1000 {
		return 1000
	}
	return s.Limit
}

func (s MobulaSmartMoneySeed) Key() string {
	tokenSymbol := strings.ToLower(strings.TrimSpace(s.TokenSymbol))
	if tokenSymbol == "" {
		tokenSymbol = strings.ToLower(s.NormalizedAddress())
	}
	return fmt.Sprintf("%s:%s", s.NormalizedBlockchain(), tokenSymbol)
}

func mobulaBlockchainForChain(chain domain.Chain) string {
	switch chain {
	case domain.ChainSolana:
		return "solana"
	default:
		return "ethereum"
	}
}

func normalizeMobulaLabel(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "smarttrader":
		return "smartTrader"
	case "protrader":
		return "proTrader"
	case "sniper":
		return "sniper"
	case "insider":
		return "insider"
	case "bundler":
		return "bundler"
	case "freshtrader":
		return "freshTrader"
	case "dev":
		return "dev"
	case "liquiditypool":
		return "liquidityPool"
	default:
		return ""
	}
}

func mobulaLabelConfidence(requested string, responseLabels []string) float64 {
	for _, label := range append([]string{requested}, responseLabels...) {
		switch normalizeMobulaLabel(label) {
		case "smartTrader":
			return 0.94
		case "proTrader":
			return 0.86
		case "insider":
			return 0.82
		case "sniper":
			return 0.72
		case "freshTrader":
			return 0.64
		case "bundler":
			return 0.58
		case "dev":
			return 0.52
		case "liquidityPool":
			return 0.4
		}
	}
	return 0.6
}

func parseMobulaObservedAt(fallback time.Time, rawValues ...string) time.Time {
	for _, raw := range rawValues {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, value)
		if err == nil {
			return parsed.UTC()
		}
	}
	return fallback.UTC()
}

func stringMetadata(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, _ := metadata[key].(string)
	return strings.TrimSpace(value)
}

func cloneAnyMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
