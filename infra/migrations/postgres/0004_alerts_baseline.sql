alter table alert_rules
  add column if not exists name text not null default '',
  add column if not exists notes text not null default '',
  add column if not exists tags jsonb not null default '[]'::jsonb,
  add column if not exists cooldown_seconds integer not null default 0,
  add column if not exists last_triggered_at timestamptz;

create table if not exists alert_events (
  id uuid primary key default gen_random_uuid(),
  alert_rule_id uuid not null references alert_rules(id) on delete cascade,
  owner_user_id text not null,
  event_key text not null,
  dedup_key text not null unique,
  signal_type text not null,
  severity text not null default 'medium',
  payload jsonb not null default '{}'::jsonb,
  observed_at timestamptz not null,
  created_at timestamptz not null default now(),
  unique (alert_rule_id, event_key, dedup_key)
);

create index if not exists idx_alert_rules_owner_updated_at
  on alert_rules (owner_user_id, updated_at desc, created_at desc);

create index if not exists idx_alert_rules_rule_type_enabled
  on alert_rules (rule_type, is_enabled);

create index if not exists idx_alert_events_rule_observed_at
  on alert_events (alert_rule_id, observed_at desc, created_at desc);

create index if not exists idx_alert_events_owner_created_at
  on alert_events (owner_user_id, created_at desc);

create index if not exists idx_alert_events_rule_event_key_created_at
  on alert_events (alert_rule_id, event_key, created_at desc);
