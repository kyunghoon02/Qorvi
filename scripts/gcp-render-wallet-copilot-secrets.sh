#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${1:?usage: gcp-render-wallet-copilot-secrets.sh <gcp-project-id> [output-file]}"
OUTPUT_FILE="${2:-.env.wallet-secrets}"
TEMP_FILE="$(mktemp)"
trap 'rm -f "${TEMP_FILE}"' EXIT

secret_value() {
  local env_name="$1"
  local secret_name="$2"
  if value="$(gcloud secrets versions access latest --project="${PROJECT_ID}" --secret="${secret_name}" 2>/dev/null)"; then
    printf '%s=%s\n' "${env_name}" "${value}" >>"${TEMP_FILE}"
  fi
}

secret_value ETHERSCAN_API_KEY qorvi-etherscan-api-key
secret_value ALCHEMY_API_KEY qorvi-alchemy-api-key
secret_value OPENAI_API_KEY qorvi-openai-api-key
secret_value QORVI_WALLET_WORKER_SECRET qorvi-wallet-worker-secret
secret_value CRON_SECRET qorvi-wallet-cron-secret
secret_value DUNE_API_KEY qorvi-dune-api-key
secret_value HELIUS_API_KEY qorvi-helius-api-key
secret_value MORALIS_API_KEY qorvi-moralis-api-key
secret_value AUTH_SECRET qorvi-auth-secret
secret_value CLERK_SECRET_KEY qorvi-clerk-secret-key
secret_value QORVI_POSTGRES_PASSWORD qorvi-postgres-password
secret_value POSTGRES_URL qorvi-postgres-url
secret_value QORVI_NEO4J_PASSWORD qorvi-neo4j-password
secret_value NEO4J_PASSWORD qorvi-neo4j-password

if ! grep -q '^ETHERSCAN_API_KEY=' "${TEMP_FILE}" &&
  ! grep -q '^ALCHEMY_API_KEY=' "${TEMP_FILE}"; then
  echo "No live Ethereum provider secret version could be loaded." >&2
  exit 1
fi

install -m 600 "${TEMP_FILE}" "${OUTPUT_FILE}"
echo "Rendered Wallet Copilot runtime secrets to ${OUTPUT_FILE}."
