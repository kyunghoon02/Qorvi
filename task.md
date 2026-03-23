# WhaleGraph Task Backlog

이 문서는 [plan.md](/Users/kh/Github/WhaleGraph/plan.md)를 실제 실행 단위로 쪼갠 작업 백로그다.  
각 task는 바로 이슈나 스프린트 티켓으로 옮길 수 있도록 우선순위, 담당 subagent, 선행 조건, 산출물, 완료 기준을 포함한다.

## 1. 사용 규칙

1. task는 기본적으로 위에서 아래 순서로 수행한다.
2. 병렬 작업은 `Depends On`이 겹치지 않는 범위에서만 진행한다.
3. 모든 기능 task는 구현 전 API/schema contract를 먼저 확인한다.
4. 점수, 라벨링, 알림 관련 변경은 `ops-admin-engineer` 검토를 포함한다.
5. 배포 직전 task는 `billing-launch-engineer`의 release gate 체크 없이는 완료 처리하지 않는다.
6. 앞으로 product-facing task는 mock, fabricated preview data, local seed record를 새로 추가하지 않는다. 구현이 미완인 경우에도 실제 `loading`, `indexing`, `empty`, `unavailable`, `error` 상태를 정의하고 그 상태를 기준으로 작업한다.
7. beta open 준비는 local `.env`와 target deployment env를 분리해 관리한다. beta preflight는 target env file을 직접 검증할 수 있어야 한다.

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
  - 90일 기본 backfill worker
  - 1-hop 및 제한된 2-hop 확장 로직
  - service address stop rule
- Current State:
  - search miss, watchlist bootstrap, seed discovery enqueue가 source-aware backfill policy metadata를 queue job에 실어 보내도록 정리
  - worker가 `backfill_window_days`, `backfill_limit`, `backfill_expansion_depth`, `backfill_stop_service_addresses`를 읽어 실제 batch window/limit를 해석하는 baseline 추가
  - 기본 정책은 `search=90d/500/1-hop`, `watchlist|seed=90d/750/2-hop`, metadata override는 상한(`365d`, `1000 limit`, `2-hop`) 내에서만 허용
  - `watchlist_bootstrap` / `seed_discovery` source에서는 normalized transaction counterparties 상위 후보를 bounded fanout으로 `wallet_backfill_expansion` job에 재enqueue하는 baseline 추가
  - expansion job은 `backfill_root_*`, `backfill_parent_*`, `backfill_expansion_hits` metadata를 남기고, Redis dedup으로 중복 re-enqueue를 방지
  - search hit라도 `indexing.status=ready` 이고 `lastIndexedAt`이 stale threshold를 넘은 wallet은 `search_stale_refresh` source로 low-priority background refresh를 enqueue
  - wallet summary/detail에 `indexing status`, `coverage window`, `last indexed`를 노출하는 baseline 추가
  - home search surface와 wallet detail이 `indexing` 상태일 때 자동 polling으로 summary/graph를 다시 불러와 `ready` 상태로 자연스럽게 전환되도록 보강
  - home/detail copy를 `background indexing`, `coverage ready`, `updated just now / 5m ago` 형태로 정리해 기술 상태 노출을 줄이고 조사 흐름 중심으로 다듬음
  - `GET /v1/search?refresh=manual`을 통해 사용자가 명시적으로 stale/fresh wallet의 background refresh를 다시 enqueue할 수 있게 하고, home/detail UI에 `Refresh now` 액션을 추가
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
  - PRD 그래프 UX 방향을 WhaleGraph 고유의 `분석형 hub-and-spoke + partial flow hints`와 signal-first investigation view로 고정
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
  - 하지만 beta open에서는 Stripe를 blocker로 두지 않고, invite-only/free beta를 우선한다.
  - Stripe activation과 checkout closeout은 post-beta monetization track으로 다룬다.
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
  - `/Users/kh/Github/WhaleGraph/docs/runbooks/launch-gates.md`를 beta closeout source of truth로 확장해 `functional`, `reliability`, `UX`, `ops` gate와 `pass / warn / block` 상태를 현재 코드 기준으로 고정
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

beta를 가장 늦추는 경로는 아래다.

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

1. `beta open env unblock`
   - `beta:open:prep` blocker로 나온 runtime secret/env 값 채우기
2. `beta open`
   - target environment에서 `corepack pnpm beta:open:prep` 재실행
3. `operator sign-off`
   - admin/ops/billing handoff 최종 확인

위 순서는 다음 세션에서도 기본 우선순위로 유지한다. 새로운 아이디어가 생겨도, 문서에 명시적으로 재정렬하기 전에는 이 순서를 깨지 않는다.
