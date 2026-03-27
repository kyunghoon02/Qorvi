CREATE TABLE IF NOT EXISTS wallet_entry_features_daily (
  wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  observed_day DATE NOT NULL,
  window_start_at TIMESTAMPTZ NOT NULL,
  window_end_at TIMESTAMPTZ NOT NULL,
  quality_wallet_overlap_count INTEGER NOT NULL DEFAULT 0,
  first_entry_before_crowding_count INTEGER NOT NULL DEFAULT 0,
  best_lead_hours_before_peers INTEGER NOT NULL DEFAULT 0,
  persistence_after_entry_proxy_count INTEGER NOT NULL DEFAULT 0,
  repeat_early_entry_success BOOLEAN NOT NULL DEFAULT FALSE,
  latest_counterparty_chain TEXT NOT NULL DEFAULT '',
  latest_counterparty_address TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (wallet_id, observed_day)
);

CREATE INDEX IF NOT EXISTS idx_wallet_entry_features_daily_observed_day
  ON wallet_entry_features_daily (observed_day DESC, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_wallet_entry_features_daily_quality
  ON wallet_entry_features_daily (quality_wallet_overlap_count DESC, first_entry_before_crowding_count DESC, updated_at DESC);
