# Beta Operator Handoff

이 문서는 Qorvi beta 운영자가 바로 참고하는 handoff 요약본이다. gate 판단은 `docs/runbooks/launch-gates.md`, release closeout 순서는 `docs/runbooks/beta-release-package.md`, 세부 운영 절차는 `docs/runbooks/ops-admin.md`를 따른다.

## 1. Scope

beta 운영자는 아래 5가지를 책임진다.

1. search / wallet / graph read path 정상 여부 확인
2. provider quota 및 ingest freshness 확인
3. alert delivery failure와 inbox 이상징후 확인
4. billing을 beta에서 켠 경우 billing reconciliation anomaly 확인
5. labels / suppressions / audit consistency 확인

## 2. Service Map

- web: Next.js app router
- api: Go HTTP API
- workers: batch/reconciliation/enrichment jobs
- Postgres: canonical transactional store
- Neo4j: wallet relationship graph
- Redis: queue, cache, dedup

기본 아키텍처 참고:

- `docs/architecture/sprint-0-bootstrap.md`

## 3. Daily Operator Checks

1. stack health

```bash
corepack pnpm dev:stack
curl http://localhost:4000/healthz
```

2. admin observability / provider quotas 확인
3. alert delivery failure 여부 확인
4. billing을 beta에서 켠 경우 billing/account anomaly 확인
5. audit trail 누락 여부 확인

핵심 문서와 경로:

- `docs/runbooks/launch-gates.md`
- `docs/runbooks/ops-admin.md`
- `apps/api/internal/server/admin_console.go`

## 4. Common Actions

read-only surface 복구:

```bash
corepack pnpm dev:stack:no-worker
```

wallet backfill 재시도:

```bash
FLOWINTEL_WORKER_MODE=wallet-backfill-drain-batch corepack pnpm dev:workers
```

optional billing subscription sync 재실행:

```bash
FLOWINTEL_WORKER_MODE=billing-subscription-sync corepack pnpm dev:workers
```

Moralis enrichment refresh 재실행:

```bash
FLOWINTEL_WORKER_MODE=moralis-enrichment-refresh corepack pnpm dev:workers
```

## 5. Incident Triage

### Search / Wallet degraded

1. `/healthz` 확인
2. observability snapshot 확인
3. read-only surface로 우선 복구
4. wallet backfill worker 재시도

### Provider pressure

1. quota warning/critical provider 확인
2. 비필수 enrichment 우선 축소
3. operator note와 audit trail 남김

### Alert delivery failure

1. delivery attempts와 retry 상태 확인
2. channel misconfiguration 여부 확인
3. suppression이 아니라 delivery failure인지 분리

### Billing mismatch

1. billing을 beta에서 켠 경우 checkout/webhook/account 상태 순서대로 확인
2. billing을 beta에서 켠 경우 billing subscription sync 재실행
3. unresolved mismatch는 `warn`로 승격 후 운영자 공유

## 6. Escalation Ownership

- ops/admin surface 문제: `ops-admin-engineer`
- API/auth/watchlist/alerts 문제: `api-platform-engineer`
- billing/account/Stripe 문제: billing을 beta에서 켰을 때만 `billing-launch-engineer`
- provider/ingest/enrichment 문제: `provider-integration-engineer`
