-- Qorvi backtest candidate query
-- Case type: bridge_return
-- Cohort: known_negative
--
-- Purpose:
-- Find wallets that bridge funds out and then return funds back within a bounded window
-- without strong downstream distribution. These are high-value false-positive candidates
-- for shadow-exit suppression.
--
-- Notes:
-- 1. This is a candidate query, not final truth.
-- 2. Review every row before promotion into backtest-manifest.json.
-- 3. Replace the bridge labels filter with the bridge universe you trust for production.

with bridge_labels as (
  select
    blockchain,
    address,
    name,
    category,
    type
  from labels.addresses
  where blockchain = 'ethereum'
    and lower(category) like '%bridge%'
),

bridge_out as (
  select
    t.block_time as out_block_time,
    t.block_number as out_block_number,
    t.tx_hash as out_tx_hash,
    t."from" as wallet_address,
    t."to" as bridge_address,
    t.amount_usd as out_amount_usd
  from tokens.transfers t
  inner join bridge_labels b
    on b.blockchain = 'ethereum'
   and b.address = t."to"
  where t.blockchain = 'ethereum'
    and t.block_time >= cast('{{window_start}}' as timestamp)
    and t.block_time < cast('{{window_end}}' as timestamp)
    and coalesce(t.amount_usd, 0) >= {{min_bridge_usd}}
),

bridge_return as (
  select
    o.wallet_address,
    o.bridge_address,
    o.out_block_time,
    o.out_block_number,
    o.out_tx_hash,
    min(r.block_time) as return_block_time,
    min_by(r.tx_hash, r.block_time) as return_tx_hash,
    min_by(r.block_number, r.block_time) as return_block_number,
    max(coalesce(r.amount_usd, 0)) as return_amount_usd
  from bridge_out o
  inner join tokens.transfers r
    on r.blockchain = 'ethereum'
   and r."from" = o.bridge_address
   and r."to" = o.wallet_address
   and r.block_time > o.out_block_time
   and r.block_time <= o.out_block_time + interval '{{max_return_hours}}' hour
  group by 1, 2, 3, 4, 5
),

post_return_distribution as (
  select
    r.wallet_address,
    count(distinct t."to") as post_return_unique_recipients,
    sum(coalesce(t.amount_usd, 0)) as post_return_outbound_usd
  from bridge_return r
  left join tokens.transfers t
    on t.blockchain = 'ethereum'
   and t."from" = r.wallet_address
   and t.block_time > r.return_block_time
   and t.block_time <= r.return_block_time + interval '{{post_return_hours}}' hour
  group by 1
),

finalized as (
  select
    'evm-known-negative-bridge-return-' ||
      lower(substr(to_hex(r.wallet_address), 1, 10)) ||
      '-' ||
      date_format(r.out_block_time, '%Y-%m-%d') as case_id,
    'evm' as chain,
    'known_negative' as cohort,
    'bridge_return' as case_type,
    '0x' || lower(to_hex(r.wallet_address)) as subject_address,
    cast(null as varchar) as entity_key,
    'primary_wallet' as subject_role,
    cast(date_trunc('day', r.out_block_time) as timestamp) as window_start_at,
    cast(date_trunc('day', r.return_block_time + interval '1' day) as timestamp) as window_end_at,
    cast(r.return_block_time as timestamp) as observation_cutoff_at,
    cast(r.return_block_time + interval '{{post_return_hours}}' hour as timestamp) as detection_deadline_at,
    'Qorvi should keep shadow_exit_risk below high and surface bridge-return contradiction evidence.' as expected_outcome,
    'shadow_exit_risk' as expected_signal,
    'bridge_return' as expected_route,
    '0x' || lower(to_hex(r.out_tx_hash)) as source_tx_hash,
    r.out_block_number as source_block_number,
    'Bridge return candidate from Dune' as source_title,
    '{{source_url}}' as source_url,
    'Wallet bridged out and received funds back from the same bridge within the configured return window, with limited downstream distribution afterwards.' as narrative,
    'Review whether the post-return path is operational routing or a genuine exit sequence.' as analyst_note,
    json_object(
      'bridgeAddress', '0x' || lower(to_hex(r.bridge_address)),
      'returnTxHash', '0x' || lower(to_hex(r.return_tx_hash)),
      'returnBlockNumber', r.return_block_number,
      'returnAmountUsd', cast(r.return_amount_usd as double),
      'postReturnUniqueRecipients', coalesce(p.post_return_unique_recipients, 0),
      'postReturnOutboundUsd', cast(coalesce(p.post_return_outbound_usd, 0) as double),
      'queryConfidence', 0.78
    ) as metadata_json
  from bridge_return r
  left join post_return_distribution p
    on p.wallet_address = r.wallet_address
  where coalesce(p.post_return_unique_recipients, 0) <= {{max_post_return_recipients}}
    and coalesce(p.post_return_outbound_usd, 0) <= {{max_post_return_outbound_usd}}
)

select
  case_id,
  chain,
  cohort,
  case_type,
  subject_address,
  entity_key,
  subject_role,
  cast(window_start_at as varchar) as window_start_at,
  cast(window_end_at as varchar) as window_end_at,
  cast(observation_cutoff_at as varchar) as observation_cutoff_at,
  cast(detection_deadline_at as varchar) as detection_deadline_at,
  expected_outcome,
  expected_signal,
  expected_route,
  source_tx_hash,
  source_block_number,
  source_title,
  source_url,
  narrative,
  analyst_note,
  metadata_json
from finalized
order by window_start_at desc
limit {{limit}};
