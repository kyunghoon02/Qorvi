# Render deployment

This repository fits Render best when you run it as separate services:

- `qorvi-web`: Next.js public web app
- `qorvi-api`: Go API with public webhook endpoints
- `qorvi-worker-drain`: continuous queue consumer
- `qorvi-worker-tracking-sync`: periodic provider subscription reconciler
- `qorvi-worker-mobula`: periodic smart-money discovery producer
- `qorvi-worker-seed-watchlist`: periodic featured-wallet watchlist seeder
- `qorvi-postgres`: managed PostgreSQL
- `qorvi-redis`: managed Redis/Key Value
- `qorvi-neo4j`: private Neo4j service with persistent disk

## Why Render

This app is not just a Next.js frontend. It needs:

- a public web app
- a public API for provider webhooks
- long-running background workers
- Redis, Postgres, and Neo4j

Render's Blueprint workflow is the cleanest fit for that shape.

## Files added

- `render.yaml`
- `Dockerfile.web`
- `Dockerfile.api`
- `Dockerfile.worker`
- `Dockerfile.neo4j`
- `.dockerignore`

## Deploy flow

1. Push the repository to GitHub.
2. In Render, create a new Blueprint from this repository.
3. Let Render create the services from `render.yaml`.
4. Fill in the `sync: false` environment variables in the Render dashboard.
5. Deploy.

## Required environment values

### Web

- `NEXT_PUBLIC_APP_BASE_URL`
  - Public URL of the web app, for example `https://app.example.com`
- `NEXT_PUBLIC_API_BASE_URL`
  - Public URL of the API, for example `https://api.example.com`
- `API_PROXY_TARGET`
  - Optional. Set this to the same public API URL only if you want Next.js to proxy `/v1/*`
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`

### API

- `APP_BASE_URL`
  - Public URL of the web app, not the API URL
  - The Clerk verifier uses this as the allowed origin
- `NEO4J_PASSWORD`
- `AUTH_SECRET`
- `CLERK_SECRET_KEY`
- `CLERK_ISSUER_URL`
- `CLERK_JWKS_URL`
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`
- billing secrets if billing is enabled

### Workers

- `APP_BASE_URL`
  - Public URL of the API, not the web URL
  - Workers use this when building provider webhook callback URLs
- `QORVI_PROVIDER_WEBHOOK_BASE_URL`
  - Set this to the same public API URL so tracking sync can auto-provision provider webhooks
- `NEO4J_PASSWORD`
- `AUTH_SECRET`
- `CLERK_SECRET_KEY`
- `CLERK_ISSUER_URL`
- `CLERK_JWKS_URL`
- `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`
- `DUNE_API_KEY`
- `ALCHEMY_API_KEY`
- `HELIUS_API_KEY`
- `MORALIS_API_KEY`
- `MOBULA_API_KEY`
- `QORVI_MOBULA_SMART_MONEY_SEEDS_JSON` for the Mobula worker
- `QORVI_SEED_DISCOVERY_TOP_N` for the seed-watchlist worker
- `QORVI_SEED_DISCOVERY_MIN_CONFIDENCE` for the seed-watchlist worker
- `ALCHEMY_NOTIFY_AUTH_TOKEN` if you want Alchemy webhook reconciliation to become `active`
- `QORVI_PROVIDER_WEBHOOK_AUTH_HEADER` if you want authenticated Helius webhook delivery

### Neo4j

- `NEO4J_AUTH`
  - Use the format `neo4j/<password>`
- Use the same password value for the API and worker `NEO4J_PASSWORD`

## Important URL split

There are two different `APP_BASE_URL` meanings in this repo today:

- API service `APP_BASE_URL` should be the public web URL
- Worker service `APP_BASE_URL` should be the public API URL

That split is intentional because the API uses it for Clerk origin validation, while workers use it for provider webhook callbacks.

## Migrations

This repository still uses file-based Postgres and Neo4j migrations in `infra/migrations`.

Before production traffic, run those migrations once against:

- `POSTGRES_URL` from `qorvi-postgres`
- `bolt://qorvi-neo4j:7687`

If you want, the next step is to add a dedicated Render migration job so this is part of deployment instead of a manual step.

## Visitor analytics

For traffic analytics on top of this deployment, the simplest stack is:

- Plausible for visitor metrics such as country, pageviews, referrers, and live visitors
- Qorvi internal events for product metrics such as wallet detail opens, graph loads, and search-to-backfill conversion

That keeps marketing-style analytics separate from product/ops telemetry.
