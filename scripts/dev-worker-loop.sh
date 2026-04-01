#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULT_WORKER_MODE="wallet-backfill-drain-batch"
WORKER_MODE_VALUE="$DEFAULT_WORKER_MODE"
EXPLICIT_WORKER_MODE="false"
INTERVAL_SECONDS="${QORVI_DEV_STACK_WORKER_INTERVAL_SECONDS:-${FLOWINTEL_DEV_STACK_WORKER_INTERVAL_SECONDS:-5}}"

usage() {
  cat <<'EOF'
Usage: ./scripts/dev-worker-loop.sh [--worker-mode MODE] [--interval SECONDS]

Options:
  --worker-mode MODE   Worker mode to run in a restart loop
  --interval SECONDS   Delay between worker runs (default: 5)
  --help               Show this message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --worker-mode)
      WORKER_MODE_VALUE="${2:-}"
      if [[ -z "$WORKER_MODE_VALUE" ]]; then
        echo "--worker-mode requires a value" >&2
        exit 1
      fi
      EXPLICIT_WORKER_MODE="true"
      shift 2
      ;;
    --interval)
      INTERVAL_SECONDS="${2:-}"
      if [[ -z "$INTERVAL_SECONDS" ]]; then
        echo "--interval requires a value" >&2
        exit 1
      fi
      shift 2
      ;;
    --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

cd "$ROOT_DIR"

if [[ -f "$ROOT_DIR/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT_DIR/.env"
  set +a
fi

export POSTGRES_URL="${POSTGRES_URL:-${QORVI_LOCAL_POSTGRES_URL:-${FLOWINTEL_LOCAL_POSTGRES_URL:-postgres://postgres:postgres@localhost:5433/qorvi}}}"
export NEO4J_URL="${NEO4J_URL:-${QORVI_LOCAL_NEO4J_URL:-${FLOWINTEL_LOCAL_NEO4J_URL:-bolt://localhost:8687}}}"
export NEO4J_USERNAME="${NEO4J_USERNAME:-${QORVI_LOCAL_NEO4J_USERNAME:-${FLOWINTEL_LOCAL_NEO4J_USERNAME:-neo4j}}}"
export NEO4J_PASSWORD="${NEO4J_PASSWORD:-${QORVI_LOCAL_NEO4J_PASSWORD:-${FLOWINTEL_LOCAL_NEO4J_PASSWORD:-neo4jpassword}}}"
export REDIS_URL="${REDIS_URL:-${QORVI_LOCAL_REDIS_URL:-${FLOWINTEL_LOCAL_REDIS_URL:-redis://localhost:6379}}}"

postgres_db_from_url() {
  local raw_url="$1"
  local db_name="${raw_url##*/}"
  db_name="${db_name%%\?*}"
  printf '%s' "$db_name"
}

resolve_existing_postgres_url() {
  local raw_url="$1"
  local preferred_db
  preferred_db="$(postgres_db_from_url "$raw_url")"
  local existing_dbs

  existing_dbs="$(docker compose -f "$ROOT_DIR/infra/docker/docker-compose.yml" exec -T postgres psql -U postgres -d postgres -tAc "SELECT datname FROM pg_database ORDER BY datname;" 2>/dev/null || true)"
  if printf '%s\n' "$existing_dbs" | grep -Fxq "$preferred_db"; then
    printf '%s' "$raw_url"
    return 0
  fi

  for legacy_db in qorvi whalegraph flowintel; do
    if printf '%s\n' "$existing_dbs" | grep -Fxq "$legacy_db"; then
      printf '%s' "${raw_url%/*}/$legacy_db"
      return 0
    fi
  done

  printf '%s' "$raw_url"
}

POSTGRES_URL="$(resolve_existing_postgres_url "$POSTGRES_URL")"
export POSTGRES_URL

if [[ "$EXPLICIT_WORKER_MODE" != "true" ]]; then
  if [[ -n "${QORVI_DEV_STACK_WORKER_MODE:-}" ]]; then
    WORKER_MODE_VALUE="$QORVI_DEV_STACK_WORKER_MODE"
  elif [[ -n "${FLOWINTEL_DEV_STACK_WORKER_MODE:-}" ]]; then
    WORKER_MODE_VALUE="$FLOWINTEL_DEV_STACK_WORKER_MODE"
  elif [[ -n "${QORVI_WORKER_MODE:-}" ]]; then
    WORKER_MODE_VALUE="$QORVI_WORKER_MODE"
  elif [[ -n "${FLOWINTEL_WORKER_MODE:-}" ]]; then
    WORKER_MODE_VALUE="$FLOWINTEL_WORKER_MODE"
  fi
fi

export QORVI_WORKER_MODE="$WORKER_MODE_VALUE"

echo "Starting dev worker loop (mode=$QORVI_WORKER_MODE, interval=${INTERVAL_SECONDS}s)..."

while true; do
  if ! corepack pnpm dev:workers; then
    echo "Worker run failed for mode=$QORVI_WORKER_MODE; retrying in ${INTERVAL_SECONDS}s..." >&2
  fi

  sleep "$INTERVAL_SECONDS"
done
