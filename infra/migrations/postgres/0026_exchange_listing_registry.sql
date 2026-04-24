CREATE TABLE IF NOT EXISTS exchange_listing_registry (
  exchange TEXT NOT NULL,
  market TEXT NOT NULL,
  base_symbol TEXT NOT NULL,
  quote_symbol TEXT NOT NULL,
  display_name TEXT NOT NULL DEFAULT '',
  market_warning TEXT NOT NULL DEFAULT '',
  normalized_asset_key TEXT NOT NULL DEFAULT '',
  token_address TEXT NOT NULL DEFAULT '',
  chain_hint TEXT NOT NULL DEFAULT '',
  listed BOOLEAN NOT NULL DEFAULT TRUE,
  listed_at_detected TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_checked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (exchange, market)
);

CREATE INDEX IF NOT EXISTS idx_exchange_listing_registry_exchange_listed
  ON exchange_listing_registry (exchange, listed, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_exchange_listing_registry_asset_key
  ON exchange_listing_registry (normalized_asset_key, listed, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_exchange_listing_registry_token_address
  ON exchange_listing_registry (chain_hint, token_address)
  WHERE token_address <> '';
