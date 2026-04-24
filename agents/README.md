# Qorvi Subagents

이 디렉터리는 Qorvi 개발을 병렬로 진행하기 위한 프로젝트 전용 subagent 정의를 담는다.

## 사용 원칙

1. 각 subagent는 아래 agent map의 워크스트림 하나를 담당한다.
2. 기능 개발 전에는 담당 subagent를 먼저 선택하고, 산출물과 완료 기준을 맞춘다.
3. cross-stream 작업은 소유권을 분리한다. 예를 들어 API contract는 `api-platform-engineer`, provider adapter는 `provider-integration-engineer`가 책임진다.
4. 점수, 알림, 라벨링처럼 운영 리스크가 큰 변경은 `ops-admin-engineer`와 함께 검토한다.
5. 배포 직전 변경은 `billing-launch-engineer`가 launch gate 관점에서 최종 체크한다.

## Agent Map

| Agent | Workstream | 주요 범위 |
| --- | --- | --- |
| `foundation-architect` | WS1 Foundation | monorepo, CI, env, auth, repo standards |
| `data-platform-engineer` | WS2 Data Platform | Postgres, Neo4j, Redis, raw storage, schema |
| `provider-integration-engineer` | WS3 Provider Adapters | Dune, Alchemy, Helius, Moralis adapter |
| `intelligence-engineer` | WS4 Intelligence Engine | cluster, shadow exit, first-connection |
| `api-platform-engineer` | WS5 API Layer | public/internal API, contracts, rate limit |
| `product-ui-engineer` | WS6 Product UI | search, wallet, cluster, signals, alerts UI |
| `ops-admin-engineer` | WS7 Ops & Admin | labeling, suppression, curated lists, audit, quota |
| `billing-launch-engineer` | WS8 Billing & Launch | pricing, Stripe, observability, beta gates |

## 권장 오케스트레이션 순서

1. `foundation-architect`
2. `data-platform-engineer`
3. `provider-integration-engineer`
4. `intelligence-engineer`
5. `api-platform-engineer`
6. `product-ui-engineer`
7. `ops-admin-engineer`
8. `billing-launch-engineer`

## ECC 기본 에이전트와의 연결

1. 복잡한 구조 설계: `architect`
2. 구현 순서 분해: `planner`
3. 테스트 우선 구현: `tdd-guide`
4. 변경 후 검토: `code-reviewer`
5. 보안 검토: `security-reviewer`
6. E2E 플로우 검증: `e2e-runner`

각 프로젝트 subagent는 위 기본 에이전트를 대체하는 것이 아니라, Qorvi 도메인에 맞는 책임 분리를 제공하는 역할이다.
