# Qorvi Dune Backtest Queries

이 디렉토리는 `실제 Dune saved query`를 만들기 위한 production-oriented SQL 템플릿을 둔다.

목표는 다음 두 가지다.

1. Qorvi backtest에 넣을 `실재한 candidate case`를 수집한다.
2. query output이 Qorvi `dune-backtest-candidates` normalizer와 바로 맞도록 한다.

## Output Contract

모든 query는 아래 컬럼을 반환해야 한다.

- `case_id`
- `chain`
- `cohort`
- `case_type`
- `subject_address`
- `entity_key`
- `subject_role`
- `window_start_at`
- `window_end_at`
- `observation_cutoff_at`
- `detection_deadline_at`
- `expected_outcome`
- `expected_signal`
- `expected_route`
- `source_tx_hash`
- `source_block_number`
- `source_title`
- `source_url`
- `narrative`
- `analyst_note`

review metadata는 query가 아니라 사람이 후처리에서 넣는다.

## Query Order

1. `01_bridge_return_negative.sql`
2. `02_aggregator_routing_negative.sql`
3. `03_smart_money_early_entry_positive.sql`

이 순서가 맞다. negative부터 precision hardening에 바로 쓰이기 때문이다.

## Production Notes

### 1. query는 truth가 아니다

query 결과는 candidate일 뿐이다.

- Qorvi normalizer
- analyst review
- manifest promotion

이 3단계를 거쳐야 release gate dataset이 된다.

### 2. smart money positive는 curated wallet universe가 필요하다

Dune 기본 curated tables만으로 “smart money”를 바로 확정하지 않는다.

production에서는 아래 둘 중 하나가 필요하다.

- 내부 curated quality-wallet table
- reviewer가 관리하는 Dune uploaded table / materialized view

즉 `03_smart_money_early_entry_positive.sql`은 production용 뼈대이며, quality wallet universe는 네 팀이 관리해야 한다.

### 3. chain scope

현재 템플릿은 주로 EVM curated tables를 기준으로 설계했다.

- `tokens.transfers`
- `labels.addresses`
- `dex.trades`

Solana backtest도 필요하지만, 첫 release gate는 EVM negative + positive부터 채우는 것이 맞다.

## Runbook

실제 saved query 파라미터 기본값과 review 절차는 아래 문서를 따른다.

- `/Users/kh/Github/Qorvi/docs/runbooks/dune-backtest-collection.md`
