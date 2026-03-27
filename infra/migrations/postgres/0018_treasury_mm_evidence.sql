CREATE TABLE IF NOT EXISTS wallet_treasury_paths (
  id BIGSERIAL PRIMARY KEY,
  wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  observed_day DATE NOT NULL,
  tx_hash TEXT NOT NULL,
  observed_at TIMESTAMPTZ NOT NULL,
  path_kind TEXT NOT NULL,
  counterparty_chain TEXT NOT NULL,
  counterparty_address TEXT NOT NULL,
  counterparty_label TEXT NOT NULL DEFAULT '',
  counterparty_entity_key TEXT NOT NULL DEFAULT '',
  counterparty_entity_type TEXT NOT NULL DEFAULT '',
  downstream_chain TEXT NOT NULL DEFAULT '',
  downstream_address TEXT NOT NULL DEFAULT '',
  downstream_label TEXT NOT NULL DEFAULT '',
  downstream_entity_key TEXT NOT NULL DEFAULT '',
  downstream_entity_type TEXT NOT NULL DEFAULT '',
  downstream_tx_hash TEXT NOT NULL DEFAULT '',
  downstream_observed_at TIMESTAMPTZ,
  amount_numeric NUMERIC(38, 18),
  token_symbol TEXT NOT NULL DEFAULT '',
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (wallet_id, tx_hash, path_kind, counterparty_chain, counterparty_address, downstream_tx_hash, observed_at)
);

CREATE INDEX IF NOT EXISTS wallet_treasury_paths_wallet_idx
  ON wallet_treasury_paths (wallet_id, observed_day DESC, observed_at DESC);

CREATE TABLE IF NOT EXISTS wallet_treasury_features_daily (
  wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  observed_day DATE NOT NULL,
  window_start_at TIMESTAMPTZ NOT NULL,
  window_end_at TIMESTAMPTZ NOT NULL,
  anchor_match_count INTEGER NOT NULL DEFAULT 0,
  fanout_signature_count INTEGER NOT NULL DEFAULT 0,
  operational_distribution_count INTEGER NOT NULL DEFAULT 0,
  rebalance_discount_count INTEGER NOT NULL DEFAULT 0,
  treasury_to_market_path_count INTEGER NOT NULL DEFAULT 0,
  latest_treasury_tx_hash TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (wallet_id, observed_day)
);

CREATE TABLE IF NOT EXISTS wallet_mm_paths (
  id BIGSERIAL PRIMARY KEY,
  wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  observed_day DATE NOT NULL,
  tx_hash TEXT NOT NULL,
  observed_at TIMESTAMPTZ NOT NULL,
  path_kind TEXT NOT NULL,
  counterparty_chain TEXT NOT NULL,
  counterparty_address TEXT NOT NULL,
  counterparty_label TEXT NOT NULL DEFAULT '',
  counterparty_entity_key TEXT NOT NULL DEFAULT '',
  counterparty_entity_type TEXT NOT NULL DEFAULT '',
  downstream_chain TEXT NOT NULL DEFAULT '',
  downstream_address TEXT NOT NULL DEFAULT '',
  downstream_label TEXT NOT NULL DEFAULT '',
  downstream_entity_key TEXT NOT NULL DEFAULT '',
  downstream_entity_type TEXT NOT NULL DEFAULT '',
  downstream_tx_hash TEXT NOT NULL DEFAULT '',
  downstream_observed_at TIMESTAMPTZ,
  amount_numeric NUMERIC(38, 18),
  token_symbol TEXT NOT NULL DEFAULT '',
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (wallet_id, tx_hash, path_kind, counterparty_chain, counterparty_address, downstream_tx_hash, observed_at)
);

CREATE INDEX IF NOT EXISTS wallet_mm_paths_wallet_idx
  ON wallet_mm_paths (wallet_id, observed_day DESC, observed_at DESC);

CREATE TABLE IF NOT EXISTS wallet_mm_features_daily (
  wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  observed_day DATE NOT NULL,
  window_start_at TIMESTAMPTZ NOT NULL,
  window_end_at TIMESTAMPTZ NOT NULL,
  mm_anchor_match_count INTEGER NOT NULL DEFAULT 0,
  inventory_rotation_count INTEGER NOT NULL DEFAULT 0,
  project_to_mm_path_count INTEGER NOT NULL DEFAULT 0,
  post_handoff_distribution_count INTEGER NOT NULL DEFAULT 0,
  repeat_mm_counterparty_count INTEGER NOT NULL DEFAULT 0,
  latest_mm_tx_hash TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (wallet_id, observed_day)
);
