CREATE TABLE IF NOT EXISTS wallet_bridge_links (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  observed_day date NOT NULL,
  tx_hash text NOT NULL,
  observed_at timestamptz NOT NULL,
  bridge_chain text NOT NULL,
  bridge_address text NOT NULL,
  bridge_label text NOT NULL DEFAULT '',
  bridge_entity_key text NOT NULL DEFAULT '',
  bridge_entity_type text NOT NULL DEFAULT '',
  amount_numeric numeric,
  token_symbol text NOT NULL DEFAULT '',
  destination_chain text NOT NULL DEFAULT '',
  destination_address text NOT NULL DEFAULT '',
  destination_label text NOT NULL DEFAULT '',
  destination_entity_key text NOT NULL DEFAULT '',
  destination_entity_type text NOT NULL DEFAULT '',
  destination_tx_hash text NOT NULL DEFAULT '',
  destination_observed_at timestamptz,
  confidence numeric(5,4) NOT NULL DEFAULT 0,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (wallet_id, tx_hash, bridge_chain, bridge_address, destination_tx_hash, observed_at)
);

CREATE INDEX IF NOT EXISTS idx_wallet_bridge_links_wallet_day
  ON wallet_bridge_links (wallet_id, observed_day DESC, observed_at DESC);

CREATE TABLE IF NOT EXISTS wallet_bridge_features_daily (
  wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  observed_day date NOT NULL,
  window_start_at timestamptz NOT NULL,
  window_end_at timestamptz NOT NULL,
  bridge_outbound_count integer NOT NULL DEFAULT 0,
  distinct_bridge_counterparties integer NOT NULL DEFAULT 0,
  distinct_bridge_protocols integer NOT NULL DEFAULT 0,
  confirmed_destination_count integer NOT NULL DEFAULT 0,
  post_bridge_fresh_wallet_count integer NOT NULL DEFAULT 0,
  post_bridge_exchange_touch_count integer NOT NULL DEFAULT 0,
  post_bridge_protocol_entry_count integer NOT NULL DEFAULT 0,
  bridge_outflow_amount numeric NOT NULL DEFAULT 0,
  bridge_outflow_share numeric(6,4) NOT NULL DEFAULT 0,
  bridge_recurrence_days integer NOT NULL DEFAULT 0,
  latest_bridge_tx_hash text NOT NULL DEFAULT '',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (wallet_id, observed_day)
);

CREATE TABLE IF NOT EXISTS wallet_exchange_paths (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  observed_day date NOT NULL,
  tx_hash text NOT NULL,
  observed_at timestamptz NOT NULL,
  path_kind text NOT NULL,
  intermediary_chain text NOT NULL DEFAULT '',
  intermediary_address text NOT NULL DEFAULT '',
  intermediary_label text NOT NULL DEFAULT '',
  intermediary_entity_key text NOT NULL DEFAULT '',
  intermediary_entity_type text NOT NULL DEFAULT '',
  exchange_chain text NOT NULL,
  exchange_address text NOT NULL,
  exchange_label text NOT NULL DEFAULT '',
  exchange_entity_key text NOT NULL DEFAULT '',
  exchange_entity_type text NOT NULL DEFAULT '',
  exchange_tx_hash text NOT NULL DEFAULT '',
  exchange_observed_at timestamptz,
  amount_numeric numeric,
  token_symbol text NOT NULL DEFAULT '',
  confidence numeric(5,4) NOT NULL DEFAULT 0,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (wallet_id, tx_hash, path_kind, exchange_chain, exchange_address, exchange_tx_hash, observed_at)
);

CREATE INDEX IF NOT EXISTS idx_wallet_exchange_paths_wallet_day
  ON wallet_exchange_paths (wallet_id, observed_day DESC, observed_at DESC);

CREATE TABLE IF NOT EXISTS wallet_exchange_flow_features_daily (
  wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  observed_day date NOT NULL,
  window_start_at timestamptz NOT NULL,
  window_end_at timestamptz NOT NULL,
  exchange_outbound_count integer NOT NULL DEFAULT 0,
  distinct_exchange_counterparties integer NOT NULL DEFAULT 0,
  deposit_like_path_count integer NOT NULL DEFAULT 0,
  exchange_fanout_count integer NOT NULL DEFAULT 0,
  fresh_recipient_count integer NOT NULL DEFAULT 0,
  exchange_outflow_amount numeric NOT NULL DEFAULT 0,
  exchange_outflow_share numeric(6,4) NOT NULL DEFAULT 0,
  exchange_recurrence_days integer NOT NULL DEFAULT 0,
  latest_exchange_tx_hash text NOT NULL DEFAULT '',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (wallet_id, observed_day)
);
