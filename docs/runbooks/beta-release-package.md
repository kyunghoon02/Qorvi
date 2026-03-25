# Beta Release Package

이 문서는 FlowIntel beta open 직전 운영자와 개발자가 함께 확인하는 handoff 패키지다. 상세 gate 판단은 `/Users/kh/Github/FlowIntel/docs/runbooks/launch-gates.md`를 따르고, 이 문서는 실제 실행 순서와 운영 인계를 다룬다.

## 1. Primary Entry Points

가장 먼저 사용할 명령:

```bash
corepack pnpm beta:open:prep
```

단계별 실행이 필요할 때:

```bash
corepack pnpm beta:prep
corepack pnpm beta:evidence
corepack pnpm beta:hardening
```

관련 스크립트:

- `/Users/kh/Github/FlowIntel/scripts/beta-hardening.sh`
- `/Users/kh/Github/FlowIntel/scripts/dev-stack.sh`
- `/Users/kh/Github/FlowIntel/scripts/dev-worker-loop.sh`

## 2. Release Day Preflight

1. 필수 환경 변수가 준비돼 있는지 확인한다.
   - Stripe keys
   - Clerk config
   - provider keys
   - Postgres / Neo4j / Redis URLs
2. 로컬 또는 target 환경에서 infra가 부팅 가능한지 확인한다.
3. migration이 clean하게 적용되는지 확인한다.
4. one-click evidence를 재실행한다.

## 3. Evidence Checklist

반드시 보관할 evidence:

1. `corepack pnpm beta:evidence:core` 실행 결과
2. `/Users/kh/Github/FlowIntel/docs/runbooks/launch-gates.md` 검토 결과
3. admin/ops operator 확인 결과
   - `/v1/admin/provider-quotas`
   - `/v1/admin/observability`
4. optional billing activation 확인 결과
   - billing을 beta에서 켤 경우에만 아래를 포함
   - `/v1/billing/plans`
   - `/v1/billing/checkout-sessions`
   - `/v1/webhooks/billing/stripe`
   - `/v1/account`

## 4. Operator Handoff

운영 인계 시 같이 전달해야 하는 문서:

- gate source of truth: `/Users/kh/Github/FlowIntel/docs/runbooks/launch-gates.md`
- launch review: `/Users/kh/Github/FlowIntel/docs/runbooks/beta-launch-review.md`
- beta open prep: `/Users/kh/Github/FlowIntel/docs/runbooks/beta-open-prep.md`
- operator handoff: `/Users/kh/Github/FlowIntel/docs/runbooks/beta-operator-handoff.md`
- admin runbook: `/Users/kh/Github/FlowIntel/docs/runbooks/ops-admin.md`
- release package: 이 문서

운영자가 launch 이후 바로 확인할 것:

1. provider quota warning / critical
2. ingest freshness lag
3. alert delivery failure
4. billing을 beta에서 켠 경우 billing reconciliation anomaly
5. admin override/audit consistency

## 5. Manual Recovery Entry Points

read-only 복구:

```bash
corepack pnpm dev:stack:no-worker
```

wallet indexing 재시도:

```bash
FLOWINTEL_WORKER_MODE=wallet-backfill-drain-batch corepack pnpm dev:workers
```

billing reconciliation 재시도:

```bash
FLOWINTEL_WORKER_MODE=billing-subscription-sync corepack pnpm dev:workers
```

Moralis enrichment 재시도:

```bash
FLOWINTEL_WORKER_MODE=moralis-enrichment-refresh corepack pnpm dev:workers
```

## 6. Residual Warn Items

현재 beta open을 막지는 않지만 launch 직후 follow-up 대상으로 유지하는 항목:

1. optional billing activation closeout
2. ops anomaly surfacing polish
3. beta 운영 중 실제 provider pressure에 따른 quota tuning

이 항목들은 `block`이 아니라 `warn`으로 유지한다.

## 7. Final Go/No-Go Rule

아래 조건을 만족하면 `go`로 본다.

1. `/Users/kh/Github/FlowIntel/docs/runbooks/launch-gates.md`에 `block`이 없다.
2. `corepack pnpm beta:evidence:core`가 통과한다.
3. operator가 admin/ops surface를 직접 확인했다.
4. billing을 beta에서 켠 경우 billing/account mixed flow가 현재 환경에서 재현 가능하다.
