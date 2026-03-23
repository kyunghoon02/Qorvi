# Beta Launch Gates

이 문서는 WhaleGraph beta closeout의 최종 source of truth다. `WG-043`는 이 문서의 각 게이트가 `pass` 또는 `warn`으로 정리되고 `block`이 남아 있지 않을 때 완료로 본다.

## 1. Gate Status Rules

- `pass`: 코드/테스트/운영 경로가 현재 저장소 기준으로 검증됨
- `warn`: beta 운영은 가능하지만 launch 직후 후속 hardening이 권장됨
- `block`: beta open 전 반드시 수정 필요

현재 기준 시점: `2026-03-23`

## 2. Current Launch Decision

| Gate | Status | Notes |
| --- | --- | --- |
| Functional Gates | `pass` | search, wallet detail, graph, alerts 핵심 흐름 검증 완료 |
| Reliability Gates | `pass` | replay, provider contract, webhook duplicate safety, worker refresh/invalidation 경로 검증 완료 |
| UX Gates | `pass` | loading/indexing/ready degraded states와 beta mixed-flow E2E 검증 완료 |
| Ops Gates | `pass` | observability, provider quotas, alert delivery, audit/admin surfaces 존재 |
| Residual Hardening | `warn` | billing activation, residual ops polish는 가능하지만 launch blocker는 아님 |

결론: `beta go`, 단 `Residual Hardening`은 launch 직후 follow-up으로 유지한다.

## 3. Functional Gates

### 3.1 Discovery And Wallet Intelligence

체크 항목:

1. `GET /v1/search`가 EVM/Solana 주소를 정상 해석한다.
2. `GET /v1/wallets/:chain/:address/summary`가 summary/indexing/coverage 상태를 반환한다.
3. `GET /v1/wallets/:chain/:address/graph`가 `live`, `summary-derived`, `unavailable` 상태를 일관되게 반환한다.
4. search -> wallet detail -> graph evidence 흐름이 브라우저에서 동작한다.

현재 상태: `pass`

증빙 명령:

```bash
corepack pnpm --filter @whalegraph/web typecheck
corepack pnpm --filter @whalegraph/web lint
GOCACHE=/tmp/whalegraph-go-cache go test ./packages/providers ./apps/api/internal/server ./apps/workers
corepack pnpm --filter @whalegraph/web test:e2e -- e2e/beta-flow.spec.ts
```

관련 경로:

- `/Users/kh/Github/WhaleGraph/apps/api/internal/server/server.go`
- `/Users/kh/Github/WhaleGraph/apps/web/app/page.tsx`
- `/Users/kh/Github/WhaleGraph/apps/web/app/wallets/[chain]/[address]/page.tsx`
- `/Users/kh/Github/WhaleGraph/apps/web/e2e/beta-flow.spec.ts`

### 3.2 Tracking And Alerts

체크 항목:

1. authenticated watchlist create/list/get이 동작한다.
2. wallet detail에서 tracked alerts flow가 열리고 `/alerts` flash state가 연결된다.
3. alert inbox read/unread, mute, snooze 경로가 동작한다.

현재 상태: `pass`

관련 경로:

- `/Users/kh/Github/WhaleGraph/apps/api/internal/server/watchlist.go`
- `/Users/kh/Github/WhaleGraph/apps/api/internal/server/alert_rule.go`
- `/Users/kh/Github/WhaleGraph/apps/web/app/alerts/page.tsx`
- `/Users/kh/Github/WhaleGraph/apps/web/app/alerts/alert-center-screen.tsx`

### 3.3 Billing Activation Readiness

체크 항목:

1. `GET /v1/billing/plans`가 plan catalog를 반환한다.
2. `POST /v1/billing/checkout-sessions`가 live checkout session을 생성한다.
3. `POST /v1/webhooks/billing/stripe`가 billing account persistence와 entitlement sync를 갱신한다.
4. `/account`와 `/pricing`가 success/cancel state를 정상 표시한다.

현재 상태: `warn`

정책:

1. billing capability는 구현 완료 상태로 유지한다.
2. beta open은 invite-only/free beta를 우선하므로 Stripe activation을 blocker로 두지 않는다.
3. billing을 beta에서 켜는 경우에만 아래 체크 항목을 launch gate로 승격한다.

관련 경로:

- `/Users/kh/Github/WhaleGraph/apps/api/internal/server/billing.go`
- `/Users/kh/Github/WhaleGraph/apps/workers/billing_sync.go`
- `/Users/kh/Github/WhaleGraph/apps/web/app/account/page.tsx`
- `/Users/kh/Github/WhaleGraph/apps/web/app/pricing/page.tsx`
- `/Users/kh/Github/WhaleGraph/apps/web/e2e/beta-flow.spec.ts`

## 4. Reliability Gates

### 4.1 Replay, Dedup, And Recovery

체크 항목:

1. raw payload replay가 동일 normalized transaction contract를 재현한다.
2. webhook duplicate-safe path가 중복 과금/중복 활성화/중복 alert를 일으키지 않는다.
3. ingest dedup failure poisoning이 release path로 복구 가능하다.
4. worker refresh가 summary/graph cache invalidation을 동반한다.

현재 상태: `pass`

관련 경로:

- `/Users/kh/Github/WhaleGraph/apps/api/internal/server/webhook_replay_test.go`
- `/Users/kh/Github/WhaleGraph/packages/db/ingest_dedup.go`
- `/Users/kh/Github/WhaleGraph/packages/db/wallet_graph_invalidation.go`
- `/Users/kh/Github/WhaleGraph/packages/db/wallet_summary.go`

### 4.2 Provider And Worker Stability

체크 항목:

1. Alchemy, Helius, Moralis provider contract tests가 fixture 기준으로 고정되어야 한다.
2. Helius paid-plan 403은 fallback 또는 graceful degradation으로 처리된다.
3. worker loop는 실패 후 재시도 가능하고 batch mode가 loop에서 다시 올라온다.

현재 상태: `pass`

운영 명령:

```bash
corepack pnpm dev:stack
WHALEGRAPH_WORKER_MODE=wallet-backfill-drain-batch corepack pnpm dev:workers
WHALEGRAPH_WORKER_MODE=billing-subscription-sync corepack pnpm dev:workers
WHALEGRAPH_WORKER_MODE=moralis-enrichment-refresh corepack pnpm dev:workers
```

## 5. UX Gates

체크 항목:

1. product-facing surface는 fabricated preview data 없이 `loading`, `indexing`, `empty`, `unavailable`, `error`, `summary-derived` 상태만 사용한다.
2. search hit/miss, stale refresh, manual refresh, indexing polling이 브라우저에서 자연스럽게 동작한다.
3. graph/detail/account/alerts는 query-state와 degraded state를 일관되게 읽는다.
4. beta 핵심 mixed flow가 Playwright로 통과한다.

현재 상태: `pass`

관련 경로:

- `/Users/kh/Github/WhaleGraph/apps/web/lib/api-boundary.ts`
- `/Users/kh/Github/WhaleGraph/apps/web/app/home-screen.tsx`
- `/Users/kh/Github/WhaleGraph/apps/web/app/wallets/[chain]/[address]/wallet-detail-screen.tsx`
- `/Users/kh/Github/WhaleGraph/apps/web/e2e/beta-flow.spec.ts`

## 6. Ops Gates

체크 항목:

1. `/v1/admin/provider-quotas`와 `/v1/admin/observability`가 operator 판단에 필요한 snapshot을 제공한다.
2. admin labels, suppressions, curated lists, audit review surface가 존재한다.
3. alert delivery attempts와 retry 경로가 보인다.
4. billing/account anomalies를 operator가 추적할 수 있다.

현재 상태: `pass`

관련 경로:

- `/Users/kh/Github/WhaleGraph/apps/api/internal/server/admin_console.go`
- `/Users/kh/Github/WhaleGraph/apps/web/app/admin/page.tsx`
- `/Users/kh/Github/WhaleGraph/docs/runbooks/ops-admin.md`

## 7. Evidence Bundle

beta closeout에서 최소로 보관해야 하는 증빙은 아래 네 묶음이다.

primary one-click entrypoint:

```bash
corepack pnpm beta:hardening
```

1. backend contract evidence
   - `GOCACHE=/tmp/whalegraph-go-cache go test ./packages/providers ./apps/api/internal/server ./apps/workers`
2. web type/lint evidence
   - `corepack pnpm --filter @whalegraph/web typecheck`
   - `corepack pnpm --filter @whalegraph/web lint`
3. browser/API mixed beta flow evidence
   - `corepack pnpm --filter @whalegraph/web test:e2e -- e2e/beta-flow.spec.ts --grep "searches a wallet and lands on tracked alerts"`
4. gate document review
   - 이 문서와 `/Users/kh/Github/WhaleGraph/plan.md`
   - `/Users/kh/Github/WhaleGraph/task.md`
   - `/Users/kh/Github/WhaleGraph/docs/runbooks/beta-release-package.md`
   - `/Users/kh/Github/WhaleGraph/docs/runbooks/beta-launch-review.md`

optional billing evidence:

- `corepack pnpm beta:evidence:billing`

## 8. Rollback And Recovery Package

### 8.1 Search Or Wallet Read Path Regression

증상:

- search가 지속 실패
- wallet detail이 `unavailable`만 반환
- graph snapshot이 stale하게 보임

조치:

1. `corepack pnpm dev:stack:no-worker`로 read-only surface를 우선 복구한다.
2. `/healthz`와 `/v1/admin/observability`를 확인한다.
3. 필요하면 worker를 개별 mode로 재기동한다.

```bash
WHALEGRAPH_WORKER_MODE=wallet-backfill-drain-batch corepack pnpm dev:workers
```

### 8.2 Billing Regression

증상:

- checkout success 이후 `/account` tier가 갱신되지 않음
- webhook delivery가 누락됨

조치:

1. `/v1/webhooks/billing/stripe` 수신 로그와 persistence row를 확인한다.
2. `billing-subscription-sync` worker를 수동 실행해 reconciliation을 다시 맞춘다.
3. billing blocker가 남으면 `/pricing`의 upgrade CTA는 유지하되 beta open 판단은 `warn`으로 남긴다.

```bash
WHALEGRAPH_WORKER_MODE=billing-subscription-sync corepack pnpm dev:workers
```

### 8.3 Enrichment Or Provider Pressure

증상:

- Moralis enrichment 지연
- provider quota warning/critical
- live graph/signal은 정상이나 holdings/balance freshness만 떨어짐

조치:

1. `/v1/admin/provider-quotas`에서 warning/critical provider를 확인한다.
2. 비필수 enrichment를 우선 낮춘다.
3. 필요 시 `moralis-enrichment-refresh`를 수동 실행한다.

```bash
WHALEGRAPH_WORKER_MODE=moralis-enrichment-refresh corepack pnpm dev:workers
```

### 8.4 Operator Action Rules

1. suppressions와 label 변경은 항상 audit trail이 남아야 한다.
2. verified false positive는 suppression을 우선 사용한다.
3. manual override가 graph/entity presentation에 영향을 주면 summary/graph cache invalidation이 자동으로 따라온다.
4. irreversible workaround는 beta blocker를 해결할 때도 금지한다.

## 9. Exit Criteria

아래 조건이 모두 만족되면 `WG-043`는 완료다.

1. 이 문서의 모든 gate가 `pass` 또는 `warn`으로 정리된다.
2. `block` 상태가 없다.
3. evidence bundle이 현재 저장소 명령으로 재실행 가능하다.
4. rollback/recovery 절차가 operator가 그대로 따라 할 수 있을 정도로 구체적이다.
