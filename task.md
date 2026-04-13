# Qorvi Task Backlog

이 문서는 [plan.md](/Users/kh/Github/Qorvi/plan.md)를 실제 실행 단위로 쪼갠 작업 백로그다.
각 task는 바로 이슈나 스프린트 티켓으로 옮길 수 있도록 우선순위, 담당 subagent, 선행 조건, 산출물, 완료 기준을 포함한다.

## 1. 사용 규칙

1. task는 기본적으로 위에서 아래 순서로 수행한다.
2. 병렬 작업은 `Depends On`이 겹치지 않는 범위에서만 진행한다.
3. 모든 기능 task는 구현 전 API/schema contract를 먼저 확인한다.
4. 점수, 라벨링, 알림 관련 변경은 `ops-admin-engineer` 검토를 포함한다.
5. 배포 직전 task는 `billing-launch-engineer`와 `ops-admin-engineer`의 production gate 체크 없이는 완료 처리하지 않는다.
6. 앞으로 product-facing task는 mock, fabricated preview data, local seed record를 새로 추가하지 않는다. 구현이 미완인 경우에도 실제 `loading`, `indexing`, `empty`, `unavailable`, `error` 상태를 정의하고 그 상태를 기준으로 작업한다.
7. production deployment 준비는 local `.env`와 target deployment env를 분리해 관리한다. production preflight는 target env file을 직접 검증할 수 있어야 한다.

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
| Sprint 2 | AI wallet brief MVP |
| Sprint 3 | behavior cohort, distribution & exit risk |
| Sprint 4 | early convergence, findings delivery, entity interpretation |
| Sprint 5 | alerts, admin, production hardening |

## 4. Task List

## Current Strategic Next

아래 task를 기존 launch/billing 마감보다 우선한다. Qorvi의 기본 제품 경험은 이제 `검색 -> 그래프`가 아니라 `findings -> brief -> evidence` 순서로 구현하며, 모든 active task는 production deployment 전제를 기준으로 마감한다.

우선 참고 문서:
- [/Users/kh/Github/Qorvi/qorvi-ai/engine-hardening-roadmap.md](/Users/kh/Github/Qorvi/qorvi-ai/engine-hardening-roadmap.md)

### WG-044 AI Findings Generation Baseline

- Status: `In Progress`
- Owner: `intelligence-engineer`
- Support: `api-platform-engineer`, `data-platform-engineer`
- Depends On: `WG-007`, `WG-020`, `WG-019`
- Deliverables:
  - `finding_candidates`
  - `finding_evidence_bundles`
  - finding classification + priority rules
  - AI summary contract wiring
- Current State:
  - migration `0016_findings_baseline.sql` 추가
  - `cluster_score`, `shadow_exit`, `first_connection` snapshot worker가 baseline finding을 materialize
  - `packages/domain/findings.go`와 `qorvi-ai/contracts/finding-object-schema.md` 기준 canonical finding object 고정
- Definition of Done:
  - 최소 5개 finding type이 evidence bundle과 함께 생성됨
  - finding duplication/merge 규칙이 정의됨

### WG-045 AI Findings Feed API/UI

- Status: `Done`
- Owner: `product-ui-engineer`
- Support: `api-platform-engineer`, `intelligence-engineer`
- Depends On: `WG-044`
- Deliverables:
  - `GET /findings`
  - findings feed ranking/filter
  - home/discover findings-first surface
- Current State:
  - `GET /v1/findings` baseline route/service/repository/store 추가
  - feed item contract에 summary, importance, confidence, evidence, next-watch 포함
  - home/discover가 `/v1/findings` live feed를 우선 읽고, unavailable일 때만 summary-derived findings로 fallback
- Definition of Done:
  - 홈에서 검색 전에도 주요 finding을 읽을 수 있음
  - feed item은 summary, importance, confidence, next-watch를 포함함

### WG-046 AI Wallet Brief

- Status: `Done`
- Owner: `api-platform-engineer`
- Support: `product-ui-engineer`, `intelligence-engineer`
- Depends On: `WG-044`, `WG-021`
- Deliverables:
  - `GET /wallet/:chain/:address/brief`
  - AI summary + key findings + evidence timeline contract
  - wallet page에서 brief-first layout
- Current State:
  - `GET /v1/wallets/:chain/:address/brief` baseline route/service 추가
  - deterministic AI summary + key findings + verified/probable/behavioral label split 포함
  - wallet detail route가 `/brief`를 실제로 load하고, 상단 `AI brief` 카드가 brief contract를 우선 사용
- Definition of Done:
  - 검색 결과에서 그래프보다 AI brief가 먼저 반환됨
  - verified / probable / behavioral labels가 함께 노출됨

### WG-047 Entity Interpretation Surface

- Status: `Done`
- Owner: `api-platform-engineer`
- Support: `product-ui-engineer`, `ops-admin-engineer`
- Depends On: `WG-044`, `WG-007`
- Deliverables:
  - `GET /entity/:id`
  - entity-linked wallets, interpreted findings, timeline
  - VC / MM / treasury / exchange / bridge 중심 page
- Current State:
  - `GET /v1/entity/:id` baseline route/service/repository/reader 추가
  - entities + member wallets + member labels + entity-scoped findings read path 연결
  - `/app/entity/[entityId]` route가 boundary `loadEntityInterpretationPreview` 기준으로 정리됨
- Definition of Done:
  - 엔터티 단위 해석과 evidence drill-down이 가능함

### WG-048 Interpretation Finding Types

- Status: `In Progress`
- Owner: `intelligence-engineer`
- Support: `provider-integration-engineer`, `ops-admin-engineer`
- Depends On: `WG-044`
- Deliverables:
  - suspected MM handoff
  - treasury redistribution
  - cross-chain rotation
  - exchange pressure
- Current State:
  - baseline finding generation은 `smart_money_convergence`, `exit_preparation`, `cross_chain_rotation`, `cex_deposit_pressure`까지 연결
  - snapshot worker가 wallet labels + score context를 읽어 `suspected_mm_handoff`, `treasury_redistribution`, `fund_adjacent_activity`, `high_conviction_entry`를 추가 materialize
  - snapshot worker가 service-specific `flow/path/evidence + next_watch` bundle을 함께 materialize 하도록 보강
  - 다음 focus는 label-aware baseline을 넘어 `flow/path/evidence bundle` 기반 해석형 finding rule 고도화
- Definition of Done:
  - 최소 4개 interpretation finding type이 규칙/evidence 기반으로 서빙됨

### WG-049 Interactive Analyst Tool Routes

- Status: `In Progress`
- Owner: `api-platform-engineer`
- Support: `product-ui-engineer`, `intelligence-engineer`
- Depends On: `WG-045`, `WG-046`, `WG-047`
- Deliverables:
  - `GET /v1/analyst/findings`
  - `GET /v1/analyst/wallets/:chain/:address/brief`
  - `GET /v1/analyst/entity/:id`
  - `GET /v1/analyst/tools/wallets/:chain/:address/counterparties`
  - `GET /v1/analyst/tools/wallets/:chain/:address/graph`
  - `GET /v1/analyst/tools/wallets/:chain/:address/behavior-patterns`
  - `GET /v1/analyst/findings/:findingId`
  - `GET /v1/analyst/findings/:findingId/evidence-timeline`
  - `GET /v1/analyst/findings/:findingId/historical-analogs`
- Current State:
  - analyst-prefixed route baseline이 실제 backend route로 연결됨
  - home/discover, wallet detail, entity page가 analyst loader를 우선 사용하도록 전환됨
  - findings feed query parsing이 repeated `type=` 형식까지 읽도록 정리됨
  - deterministic analyst tool endpoint 3종(`counterparties`, `graph`, `behavior-patterns`)이 실제 backend route로 연결됨
  - finding drill-down endpoint 3종(`detail`, `evidence-timeline`, `historical-analogs`)이 실제 backend route로 연결됨
  - `counterparties`가 이제 `returnedCount`, `requestedLimit`, `minInteractions`를 함께 반환해 analyst tool 호출 결과를 그대로 재사용 가능
  - `behavior-patterns`가 `keyFindings`, `entryFeatures`, `returnedCount`를 함께 반환해 label/finding/entry-outcome 맥락을 한 번에 소비 가능
  - `evidence-timeline`이 treasury/MM path-quality metadata와 early-entry outcome items를 top-level field로 lift하고, `next_watch`도 timeline item으로 노출
  - `historical-analogs`가 `similarityScore`, `matchedFeatures`, `similarAnalogCount`를 반환해 analyst surface에서 유사도 설명이 가능
  - `POST /v1/analyst/findings/:findingId/explain` baseline이 추가되어 finding bundle 기준 on-demand AI explanation을 요청할 수 있음
  - `ai_explanations` cache store가 same-input hash cache와 scope cooldown을 관리해 과도한 LLM 재호출을 억제함
  - OpenAI client 미설정 시에도 deterministic fallback explanation을 반환해 product path를 유지함
  - `POST /v1/analyst/wallets/:chain/:address/explain` 추가되어 wallet brief 기준 on-demand AI explanation을 요청할 수 있음
  - explanation daily quota는 `audit_logs` 기준 append-only generation count를 사용해 shared cache row overwrite 문제를 피함
  - exact cache hit는 quota를 차감하지 않고, `forceRefresh`는 scope cooldown만 우회하며, `async`는 queued regeneration을 `202`로 반환
  - `POST /v1/analyst/wallets/:chain/:address/analyze` 추가되어 question -> brief/patterns/counterparties/graph/timeline/analogs orchestration을 bounded analyst answer로 반환할 수 있음
- Definition of Done:
  - analyst-prefixed read routes가 실서빙 경로로 동작함
  - deterministic tool/drill-down 응답이 analyst surface에서 추가 재조합 없이 바로 소비 가능함

### WG-050 Bridge and Exchange Evidence Engine

- Status: `Done`
- Owner: `provider-integration-engineer`
- Support: `data-platform-engineer`, `intelligence-engineer`
- Depends On: `WG-048`, `WG-049`
- Deliverables:
  - `wallet_bridge_links`
  - `wallet_bridge_features_daily`
  - `wallet_exchange_flow_features_daily`
  - `bridge_link_confirmation`, `deposit_like_path`, `exchange_pressure_ratio` evidence
  - stronger `cross_chain_rotation` / `cex_deposit_pressure` finding rules
- Current State:
  - migration `0017_bridge_exchange_evidence.sql` 추가
  - `packages/db/bridge_exchange_evidence.go`에서 bridge/exchange path observation + daily feature store 구현
  - shadow-exit snapshot worker가 bridge/exchange evidence store를 읽고 findings/timeline bundle에 `tx/path/entity/counterparty ref`를 materialize
  - analyst finding timeline이 evidence metadata에서 `txRef`, `pathRef`, `entityRef`, `counterpartyRef`를 직접 반환
- Definition of Done:
  - bridge/exchange finding이 single-touch heuristic가 아니라 recurrence + path evidence를 요구함
  - analyst drill-down에서 tx/path/entity ref를 함께 반환함

### WG-051 Treasury Redistribution and MM Handoff Engine

- Status: `Done`
- Owner: `intelligence-engineer`
- Support: `provider-integration-engineer`, `ops-admin-engineer`
- Depends On: `WG-050`
- Deliverables:
  - `wallet_treasury_features_daily`
  - `wallet_mm_features_daily`
  - `treasury_anchor_match`, `treasury_fanout_signature`, `project_to_mm_path`, `inventory_rotation_pattern` evidence
  - stronger `treasury_redistribution` / `suspected_mm_handoff` finding rules
- Current State:
  - migration `0018_treasury_mm_evidence.sql` 추가
  - `packages/db/treasury_mm_evidence.go`에서 treasury/MM path observation + daily feature store baseline 구현
  - shadow-exit snapshot worker가 treasury/MM evidence store를 읽고 `treasury_redistribution` / `suspected_mm_handoff` gating과 evidence bundle을 강화
  - treasury/MM finding bundle이 `entityRef`, `downstreamRef`, `pathRef`, `txRef`를 직접 포함하고, `suspected_mm_handoff`는 이제 root fund/treasury anchor 없이 발화하지 않음
  - direct `RunSnapshot` 경로에서는 label-only treasury/MM interpretation을 더 이상 만들지 않고, treasury/MM evidence report가 있는 snapshot만 interpretation finding을 발화
  - `treasury_redistribution`는 `anchor + operational fanout + stronger market path`, `suspected_mm_handoff`는 `root anchor + project_to_mm + post-handoff evidence`를 최소 조건으로 요구하도록 tightened
  - migration `0020_treasury_mm_path_quality.sql` 추가
  - treasury market path를 `exchange / bridge / MM`로 분리하고, MM post-handoff를 `exchange touch / bridge touch`로 나눠 path-quality feature store를 더 세분화
  - `treasury_operational_distribution`과 `project_to_mm_contact`를 explicit path kind로 분리해, operational-only treasury flow와 contact-only MM adjacency를 confirmed finding과 구분
  - migration `0021_treasury_mm_contact_refinement.sql` 추가
  - treasury operational flow를 `internal ops / external ops`로, MM contact-only flow를 `routed candidate / adjacency`로 다시 분해해 confirmed path 전단의 약한 신호를 더 세밀하게 구분
  - `project_to_mm_routed_candidate`는 이제 actual downstream continuation이 관찰될 때만 생성되고, treasury external ops도 `market-adjacent / non-market`로 더 세분화
  - treasury external ops는 `direct / routed market-adjacent` confidence ladder를 갖고, MM contact-only도 `desk / liquidity / router` subtype을 path kind에 남김
  - treasury/MM path metadata가 `sourceSubtype`, `downstreamSubtype`, `pathStrength`, `confidenceTier`를 포함하도록 보강됨
  - analyst `evidence-timeline`이 treasury/MM path-quality metadata를 top-level field로 lift하고, `next_watch`를 timeline 항목으로 포함해 drill-down에서 직접 소비 가능
  - bridge-only weak treasury path, post-handoff 없는 MM contact, bridge-only MM downstream without rotation/repeat를 suppress하는 회귀 테스트 추가
  - `RunSnapshotForWallet` 기준 treasury/MM evidence-backed finding regression test 추가
- Definition of Done:
  - treasury/MM finding이 label presence만으로 발화하지 않음
  - anchor + flow/path evidence가 최소 조건이 됨

### WG-052 Early Rotation and High Conviction Redefinition

- Status: `Done`
- Owner: `intelligence-engineer`
- Support: `data-platform-engineer`, `product-ui-engineer`
- Depends On: `WG-050`, `WG-051`
- Deliverables:
  - `wallet_entry_features_daily`
  - `first_entry_before_crowding`, `quality_wallet_overlap`, `persistence_after_entry`, `repeat_early_entry_success` evidence
  - `early_rotation_pattern` rename/redefinition
  - stronger `high_conviction_entry` / later `high_conviction_holder` contract
- Current State:
  - first-connection snapshot report가 `quality_wallet_overlap_count`, `first_entry_before_crowding_count`, `best_lead_hours_before_peers`, `persistence_after_entry_proxy_count`, `top_counterparties`를 함께 materialize
  - `high_conviction_entry` gate가 novelty-only 조건에서 `quality overlap + first entry before crowding + persistence/repeat proxy` 중심으로 재작성됨
  - signal payload와 finding bundle이 richer early-entry evidence를 함께 싣도록 보강되고, top counterparty overlap evidence가 `leadHoursBeforePeers`까지 포함함
  - `wallet_entry_features_daily` migration/store baseline 추가, first-connection snapshot worker가 early-entry evidence를 daily feature row로도 upsert
  - wallet brief API가 latest entry features를 실제로 읽고, finding이 비어 있을 때도 early-entry overlap 문구를 deterministic brief에 반영
  - wallet brief/feed wording이 top counterparty overlap ref와 lead/persistence evidence를 직접 쓰도록 보강
  - `repeatEarlyEntrySuccess`가 overlap 강도, lead hour, persistence, repeatable counterparty를 모두 만족할 때만 서도록 보수화
  - candidate semantics도 `qualified overlap / qualified first-entry / qualified persistence` 기준으로 올라가 raw peer hit를 바로 고확신 evidence로 쓰지 않음
  - `wallet_entry_features_daily`가 `sustained overlap`과 `strong lead`를 별도 feature로 저장하고, high-conviction gate와 brief/feed wording이 이를 직접 사용
  - first-connection feed reader가 latest `wallet_entry_features_daily`를 직접 조인해 `quality overlap / first entry before crowding / best lead / persistence / repeat success / top counterparty overlap` evidence를 recommendation과 score evidence에 우선 반영
  - worker가 이전 early-entry row를 later pass로 다시 읽어 `post_window_follow_through`, `max_post_window_persistence_hours`, `short_lived_overlap_count`, `holding_persistence_state`, `outcome_resolved_at` metadata를 성숙시키는 baseline 추가
  - wallet brief/feed가 이제 `holding_persistence_state` 기준으로 `sustained / monitoring / short-lived` wording을 구분하고, short-lived early-entry를 high-conviction처럼 말하지 않도록 보수화
  - `repeatEarlyEntrySuccess`가 현재 snapshot proxy만이 아니라 과거 `sustained` outcome row를 다시 읽는 historical repeat quality를 반영하고, high-conviction finding confidence / next-watch ranking도 그 결과를 사용
  - analyst `evidence-timeline`이 latest `wallet_entry_features_daily`를 읽어 `quality overlap`, `first entry before crowding`, `best lead`, `holding persistence state`, `repeat early-entry quality`, `top counterparty overlap`을 timeline item으로 직접 노출
- Definition of Done:
  - alpha score alias가 아니라 entry/persistence/outcome evidence 기반 finding으로 동작함
  - wallet brief, findings feed, analyst drill-down에서 evidence-driven wording과 outcome-aware context를 일관되게 사용함

### WG-053 Pre-listing and Early Entry Engine

- Status: `Todo`
- Owner: `intelligence-engineer`
- Support: `data-platform-engineer`, `provider-integration-engineer`
- Depends On: `WG-052`
- Deliverables:
  - `token_entry_features_daily`
  - `first_entry_before_listing`, `quality_wallet_overlap`, `entry_persistence`, `listing_event_linkage` evidence
  - pre-listing accumulation / early-entry finding contract
- Current State:
  - later backlog only
  - 현재 active priority가 아니며, `WG-050`~`WG-052` core engine hardening 이후에만 착수
- Definition of Done:
  - token-first early-entry signal이 wallet alpha alias가 아니라 독립 evidence engine으로 동작함
  - listing/TGE/event linkage와 quality-wallet overlap이 finding bundle에 직접 포함됨

## Sprint 0

### WG-001 Scope Freeze

- Status: `Done`
- Owner: `foundation-architect`
- Support: `billing-launch-engineer`
- Depends On: 없음
- Deliverables:
  - production scope matrix
  - Must/Should/Later 분류표
  - provider budget sheet 초안
- Definition of Done:
  - production launch 필수 기능 범위가 문서로 고정됨
  - 비목표 기능이 별도 목록으로 분리됨

### WG-002 Domain Contract Baseline

- Status: `Done`
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

- Status: `Done`
- Owner: `foundation-architect`
- Depends On: `WG-001`
- Deliverables:
  - `apps/`, `services/`, `libs/`, `infra/`, `docs/` 구조
  - `pnpm` workspace와 Go workspace/module support
  - lint/format/typecheck/test 기본 구성
- Definition of Done:
  - web workspace install과 backend Go module check가 동작함

### WG-004 Environment and Secret Validation

- Status: `Done`
- Owner: `foundation-architect`
- Depends On: `WG-003`
- Deliverables:
  - env loader
  - startup validation
  - `.env.example` 정책
  - current state: local docker compose Postgres port는 `5433`, Neo4j 포트는 `8687` 기준으로 정리됐고, 루트 `dev:stack`/`dev:stack:no-worker` 스크립트가 `.env` 자동 로드 + docker infra + api/web boot path를 제공하며, `dev:worker:loop`가 batch worker를 재시작 루프로 실행하도록 정리됨
  - current state: home search는 public wallet-address query만 URL `?q=`에 유지하고, ENS-like/unknown query는 로컬 상태로만 유지하도록 정리됨
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
  - current state: Clerk JWT/JWKS verification baseline 완료, legacy header verifier fallback 유지, local development에서는 example Clerk config 사용 시 legacy header verifier로 부팅 가능
  - current state: web app router import/tsconfig는 Next bundler 규칙으로 정리되어 local `dev:web` compile 오류 없이 부팅 가능
- Definition of Done:
  - role 기반 접근 제어가 API와 웹 모두에서 가능함

## Sprint 1

### WG-006 Local Infra Stack

- Status: `Done`
- Owner: `data-platform-engineer`
- Depends On: `WG-003`
- Deliverables:
  - local Postgres
  - local Neo4j
  - local Redis
  - docker compose or equivalent
  - current state: `corepack pnpm dev:infra`로 infra만, `corepack pnpm dev:stack`으로 infra + api + worker loop + web 전체 stack을 한 번에 실행 가능
  - current state: home search 입력창은 URL `q` hydrate 후에도 사용자가 직접 수정/삭제 가능하도록 상태 동기화 버그 수정 완료
- Definition of Done:
  - 개발자가 로컬에서 핵심 인프라를 재현 가능

### WG-007 Postgres Schema v1

- Status: `In Progress`
- Owner: `data-platform-engineer`
- Support: `ops-admin-engineer`, `billing-launch-engineer`
- Depends On: `WG-002`, `WG-006`
- Deliverables:
  - users, subscriptions, wallets, tokens, entities, transactions, clusters, signals, alerts, watchlists, suppressions, audit 관련 migration
- Current State:
  - `entities.display_name` / `updated_at` baseline 컬럼 추가
  - admin curated list를 source로 `entities`와 `wallets.entity_key`를 재동기화하는 baseline index sync 추가
  - `heuristic entity assignment` store baseline 추가: ingest/backfill이 `heuristic:*` entity를 upsert하고 `wallets.entity_key`를 채울 수 있음
  - heuristic assignment는 기존 `curated:*` 및 기타 non-heuristic mapping을 덮어쓰지 않도록 보존
  - migration `0015_wallet_labeling_baseline.sql`로 `entity_labels`, `entity_label_memberships`, `wallet_evidence` 스키마 추가
  - `packages/db/wallet_labels.go`가 label/evidence upsert와 changed-wallet summary/graph invalidation baseline 제공
- Definition of Done:
  - `plan.md`의 핵심 테이블이 생성 가능
  - 주요 인덱스와 unique key가 포함됨

### WG-008 Neo4j Schema Bootstrap

- Status: `In Progress`
- Owner: `data-platform-engineer`
- Depends On: `WG-006`
- Deliverables:
  - node labels
  - relationship constraints
  - graph index bootstrap script
- Current State:
  - graph read model에서 `wallet.entity_key` 기반 `entity_linked` edge와 `Entity` node 노출 baseline 완료
  - admin curated list sync로 들어간 entity label이 graph node label로 바로 반영되도록 read path 보강
- Definition of Done:
  - Wallet, Token, Entity, Cluster 및 핵심 관계 upsert가 가능함

### WG-009 Raw Payload Storage Pipeline

- Status: `Done`
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

- Status: `In Progress`
- Owner: `data-platform-engineer`
- Support: `api-platform-engineer`
- Depends On: `WG-007`, `WG-009`
- Deliverables:
  - dedup key strategy
  - Redis/Postgres dedup store
  - duplicate-safe ingest helper
- Current State:
  - Redis `IngestDedupStore`가 `Claim` + `Release`를 지원하도록 확장됨
  - historical backfill worker와 provider webhook ingest가 `DB upsert/materialization` 실패 시 claimed dedup key를 release해 retry poisoning을 막는 baseline 추가
- Definition of Done:
  - 동일 이벤트 재수신 시 중복 저장/중복 알림이 발생하지 않음

### WG-011 Normalized Transaction Schema

- Status: `Done`
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

- Status: `Done`
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

- Status: `In Progress`
- Owner: `provider-integration-engineer`
- Depends On: `WG-011`, `WG-012`
- Deliverables:
  - Dune export adapter
  - seed candidate input parser
  - fixture-based contract tests
- Current State:
  - Dune adapter가 `fixture 1건 반환`에서 `export row -> SeedDiscoveryCandidate` parser 경로로 올라감
  - row 기반 chain normalize, confidence clamp, observed-at parse, metadata carry-through baseline 추가
  - invalid row skip와 fixture-based contract tests로 seed discovery input 계약을 고정
  - `DUNE_SEED_EXPORT_JSON` / `DUNE_SEED_EXPORT_PATH`를 통해 env/file 기반으로 seed export row를 실제 주입 가능
  - configured registry가 주입된 row를 Dune adapter에 실어 worker seed discovery 경로에서 바로 사용 가능
- Definition of Done:
  - Dune 결과를 seed candidate 입력으로 변환 가능

### WG-014 Alchemy Adapter

- Status: `In Progress`
- Owner: `provider-integration-engineer`
- Depends On: `WG-011`, `WG-012`
- Deliverables:
  - historical transfer adapter
  - realtime webhook adapter
  - retry/backoff handling
  - current state: local lazy indexing/worker backfill 경로에서 `alchemy_getAssetTransfers.maxCount`를 hex quantity로 맞춰 `Invalid hex string` 오류를 제거함
  - current state: historical fetch가 `FromAddress + ToAddress` 양방향 counterparty 수집과 transfer-key dedup baseline까지 포함하도록 확장됨
  - current state: wallet detail related-address surface를 위해 inbound/outbound counterparty 집계 품질을 높이는 방향으로 EVM historical fetch baseline 보강됨
  - current state: nil transfer value가 `"<nil>"`로 정규화돼 `amount_numeric` 캐스팅을 깨뜨리던 경로를 provider/domain/db 3중 sanitize로 차단해 lazy indexing worker crash를 제거함
  - current state: `ALCHEMY_SOLANA_BASE_URL` 기준 standard Solana RPC fallback(`getSignaturesForAddress` + `getTransaction`)과 native/token delta 기반 counterparty 추론 baseline 추가
  - current state: explicit `wallet_entity_*` / `counterparty_entity_*` metadata가 존재할 경우 ingest/backfill이 heuristic entity assignment로 바로 연결 가능
- Definition of Done:
  - EVM backfill과 watchlist realtime ingest가 가능

### WG-015 Helius Adapter

- Status: `In Progress`
- Owner: `provider-integration-engineer`
- Depends On: `WG-011`, `WG-012`
- Deliverables:
  - wallet history adapter
  - transfers/funded-by/identity adapter
  - schema-versioned parser
- Current State:
  - historical wallet activity baseline은 `getTransactionsForAddress` + Helius Data API `parseTransactions` 경로로 동작
  - historical enrichment에서 `direction`, `counterparty_address`, `amount`, `token_*`, `funder_address`, `helius_identity_*` metadata를 추출해 normalized transaction path와 summary/graph 집계에 바로 기여하도록 보강 완료
  - `schema_version=2` metadata를 통해 parsed enrichment 경로를 구분하는 baseline 추가
  - Helius Data API `parseTransactions`가 paid-plan 403을 반환해도 enrichment만 생략하고 wallet history ingest는 계속 진행되도록 graceful fallback 추가
  - `getTransactionsForAddress` 자체가 paid-plan 403인 경우에도 client fallback + runner fallback 이중 안전장치로 `Alchemy Solana RPC fallback` historical ingest를 계속 진행하도록 보강
  - paid-plan fallback 자체가 실패해도 원래 Helius 403만 남기지 않고 실제 fallback failure를 반환하도록 보강해 worker 로그에서 원인을 직접 확인 가능
  - `.env`에 `HELIUS_BASE_URL`/`HELIUS_DATA_API_BASE_URL`를 full RPC URL + `api-key` query 형태로 넣어도 canonical base URL로 정규화되도록 보강
  - local `.env`에도 `ALCHEMY_SOLANA_BASE_URL`를 명시해 Solana fallback RPC host를 EVM용 Alchemy host와 분리해 사용하도록 정리
  - provider HTTP client 기본 timeout과 wallet backfill 시작 로그를 추가해 Solana historical fallback이 오래 대기하는 경우에도 worker가 멈춘 것처럼 보이지 않게 보강
  - `helius_identity_source` / `helius_source` 기반 heuristic entity assignment baseline 추가: generic program source는 제외하고 counterparty/funder wallet만 `heuristic:<chain>:<slug>` namespace로 자동 연결
- Definition of Done:
  - Solana profile, funder, counterparty 추출이 가능

### WG-016 Moralis Enrichment Adapter

- Status: `In Progress`
- Owner: `provider-integration-engineer`
- Depends On: `WG-011`, `WG-012`
- Deliverables:
  - optional enrichment client
  - on-demand cache policy
- Current State:
  - Moralis live client baseline은 `wallets/:address/net-worth` + `wallets/:address/chains` + `wallets/:address/tokens` on-demand fetch 경로로 구현
  - Redis-backed on-demand cache policy baseline을 추가해 동일 EVM wallet enrichment를 cache-first로 재사용 가능
  - `wallet summary` service가 Moralis enrichment를 optional overlay로 얹을 수 있게 정리했고, 상세 화면은 `net worth`, `native balance`, `active chains`, `top holdings`를 바로 표시 가능
  - Moralis endpoint 일부(`net-worth`, `chains`, `holdings`)가 실패해도 남은 데이터로 partial enrichment를 계속 반환하도록 보강
  - wallet detail enrichment 카드에서 `updated/source/active chains` compact polish와 `top holdings` surface를 반영
  - 별도 `moralis-enrichment-refresh` worker mode baseline을 추가해 특정 wallet target의 enrichment cache를 수동/스케줄 기반으로 갱신할 수 있는 경로를 마련
- Definition of Done:
  - 상세 화면용 enrichment를 보조 계층으로 호출 가능

## Sprint 2

### WG-017 Seed Discovery Batch

- Status: `In Progress`
- Owner: `provider-integration-engineer`
- Support: `intelligence-engineer`
- Depends On: `WG-013`, `WG-007`
- Deliverables:
  - candidate scoring batch
  - seed watchlist seeding job
- Current State:
  - `seed-discovery-enqueue` worker mode를 추가해 Dune seed candidate를 기존 wallet backfill queue로 handoff 가능
  - seed-discovery candidate metadata/source/confidence가 backfill job metadata에 그대로 실리도록 baseline 정리
  - Redis dedup + job run logging을 재사용해 최소 seed enqueue batch path 확보
  - `seed-discovery-seed-watchlist` worker mode를 추가해 top-N high-confidence `seed_label` 후보를 system-owned seed watchlist로 편입 가능
  - 기본 threshold는 `confidence >= 0.8`, 기본 selection budget은 `top 10`으로 보수적으로 제한
- Definition of Done:
  - seed whale 후보가 주기적으로 생성되고 편입 가능

### WG-018 Historical Backfill Worker

- Status: `In Progress`
- Owner: `provider-integration-engineer`
- Support: `data-platform-engineer`
- Depends On: `WG-014`, `WG-015`, `WG-010`, `WG-011`
- Deliverables:
  - 180일 기본 backfill worker
  - 1-hop 및 제한된 2-hop 확장 로직
  - service address stop rule
- Current State:
  - search miss, watchlist bootstrap, seed discovery enqueue가 source-aware backfill policy metadata를 queue job에 실어 보내도록 정리
  - worker가 `backfill_window_days`, `backfill_limit`, `backfill_expansion_depth`, `backfill_stop_service_addresses`를 읽어 실제 batch window/limit를 해석하는 baseline 추가
  - 기본 정책은 `search=180d/750/1-hop`, `watchlist|seed=365d/750/2-hop`, metadata override는 상한(`365d`, `1000 limit`, `2-hop`) 내에서만 허용
  - `watchlist_bootstrap` / `seed_discovery` source에서는 normalized transaction counterparties 상위 후보를 bounded fanout으로 `wallet_backfill_expansion` job에 재enqueue하는 baseline 추가
  - expansion job은 `backfill_root_*`, `backfill_parent_*`, `backfill_expansion_hits` metadata를 남기고, Redis dedup으로 중복 re-enqueue를 방지
  - search hit라도 `indexing.status=ready` 이고 `lastIndexedAt`이 stale threshold를 넘은 wallet은 `search_stale_refresh` source로 low-priority background refresh를 enqueue
  - wallet summary/detail에 `indexing status`, `coverage window`, `last indexed`를 노출하는 baseline 추가
  - home search surface와 wallet detail이 `indexing` 상태일 때 자동 polling으로 summary/graph를 다시 불러와 `ready` 상태로 자연스럽게 전환되도록 보강
  - home/detail copy를 `background indexing`, `coverage ready`, `updated just now / 5m ago` 형태로 정리해 기술 상태 노출을 줄이고 조사 흐름 중심으로 다듬음
  - `GET /v1/search?refresh=manual`을 통해 사용자가 명시적으로 stale/fresh wallet의 background refresh를 다시 enqueue할 수 있게 하고, home/detail UI에 `Expand coverage` 액션을 추가
  - related addresses와 home snapshot에 `showing X of Y indexed`, `180d/365d indexed` 형태의 coverage visibility baseline 추가
- Definition of Done:
  - 기본 기간 내 wallet graph용 데이터가 적재됨

### WG-019 Graph Materialization Pipeline

- Status: `Done`
- Owner: `data-platform-engineer`
- Support: `intelligence-engineer`
- Depends On: `WG-008`, `WG-011`, `WG-018`
- Deliverables:
  - normalized event -> Neo4j edge upsert pipeline
  - neighborhood snapshot strategy
- Current State:
  - inbound normalized transfers에 대해 `FUNDED_BY` 관계를 Neo4j materialization/read path에 반영하는 baseline 완료
  - wallet graph 계약에 `base` / `derived` edge family를 추가하고, graph UI에 family filter(`All / Base / Derived`)와 derived edge 시각 구분을 반영
  - graph edge에 `evidence` + `tokenFlow` 계약을 추가하고, `Neo4j edge metadata + Postgres counterparty aggregate`를 합성해 `selected relationship` 패널에서 token flow / confidence / latest tx evidence를 읽을 수 있게 정리
  - `INTERACTED_WITH` edge가 Neo4j materialization 시 inbound/outbound directional count를 함께 누적하고, reader/evidence/UI가 이를 `Transfer activity`, `Sent`, `Received`, `Mixed flow` copy로 노출하도록 강화
  - graph edge 계약에 explicit `directionality(sent/received/mixed/linked)`를 추가하고, live graph / summary-derived graph / selected relationship panel이 더 이상 `lastDirection`나 token-flow count를 각자 재해석하지 않고 동일 계약 필드를 직접 사용하도록 정리
  - canonical `1-hop` graph를 `wallet_graph_snapshots` Postgres store에 durable snapshot으로 저장하고, `Redis -> Postgres snapshot -> live Neo4j` read order와 backfill/webhook write-path invalidation baseline까지 연결
  - provider metadata(`helius_source` / `helius_identity_source`)를 `heuristic:<chain>:<slug>` entity namespace로 정규화해 worker ingest 중 counterparty/funder/fee-payer wallet assignment를 자동 upsert하는 baseline 추가
  - explicit provider source가 없는 activity도 chain별 known address catalog(`Seaport`, `SeaDrop`, `OpenSea`, `Relay.link`, `Wrapped Ether`, `Uniswap`, `1inch`, `Cow Swap`, `Jupiter`)와 metadata label/service pattern(`OpenSea: Fees`, `Binance Hot Wallet`, `Jupiter Aggregator`) 및 source alias normalization(`JUP-AG`)을 통해 heuristic entity assignment를 계속 생성하도록 보강
  - `entity_linked` edge evidence source가 `provider-heuristic-identity` / `curated-identity-index` / `postgres-wallet-identity`를 구분하도록 보강해 graph selected-node / related-address row에서 source badge가 직접 읽히게 정리
  - curated entity index sync와 heuristic entity assignment가 `wallets.entity_key` / `entities.display_name`을 갱신할 때, 영향을 받는 wallet의 canonical graph cache/`wallet_graph_snapshots`도 함께 invalidation 하도록 연결해 entity-linked graph stale 문제를 줄임
  - canonical snapshot, directionality, entity-linked evidence source, graph cache/snapshot invalidation baseline까지 완료되어 backend/data 기준 DoD를 충족
- Definition of Done:
  - wallet 중심 1-hop neighborhood 조회가 가능

### WG-020 Wallet Summary Materializer

- Status: `Done`
- Owner: `api-platform-engineer`
- Support: `data-platform-engineer`
- Depends On: `WG-007`, `WG-018`
- Deliverables:
  - wallet daily stats 집계
  - top counterparties 집계
  - netflow/holdings/latest signals aggregate
  - latest cluster score snapshot read path from `signal_events`
  - counterparty amount/token breakdown aggregate
- Current State:
  - worker historical backfill 성공 시 `wallet_daily_stats` cumulative snapshot refresh baseline 추가, `wallet_id` 기준 transaction/counterparty/inbound/outbound/latest-activity 누적 스냅샷을 생성
  - wallet summary stats query가 `wallet_daily_stats`를 우선 읽고, `transactions`는 top counterparties / exact activity bounds 보강용으로만 쓰도록 정리
  - Alchemy/Helius webhook live ingest도 `wallet_daily_stats` refresh를 함께 수행해, realtime webhook 반영 후 summary aggregate freshness가 historical backfill 경로와 동일하게 유지되도록 보강
  - `top counterparties`, `recent flow`, `counterparty amount/token breakdown`은 summary read path에 반영 완료
  - counterparty aggregate가 `entity_key` / `entity_type` / `entity_label`을 직접 포함해 related-address table과 summary-derived graph가 graph warmup 전에도 entity context를 유지하도록 보강
  - provider metadata가 약한 주소도 known-address + metadata label/service heuristic assignment를 타면 summary/related-address surface에 entity label이 더 빨리 보이도록 보강
  - Moralis enrichment overlay를 통해 `holdings/balance` surface가 wallet summary/detail 계약에 연결됨
  - `wallet_enrichment_snapshots` durable snapshot baseline 추가: summary repo가 `chain/address` 기준 Postgres snapshot을 읽어 Redis cold-start나 live fetch 실패 상황에서도 `holdings/balance` surface를 유지
  - live Moralis fetch가 발생하면 Redis cache뿐 아니라 Postgres enrichment snapshot도 함께 upsert되도록 보강, worker `moralis-enrichment-refresh`와 backfill-triggered enrichment refresh가 같은 snapshot 경로를 공유
  - `latest signals` surface를 wallet summary/detail 계약과 UI에 반영하는 baseline 완료
  - `signal_events` 기반 `materialized latest signals aggregate` read model 추가: summary cache/input이 worker snapshot 최신 row를 직접 읽고, API repository는 materialized latest signals가 있으면 이를 우선 사용하며 없을 때만 score evidence 재조합 fallback을 사용
  - EVM historical backfill 성공 후 Moralis cache를 prewarm하는 `async enrichment refresh` baseline과 별도 `moralis-enrichment-refresh` worker mode baseline을 worker에 추가
  - backfill, webhook ingest, signal snapshot, live Moralis refresh가 모두 wallet summary cache invalidation을 수행하도록 연결해 TTL까지 stale summary가 남는 문제를 정리
- Definition of Done:
  - wallet summary 응답에 필요한 캐시 데이터가 생성됨

### WG-021 Wallet Summary API

- Status: `Done`
- Owner: `api-platform-engineer`
- Depends On: `WG-020`, `WG-005`
- Deliverables:
  - `GET /wallet/:chain/:address/summary`
  - unknown/new wallet fallback 처리
- Definition of Done:
  - summary API가 핵심 카드 정보를 반환함
  - 신규 주소도 빈 응답 대신 설명 가능한 fallback을 가짐

### WG-022 Wallet Graph API

- Status: `Done`
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

- Status: `Done`
- Owner: `api-platform-engineer`
- Support: `product-ui-engineer`
- Depends On: `WG-021`, `WG-007`
- Deliverables:
  - address/token/entity search endpoint
  - parser for EVM, Solana, ENS
- Definition of Done:
  - 주요 검색 타입을 하나의 entry point로 처리 가능

### WG-024 Product Search and Wallet UI

- Status: `Done`
- Owner: `product-ui-engineer`
- Depends On: `WG-021`, `WG-022`, `WG-023`
- Deliverables:
  - global search UI
  - wallet summary page
  - wallet graph visual baseline
  - counterparties/timeline base UI
- Current State:
  - search -> wallet detail route와 wallet summary/graph preview baseline 완료
  - wallet detail SVG graph, edge-kind filter, node/entity visual distinction baseline 완료
  - neighborhood summary, density guardrail, low-confidence dashed edge baseline 완료
  - live/fallback wallet graph response에 neighborhood summary read path 연결 완료
  - Redis-backed wallet graph snapshot cache와 snapshot metadata read path baseline 완료
  - wallet detail summary 카드에 `related addresses(top counterparties)`와 `recent flow` baseline 노출 완료
  - related addresses에 `direction`, `inbound/outbound count`, `first seen`, `latest activity` baseline 노출 완료
  - related addresses를 분석용 table로 정리하고 `direction filter` + `sort` baseline 완료
  - related addresses table에 `amount` column을 추가하고 `inbound/outbound amount + primary token` baseline 노출 완료
  - related addresses toolbar에 `token filter`를 추가하고 `total/outbound/inbound volume sort` baseline 완료
  - related addresses 각 row에서 token breakdown을 expand해서 token별 `IN/OUT/total`을 바로 읽을 수 있는 baseline 완료
  - related addresses 각 row에서 `copy address`와 `focus in graph` 액션을 제공하고, graph selection state를 상세 화면이 제어할 수 있도록 baseline 연결 완료
  - related addresses expanded row에서 `copy summary`와 `open in search` 액션을 제공하고, 홈 검색 surface가 `?q=` query param으로 같은 주소를 즉시 다시 여는 baseline 완료
  - graph API가 depth gate/empty neighborhood로 fallback될 때 summary counterparties에서 파생한 `summary-derived` graph로 관련 주소 시각화 유지
  - wallet detail hero/summary/graph 패널에서 `GET /v1/...` route 노출을 제거하고 compact graph variant 기준으로 정보 위계를 정리
  - PRD 그래프 UX 방향을 Qorvi 고유의 `분석형 hub-and-spoke + partial flow hints`와 signal-first investigation view로 고정
  - focal wallet 중심 `hub-and-spoke + partial flow hints` visual layout baseline 완료
  - home search surface를 result-first layout으로 단순화하고 compact wallet graph preview 연결 완료
  - home main page를 graph-first layout으로 재구성하고, summary는 compact side card로 축소
  - home main page를 `search + hero graph + right rail snapshot` 구조로 재정리했고, 홈 그래프는 `hero` variant로 확대되어 보조 설명 밀도를 더 줄임
  - React Flow 기반 `pan/zoom`, `MiniMap`, `Controls`, node selection baseline 완료
  - selected node panel에 `wallet/cluster` direct CTA를 추가하고, node double-click으로 detail route 이동 가능
  - home/detail graph 패널에 `Visible relationships` 리스트를 추가해 live Neo4j graph와 summary-derived fallback 관계를 더 직접적으로 읽을 수 있게 정리
  - selected wallet node 기준으로 `Expand 2-hop` 액션을 추가하고, fetched neighborhood를 현재 canvas에 local merge하는 bounded 2-hop expand baseline 완료
  - selected node 기준 `cluster/entity stop rule`, local expansion budget cap, selected node inspector baseline 완료
  - entity node subtitle을 `indexed entity label` 기준으로 정리하고, selected node inspector에서 `linked entities / entity linkage` strip과 `Search label` CTA baseline을 노출하도록 보강
  - `entity_linked` edge evidence source를 읽어 selected node inspector와 related-address row에 `heuristic/provider/curated` assignment source badge를 함께 노출하는 baseline 추가
  - `Uncodixfy.md` 기준으로 home/detail 스타일을 정리해 과한 pill, blur, gradient, route copy를 줄이고 graph-first layout은 유지한 채 flat card + quieter typography 중심으로 재배치
  - home/detail/search가 `live / summary-derived / unavailable` 상태와 counterparty entity fallback/source badge를 shared presenter helper로 같은 규칙으로 읽도록 정리
- Definition of Done:
  - search -> wallet detail 흐름이 웹에서 동작함
  - wallet graph preview가 텍스트 목록이 아니라 시각 그래프로 제공됨

## Sprint 3

### WG-025 Cluster Signal Calculators

- Status: `In Progress`
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

- Status: `Done`
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

- Status: `Done`
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

- Status: `Done`
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

- Status: `Done`
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

- Status: `In Progress`
- Owner: `intelligence-engineer`
- Depends On: `WG-017`, `WG-018`, `WG-019`
- Deliverables:
  - 90일 무이력 판정기
  - 24시간 내 신규 공통 진입 감지
  - `alpha_score` 계산
  - `signal_events` first-connection snapshot baseline
- Definition of Done:
  - token/protocol 단위 신규 공통 진입 이벤트를 생성 가능

### WG-031 First-Connection API and Hot Feed

- Status: `Done`
- Owner: `api-platform-engineer`
- Support: `product-ui-engineer`
- Depends On: `WG-030`
- Deliverables:
  - `GET /signals/first-connections`
  - hot feed 정렬: 최신순/점수순
- Definition of Done:
  - feed에서 첫 진입 고래, 시각, 규모, score가 노출됨

### WG-032 Watchlist Service

- Status: `In Progress`
- Owner: `api-platform-engineer`
- Support: `product-ui-engineer`
- Depends On: `WG-007`, `WG-005`
- Deliverables:
  - watchlist CRUD
  - watchlist item tags/notes
  - tier-based watchlist limits
  - current state: `wallet watchlist` CRUD/API/Postgres persistence 완료, `cluster/token/entity` item type는 미완
- Definition of Done:
  - wallet, cluster, token, entity를 watchlist에 저장 가능

### WG-033 Alert Rules and Dedup Engine

- Status: `Done`
- Owner: `api-platform-engineer`
- Support: `intelligence-engineer`, `ops-admin-engineer`
- Depends On: `WG-010`, `WG-026`, `WG-028`, `WG-030`, `WG-032`
- Deliverables:
  - alert rule CRUD
  - dedup/cooldown/re-notify engine
  - alert event audit trail
  - current state: owner-scoped protected CRUD, `alert_events` audit trail 조회, severity-aware dedup/cooldown/re-notify baseline, worker `signal snapshot -> alert rule evaluation` 자동 연결 완료. delivery와 alert center는 `WG-034`, `WG-035` 범위
- Definition of Done:
  - 동일 이벤트 중복 발송 방지
  - severity 상승 시 재알림 가능

### WG-034 Alert Delivery Channels

- Status: `Done`
- Owner: `api-platform-engineer`
- Support: `billing-launch-engineer`
- Depends On: `WG-033`
- Deliverables:
  - in-app inbox
  - email delivery
  - Telegram delivery
  - Discord webhook delivery
  - current state: protected `GET /v1/alerts` inbox API, `alert-delivery-channels` CRUD API, worker `alert_events -> email/discord/telegram delivery attempt` baseline, retry batch worker, delivery audit persistence 완료
- Definition of Done:
  - 최소 2개 외부 채널과 in-app이 동작함

### WG-035 Alert Center UI

- Status: `Done`
- Owner: `product-ui-engineer`
- Depends On: `WG-033`, `WG-034`
- Deliverables:
  - active rules 화면
  - triggered events 리스트
  - mute/snooze/severity filter UI
  - current state: protected `/alerts` page baseline, triggered events 리스트, active rules 리스트, delivery channels 리스트, severity/signal filter UI 완료. unread inbox state, cursor pagination, snooze state 표시 baseline 완료. inbox read/unread mutation, explicit mute/resume, snooze/clear snooze in-product mutation UX 완료
- Definition of Done:
  - 사용자가 제품 내에서 알림 상태와 규칙을 관리 가능

### WG-036 Admin Console MVP

- Status: `Done`
- Owner: `ops-admin-engineer`
- Support: `api-platform-engineer`
- Depends On: `WG-007`, `WG-012`, `WG-033`
- Deliverables:
  - label editor
  - suppression rule management
  - curated list editor
  - provider usage dashboard
  - current state: protected admin labels CRUD API, suppressions CRUD API, provider quota snapshot API, curated list CRUD/read API, admin audit log read API, `/admin` labels/suppressions/quotas/curated lists/audit logs UI baseline 완료. in-product suppression add/remove human override action과 deeper quota dashboard 완료
- Definition of Done:
  - 운영자가 DB 직접 접근 없이 핵심 운영 작업을 수행 가능

## Sprint 5

### WG-037 Entitlement Matrix

- Status: `In Progress`
- Owner: `billing-launch-engineer`
- Support: `api-platform-engineer`, `product-ui-engineer`
- Depends On: `WG-005`, `WG-021`, `WG-022`, `WG-032`
- Deliverables:
  - Free/Pro/Team 기능 매트릭스
  - API/UI gating 규칙
  - current state: `packages/billing` entitlement snapshot/helper baseline 완료, `GET /v1/account` / `GET /v1/account/entitlements` baseline 완료, `/account` billing page와 fallback/live entitlement preview baseline 완료. persisted billing account가 있으면 entitlement snapshot에 반영되도록 account service/API wiring 완료
- Definition of Done:
  - 플랜별 허용 범위가 코드와 문서에 반영됨

### WG-038 Stripe Billing Flow

- Status: `Done`
- Owner: `billing-launch-engineer`
- Support: `api-platform-engineer`
- Depends On: `WG-037`
- Deliverables:
  - checkout flow
  - billing webhook
  - subscription reconciliation
  - current state: `apps/api` checkout session endpoint, public `GET /v1/billing/plans`, billing webhook endpoint baseline 완료, Postgres/in-memory billing account persistence baseline 완료, webhook reconciliation으로 `/v1/account` plan override 가능, `.env.example` Stripe env와 `/account`/`/pricing` checkout route 정합성 반영 완료, live Stripe HTTP client baseline과 `billing_checkout_sessions` / `billing_subscriptions` / `billing_subscription_reconciliations` persistence baseline 완료, worker `billing-subscription-sync` mode로 subscription status sync baseline 완료
- Launch Policy:
  - billing capability는 구현 완료 상태로 유지한다.
  - 초기 production launch에서는 Stripe activation을 필수 blocker로 두지 않고, entitlement policy와 운영 흐름을 우선 안정화한다.
  - Stripe activation과 checkout closeout은 post-launch monetization track으로 다룬다.
- Definition of Done:
  - 최소 1개 유료 플랜 결제가 가능

### WG-039 Pricing and Account UI

- Status: `Done`
- Owner: `product-ui-engineer`
- Support: `billing-launch-engineer`
- Depends On: `WG-038`
- Deliverables:
  - pricing page
  - account/billing page
  - current plan display
  - current state: `apps/api` billing pricing catalog endpoint baseline 완료, `/account` billing page와 current plan display baseline 완료, `/pricing` plan cards와 live checkout mutation baseline 완료, graceful fallback/redirect 메시지와 `?checkout=success|cancel&plan=` query-state 기반 success/cancel UX 정리 완료
- Definition of Done:
  - 사용자가 플랜 확인과 업그레이드를 UI에서 수행 가능

### WG-040 Observability Dashboard

- Status: `Done`
- Owner: `billing-launch-engineer`
- Support: `ops-admin-engineer`, `provider-integration-engineer`
- Depends On: `WG-012`, `WG-033`, `WG-034`
- Deliverables:
  - provider usage dashboard
  - ingest lag dashboard
  - alert delivery dashboard
  - error tracking baseline
- Current State:
  - `GET /v1/admin/observability` snapshot route 추가
  - `provider_usage_logs`, `job_runs`, `alert_delivery_attempts` 기반 `provider usage / ingest freshness / recent runs / recent failures / alert delivery health` aggregate 완료
  - `/admin` admin console web boundary와 compact observability panel 연결 완료
- Definition of Done:
  - beta 운영 필수 메트릭이 한 곳에서 확인 가능

### WG-041 Replay and Contract Test Suite

- Status: `Done`
- Owner: `data-platform-engineer`
- Support: `provider-integration-engineer`, `intelligence-engineer`
- Depends On: `WG-009`, `WG-011`, `WG-026`, `WG-028`, `WG-030`
- Deliverables:
  - replay test fixtures
  - provider contract tests
  - scoring consistency tests
- Current State:
  - `packages/providers/testdata`에 `Alchemy`, `Helius`, `Moralis` fixture corpus 추가
  - provider contract tests가 fixture-based shape 검증과 normalized transaction/enrichment 계약까지 고정
  - `apps/api/internal/server/testdata/replay`에 `Alchemy`, `Helius` raw webhook replay fixture 추가
  - replayed raw payload가 동일 normalized transaction set과 동일 `cluster`, `shadow_exit`, `first_connection` score summary를 재현하는 consistency test baseline 추가
- Definition of Done:
  - raw payload 재처리 시 핵심 score 결과가 재현 가능

### WG-042 E2E Beta Flow Test

- Status: `Done`
- Owner: `product-ui-engineer`
- Support: `api-platform-engineer`, `billing-launch-engineer`
- Depends On: `WG-024`, `WG-027`, `WG-031`, `WG-035`, `WG-039`
- Deliverables:
  - browser/API mixed beta-flow Playwright spec
  - search -> wallet -> tracked alerts flash flow 검증
  - checkout -> webhook reconciliation -> upgraded account flow 검증
- Definition of Done:
  - beta 핵심 사용자 플로우가 자동화 또는 문서화된 시나리오로 검증됨

### WG-043 Beta Launch Checklist

- Status: `Done`
- Owner: `billing-launch-engineer`
- Support: 전체
- Depends On: `WG-036`, `WG-038`, `WG-040`, `WG-041`, `WG-042`
- Deliverables:
  - functional gate checklist
  - reliability gate checklist
  - UX gate checklist
  - rollback/runbook package
- Current State:
  - `/Users/kh/Github/Qorvi/docs/runbooks/launch-gates.md`를 beta closeout source of truth로 확장해 `functional`, `reliability`, `UX`, `ops` gate와 `pass / warn / block` 상태를 현재 코드 기준으로 고정
  - evidence bundle 명령을 문서에 직접 명시하고 현재 저장소 기준으로 재실행 가능한 형태로 정리
  - rollback/recovery package에 `search/wallet`, `billing`, `enrichment/provider pressure` 조치와 수동 worker mode 명령을 포함
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

production launch를 가장 늦추는 경로는 아래다.

1. `WG-003` -> `WG-006` -> `WG-007` -> `WG-011`
2. `WG-011` -> `WG-014` + `WG-015` -> `WG-018`
3. `WG-018` -> `WG-020` -> `WG-021` -> `WG-024`
4. `WG-018` + `WG-019` -> `WG-025` -> `WG-026` -> `WG-033` (`WG-018`, `WG-019` 선행은 완료)
5. `WG-030` -> `WG-031`
6. `WG-033` -> `WG-034` -> `WG-035`
7. `WG-037` -> `WG-038` -> `WG-039`
8. `WG-040` + `WG-041` + `WG-042` -> `WG-043`

## 7. Immediate Next Tasks

현재 저장소 상태에서 다음부터는 아래 순서를 그대로 따른다.

1. analyst tool layer / finding drill-down production hardening
   - evidence timeline, historical analogs, wallet/entity drill-down을 analyst surface에서 바로 소비 가능한 수준으로 보강
   - `finding explain`, `wallet explain`, `wallet analyze` 이후 interactive analyst chat/tool orchestration을 실제 surface로 연결
2. `production deployment readiness`
   - `corepack pnpm prod:*` 엔트리포인트, `/.env.production.example`, production runbook 세트를 기준으로 target environment preflight, replay, rollback, operator sign-off 최종 확인

참고:
- `WG-053 Pre-listing and Early Entry Engine`은 later backlog로만 유지한다. 현재 구현 우선순위는 아니다.

위 순서는 다음 세션에서도 기본 우선순위로 유지한다. 새로운 아이디어가 생겨도, 문서에 명시적으로 재정렬하기 전에는 이 순서를 깨지 않는다.
## AI Workspace

- AI analyst workstream is now split into `/Users/kh/Github/Qorvi/qorvi-ai`
- Use it for dataset specs, agent payload contracts, and evaluation assets
- Keep core ingestion, labeling, and scoring changes in the main product stack
- Initial completed specs:
  - `datasets/wallet-tracking-state.md`
  - `datasets/wallet-tracking-state-migration-outline.md`
  - `datasets/signal-outcomes.md`
  - `contracts/wallet-analyst.md`
  - `contracts/signal-explainer.md`
  - `contracts/alert-briefing.md`
  - `contracts/interactive-analyst.md`
  - `contracts/background-detection-agent.md`
  - `contracts/finding-object-schema.md`
- Implemented first product-side bridge:
  - migration `0014_wallet_tracking_state.sql`
  - DB store `packages/db/wallet_tracking_state.go`
  - search/watchlist/seed/backfill metadata now populate candidate discovery and tracked lifecycle state
  - admin observability includes wallet tracking + subscription overview snapshots
  - `packages/db/wallet_tracking_registry.go` and `apps/workers/tracking_subscription_sync.go` keep tracked-wallet subscription rows in sync and reconcile configured provider webhook address sets
  - webhook ingest updates `wallet_tracking_subscriptions.last_event_at` from real provider deliveries
  - migration `0015_wallet_labeling_baseline.sql` now adds `entity_labels`, `entity_label_memberships`, and `wallet_evidence`
  - `packages/providers/wallet_labeling.go` now derives baseline inferred labels for exchange / bridge / treasury / market-maker patterns and behavioral flow labels
  - `packages/db/wallet_labels.go` now persists evidence-first label memberships and invalidates summary/graph caches for affected wallets
  - wallet summary and wallet graph now expose separated `verified`, `inferred`, and `behavioral` labels on wallet/counterparty/node surfaces
