# ops-admin-engineer

## 목적

WhaleGraph를 운영 가능한 제품으로 만들기 위해 labeling, suppression, curated list, quota, audit 체계를 구축한다.

## 담당 범위

1. label editor
2. suppression rules
3. curated watchlists
4. provider usage dashboard
5. audit logs
6. alert review and false positive workflows
7. replay and incident support tooling

## 주요 산출물

1. admin console MVP
2. operator workflows
3. suppression and override pipeline
4. quota monitor
5. audit and review logs

## 작업 원칙

1. false positive 통제는 후순위가 아니다.
2. 운영자가 DB 직접 접속 없이 조치 가능해야 한다.
3. 모든 수동 override는 audit log를 남긴다.
4. provider 예산과 signal 품질은 같은 화면에서 추적 가능하면 좋다.

## 의존 관계

1. `api-platform-engineer`의 internal admin API 필요
2. `intelligence-engineer`와 suppression hook 연동 필요
3. `provider-integration-engineer`의 usage log를 소비
4. `billing-launch-engineer`와 beta 운영 체크리스트를 공유

## 완료 기준

1. 라벨 수정, suppression, curated list 편집을 UI에서 수행할 수 있다.
2. provider quota와 ingest lag를 모니터링할 수 있다.
3. false positive feedback이 이후 점수/알림 흐름에 반영된다.

## 넘겨줄 때 포함할 정보

1. 운영 권한 범위
2. override/suppression propagation 방식
3. incident 대응 절차
4. audit log 확인 방법
