# Production Launch Review

이 문서는 Qorvi production launch 직전 실제 gate review 결과를 기록하는 문서다. 상세 기준은 `docs/runbooks/launch-gates.md`를 따르고, 운영 준비는 `docs/runbooks/production-release-package.md`와 `docs/runbooks/production-operator-handoff.md`를 따른다.

## 1. Review Snapshot

- Review date: `2026-03-29`
- Decision: `pass for seeded production rehearsal / pending real target env operator sign-off`
- Reviewer set:
  - engineering closeout
  - operator handoff
  - production launch review

## 2. Executed Evidence

실행 명령:

```bash
./scripts/production-open-prep.sh --env-only --env-file .env
corepack pnpm prod:prep
corepack pnpm prod:evidence:core
corepack pnpm prod:open:prep --env-file .env.production.seeded.draft
```

현재 로컬 dry-run 결과:

1. `./scripts/production-open-prep.sh --env-only --env-file .env`
   - `PASS=24 WARN=8 BLOCK=0`
   - warn만 남음: `DUNE_API_KEY`, Stripe keys, SMTP host
2. `corepack pnpm prod:prep`
   - infra up + Postgres/Neo4j migrations 적용 성공
3. `corepack pnpm prod:evidence:core`
   - `@qorvi/web typecheck` 통과
   - `@qorvi/web lint` 통과
   - backend/provider/worker contracts 통과
   - tracked wallet flow E2E 통과
   - billing/account reconciliation E2E 통과
4. local operator handoff smoke
   - `/v1/admin/provider-quotas` 응답 확인
   - `/v1/admin/observability` 응답 확인
   - `/v1/wallets/:chain/:address/brief` 응답 확인
   - `/v1/analyst/findings` 응답 확인
5. seeded production rehearsal
   - `corepack pnpm prod:open:prep --env-file .env.production.seeded.draft`
   - 결과: `PASS Ready for production launch`
   - residual warn만 남음: `DUNE_API_KEY`, Stripe values, SMTP host

## 3. Gate Outcome

| Gate | Outcome | Basis |
| --- | --- | --- |
| Functional | `pass(rehearsal) / pending(target)` | local dry-run, operator smoke, seeded production rehearsal에서 wallet brief/admin/analyst surface 확인 완료 |
| Reliability | `pass(rehearsal) / pending(target)` | replay, provider contract, worker refresh/invalidation, webhook duplicate safety local evidence와 seeded rehearsal에서 통과 |
| UX | `pass` | `prod:evidence:core`가 web typecheck/lint와 representative E2E를 통과 |
| Ops | `pass(rehearsal) / pending(target)` | provider quotas와 observability 응답 확인 완료, 실제 target env operator sign-off만 남음 |
| Launch Residuals | `warn` | billing activation, ops polish, provider quota tuning은 rollout 이후에도 follow-up 유지 가능 |

## 4. Blocking Issues

production launch 전 해소해야 할 항목:

1. real target production env 값으로 동일 preflight 재실행
2. real target env operator sign-off
3. replay/rollback package 실제 target env 기준 재확인
4. representative wallet set 기준 engine replay diff와 evidence completeness 확인

참고:

- local operator smoke 당시 보였던 `wallet brief lookup failed`는 wallet summary stats SQL grouping 오류 수정으로 해소됨.
- `/v1/admin/observability`의 recent failures는 과거 worker 실행 이력이며, 현재 launch blocker가 아님.

## 5. Open Conditions

아래 조건이 그대로 유지되면 production launch 상태를 유지한다.

1. `docs/runbooks/launch-gates.md`에 새로운 `block`이 추가되지 않는다.
2. `corepack pnpm prod:open:prep`가 target production environment에서 통과한다.
3. `corepack pnpm prod:evidence:core`가 통과한다.
4. operator가 `/v1/admin/provider-quotas`, `/v1/admin/observability`를 직접 확인한다.

billing을 production launch에 함께 활성화할 경우:

1. `corepack pnpm prod:evidence:billing`이 통과한다.
2. billing/account mixed flow가 target production environment에서도 재현 가능하다.

## 6. Next Action

launch 직전 마지막 순서:

1. `corepack pnpm prod:open:prep --env-file <real-target-env-file>`
2. real target env 기준 `corepack pnpm prod:evidence:core` 재실행
3. `docs/runbooks/production-release-package.md` 검토
4. operator sign-off 기록
