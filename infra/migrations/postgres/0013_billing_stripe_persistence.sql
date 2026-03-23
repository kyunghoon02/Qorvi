CREATE TABLE IF NOT EXISTS billing_checkout_sessions (
  session_id text PRIMARY KEY,
  customer_id text NOT NULL DEFAULT '',
  customer_email text NOT NULL DEFAULT '',
  subscription_id text NOT NULL DEFAULT '',
  tier text NOT NULL DEFAULT 'free',
  stripe_price_id text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'open',
  success_url text NOT NULL DEFAULT '',
  cancel_url text NOT NULL DEFAULT '',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz
);

CREATE TABLE IF NOT EXISTS billing_subscriptions (
  subscription_id text PRIMARY KEY,
  customer_id text NOT NULL DEFAULT '',
  customer_email text NOT NULL DEFAULT '',
  stripe_price_id text NOT NULL DEFAULT '',
  tier text NOT NULL DEFAULT 'free',
  status text NOT NULL DEFAULT 'active',
  current_period_start timestamptz NOT NULL DEFAULT now(),
  current_period_end timestamptz NOT NULL DEFAULT now(),
  cancel_at timestamptz,
  canceled_at timestamptz,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  synced_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS billing_subscription_reconciliations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id text NOT NULL DEFAULT '',
  provider text NOT NULL DEFAULT 'stripe',
  customer_id text NOT NULL DEFAULT '',
  subscription_id text NOT NULL DEFAULT '',
  previous_tier text NOT NULL DEFAULT 'free',
  current_tier text NOT NULL DEFAULT 'free',
  stripe_price_id text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'processed',
  observed_at timestamptz NOT NULL DEFAULT now(),
  reconciled_at timestamptz,
  notes text NOT NULL DEFAULT '',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_billing_checkout_sessions_customer_created_at
  ON billing_checkout_sessions (customer_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_checkout_sessions_subscription_created_at
  ON billing_checkout_sessions (subscription_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_subscriptions_customer_updated_at
  ON billing_subscriptions (customer_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_subscriptions_status_updated_at
  ON billing_subscriptions (status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_subscription_reconciliations_subscription_observed_at
  ON billing_subscription_reconciliations (subscription_id, observed_at DESC);
