#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULT_WORKER_MODE="wallet-backfill-drain-batch"
WORKER_MODE_VALUE="$DEFAULT_WORKER_MODE"
INTERVAL_SECONDS="${FLOWINTEL_DEV_STACK_WORKER_INTERVAL_SECONDS:-5}"

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

export POSTGRES_URL="${FLOWINTEL_LOCAL_POSTGRES_URL:-postgres://postgres:postgres@localhost:5433/flowintel}"
export NEO4J_URL="${FLOWINTEL_LOCAL_NEO4J_URL:-bolt://localhost:8687}"
export NEO4J_USERNAME="${FLOWINTEL_LOCAL_NEO4J_USERNAME:-neo4j}"
export NEO4J_PASSWORD="${FLOWINTEL_LOCAL_NEO4J_PASSWORD:-neo4jpassword}"
export REDIS_URL="${FLOWINTEL_LOCAL_REDIS_URL:-redis://localhost:6379}"

if [[ -n "${FLOWINTEL_DEV_STACK_WORKER_MODE:-}" ]]; then
  WORKER_MODE_VALUE="$FLOWINTEL_DEV_STACK_WORKER_MODE"
fi

export FLOWINTEL_WORKER_MODE="$WORKER_MODE_VALUE"

echo "Starting dev worker loop (mode=$FLOWINTEL_WORKER_MODE, interval=${INTERVAL_SECONDS}s)..."

while true; do
  if ! corepack pnpm dev:workers; then
    echo "Worker run failed for mode=$FLOWINTEL_WORKER_MODE; retrying in ${INTERVAL_SECONDS}s..." >&2
  fi

  sleep "$INTERVAL_SECONDS"
done
