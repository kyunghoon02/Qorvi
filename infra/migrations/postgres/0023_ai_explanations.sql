CREATE TABLE IF NOT EXISTS ai_explanations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  scope_type TEXT NOT NULL,
  scope_key TEXT NOT NULL,
  input_hash TEXT NOT NULL,
  requested_by_user_id TEXT NOT NULL,
  model TEXT NOT NULL,
  prompt_version TEXT NOT NULL,
  status TEXT NOT NULL,
  response_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  request_count INTEGER NOT NULL DEFAULT 1,
  last_requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_explanations_scope_hash_model_version
  ON ai_explanations (scope_type, scope_key, input_hash, model, prompt_version);

CREATE INDEX IF NOT EXISTS idx_ai_explanations_scope_last_requested
  ON ai_explanations (scope_type, scope_key, last_requested_at DESC);

CREATE INDEX IF NOT EXISTS idx_ai_explanations_status_updated
  ON ai_explanations (status, updated_at DESC);
