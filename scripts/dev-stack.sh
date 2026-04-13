#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/infra/docker/docker-compose.yml"
DEFAULT_WORKER_MODES=("curated-wallet-seed-enqueue" "wallet-backfill-drain-priority" "wallet-backfill-drain-batch" "wallet-tracking-subscription-sync")
declare -a WORKER_MODES=()
declare -a WORKER_PIDS=()
RUN_WORKER="true"
EXPLICIT_WORKER_MODES="false"

cd "$ROOT_DIR"

usage() {
  cat <<'EOF'
Usage: ./scripts/dev-stack.sh [--no-worker] [--worker-mode MODE]

Options:
  --no-worker          Start only docker infra + api + web
  --worker-mode MODE   Add a worker mode for this run (repeatable)
  --help               Show this message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-worker)
      RUN_WORKER="false"
      shift
      ;;
    --worker-mode)
      worker_mode="${2:-}"
      if [[ -z "$worker_mode" ]]; then
        echo "--worker-mode requires a value" >&2
        exit 1
      fi
      EXPLICIT_WORKER_MODES="true"
      WORKER_MODES+=("$worker_mode")
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

  existing_dbs="$(docker compose -f "$COMPOSE_FILE" exec -T postgres psql -U postgres -d postgres -tAc "SELECT datname FROM pg_database ORDER BY datname;" 2>/dev/null || true)"
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

cleanup() {
  local exit_code=$?

  for pid in ${WEB_PID:-} ${API_PID:-} "${WORKER_PIDS[@]:-}"; do
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" >/dev/null 2>&1; then
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done

  wait ${WEB_PID:-} ${API_PID:-} "${WORKER_PIDS[@]:-}" >/dev/null 2>&1 || true
  exit "$exit_code"
}

trap cleanup EXIT INT TERM

append_worker_mode_once() {
  local candidate="$1"
  local existing

  for existing in "${WORKER_MODES[@]}"; do
    if [[ "$existing" == "$candidate" ]]; then
      return 0
    fi
  done

  WORKER_MODES+=("$candidate")
}

wait_for_any() {
  local pids=("$@")

  while true; do
    local active=0

    for pid in "${pids[@]}"; do
      if [[ -z "${pid:-}" ]]; then
        continue
      fi

      if kill -0 "$pid" >/dev/null 2>&1; then
        active=1
      else
        wait "$pid" >/dev/null 2>&1 || true
        return 0
      fi
    done

    if [[ "$active" -eq 0 ]]; then
      return 0
    fi

    sleep 1
  done
}

stop_qorvi_listener_on_port() {
  local port="$1"
  local pids

  pids="$(lsof -nP -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -z "$pids" ]]; then
    return 0
  fi

  local pid
  for pid in $pids; do
    local cwd_info
    cwd_info="$(lsof -a -p "$pid" -d cwd -Fn 2>/dev/null || true)"
    if printf '%s\n' "$cwd_info" | grep -Fq "n$ROOT_DIR"; then
      echo "Stopping stale Qorvi listener on port $port (pid=$pid)..."
      kill "$pid" >/dev/null 2>&1 || true
      sleep 1
    else
      echo "Port $port is already used by a non-Qorvi process (pid=$pid)."
    fi
  done
}

echo "Starting local infra via docker compose..."
docker compose -f "$COMPOSE_FILE" up -d

POSTGRES_URL="$(resolve_existing_postgres_url "$POSTGRES_URL")"
export POSTGRES_URL

if [[ "${#WORKER_MODES[@]}" -eq 0 ]]; then
  if [[ -n "${QORVI_DEV_STACK_WORKER_MODES:-}" ]]; then
    IFS=',' read -r -a WORKER_MODES <<<"$QORVI_DEV_STACK_WORKER_MODES"
  elif [[ -n "${FLOWINTEL_DEV_STACK_WORKER_MODES:-}" ]]; then
    IFS=',' read -r -a WORKER_MODES <<<"$FLOWINTEL_DEV_STACK_WORKER_MODES"
  elif [[ -n "${QORVI_DEV_STACK_WORKER_MODE:-}" ]]; then
    WORKER_MODES=("$QORVI_DEV_STACK_WORKER_MODE")
  elif [[ -n "${FLOWINTEL_DEV_STACK_WORKER_MODE:-}" ]]; then
    WORKER_MODES=("$FLOWINTEL_DEV_STACK_WORKER_MODE")
  else
    WORKER_MODES=("${DEFAULT_WORKER_MODES[@]}")
  fi
fi

for i in "${!WORKER_MODES[@]}"; do
  WORKER_MODES[$i]="$(printf '%s' "${WORKER_MODES[$i]}" | xargs)"
done

if [[ "$EXPLICIT_WORKER_MODES" != "true" ]] && [[ -n "${MOBULA_API_KEY:-}" ]] && [[ -n "${QORVI_MOBULA_SMART_MONEY_SEEDS_JSON:-${MOBULA_SMART_MONEY_SEEDS_JSON:-}}" ]]; then
  append_worker_mode_once "mobula-smart-money-enqueue"
fi

echo "Applying local migrations..."
./scripts/dev-migrate.sh

stop_qorvi_listener_on_port 4000
stop_qorvi_listener_on_port 3000

echo "Starting API on http://localhost:4000 ..."
corepack pnpm dev:api &
API_PID=$!

if [[ "$RUN_WORKER" == "true" ]]; then
  for worker_mode in "${WORKER_MODES[@]}"; do
    if [[ -z "$worker_mode" ]]; then
      continue
    fi
    worker_interval=""
    if [[ "$worker_mode" == "mobula-smart-money-enqueue" ]]; then
      worker_interval="${QORVI_DEV_STACK_MOBULA_INTERVAL_SECONDS:-${FLOWINTEL_DEV_STACK_MOBULA_INTERVAL_SECONDS:-1800}}"
    elif [[ "$worker_mode" == "curated-wallet-seed-enqueue" ]]; then
      worker_interval="${QORVI_DEV_STACK_CURATED_SEED_INTERVAL_SECONDS:-21600}"
    fi

    if [[ -n "$worker_interval" ]]; then
      echo "Starting worker loop mode: $worker_mode (interval=${worker_interval}s) ..."
      ./scripts/dev-worker-loop.sh --worker-mode "$worker_mode" --interval "$worker_interval" &
    else
      echo "Starting worker loop mode: $worker_mode ..."
      ./scripts/dev-worker-loop.sh --worker-mode "$worker_mode" &
    fi
    WORKER_PIDS+=("$!")
  done
fi

echo "Starting web on http://localhost:3000 ..."
corepack pnpm dev:web &
WEB_PID=$!

echo "Qorvi dev stack is starting."
if [[ "$RUN_WORKER" == "true" ]]; then
  echo "Worker modes: $(IFS=', '; printf '%s' "${WORKER_MODES[*]}") (loop)"
else
  echo "Worker mode: disabled"
fi
echo "Web:   http://localhost:3000"
echo "API:   http://localhost:4000"
echo "Health: http://localhost:4000/healthz"

if [[ "$RUN_WORKER" == "true" ]]; then
  wait_for_any "$API_PID" "${WORKER_PIDS[@]}" "$WEB_PID"
else
  wait_for_any "$API_PID" "$WEB_PID"
fi
