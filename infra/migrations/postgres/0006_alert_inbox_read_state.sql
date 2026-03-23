alter table alert_events
  add column if not exists read_at timestamptz;

create index if not exists idx_alert_events_owner_read_observed_at
  on alert_events (owner_user_id, read_at, observed_at desc, id desc);
