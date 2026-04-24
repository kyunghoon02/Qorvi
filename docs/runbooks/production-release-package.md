# Production Release Package

이 문서는 Qorvi production launch 직전 운영자와 개발자가 함께 확인하는 handoff 패키지다. 상세 gate 판단은 `docs/runbooks/launch-gates.md`를 따르고, 이 문서는 실제 실행 순서와 운영 인계를 다룬다.

## 1. Primary Entry Points

가장 먼저 사용할 명령:

```bash
corepack pnpm prod:open:prep
```

단계별 실행이 필요할 때:

```bash
corepack pnpm prod:prep
corepack pnpm prod:evidence
corepack pnpm prod:hardening
```

관련 스크립트:

- `scripts/production-hardening.sh`
- `scripts/production-open-prep.sh`
- `scripts/dev-stack.sh`
- `scripts/dev-worker-loop.sh`

## 2. Release Day Preflight

1. 필수 환경 변수가 준비돼 있는지 확인한다.
2. target environment에서 infra가 부팅 가능한지 확인한다.
3. migration이 clean하게 적용되는지 확인한다.
4. one-click evidence를 재실행한다.

## 3. Evidence Checklist

반드시 보관할 evidence:

1. `corepack pnpm prod:evidence:core` 실행 결과
2. `docs/runbooks/launch-gates.md` 검토 결과
3. admin/ops operator 확인 결과
   - `/v1/admin/provider-quotas`
   - `/v1/admin/observability`
4. optional billing activation 확인 결과
5. engine replay / evidence completeness 확인 결과
   - 최근 representative wallet set 기준 pre/post replay diff
   - treasury/MM finding의 `txRef`, `pathRef`, `entityRef`, `counterpartyRef`, `pathStrength`, `confidenceTier` 누락률
   - bridge/exchange finding의 downstream confirmation 누락률
6. backtest dataset evidence
   - `docs/runbooks/dune-backtest-collection.md` 기준 reviewed candidate 수집 결과
   - `analysis-backtest-manifest-validate` 결과
   - `bridge_return`, `aggregator_routing`, `smart_money_early_entry` 최소 reviewed case 현황

## 4. Operator Handoff

운영 인계 시 같이 전달해야 하는 문서:

- gate source of truth: `docs/runbooks/launch-gates.md`
- launch review: `docs/runbooks/production-launch-review.md`
- production open prep: `docs/runbooks/production-open-prep.md`
- operator handoff: `docs/runbooks/production-operator-handoff.md`
- admin runbook: `docs/runbooks/ops-admin.md`
- admin checklist: `docs/runbooks/admin-operations-checklist.md`
- release package: 이 문서

## 5. Manual Recovery Entry Points

read-only 복구:

```bash
corepack pnpm dev:stack:no-worker
```

wallet indexing 재시도:

```bash
QORVI_WORKER_MODE=wallet-backfill-drain-batch corepack pnpm dev:workers
```

billing reconciliation 재시도:

```bash
QORVI_WORKER_MODE=billing-subscription-sync corepack pnpm dev:workers
```

Moralis enrichment 재시도:

```bash
QORVI_WORKER_MODE=moralis-enrichment-refresh corepack pnpm dev:workers
```

## 6. Engine Safety Checks

production rollout 전 아래를 확인한다.

1. `wallet_bridge_links`, `wallet_exchange_paths`, `wallet_treasury_paths`, `wallet_mm_paths` replay 결과가 idempotent하다.
2. finding count가 representative sample에서 비정상 급증하지 않는다.
3. `contact-only`, `ops-only`, `single-touch`, `no downstream confirmation` 케이스가 finding으로 승격되지 않는다.
4. alert blast 방지를 위해 초기 rollout은 high-threshold 또는 observe-only rule 세트로 시작한다.

## 7. Final Go/No-Go Rule

아래 조건을 만족하면 `go`로 본다.

1. `docs/runbooks/launch-gates.md`에 `block`이 없다.
2. `corepack pnpm prod:evidence:core`가 통과한다.
3. operator가 admin/ops surface를 직접 확인했다.
4. billing을 production launch에 켠 경우 billing/account mixed flow가 현재 환경에서 재현 가능하다.
