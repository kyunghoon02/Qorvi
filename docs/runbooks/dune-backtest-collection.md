# Dune Backtest Collection Runbook

이 문서는 Qorvi의 Dune backtest candidate 수집을 실제 운영 기준으로 실행하는 절차다.

목표는 `실행 가능한 query`를 만드는 것이 아니라, `release gate에 올려도 되는 quality candidate`를 만드는 것이다.

## 1. 초기 rollout 순서

첫 rollout은 아래 순서로 간다.

1. `bridge_return`
2. `aggregator_routing`
3. `smart_money_early_entry`

이 순서가 맞다. `known_negative`가 precision hardening에 더 직접적이기 때문이다.

## 2. Bridge Return Saved Query Defaults

대상 SQL:

- `/Users/kh/Github/Qorvi/queries/dune/backtest/01_bridge_return_negative.sql`

초기 파라미터 권장값:

- `window_start`
  - 최근 30일 시작 시점
- `window_end`
  - 현재 시점
- `min_bridge_usd`
  - `25000`
- `max_return_hours`
  - `48`
- `post_return_hours`
  - `24`
- `max_post_return_recipients`
  - `3`
- `max_post_return_outbound_usd`
  - `50000`
- `limit`
  - `100`
- `source_url`
  - query runbook 또는 internal review link

운영 의도:

- 너무 작은 bridge noise는 제외
- bridge out 후 빠르게 돌아온 operational round-trip을 우선 포착
- return 이후 fan-out이 작아야 false-positive negative candidate로 쓸 수 있음

## 3. Aggregator Routing Saved Query Defaults

대상 SQL:

- `/Users/kh/Github/Qorvi/queries/dune/backtest/02_aggregator_routing_negative.sql`

초기 파라미터 권장값:

- `window_start`
  - 최근 30일 시작 시점
- `window_end`
  - 현재 시점
- `min_router_touch_count`
  - `8`
- `min_unique_router_count`
  - `2`
- `min_router_touch_ratio`
  - `0.55`
- `limit`
  - `100`
- `source_url`
  - query runbook 또는 internal review link

운영 의도:

- router 1개만 잠깐 찍은 지갑은 제외
- 여러 router와 반복 상호작용한 noisy wallet을 우선 수집
- cluster / alpha false positive control로 쓰기 적합한 후보만 남김

## 4. Smart Money Early Entry Defaults

대상 SQL:

- `/Users/kh/Github/Qorvi/queries/dune/backtest/03_smart_money_early_entry_positive.sql`

전제:

- `quality_wallets` CTE를 실제 curated universe로 교체해야 한다.

초기 파라미터 권장값:

- `window_start`
  - 최근 90일 시작 시점
- `window_end`
  - 현재 시점
- `min_entry_usd`
  - `20000`
- `min_broader_wallets`
  - `25`
- `min_lead_hours`
  - `12`
- `hold_window_hours`
  - `72`
- `min_subsequent_trades`
  - `1`
- `limit`
  - `100`
- `source_url`
  - curated wallet universe 설명 문서

운영 의도:

- 너무 작은 우연한 진입은 제외
- broader crowding보다 의미 있는 선행 시간차가 있어야 함
- entry 직후 dump가 아닌 최소한의 hold / follow-through가 있어야 함

## 5. Review Checklist

candidate export로 승격하기 전에 아래를 반드시 본다.

1. `주소가 실제로 case narrative와 일치하는가`
2. `anchor tx가 진짜 핵심 tx인가`
3. `기간이 너무 넓어서 unrelated flow가 섞이지 않았는가`
4. `같은 entity family를 과다 중복 채우고 있지 않은가`
5. `expected_signal`과 `expected_route`가 analyst 관점에서 맞는가`

negative 추가 체크:

1. 실제로 benign / operational explanation이 강한가
2. return / router dominance가 단발성이 아니라 case로 쓸 만큼 반복적인가
3. 나중에 high score가 뜨면 진짜 false positive라고 부를 수 있는가

positive 추가 체크:

1. hindsight만으로 선정한 게 아닌가
2. curated quality-wallet universe에 넣을 만큼 quality가 있는 wallet인가
3. token event 자체가 너무 micro-noise는 아닌가

## 6. Candidate Export Execution

1. Dune saved query 실행
2. API 또는 export로 raw result JSON 저장
3. 아래 명령으로 candidate export 생성

```bash
QORVI_DUNE_QUERY_RESULT_PATH=/path/to/dune-result.json \
QORVI_DUNE_QUERY_NAME=bridge-return-negative \
QORVI_DUNE_CANDIDATE_EXPORT_PATH=packages/intelligence/test/dune-backtest-candidates.json \
QORVI_WORKER_MODE=analysis-dune-backtest-normalize \
go run ./apps/workers
```

4. reviewer가 `caseId`와 `review.*` 필드를 채운다
5. candidate export validation

```bash
QORVI_DUNE_CANDIDATE_EXPORT_PATH=packages/intelligence/test/dune-backtest-candidates.json \
QORVI_WORKER_MODE=analysis-dune-backtest-candidate-validate \
go run ./apps/workers
```

6. reviewed candidates를 manifest로 승격

```bash
QORVI_DUNE_CANDIDATE_EXPORT_PATH=packages/intelligence/test/dune-backtest-candidates.json \
QORVI_BACKTEST_MANIFEST_PATH=packages/intelligence/test/backtest-manifest.json \
QORVI_WORKER_MODE=analysis-dune-backtest-promote \
go run ./apps/workers
```

## 7. Naming Convention

### Saved query naming

Dune saved query 이름은 아래 형식을 고정한다.

`qorvi_backtest_<chain>_<cohort>_<case_type>_v<version>`

예:

- `qorvi_backtest_evm_known_negative_bridge_return_v1`
- `qorvi_backtest_evm_known_negative_aggregator_routing_v1`
- `qorvi_backtest_evm_known_positive_smart_money_early_entry_v1`

규칙:

1. `qorvi_backtest_` prefix 고정
2. chain / cohort / case_type를 이름에 드러낸다
3. query 수정이 release gate 의미를 바꾸면 `v2`로 올린다
4. 같은 이름에서 ad-hoc overwrite를 반복하지 않는다

### Candidate export file naming

candidate export 파일명은 아래 형식을 권장한다.

`dune-backtest-candidates-<chain>-<case-type>-<yyyy-mm-dd>.json`

예:

- `dune-backtest-candidates-evm-bridge-return-2026-03-31.json`
- `dune-backtest-candidates-evm-aggregator-routing-2026-03-31.json`
- `dune-backtest-candidates-evm-smart-money-early-entry-2026-03-31.json`

규칙:

1. 한 파일에 한 query family를 우선 저장한다
2. reviewer 전/후 파일을 덮어쓰지 말고 Git diff가 남게 관리한다
3. release candidate로 넘어간 파일만 `backtest-manifest.json`에 반영한다

### Case ID naming

`caseId`는 아래 형식을 권장한다.

`<chain>-<cohort>-<case-type>-<short-anchor>-<yyyy-mm-dd>`

예:

- `evm-known-negative-bridge-return-0x1234abcd-2026-03-01`
- `evm-known-positive-smart-money-0xabcd1234-2026-02-14`

규칙:

1. 사람이 보고도 어떤 케이스인지 유추 가능해야 한다
2. 같은 이벤트 family 안에서 stable해야 한다
3. random uuid를 기본값으로 쓰지 않는다

## 8. Query Preset File

실행용 기본값은 아래 파일에 고정한다.

- `/Users/kh/Github/Qorvi/queries/dune/backtest/query-presets.json`

이 파일은 아래를 관리한다.

- saved query 이름
- SQL 경로
- cohort / caseType / chain
- candidate export 출력 경로
- query 파라미터 기본값

preset validation:

```bash
QORVI_DUNE_QUERY_PRESET_PATH=queries/dune/backtest/query-presets.json \
QORVI_WORKER_MODE=analysis-dune-backtest-preset-validate \
go run ./apps/workers
```

규칙:

1. query 실행 시 사람마다 임의로 파라미터를 바꾸지 않는다
2. release gate 의미를 바꾸는 수정이면 preset도 version bump에 준해 관리한다
3. ad-hoc 실험값은 preset 파일이 아니라 별도 메모에 남긴다

## 9. Promotion Gate

candidate를 manifest로 올리기 전에 아래 조건을 만족해야 한다.

- `review.curatedBy` 존재
- `review.reviewStatus`가 `reviewed` 또는 `approved`
- `source_tx_hash` 존재
- `source_url` 존재
- `expected_signal`, `expected_route`, `expected_outcome` 존재

## 10. First Sprint Checklist

첫 sprint에서는 아래 순서로 움직인다.

1. `bridge_return` saved query 생성
2. 최근 30일 기준 candidate 10~20개 추출
3. reviewer가 10개까지 줄이고 `review.*` 필드 기입
4. candidate validate 실행
5. manifest promote 실행
6. 같은 순서로 `aggregator_routing` 반복
7. 그 다음 `smart_money_early_entry`로 이동

각 단계에서 확인할 것:

- query output contract 누락 없음
- anchor tx가 실제 narrative를 대표함
- duplicate entity family 과다 없음
- expected signal / route가 analyst judgement와 일치
- promoted manifest가 validation 통과

완료 기준:

- `bridge_return` reviewed case 10개
- `aggregator_routing` reviewed case 10개
- `smart_money_early_entry` reviewed case 10개
- promote 후 manifest validation 통과

## 11. Initial Target

첫 번째 collection sprint 목표:

- `bridge_return` 10개
- `aggregator_routing` 10개
- `smart_money_early_entry` 10개

그 다음 목표:

- known_negative 20
- known_positive 20
- control 20
