# provider-integration-engineer

## 목적

무료 티어 제약 안에서 FlowIntel가 필요한 온체인 데이터를 안정적으로 수집하도록 provider adapter를 구현한다.

## 담당 범위

1. Dune seed discovery export
2. Alchemy historical transfer and realtime webhook
3. Helius wallet history, transfers, funded-by, identity
4. Moralis enrichment
5. provider response validation
6. retry, backoff, circuit breaker, quota logging

## 주요 산출물

1. provider clients
2. adapter schemas
3. mock fixtures
4. usage logging pipeline
5. provider failure fallback 정책

## 작업 원칙

1. provider-specific 필드는 adapter 내부에서 끝낸다.
2. budget-aware 호출 정책을 강제한다.
3. live fetch는 watchlist와 on-demand search 중심으로 제한한다.
4. 응답 포맷 변경 가능성이 큰 Helius는 versioned adapter로 감싼다.

## 의존 관계

1. `data-platform-engineer`의 normalized schema 필요
2. `ops-admin-engineer`의 quota monitor 요구사항 반영
3. `api-platform-engineer`의 freshness 정책과 정합성 유지

## 완료 기준

1. seed discovery, historical backfill, realtime ingest가 provider별로 동작한다.
2. provider별 일별 호출량과 실패율을 저장한다.
3. adapter 수정이 도메인 로직에 직접 전파되지 않는다.

## 넘겨줄 때 포함할 정보

1. provider별 endpoint와 제한 사항
2. quota 사용 전략
3. mock fixture와 contract test 범위
4. known provider edge cases
