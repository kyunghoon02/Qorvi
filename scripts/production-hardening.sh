#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="full"

usage() {
  cat <<'EOF'
Usage: ./scripts/production-hardening.sh [--prep] [--evidence-core] [--evidence-billing]

Options:
  --prep              Run production preflight infra + migrations only
  --evidence-core     Run repeatable production evidence commands only
  --evidence-billing  Run optional billing evidence commands
  --help              Show this message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --prep)
      MODE="prep"
      shift
      ;;
    --evidence-core)
      MODE="evidence-core"
      shift
      ;;
    --evidence-billing)
      MODE="evidence-billing"
      shift
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

run_prep() {
  echo "==> Production prep: infra"
  corepack pnpm dev:infra

  echo "==> Production prep: migrations"
  corepack pnpm dev:migrate
}

run_evidence_core() {
  echo "==> Production evidence: web typecheck"
  corepack pnpm --filter @flowintel/web typecheck

  echo "==> Production evidence: web lint"
  corepack pnpm --filter @flowintel/web lint

  echo "==> Production evidence: backend/provider/worker contracts"
  GOCACHE=/tmp/flowintel-go-cache go test ./packages/providers ./apps/api/internal/server ./apps/workers

  echo "==> Production evidence: browser/API tracked wallet flow"
  corepack pnpm --filter @flowintel/web test:e2e -- e2e/beta-flow.spec.ts --grep "searches a wallet and lands on tracked alerts"
}

run_evidence_billing() {
  echo "==> Production evidence: optional billing checkout flow"
  corepack pnpm --filter @flowintel/web test:e2e -- e2e/beta-flow.spec.ts --grep "creates checkout intent, reconciles billing, and shows upgraded account"
}

case "$MODE" in
  prep)
    run_prep
    ;;
  evidence-core)
    run_evidence_core
    ;;
  evidence-billing)
    run_evidence_billing
    ;;
  full)
    run_prep
    run_evidence_core
    ;;
esac
