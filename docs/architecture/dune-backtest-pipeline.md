# Qorvi Dune Backtest Collection Pipeline

## 목적

이 문서는 `실재한 온체인 케이스`를 Dune에서 수집해 Qorvi backtest dataset으로 연결하는 운영 파이프라인을 정의한다.

목표는 단순히 주소를 많이 모으는 것이 아니다.

1. 실제 존재했던 positive / negative / control 케이스만 수집한다.
2. 각 케이스에 `기간`, `tx 근거`, `citation`, `review 상태`를 남긴다.
3. Dune query 결과가 곧바로 Qorvi backtest manifest review 흐름으로 이어지게 한다.

이 파이프라인은 `웹 크롤링`을 쓰지 않는다. 공식 Dune API와 저장된 query 결과만 사용한다.

## 왜 Dune인가

- query 결과를 JSON으로 안정적으로 수집할 수 있다.
- 쿼리 로직을 Git/문서와 함께 관리하기 쉽다.
- positive / negative / control 케이스를 같은 스키마로 추출할 수 있다.
- 실데이터 기반 backtest에 필요한 기간/route/filter를 세밀하게 정의할 수 있다.

## 공식 API 기준 흐름

1. Dune에 `saved query`를 만든다.
2. API로 query를 실행하거나 최신 결과를 읽는다.
3. 결과를 `candidate export` JSON으로 저장한다.
4. 사람이 검토해 실제 backtest manifest로 승격한다.

참고할 공식 문서:

- [Data API Overview](https://docs.dune.com/api-reference/api-overview)
- [Get Latest Query Result](https://docs.dune.com/api-reference/executions/endpoint/get-query-result)
- [Get Execution Status](https://docs.dune.com/api-reference/executions/endpoint/get-execution-status)
- [Get Execution Result](https://docs.dune.com/api-reference/executions/endpoint/get-execution-result)

## Qorvi 기준 파이프라인

### 1. Query families

최소한 아래 query family를 운영한다.

#### known_positive

- `smart_money_early_entry`
- `treasury_redistribution`
- `market_maker_handoff`
- `cross_chain_rotation`

#### known_negative

- `bridge_return`
- `aggregator_routing`
- `exchange_heavy_retail`
- `treasury_rebalance`
- `deployer_fee_collector`

#### control

- `active_wallet_control`

### 2. Query output contract

Dune query는 아래 컬럼을 반드시 반환해야 한다.

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

권장 추가 컬럼:

- `counterparty_address`
- `token_address`
- `token_symbol`
- `project_slug`
- `bridge_name`
- `exchange_name`
- `query_confidence`
- `dedup_key`

### 3. Candidate export shape

Dune 결과는 바로 release gate에 쓰지 않는다. 먼저 아래 intermediate export로 저장한다.

- 파일 위치 예시:
  - `/Users/kh/Github/Qorvi/packages/intelligence/test/dune-backtest-candidates.json`
- 템플릿:
  - `/Users/kh/Github/Qorvi/packages/intelligence/test/dune-backtest-candidates.template.json`

이 candidate export는 `raw but structured` 단계다.

- Dune query metadata는 남긴다.
- 실제 tx와 citation은 유지한다.
- 하지만 reviewer / approval은 아직 비어 있을 수 있다.

### 4. Human review

candidate export는 analyst가 검토한 뒤에만 manifest로 승격한다.

필수 review 체크:

1. 이 케이스가 실제 positive / negative / control인지
2. 기간이 올바른지
3. tx hash가 실제 narrative를 지지하는지
4. entity family 중복이 과한지
5. 예상 signal / route가 맞는지

review를 통과한 케이스만 아래 파일로 옮긴다.

- `/Users/kh/Github/Qorvi/packages/intelligence/test/backtest-manifest.json`

candidate export에서 reviewer가 채워야 하는 필드:

- `caseId`
- `review.curatedBy`
- `review.reviewStatus`
- `review.caseTicket`
- 필요하면
  - `review.expectedHighSignals`
  - `review.expectedSuppressed`

## 운영 규칙

### Rule 1: Dune는 `candidate sourcing`까지만

Dune query 결과를 바로 truth로 취급하지 않는다.

- Dune는 후보를 만든다.
- reviewer가 truth를 확정한다.
- Qorvi manifest가 최종 ground truth다.

### Rule 2: one case, one anchor tx

모든 케이스는 최소한 하나의 anchor tx를 가져야 한다.

- positive:
  - entry tx, distribution tx, handoff tx 등
- negative:
  - bridge out / return tx, rebalance tx 등
- control:
  - representative active tx

### Rule 3: case family dedup

같은 entity family나 같은 이벤트 wave를 과도하게 중복 샘플링하지 않는다.

예:

- 같은 treasury가 하루에 10번 보낸 redistribution을 10개 case로 세지 않는다.
- 같은 router hub에서 파생된 비슷한 negative를 과하게 중복 채우지 않는다.

### Rule 4: case quality over volume

초기 목표는 많이 모으는 것이 아니라, `정확한 20/20/20`을 먼저 채우는 것이다.

우선순위:

1. known_negative 20
2. known_positive 20
3. control 20

## 추천 쿼리 작성 전략

### known_positive

- smart money / quality wallet overlap을 선행 조건으로 둔다.
- 이후 price expansion / volume expansion / copy-trade wave를 후행 조건으로 둔다.
- “결과가 좋았던 진입”만 남겨 hindsight-positive를 만든다.

주의:

- 결과가 좋았다는 이유만으로 무조건 smart money로 분류하지 않는다.
- 반드시 early entry timing과 route evidence가 같이 있어야 한다.

### known_negative

- false positive bucket을 더 엄격히 만든다.
- 예:
  - bridge out 후 return
  - exchange 접점은 많지만 alpha route 없음
  - aggregator routing 반복
  - treasury internal rebalance

negative는 precision hardening에 직접 연결되므로, positive보다 먼저 채운다.

### control

- activity는 높지만 quality overlap / route conviction이 낮은 지갑을 뽑는다.
- “점수가 과하게 오르면 안 되는” 샘플이다.

## Dune -> Qorvi 매핑

candidate export row 한 건은 최종적으로 아래처럼 매핑된다.

- `chain` -> `dataset.chain`
- `cohort` -> `dataset.cohort`
- `case_type` -> `dataset.caseType`
- `subject_address` / `entity_key` -> `dataset.subjects`
- `window_*` -> `dataset.window`
- `expected_outcome` / `narrative` / `expected_signal` / `expected_route` -> `dataset.groundTruth`
- `source_tx_hash`, `source_block_number` -> `dataset.groundTruth.onchainEvidence`
- `source_title`, `source_url` -> `dataset.groundTruth.sourceCitations`

reviewer가 채우는 필드:

- `curatedBy`
- `reviewStatus`
- `caseTicket`
- acceptance expectations

## 초기 실행 순서

1. Dune query 3개부터 시작
   - `bridge_return`
   - `aggregator_routing`
   - `smart_money_early_entry`
2. 각 query에서 10~20개 candidate를 수집
3. candidate export JSON으로 저장
4. analyst review 후 manifest로 승격
5. `analysis-backtest-manifest-validate`로 검증

## 현재 Qorvi 저장소 기준 산출물

- backtest plan:
  - `/Users/kh/Github/Qorvi/docs/architecture/analysis-engine-hardening-plan.md`
- backtest manifest schema:
  - `/Users/kh/Github/Qorvi/packages/intelligence/test/backtest-manifest-schema.md`
- backtest manifest template:
  - `/Users/kh/Github/Qorvi/packages/intelligence/test/backtest-manifest.template.json`
- Dune candidate template:
  - `/Users/kh/Github/Qorvi/packages/intelligence/test/dune-backtest-candidates.template.json`
- Dune candidate normalizer worker mode:
  - `QORVI_WORKER_MODE=analysis-dune-backtest-normalize`

## 현재 구현된 normalizer

raw Dune execution result JSON이 아래 env로 들어오면 candidate export JSON으로 정규화된다.

- `QORVI_DUNE_QUERY_RESULT_PATH`
- `QORVI_DUNE_QUERY_NAME`
- `QORVI_DUNE_CANDIDATE_EXPORT_PATH`

실행 예시:

```bash
QORVI_DUNE_QUERY_RESULT_PATH=/path/to/dune-result.json \
QORVI_DUNE_QUERY_NAME=bridge-return-negative \
QORVI_DUNE_CANDIDATE_EXPORT_PATH=packages/intelligence/test/dune-backtest-candidates.json \
QORVI_WORKER_MODE=analysis-dune-backtest-normalize \
go run ./apps/workers
```

이 단계에서 하는 일:

1. Dune result JSON 로드
2. Qorvi candidate export row로 정규화
3. required field 검증
4. candidate export JSON 파일 저장

candidate export만 검증하고 싶으면:

```bash
QORVI_DUNE_CANDIDATE_EXPORT_PATH=packages/intelligence/test/dune-backtest-candidates.json \
QORVI_WORKER_MODE=analysis-dune-backtest-candidate-validate \
go run ./apps/workers
```

주의:

- 이 output은 아직 reviewed manifest가 아니다.
- reviewer 확인 후에만 `backtest-manifest.json`으로 승격한다.

reviewed candidate export를 manifest로 승격하려면:

```bash
QORVI_DUNE_CANDIDATE_EXPORT_PATH=packages/intelligence/test/dune-backtest-candidates.json \
QORVI_BACKTEST_MANIFEST_PATH=packages/intelligence/test/backtest-manifest.json \
QORVI_WORKER_MODE=analysis-dune-backtest-promote \
go run ./apps/workers
```

## 다음 구현 우선순위

1. Dune candidate export validator
2. Dune query result -> candidate export normalizer
3. reviewed candidate export -> backtest manifest writer
4. replay runner와 backtest metrics collector
