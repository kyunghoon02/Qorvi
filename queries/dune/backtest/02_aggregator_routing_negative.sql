-- Qorvi backtest candidate query
-- Case type: aggregator_routing
-- Cohort: known_negative
--
-- Purpose:
-- Find wallets whose activity is dominated by aggregator / router infrastructure so they can
-- serve as cluster / alpha false-positive controls.
--
-- Notes:
-- 1. Replace the category filter with the exact router universe your analysts trust.
-- 2. Query output is for review only; do not treat it as final truth.

with router_labels as (
  select
    blockchain,
    address,
    name,
    category,
    type
  from labels.addresses
  where blockchain = 'ethereum'
    and (
      lower(category) like '%aggregator%'
      or lower(category) like '%router%'
    )
),

wallet_router_flows as (
  select
    t.block_time,
    t.block_number,
    t.tx_hash,
    case
      when rl.address = t."to" then t."from"
      else t."to"
    end as wallet_address,
    rl.address as router_address,
    coalesce(t.amount_usd, 0) as amount_usd
  from tokens.transfers t
  inner join router_labels rl
    on rl.blockchain = 'ethereum'
   and (rl.address = t."to" or rl.address = t."from")
  where t.blockchain = 'ethereum'
    and t.block_time >= cast('{{window_start}}' as timestamp)
    and t.block_time < cast('{{window_end}}' as timestamp)
),

wallet_totals as (
  select
    wallet_address,
    count(*) as router_touch_count,
    count(distinct router_address) as unique_router_count,
    min(block_time) as first_router_touch_at,
    max(block_time) as last_router_touch_at,
    min_by(tx_hash, block_time) as anchor_tx_hash,
    min_by(block_number, block_time) as anchor_block_number,
    sum(amount_usd) as router_amount_usd
  from wallet_router_flows
  group by 1
),

wallet_transfer_totals as (
  select
    wallet_address,
    sum(total_transfers) as total_transfers
  from (
    select t."from" as wallet_address, count(*) as total_transfers
    from tokens.transfers t
    where t.blockchain = 'ethereum'
      and t.block_time >= cast('{{window_start}}' as timestamp)
      and t.block_time < cast('{{window_end}}' as timestamp)
    group by 1

    union all

    select t."to" as wallet_address, count(*) as total_transfers
    from tokens.transfers t
    where t.blockchain = 'ethereum'
      and t.block_time >= cast('{{window_start}}' as timestamp)
      and t.block_time < cast('{{window_end}}' as timestamp)
    group by 1
  )
  group by 1
),

router_dominant as (
  select
    w.wallet_address,
    w.router_touch_count,
    w.unique_router_count,
    w.first_router_touch_at,
    w.last_router_touch_at,
    w.anchor_tx_hash,
    w.anchor_block_number,
    w.router_amount_usd,
    coalesce(t.total_transfers, 0) as total_transfers,
    case
      when coalesce(t.total_transfers, 0) = 0 then 0
      else cast(w.router_touch_count as double) / cast(t.total_transfers as double)
    end as router_touch_ratio
  from wallet_totals w
  left join wallet_transfer_totals t
    on t.wallet_address = w.wallet_address
  where w.router_touch_count >= {{min_router_touch_count}}
    and w.unique_router_count >= {{min_unique_router_count}}
),

finalized as (
  select
    'evm-known-negative-aggregator-routing-' ||
      lower(substr(to_hex(wallet_address), 1, 10)) ||
      '-' ||
      date_format(first_router_touch_at, '%Y-%m-%d') as case_id,
    'evm' as chain,
    'known_negative' as cohort,
    'aggregator_routing' as case_type,
    '0x' || lower(to_hex(wallet_address)) as subject_address,
    cast(null as varchar) as entity_key,
    'primary_wallet' as subject_role,
    cast(date_trunc('day', first_router_touch_at) as timestamp) as window_start_at,
    cast(date_trunc('day', last_router_touch_at + interval '1' day) as timestamp) as window_end_at,
    cast(last_router_touch_at as timestamp) as observation_cutoff_at,
    cast(last_router_touch_at + interval '12' hour as timestamp) as detection_deadline_at,
    'Qorvi should keep cluster and alpha narratives below high for router-dominant wallets.' as expected_outcome,
    'cluster_score' as expected_signal,
    'aggregator_routing' as expected_route,
    '0x' || lower(to_hex(anchor_tx_hash)) as source_tx_hash,
    anchor_block_number as source_block_number,
    'Aggregator routing candidate from Dune' as source_title,
    '{{source_url}}' as source_url,
    'Wallet activity is dominated by aggregator or router infrastructure across the selected window, making it a high-value false-positive control.' as narrative,
    'Review whether the wallet still shows any non-router peer overlap before promoting it as a negative control.' as analyst_note,
    json_object(
      'routerTouchCount', router_touch_count,
      'uniqueRouterCount', unique_router_count,
      'totalTransfers', total_transfers,
      'routerTouchRatio', cast(router_touch_ratio as double),
      'routerAmountUsd', cast(router_amount_usd as double),
      'queryConfidence', 0.74
    ) as metadata_json
  from router_dominant
  where router_touch_ratio >= {{min_router_touch_ratio}}
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
