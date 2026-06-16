#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-qorvi-493115}"
ZONE_NAME="${ZONE_NAME:-qorvi-app}"
DOMAIN="${1:-qorvi.app}"
TARGET_IP="${2:-34.87.143.25}"

DOMAIN="${DOMAIN%.}"
DNS_NAME="${DOMAIN}."

if ! command -v gcloud >/dev/null 2>&1; then
  echo "gcloud CLI is required to update GCP Cloud DNS records." >&2
  exit 127
fi

gcloud services enable dns.googleapis.com --project="${PROJECT_ID}" >/dev/null

if ! gcloud dns managed-zones describe "${ZONE_NAME}" \
  --project="${PROJECT_ID}" >/dev/null 2>&1; then
  gcloud dns managed-zones create "${ZONE_NAME}" \
    --project="${PROJECT_ID}" \
    --dns-name="${DNS_NAME}" \
    --description="Qorvi public DNS zone on GCP" \
    --visibility=public
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

(
  cd "${TMP_DIR}"
  gcloud dns record-sets transaction start \
    --zone="${ZONE_NAME}" \
    --project="${PROJECT_ID}" >/dev/null

  for record in "${DNS_NAME}:A" "api.${DNS_NAME}:A" "www.${DNS_NAME}:CNAME"; do
    name="${record%%:*}"
    type="${record##*:}"
    existing="$(
      gcloud dns record-sets list \
        --zone="${ZONE_NAME}" \
        --project="${PROJECT_ID}" \
        --name="${name}" \
        --type="${type}" \
        --format="value(rrdatas[0])"
    )"
    if [[ -n "${existing}" ]]; then
      gcloud dns record-sets transaction remove "${existing}" \
        --name="${name}" \
        --ttl=300 \
        --type="${type}" \
        --zone="${ZONE_NAME}" \
        --project="${PROJECT_ID}" >/dev/null
    fi
  done

  gcloud dns record-sets transaction add "${TARGET_IP}" \
    --name="${DNS_NAME}" \
    --ttl=300 \
    --type=A \
    --zone="${ZONE_NAME}" \
    --project="${PROJECT_ID}" >/dev/null
  gcloud dns record-sets transaction add "${TARGET_IP}" \
    --name="api.${DNS_NAME}" \
    --ttl=300 \
    --type=A \
    --zone="${ZONE_NAME}" \
    --project="${PROJECT_ID}" >/dev/null
  gcloud dns record-sets transaction add "${DNS_NAME}" \
    --name="www.${DNS_NAME}" \
    --ttl=300 \
    --type=CNAME \
    --zone="${ZONE_NAME}" \
    --project="${PROJECT_ID}" >/dev/null

  gcloud dns record-sets transaction execute \
    --zone="${ZONE_NAME}" \
    --project="${PROJECT_ID}"
)

gcloud dns managed-zones describe "${ZONE_NAME}" \
  --project="${PROJECT_ID}" \
  --format="value(nameServers)"
