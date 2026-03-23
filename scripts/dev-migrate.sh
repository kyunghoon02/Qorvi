#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/infra/docker/docker-compose.yml"
POSTGRES_USER="postgres"
POSTGRES_DB="whalegraph"
NEO4J_USER="neo4j"
NEO4J_PASSWORD="neo4jpassword"
NEO4J_DB="neo4j"

usage() {
  cat <<'EOF'
Usage: ./scripts/dev-migrate.sh

Applies local Postgres and Neo4j migrations using the docker compose stack.
EOF
}

if [[ "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

cd "$ROOT_DIR"

wait_for_postgres() {
  until docker compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; do
    sleep 1
  done
}

wait_for_neo4j() {
  until docker compose -f "$COMPOSE_FILE" exec -T neo4j cypher-shell -u "$NEO4J_USER" -p "$NEO4J_PASSWORD" -d "$NEO4J_DB" "RETURN 1" >/dev/null 2>&1; do
    sleep 1
  done
}

apply_postgres_migrations() {
  local file
  for file in "$ROOT_DIR"/infra/migrations/postgres/*.sql; do
    echo "Applying Postgres migration: $(basename "$file")"
    docker compose -f "$COMPOSE_FILE" exec -T postgres \
      psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f - < "$file"
  done
}

apply_neo4j_migrations() {
  local file
  for file in "$ROOT_DIR"/infra/migrations/neo4j/*.cypher; do
    echo "Applying Neo4j migration: $(basename "$file")"
    docker compose -f "$COMPOSE_FILE" exec -T neo4j \
      cypher-shell -u "$NEO4J_USER" -p "$NEO4J_PASSWORD" -d "$NEO4J_DB" < "$file"
  done
}

echo "Waiting for Postgres..."
wait_for_postgres
echo "Waiting for Neo4j..."
wait_for_neo4j

apply_postgres_migrations
apply_neo4j_migrations

echo "Local migrations applied."
