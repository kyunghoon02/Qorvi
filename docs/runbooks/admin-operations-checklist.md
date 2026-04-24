# Admin Operations Checklist

이 문서는 Qorvi `/admin` 콘솔을 기준으로 실제 운영자가 확인해야 하는 항목을 정리한 체크리스트다. 목표는 "페이지가 열린다"가 아니라, 유료 production 제품으로서 운영자가 release 전후에 무엇을 확인해야 하는지 명확하게 만드는 것이다.

관련 화면:

- admin web route: `/admin`
- admin API routes: `/v1/admin/*`
- admin surface source: [apps/web/app/admin/admin-console-screen.tsx](apps/web/app/admin/admin-console-screen.tsx)

관련 runbook:

- admin ops baseline: [docs/runbooks/ops-admin.md](docs/runbooks/ops-admin.md)
- Dune backtest collection: [docs/runbooks/dune-backtest-collection.md](docs/runbooks/dune-backtest-collection.md)
- production release package: [docs/runbooks/production-release-package.md](docs/runbooks/production-release-package.md)

## 1. Access Gate

출근 직후 또는 release 전 가장 먼저 확인할 것:

1. `/admin` 접근이 allowlist 계정으로만 되는지 확인한다.
2. 일반 사용자 세션으로는 `/admin`이 보이지 않는지 확인한다.
3. 운영 계정의 Clerk role이 `admin`으로 유지되는지 확인한다.
4. allowlist env가 production에 정확히 들어가 있는지 확인한다.

필수 env:

- `QORVI_ADMIN_ALLOWLIST_USER_IDS`
- `QORVI_ADMIN_ALLOWLIST_EMAILS`

## 2. Launch-Day Preflight

release 직전 `/admin`에서 반드시 확인할 것:

1. `Backtest Ops`
2. `Provider quotas`
3. `Observability`
4. `Suppressions`
5. `Curated lists`
6. `Audit logs`

go/no-go 기준:

1. backtest release-gate가 모두 성공한다.
2. provider quota가 critical 또는 exhausted 상태가 아니다.
3. ingest freshness가 unhealthy하지 않다.
4. recent failures가 explainable하고 blast-radius가 제한적이다.
5. false positive suppressions가 누락되지 않았다.
6. 최근 label/suppression 조작이 audit log에 남아 있다.

## 3. Daily Operator Loop

매일 `/admin`에서 반복할 것:

### Backtest Ops

1. `Analysis benchmark fixture`를 실행한다.
2. `Backtest manifest validate`를 실행한다.
3. `Dune query presets validate`를 실행한다.
4. reviewed candidate export가 있으면 `Dune candidate export validate`를 실행한다.
5. Dune saved query가 연결된 preset은 `Fetch Dune ...`를 눌러 최신 candidate export를 갱신한다.

실패 시:

1. 실패한 check key와 timestamp를 기록한다.
2. input path, preset, query ID, env 누락 여부를 확인한다.
3. 같은 failure를 suppressions나 labels로 덮지 않는다. 먼저 원인을 분리한다.

### Provider Quotas

1. `used + reserved`가 경고 구간으로 올라갔는지 확인한다.
2. warning이면 non-essential enrichment를 줄일지 판단한다.
3. critical이면 collection cadence와 enrichment worker를 제한한다.
4. exhausted면 release candidate replay와 bulk refresh를 중단한다.

### Observability

1. ingest freshness가 정상인지 확인한다.
2. provider usage error rate가 급증하지 않았는지 확인한다.
3. recent runs 중 failed가 반복되는 worker가 있는지 확인한다.
4. recent failures의 source/kind가 한 provider에 쏠리는지 확인한다.
5. alert delivery 실패가 spike인지 구조적 실패인지 구분한다.

### Suppressions

1. 최근 false positive가 suppression 없이 반복 발생하지 않는지 본다.
2. 임시 suppression은 expiry가 붙어 있는지 확인한다.
3. suppression reason이 analyst review 가능한 문장인지 확인한다.
4. release 전에 오래된 suppression이 product debt인지 운영상 필요한지 분류한다.

### Curated Lists

1. watchlist / curated entity set이 비어 있지 않은지 확인한다.
2. production focus cohort가 현재 discovery와 맞는지 검토한다.
3. 새로 중요한 entity가 발견됐으면 curated list에 올릴지 판단한다.

### Audit Logs

1. 모든 수동 override가 audit trail로 남았는지 본다.
2. actor/action/target이 누락된 로그가 없는지 확인한다.
3. release 당일엔 label/suppression 변경 건을 별도로 캡처한다.

## 4. Weekly Release Readiness

주간 release 전 `/admin`에서 확인할 것:

1. benchmark fixture가 연속 통과하는지 본다.
2. reviewed backtest manifest가 최신인지 확인한다.
3. 최신 Dune candidate export가 validation을 통과하는지 본다.
4. representative wallet set에서 stale refresh가 실제로 잘 돌았는지 본다.
5. active/pending subscription 비율을 확인한다.
6. provider error rate와 quota headroom이 release 주간 traffic을 견딜지 판단한다.

release blocker:

1. manifest validation 실패
2. reviewed dataset 부족
3. ingest freshness 저하
4. provider exhausted 상태
5. audit trail gap

## 5. Dune Candidate Collection Workflow

`/admin`에서 수동으로 돌릴 때의 표준 순서:

1. `Dune query presets validate`
2. 각 preset별 `Fetch Dune ...`
3. output candidate export를 analyst가 검토
4. `Dune candidate export validate`
5. reviewed dataset을 promotion flow로 넘김
6. `Backtest manifest validate`

원칙:

1. `Fetch Dune ...`는 candidate sourcing 단계일 뿐이다.
2. fetched candidate를 자동으로 manifest로 승격하지 않는다.
3. reviewer 없는 candidate export는 release evidence로 쓰지 않는다.

## 6. Incident Triage Sequence

문제가 생기면 `/admin` 기준으로 이 순서로 본다.

1. `Observability`
2. `Provider quotas`
3. `Audit logs`
4. `Suppressions`
5. `Backtest Ops`

판단 기준:

1. ingest lag면 pipeline 문제다.
2. quota warning이면 capacity 문제다.
3. recent failures가 동일 provider에 몰리면 vendor/API 문제일 가능성이 높다.
4. repeated false positive면 scoring/suppression 문제다.
5. backtest gate failure면 release 품질 문제다.

## 7. Minimum Evidence to Save

운영자가 release 전에 남겨야 하는 evidence:

1. `/admin`의 `Backtest Ops` 최신 성공 스냅샷
2. `/admin`의 `Provider quotas` 스냅샷
3. `/admin`의 `Observability` 스냅샷
4. `/admin`의 최근 suppression / label audit 스냅샷
5. reviewed backtest dataset count와 cohort 분포

## 8. Not Good Enough

아래 상태는 production 운영 기준에서 부족하다고 본다.

1. benchmark만 통과하고 reviewed real-world manifest가 비어 있음
2. false positive가 suppression에만 의존하고 root-cause fix가 없음
3. provider quota warning이 장기간 지속됨
4. active subscription 없이 pending만 쌓여 있음
5. operator action이 audit에 남지 않음

## 9. Production Bar

`/admin`은 단순 운영 툴이 아니라 release gate surface로 취급한다. 운영자가 이 화면만 보고 아래를 판단할 수 있어야 한다.

1. 지금 배포해도 되는가
2. 지금 들어오는 finding을 믿어도 되는가
3. false positive가 늘고 있는가
4. data pipeline이 stale해졌는가
5. backtest evidence가 이번 release를 지지하는가
