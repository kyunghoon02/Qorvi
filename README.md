# Qorvi

Qorvi is an onchain intelligence product focused on wallet investigation, graph analysis, alerting, and operator workflows.

## Repo layout

- `apps/web` contains the Next.js frontend.
- `apps/api` contains the Go API.
- `apps/workers` contains background ingestion, scoring, alerting, and backfill workers.
- `packages/` contains shared Go and TypeScript packages.
- `infra/` and `terraform/` contain local and production infrastructure assets.
- `flowintel-ai/` contains AI contracts, datasets, evals, and analyst workspace material.

## Local development

Prerequisites:

- Node.js 22+
- `corepack`
- Go 1.24+
- Docker

Common commands:

- `corepack pnpm dev:infra`
- `corepack pnpm dev:migrate`
- `corepack pnpm dev:web`
- `corepack pnpm dev:api`
- `corepack pnpm dev:workers`
- `corepack pnpm check`

## Environment

Environment templates live at:

- `.env.example`
- `.env.beta.example`
- `.env.production.example`

Do not commit real secrets. Local `.env` files are intentionally ignored.

## Deployment

- `render.yaml` contains a Render deployment shape.
- `infra/docker/docker-compose.prod.yml` contains the single-host Docker Compose shape.
- `terraform/README.md` documents the GCP VM deployment path for the backend.

## Notes

- The package namespace still uses `flowintel` internally while the product is being renamed to `Qorvi`.
- This repository is public-source but not open-source. See [LICENSE](/mnt/c/Github/Qorvi/LICENSE).
- Repository-managed seed data is documented in [infra/seeds/README.md](/mnt/c/Github/Qorvi/infra/seeds/README.md).
