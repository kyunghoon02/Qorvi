CREATE TABLE IF NOT EXISTS wallet_copilot_index_status (
  address TEXT PRIMARY KEY,
  chain_id BIGINT NOT NULL DEFAULT 1,
  stage TEXT NOT NULL DEFAULT 'queued',
  lifetime_start_block BIGINT,
  lifetime_end_block BIGINT,
  indexed_start_block BIGINT,
  indexed_end_block BIGINT,
  completeness TEXT NOT NULL DEFAULT 'partial',
  last_error_code TEXT,
  last_error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wallet_copilot_evidence (
  evidence_id TEXT PRIMARY KEY,
  address TEXT NOT NULL,
  evidence_type TEXT NOT NULL,
  tx_hash TEXT,
  block_number BIGINT,
  contract_address TEXT,
  decoder_source TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wallet_copilot_raw_transactions (
  address TEXT NOT NULL,
  tx_hash TEXT NOT NULL,
  block_number BIGINT,
  tx_timestamp TIMESTAMPTZ NOT NULL,
  from_address TEXT NOT NULL,
  to_address TEXT,
  value_eth NUMERIC NOT NULL DEFAULT 0,
  input_data TEXT NOT NULL DEFAULT '0x',
  function_name TEXT,
  provider TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (address, tx_hash)
);

CREATE TABLE IF NOT EXISTS wallet_copilot_erc20_transfers (
  address TEXT NOT NULL,
  tx_hash TEXT NOT NULL,
  transfer_index INTEGER NOT NULL,
  tx_timestamp TIMESTAMPTZ NOT NULL,
  from_address TEXT NOT NULL,
  to_address TEXT NOT NULL,
  token_address TEXT NOT NULL,
  token_symbol TEXT NOT NULL,
  amount NUMERIC NOT NULL DEFAULT 0,
  provider TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (address, tx_hash, transfer_index)
);

CREATE TABLE IF NOT EXISTS wallet_copilot_asset_movements (
  address TEXT NOT NULL,
  tx_hash TEXT NOT NULL,
  movement_index INTEGER NOT NULL,
  asset_address TEXT,
  asset_symbol TEXT NOT NULL,
  amount NUMERIC NOT NULL DEFAULT 0,
  direction TEXT NOT NULL,
  tx_timestamp TIMESTAMPTZ NOT NULL,
  evidence_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (address, tx_hash, movement_index)
);

CREATE TABLE IF NOT EXISTS wallet_copilot_decoded_actions (
  address TEXT NOT NULL,
  tx_hash TEXT NOT NULL,
  action_index INTEGER NOT NULL,
  protocol TEXT NOT NULL,
  action_type TEXT NOT NULL,
  token_amounts JSONB NOT NULL DEFAULT '[]'::jsonb,
  usd_values JSONB NOT NULL DEFAULT '[]'::jsonb,
  confidence TEXT NOT NULL,
  evidence_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (address, tx_hash, action_index)
);

CREATE TABLE IF NOT EXISTS wallet_copilot_bridge_movements (
  address TEXT NOT NULL,
  tx_hash TEXT NOT NULL,
  movement_index INTEGER NOT NULL,
  bridge TEXT NOT NULL,
  direction TEXT NOT NULL,
  destination_chain_hint TEXT,
  assets JSONB NOT NULL DEFAULT '[]'::jsonb,
  evidence_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (address, tx_hash, movement_index)
);

CREATE TABLE IF NOT EXISTS wallet_copilot_historical_prices (
  asset_address TEXT NOT NULL,
  price_timestamp TIMESTAMPTZ NOT NULL,
  provider TEXT NOT NULL,
  value_usd NUMERIC,
  available BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (asset_address, price_timestamp, provider)
);

CREATE TABLE IF NOT EXISTS wallet_copilot_analysis_snapshots (
  address TEXT NOT NULL,
  period_days INTEGER NOT NULL,
  generated_at TIMESTAMPTZ NOT NULL,
  performance_status TEXT NOT NULL,
  coverage_status TEXT NOT NULL,
  analysis JSONB NOT NULL,
  evidence_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (address, period_days, generated_at)
);

CREATE INDEX IF NOT EXISTS idx_wallet_copilot_evidence_address_created
  ON wallet_copilot_evidence (address, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_copilot_transactions_address_timestamp
  ON wallet_copilot_raw_transactions (address, tx_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_copilot_transfers_address_timestamp
  ON wallet_copilot_erc20_transfers (address, tx_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_copilot_actions_address_created
  ON wallet_copilot_decoded_actions (address, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_copilot_bridge_address_created
  ON wallet_copilot_bridge_movements (address, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wallet_copilot_snapshot_latest
  ON wallet_copilot_analysis_snapshots (address, period_days, generated_at DESC);
