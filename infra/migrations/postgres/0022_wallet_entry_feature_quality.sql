ALTER TABLE wallet_entry_features_daily
  ADD COLUMN IF NOT EXISTS sustained_overlap_counterparty_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS strong_lead_counterparty_count INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_wallet_entry_features_daily_repeatability
  ON wallet_entry_features_daily (
    sustained_overlap_counterparty_count DESC,
    strong_lead_counterparty_count DESC,
    updated_at DESC
  );
