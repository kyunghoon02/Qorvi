ALTER TABLE wallet_copilot_erc20_transfers
  ADD COLUMN IF NOT EXISTS block_number BIGINT;

ALTER TABLE wallet_copilot_index_status
  ADD COLUMN IF NOT EXISTS receipt_log_coverage TEXT NOT NULL DEFAULT 'unavailable',
  ADD COLUMN IF NOT EXISTS historical_price_coverage TEXT NOT NULL DEFAULT 'unavailable';

CREATE TABLE IF NOT EXISTS wallet_copilot_receipts (
  address TEXT NOT NULL,
  tx_hash TEXT NOT NULL,
  block_number BIGINT,
  status TEXT NOT NULL,
  provider TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (address, tx_hash)
);

CREATE TABLE IF NOT EXISTS wallet_copilot_logs (
  address TEXT NOT NULL,
  tx_hash TEXT NOT NULL,
  log_index INTEGER NOT NULL,
  block_number BIGINT,
  contract_address TEXT NOT NULL,
  topic0 TEXT,
  topics JSONB NOT NULL DEFAULT '[]'::jsonb,
  data TEXT NOT NULL DEFAULT '0x',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (address, tx_hash, log_index)
);

CREATE TABLE IF NOT EXISTS wallet_copilot_aave_reserve_snapshots (
  market TEXT NOT NULL,
  asset_address TEXT NOT NULL,
  symbol TEXT NOT NULL,
  decimals INTEGER NOT NULL,
  discovery_source TEXT NOT NULL,
  observed_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (market, asset_address, observed_at)
);

CREATE INDEX IF NOT EXISTS idx_wallet_copilot_receipts_address_block
  ON wallet_copilot_receipts (address, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_copilot_logs_contract_topic
  ON wallet_copilot_logs (contract_address, topic0, block_number DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_copilot_historical_prices_lookup
  ON wallet_copilot_historical_prices (asset_address, price_timestamp DESC)
  WHERE available = TRUE;
CREATE INDEX IF NOT EXISTS idx_wallet_copilot_aave_reserves_latest
  ON wallet_copilot_aave_reserve_snapshots (market, observed_at DESC);
