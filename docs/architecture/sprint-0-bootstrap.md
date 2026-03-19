# Sprint 0 Bootstrap Decisions

이 문서는 `WG-001`, `WG-002`, `WG-003`의 최소 실행안을 고정한다. 목표는 문서 작성 자체가 아니라, Sprint 0 안에 첫 번째 수직 슬라이스인 `wallet summary` 개발을 시작할 수 있게 만드는 것이다.

## 1. Scope Freeze

### Must

| 영역 | Beta 기준 |
| --- | --- |
| Search | EVM/Solana wallet search |
| Intelligence | wallet summary, basic graph, cluster/shadow/first-connection score |
| Operations | watchlist, alerts, admin suppression, provider usage visibility |
| Commercial | Free/Pro plan gate, Stripe subscription state |

### Should

| 영역 | Beta 이후로 미루지 않되 첫 슬라이스 범위 밖 |
| --- | --- |
| Graph UX | progressive rendering, depth gate |
| Admin | curated list tooling, audit drill-down |
| Alerts | email, Telegram, Discord delivery hardening |

### Later

| 영역 | Beta 비포함 |
| --- | --- |
| AI explainability | 자연어 설명 고도화 |
| Collaboration | 조사 리포트, 팀 공유 |
| Export | bulk export job |
| Analytics | advanced PnL |

## 2. Foundation Decisions

| 영역 | 결정 |
| --- | --- |
| Workspace | `pnpm` workspace + `turbo` for web, `go work` + modules for backend |
| Web | Next.js App Router + TypeScript |
| Backend services | Go |
| Backend libraries | Go |
| Data access | Go SQL + migration |
| Auth | Clerk JWT/JWKS verification |
| Billing | Stripe |
| Infra | Docker Compose 기반 Postgres, Neo4j, Redis |

이 결정은 "가장 빨리 첫 vertical slice를 올릴 수 있는가"를 기준으로 했다. `Node` 기반 백엔드 프레임워크와 ORM은 기본 선택이 아니며, 백엔드는 Go로 통일한다. Clerk 인증은 헤더 스텁이 아니라 issuer/JWKS 기반 검증 경로를 전제로 한다.

## 3. Domain Contract Baseline

모든 API 응답은 아래 envelope를 사용한다.

```ts
type ResponseEnvelope<T> = {
  success: boolean;
  data: T | null;
  error: ApiError | null;
  meta: {
    requestId: string;
    timestamp: string;
    chain?: "evm" | "solana";
    tier?: "free" | "pro" | "team";
    freshness: FreshnessMeta;
    pagination?: PaginationMeta;
  };
};
```

핵심 원칙은 다음과 같다.

1. 점수는 반드시 `evidence[]`와 함께 반환한다.
2. `role`과 `plan`은 분리한다.
3. `role`은 보안 주체다: `anonymous`, `user`, `admin`, `operator`
4. `plan`은 상품 entitlement다: `free`, `pro`, `team`
5. 첫 vertical slice는 `GET /v1/wallets/:chain/:address/summary`를 기준으로 계약을 검증한다.

## 4. Bootstrap Sequence

1. root workspace 정책과 release scripts를 고정한다.
2. web 쪽 `pnpm` workspace를 고정하고 Next.js app shell을 올린다.
3. backend 쪽 Go workspaces/modules와 shared libraries를 고정한다.
4. web env schema와 backend env/config loader를 분리한다.
5. Clerk issuer/JWKS/audience 설정을 backend env에 넣고 verifier를 초기화한다.
6. Go backend에서 health route와 wallet summary vertical slice를 올린다.
7. `apps/web`에서 검색 진입 화면과 상태 패널을 만든다.
8. backend workers/services에서 env 검증과 boot 로그를 먼저 연결한다.
9. `infra/docker`에서 Postgres, Neo4j, Redis 로컬 스택을 띄운다.

## 5. Immediate Next Slice

Sprint 0의 다음 구현 목표는 아래와 같다.

1. `wallet summary` route를 fixture 기반 응답에서 실제 Postgres 조회로 바꾼다.
2. auth/RBAC skeleton을 Clerk session JWT/JWKS 검증으로 대체한다.
3. `WG-006`부터 로컬 인프라와 schema v1을 연결한다.
