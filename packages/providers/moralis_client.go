package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

const DefaultMoralisBaseURL = defaultMoralisBaseURL

type MoralisWalletEnrichment struct {
	NetWorthUSD            string
	NativeBalance          string
	NativeBalanceFormatted string
	ActiveChains           []string
	Holdings               []domain.WalletHolding
	ObservedAt             time.Time
	MetadataOverlay        map[string]any
}

func (e MoralisWalletEnrichment) Metadata() map[string]any {
	return mergeMetadata(
		map[string]any{
			"moralis_net_worth_usd":             e.NetWorthUSD,
			"moralis_native_balance":            e.NativeBalance,
			"moralis_native_balance_formatted":  e.NativeBalanceFormatted,
			"moralis_active_chains":             append([]string(nil), e.ActiveChains...),
			"moralis_active_chain_count":        len(e.ActiveChains),
			"moralis_holdings":                  append([]domain.WalletHolding(nil), e.Holdings...),
			"moralis_holding_count":             len(e.Holdings),
			"moralis_enrichment_observed_at":    e.ObservedAt.UTC().Format(time.RFC3339),
			"moralis_enrichment_provider":       string(ProviderMoralis),
			"moralis_enrichment_schema_version": 1,
		},
		cloneMetadata(e.MetadataOverlay),
	)
}

func (e MoralisWalletEnrichment) ToDomain(source string) *domain.WalletEnrichment {
	if strings.TrimSpace(e.NetWorthUSD) == "" && len(e.ActiveChains) == 0 && len(e.Holdings) == 0 {
		return nil
	}

	return &domain.WalletEnrichment{
		Provider:               string(ProviderMoralis),
		NetWorthUSD:            e.NetWorthUSD,
		NativeBalance:          e.NativeBalance,
		NativeBalanceFormatted: e.NativeBalanceFormatted,
		ActiveChains:           append([]string(nil), e.ActiveChains...),
		ActiveChainCount:       len(e.ActiveChains),
		Holdings:               append([]domain.WalletHolding(nil), e.Holdings...),
		HoldingCount:           len(e.Holdings),
		Source:                 source,
		UpdatedAt:              e.ObservedAt.UTC().Format(time.RFC3339),
	}
}

type MoralisClient struct {
	baseURL string
	apiKey  string
	http    jsonHTTPClient
}

func NewMoralisClient(credentials ProviderCredentials, client *http.Client) *MoralisClient {
	baseURL := strings.TrimSpace(credentials.BaseURL)
	if baseURL == "" {
		baseURL = DefaultMoralisBaseURL
	}

	return &MoralisClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  strings.TrimSpace(credentials.APIKey),
		http:    newJSONHTTPClient(client),
	}
}

func (c *MoralisClient) FetchWalletEnrichment(
	ctx context.Context,
	request ProviderRequestContext,
) (MoralisWalletEnrichment, error) {
	if c == nil {
		return MoralisWalletEnrichment{}, fmt.Errorf("moralis client is nil")
	}
	if request.Chain != domain.ChainEVM {
		return MoralisWalletEnrichment{}, nil
	}
	if strings.TrimSpace(request.WalletAddress) == "" {
		return MoralisWalletEnrichment{}, fmt.Errorf("wallet address is required")
	}

	observedAt := time.Now().UTC()
	netWorth, netWorthMetadata, netWorthErr := c.fetchWalletNetWorth(
		ctx,
		request.WalletAddress,
		observedAt,
	)
	activeChains, chainsMetadata, chainsErr := c.fetchWalletChains(
		ctx,
		request.WalletAddress,
		observedAt,
	)
	holdings, holdingsMetadata, holdingsErr := c.fetchWalletHoldings(
		ctx,
		request.WalletAddress,
		observedAt,
	)
	if netWorthErr != nil && chainsErr != nil && holdingsErr != nil {
		return MoralisWalletEnrichment{}, fmt.Errorf(
			"moralis enrichment unavailable: net worth: %w; chains: %v; holdings: %v",
			netWorthErr,
			chainsErr,
			holdingsErr,
		)
	}

	failedScopes := make([]string, 0, 3)
	if netWorthErr != nil {
		failedScopes = append(failedScopes, "net-worth")
	}
	if chainsErr != nil {
		failedScopes = append(failedScopes, "chains")
	}
	if holdingsErr != nil {
		failedScopes = append(failedScopes, "holdings")
	}

	return MoralisWalletEnrichment{
		NetWorthUSD:            netWorth.NetWorthUSD,
		NativeBalance:          netWorth.NativeBalance,
		NativeBalanceFormatted: netWorth.NativeBalanceFormatted,
		ActiveChains:           activeChains,
		Holdings:               holdings,
		ObservedAt:             observedAt,
		MetadataOverlay: mergeMetadata(
			mergeMetadata(mergeMetadata(netWorthMetadata, chainsMetadata), holdingsMetadata),
			buildMoralisPartialFailureMetadata(failedScopes),
		),
	}, nil
}

func buildMoralisPartialFailureMetadata(failedScopes []string) map[string]any {
	if len(failedScopes) == 0 {
		return nil
	}

	return map[string]any{
		"moralis_partial_failure":        true,
		"moralis_partial_failure_scopes": append([]string(nil), failedScopes...),
	}
}

type moralisWalletNetWorthSnapshot struct {
	NetWorthUSD            string
	NativeBalance          string
	NativeBalanceFormatted string
}

func (c *MoralisClient) fetchWalletNetWorth(
	ctx context.Context,
	address string,
	observedAt time.Time,
) (moralisWalletNetWorthSnapshot, map[string]any, error) {
	endpoint, err := c.netWorthEndpoint(address)
	if err != nil {
		return moralisWalletNetWorthSnapshot{}, nil, err
	}

	req, err := newJSONRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return moralisWalletNetWorthSnapshot{}, nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	rawBody, err := c.http.doJSONRequestWithRaw(req, nil)
	if err != nil {
		return moralisWalletNetWorthSnapshot{}, nil, err
	}

	snapshot, err := parseMoralisWalletNetWorth(rawBody)
	if err != nil {
		return moralisWalletNetWorthSnapshot{}, nil, err
	}

	metadata := prefixMetadataKeys(
		capturePagePayloadMetadata(
			ProviderMoralis,
			"wallet-net-worth",
			observedAt,
			address,
			rawBody,
			map[string]any{
				"endpoint": c.baseURL,
			},
		),
		"moralis_net_worth_",
	)

	return snapshot, metadata, nil
}

func (c *MoralisClient) fetchWalletChains(
	ctx context.Context,
	address string,
	observedAt time.Time,
) ([]string, map[string]any, error) {
	endpoint, err := c.walletChainsEndpoint(address)
	if err != nil {
		return nil, nil, err
	}

	req, err := newJSONRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	rawBody, err := c.http.doJSONRequestWithRaw(req, nil)
	if err != nil {
		return nil, nil, err
	}

	chains, err := parseMoralisWalletChains(rawBody)
	if err != nil {
		return nil, nil, err
	}

	metadata := prefixMetadataKeys(
		capturePagePayloadMetadata(
			ProviderMoralis,
			"wallet-chains",
			observedAt,
			address,
			rawBody,
			map[string]any{
				"endpoint": c.baseURL,
				"count":    len(chains),
			},
		),
		"moralis_wallet_chains_",
	)

	return chains, metadata, nil
}

func (c *MoralisClient) fetchWalletHoldings(
	ctx context.Context,
	address string,
	observedAt time.Time,
) ([]domain.WalletHolding, map[string]any, error) {
	endpoint, err := c.walletHoldingsEndpoint(address)
	if err != nil {
		return nil, nil, err
	}

	req, err := newJSONRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	rawBody, err := c.http.doJSONRequestWithRaw(req, nil)
	if err != nil {
		return nil, nil, err
	}

	holdings, err := parseMoralisWalletHoldings(rawBody)
	if err != nil {
		return nil, nil, err
	}

	metadata := prefixMetadataKeys(
		capturePagePayloadMetadata(
			ProviderMoralis,
			"wallet-holdings",
			observedAt,
			address,
			rawBody,
			map[string]any{
				"endpoint": c.baseURL,
				"count":    len(holdings),
			},
		),
		"moralis_wallet_holdings_",
	)

	return holdings, metadata, nil
}

func (c *MoralisClient) netWorthEndpoint(address string) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse moralis base url: %w", err)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/wallets/" + url.PathEscape(address) + "/net-worth"
	query := base.Query()
	query.Set("exclude_spam", "true")
	query.Set("exclude_unverified_contracts", "true")
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func (c *MoralisClient) walletChainsEndpoint(address string) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse moralis base url: %w", err)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/wallets/" + url.PathEscape(address) + "/chains"
	return base.String(), nil
}

func (c *MoralisClient) walletHoldingsEndpoint(address string) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse moralis base url: %w", err)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/wallets/" + url.PathEscape(address) + "/tokens"
	query := base.Query()
	query.Set("exclude_spam", "true")
	query.Set("exclude_unverified_contracts", "true")
	query.Set("limit", "5")
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func parseMoralisWalletNetWorth(raw []byte) (moralisWalletNetWorthSnapshot, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return moralisWalletNetWorthSnapshot{}, fmt.Errorf("decode moralis wallet net worth: %w", err)
	}

	snapshot := moralisWalletNetWorthSnapshot{
		NetWorthUSD: firstProviderString(
			payload["total_networth_usd"],
			payload["totalNetworthUsd"],
			payload["networth_usd"],
			payload["netWorthUsd"],
		),
		NativeBalance: firstProviderString(
			payload["native_balance"],
			payload["nativeBalance"],
		),
		NativeBalanceFormatted: firstProviderString(
			payload["native_balance_formatted"],
			payload["nativeBalanceFormatted"],
		),
	}

	if chains, ok := payload["chains"].([]any); ok {
		for _, chainValue := range chains {
			chainMap, ok := chainValue.(map[string]any)
			if !ok {
				continue
			}
			if snapshot.NativeBalance == "" {
				snapshot.NativeBalance = firstProviderString(
					chainMap["native_balance"],
					chainMap["nativeBalance"],
				)
			}
			if snapshot.NativeBalanceFormatted == "" {
				snapshot.NativeBalanceFormatted = firstProviderString(
					chainMap["native_balance_formatted"],
					chainMap["nativeBalanceFormatted"],
				)
			}
			if snapshot.NativeBalance != "" && snapshot.NativeBalanceFormatted != "" {
				break
			}
		}
	}

	return snapshot, nil
}

func parseMoralisWalletChains(raw []byte) ([]string, error) {
	type walletChainsEnvelope struct {
		Result []map[string]any `json:"result"`
	}

	var envelope walletChainsEnvelope
	if err := json.Unmarshal(raw, &envelope); err == nil && len(envelope.Result) > 0 {
		return collectMoralisChainLabels(envelope.Result), nil
	}

	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("decode moralis wallet chains: %w", err)
	}

	return collectMoralisChainLabels(items), nil
}

func parseMoralisWalletHoldings(raw []byte) ([]domain.WalletHolding, error) {
	type moralisWalletHoldingsEnvelope struct {
		Result []map[string]any `json:"result"`
	}

	var envelope moralisWalletHoldingsEnvelope
	if err := json.Unmarshal(raw, &envelope); err == nil {
		if len(envelope.Result) > 0 {
			return collectMoralisWalletHoldings(envelope.Result), nil
		}
	}

	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("decode moralis wallet holdings: %w", err)
	}

	return collectMoralisWalletHoldings(items), nil
}

func collectMoralisWalletHoldings(items []map[string]any) []domain.WalletHolding {
	holdings := make([]domain.WalletHolding, 0, len(items))
	for _, item := range items {
		symbol := firstProviderString(item["symbol"], item["ticker"])
		balanceFormatted := firstProviderString(
			item["balance_formatted"],
			item["balanceFormatted"],
		)
		balance := firstProviderString(item["balance"])
		valueUSD := firstProviderString(item["usd_value"], item["usdValue"])
		tokenAddress := firstProviderString(item["token_address"], item["tokenAddress"])
		if symbol == "" && tokenAddress == "" {
			continue
		}

		holdings = append(holdings, domain.WalletHolding{
			Symbol:              symbol,
			TokenAddress:        tokenAddress,
			Balance:             balance,
			BalanceFormatted:    balanceFormatted,
			ValueUSD:            valueUSD,
			PortfolioPercentage: firstProviderFloat64(item["portfolio_percentage"], item["portfolioPercentage"]),
			IsNative:            firstProviderBool(item["native_token"], item["nativeToken"]),
		})
	}

	return holdings
}

func collectMoralisChainLabels(items []map[string]any) []string {
	if len(items) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(items))
	labels := make([]string, 0, len(items))
	for _, item := range items {
		label := normalizeMoralisChainLabel(
			firstProviderString(
				item["chain_name"],
				item["chainName"],
				item["name"],
				item["chain"],
			),
		)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}

	return labels
}

func firstProviderString(values ...any) string {
	for _, value := range values {
		next := stringifyProviderValue(value)
		if next != "" {
			return next
		}
	}

	return ""
}

func stringifyProviderValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return strings.TrimSpace(typed.String())
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%g", typed))
	case float32:
		return strings.TrimSpace(fmt.Sprintf("%g", typed))
	case int:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case int64:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case int32:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func firstProviderFloat64(values ...any) float64 {
	for _, value := range values {
		switch typed := value.(type) {
		case nil:
			continue
		case float64:
			return typed
		case float32:
			return float64(typed)
		case int:
			return float64(typed)
		case int32:
			return float64(typed)
		case int64:
			return float64(typed)
		case json.Number:
			if parsed, err := typed.Float64(); err == nil {
				return parsed
			}
		case string:
			if parsed, err := json.Number(strings.TrimSpace(typed)).Float64(); err == nil {
				return parsed
			}
		}
	}

	return 0
}

func firstProviderBool(values ...any) bool {
	for _, value := range values {
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			switch strings.ToLower(strings.TrimSpace(typed)) {
			case "true", "1":
				return true
			case "false", "0":
				return false
			}
		}
	}

	return false
}

func normalizeMoralisChainLabel(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "":
		return ""
	case "eth", "ethereum":
		return "Ethereum"
	case "base":
		return "Base"
	case "arbitrum":
		return "Arbitrum"
	case "optimism":
		return "Optimism"
	case "polygon":
		return "Polygon"
	case "blast":
		return "Blast"
	case "bsc":
		return "BNB Chain"
	default:
		if len(value) == 0 {
			return ""
		}
		return strings.TrimSpace(value)
	}
}
