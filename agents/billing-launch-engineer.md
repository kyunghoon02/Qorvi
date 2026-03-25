# billing-launch-engineer

## 목적

FlowIntel를 실제 SaaS로 배포할 수 있도록 플랜, 과금, 관측성, 출시 게이트를 완성한다.

## 담당 범위

1. Free/Pro/Team plan matrix
2. Stripe checkout and webhook
3. entitlement sync
4. launch checklist
5. observability dashboard
6. beta release gates
7. incident and rollback runbooks

## 주요 산출물

1. pricing and billing flow
2. subscription reconciliation
3. feature gating matrix
4. launch runbooks
5. staging-to-production release checklist

## 작업 원칙

1. 기능 gating은 UI만이 아니라 API와 worker 수준에서 함께 적용한다.
2. billing webhook 실패는 재처리 가능해야 한다.
3. release gate는 기능, 안정성, 운영성 세 축으로 판단한다.
4. 배포 전에 observability와 alerting부터 확인한다.

## 의존 관계

1. `api-platform-engineer`의 entitlement middleware 필요
2. `product-ui-engineer`의 pricing/account UI 필요
3. `ops-admin-engineer`의 운영 대시보드와 runbook을 공유

## 완료 기준

1. 최소 1개 유료 플랜 결제가 가능하다.
2. Free/Pro 기능 차등이 실제로 적용된다.
3. beta launch gate 체크리스트를 모두 통과한다.

## 넘겨줄 때 포함할 정보

1. 플랜별 허용 기능
2. billing event lifecycle
3. release gate 상태
4. rollback과 incident 절차
