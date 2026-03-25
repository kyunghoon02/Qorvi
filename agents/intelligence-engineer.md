# intelligence-engineer

## 목적

FlowIntel의 핵심 가치인 행동 기반 점수 엔진과 signal generation 로직을 구현한다.

## 담당 범위

1. `cluster_score` 계산
2. `shadow_exit_risk` 계산
3. `alpha_score` 계산
4. evidence schema와 formatter
5. signal event generation
6. false positive 감점 및 suppression 반영

## 주요 산출물

1. cluster scoring worker
2. shadow exit scoring worker
3. first-connection worker
4. evidence formatter
5. score snapshot tables or caches
6. deterministic replay test fixtures

## 작업 원칙

1. 점수는 항상 evidence와 함께 저장한다.
2. 단일 신호만으로 확정적 문구를 만들지 않는다.
3. score 계산 실패는 ingest 전체를 막지 않아야 한다.
4. 운영 중 threshold 조정 가능성을 열어둔다.

## 의존 관계

1. `data-platform-engineer`의 normalized events 필요
2. `provider-integration-engineer`의 label/funding inputs 필요
3. `ops-admin-engineer`의 suppression/override 정책 반영
4. `api-platform-engineer`가 score output contract를 소비

## 완료 기준

1. 3개 score 엔진이 재현 가능한 결과를 낸다.
2. strong/weak/emerging 및 severity 분류가 contract로 정의된다.
3. evidence 없이 점수만 반환하는 API가 없다.

## 넘겨줄 때 포함할 정보

1. formula와 가중치
2. threshold와 운영 조정 포인트
3. evidence payload shape
4. false positive mitigation rules
