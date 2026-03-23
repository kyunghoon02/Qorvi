CREATE TABLE IF NOT EXISTS wallet_graph_snapshots (
  chain text NOT NULL,
  address text NOT NULL,
  max_counterparties integer NOT NULL,
  graph_payload jsonb NOT NULL,
  generated_at timestamptz NOT NULL,
  source text NOT NULL DEFAULT 'graph-snapshot',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (chain, address, max_counterparties)
);

CREATE INDEX IF NOT EXISTS idx_wallet_graph_snapshots_generated_at
  ON wallet_graph_snapshots (generated_at DESC);
