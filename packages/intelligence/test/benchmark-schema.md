# Intelligence Benchmark Schema

`packages/intelligence`의 benchmark runner는 아래 개념을 기준으로 fixture를 평가한다.

## Scenario

- `name`
  - 고유 시나리오 이름
- `cohort`
  - `known_positive`
  - `false_positive`
  - `negative`
- `description`
  - 사람이 읽는 설명
- `cluster`
  - `ClusterSignal` fixture
- `shadowExit`
  - `ShadowExitSignal` fixture
- `firstConnection`
  - `FirstConnectionSignal` fixture
- `expectations`
  - 하나 이상의 score expectation

## Expectation

- `scoreName`
  - `cluster_score`
  - `shadow_exit_risk`
  - `alpha_score`
- `minRating`
  - 허용 최소 rating
- `maxRating`
  - 허용 최대 rating
- `requireSuppression`
  - suppression metadata가 반드시 있어야 하는지
- `requireContradiction`
  - contradiction metadata가 반드시 있어야 하는지
- `requireRatingBlock`
  - rating block metadata가 반드시 있어야 하는지

## Summary Metrics

- `precisionAtHigh`
  - high prediction 중 known positive 비율
- `falsePositiveRate`
  - false_positive / negative cohort에서 high가 나온 비율
- `truePositiveHigh`
  - known positive cohort에서 high로 유지된 수
- `falsePositiveHigh`
  - false_positive / negative cohort에서 잘못 high가 나온 수

## Current Goal

release 전에 최소한 아래는 자동으로 확인한다.

- known positive가 여전히 high로 유지되는지
- treasury/MM/internal rebalance false positive가 high로 회귀하지 않는지
- contradiction/suppression/rating-block metadata가 여전히 surfacing되는지

## Current Runner

로컬이나 CI에서 아래 worker mode로 benchmark를 바로 실행할 수 있다.

```bash
QORVI_WORKER_MODE=analysis-benchmark-fixture go run ./apps/workers
```
