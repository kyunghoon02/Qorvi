-- Qorvi backtest candidate query
-- Case type: smart_money_early_entry
-- Cohort: known_positive
--
-- Purpose:
-- Find high-quality wallets that entered a token before broader crowding and held through the
-- initial breakout window. This is a production template and requires a curated quality-wallet
-- universe supplied by your team.
--
-- Required upstream input:
-- Replace the `quality_wallets` CTE with an uploaded table or saved query that contains your
-- reviewed smart-money / quality-wallet universe.
--
-- Notes:
-- 1. Do not rely on raw Dune labels alone for smart-money attribution.
-- 2. Human review is still required before promotion to backtest-manifest.json.

with quality_wallets as (
  -- Replace this CTE with your curated smart-money universe.
  -- Required columns:
  --   wallet_address (varbinary)
  --   quality_label (varchar)
  --   source_url (varchar)
  select
    cast(null as varbinary) as wallet_address,
    cast(null as varchar) as quality_label,
    cast(null as varchar) as source_url
  where false
),

entry_candidates as (
  select
    dt.block_time as entry_block_time,
    dt.block_number as entry_block_number,
    dt.tx_hash as entry_tx_hash,
    dt.blockchain,
    dt.trader as wallet_address,
    dt.token_bought_address as token_address,
    coalesce(dt.token_bought_symbol, 'UNKNOWN') as token_symbol,
    sum(coalesce(dt.amount_usd, 0)) as entry_amount_usd
  from dex.trades dt
  inner join quality_wallets qw
    on qw.wallet_address = dt.trader
  where dt.blockchain = 'ethereum'
    and dt.block_time >= cast('{{window_start}}' as timestamp)
    and dt.block_time < cast('{{window_end}}' as timestamp)
    and coalesce(dt.amount_usd, 0) >= {{min_entry_usd}}
  group by 1, 2, 3, 4, 5, 6, 7
),

broader_crowding as (
  select
    token_bought_address as token_address,
    min(block_time) as broader_crowding_at
  from dex.trades
  where blockchain = 'ethereum'
    and block_time >= cast('{{window_start}}' as timestamp)
    and block_time < cast('{{window_end}}' as timestamp)
  group by 1
  having count(distinct trader) >= {{min_broader_wallets}}
),

early_entries as (
  select
    e.*,
    c.broader_crowding_at,
    date_diff('hour', e.entry_block_time, c.broader_crowding_at) as lead_hours_before_crowding
  from entry_candidates e
  inner join broader_crowding c
    on c.token_address = e.token_address
  where e.entry_block_time < c.broader_crowding_at
    and date_diff('hour', e.entry_block_time, c.broader_crowding_at) >= {{min_lead_hours}}
),

hold_check as (
  select
    e.wallet_address,
    e.token_address,
    max(dt.block_time) as latest_trade_at,
    count(*) as subsequent_trades
  from early_entries e
  left join dex.trades dt
    on dt.blockchain = 'ethereum'
   and dt.trader = e.wallet_address
   and dt.token_bought_address = e.token_address
   and dt.block_time > e.entry_block_time
   and dt.block_time <= e.entry_block_time + interval '{{hold_window_hours}}' hour
  group by 1, 2
),

priced_breakout as (
  select
    p.contract_address as token_address,
    max(p.price) as max_price_in_window,
    min(p.price) as min_price_in_window
  from prices.latest p
  where p.blockchain = 'ethereum'
  group by 1
),

finalized as (
  select
    'evm-known-positive-smart-money-' ||
      lower(substr(to_hex(e.wallet_address), 1, 10)) ||
      '-' ||
      lower(substr(to_hex(e.token_address), 1, 10)) ||
      '-' ||
      date_format(e.entry_block_time, '%Y-%m-%d') as case_id,
    'evm' as chain,
    'known_positive' as cohort,
    'smart_money_early_entry' as case_type,
    '0x' || lower(to_hex(e.wallet_address)) as subject_address,
    cast(null as varchar) as entity_key,
    'primary_wallet' as subject_role,
    cast(date_trunc('day', e.entry_block_time) as timestamp) as window_start_at,
    cast(date_trunc('day', e.entry_block_time + interval '{{hold_window_hours}}' hour) as timestamp) as window_end_at,
    cast(e.broader_crowding_at as timestamp) as observation_cutoff_at,
    cast(e.broader_crowding_at + interval '12' hour as timestamp) as detection_deadline_at,
    'Qorvi should surface a high alpha_score before the broader follow-on wave for this wallet and token pair.' as expected_outcome,
    'alpha_score' as expected_signal,
    'funding_inflow' as expected_route,
    '0x' || lower(to_hex(e.entry_tx_hash)) as source_tx_hash,
    e.entry_block_number as source_block_number,
    'Smart money early-entry candidate from Dune' as source_title,
    coalesce(qw.source_url, '{{source_url}}') as source_url,
    'Curated quality wallet entered the token before broader crowding and maintained exposure through the configured hold window.' as narrative,
    'Review whether the token narrative is real alpha or a short-lived reflexive burst before promotion.' as analyst_note,
    json_object(
      'tokenAddress', '0x' || lower(to_hex(e.token_address)),
      'tokenSymbol', e.token_symbol,
      'entryAmountUsd', cast(e.entry_amount_usd as double),
      'leadHoursBeforeCrowding', e.lead_hours_before_crowding,
      'subsequentTrades', coalesce(h.subsequent_trades, 0),
      'qualityLabel', qw.quality_label,
      'queryConfidence', 0.81
    ) as metadata_json
  from early_entries e
  inner join quality_wallets qw
    on qw.wallet_address = e.wallet_address
  left join hold_check h
    on h.wallet_address = e.wallet_address
   and h.token_address = e.token_address
  left join priced_breakout pb
    on pb.token_address = e.token_address
  where coalesce(h.subsequent_trades, 0) >= {{min_subsequent_trades}}
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
