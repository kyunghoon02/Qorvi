ALTER TABLE wallet_treasury_features_daily
  ADD COLUMN IF NOT EXISTS treasury_to_exchange_path_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS treasury_to_bridge_path_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS treasury_to_mm_path_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS distinct_market_counterparty_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS operational_only_distribution_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE wallet_mm_features_daily
  ADD COLUMN IF NOT EXISTS post_handoff_exchange_touch_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS post_handoff_bridge_touch_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS project_to_mm_contact_count INTEGER NOT NULL DEFAULT 0;
