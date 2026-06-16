#!/usr/bin/env bash
set -euo pipefail

APP_DIR="/opt/qorvi/app"
COMPOSE_FILE="${APP_DIR}/infra/docker/docker-compose.prod.yml"
BASE_ENV="${APP_DIR}/.env.backend"
SECRET_ENV="${APP_DIR}/.env.wallet-secrets"

systemctl start docker
systemctl start nginx

if [[ ! -f "${COMPOSE_FILE}" || ! -f "${BASE_ENV}" || ! -f "${SECRET_ENV}" ]]; then
  echo "Qorvi runtime files are not ready; deploy before restarting containers." >&2
  exit 0
fi

chmod 600 "${BASE_ENV}" "${SECRET_ENV}"
cd "${APP_DIR}"
docker compose --env-file "${BASE_ENV}" --env-file "${SECRET_ENV}" \
  -f "${COMPOSE_FILE}" up -d postgres redis neo4j api web
