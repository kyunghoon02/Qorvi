create table if not exists wallet_tracking_state (
  wallet_id uuid primary key references wallets(id) on delete cascade,
  status text not null,
  source_type text not null,
  source_ref text not null default '',
  tracking_priority integer not null default 0,
  candidate_score numeric not null default 0,
  label_confidence numeric not null default 0,
  entity_confidence numeric not null default 0,
  smart_money_confidence numeric not null default 0,
  first_discovered_at timestamptz not null,
  last_activity_at timestamptz,
  last_backfill_at timestamptz,
  last_realtime_at timestamptz,
  stale_after_at timestamptz,
  notes jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_wallet_tracking_state_status_priority
  on wallet_tracking_state (status, tracking_priority desc, updated_at desc);

create index if not exists idx_wallet_tracking_state_source_updated
  on wallet_tracking_state (source_type, updated_at desc);

create index if not exists idx_wallet_tracking_state_stale_after
  on wallet_tracking_state (stale_after_at);

create table if not exists wallet_candidate_source_events (
  id uuid primary key default gen_random_uuid(),
  wallet_id uuid not null references wallets(id) on delete cascade,
  chain text not null,
  address text not null,
  source_type text not null,
  source_ref text not null default '',
  discovery_reason text not null,
  confidence numeric not null default 0,
  payload jsonb not null default '{}'::jsonb,
  observed_at timestamptz not null,
  created_at timestamptz not null default now()
);

create index if not exists idx_wallet_candidate_source_events_wallet_observed
  on wallet_candidate_source_events (wallet_id, observed_at desc);

create index if not exists idx_wallet_candidate_source_events_source_observed
  on wallet_candidate_source_events (source_type, observed_at desc);

create table if not exists wallet_tracking_subscriptions (
  wallet_id uuid not null references wallets(id) on delete cascade,
  chain text not null,
  address text not null,
  provider text not null,
  subscription_key text not null,
  status text not null,
  last_synced_at timestamptz,
  last_event_at timestamptz,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (wallet_id, provider, subscription_key)
);

create index if not exists idx_wallet_tracking_subscriptions_provider_status
  on wallet_tracking_subscriptions (provider, status, updated_at desc);

create index if not exists idx_wallet_tracking_subscriptions_last_event
  on wallet_tracking_subscriptions (last_event_at);
