alter table watchlists
  alter column owner_user_id type text using owner_user_id::text;

alter table alert_rules
  alter column owner_user_id type text using owner_user_id::text;

alter table audit_logs
  alter column actor_user_id type text using actor_user_id::text;

alter table watchlists
  add column if not exists notes text not null default '',
  add column if not exists tags jsonb not null default '[]'::jsonb,
  add column if not exists updated_at timestamptz not null default now();

alter table watchlist_items
  add column if not exists tags jsonb not null default '[]'::jsonb,
  add column if not exists notes text not null default '',
  add column if not exists updated_at timestamptz not null default now();

create index if not exists idx_watchlists_owner_updated_at
  on watchlists (owner_user_id, updated_at desc, created_at desc);

create index if not exists idx_watchlist_items_watchlist_updated_at
  on watchlist_items (watchlist_id, updated_at desc, created_at desc);

create index if not exists idx_watchlist_items_item_type_item_key
  on watchlist_items (item_type, item_key);
