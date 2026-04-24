# Beta Launch Review

이 문서는 Qorvi beta open 직전 실제 gate review 결과를 기록하는 문서다. 상세 기준은 `docs/runbooks/launch-gates.md`를 따르고, 운영 준비는 `docs/runbooks/beta-release-package.md`와 `docs/runbooks/beta-operator-handoff.md`를 따른다.

## 1. Review Snapshot

- Review date: `2026-03-23`
- Decision: `go after env unblock`
- Reviewer set:
  - engineering closeout
  - operator handoff
  - launch review

## 2. Executed Evidence

실행 명령:

```bash
corepack pnpm beta:evidence:core
```

실행 결과:

1. `@flowintel/web typecheck` 통과
2. `@flowintel/web lint` 통과
3. `go test ./packages/providers ./apps/api/internal/server ./apps/workers` 통과
4. `corepack pnpm --filter @flowintel/web test:e2e -- e2e/beta-flow.spec.ts --grep "searches a wallet and lands on tracked alerts"` 통과
   - `searches a wallet and lands on tracked alerts`

## 3. Gate Outcome

| Gate | Outcome | Basis |
| --- | --- | --- |
| Functional | `pass` | wallet/search/graph/alerts core flow 검증 완료 |
| Reliability | `pass` | provider contracts, replay, worker refresh/invalidation, webhook duplicate safety 확인 |
| UX | `pass` | degraded state 정책과 mixed E2E 검증 완료 |
| Ops | `pass` | admin observability, provider quotas, alert delivery/admin surfaces 존재 |
| Residual Hardening | `warn` | billing activation, ops polish, quota tuning은 beta 이후 follow-up 유지 |

## 4. Blocking Issues

코드/테스트 gate 기준 `block`은 없다. 다만 실제 beta open prep 기준으로는 runtime env blocker가 남아 있다.

현재 open 전에 해소해야 할 항목:

1. Clerk secret / publishable key

launch 이후 follow-up으로 남길 `warn`:

1. billing activation closeout
2. ops anomaly surfacing polish
3. provider pressure에 따른 quota tuning

## 5. Open Conditions

아래 조건이 그대로 유지되면 beta open 상태를 유지한다.

1. `docs/runbooks/launch-gates.md`에 새로운 `block`이 추가되지 않는다.
2. `corepack pnpm beta:open:prep`가 target environment에서 통과한다.
3. operator가 `/v1/admin/provider-quotas`, `/v1/admin/observability`를 직접 확인한다.

billing을 beta에서 함께 활성화할 경우:

1. `corepack pnpm beta:evidence:billing`이 통과한다.
2. billing/account mixed flow가 target beta environment에서도 재현 가능하다.

## 6. Next Action

launch 이후 follow-up 순서:

1. residual `warn` 정리
2. beta 운영 중 anomaly 샘플 수집
3. post-beta backlog reset
