#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/infra/docker/docker-compose.yml"
POSTGRES_USER="postgres"
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

if [[ -f "$ROOT_DIR/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT_DIR/.env"
  set +a
fi

POSTGRES_URL_VALUE="${POSTGRES_URL:-${QORVI_LOCAL_POSTGRES_URL:-${FLOWINTEL_LOCAL_POSTGRES_URL:-postgres://postgres:postgres@localhost:5433/qorvi}}}"
POSTGRES_DB="${QORVI_LOCAL_POSTGRES_DB:-${FLOWINTEL_LOCAL_POSTGRES_DB:-${POSTGRES_URL_VALUE##*/}}}"
POSTGRES_DB="${POSTGRES_DB%%\?*}"

wait_for_postgres() {
  until docker compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U "$POSTGRES_USER" -d postgres >/dev/null 2>&1; do
    sleep 1
  done
}

postgres_db_exists() {
  local db_name="$1"
  docker compose -f "$COMPOSE_FILE" exec -T postgres \
    psql -U "$POSTGRES_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '${db_name}'" 2>/dev/null |
    grep -qx '1'
}

ensure_postgres_db() {
  local preferred_db="$POSTGRES_DB"

  if postgres_db_exists "$preferred_db"; then
    return 0
  fi

  for legacy_db in qorvi whalegraph flowintel; do
    if postgres_db_exists "$legacy_db"; then
      echo "Using existing Postgres database: $legacy_db"
      POSTGRES_DB="$legacy_db"
      return 0
    fi
  done

  echo "Creating missing Postgres database: $preferred_db"
  docker compose -f "$COMPOSE_FILE" exec -T postgres \
    psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d postgres \
    -c "CREATE DATABASE \"$preferred_db\";" >/dev/null
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
ensure_postgres_db
echo "Waiting for Neo4j..."
wait_for_neo4j

apply_postgres_migrations
apply_neo4j_migrations

echo "Local migrations applied."
