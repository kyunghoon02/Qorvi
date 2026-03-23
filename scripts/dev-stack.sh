#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/infra/docker/docker-compose.yml"
DEFAULT_WORKER_MODE="wallet-backfill-drain-batch"
WORKER_MODE_VALUE="$DEFAULT_WORKER_MODE"
RUN_WORKER="true"

cd "$ROOT_DIR"

usage() {
  cat <<'EOF'
Usage: ./scripts/dev-stack.sh [--no-worker] [--worker-mode MODE]

Options:
  --no-worker          Start only docker infra + api + web
  --worker-mode MODE   Override WHALEGRAPH_WORKER_MODE for this run
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
      WORKER_MODE_VALUE="${2:-}"
      if [[ -z "$WORKER_MODE_VALUE" ]]; then
        echo "--worker-mode requires a value" >&2
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

if [[ -f "$ROOT_DIR/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT_DIR/.env"
  set +a
fi

export POSTGRES_URL="${WHALEGRAPH_LOCAL_POSTGRES_URL:-postgres://postgres:postgres@localhost:5433/whalegraph}"
export NEO4J_URL="${WHALEGRAPH_LOCAL_NEO4J_URL:-bolt://localhost:8687}"
export NEO4J_USERNAME="${WHALEGRAPH_LOCAL_NEO4J_USERNAME:-neo4j}"
export NEO4J_PASSWORD="${WHALEGRAPH_LOCAL_NEO4J_PASSWORD:-neo4jpassword}"
export REDIS_URL="${WHALEGRAPH_LOCAL_REDIS_URL:-redis://localhost:6379}"
export WHALEGRAPH_DEV_STACK_WORKER_MODE="$WORKER_MODE_VALUE"

cleanup() {
  local exit_code=$?

  for pid in ${WEB_PID:-} ${WORKER_PID:-} ${API_PID:-}; do
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" >/dev/null 2>&1; then
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done

  wait ${WEB_PID:-} ${WORKER_PID:-} ${API_PID:-} >/dev/null 2>&1 || true
  exit "$exit_code"
}

trap cleanup EXIT INT TERM

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

stop_whalegraph_listener_on_port() {
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
      echo "Stopping stale WhaleGraph listener on port $port (pid=$pid)..."
      kill "$pid" >/dev/null 2>&1 || true
      sleep 1
    else
      echo "Port $port is already used by a non-WhaleGraph process (pid=$pid)."
    fi
  done
}

echo "Starting local infra via docker compose..."
docker compose -f "$COMPOSE_FILE" up -d

echo "Applying local migrations..."
./scripts/dev-migrate.sh

stop_whalegraph_listener_on_port 4000
stop_whalegraph_listener_on_port 3000

echo "Starting API on http://localhost:4000 ..."
corepack pnpm dev:api &
API_PID=$!

if [[ "$RUN_WORKER" == "true" ]]; then
  echo "Starting worker loop mode: $WHALEGRAPH_DEV_STACK_WORKER_MODE ..."
  ./scripts/dev-worker-loop.sh --worker-mode "$WHALEGRAPH_DEV_STACK_WORKER_MODE" &
  WORKER_PID=$!
fi

echo "Starting web on http://localhost:3000 ..."
corepack pnpm dev:web &
WEB_PID=$!

echo "WhaleGraph dev stack is starting."
if [[ "$RUN_WORKER" == "true" ]]; then
  echo "Worker mode: $WHALEGRAPH_DEV_STACK_WORKER_MODE (loop)"
else
  echo "Worker mode: disabled"
fi
echo "Web:   http://localhost:3000"
echo "API:   http://localhost:4000"
echo "Health: http://localhost:4000/healthz"

if [[ "$RUN_WORKER" == "true" ]]; then
  wait_for_any "$API_PID" "$WORKER_PID" "$WEB_PID"
else
  wait_for_any "$API_PID" "$WEB_PID"
fi
