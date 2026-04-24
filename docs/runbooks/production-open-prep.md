# Production Open Prep

이 문서는 Qorvi production launch 직전 마지막 환경/운영 준비 체크리스트다. gate 판단은 `docs/runbooks/production-launch-review.md`를 따르고, 세부 복구 절차는 `docs/runbooks/production-release-package.md`를 따른다.

env 기준:

- 로컬 개발 템플릿: [`.env.example`](../../.env.example)
- target production 템플릿: [`.env.production.example`](../../.env.production.example)

현재 로컬 dry-run 결과:

- `./scripts/production-open-prep.sh --env-only --env-file .env` -> `PASS=24 WARN=8 BLOCK=0`
- `corepack pnpm prod:prep` -> 통과
- `corepack pnpm prod:evidence:core` -> 통과
- `corepack pnpm prod:open:prep --env-file .env.production.seeded.draft` -> `PASS Ready for production launch`
- local operator smoke -> 통과
  - `/v1/admin/provider-quotas`
  - `/v1/admin/observability`
  - `/v1/wallets/:chain/:address/brief`
  - `/v1/analyst/findings`

주의:

- local smoke의 admin 확인은 development mock Clerk headers 기준이다.
- target production environment에서는 실제 Clerk session/role로 다시 확인해야 한다.
- `.env.production.seeded.draft`는 seeded rehearsal 파일이다. 실제 production launch 전에는 real target env 값으로 한 번 더 동일 preflight를 돌려야 한다.

## 1. Environment Checklist

다음 값이 target production environment에 준비돼 있어야 한다.

1. Clerk auth config
2. Alchemy / Helius / Moralis provider keys
3. app/web/api base URL values
4. Postgres / Neo4j / Redis connection values
5. raw payload storage 경로

optional production-launch values:

1. Stripe secret / publishable / webhook secret
2. Stripe success / cancel URL
3. SMTP alert delivery config

추가 sanity rules:

1. `CLERK_ISSUER_URL`, `CLERK_JWKS_URL`는 비워둘 수 있고, 비어 있으면 `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`에서 유도한다.
2. Stripe를 활성화할 경우 `STRIPE_SUCCESS_URL`, `STRIPE_CANCEL_URL`의 origin은 `APP_BASE_URL`과 일치해야 한다.

## 2. Preflight Order

1. 환경 변수 확인
2. infra 및 migrations 확인
3. evidence 재실행
4. operator handoff 확인
5. production launch decision 재확인

권장 명령:

```bash
corepack pnpm prod:open:prep
```

특정 target env 파일을 직접 검증할 때:

```bash
./scripts/production-open-prep.sh --env-only --env-file .env.production
corepack pnpm prod:open:prep --env-file .env.production.seeded.draft
```

단계별 실행이 필요할 때:

```bash
corepack pnpm prod:prep
corepack pnpm prod:evidence
```

## 3. Operator Confirmation

운영자는 아래를 직접 확인한다.

1. `/v1/admin/provider-quotas`
2. `/v1/admin/observability`
3. alert delivery failure 여부
4. audit trail 누락 여부
5. representative finding sample에서 `evidence-timeline`이 `txRef/pathRef/entityRef/counterpartyRef`를 모두 싣는지 확인
6. representative wallet brief가 `200`으로 응답하고 `lookup failed`가 재발하지 않는지 확인
7. real target env 기준 origin/app/api routing 값이 seeded rehearsal과 다를 경우 다시 smoke

billing을 production launch에 함께 켤 경우 추가 확인:

1. billing/account anomaly 여부

참고 문서:

- `docs/runbooks/production-operator-handoff.md`
- `docs/runbooks/ops-admin.md`

## 4. Final Ready Check

아래가 모두 참이면 ready다.

1. `docs/runbooks/production-launch-review.md`가 `go`
2. `corepack pnpm prod:evidence:core`가 통과
3. 운영자 확인 완료
4. representative replay diff에서 finding count 급증/누락이 없음
5. real target env 값으로도 `prod:open:prep`이 다시 통과

billing을 production launch에 함께 활성화할 경우:

1. `corepack pnpm prod:evidence:billing`이 통과
2. billing/account mixed flow가 target environment에서 재현 가능
