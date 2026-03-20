create extension if not exists pgcrypto;

create table if not exists wallets (
  id uuid primary key default gen_random_uuid(),
  chain text not null,
  address text not null,
  display_name text not null,
  entity_key text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (chain, address)
);

create table if not exists wallet_daily_stats (
  wallet_id uuid not null references wallets(id) on delete cascade,
  as_of_date date not null,
  transaction_count integer not null default 0,
  counterparty_count integer not null default 0,
  latest_activity_at timestamptz,
  incoming_tx_count integer not null default 0,
  outgoing_tx_count integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  primary key (wallet_id, as_of_date)
);

create table if not exists tokens (
  id uuid primary key default gen_random_uuid(),
  chain text not null,
  token_address text not null,
  symbol text not null,
  decimals integer not null,
  created_at timestamptz not null default now(),
  unique (chain, token_address)
);

create table if not exists entities (
  id uuid primary key default gen_random_uuid(),
  entity_key text not null unique,
  entity_type text not null,
  created_at timestamptz not null default now()
);

create table if not exists clusters (
  id uuid primary key default gen_random_uuid(),
  cluster_key text not null unique,
  cluster_type text not null,
  cluster_score integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists cluster_members (
  cluster_id uuid not null references clusters(id) on delete cascade,
  wallet_id uuid not null references wallets(id) on delete cascade,
  role text not null default 'member',
  created_at timestamptz not null default now(),
  primary key (cluster_id, wallet_id)
);

create table if not exists transactions (
  id uuid primary key default gen_random_uuid(),
  chain text not null,
  tx_hash text not null,
  wallet_id uuid not null references wallets(id) on delete cascade,
  direction text not null default 'unknown',
  counterparty_chain text,
  counterparty_address text,
  raw_payload_path text not null,
  schema_version integer not null default 1,
  observed_at timestamptz not null,
  created_at timestamptz not null default now(),
  unique (chain, tx_hash, wallet_id)
);

create table if not exists signal_events (
  id uuid primary key default gen_random_uuid(),
  signal_type text not null,
  wallet_id uuid references wallets(id) on delete cascade,
  cluster_id uuid references clusters(id) on delete cascade,
  payload jsonb not null,
  observed_at timestamptz not null,
  created_at timestamptz not null default now()
);

create table if not exists alert_rules (
  id uuid primary key default gen_random_uuid(),
  owner_user_id uuid,
  rule_type text not null,
  is_enabled boolean not null default true,
  definition jsonb not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists watchlists (
  id uuid primary key default gen_random_uuid(),
  owner_user_id uuid not null,
  name text not null,
  created_at timestamptz not null default now()
);

create table if not exists watchlist_items (
  id uuid primary key default gen_random_uuid(),
  watchlist_id uuid not null references watchlists(id) on delete cascade,
  item_type text not null,
  item_key text not null,
  created_at timestamptz not null default now(),
  unique (watchlist_id, item_type, item_key)
);

create table if not exists suppressions (
  id uuid primary key default gen_random_uuid(),
  suppression_key text not null unique,
  suppression_type text not null,
  reason text not null,
  created_at timestamptz not null default now(),
  expires_at timestamptz
);

create table if not exists audit_logs (
  id uuid primary key default gen_random_uuid(),
  actor_user_id uuid,
  action text not null,
  target_type text not null,
  target_key text not null,
  payload jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists provider_usage_logs (
  id uuid primary key default gen_random_uuid(),
  provider text not null,
  operation text not null,
  status_code integer not null,
  latency_ms integer not null,
  created_at timestamptz not null default now()
);

create table if not exists job_runs (
  id uuid primary key default gen_random_uuid(),
  job_name text not null,
  status text not null,
  started_at timestamptz not null default now(),
  finished_at timestamptz,
  details jsonb not null default '{}'::jsonb
);

create index if not exists idx_transactions_wallet_observed_at on transactions (wallet_id, observed_at desc);
create index if not exists idx_wallet_daily_stats_wallet_as_of_date on wallet_daily_stats (wallet_id, as_of_date desc);
create index if not exists idx_provider_usage_logs_provider_created_at on provider_usage_logs (provider, created_at desc);
create index if not exists idx_signal_events_wallet_observed_at on signal_events (wallet_id, observed_at desc);
create index if not exists idx_signal_events_cluster_observed_at on signal_events (cluster_id, observed_at desc);
create index if not exists idx_audit_logs_target_created_at on audit_logs (target_type, created_at desc);
