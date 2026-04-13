ALTER TABLE ai_explanations
  ADD COLUMN IF NOT EXISTS retry_after TIMESTAMPTZ;

ALTER TABLE ai_explanations
  ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '';

ALTER TABLE ai_explanations
  ADD COLUMN IF NOT EXISTS generation_started_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_ai_explanations_status_retry_after
  ON ai_explanations (status, retry_after DESC, updated_at DESC);
