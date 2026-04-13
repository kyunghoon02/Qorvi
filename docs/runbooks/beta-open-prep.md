# Beta Open Prep

이 문서는 Qorvi beta open 직전 마지막 환경/운영 준비 체크리스트다. gate 판단은 `/Users/kh/Github/Qorvi/docs/runbooks/beta-launch-review.md`를 따르고, 세부 복구 절차는 `/Users/kh/Github/Qorvi/docs/runbooks/beta-release-package.md`를 따른다.

env 기준:

- 로컬 개발 템플릿: [/.env.example](/Users/kh/Github/Qorvi/.env.example)
- beta 배포 템플릿: [/.env.beta.example](/Users/kh/Github/Qorvi/.env.beta.example)

현재 로컬 runtime 검증 결과:

- `corepack pnpm beta:open:prep`는 아직 `block`
- 채워야 하는 값:
  - `CLERK_SECRET_KEY`
  - `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`

현재 `warn`로만 남는 값:

- `STRIPE_SECRET_KEY`
- `STRIPE_WEBHOOK_SECRET`
- `STRIPE_PUBLISHABLE_KEY`
- `STRIPE_SUCCESS_URL`
- `STRIPE_CANCEL_URL`

beta launch policy:

- beta는 `invite-only / free beta` 기준으로 연다.
- alerts와 alert delivery는 beta 동안 `signed-in free tier`에도 개방한다.
- 따라서 Stripe checkout과 subscription sync는 코드상 유지하되, beta open blocker로 취급하지 않는다.
- billing activation은 post-beta monetization track에서 다시 올린다.

Moralis 설정 원칙:

- 현재 Qorvi의 Moralis integration은 `체인별 node URL` 모델이 아니라 `전역 API base URL + API key` 모델이다.
- 즉 `MORALIS_BASE_URL`은 체인마다 여러 개를 만들지 않고 단일 값으로 유지한다.
- 권장값:
  - `MORALIS_BASE_URL=https://deep-index.moralis.io/api/v2.2`
- 현재 Moralis enrichment는 `EVM wallet enrichment`에만 사용되므로, 체인별 RPC endpoint를 Moralis에 맞춰 별도 설계할 필요는 없다.

## 1. Environment Checklist

다음 값이 target beta environment에 준비돼 있어야 한다.

1. Clerk auth config
2. Alchemy / Helius / Moralis provider keys
3. app/web/api base URL values
4. Postgres / Neo4j / Redis connection values
5. raw payload storage 경로

optional beta-open values:

1. Stripe secret / publishable / webhook secret
2. Stripe success / cancel URL

추가 sanity rules:

1. `CLERK_ISSUER_URL`, `CLERK_JWKS_URL`는 비워둘 수 있고, 비어 있으면 `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`에서 유도한다.
2. Stripe를 활성화할 경우 `STRIPE_SUCCESS_URL`, `STRIPE_CANCEL_URL`의 origin은 `APP_BASE_URL`과 일치해야 한다.

## 2. Preflight Order

1. 환경 변수 확인
2. infra 및 migrations 확인
3. evidence 재실행
4. operator handoff 확인
5. beta open decision 재확인

권장 명령:

```bash
corepack pnpm beta:open:prep
```

특정 target env 파일을 직접 검증할 때:

```bash
./scripts/beta-open-prep.sh --env-only --env-file .env.beta
```

단계별 실행이 필요할 때:

```bash
corepack pnpm beta:prep
corepack pnpm beta:evidence
```

## 3. Operator Confirmation

운영자는 아래를 직접 확인한다.

1. `/v1/admin/provider-quotas`
2. `/v1/admin/observability`
3. alert delivery failure 여부
4. audit trail 누락 여부

billing을 beta에서 함께 켤 경우 추가 확인:

1. billing/account anomaly 여부

참고 문서:

- `/Users/kh/Github/Qorvi/docs/runbooks/beta-operator-handoff.md`
- `/Users/kh/Github/Qorvi/docs/runbooks/ops-admin.md`

## 4. Final Ready Check

아래가 모두 참이면 ready다.

1. `/Users/kh/Github/Qorvi/docs/runbooks/beta-launch-review.md`가 `go`
2. `corepack pnpm beta:evidence:core`가 통과
3. 운영자 확인 완료

billing을 beta에서 함께 활성화할 경우:

1. `corepack pnpm beta:evidence:billing`이 통과
2. billing/account mixed flow가 target environment에서 재현 가능
