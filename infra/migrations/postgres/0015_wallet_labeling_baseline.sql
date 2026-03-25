CREATE TABLE IF NOT EXISTS entity_labels (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  label_key text NOT NULL UNIQUE,
  label_name text NOT NULL,
  label_class text NOT NULL,
  entity_type text NOT NULL DEFAULT '',
  source text NOT NULL DEFAULT '',
  default_confidence numeric NOT NULL DEFAULT 1,
  verified boolean NOT NULL DEFAULT false,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS entity_label_memberships (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  label_id uuid NOT NULL REFERENCES entity_labels(id) ON DELETE CASCADE,
  entity_key text,
  source text NOT NULL DEFAULT '',
  confidence numeric NOT NULL DEFAULT 0,
  evidence_summary text NOT NULL DEFAULT '',
  observed_at timestamptz NOT NULL DEFAULT now(),
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (wallet_id, label_id)
);

CREATE TABLE IF NOT EXISTS wallet_evidence (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  wallet_id uuid NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  evidence_key text NOT NULL,
  evidence_type text NOT NULL,
  source text NOT NULL DEFAULT '',
  confidence numeric NOT NULL DEFAULT 0,
  observed_at timestamptz NOT NULL,
  summary text NOT NULL DEFAULT '',
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (wallet_id, evidence_key)
);

CREATE INDEX IF NOT EXISTS idx_entity_labels_class_updated_at
  ON entity_labels (label_class, updated_at DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_entity_label_memberships_wallet_observed_at
  ON entity_label_memberships (wallet_id, observed_at DESC, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_wallet_evidence_wallet_observed_at
  ON wallet_evidence (wallet_id, observed_at DESC, updated_at DESC);
