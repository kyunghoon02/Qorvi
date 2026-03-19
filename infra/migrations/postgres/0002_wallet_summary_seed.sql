insert into wallets (id, chain, address, display_name)
values (
  '11111111-1111-1111-1111-111111111111',
  'evm',
  '0x1234567890abcdef1234567890abcdef12345678',
  'Seed Whale'
)
on conflict (chain, address) do update
set display_name = excluded.display_name,
    updated_at = now();

insert into wallet_daily_stats (
  wallet_id,
  as_of_date,
  transaction_count,
  counterparty_count,
  latest_activity_at,
  incoming_tx_count,
  outgoing_tx_count
)
select
  w.id,
  current_date,
  42,
  18,
  '2026-03-19T01:02:03Z'::timestamptz,
  13,
  29
from wallets w
where w.chain = 'evm'
  and w.address = '0x1234567890abcdef1234567890abcdef12345678'
on conflict (wallet_id, as_of_date) do update
set transaction_count = excluded.transaction_count,
    counterparty_count = excluded.counterparty_count,
    latest_activity_at = excluded.latest_activity_at,
    incoming_tx_count = excluded.incoming_tx_count,
    outgoing_tx_count = excluded.outgoing_tx_count;
