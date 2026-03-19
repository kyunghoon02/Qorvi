# WhaleGraph Task Backlog

이 문서는 [plan.md](/Users/kh/Github/WhaleGraph/plan.md)를 실제 실행 단위로 쪼갠 작업 백로그다.  
각 task는 바로 이슈나 스프린트 티켓으로 옮길 수 있도록 우선순위, 담당 subagent, 선행 조건, 산출물, 완료 기준을 포함한다.

## 1. 사용 규칙

1. task는 기본적으로 위에서 아래 순서로 수행한다.
2. 병렬 작업은 `Depends On`이 겹치지 않는 범위에서만 진행한다.
3. 모든 기능 task는 구현 전 API/schema contract를 먼저 확인한다.
4. 점수, 라벨링, 알림 관련 변경은 `ops-admin-engineer` 검토를 포함한다.
5. 배포 직전 task는 `billing-launch-engineer`의 release gate 체크 없이는 완료 처리하지 않는다.

## 2. 상태 규칙

- `Todo`: 아직 시작하지 않음
- `In Progress`: 작업 중
- `Blocked`: 선행 조건 미충족 또는 외부 이슈 존재
- `Review`: 구현 완료, 검토/테스트 대기
- `Done`: 완료 기준 충족

## 3. Sprint Order

| Sprint | 목표 |
| --- | --- |
| Sprint 0 | scope freeze, repo bootstrap, schema baseline |
| Sprint 1 | data platform, provider adapters, ingest skeleton |
| Sprint 2 | wallet intelligence MVP |
| Sprint 3 | cluster engine, shadow exit engine |
| Sprint 4 | first-connection, alerts, watchlists, admin |
| Sprint 5 | billing, launch hardening, beta release |

## 4. Task List

## Sprint 0

### WG-001 Scope Freeze

- Status: `Done`
- Owner: `foundation-architect`
- Support: `billing-launch-engineer`
- Depends On: 없음
- Deliverables:
  - beta scope matrix
  - Must/Should/Later 분류표
  - provider budget sheet 초안
- Definition of Done:
  - beta 필수 기능 범위가 문서로 고정됨
  - 비목표 기능이 별도 목록으로 분리됨

### WG-002 Domain Contract Baseline

- Status: `In Progress`
- Owner: `api-platform-engineer`
- Support: `foundation-architect`, `intelligence-engineer`
- Depends On: `WG-001`
- Deliverables:
  - response envelope 초안
  - evidence schema v0
  - score output contract 초안
- Definition of Done:
  - summary/detail/signals 응답 형식이 문서화됨
  - score가 evidence 없이 반환되지 않도록 contract가 정리됨

### WG-003 Monorepo Bootstrap

- Status: `Review`
- Owner: `foundation-architect`
- Depends On: `WG-001`
- Deliverables:
  - `apps/`, `services/`, `libs/`, `infra/`, `docs/` 구조
  - `pnpm` workspace와 Go workspace/module support
  - lint/format/typecheck/test 기본 구성
- Definition of Done:
  - web workspace install과 backend Go module check가 동작함

### WG-004 Environment and Secret Validation

- Status: `Review`
- Owner: `foundation-architect`
- Depends On: `WG-003`
- Deliverables:
  - env loader
  - startup validation
  - `.env.example` 정책
- Definition of Done:
  - 필수 secret 누락 시 앱이 즉시 실패함

### WG-005 Auth and RBAC Skeleton

- Status: `In Progress`
- Owner: `foundation-architect`
- Support: `api-platform-engineer`
- Depends On: `WG-003`
- Deliverables:
  - auth provider skeleton
  - role contract: `user`, `pro`, `admin`, `operator`
  - protected route/middleware baseline
- Definition of Done:
  - role 기반 접근 제어가 API와 웹 모두에서 가능함

## Sprint 1

### WG-006 Local Infra Stack

- Status: `Todo`
- Owner: `data-platform-engineer`
- Depends On: `WG-003`
- Deliverables:
  - local Postgres
  - local Neo4j
  - local Redis
  - docker compose or equivalent
- Definition of Done:
  - 개발자가 로컬에서 핵심 인프라를 재현 가능

### WG-007 Postgres Schema v1

- Status: `Todo`
- Owner: `data-platform-engineer`
- Support: `ops-admin-engineer`, `billing-launch-engineer`
- Depends On: `WG-002`, `WG-006`
- Deliverables:
  - users, subscriptions, wallets, tokens, entities, transactions, clusters, signals, alerts, watchlists, suppressions, audit 관련 migration
- Definition of Done:
  - `plan.md`의 핵심 테이블이 생성 가능
  - 주요 인덱스와 unique key가 포함됨

### WG-008 Neo4j Schema Bootstrap

- Status: `Todo`
- Owner: `data-platform-engineer`
- Depends On: `WG-006`
- Deliverables:
  - node labels
  - relationship constraints
  - graph index bootstrap script
- Definition of Done:
  - Wallet, Token, Entity, Cluster 및 핵심 관계 upsert가 가능함

### WG-009 Raw Payload Storage Pipeline

- Status: `Todo`
- Owner: `data-platform-engineer`
- Depends On: `WG-006`, `WG-007`
- Deliverables:
  - raw webhook/object storage writer
  - payload metadata schema
  - replay reference key
- Definition of Done:
  - raw event가 정규화 전에 저장됨
  - 재처리 가능한 참조 키가 남음

### WG-010 Idempotency and Dedup Framework

- Status: `Todo`
- Owner: `data-platform-engineer`
- Support: `api-platform-engineer`
- Depends On: `WG-007`, `WG-009`
- Deliverables:
  - dedup key strategy
  - Redis/Postgres dedup store
  - duplicate-safe ingest helper
- Definition of Done:
  - 동일 이벤트 재수신 시 중복 저장/중복 알림이 발생하지 않음

### WG-011 Normalized Transaction Schema

- Status: `Todo`
- Owner: `data-platform-engineer`
- Support: `provider-integration-engineer`
- Depends On: `WG-002`, `WG-007`
- Deliverables:
  - EVM/Solana 공통 transaction schema
  - wallet/token/entity canonical key 규칙
  - schema validation tests
- Definition of Done:
  - 두 체인 이벤트가 동일 내부 모델로 변환 가능

### WG-012 Provider Usage Logging

- Status: `Todo`
- Owner: `provider-integration-engineer`
- Support: `ops-admin-engineer`
- Depends On: `WG-007`, `WG-011`
- Deliverables:
  - provider usage log writer
  - quota meter baseline
  - error rate logging
- Definition of Done:
  - provider별 호출량과 실패율이 저장됨

### WG-013 Dune Seed Discovery Adapter

- Status: `Todo`
- Owner: `provider-integration-engineer`
- Depends On: `WG-011`, `WG-012`
- Deliverables:
  - Dune export adapter
  - seed candidate input parser
  - fixture-based contract tests
- Definition of Done:
  - Dune 결과를 seed candidate 입력으로 변환 가능

### WG-014 Alchemy Adapter

- Status: `Todo`
- Owner: `provider-integration-engineer`
- Depends On: `WG-011`, `WG-012`
- Deliverables:
  - historical transfer adapter
  - realtime webhook adapter
  - retry/backoff handling
- Definition of Done:
  - EVM backfill과 watchlist realtime ingest가 가능

### WG-015 Helius Adapter

- Status: `Todo`
- Owner: `provider-integration-engineer`
- Depends On: `WG-011`, `WG-012`
- Deliverables:
  - wallet history adapter
  - transfers/funded-by/identity adapter
  - schema-versioned parser
- Definition of Done:
  - Solana profile, funder, counterparty 추출이 가능

### WG-016 Moralis Enrichment Adapter

- Status: `Todo`
- Owner: `provider-integration-engineer`
- Depends On: `WG-011`, `WG-012`
- Deliverables:
  - optional enrichment client
  - on-demand cache policy
- Definition of Done:
  - 상세 화면용 enrichment를 보조 계층으로 호출 가능

## Sprint 2

### WG-017 Seed Discovery Batch

- Status: `Todo`
- Owner: `provider-integration-engineer`
- Support: `intelligence-engineer`
- Depends On: `WG-013`, `WG-007`
- Deliverables:
  - candidate scoring batch
  - seed watchlist seeding job
- Definition of Done:
  - seed whale 후보가 주기적으로 생성되고 편입 가능

### WG-018 Historical Backfill Worker

- Status: `Todo`
- Owner: `provider-integration-engineer`
- Support: `data-platform-engineer`
- Depends On: `WG-014`, `WG-015`, `WG-010`, `WG-011`
- Deliverables:
  - 90일 기본 backfill worker
  - 1-hop 및 제한된 2-hop 확장 로직
  - service address stop rule
- Definition of Done:
  - 기본 기간 내 wallet graph용 데이터가 적재됨

### WG-019 Graph Materialization Pipeline

- Status: `Todo`
- Owner: `data-platform-engineer`
- Support: `intelligence-engineer`
- Depends On: `WG-008`, `WG-011`, `WG-018`
- Deliverables:
  - normalized event -> Neo4j edge upsert pipeline
  - neighborhood snapshot strategy
- Definition of Done:
  - wallet 중심 1-hop neighborhood 조회가 가능

### WG-020 Wallet Summary Materializer

- Status: `Todo`
- Owner: `api-platform-engineer`
- Support: `data-platform-engineer`
- Depends On: `WG-007`, `WG-018`
- Deliverables:
  - wallet daily stats 집계
  - top counterparties 집계
  - netflow/holdings/latest signals aggregate
- Definition of Done:
  - wallet summary 응답에 필요한 캐시 데이터가 생성됨

### WG-021 Wallet Summary API

- Status: `Todo`
- Owner: `api-platform-engineer`
- Depends On: `WG-020`, `WG-005`
- Deliverables:
  - `GET /wallet/:chain/:address/summary`
  - unknown/new wallet fallback 처리
- Definition of Done:
  - summary API가 핵심 카드 정보를 반환함
  - 신규 주소도 빈 응답 대신 설명 가능한 fallback을 가짐

### WG-022 Wallet Graph API

- Status: `Todo`
- Owner: `api-platform-engineer`
- Depends On: `WG-019`, `WG-005`
- Deliverables:
  - `GET /wallet/:chain/:address/graph`
  - depth gating 정책
  - density guardrails
- Definition of Done:
  - 기본 1-hop 조회가 가능
  - Pro 전용 2-hop gating 포인트가 정의됨

### WG-023 Search API

- Status: `Todo`
- Owner: `api-platform-engineer`
- Support: `product-ui-engineer`
- Depends On: `WG-021`, `WG-007`
- Deliverables:
  - address/token/entity search endpoint
  - parser for EVM, Solana, ENS
- Definition of Done:
  - 주요 검색 타입을 하나의 entry point로 처리 가능

### WG-024 Product Search and Wallet UI

- Status: `Todo`
- Owner: `product-ui-engineer`
- Depends On: `WG-021`, `WG-022`, `WG-023`
- Deliverables:
  - global search UI
  - wallet summary page
  - counterparties/timeline base UI
- Definition of Done:
  - search -> wallet detail 흐름이 웹에서 동작함

## Sprint 3

### WG-025 Cluster Signal Calculators

- Status: `Todo`
- Owner: `intelligence-engineer`
- Depends On: `WG-018`, `WG-019`
- Deliverables:
  - same_funder
  - co_trading
  - shared_counterparties
  - cex_pattern
  - temporal_sync
  - bridge_similarity calculators
- Definition of Done:
  - 각 하위 신호가 독립 테스트 가능 상태로 구현됨

### WG-026 Cluster Scoring Worker

- Status: `Todo`
- Owner: `intelligence-engineer`
- Support: `ops-admin-engineer`
- Depends On: `WG-025`
- Deliverables:
  - `cluster_score` worker
  - strong/weak/emerging classification
  - evidence formatter
- Definition of Done:
  - cluster score와 evidence를 snapshot으로 저장 가능

### WG-027 Cluster API and UI

- Status: `Todo`
- Owner: `api-platform-engineer`
- Support: `product-ui-engineer`
- Depends On: `WG-026`
- Deliverables:
  - `GET /cluster/:id`
  - cluster detail page
  - explanation card
- Definition of Done:
  - cluster detail이 members, common actions, evidence를 표시함

### WG-028 Shadow Exit Detector

- Status: `Todo`
- Owner: `intelligence-engineer`
- Support: `ops-admin-engineer`
- Depends On: `WG-018`, `WG-019`
- Deliverables:
  - fan-out candidate detection
  - cex proximity, bridge escape, outflow ratio 계산
  - treasury whitelist discount
- Definition of Done:
  - shadow exit risk candidate가 생성되고 evidence timeline이 남음

### WG-029 Shadow Exit API and Feed

- Status: `Todo`
- Owner: `api-platform-engineer`
- Support: `product-ui-engineer`
- Depends On: `WG-028`
- Deliverables:
  - `GET /signals/shadow-exits`
  - shadow exit feed 카드
- Definition of Done:
  - risk score와 non-absolute wording이 함께 노출됨

## Sprint 4

### WG-030 First-Connection Detector

- Status: `Todo`
- Owner: `intelligence-engineer`
- Depends On: `WG-017`, `WG-018`, `WG-019`
- Deliverables:
  - 90일 무이력 판정기
  - 24시간 내 신규 공통 진입 감지
  - `alpha_score` 계산
- Definition of Done:
  - token/protocol 단위 신규 공통 진입 이벤트를 생성 가능

### WG-031 First-Connection API and Hot Feed

- Status: `Todo`
- Owner: `api-platform-engineer`
- Support: `product-ui-engineer`
- Depends On: `WG-030`
- Deliverables:
  - `GET /signals/first-connections`
  - hot feed 정렬: 최신순/점수순
- Definition of Done:
  - feed에서 첫 진입 고래, 시각, 규모, score가 노출됨

### WG-032 Watchlist Service

- Status: `Todo`
- Owner: `api-platform-engineer`
- Support: `product-ui-engineer`
- Depends On: `WG-007`, `WG-005`
- Deliverables:
  - watchlist CRUD
  - watchlist item tags/notes
  - tier-based watchlist limits
- Definition of Done:
  - wallet, cluster, token, entity를 watchlist에 저장 가능

### WG-033 Alert Rules and Dedup Engine

- Status: `Todo`
- Owner: `api-platform-engineer`
- Support: `intelligence-engineer`, `ops-admin-engineer`
- Depends On: `WG-010`, `WG-026`, `WG-028`, `WG-030`, `WG-032`
- Deliverables:
  - alert rule CRUD
  - dedup/cooldown/re-notify engine
  - alert event audit trail
- Definition of Done:
  - 동일 이벤트 중복 발송 방지
  - severity 상승 시 재알림 가능

### WG-034 Alert Delivery Channels

- Status: `Todo`
- Owner: `api-platform-engineer`
- Support: `billing-launch-engineer`
- Depends On: `WG-033`
- Deliverables:
  - in-app inbox
  - email delivery
  - Telegram delivery
  - Discord webhook delivery
- Definition of Done:
  - 최소 2개 외부 채널과 in-app이 동작함

### WG-035 Alert Center UI

- Status: `Todo`
- Owner: `product-ui-engineer`
- Depends On: `WG-033`, `WG-034`
- Deliverables:
  - active rules 화면
  - triggered events 리스트
  - mute/snooze/severity filter UI
- Definition of Done:
  - 사용자가 제품 내에서 알림 상태와 규칙을 관리 가능

### WG-036 Admin Console MVP

- Status: `Todo`
- Owner: `ops-admin-engineer`
- Support: `api-platform-engineer`
- Depends On: `WG-007`, `WG-012`, `WG-033`
- Deliverables:
  - label editor
  - suppression rule management
  - curated list editor
  - provider usage dashboard
- Definition of Done:
  - 운영자가 DB 직접 접근 없이 핵심 운영 작업을 수행 가능

## Sprint 5

### WG-037 Entitlement Matrix

- Status: `Todo`
- Owner: `billing-launch-engineer`
- Support: `api-platform-engineer`, `product-ui-engineer`
- Depends On: `WG-005`, `WG-021`, `WG-022`, `WG-032`
- Deliverables:
  - Free/Pro/Team 기능 매트릭스
  - API/UI gating 규칙
- Definition of Done:
  - 플랜별 허용 범위가 코드와 문서에 반영됨

### WG-038 Stripe Billing Flow

- Status: `Todo`
- Owner: `billing-launch-engineer`
- Support: `api-platform-engineer`
- Depends On: `WG-037`
- Deliverables:
  - checkout flow
  - billing webhook
  - subscription reconciliation
- Definition of Done:
  - 최소 1개 유료 플랜 결제가 가능

### WG-039 Pricing and Account UI

- Status: `Todo`
- Owner: `product-ui-engineer`
- Support: `billing-launch-engineer`
- Depends On: `WG-038`
- Deliverables:
  - pricing page
  - account/billing page
  - current plan display
- Definition of Done:
  - 사용자가 플랜 확인과 업그레이드를 UI에서 수행 가능

### WG-040 Observability Dashboard

- Status: `Todo`
- Owner: `billing-launch-engineer`
- Support: `ops-admin-engineer`, `provider-integration-engineer`
- Depends On: `WG-012`, `WG-033`, `WG-034`
- Deliverables:
  - provider usage dashboard
  - ingest lag dashboard
  - alert delivery dashboard
  - error tracking baseline
- Definition of Done:
  - beta 운영 필수 메트릭이 한 곳에서 확인 가능

### WG-041 Replay and Contract Test Suite

- Status: `Todo`
- Owner: `data-platform-engineer`
- Support: `provider-integration-engineer`, `intelligence-engineer`
- Depends On: `WG-009`, `WG-011`, `WG-026`, `WG-028`, `WG-030`
- Deliverables:
  - replay test fixtures
  - provider contract tests
  - scoring consistency tests
- Definition of Done:
  - raw payload 재처리 시 핵심 score 결과가 재현 가능

### WG-042 E2E Beta Flow Test

- Status: `Todo`
- Owner: `product-ui-engineer`
- Support: `api-platform-engineer`, `billing-launch-engineer`
- Depends On: `WG-024`, `WG-027`, `WG-031`, `WG-035`, `WG-039`
- Deliverables:
  - search -> wallet -> watchlist -> alert -> upgrade 핵심 흐름 테스트
- Definition of Done:
  - beta 핵심 사용자 플로우가 자동화 또는 문서화된 시나리오로 검증됨

### WG-043 Beta Launch Checklist

- Status: `Todo`
- Owner: `billing-launch-engineer`
- Support: 전체
- Depends On: `WG-036`, `WG-038`, `WG-040`, `WG-041`, `WG-042`
- Deliverables:
  - functional gate checklist
  - reliability gate checklist
  - UX gate checklist
  - rollback/runbook package
- Definition of Done:
  - `plan.md`의 release criteria가 전부 체크됨

## 5. Parallel Execution Guide

아래 task들은 병렬 진행이 가능하다.

1. `WG-003`, `WG-004`, `WG-005`
2. `WG-007`, `WG-008`, `WG-009`
3. `WG-013`, `WG-014`, `WG-015`, `WG-016`
4. `WG-021`, `WG-022`, `WG-023`, `WG-024`
5. `WG-026`, `WG-028`
6. `WG-031`, `WG-032`, `WG-033`
7. `WG-038`, `WG-039`, `WG-040`

## 6. Critical Path

beta를 가장 늦추는 경로는 아래다.

1. `WG-003` -> `WG-006` -> `WG-007` -> `WG-011`
2. `WG-011` -> `WG-014` + `WG-015` -> `WG-018`
3. `WG-018` -> `WG-020` -> `WG-021` -> `WG-024`
4. `WG-018` + `WG-019` -> `WG-025` -> `WG-026` -> `WG-033`
5. `WG-030` -> `WG-031`
6. `WG-033` -> `WG-034` -> `WG-035`
7. `WG-037` -> `WG-038` -> `WG-039`
8. `WG-040` + `WG-041` + `WG-042` -> `WG-043`

## 7. Immediate Next Tasks

현재 저장소 상태에서 바로 시작해야 할 우선 작업은 다음 5개다.

1. `WG-001 Scope Freeze`
2. `WG-003 Monorepo Bootstrap`
3. `WG-006 Local Infra Stack`
4. `WG-007 Postgres Schema v1`
5. `WG-011 Normalized Transaction Schema`
