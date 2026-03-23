#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="full"

usage() {
  cat <<'EOF'
Usage: ./scripts/beta-hardening.sh [--prep] [--evidence-core]

Options:
  --prep           Run beta preflight infra + migrations only
  --evidence-core  Run repeatable beta evidence commands only
  --help           Show this message
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
  echo "==> Beta prep: infra"
  corepack pnpm dev:infra

  echo "==> Beta prep: migrations"
  corepack pnpm dev:migrate
}

run_evidence_core() {
  echo "==> Beta evidence: web typecheck"
  corepack pnpm --filter @whalegraph/web typecheck

  echo "==> Beta evidence: web lint"
  corepack pnpm --filter @whalegraph/web lint

  echo "==> Beta evidence: backend/provider/worker contracts"
  GOCACHE=/tmp/whalegraph-go-cache go test ./packages/providers ./apps/api/internal/server ./apps/workers

  echo "==> Beta evidence: mixed browser/API beta flow"
  corepack pnpm --filter @whalegraph/web test:e2e -- e2e/beta-flow.spec.ts
}

case "$MODE" in
  prep)
    run_prep
    ;;
  evidence-core)
    run_evidence_core
    ;;
  full)
    run_prep
    run_evidence_core
    ;;
esac
