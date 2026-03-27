ALTER TABLE wallet_treasury_features_daily
  ADD COLUMN IF NOT EXISTS internal_ops_distribution_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS external_ops_distribution_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS external_market_adjacent_distribution_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS external_non_market_distribution_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE wallet_mm_features_daily
  ADD COLUMN IF NOT EXISTS project_to_mm_routed_candidate_count INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS project_to_mm_adjacency_count INTEGER NOT NULL DEFAULT 0;
