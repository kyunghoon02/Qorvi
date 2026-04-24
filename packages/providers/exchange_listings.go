package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultUpbitPublicBaseURL   = "https://api.upbit.com"
	defaultBithumbPublicBaseURL = "https://api.bithumb.com"
	defaultExchangeHTTPTimeout  = 15 * time.Second
)

type ExchangeName string

const (
	ExchangeUpbit   ExchangeName = "upbit"
	ExchangeBithumb ExchangeName = "bithumb"
)

type ExchangeListing struct {
	Exchange           ExchangeName
	Market             string
	BaseSymbol         string
	QuoteSymbol        string
	DisplayName        string
	MarketWarning      string
	NormalizedAssetKey string
	TokenAddress       string
	ChainHint          string
	Metadata           map[string]any
}

type ExchangeListingClient struct {
	baseURL string
	http    jsonHTTPClient
}

type upbitMarketResponse struct {
	Market        string         `json:"market"`
	KoreanName    string         `json:"korean_name"`
	EnglishName   string         `json:"english_name"`
	MarketWarning string         `json:"market_warning"`
	MarketEvent   map[string]any `json:"market_event"`
}

type bithumbMarketResponse struct {
	Market        string `json:"market"`
	KoreanName    string `json:"korean_name"`
	EnglishName   string `json:"english_name"`
	MarketWarning string `json:"market_warning"`
}

func NewUpbitExchangeListingClient(baseURL string, client *http.Client) *ExchangeListingClient {
	return newExchangeListingClient(baseURL, defaultUpbitPublicBaseURL, client)
}

func NewBithumbExchangeListingClient(baseURL string, client *http.Client) *ExchangeListingClient {
	return newExchangeListingClient(baseURL, defaultBithumbPublicBaseURL, client)
}

func newExchangeListingClient(baseURL, fallback string, client *http.Client) *ExchangeListingClient {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		trimmed = fallback
	}
	if client == nil {
		client = &http.Client{Timeout: defaultExchangeHTTPTimeout}
	}
	return &ExchangeListingClient{
		baseURL: trimmed,
		http:    newJSONHTTPClient(client),
	}
}

func (c *ExchangeListingClient) FetchUpbitListings(ctx context.Context) ([]ExchangeListing, error) {
	if c == nil {
		return nil, fmt.Errorf("upbit exchange listing client is nil")
	}

	reqURL, err := url.Parse(c.baseURL + "/v1/market/all")
	if err != nil {
		return nil, fmt.Errorf("parse upbit listing url: %w", err)
	}
	query := reqURL.Query()
	query.Set("is_details", "true")
	reqURL.RawQuery = query.Encode()

	req, err := newJSONRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	var response []upbitMarketResponse
	if err := c.http.doJSONRequest(req, &response); err != nil {
		return nil, err
	}

	listings := make([]ExchangeListing, 0, len(response))
	for _, item := range response {
		listing, ok := normalizeExchangeListing(
			ExchangeUpbit,
			item.Market,
			item.EnglishName,
			item.KoreanName,
			item.MarketWarning,
			map[string]any{
				"korean_name":  item.KoreanName,
				"english_name": item.EnglishName,
				"market_event": item.MarketEvent,
			},
		)
		if !ok {
			continue
		}
		listings = append(listings, listing)
	}

	return listings, nil
}

func (c *ExchangeListingClient) FetchBithumbListings(ctx context.Context) ([]ExchangeListing, error) {
	if c == nil {
		return nil, fmt.Errorf("bithumb exchange listing client is nil")
	}

	reqURL, err := url.Parse(c.baseURL + "/v1/market/all")
	if err != nil {
		return nil, fmt.Errorf("parse bithumb listing url: %w", err)
	}
	query := reqURL.Query()
	query.Set("isDetails", "true")
	reqURL.RawQuery = query.Encode()

	req, err := newJSONRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	var response []bithumbMarketResponse
	if err := c.http.doJSONRequest(req, &response); err != nil {
		return nil, err
	}

	listings := make([]ExchangeListing, 0, len(response))
	for _, item := range response {
		listing, ok := normalizeExchangeListing(
			ExchangeBithumb,
			item.Market,
			item.EnglishName,
			item.KoreanName,
			item.MarketWarning,
			map[string]any{
				"korean_name":  item.KoreanName,
				"english_name": item.EnglishName,
			},
		)
		if !ok {
			continue
		}
		listings = append(listings, listing)
	}

	return listings, nil
}

func normalizeExchangeListing(
	exchange ExchangeName,
	market string,
	englishName string,
	koreanName string,
	marketWarning string,
	metadata map[string]any,
) (ExchangeListing, bool) {
	market = strings.TrimSpace(market)
	parts := strings.Split(market, "-")
	if len(parts) != 2 {
		return ExchangeListing{}, false
	}

	quote := strings.ToUpper(strings.TrimSpace(parts[0]))
	base := strings.ToUpper(strings.TrimSpace(parts[1]))
	if quote == "" || base == "" {
		return ExchangeListing{}, false
	}

	displayName := strings.TrimSpace(englishName)
	if displayName == "" {
		displayName = strings.TrimSpace(koreanName)
	}
	if displayName == "" {
		displayName = market
	}

	return ExchangeListing{
		Exchange:           exchange,
		Market:             market,
		BaseSymbol:         base,
		QuoteSymbol:        quote,
		DisplayName:        displayName,
		MarketWarning:      strings.TrimSpace(marketWarning),
		NormalizedAssetKey: strings.ToLower(base),
		Metadata:           cloneMetadataMap(metadata),
	}, true
}

func cloneMetadataMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
