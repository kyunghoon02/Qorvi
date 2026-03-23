ALTER TABLE entities
  ADD COLUMN IF NOT EXISTS display_name text;

ALTER TABLE entities
  ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_entities_updated_at
  ON entities (updated_at DESC, created_at DESC);
