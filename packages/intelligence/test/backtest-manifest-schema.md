# Intelligence Backtest Manifest Schema

`packages/intelligence/test/backtest-manifest.json`은 `실제 존재했던 온체인 케이스`만 담는 운영용 backtest 입력 파일이다.

이 manifest는 fixture benchmark와 다르다.

- benchmark
  - 합성된 signal fixture 중심
  - score regression gate 용도
- backtest manifest
  - 실재한 주소, 실재한 기간, 실재한 tx ref, 실재한 analyst 근거가 필수
  - 출시 전 성능 검증과 주간 analyst review 용도

## Hard Rules

1. 합성 케이스 금지
   - `provenance.synthetic`은 항상 `false`
2. 실제 근거 필수
   - `groundTruth.sourceCitations` 최소 1개
   - `groundTruth.onchainEvidence` 최소 1개
3. reviewer 상태 필수
   - `provenance.reviewStatus`는 `reviewed` 또는 `approved`
4. 기간 명시 필수
   - `window.startAt`
   - `window.endAt`
   - 둘 다 `RFC3339`

## Top-Level Fields

- `version`
  - manifest 버전 문자열
- `policy`
  - 실제 데이터 강제 정책
- `datasets`
  - 개별 backtest case 배열

## Policy

- `requireRealWorldData`
- `requireSourceCitations`
- `requireOnchainEvidence`
- `requireReviewedCases`
- `minimumCasesPerCohort`
- `minimumCasesPerCaseType`

운영용 manifest는 이 값들을 모두 적극적으로 사용한다.

## Dataset

- `id`
  - 고유 case id
- `chain`
  - `evm`, `solana` 등
- `cohort`
  - `known_positive`
  - `known_negative`
  - `control`
- `caseType`
  - 예:
  - `smart_money_early_entry`
  - `treasury_redistribution`
  - `market_maker_handoff`
  - `bridge_return`
  - `aggregator_routing`
  - `active_wallet_control`
- `description`
  - analyst가 읽는 요약
- `subjects`
  - case 핵심 subject 주소 / entity
- `window`
  - replay 기간
- `groundTruth`
  - 기대 outcome, expected signal/route, 근거 링크와 tx
- `acceptance`
  - 기대 high signal 또는 suppress 대상
- `provenance`
  - 누가 curated했고 reviewer 상태가 어떤지

## Subject

- `chain`
- `address` 또는 `entityKey`
- `role`
  - `primary_wallet`
  - `funding_wallet`
  - `exit_wallet`
  - `entity_anchor`

## Ground Truth

- `expectedOutcome`
  - 사람이 읽는 expected behavior
- `narrative`
  - 왜 이 케이스를 positive / negative / control로 보는지
- `expectedSignals`
  - 기대 score/finding 이름
- `expectedRoutes`
  - 기대 route semantic
- `sourceCitations`
  - 내부 analyst note, post-mortem, research memo, incident review 등
- `onchainEvidence`
  - tx hash 기준 근거

## Validation

로컬이나 CI에서 아래 worker mode로 manifest를 바로 검증할 수 있다.

```bash
QORVI_BACKTEST_MANIFEST_PATH=packages/intelligence/test/backtest-manifest.json \
QORVI_WORKER_MODE=analysis-backtest-manifest-validate \
go run ./apps/workers
```

기본 경로는 `packages/intelligence/test/backtest-manifest.json`이다.

## Curation Guidance

운영용 dataset은 아래 비율을 권장한다.

- `known_positive`
  - smart money early entry
  - treasury redistribution
  - market maker handoff
  - cross-chain rotation
- `known_negative`
  - bridge return
  - exchange-heavy retail
  - treasury rebalance
  - deployer / fee collector
- `control`
  - 활동량은 높지만 특별한 alpha narrative가 없는 일반 active wallets

최소 기준:

- cohort별 최소 20개
- caseType별 최소 20개
- entity family 중복 과다 샘플 금지

## Storage Convention

- 실제 운영 파일: `packages/intelligence/test/backtest-manifest.json`
- 예시 템플릿: `packages/intelligence/test/backtest-manifest.template.json`

템플릿은 validation 대상이 아니다. 실제 운영 파일만 release gate에 연결한다.
