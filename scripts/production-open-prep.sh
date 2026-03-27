#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="full"

usage() {
  cat <<'EOF'
Usage: ./scripts/production-open-prep.sh [--env-only] [--env-file <path>]

Options:
  --env-only         Validate required production env values only
  --env-file <path>  Load env values from a specific file instead of .env
  --help             Show this message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-only)
      MODE="env-only"
      shift
      ;;
    --env-file)
      if [[ $# -lt 2 ]]; then
        echo "--env-file requires a path" >&2
        exit 1
      fi
      ENV_FILE="$2"
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

ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ENV_FILE"
  set +a
fi

pass_count=0
warn_count=0
block_count=0

report() {
  local status="$1"
  local key="$2"
  local note="$3"

  printf '%-5s %s' "$status" "$key"
  if [[ -n "$note" ]]; then
    printf ' (%s)' "$note"
  fi
  printf '\n'
}

is_placeholder() {
  local value="${1:-}"
  local normalized

  normalized="$(printf '%s' "$value" | tr '[:upper:]' '[:lower:]')"
  [[ -z "$normalized" || "$normalized" == "replace-me" || "$normalized" == replace-me-* || "$normalized" == "example" || "$normalized" == "changeme" ]]
}

check_required() {
  local key="$1"
  local value="${!key:-}"

  if is_placeholder "$value"; then
    report "BLOCK" "$key" "missing or placeholder"
    block_count=$((block_count + 1))
    return 1
  fi

  report "PASS" "$key" ""
  pass_count=$((pass_count + 1))
  return 0
}

check_optional() {
  local key="$1"
  local value="${!key:-}"

  if is_placeholder "$value"; then
    report "WARN" "$key" "unset"
    warn_count=$((warn_count + 1))
    return 1
  fi

  report "PASS" "$key" ""
  pass_count=$((pass_count + 1))
  return 0
}

check_same_origin() {
  local lhs_key="$1"
  local rhs_key="$2"
  local lhs_value="${!lhs_key:-}"
  local rhs_value="${!rhs_key:-}"

  if is_placeholder "$lhs_value" || is_placeholder "$rhs_value"; then
    return 0
  fi

  local lhs_origin="$lhs_value"
  lhs_origin="${lhs_origin%%\?*}"
  lhs_origin="${lhs_origin%%#*}"
  if [[ "$lhs_origin" =~ ^https?://[^/]+ ]]; then
    lhs_origin="${BASH_REMATCH[0]}"
  fi

  if [[ "$rhs_value" != "$lhs_origin"* ]]; then
    report "BLOCK" "$lhs_key" "origin must match $rhs_key"
    block_count=$((block_count + 1))
    return 1
  fi

  report "PASS" "$lhs_key" "origin matches $rhs_key"
  pass_count=$((pass_count + 1))
  return 0
}

echo "FlowIntel Production Open Prep"
echo "Date: $(date '+%Y-%m-%d %H:%M:%S %Z')"
echo "Repo: $ROOT_DIR"
echo "Env file: $ENV_FILE"
echo

echo "[1/3] Env sanity"
required_keys=(
  APP_BASE_URL
  NEXT_PUBLIC_APP_BASE_URL
  NEXT_PUBLIC_API_BASE_URL
  API_HOST
  API_PORT
  POSTGRES_URL
  NEO4J_URL
  NEO4J_USERNAME
  NEO4J_PASSWORD
  REDIS_URL
  FLOWINTEL_RAW_PAYLOAD_ROOT
  AUTH_PROVIDER
  AUTH_SECRET
  CLERK_SECRET_KEY
  CLERK_AUDIENCE
  NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY
  ALCHEMY_API_KEY
  HELIUS_API_KEY
  MORALIS_API_KEY
  ALCHEMY_BASE_URL
  ALCHEMY_SOLANA_BASE_URL
  HELIUS_BASE_URL
  HELIUS_DATA_API_BASE_URL
  MORALIS_BASE_URL
)

for key in "${required_keys[@]}"; do
  check_required "$key" || true
done

optional_keys=(
  DUNE_API_KEY
  STRIPE_SECRET_KEY
  STRIPE_WEBHOOK_SECRET
  STRIPE_PUBLISHABLE_KEY
  STRIPE_SUCCESS_URL
  STRIPE_CANCEL_URL
  STRIPE_BASE_URL
  FLOWINTEL_ALERT_SMTP_HOST
)

for key in "${optional_keys[@]}"; do
  check_optional "$key" || true
done

check_same_origin "STRIPE_SUCCESS_URL" "APP_BASE_URL" || true
check_same_origin "STRIPE_CANCEL_URL" "APP_BASE_URL" || true

if [[ "$MODE" == "env-only" ]]; then
  echo
  echo "Summary: PASS=$pass_count WARN=$warn_count BLOCK=$block_count"
  if [[ "$block_count" -gt 0 ]]; then
    echo "Next: fix BLOCK env values before production launch."
    exit 1
  fi
  echo "Next: run corepack pnpm prod:prep && corepack pnpm prod:evidence"
  exit 0
fi

if [[ "$block_count" -gt 0 ]]; then
  echo
  echo "Summary: PASS=$pass_count WARN=$warn_count BLOCK=$block_count"
  echo "Next: fix BLOCK env values before production launch."
  exit 1
fi

echo
echo "[2/3] Prep"
corepack pnpm prod:prep

echo
echo "[3/3] Evidence"
corepack pnpm prod:evidence

echo
echo "Launch summary"
echo "PASS  Ready for production launch"
if [[ "$warn_count" -gt 0 ]]; then
  echo "WARN  Optional values remain unset; review production release package before opening."
fi
echo "Next:"
echo "- Review docs/runbooks/production-launch-review.md"
echo "- Review docs/runbooks/production-open-prep.md"
echo "- Confirm operator handoff is complete"
