# Qorvi Analysis Engine Hardening Plan

## 0. 목적

이 문서는 Qorvi의 자금 추적, 지갑 해석, AI 분석 계층을 `데모 가능한 MVP` 수준에서 `온체인 데이터 분석가와 Web3 업계 종사자가 봐도 신뢰할 수 있는 analyst-grade product` 수준으로 끌어올리기 위한 실행 계획서다.

목표는 단순히 점수를 더 많이 만드는 것이 아니다. 다음 세 가지를 동시에 달성해야 한다.

1. `정확성`
   - false positive를 줄이고, 반증이 있는 케이스를 과도하게 높게 평가하지 않는다.
2. `설명 가능성`
   - 모든 주요 판단에 대해 어떤 evidence가 어떻게 점수에 반영됐는지 재현 가능해야 한다.
3. `운영 가능성`
   - rule, score, labeling, AI explanation이 backtest와 regression test로 유지 가능해야 한다.

## 1. 현재 상태 진단

현재 Qorvi는 좋은 뼈대가 있다.

- wallet summary / graph / findings / watchlist / alerts / admin이 분리돼 있다.
- cluster / shadow-exit / first-connection 엔진이 독립 worker로 존재한다.
- wallet labeling, tracking promotion, AI explanation layer가 이미 연결돼 있다.

하지만 현재 상태는 `analyst-grade`라기보다 `heuristic-heavy MVP`에 가깝다.

### 1.1 주요 약점

1. 점수 엔진이 너무 선형적이다.
   - cluster는 고정 가중치 합산에 가깝다.
   - shadow exit도 linear formula 비중이 크다.
   - first connection도 raw count 기반 영향력이 너무 직접적이다.

2. evidence sufficiency 개념이 약하다.
   - 증거 수가 적거나 coverage가 얕아도 점수가 과하게 오를 수 있다.

3. contradiction / suppression 계층이 불완전하다.
   - treasury ops, internal rebalance, market maker inventory rotation, bridge return flow 같은 false positive suppressor가 더 체계화돼야 한다.

4. label/entity 해석이 아직 baseline rule-engine 비중이 높다.
   - known address + pattern match만으로는 업계 체감 신뢰도를 만들기 어렵다.

5. AI explanation이 evidence-grounded memo라기보다 요약 계층에 가깝다.
   - 주장과 근거, 반증, 불확실성, 다음 확인 포인트가 분리돼야 한다.

6. backtest와 calibration 체계가 없다.
   - precision / recall / false positive rate를 모르고는 “정확하다”고 말할 수 없다.

## 2. Hardening 원칙

### 2.1 판단 구조 원칙

모든 고도화는 아래 구조를 따른다.

`raw activity -> normalized route/event -> evidence -> score -> finding -> AI memo`

중간 계층을 생략하지 않는다. 특히 AI는 raw activity를 직접 해석하지 않고, 정규화된 evidence 위에서만 설명한다.

### 2.2 analyst-grade 기준

다음 질문에 답할 수 있어야 analyst-grade에 가깝다.

1. 왜 이 지갑을 smart money / treasury-adjacent / exit-risk로 봤는가?
2. 그 결론을 뒷받침하는 transaction path는 무엇인가?
3. 어떤 counter-evidence가 있었고, 왜 최종 결론을 뒤집지 못했는가?
4. 이 결론은 얼마나 불확실한가?
5. 같은 규칙을 과거 케이스에 돌렸을 때 precision이 얼마나 나오는가?

### 2.3 구현 우선순위 원칙

우선순위는 다음 순서로 고정한다.

1. score calibration
2. route classification
3. false-positive suppression
4. entity corroboration
5. AI memo hardening
6. backtest/benchmark

AI는 마지막에 강하게 붙인다. 분석 엔진이 부실한 상태에서 AI 설명만 강화하면 “말은 그럴듯하지만 믿기 어려운 제품”이 된다.

## 3. 목표 상태

고도화 이후 Qorvi는 아래 상태를 목표로 한다.

### 3.1 Wallet analysis

- wallet별로 `facts`, `inferences`, `uncertainties`가 분리돼 보인다.
- 주요 지표는 raw count가 아니라 quality-adjusted score로 계산된다.
- 지갑이 왜 `candidate`, `tracked`, `labeled`, `scored`가 되었는지 상태 전이 이유가 보인다.

### 3.2 Flow analysis

- transfer를 단순 송수신이 아니라 `funding`, `distribution`, `bridge escape`, `exchange deposit`, `market maker rotation`, `treasury operation`, `internal rebalance` 등으로 분류한다.
- route strength, temporal persistence, counterparty quality가 반영된다.

### 3.3 AI analyst output

- AI brief는 narrative가 아니라 `evidence memo`처럼 읽혀야 한다.
- 모든 주장에 route ref, tx ref, entity ref, confidence note가 붙는다.
- “왜 높다”와 함께 “왜 더 높지 않은가”도 말한다.

### 3.4 Measurement

- 평가 데이터셋과 regression suite가 있다.
- release 전 score drift, precision 변화, false positive 패턴을 확인할 수 있다.

## 4. Workstreams

## 4.1 Workstream A: Score Calibration Layer

### 목표

현재 엔진의 고정 가중치 합산을 `증거 품질 보정형 score layer`로 치환한다.

### 해야 할 일

1. `shared scoring framework` 도입
   - 모든 score가 아래 항목을 공통으로 갖도록 정리
   - `signal_strength`
   - `evidence_sufficiency`
   - `source_quality`
   - `freshness`
   - `contradiction_penalty`
   - `suppression_discount`

2. `minimum evidence threshold` 추가
   - 핵심 evidence 종류가 특정 수 이하이면 high rating으로 못 올라가게 막는다.

3. `confidence calibration` 도입
   - score value와 confidence note를 분리
   - 같은 70점이어도 evidence quality가 낮으면 confidence를 낮춘다.

4. `rating gate` 강화
   - high rating은 raw score만으로 되지 않고 evidence sufficiency 조건도 충족해야 한다.

### 산출물

- shared scoring helper
- cluster/shadow/first engine 공통 scoring contract
- calibrated metadata schema
- regression tests for threshold / contradiction / suppression behavior

### 완료 기준

- 동일 raw score라도 evidence 부족 케이스는 medium/low로 떨어진다.
- contradiction가 있는 케이스는 rating이 눈에 띄게 낮아진다.
- 점수 계산식이 evidence metadata로 역추적 가능하다.

## 4.2 Workstream B: Route Classification Engine

### 목표

거래 흐름을 analyst가 이해하는 route semantic으로 정규화한다.

### 해야 할 일

1. route taxonomy 확장
   - `cex_deposit`
   - `cex_withdrawal`
   - `bridge_escape`
   - `bridge_return`
   - `treasury_distribution`
   - `treasury_rebalance`
   - `market_maker_inventory_rotation`
   - `otc_handoff`
   - `aggregator_routing`
   - `funding_inflow`
   - `wash_like_internal_loop`

2. route confidence 계산
   - 단일 counterparty label로 확정하지 않고 multi-evidence corroboration 반영

3. route-level evidence reference 강화
   - tx hash
   - hop path
   - counterparty labels
   - temporal lag
   - chain context

### 산출물

- normalized route classifier
- route-level evidence bundle schema
- graph edge metadata enrichment

### 완료 기준

- wallet detail과 finding timeline에서 route type이 analyst vocabulary로 보인다.
- shadow exit와 first connection 점수가 raw transfer count 대신 normalized route count를 읽는다.

## 4.3 Workstream C: Cluster Extraction Precision

### 목표

cluster를 단순 `shared counterparty heuristic`에서 analyst가 납득할 수 있는 `peer / overlap / flow reciprocity / hub suppression` 기반 추출기로 바꾼다.

### 현재 문제

1. `overlapping_wallets`와 `shared_counterparties`가 사실상 동일 신호를 이중 반영하고 있다.
2. `mutual_transfer_count`가 실제 양방향 자금 흐름이 아니라 반복 상호작용 count에 가깝다.
3. dense wallet / hub wallet은 counterparty cap 때문에 graph shape가 왜곡될 수 있다.
4. aggregator / router / exchange hub 성격이 cluster extraction 단계에서 직접 penalty 되지 않는다.

### 해야 할 일

1. cluster signal 정의 재분리
   - `peer_wallets`
   - `shared_counterparties`
   - `shared_entities`
   - `bidirectional_flow_peers`
   - `temporal_cohort_overlap`
2. `mutual transfer` 재정의
   - 실제 inbound / outbound 흐름이 둘 다 확인되는 peer만 집계
   - 가능하면 asset / hop / time-window consistency도 반영
3. hub suppression 추가
   - aggregator / router / exchange / bridge infra neighbor가 많은 경우 contradiction 또는 suppression으로 직접 반영
4. graph sampling 개선
   - 고정 max counterparties만 자르는 방식에서 벗어나
   - recent / weighted / entity-diverse sampling 또는 adaptive cap 검토
5. cluster evidence schema 강화
   - 어떤 peer / 어떤 shared entity / 어떤 time overlap 때문에 cluster로 봤는지 metadata 남기기

### 산출물

- cluster signal v2 schema
- hub-aware cluster extractor
- adaptive graph sampling rules
- cluster-specific regression fixtures

### 완료 기준

- `shared counterparty 많은 hub wallet`이 cluster high score로 과잉 승격되는 비율이 줄어든다.
- cluster score가 실제 `peer overlap`과 `flow reciprocity`를 구분해서 설명한다.
- cluster finding에서 어떤 peer / entity evidence가 핵심이었는지 drill-down 가능하다.

## 4.4 Workstream D: False Positive Suppression

### 목표

전문가가 가장 먼저 불신하는 케이스를 체계적으로 suppress한다.

### 우선 suppress 대상

1. treasury operational distribution
2. internal rebalance between owned wallets
3. market maker inventory rotation
4. bridge round-trip / bridge return
5. exchange-heavy but non-informative retail flow
6. aggregator / router induced hub nodes
7. deployer or fee collector wallets mistaken for alpha wallets

### 해야 할 일

1. contradiction evidence schema 추가
2. suppression reason을 finding과 score metadata에 기록
3. suppression이 적용된 케이스는 AI가 이를 explicitly mention 하도록 계약 추가

### 산출물

- contradiction evidence type
- suppression rule set
- false-positive regression fixtures

### 완료 기준

- known treasury / MM / internal ops 샘플에서 과잉 알람이 유의미하게 감소한다.
- suppression reason이 UI drill-down에 노출된다.

## 4.5 Workstream E: Entity Corroboration and Label Quality

### 목표

현재 baseline label engine을 `corroborated entity engine`으로 확장한다.

### 해야 할 일

1. source hierarchy 정의
   - curated
   - provider verified
   - cross-source corroborated
   - heuristic inferred

2. entity confidence 계산
   - source quality
   - recurrence
   - cross-chain persistence
   - counterparty consistency
   - path consistency

3. label conflict resolution
   - exchange vs fund vs bridge vs treasury 충돌 시 우선순위와 confidence 조정 규칙 정의

4. label drift detection
   - 동일 주소가 과거와 다른 행동을 보일 때 stale or conflicting state를 표기

### 산출물

- label confidence framework
- corroboration metadata
- conflicting label resolution rules

### 완료 기준

- label이 단순 이름표가 아니라 “왜 그렇게 판단했는지”가 메타데이터로 남는다.
- verified / inferred / behavioral 구분이 제품 전반에서 일관되게 보인다.

## 4.6 Workstream F: AI Evidence Memo Layer

### 목표

AI를 “예쁜 요약기”가 아니라 evidence-grounded analyst copilot로 제한한다.

### 출력 계약

모든 AI output은 최소한 아래 구조를 가진다.

1. `summary`
2. `key evidence`
3. `counter-evidence`
4. `confidence note`
5. `what to verify next`
6. `why this is not just noise`

### 해야 할 일

1. prompt contract 개편
   - claim without evidence 금지
   - unsupported certainty 금지
   - tx/path/entity reference 없는 결론 금지

2. explanation input schema 강화
   - score summary
   - top evidence refs
   - top contradiction refs
   - suppression reasons
   - data coverage and freshness

3. memo style output으로 변경
   - marketing 문구 금지
   - analyst note 스타일 강제

### 산출물

- new prompt versions
- explanation input contract v2
- finding / wallet brief AI memo templates

### 완료 기준

- AI output에서 과도한 단정 표현이 줄어든다.
- analyst가 바로 follow-up investigation에 쓸 수 있는 memo 형태가 된다.

## 4.7 Workstream G: Benchmarking, Backtests, and Evaluation

### 목표

“좋아 보인다”가 아니라 실제로 더 정확해졌는지 측정한다.

### 평가 세트 구성

1. known smart money wallets
2. known treasury wallets
3. known market maker wallets
4. known exchange / bridge infra wallets
5. normal active trader negative set
6. noisy hub wallet negative set

### 측정 항목

1. precision@high
2. false positive rate
3. entity misclassification rate
4. route classification accuracy
5. score drift over time
6. AI unsupported-claim rate

### 해야 할 일

1. benchmark fixture directory 추가
2. replayable evaluation runner 작성
3. release gate에 evaluation summary 포함

### 산출물

- benchmark dataset schema
- evaluation CLI / worker
- weekly score drift report

### 완료 기준

- core scenarios에 대해 baseline 대비 precision 개선 수치가 나온다.
- release 전 regression summary를 자동 확인할 수 있다.

## 4.8 Workstream H: Backtest Program

### 목표

fixture benchmark를 넘어서, 실제 과거 온체인 케이스를 재생해 Qorvi가 analyst workflow에서 얼마나 유효한지 검증한다.

### benchmark와 backtest의 차이

- benchmark
  - 작은 고정 fixture 세트
  - release regression gate 용도
  - 점수식이 깨졌는지 빠르게 확인
- backtest
  - 실제 과거 기간의 wallet / route / label 이벤트 재생
  - 탐지 성능, latency, false positive pattern 검증
  - analyst review와 calibration 회의용

### backtest 평가 질문

1. 실제 smart money 진입을 Qorvi가 얼마나 빨리 감지했는가?
2. treasury redistribution과 market maker handoff를 실제로 구분했는가?
3. bridge return, exchange-heavy retail, aggregator hub 같은 false positive를 얼마나 잘 억제했는가?
4. AI memo가 analyst judgement를 보조하는 데 충분했는가, 아니면 noise를 늘렸는가?

### 데이터셋 구성

1. `known positive event set`
   - smart money early entry
   - treasury redistribution
   - market maker handoff
   - cross-chain rotation
2. `known negative set`
   - exchange-heavy retail wallets
   - bridge return wallets
   - internal rebalance / treasury ops
   - aggregator / router hub wallets
   - deployer / fee collector wallets
3. `control set`
   - 활동량은 높지만 특별한 alpha나 coordinated behavior가 없는 일반 active wallets

### 샘플링 원칙

1. 체인별로 분리한다.
   - EVM
   - Solana
2. 기간별로 분리한다.
   - 최근 7일
   - 최근 30일
   - 최근 90일
3. 이벤트 타입별 최소 샘플 수를 강제한다.
   - route type별 최소 20개 이상
   - false positive class별 최소 20개 이상
4. 같은 entity family가 과도하게 중복되지 않도록 dedup한다.

### 실행 방식

1. `snapshot backtest`
   - 특정 날짜 기준 state를 재현
   - score, finding, memo 결과를 저장
2. `rolling backtest`
   - N일 window를 하루 단위로 이동시키며 replay
   - detection lead time과 score drift 확인
3. `ablation backtest`
   - suppression off / route classifier off / corroboration off 상태와 비교
   - 어떤 레이어가 precision 개선에 실제로 기여하는지 측정

### 측정 항목

1. detection precision
2. detection recall
3. median time-to-detection
4. false positive rate by route class
5. false positive rate by wallet class
6. score stability across adjacent windows
7. memo usefulness score
   - analyst manual review 기반
8. unsupported-claim count
   - AI memo에서 evidence로 뒷받침되지 않는 주장 수

### 리뷰 프로세스

1. backtest run 결과를 route class별로 집계한다.
2. top false positives를 analyst review queue에 올린다.
3. 각 false positive마다 원인을 분류한다.
   - route misclassification
   - label conflict
   - insufficient suppression
   - stale data coverage
   - AI overstatement
4. calibration action item을 다음 주 release plan에 반영한다.

### 산출물

1. backtest dataset manifest
2. replay runner
3. analyst review worksheet
4. weekly backtest report
5. release delta summary
   - 이전 release 대비 precision / recall / latency 변화

### 초기 구현 메모

- manifest schema 문서:
  - `/Users/kh/Github/Qorvi/packages/intelligence/test/backtest-manifest-schema.md`
- template:
  - `/Users/kh/Github/Qorvi/packages/intelligence/test/backtest-manifest.template.json`
- validation worker mode:
  - `QORVI_WORKER_MODE=analysis-backtest-manifest-validate`
- 실제 release gate는 template가 아니라
  - `/Users/kh/Github/Qorvi/packages/intelligence/test/backtest-manifest.json`
  - 이 파일을 채운 뒤 validation + replay runner를 같이 통과해야 한다.

### 완료 기준

- 최근 30일 backtest에서 주요 positive class의 detection precision이 baseline 대비 개선된다.
- false positive top bucket이 route / entity / suppression 기준으로 명확히 설명 가능하다.
- release 전 benchmark와 별도로 backtest summary가 리뷰된다.

## 5. 실행 순서

## Phase 1: Foundation Hardening

기간 목표: 점수 체계와 suppression 기반을 먼저 바로잡는다.

1. shared scoring layer
2. evidence sufficiency gates
3. contradiction / suppression schema
4. cluster / shadow / first 엔진 공통 metadata 정리
5. cluster extraction v2 signal 분리와 hub suppression 착수

## Phase 2: Flow and Entity Intelligence

기간 목표: route semantic과 entity corroboration을 analyst vocabulary 수준으로 올린다.

1. normalized route classification
2. entity source hierarchy
3. conflicting label resolution
4. graph / finding timeline evidence enrichment
5. cluster adaptive sampling과 peer/entity evidence enrichment

## Phase 3: AI and Evaluation

기간 목표: AI를 evidence memo 계층으로 고정하고, 성능을 측정 가능하게 만든다.

1. explanation contract v2
2. analyst memo prompts
3. benchmark datasets
4. evaluation pipeline and release gates
5. rolling backtest and weekly analyst review loop
6. cluster precision backtest slices and hub-wallet regression review

## 6. Immediate Next Tasks

지금 바로 착수할 작업은 아래 순서로 고정한다.

1. `cluster / shadow / first` 공통 scoring metadata 설계
2. evidence sufficiency gate 구현
3. contradiction penalty와 suppression reason schema 추가
4. cluster signal v2 설계와 duplicated signal 제거
5. aggregator / router / exchange hub cluster suppressor 추가
6. known false-positive fixture 작성
7. wallet brief / finding explanation input에 contradiction, suppression, coverage 정보 추가
8. benchmark 이후 운영용 backtest dataset manifest 초안 작성

## 7. Definition of Done

아래 조건이 충족되면 Qorvi 분석 엔진이 analyst-grade에 가까워졌다고 판단한다.

1. high score는 항상 충분한 evidence와 함께 나온다.
2. treasury / MM / bridge / CEX false positive가 현재 대비 유의미하게 줄었다.
3. cluster score가 duplicated heuristic이 아니라 peer overlap, shared entity, bidirectional flow를 구분해 계산된다.
4. hub wallet / aggregator / router 성격 주소가 cluster high score로 과잉 승격되지 않는다.
5. finding timeline이 route, tx, entity reference를 통해 재현 가능하다.
6. AI memo가 unsupported claim 없이 evidence-grounded 형식으로 나온다.
7. benchmark 결과가 release gate에 포함된다.
8. 주간 backtest 결과에서 precision 변화와 주요 false positive 원인이 추적된다.

## 8. 비고

이 계획은 “더 많은 기능 추가” 계획이 아니다. 핵심은 `정확성과 신뢰도를 높이는 재구성`이다.

새 체인 추가, 더 많은 피드 추가, 더 화려한 AI UX보다 먼저 해야 할 일은 다음 두 가지다.

- 판단 로직을 반증 가능하게 만들기
- 품질을 측정 가능하게 만들기

이 두 가지가 먼저 고정돼야, 이후 확장도 제품 신뢰도를 해치지 않는다.
