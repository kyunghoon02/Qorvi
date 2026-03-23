# WhaleGraph Development Plan

## 0. 진행 상태 범례

- `[O]` 완료
- `[~]` 진행 중
- `[ ]` 미착수

## 0.1 현재 진행 스냅샷

기준 시점: `2026-03-22`

- `[O]` monorepo bootstrap 완료: `apps/web`, `apps/api`, `apps/workers`, `packages/*`, `infra/*`
- `[O]` `pnpm workspace` + `go work` 기반 기본 lint/typecheck/test 흐름 완료
- `[O]` `.env.example`, runtime env validation, Clerk JWT/JWKS verification baseline 완료
- `[O]` local web dev boot path 정리: `next.config.mjs` + web `tsconfig` bundler resolution 기준으로 `pnpm dev:web` 실행 가능 상태로 정리
- `[O]` Postgres/Neo4j/Redis handles, migrations, local infra skeleton 완료
- `[O]` local compose Neo4j 포트 충돌 회피 정리: WhaleGraph local Neo4j는 `localhost:8687` 기준으로 고정
- `[O]` local dev stack one-command boot path 추가: `corepack pnpm dev:stack`으로 docker infra + api + worker loop + web 동시 실행 가능, `dev:stack:no-worker`와 `dev:worker:loop` 지원, macOS 기본 `bash 3.x`에서도 동작하도록 `wait -n` 비의존 경로로 정리, local compose 기준 `Postgres/Neo4j/Redis` 연결값을 스크립트가 강제로 맞춤
- `[O]` `wallet summary`, `wallet graph`, `search`, `wallet detail` 기본 vertical slice 완료
- `[O]` product preview cleanup baseline 완료: 앞으로 product-facing UI/API에는 fabricated mock preview data를 추가하지 않고, `loading / indexing / empty / unavailable / error` 상태와 실데이터 기반 degraded state(`summary-derived`)만 사용
- `[O]` wallet detail `graph UX` baseline 완료: edge-kind toggle, node/entity visual distinction, neighborhood summary API/read path, density guardrail, low-confidence dashed edge, Redis-backed precomputed neighborhood snapshot cache baseline, canonical `1-hop` graph의 `Redis -> Postgres snapshot -> live Neo4j` read order, focal wallet 중심 `hub-and-spoke + partial flow hints`, React Flow 기반 pan/zoom/minimap/node selection, bounded `Expand 2-hop`, selected node/relationship inspector, entity-linked source badge와 token flow/evidence 패널까지 연결
- `[O]` home product surface baseline 완료: search bar slim entry, graph-first home layout, compact summary rail, indexing polling, `Refresh now`, summary-derived degraded graph 유지, 그리고 home/detail/search가 `live / summary-derived / unavailable` 상태와 entity fallback/source badge를 같은 shared presenter 규칙으로 읽도록 정리
- `[O]` home search input editability fix: URL `q` hydrate 이후 입력 중에는 로컬 상태를 유지하도록 정리해 검색창 값이 잠기거나 덮어써지지 않게 수정
- `[O]` home search URL privacy boundary: public wallet-address search만 `?q=`에 유지하고 ENS-like/unknown query는 로컬 상태로만 유지하도록 정리
- `[~]` graph interaction polish 진행 중: graph selected-node panel에 `wallet/cluster` CTA 추가, node double-click으로 detail 이동 가능, home/detail 화면 모두에서 visible relationship list를 노출해 live graph와 summary-derived fallback을 더 직접적으로 구분 가능
- `[O]` graph relation taxonomy baseline 완료: 기존 `INTERACTED_WITH` 단일 관찰 관계에 더해 inbound funding을 `FUNDED_BY` derived edge로 materialize/read하고, base fact와 derived inference를 family filter로 분리해 사실/추론 관계를 따로 읽을 수 있게 정리, edge-level `evidence`와 `tokenFlow`를 `Neo4j metadata + Postgres aggregate` 조합으로 내려 관계 근거와 주요 토큰 흐름을 바로 해석 가능하게 보강, `INTERACTED_WITH` edge의 inbound/outbound directional aggregate를 materialization/read path와 동일 계약(`directionality`)으로 통일, provider metadata(`helius_source`/`helius_identity_source`) 및 known address catalog 기반 heuristic entity assignment를 worker ingest와 graph/detail source badge에 연결, curated/heuristic entity assignment 변경 시 canonical graph cache/`wallet_graph_snapshots` invalidation까지 연결
- `[~]` wallet detail `related addresses` surface 진행 중: summary 기반 `top counterparties`/`recent flow` 카드 노출 완료, direction/inbound/outbound/first-seen baseline 추가 완료, direction filter + sortable related-address table baseline 완료, token filter + volume sort baseline 완료, row-expand token breakdown baseline 완료, `copy address`/`copy summary`/`open in search`/`focus in graph` 조사 액션 baseline 완료, graph neighborhood에 `entity_linked` assignment가 있는 row는 entity label + source badge를 함께 노출하도록 보강, summary aggregate도 counterparty entity label/type/key를 직접 내려 summary-derived fallback graph와 related-address table에서 heuristic/curated entity가 유지되도록 보강, investigation-grade filtering과 operator review action은 계속 고도화 필요
- `[~]` related-address amount visibility 진행 중: `transactions`에 amount/token persistence baseline 추가, summary counterparty aggregate에 `inbound/outbound amount + primary token + token breakdown` read path 반영, wallet detail table에 amount column 노출 완료
- `[~]` wallet detail UI cleanup 진행 중: hero/summary/graph 패널에서 기술용 route 텍스트 제거, compact graph variant 적용, long address block과 map metric strip으로 정보 위계 정리 완료, `indexing status / coverage window / last indexed` 카드 baseline 추가, `indexing -> ready` 자동 polling/revalidate baseline 추가, `background indexing / just now / 5m ago` 형태의 live copy 정리, hero action 영역 `Refresh now`로 manual refresh enqueue + summary/graph reload baseline 완료, `Uncodixfy.md` 기준으로 detail/home 공통 CSS를 flat surface + tighter spacing + lower emphasis label 체계로 재정리
- `[O]` Alchemy/Helius historical ingest scaffold, raw payload 저장, usage/job logging 완료
- `[O]` Alchemy historical backfill request serialization 정리: `maxCount`를 hex quantity로 전송하도록 수정해 local lazy indexing worker에서 `Invalid hex string` 오류 제거
- `[~]` Alchemy historical backfill coverage 강화: `ToAddress` 단방향 fetch에서 `FromAddress + ToAddress` 양방향 fetch + transfer dedup baseline 완료, Solana standard RPC fallback(`getSignaturesForAddress` + `getTransaction`) 경로와 native/token delta 기반 counterparty 추론 baseline 추가
- `[O]` lazy indexing worker nil-amount crash fix: Alchemy nil transfer value가 `"<nil>"`로 저장돼 `amount_numeric` 캐스팅을 깨뜨리던 경로를 provider/domain/db sanitize로 차단
- `[O]` ingest dedup retry-poisoning fix: historical backfill worker와 webhook ingest가 write/materialization 실패 시 claimed dedup key를 release하도록 보강
- `[~]` Helius historical enrichment coverage 강화: Data API parse 결과에서 `direction/counterparty/amount/token/funder/identity` metadata를 추출해 Solana backfill이 summary/graph read model에 바로 기여하도록 baseline 보강 완료, paid-plan 403일 때는 client fallback + runner fallback 이중 안전장치로 `Alchemy Solana RPC fallback` historical ingest를 계속 진행하도록 보강
- `[O]` Helius fallback observability/URL normalization fix: paid-plan 403 이후 `Alchemy Solana` fallback이 실패해도 원래 Helius 403만 남기지 않고 실제 fallback failure를 surface하도록 수정, `.env`에 `HELIUS_BASE_URL`/`HELIUS_DATA_API_BASE_URL`를 full RPC URL 형태로 넣어도 canonical base URL로 정규화되도록 보강
- `[O]` local Solana fallback env alignment: local `.env`에 `ALCHEMY_SOLANA_BASE_URL`를 명시해 Solana historical fallback이 EVM RPC host와 분리된 Alchemy Solana RPC host를 직접 사용하도록 정리
- `[O]` provider timeout/worker visibility fix: provider HTTP client 기본 timeout을 추가하고 worker가 wallet backfill 시작 로그를 남기도록 보강해 Solana fallback이 오래 대기할 때도 멈춘 것처럼 보이지 않게 정리
- `[O]` Helius paid-plan fallback 정리: `parseTransactions` 403이 나도 enrichment만 건너뛰고 wallet history ingest 자체는 계속 진행되도록 보강
- `[~]` Moralis wallet enrichment baseline 진행 중: `wallets/:address/net-worth` + `wallets/:address/chains` + `wallets/:address/tokens` 기반 optional client, Redis cache-first on-demand policy, wallet summary/detail optional overlay baseline 완료, `net-worth`/`chains`/`holdings` endpoint 중 일부만 실패해도 partial enrichment를 계속 반환하도록 보강, wallet detail에는 `updated/source/active chains` compact polish와 `top holdings` surface 반영, 별도 `moralis-enrichment-refresh` worker mode baseline 추가
- `[O]` Alchemy/Helius webhook ingest와 Neo4j transaction graph materialization baseline 완료
- `[~]` 제품용 2차 인덱스 구축 진행 중: `wallet interaction graph`, `summary aggregate`, `top counterparties`, `recent flow`, `cluster score snapshot` read path 구현
- `[O]` `cluster_score` snapshot worker와 `GET /clusters/:id` + cluster detail UI baseline 완료
- `[O]` `shadow_exit_snapshot` worker, signal snapshot read path, wallet summary 반영 baseline 완료
- `[O]` `GET /signals/shadow-exits` API와 `shadow exit feed` UI baseline 완료
- `[~]` `first_connection_snapshot` worker와 wallet summary `alpha_score` snapshot read path baseline 진행 중
- `[O]` `GET /signals/first-connections` API와 `first connection hot feed` UI baseline 완료
- `[~]` lazy indexing + watchlist 선인덱싱 orchestrator 진행 중
- `[~]` historical backfill worker policy 진행 중: search/watchlist/seed source별 queue metadata 정책(`90일 window`, `limit`, `expansion_depth`, `stop-service-addresses`)을 태우고 worker가 이를 실제 batch window/limit로 해석하도록 baseline 보강 완료, watchlist/seed source는 top-counterparty 기준 bounded `2-hop` re-enqueue baseline까지 구현, wallet summary/detail에는 `indexing in progress / coverage window / last indexed` 상태 노출 baseline 추가, 검색 hit라도 `ready` 상태의 `lastIndexedAt`이 오래된 wallet은 low-priority stale refresh를 background enqueue하도록 보강, `GET /v1/search?refresh=manual` 기반 manual refresh enqueue baseline 추가
- `[~]` protected watchlist CRUD API, item tags/notes, tier-based limits, Postgres persistence baseline 완료
- `[~]` `alert_rules` protected CRUD, owner-scoped `alert_events` audit trail, dedup/cooldown/re-notify, worker `signal snapshot -> alert rule evaluation` baseline 완료
- `[x]` `GET /alerts` inbox API, `alert-delivery-channels` CRUD API, worker `email/discord/telegram` delivery attempt + retry batch baseline 완료, `/alerts` alert center UI baseline 완료, unread state/cursor pagination/snooze state baseline 완료, explicit mute action과 in-product mutation UX 완료
- `[x]` protected admin labels CRUD API, suppressions CRUD API, provider quota snapshot API, curated list CRUD/read API, admin audit log read API, `/admin` admin console baseline 완료, human override action과 deeper quota dashboard 완료
- `[O]` `WG-040 Observability Dashboard` 완료: `/v1/admin/observability` snapshot route 추가, `provider usage / ingest freshness / alert delivery / recent runs / recent failures` aggregate를 기존 admin console에 compact operator panel로 연결
- `[O]` `GET /v1/account` / `GET /v1/account/entitlements` baseline과 `/account` billing surface 완료, `apps/api` checkout session endpoint와 billing webhook endpoint baseline 완료, `GET /v1/billing/plans` pricing catalog baseline 완료, `/pricing` plan cards와 live checkout mutation baseline 완료, Postgres/in-memory billing account persistence와 webhook 기반 plan reconciliation baseline 완료, real Stripe HTTP client, checkout session/subscription persistence, webhook reconciliation audit, `billing-subscription-sync` worker baseline까지 완료
- `[~]` local API dev는 example Clerk config일 때 legacy header verifier로 fallback 가능, Redis와 DB는 여전히 로컬 infra 필요하며 현재는 실행 전에 `.env` export가 필요
- `[~]` label/entity indexing pipeline: admin curated lists를 source로 `entities + wallets.entity_key`를 재구성하는 baseline sync 추가, graph read model이 `entity_linked` node/edge를 실제 display label 기준으로 노출 가능, provider/heuristic baseline으로 Helius metadata source를 `heuristic:<chain>:<slug>` entity namespace에 매핑해 counterparty/funder wallet assignment를 자동 upsert하도록 보강, explicit provider source가 비어도 known address catalog와 metadata label/service pattern, source alias normalization으로 entity assignment를 보완, 기존 `curated:*`와 기타 non-heuristic mapping은 ingest/backfill heuristic assignment가 덮어쓰지 않도록 보존
- `[x]` serving aggregate materializers: `wallet_daily_stats` cumulative snapshot refresh baseline과 summary daily-snapshot 우선 read path 추가, `top counterparties`, `recent flow`, `cluster score snapshot` read path 반영 완료, Moralis enrichment overlay를 통해 `holdings/balance` surface를 summary/detail contract에 연결한 baseline 완료, `wallet_enrichment_snapshots` durable snapshot baseline을 추가해 summary repo가 Redis cold-start에도 Postgres snapshot으로 `holdings/balance`를 읽을 수 있도록 정리, wallet summary/detail에 `latest signals` surface baseline 추가, `signal_events` 기반 materialized latest signals read model을 summary cache/input path에 연결해 API repository가 worker snapshot 최신 row를 우선 사용하고 없을 때만 score evidence fallback을 쓰도록 정리, EVM historical backfill 성공 후 Moralis cache를 prewarm하는 `async enrichment refresh` baseline과 별도 `moralis-enrichment-refresh` worker mode baseline 추가, live Moralis fetch가 발생하면 Redis cache와 Postgres enrichment snapshot을 함께 갱신하도록 보강, Alchemy/Helius webhook live ingest도 `wallet_daily_stats` refresh를 함께 수행하도록 정리해 live path summary aggregate freshness를 historical backfill path와 맞춤, backfill/webhook/signal snapshot/Moralis live refresh 경로가 모두 wallet summary cache invalidation을 호출하도록 마감
- `[~]` watchlist/alerts/admin/billing beta 완성
- `[O]` `WG-041 Replay and Contract Test Suite` 완료: provider fixture corpus(`Alchemy`, `Helius`, `Moralis`)와 raw webhook replay fixture(`Alchemy`, `Helius`) 추가, replayed raw payload가 동일 normalized transaction set과 동일 `cluster/shadow-exit/first-connection` score summary를 재현하는 contract test baseline 완료
- `[O]` `WG-043 Beta Launch Checklist` 완료: `docs/runbooks/launch-gates.md`를 functional/reliability/UX/ops gate, evidence bundle, rollback/recovery package를 포함한 beta closeout source of truth로 확장했고, current gate status를 `pass/warn/block`으로 정리해 `block` 없는 beta go 상태로 고정
- `[O]` beta hardening/release packaging baseline 완료: `corepack pnpm beta:hardening` one-click evidence entrypoint와 `/docs/runbooks/beta-release-package.md`, `/docs/runbooks/beta-operator-handoff.md` launch handoff 문서 추가
- `[O]` launch gate review/beta open prep baseline 완료: `/docs/runbooks/beta-launch-review.md`에 current `go` decision과 evidence 결과를 고정했고, `/docs/runbooks/beta-open-prep.md`와 `corepack pnpm beta:open:prep`로 환경/운영 최종 확인을 one-click로 정리
- `[~]` beta open prep 현재 block: local runtime env 기준 `corepack pnpm beta:open:prep`가 `WHALEGRAPH_RAW_PAYLOAD_ROOT`, Clerk secret/publishable key, Stripe keys/URLs, Moralis key/base URL의 placeholder/누락을 보고해 실제 open-prep는 아직 미통과

## 0.2 현재 실행 순서

앞으로는 아래 순서대로만 진행한다. 중간에 새로운 아이디어가 생겨도, 명시적으로 우선순위를 바꾸지 않는 한 이 순서를 유지한다.

1. `beta open env unblock`
   - 목표: `corepack pnpm beta:open:prep`가 보고한 runtime env blocker 해소
2. `beta open`
   - 목표: target environment에서 `corepack pnpm beta:open:prep` 재실행 후 최종 open decision 고정
3. `post-beta backlog reset`
   - 목표: beta 이후 product/data roadmap 재정렬

## 0.3 현재 Immediate Next

바로 다음 구현 우선순위는 아래 3개로 고정한다.

1. `beta open env unblock`: `beta:open:prep` blocker로 나온 runtime secret/env 값 채우기
2. `beta open`: target environment에서 `corepack pnpm beta:open:prep` 재실행
3. `operator sign-off`: admin/ops/billing handoff 최종 확인

`WG-043`는 완료 처리했고, 다음부터는 launch checklist에서 남긴 `warn`와 beta open decision 정리만 수행한다.

## 1. 문서 목적

이 문서는 [WhaleGraph_PRD_v2_0.md](/Users/kh/Github/WhaleGraph/WhaleGraph_PRD_v2_0.md)를 실제 개발 가능한 작업 단위로 분해한 실행 계획서다.  
목표는 `public beta` 오픈에 필요한 범위를 명확히 고정하고, 각 단계별 산출물과 완료 기준을 정의하는 것이다.

WhaleGraph는 단순한 지갑 조회 앱이 아니라 다음 3가지 인텔리전스 엔진이 중심인 멀티체인 SaaS로 구현한다.

1. `Coordinated Whale Cluster Engine`
2. `Shadow Exit Detection Engine`
3. `First-Connection Discovery Engine`

---

## 2. 개발 목표

### 2.1 Beta 목표

beta 시점에 반드시 가능한 상태는 다음과 같다.

1. EVM/Solana 주소 검색 시 `wallet summary`와 기본 관계 그래프를 제공한다.
2. seed whale 세트를 주기적으로 갱신하고 watchlist 기반 실시간 감시를 수행한다.
3. `cluster_score`, `shadow_exit_risk`, `alpha_score`를 계산하고 evidence와 함께 노출한다.
4. `watchlist`, `alerts`, `admin console`, `suppression`, `query budget monitor`를 운영 가능 상태로 제공한다.
5. Free/Pro 최소 2개 플랜과 구독 흐름을 제공한다.

### 2.2 Beta 비목표

1. 체인 전체 무제한 인덱싱
2. 법적 포렌식 수준의 attribution
3. 자동매매, 주문 실행, 포트폴리오 관리
4. Base/BNB/Sui 등 추가 체인 확장

---

## 3. 제품 범위 고정

### 3.1 Beta 필수 기능

| 영역 | Beta 포함 범위 |
| --- | --- |
| Search | EVM address, Solana address, token, entity 검색 |
| Wallet Intelligence | summary, counterparties, timeline, cluster candidates, latest signals |
| Graph | 기본 1-hop, Pro는 2-hop, progressive rendering |
| Cluster | strong/weak/emerging cluster 계산 및 설명 카드 |
| Shadow Exit | fan-out, CEX proximity, bridge escape 기반 위험 점수 |
| First-Connection | 24시간 내 신규 공통 진입 탐지 및 hot feed |
| Alerts | in-app, email, Telegram, Discord webhook |
| Watchlist | wallet, cluster, token, entity 저장 및 태그 관리 |
| Admin | label editor, suppression, curated list, quota monitor |
| API | PRD 명시 5개 핵심 endpoint + alerts |
| Billing | Free, Pro 최소 플랜, 구독 상태에 따른 rate/feature gating |

### 3.2 Beta 이후로 미루는 기능

1. AI 자연어 설명 자동 생성 고도화
2. 팀 협업 조사 리포트
3. export bulk job
4. 추가 유료 라벨 데이터셋 연동
5. advanced PnL analytics

---

## 4. 핵심 개발 원칙

### 4.1 Architecture Principles

1. `budget-aware`: 무료 티어 기준에서도 운영 가능한 호출 구조를 우선한다.
2. `evidence-first`: 모든 점수와 알림은 증거 객체를 함께 저장하고 응답한다.
3. `raw-first`: webhook/event는 분석 전에 원본 저장부터 수행한다.
4. `idempotent-by-default`: ingest, scoring, alert delivery는 모두 중복 안전해야 한다.
5. `progressive UX`: 빠른 summary 응답 후 상세 분석은 비동기로 확장한다.

### 4.2 Delivery Principles

1. 데이터 수집기와 제품 UI를 병렬 개발 가능하도록 API contract를 먼저 고정한다.
2. 점수 엔진은 규칙 기반 MVP로 시작하고, 운영 데이터 축적 후 가중치를 조정한다.
3. false positive 제어 도구를 초기에 넣는다. 운영 override는 후순위가 아니다.
4. observability, quota tracking, audit log는 beta 이전 필수 범위로 간주한다.
5. `real-state-first`: 앞으로 product-facing UI와 API는 mock, fabricated preview, local seed record를 새로 추가하지 않는다. 구현이 덜 끝난 기능도 실제 `loading`, `indexing`, `empty`, `unavailable`, `error` 상태로 노출하고, degraded path가 필요하면 실데이터 기반 read model에서만 파생한다.

---

## 5. 제안 기술 구조

현재 저장소에는 foundation과 첫 번째 product vertical slice가 구현돼 있으므로, 아래 구조를 `기준 아키텍처`로 유지하면서 남은 beta 범위를 확장한다.

### 5.1 Monorepo 구조

```text
apps/
  web/              # public web app + authenticated product UI
  api/              # Go public/internal API service
  workers/          # Go batch, scoring, alert, backfill jobs
packages/
  domain/           # Go entity, scoring contracts, validation schema
  providers/        # Go Dune, Alchemy, Helius, Moralis adapters
  db/               # Go postgres/neo4j/redis access layers
  config/           # Go env, feature flags, plan gates
  intelligence/     # Go cluster/shadow/first-connection scoring
  ops/              # Go suppression, labeling, quota, audit helpers
  billing/          # Go entitlement and launch gate helpers
  ui/               # shared UI components for Next.js
infra/
  docker/           # local postgres, neo4j, redis
  migrations/       # database and graph bootstrap
docs/
  runbooks/         # provider quota, incident, labeling policy
```

### 5.2 권장 스택

| 영역 | 권장안 | 이유 |
| --- | --- | --- |
| Frontend | Next.js + TypeScript | 검색, SEO landing, auth UI, API integration에 적합 |
| Backend services | Go | 서비스와 라이브러리를 단일 언어로 유지 |
| Backend libraries | Go | 도메인, provider, DB helper 공유가 쉬움 |
| Queue | Redis 기반 job queue | dedup, cooldown, retries를 단순하게 구현 가능 |
| RDBMS | PostgreSQL | 제품 도메인, alert, billing, audit 저장 |
| Graph | Neo4j | cluster, path, neighborhood query 최적화 |
| Cache | Redis | hot cache, dedup key, rate limit |
| Object Storage | S3/R2 호환 스토리지 | raw payload, snapshots, audit artifacts 저장 |
| ORM/Query | Go SQL + migration | schema 관리 및 빠른 개발 |
| Validation | Go structs + JSON Schema | provider adapter와 API contract의 경계 통제 |
| Auth | Clerk JWT/JWKS verification | 빠른 SaaS 인증/권한 구현 |
| Billing | Stripe | 구독 플랜, webhook, entitlements 구현 용이 |

### 5.3 환경 분리

1. `local`: docker 기반 Postgres/Neo4j/Redis
2. `staging`: beta 직전 실제 provider와 연결되는 통합 환경
3. `production`: quota 모니터링, alert delivery, billing webhook 분리 운영

### 5.4 Sprint 0 확정 결정

Sprint 0에서는 기술 검토를 계속 열어두지 않고 아래를 기준선으로 고정한다.

1. web workspace: `pnpm` + `turbo`
2. web app: Next.js App Router + TypeScript
3. backend workspace: Go modules + `go work`
4. backend services/libraries: Go
5. data access: Go SQL + migrations
6. auth: Clerk JWT/JWKS verification
7. billing: Stripe

위 결정의 목적은 최종 기술 정답을 찾는 것이 아니라 `WG-003` 이후 바로 `wallet summary` 수직 슬라이스 구현으로 넘어가기 위한 개발 속도 확보에 있다.

---

## 6. 핵심 시스템 설계 방향

### 6.1 상위 컴포넌트

1. `Collector Service`
2. `Normalization Service`
3. `Enrichment Service`
4. `Graph Service`
5. `Scoring Service`
6. `Alert Service`
7. `API Service`
8. `Web Frontend`
9. `Admin Console`

### 6.2 실시간 처리 순서

1. raw payload 저장
2. idempotency key 계산 및 dedup
3. chain-specific normalization
4. enrichment 및 label 연결
5. graph edge upsert
6. score recompute enqueue
7. alert rule evaluation
8. delivery 및 audit log 저장

### 6.3 조회 전략

1. summary API는 Postgres cache/materialized view 우선 조회
2. graph query는 Neo4j neighborhood snapshot 우선 사용
3. detail panel은 lazy fetch
4. provider live fetch는 cache miss와 user-demand 상황에서만 수행

---

## 7. 데이터 모델 구현 계획

### 7.1 Postgres 우선 테이블

초기 1차 마이그레이션에 아래 테이블을 포함한다.

1. `users`
2. `organizations`
3. `subscriptions`
4. `wallets`
5. `tokens`
6. `entities`
7. `wallet_labels`
8. `transactions`
9. `wallet_daily_stats`
10. `clusters`
11. `cluster_members`
12. `signal_events`
13. `alert_rules`
14. `alert_events`
15. `watchlists`
16. `watchlist_items`
17. `provider_usage_logs`
18. `suppression_rules`
19. `audit_logs`
20. `job_runs`

### 7.2 Neo4j 스키마

아래 노드/관계를 beta 범위에 맞춰 고정한다.

**Nodes**

1. `Wallet`
2. `Token`
3. `Entity`
4. `Cluster`

**Relationships**

1. `TRANSFERRED_TO`
2. `FUNDED_BY`
3. `INTERACTED_WITH`
4. `DEPOSITED_TO`
5. `BRIDGED_TO`
6. `MEMBER_OF`
7. `ACQUIRED`

### 7.3 저장 우선순위

1. 원본 이벤트 저장
2. 정규화 이벤트 저장
3. 통계/스냅샷 저장
4. 그래프 materialization
5. 점수 결과 캐시

이 순서를 지키면 provider 응답 포맷이 변해도 재처리가 가능하다.

---

## 8. Workstreams

개발은 8개의 병렬 워크스트림으로 운영한다.

| 상태 | 스트림 | 목적 | 선행 의존성 |
| --- | --- | --- | --- |
| `[O]` | WS1 Foundation | monorepo, CI, env, auth, design system | 없음 |
| `[~]` | WS2 Data Platform | Postgres/Neo4j/Redis/Object Storage 연결 | WS1 |
| `[~]` | WS3 Provider Adapters | Dune/Alchemy/Helius/Moralis adapter | WS1, WS2 |
| `[~]` | WS4 Intelligence Engine | cluster/shadow/first-connection 계산 | WS2, WS3 |
| `[~]` | WS5 API Layer | public/internal API, rate limits, entitlements | WS1, WS2, WS4 |
| `[~]` | WS6 Product UI | search, wallet, cluster, feed, alert center | WS1, WS5 |
| `[~]` | WS7 Ops & Admin | labeling, suppression, curated lists, quota monitor | WS2, WS5 |
| `[~]` | WS8 Billing & Launch | Stripe, plans, launch checklist, observability | WS1, WS5 |

---

## 9. 단계별 실행 계획

아래 일정은 `10주 기준 beta 플랜`이다.  
팀 규모는 최소 `2 backend + 1 frontend + 1 fullstack/operator`를 가정한다.

## [O] Phase 0. Inception & Scope Freeze

**기간:** 3~4일

**목표**

1. beta 범위 확정
2. provider quota 가정치 수립
3. 도메인 용어와 API contract 초안 확정

**주요 작업**

1. PRD 기능을 `Must / Should / Later`로 재분류
2. 체인별 데이터 소스별 호출 단가와 free tier 한도 문서화
3. score/evidence JSON schema 초안 작성
4. alert severity taxonomy 정의
5. UX route map 및 페이지 IA 확정

**산출물**

1. 범위 고정 문서
2. provider budget sheet
3. API contract v0
4. scoring/evidence schema v0

**완료 기준**

1. beta에 넣을 endpoint, page, score 종류가 합의됨
2. 각 provider에 대해 호출 예산과 fallback 전략이 정의됨

---

## [O] Phase 1. Repository Foundation

**기간:** 1주

**목표**

제품 개발이 가능한 기본 저장소와 공통 개발 환경을 만든다.

**주요 작업**

1. monorepo bootstrap
2. lint, format, typecheck, test workflow 구성
3. 환경변수 로더 및 secret validation 구현
4. 공통 domain package와 schema package 생성
5. local docker compose로 Postgres/Neo4j/Redis 기동
6. Clerk issuer/JWKS/env 설정을 추가하고 auth skeleton 및 RBAC role 정의

**산출물**

1. `apps/web`, `apps/api`, `apps/workers` 초기 구조
2. 공통 logger, config, error envelope
3. CI pipeline 초안
4. 로컬 개발 가이드

**완료 기준**

1. 신규 개발자가 저장소 clone 후 로컬 부팅 가능
2. typecheck/lint/test 기본 파이프라인 통과
3. role: `user`, `pro`, `admin`, `operator` 구분 가능

---

## [~] Phase 2. Data Platform & Schemas

**기간:** 1주

**목표**

WhaleGraph의 핵심 저장 구조와 재처리 가능한 ingest 골격을 만든다.

**주요 작업**

1. Postgres 1차 스키마 마이그레이션 작성
2. Neo4j bootstrap 및 인덱스/constraint 설정
3. raw event object storage writer 구현
4. idempotency key 전략 설계
5. transaction normalization schema 정의
6. wallet/token/entity canonical key 규칙 확정

**산출물**

1. DB migration 세트
2. graph schema bootstrap 스크립트
3. raw payload writer
4. normalized transaction schema v1

**완료 기준**

1. EVM/Solana 이벤트를 공통 transaction schema로 적재 가능
2. 동일 이벤트 중복 수신 시 중복 저장되지 않음
3. graph 관계를 wallet 단위로 upsert 가능

**현재 상태**

- `[O]` Postgres/Neo4j/Redis 연결 및 기본 migration
- `[O]` raw payload 저장
- `[O]` normalized transaction schema와 graph materialization baseline
- `[~]` idempotency / dedup framework
- `[ ]` object storage production 경로

---

## [~] Phase 3. Provider Adapter Layer

**기간:** 1.5주

**목표**

체인별 데이터를 budget-aware하게 수집하는 adapter 계층을 완성한다.

**주요 작업**

1. Dune seed discovery export adapter
2. Alchemy historical transfer + realtime webhook adapter
3. Helius wallet history / transfers / funded-by / identity adapter
4. Moralis enrichment adapter
5. provider response schema validation
6. retry/backoff/circuit breaker 전략 적용
7. usage metering 및 quota logging 구현

**산출물**

1. provider clients
2. contract fixtures and unavailable-state tests
3. usage logging pipeline
4. provider health dashboard 최소 버전

**완료 기준**

1. seed wallet batch, on-demand backfill, realtime watchlist ingest가 각각 동작
2. provider 응답 포맷 변경 시 adapter layer에서만 수정 가능
3. provider별 일별 호출량과 실패율 확인 가능

**현재 상태**

- `[O]` Alchemy historical transfers scaffold
- `[O]` Helius history/data-api scaffold
- `[O]` usage metering 및 quota logging baseline
- `[O]` Alchemy/Helius webhook ingest baseline
- `[~]` Dune seed discovery adapter env/file row-injection baseline
- `[~]` Moralis enrichment adapter baseline
- `[ ]` retry/backoff/circuit breaker 고도화

---

## [~] Phase 4. Seed Discovery & Historical Backfill

**기간:** 1주

**목표**

seed whale 생성과 초기 데이터 적재를 자동화한다.

**주요 작업**

1. Dune 결과 기반 seed candidate scoring
2. 상위 후보의 watchlist 편입 규칙 구현
3. 주소별 90일 기본 backfill 작업기 구현
4. 1-hop 확장 및 상위 N counterparty 정책 적용
5. 2-hop 확장 제한 로직 구현
6. 서비스/거래소/브릿지 주소에 대한 탐색 중단점 구현

**산출물**

1. seed discovery batch
2. watchlist seeding worker
3. backfill worker
4. counterparty aggregation materialized data

**완료 기준**

1. seed whale 리스트가 배치로 생성 및 갱신됨
2. 주소 검색 시 이미 backfill된 경우 즉시 summary 계산 가능
3. budget 초과 없이 90일 기본 데이터 적재 정책이 유지됨

**현재 상태**

- `[O]` historical backfill worker baseline
- `[O]` webhook -> normalized transaction -> graph 반영 baseline
- `[~]` seed discovery batch env/file row-parser + top-N watchlist seeding baseline
- `[~]` watchlist seeding worker/bootstrap enqueue
- `[~]` wallet backfill queue drain + batch drain
- `[~]` lazy indexing orchestrator
- `[~]` 90일 기본 backfill 정책과 1-hop/2-hop 확장 제어

---

## [~] Phase 5. Wallet Intelligence MVP

**기간:** 1주

**목표**

가장 먼저 사용자 가치가 드러나는 `Wallet Intelligence Profile`을 완성한다.

**주요 작업**

1. `/wallet/:chain/:address/summary` API 구현
2. `/wallet/:chain/:address/graph` API 구현
3. wallet summary materializer 작성
4. top counterparties, netflow, holdings, latest signals 집계
5. async enrichment refresh job 구현
6. 신규/휴면/unknown address fallback 처리

**산출물**

1. wallet summary API
2. wallet graph API
3. wallet detail query service
4. search-to-summary UX 흐름

**완료 기준**

1. p95 3초 이내 summary 응답
2. unknown/new wallet도 빈 상태가 아닌 설명 가능한 결과 반환
3. graph와 counterparties 데이터가 동일한 기간 기준으로 정렬됨

**현재 상태**

- `[O]` `GET /wallet/:chain/:address/summary`
- `[O]` `GET /wallet/:chain/:address/graph`
- `[O]` search-to-wallet flow
- `[O]` wallet detail UI baseline
- `[~]` summary aggregate를 transactions 기반으로 반영
- `[~]` top counterparties / recent flow 반영
- `[~]` cluster score snapshot read path 반영
- `[~]` holdings/latest signals aggregate
- `[~]` async enrichment refresh job

---

## [~] Phase 6. Cluster Engine

**기간:** 1.5주

**목표**

고래 간 동조 그룹을 점수화하고 시각화 가능한 상태로 만든다.

**주요 작업**

1. `same_funder`, `co_trading`, `shared_counterparties`, `cex_pattern`, `temporal_sync`, `bridge_similarity` 계산기 작성
2. `cluster_score` 공식 구현
3. strong/weak/emerging classification 구현
4. cluster materialization 및 member relationship upsert
5. reason summary/evidence formatter 작성
6. cluster change diff 저장
7. false positive 감점 룰과 suppression 연결

**산출물**

1. cluster scoring worker
2. `/cluster/:id` API
3. cluster explanation payload
4. cluster snapshot 테이블

**완료 기준**

1. 동일 seed set에 대해 재계산 시 결정론적으로 비슷한 결과가 나옴
2. cluster response가 evidence 없이 점수만 반환하지 않음
3. weak/strong threshold가 운영자가 조정 가능

**현재 상태**

- `[~]` cluster signal calculators 일부 구현: `shared counterparties`, `interaction persistence`
- `[O]` `cluster_score` snapshot worker
- `[O]` strong/weak/emerging classification baseline
- `[O]` `GET /cluster/:id` API와 cluster detail UI baseline
- `[ ]` cluster materialization / diff 저장
- `[ ]` suppression/false positive 감점 룰 연결

---

## [~] Phase 7. Shadow Exit Engine

**기간:** 1주

**목표**

fan-out 기반 유출 위험을 조기에 포착하는 엔진을 구현한다.

**주요 작업**

1. 24시간 fan-out 후보 생성 로직
2. 신규 하위 지갑 감지 로직
3. CEX proximity, bridge escape, outflow ratio 계산
4. treasury rebalancing whitelist 감점 로직
5. risk timeline 생성
6. signal event 저장 및 severity 계산

**산출물**

1. shadow exit scoring worker
2. `/signals/shadow-exits` API
3. risk evidence timeline formatter

**완료 기준**

1. 동일 소스 지갑 fan-out이 alert candidate로 포착됨
2. 내부 리밸런싱으로 추정되는 경우 risk가 하향 보정됨
3. alert 문구가 “매도 확정”처럼 단정적 표현을 쓰지 않음

**현재 상태**

- `[O]` `shadow_exit_snapshot` worker baseline
- `[O]` `signal_events` 기준 latest shadow exit snapshot read path
- `[O]` wallet summary의 `shadow_exit_risk` snapshot override baseline
- `[O]` `/signals/shadow-exits` API baseline
- `[O]` shadow exit feed page와 API boundary contract baseline
- `[O]` 24시간 fan-out 후보 생성 로직
- `[O]` treasury whitelist discount
- `[O]` shadow exit feed UI

---

## [~] Phase 8. First-Connection Engine

**기간:** 1주

**목표**

고래들의 신규 공통 진입 이벤트를 hot feed로 제공한다.

**주요 작업**

1. 90일 무이력 판정기 구현
2. 24시간 내 2개 이상 seed/cluster member 동시 진입 감지
3. `alpha_score` 계산기 구현
4. token/protocol 단위 event grouping
5. 과거 유사 사례 링크용 저장 구조 추가
6. hot feed 정렬 및 최신순/점수순 응답

**산출물**

1. first-connection worker
2. `/signals/first-connections` API
3. hot feed cache

**완료 기준**

1. 동일 토큰에 대한 중복 이벤트가 병합됨
2. novelty와 whale quality가 분리된 evidence로 설명 가능
3. hot feed에서 최신순과 점수순 둘 다 지원됨

**현재 상태**

- `[O]` first-connection score contract와 `alpha_score` baseline
- `[O]` `first_connection_snapshot` worker baseline
- `[O]` `signal_events` 기준 latest first-connection snapshot read path
- `[O]` wallet summary의 `alpha_score` snapshot override baseline
- `[~]` 90일 novelty 판정과 24시간 공통 진입 detector baseline
- `[O]` `/signals/first-connections` API와 hot feed UI baseline
- `[O]` hot feed cache 및 explicit 최신순/점수순 정렬 옵션 baseline

---

## Phase 9. Alerts, Watchlists, Admin

**기간:** 1.5주

**목표**

제품 운영성과 사용자 반복 사용성을 만드는 기능을 완성한다.

**주요 작업**

1. watchlist CRUD
2. alert rule CRUD
3. dedup/cooldown/re-notify 로직 구현
4. in-app inbox 구현
5. email/Telegram/Discord channel delivery 구현
6. curated list 편집 UI
7. label editor, suppression list, human override 구현
8. quota monitor 및 provider usage dashboard 구현

**산출물**

1. alert engine
2. alert center UI
3. admin console MVP
4. operator audit logs

**완료 기준**

1. 같은 이벤트가 중복 발송되지 않음
2. severity 상승 시 재알림 가능
3. false positive suppress 후 신규 점수/알림에 반영됨
4. 운영자가 별도 DB 접속 없이 라벨/억제 관리 가능

**현재 상태**

- `[~]` watchlist service: protected CRUD API, wallet item tags/notes, plan limits, Postgres persistence baseline 완료
- `[O]` alert rule CRUD, owner-scoped audit trail endpoint, dedup/cooldown/re-notify service baseline, worker `signal snapshot -> alert rule evaluation` 자동 연결 완료
- `[O]` in-app inbox, delivery channel CRUD, worker `email/discord/telegram` delivery + retry batch baseline 완료
- `[x]` `/alerts` alert center baseline: inbox/rules/channels 화면, severity/signal filter, unread state, cursor pagination, live/fallback API boundary 완료
- `[x]` alert center snooze state baseline 완료, explicit mute/resume와 inbox read/unread, snooze/clear snooze in-product mutation UX 완료
- `[~]` admin console baseline: protected labels CRUD, suppressions CRUD, provider quota snapshot API, curated list CRUD/read API, admin audit log read API, `/admin` UI 완료
- `[x]` human override action과 deeper quota dashboard 완료

---

## Phase 10. Product UI, Billing, Beta Hardening

**기간:** 1.5주

**목표**

실제 유저가 사용할 수 있는 SaaS 제품 형태로 마감한다.

**주요 작업**

1. global search UI 완성
2. wallet page, cluster page, hot feed, alert center polish/mute-snooze
3. free/pro entitlement gating
4. Stripe checkout + billing webhook 구현
5. public landing/pricing 페이지 구현
6. observability dashboard, error tracking, audit log 검증
7. load test, rate limit, security checklist 수행
8. launch checklist와 runbook 작성

**산출물**

1. beta 배포본
2. pricing/billing 흐름
3. 운영 runbooks
4. SLO/alerting dashboard

**완료 기준**

1. Free/Pro 기능 차등 적용 확인
2. release criteria 7개 항목 모두 충족
3. critical blocker 없이 staging에서 end-to-end 시나리오 통과

---

## 10. 기능별 세부 백로그

## 10.1 Search & Identity

1. address parser: EVM/Solana/ENS 구분
2. token/entity search 인덱스
3. unknown label fallback
4. identity confidence 표기
5. 최근 검색, pinned entities

## 10.2 Wallet Intelligence

1. 24h/7d/30d activity cards
2. netflow and holdings summary
3. top counterparties table
4. signal history timeline
5. cluster candidate panel
6. async deep analysis loader

## 10.3 Graph Experience

1. neighborhood summary API
2. evidence-type edge toggle
3. entity-type coloring
4. confidence 낮은 엣지 점선 처리
5. node/edge density guardrails
6. precomputed neighborhood snapshots
7. hub-and-spoke + partial flow hints visual layout

현재 상태:
- `[~]` wallet detail에서 SVG 기반 graph preview, edge-kind filter, node/entity 색상 구분, neighborhood summary, density guardrail, low-confidence dashed edge baseline 완료
- `[O]` neighborhood summary API/read path
- `[~]` confidence 낮은 edge dashed 처리
- `[~]` density guardrail 시각화
- `[~]` Redis-backed precomputed neighborhood snapshot cache baseline
- `[O]` focal wallet 중심 `hub-and-spoke + partial flow hints` visual layout baseline
- `[O]` React Flow 기반 `pan/zoom`, `MiniMap`, `Controls`, node selection baseline
- `[O]` home search surface compact graph preview + simplified result-first layout
- `[~]` PRD 그래프 UX 방향을 WhaleGraph 고유의 `분석형 hub-and-spoke + partial flow hints`와 signal-first investigation view로 고정

## 10.4 Cluster Intelligence

1. cluster list and detail
2. member wallet diff history
3. common tokens/funders view
4. natural-language explanation card
5. operator review queue

## 10.5 Signal Feed

1. first-connection feed
2. shadow exit feed
3. emerging cluster feed
4. severity filter
5. feed dedup/grouping

## 10.6 Alerts

1. wallet/cluster/token/funding-source rule builder
2. cooldown and mute
3. severity escalation
4. alert delivery audit
5. failed delivery retry

## 10.7 Admin & Ops

1. label editor
2. suppression rule management
3. curated list management
4. provider quota dashboard
5. job failure replay
6. false positive feedback queue

## 10.8 Billing & Plans

1. plan matrix
2. feature flag gating
3. rate limit by tier
4. billing webhook reconciliation
5. subscription status sync
6. checkout session persistence and `/account` upgrade surface

---

## 11. API 설계 우선순위

beta 이전 반드시 제공할 API는 아래 순서로 개발한다.

### P0

1. `GET /wallet/:chain/:address/summary`
2. `GET /wallet/:chain/:address/graph`
3. `GET /cluster/:id`
4. `GET /signals/first-connections`
5. `GET /signals/shadow-exits`
6. `GET /alerts`

### P1

1. `POST /watchlists`
2. `POST /alert-rules`
3. `GET /search`
4. `GET /admin/provider-usage`
5. `POST /admin/suppressions`

### 공통 원칙

1. 모든 응답에 `chain`, `timestamp`, `confidence/evidence` 포함
2. summary와 detail의 캐시 정책 분리
3. free tier는 낮은 depth와 더 긴 freshness 허용

---

## 12. 품질 보증 계획

### 12.1 테스트 계층

1. Unit Test: score calculators, schema validators, dedup logic
2. Integration Test: provider adapters, DB writes, Neo4j materialization
3. Contract Test: public API response shape, provider response fixtures
4. E2E Test: search -> wallet -> watchlist -> alert rule -> alert center 흐름
5. Replay Test: 과거 raw payload를 재처리해 score 일관성 검증

### 12.2 테스트 우선순위

가장 먼저 자동화해야 하는 영역은 다음과 같다.

1. normalization schema
2. idempotency/dedup
3. scoring formulas
4. alert cooldown/re-notify
5. plan entitlements

### 12.3 수용 테스트 시나리오

1. 신규 EVM 주소 검색 시 3초 내 summary 반환
2. Solana funded-by 관계가 cluster evidence에 반영됨
3. 24시간 fan-out 발생 후 60초 내 alert candidate 생성
4. 동일 이벤트 재수신 시 중복 alert 미발송
5. Pro 사용자만 2-hop graph 접근 가능

---

## 13. 관측성 및 운영 계획

### 13.1 필수 메트릭

1. provider별 호출량, 성공률, 오류 코드
2. ingest lag
3. dedup hit rate
4. cache hit rate
5. score recompute latency
6. alert delivery success/failure
7. false positive feedback ratio

### 13.2 필수 로그

1. provider request/response metadata
2. scoring decision logs
3. alert rule evaluation logs
4. admin override logs
5. billing webhook logs

### 13.3 운영 런북

1. provider quota 초과 시 다운그레이드 정책
2. Helius 응답 포맷 변경 시 adapter hotfix 절차
3. Neo4j 지연 시 summary-only fallback 절차
4. alert 채널 장애 시 failover 순서

---

## 14. 보안 및 권한 계획

1. API key와 webhook secret 분리 저장
2. admin/operator API는 일반 사용자 경로와 별도 권한 체계 적용
3. 모든 alert, suppression, label 변경은 audit log 저장
4. rate limit은 anonymous search와 authenticated API를 분리 적용
5. object storage raw payload는 private bucket에 저장
6. PII 최소 수집 원칙 유지

---

## 15. 출시 게이트

beta 배포 전 아래 체크를 모두 통과해야 한다.

### Functional Gates

1. EVM/Solana wallet summary 정상 동작
2. seed whale batch 정상 갱신
3. 3개 score 엔진 정상 계산
4. alert dedup/cooldown 동작
5. admin label/suppression 동작
6. provider usage dashboard 존재
7. 최소 1개 유료 플랜 결제 가능

### Reliability Gates

1. webhook duplicate safe 보장
2. provider 장애 시 재시도 및 degradation 확인
3. score 실패가 ingest 전체를 block하지 않음

### UX Gates

1. wallet summary p95 3초 이하
2. cached cluster detail p95 5초 이하
3. alert 이벤트가 60초 내 제품에 표시

---

## 16. 리스크 대응 실행안

| 리스크 | 대응 계획 |
| --- | --- |
| 무료 티어 초과 | watchlist 수 제한, hot cache, 온디맨드 확장 상한, provider usage dashboard |
| 라벨 품질 낮음 | confidence 표기, 확정/추정 분리, admin override, evidence 공개 |
| false positive | 단일 규칙 경보 금지, suppression rule, useful alert feedback 수집 |
| Solana API 변경 | adapter layer 고립, fixture contract test, schema versioning |
| graph UI 과부하 | neighborhood snapshot, 1-hop 기본, density cap, progressive rendering |
| 운영 복잡성 증가 | runbook, admin console, audit log, replayable raw payload 저장 |

---

## 17. 팀 운영 방식

### 17.1 역할 제안

1. Backend A: provider adapters + ingest + normalization
2. Backend B: scoring + graph + alert engine
3. Frontend: web product UI + graph UX + admin UI
4. Fullstack/Operator: auth, billing, watchlist, ops tooling, QA support

### 17.2 주간 운영 리듬

1. 주간 계획: phase goal, quota budget, blocker 확인
2. 중간 점검: scoring 품질 및 false positive 샘플 리뷰
3. 주간 데모: wallet, cluster, signal, alert 흐름 시연
4. 운영 리뷰: provider usage, lag, error, suppressions 확인

---

## 18. 즉시 시작할 구현 순서

현재 저장소 상태 기준으로 가장 먼저 착수할 순서는 다음이다.

1. monorepo 초기화와 공통 config/schema 패키지 생성
2. Postgres/Neo4j/Redis 로컬 환경 구성
3. transaction normalization schema와 DB 마이그레이션 작성
4. Alchemy/Helius 최소 adapter 구현
5. wallet summary API 및 기본 UI 구현
6. seed discovery batch와 watchlist seed 파이프라인 구현
7. cluster/shadow/first-connection 엔진을 순차 도입
8. alerts/admin/billing 마감 후 beta hardening 수행

---

## 19. 정의된 완료 상태

이 계획서 기준 `beta ready`는 아래를 의미한다.

1. 핵심 탐지 엔진 3종이 evidence와 함께 작동한다.
2. 사용자는 검색, 추적, 알림, 구독 결제까지 제품 내에서 끝낼 수 있다.
3. 운영자는 라벨, suppressions, curated lists, provider budget을 직접 관리할 수 있다.
4. 시스템은 무료 티어 제약 안에서 점진적으로 확장 가능하다.

WhaleGraph의 첫 버전은 “모든 데이터를 다 보여주는 플랫폼”이 아니라,  
“제한된 예산 안에서 고래 행동의 의미를 빠르게 포착해주는 인텔리전스 제품”으로 완성되어야 한다.
