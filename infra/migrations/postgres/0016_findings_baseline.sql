CREATE TABLE IF NOT EXISTS finding_candidates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  finding_type text NOT NULL,
  wallet_id uuid REFERENCES wallets(id) ON DELETE CASCADE,
  cluster_id uuid REFERENCES clusters(id) ON DELETE CASCADE,
  subject_type text NOT NULL,
  subject_chain text,
  subject_address text,
  subject_key text NOT NULL,
  subject_label text NOT NULL DEFAULT '',
  confidence numeric(5,4) NOT NULL DEFAULT 0,
  importance_score numeric(5,4) NOT NULL DEFAULT 0,
  summary text NOT NULL,
  dedup_key text NOT NULL,
  status text NOT NULL DEFAULT 'active',
  observed_at timestamptz NOT NULL,
  coverage_start_at timestamptz,
  coverage_end_at timestamptz,
  coverage_window_days integer NOT NULL DEFAULT 0,
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (dedup_key)
);

CREATE INDEX IF NOT EXISTS idx_finding_candidates_observed_at
  ON finding_candidates (observed_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_finding_candidates_wallet_observed_at
  ON finding_candidates (wallet_id, observed_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_finding_candidates_type_observed_at
  ON finding_candidates (finding_type, observed_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS finding_evidence_bundles (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  finding_id uuid NOT NULL REFERENCES finding_candidates(id) ON DELETE CASCADE,
  bundle jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (finding_id)
);

CREATE INDEX IF NOT EXISTS idx_finding_evidence_bundles_finding
  ON finding_evidence_bundles (finding_id);
