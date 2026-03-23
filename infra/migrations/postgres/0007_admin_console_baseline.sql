CREATE TABLE IF NOT EXISTS admin_labels (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL UNIQUE,
  description text NOT NULL DEFAULT '',
  color text NOT NULL DEFAULT '',
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE suppressions
  ADD COLUMN IF NOT EXISTS target text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS created_by text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS active boolean NOT NULL DEFAULT true,
  ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now();

UPDATE suppressions
SET
  target = CASE WHEN target = '' THEN suppression_key ELSE target END,
  created_by = CASE WHEN created_by = '' THEN 'system' ELSE created_by END,
  updated_at = COALESCE(updated_at, created_at);

CREATE INDEX IF NOT EXISTS idx_admin_labels_updated_at
  ON admin_labels (updated_at DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_suppressions_active_updated_at
  ON suppressions (active, updated_at DESC, created_at DESC);
