CREATE TABLE IF NOT EXISTS wallet_enrichment_snapshots (
  chain text NOT NULL,
  address text NOT NULL,
  provider text NOT NULL DEFAULT '',
  net_worth_usd text NOT NULL DEFAULT '',
  native_balance text NOT NULL DEFAULT '',
  native_balance_formatted text NOT NULL DEFAULT '',
  active_chains text[] NOT NULL DEFAULT '{}',
  holding_count integer NOT NULL DEFAULT 0,
  holdings jsonb NOT NULL DEFAULT '[]'::jsonb,
  observed_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (chain, address)
);

CREATE INDEX IF NOT EXISTS idx_wallet_enrichment_snapshots_observed_at
  ON wallet_enrichment_snapshots (observed_at DESC, updated_at DESC);
