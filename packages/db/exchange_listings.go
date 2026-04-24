package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ExchangeListingRegistryEntry struct {
	Exchange           string
	Market             string
	BaseSymbol         string
	QuoteSymbol        string
	DisplayName        string
	MarketWarning      string
	NormalizedAssetKey string
	TokenAddress       string
	ChainHint          string
	Listed             bool
	ListedAtDetected   time.Time
	LastCheckedAt      time.Time
	Metadata           map[string]any
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ExchangeListingRegistryStore interface {
	UpsertExchangeListings(context.Context, []ExchangeListingRegistryEntry) error
	ListExchangeListings(context.Context, string) ([]ExchangeListingRegistryEntry, error)
}

type PostgresExchangeListingRegistryStore struct {
	Querier postgresQuerier
	Execer  postgresExecExecer
	Now     func() time.Time
}

const upsertExchangeListingRegistrySQL = `
INSERT INTO exchange_listing_registry (
  exchange,
  market,
  base_symbol,
  quote_symbol,
  display_name,
  market_warning,
  normalized_asset_key,
  token_address,
  chain_hint,
  listed,
  listed_at_detected,
  last_checked_at,
  metadata,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, now()
)
ON CONFLICT (exchange, market) DO UPDATE SET
  base_symbol = EXCLUDED.base_symbol,
  quote_symbol = EXCLUDED.quote_symbol,
  display_name = EXCLUDED.display_name,
  market_warning = EXCLUDED.market_warning,
  normalized_asset_key = EXCLUDED.normalized_asset_key,
  token_address = EXCLUDED.token_address,
  chain_hint = EXCLUDED.chain_hint,
  listed = EXCLUDED.listed,
  listed_at_detected = EXCLUDED.listed_at_detected,
  last_checked_at = EXCLUDED.last_checked_at,
  metadata = EXCLUDED.metadata,
  updated_at = now()
`

const listExchangeListingRegistrySQL = `
SELECT
  exchange,
  market,
  base_symbol,
  quote_symbol,
  display_name,
  market_warning,
  normalized_asset_key,
  token_address,
  chain_hint,
  listed,
  listed_at_detected,
  last_checked_at,
  metadata,
  created_at,
  updated_at
FROM exchange_listing_registry
WHERE ($1 = '' OR exchange = $1)
ORDER BY exchange ASC, market ASC
`

func NewPostgresExchangeListingRegistryStore(
	querier postgresQuerier,
	execer postgresExecExecer,
) *PostgresExchangeListingRegistryStore {
	return &PostgresExchangeListingRegistryStore{
		Querier: querier,
		Execer:  execer,
		Now:     time.Now,
	}
}

func NewPostgresExchangeListingRegistryStoreFromPool(pool interface {
	postgresQuerier
	postgresExecExecer
}) *PostgresExchangeListingRegistryStore {
	if pool == nil {
		return nil
	}
	return NewPostgresExchangeListingRegistryStore(pool, pool)
}

func (s *PostgresExchangeListingRegistryStore) UpsertExchangeListings(
	ctx context.Context,
	entries []ExchangeListingRegistryEntry,
) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("exchange listing registry store is nil")
	}
	for _, entry := range entries {
		now := time.Now
		if s.Now != nil {
			now = s.Now
		}
		normalized, err := normalizeExchangeListingRegistryEntry(entry, now())
		if err != nil {
			return err
		}
		payload, err := json.Marshal(normalized.Metadata)
		if err != nil {
			return fmt.Errorf("marshal exchange listing metadata: %w", err)
		}
		if _, err := s.Execer.Exec(
			ctx,
			upsertExchangeListingRegistrySQL,
			normalized.Exchange,
			normalized.Market,
			normalized.BaseSymbol,
			normalized.QuoteSymbol,
			normalized.DisplayName,
			normalized.MarketWarning,
			normalized.NormalizedAssetKey,
			normalized.TokenAddress,
			normalized.ChainHint,
			normalized.Listed,
			normalized.ListedAtDetected.UTC(),
			normalized.LastCheckedAt.UTC(),
			payload,
		); err != nil {
			return fmt.Errorf("upsert exchange listing %s:%s: %w", normalized.Exchange, normalized.Market, err)
		}
	}
	return nil
}

func (s *PostgresExchangeListingRegistryStore) ListExchangeListings(
	ctx context.Context,
	exchange string,
) ([]ExchangeListingRegistryEntry, error) {
	if s == nil || s.Querier == nil {
		return nil, fmt.Errorf("exchange listing registry store is nil")
	}
	rows, err := s.Querier.Query(ctx, listExchangeListingRegistrySQL, strings.ToLower(strings.TrimSpace(exchange)))
	if err != nil {
		return nil, fmt.Errorf("list exchange listings: %w", err)
	}
	defer rows.Close()

	items := make([]ExchangeListingRegistryEntry, 0)
	for rows.Next() {
		item, err := scanExchangeListingRegistryRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exchange listings: %w", err)
	}
	return items, nil
}

func normalizeExchangeListingRegistryEntry(
	entry ExchangeListingRegistryEntry,
	now time.Time,
) (ExchangeListingRegistryEntry, error) {
	entry.Exchange = strings.ToLower(strings.TrimSpace(entry.Exchange))
	entry.Market = strings.ToUpper(strings.TrimSpace(entry.Market))
	entry.BaseSymbol = strings.ToUpper(strings.TrimSpace(entry.BaseSymbol))
	entry.QuoteSymbol = strings.ToUpper(strings.TrimSpace(entry.QuoteSymbol))
	entry.DisplayName = strings.TrimSpace(entry.DisplayName)
	entry.MarketWarning = strings.TrimSpace(entry.MarketWarning)
	entry.NormalizedAssetKey = strings.ToLower(strings.TrimSpace(entry.NormalizedAssetKey))
	entry.TokenAddress = strings.TrimSpace(entry.TokenAddress)
	entry.ChainHint = strings.TrimSpace(entry.ChainHint)
	if entry.Exchange == "" {
		return ExchangeListingRegistryEntry{}, fmt.Errorf("exchange is required")
	}
	if entry.Market == "" {
		return ExchangeListingRegistryEntry{}, fmt.Errorf("market is required")
	}
	if entry.BaseSymbol == "" || entry.QuoteSymbol == "" {
		return ExchangeListingRegistryEntry{}, fmt.Errorf("base_symbol and quote_symbol are required")
	}
	if entry.DisplayName == "" {
		entry.DisplayName = entry.Market
	}
	if entry.NormalizedAssetKey == "" {
		entry.NormalizedAssetKey = strings.ToLower(entry.BaseSymbol)
	}
	if entry.Metadata == nil {
		entry.Metadata = map[string]any{}
	}
	if entry.ListedAtDetected.IsZero() {
		entry.ListedAtDetected = now
	}
	if entry.LastCheckedAt.IsZero() {
		entry.LastCheckedAt = now
	}
	return entry, nil
}

func scanExchangeListingRegistryRow(scanner interface {
	Scan(...any) error
}) (ExchangeListingRegistryEntry, error) {
	var (
		item        ExchangeListingRegistryEntry
		rawMetadata []byte
	)
	if err := scanner.Scan(
		&item.Exchange,
		&item.Market,
		&item.BaseSymbol,
		&item.QuoteSymbol,
		&item.DisplayName,
		&item.MarketWarning,
		&item.NormalizedAssetKey,
		&item.TokenAddress,
		&item.ChainHint,
		&item.Listed,
		&item.ListedAtDetected,
		&item.LastCheckedAt,
		&rawMetadata,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return ExchangeListingRegistryEntry{}, fmt.Errorf("scan exchange listing registry row: %w", err)
	}
	item.Metadata = map[string]any{}
	if len(rawMetadata) > 0 {
		if err := json.Unmarshal(rawMetadata, &item.Metadata); err != nil {
			return ExchangeListingRegistryEntry{}, fmt.Errorf("decode exchange listing metadata: %w", err)
		}
	}
	return item, nil
}

var _ ExchangeListingRegistryStore = (*PostgresExchangeListingRegistryStore)(nil)
