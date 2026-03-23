CREATE TABLE IF NOT EXISTS billing_accounts (
  owner_user_id text PRIMARY KEY,
  email text NOT NULL DEFAULT '',
  current_tier text NOT NULL DEFAULT 'free',
  stripe_customer_id text NOT NULL DEFAULT '',
  active_subscription_id text NOT NULL DEFAULT '',
  current_price_id text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'inactive',
  current_period_end timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS billing_webhook_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  provider_event_id text NOT NULL UNIQUE,
  event_type text NOT NULL,
  owner_user_id text NOT NULL DEFAULT '',
  customer_id text NOT NULL DEFAULT '',
  subscription_id text NOT NULL DEFAULT '',
  plan_tier text NOT NULL DEFAULT 'free',
  status text NOT NULL DEFAULT 'received',
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  received_at timestamptz NOT NULL DEFAULT now(),
  processed_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_billing_accounts_tier_updated_at
  ON billing_accounts (current_tier, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_webhook_events_owner_received_at
  ON billing_webhook_events (owner_user_id, received_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_webhook_events_subscription_received_at
  ON billing_webhook_events (subscription_id, received_at DESC);
