create table if not exists alert_delivery_channels (
  id uuid primary key default gen_random_uuid(),
  owner_user_id text not null,
  label text not null default '',
  channel_type text not null,
  target text not null,
  metadata jsonb not null default '{}'::jsonb,
  is_enabled boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (owner_user_id, channel_type, target)
);

create table if not exists alert_delivery_attempts (
  id uuid primary key default gen_random_uuid(),
  alert_event_id uuid not null references alert_events(id) on delete cascade,
  channel_id uuid not null references alert_delivery_channels(id) on delete cascade,
  owner_user_id text not null,
  delivery_key text not null unique,
  channel_type text not null,
  target text not null,
  status text not null default 'queued',
  response_code integer not null default 0,
  details jsonb not null default '{}'::jsonb,
  attempted_at timestamptz,
  delivered_at timestamptz,
  failed_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists idx_alert_delivery_channels_owner_updated_at
  on alert_delivery_channels (owner_user_id, updated_at desc, created_at desc);

create index if not exists idx_alert_delivery_channels_owner_enabled
  on alert_delivery_channels (owner_user_id, is_enabled, channel_type);

create index if not exists idx_alert_delivery_attempts_owner_created_at
  on alert_delivery_attempts (owner_user_id, created_at desc);

create index if not exists idx_alert_delivery_attempts_event_created_at
  on alert_delivery_attempts (alert_event_id, created_at desc);
