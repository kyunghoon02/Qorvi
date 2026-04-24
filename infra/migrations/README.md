# Migration Bootstrap

이 디렉터리는 Sprint 0/1 데이터 플랫폼 부트스트랩을 위한 초기 마이그레이션과 그래프 bootstrap을 담는다.

## Layout

- `postgres/`: Postgres schema bootstrap SQL
- `neo4j/`: Neo4j constraint/index bootstrap Cypher. Graph relationships live here, not in Postgres.
- `redis/`: Redis는 스키마가 없으므로 keyspace 규약과 운영 메모만 둔다

## Seed / slice bootstrap

- `postgres/0002_wallet_summary_seed.sql`: `wallet_summary` 첫 슬라이스 검증용 local seed wallet과 일간 stats
- `neo4j/0002_wallet_summary_seed.cypher`: local seed wallet의 cluster/member 관계와 graph signal 샘플

실제 `packages/db` reader는 `postgres/0001_init.sql`과 `neo4j/0001_constraints.cypher`가 만드는 스키마를 읽는다. `0002_*` 파일은 로컬 개발/스모크 테스트용 fixture다.

## Local bootstrap

1. `docker compose -f infra/docker/docker-compose.yml up -d`
2. Postgres bootstrap 실행
3. Neo4j bootstrap 실행
4. 애플리케이션의 `packages/db` 연결 문자열을 로컬 `.env` 값에 맞춘다
