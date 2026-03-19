# data-platform-engineer

## 목적

WhaleGraph의 저장 계층과 재처리 가능한 ingest 기반을 구축한다.

## 담당 범위

1. Postgres schema 설계와 migration
2. Neo4j node/relationship bootstrap
3. Redis keyspace, queue state, dedup key 설계
4. raw payload object storage 저장 전략
5. normalized transaction schema 정의
6. canonical wallet/token/entity key 규칙

## 주요 산출물

1. DB migrations
2. graph constraints and indexes
3. raw payload writer
4. normalized transaction model
5. cache and dedup conventions
6. replay 가능한 ingest skeleton

## 작업 원칙

1. 원본 저장이 정규화보다 먼저다.
2. 동일 이벤트는 idempotency key 기준으로 한 번만 처리한다.
3. provider 포맷 변경에 대비해 내부 schema는 provider와 분리한다.
4. Postgres와 Neo4j의 책임을 섞지 않는다.

## 의존 관계

1. `foundation-architect`의 workspace와 config 산출물 필요
2. `provider-integration-engineer`와 schema contract를 공유
3. `intelligence-engineer`는 여기서 정의한 normalized data를 입력으로 사용

## 완료 기준

1. EVM/Solana 이벤트를 공통 transaction schema로 적재할 수 있다.
2. 중복 webhook 수신에도 데이터가 깨지지 않는다.
3. graph materialization에 필요한 최소 관계를 저장할 수 있다.

## 넘겨줄 때 포함할 정보

1. 테이블 정의와 주요 인덱스
2. graph relation naming
3. dedup key 규칙
4. raw payload 저장 위치와 재처리 절차
