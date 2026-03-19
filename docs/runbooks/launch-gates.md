# Launch Gates

이 문서는 WhaleGraph beta 출시 전 `billing-launch-engineer`가 확인해야 하는 최소 게이트를 정의한다.

## Gate 1. Pricing And Entitlements

1. Free, Pro, Team 플랜이 문서와 코드에서 동일하게 정의되어야 한다.
2. 각 플랜의 허용 기능이 `web`, `api`, `worker` 경계에서 일관되게 적용되어야 한다.
3. `plan`과 `role`은 분리되어야 한다. `plan`은 상품, `role`은 보안 주체다.
4. `packages/billing`의 엔타이틀먼트 조회가 최소 1개 수직 슬라이스에서 사용되어야 한다.

## Gate 2. Stripe Readiness

1. Checkout 세션 생성 경로가 정의되어야 한다.
2. Webhook 이벤트 수신 경로가 정의되어야 한다.
3. Billing webhook 실패가 재처리 가능해야 한다.
4. Stripe secret, webhook secret, publishable key는 환경 변수로만 주입되어야 한다.

## Gate 3. Reconciliation

1. 구독 상태와 엔타이틀먼트 상태가 재동기화 가능해야 한다.
2. 이벤트 중복 수신 시 중복 과금이나 중복 활성화가 발생하지 않아야 한다.
3. 가격 변경이나 플랜 변경 시 기존 구독의 상태 전이가 문서화되어야 한다.

## Gate 4. Observability

1. Billing event lifecycle 로그가 남아야 한다.
2. Checkout 성공, 실패, webhook 수신, reconciliation 결과를 구분해야 한다.
3. 운영자가 quota 또는 billing 이상징후를 추적할 수 있어야 한다.

## Gate 5. Release Approval

1. `pricing matrix`가 고정되어야 한다.
2. `checkout`과 `webhook` placeholder가 실제 구현으로 교체될 준비가 되어야 한다.
3. `api-platform-engineer`의 entitlement middleware와 계약이 맞아야 한다.
4. `ops-admin-engineer`의 운영 대시보드와 runbook이 연결되어야 한다.

## Exit Criteria

1. 최소 1개 유료 플랜 결제 경로가 end-to-end로 검증된다.
2. Free/Pro 기능 차등이 코드로 강제된다.
3. 배포 전 gate 상태가 `pass` 또는 `warn`으로 정리되고, `block`이 남아 있지 않아야 한다.
