# GCP Wallet Copilot Deployment

This runbook treats the GCP VM as the single production runtime for Qorvi web, Go API, and Qorvi AI Wallet Copilot.

## Runtime Split

- `qorvi-web`: Next.js app on port `3000`.
- `qorvi-api`: Go API on port `4000`.
- `qorvi-redis`: Redis queue/cache used by both the Go stack and wallet copilot jobs.
- `qorvi-postgres`: Postgres for the Go API and Wallet Copilot evidence/snapshot store.
- `qorvi-neo4j`: Neo4j for the existing graph features.

Suggested public routing:

```text
qorvi.app
-> GCP Nginx
-> qorvi-web:3000
-> /api/wallet/* handled by Next.js wallet copilot API routes
-> /v1/* proxied internally to qorvi-api:4000

api.qorvi.app
-> GCP Nginx
-> qorvi-api:4000
```

## Compose File

Production compose file:

```text
infra/docker/docker-compose.prod.yml
```

It builds:

- `Dockerfile.web`
- `Dockerfile.api`
- `Dockerfile.worker`

The web container uses `REDIS_URL=redis://redis:6379/0` for queue/quota/cache and `POSTGRES_URL` for canonical Wallet Copilot evidence/snapshots, so Upstash is not required on GCP.

## Required Environment

The Terraform startup script currently writes `/opt/qorvi/app/.env.backend`. Keep using that file, but add the wallet copilot values there.

```bash
NODE_ENV=production

NEXT_PUBLIC_APP_URL=https://qorvi.app
NEXT_PUBLIC_APP_BASE_URL=https://qorvi.app
NEXT_PUBLIC_API_BASE_URL=https://api.qorvi.app
API_PROXY_TARGET=http://api:4000

REDIS_URL=redis://redis:6379/0
POSTGRES_URL=postgres://qorvi:<password>@postgres:5432/qorvi?sslmode=disable
QORVI_WALLET_ANALYSIS_EXECUTION_MODE=worker
QORVI_WALLET_PROVIDER=auto
QORVI_WALLET_WORKER_SECRET=<generate-a-long-random-secret>
CRON_SECRET=<same-value-or-another-long-random-secret>

ALCHEMY_BASE_URL=https://eth-mainnet.g.alchemy.com
QORVI_ETH_RPC_RATE_LIMIT_MS=350
QORVI_ETH_RPC_MAX_RETRIES=2
QORVI_ETH_RPC_RETRY_MS=1500
QORVI_RECEIPT_MAX_TRANSACTIONS=80
```

Optional LLM keys:

```bash
OPENAI_API_KEY=<openai-api-key>
ANTHROPIC_API_KEY=<anthropic-api-key>
```

Optional tuning:

```bash
QORVI_WALLET_ANALYSIS_CACHE_SECONDS=300
QORVI_WALLET_ANALYSIS_JOB_TTL_SECONDS=3600
QORVI_WALLET_WORKER_CRON_LIMIT=1
QORVI_ETHERSCAN_RATE_LIMIT_MS=500
QORVI_PROVIDER_DAILY_BUDGET=90000
QORVI_ANON_ANALYSIS_DAILY_LIMIT=3
QORVI_AUTH_ANALYSIS_DAILY_LIMIT=10
QORVI_ANON_CHAT_DAILY_LIMIT=30
QORVI_AUTH_CHAT_DAILY_LIMIT=50
QORVI_ETHERSCAN_MAX_PAGES=10
QORVI_ALCHEMY_MAX_PAGES=8
QORVI_PROVIDER_TIMEOUT_MS=20000
QORVI_ETH_RPC_RATE_LIMIT_MS=350
QORVI_ETH_RPC_MAX_RETRIES=2
QORVI_ETH_RPC_RETRY_MS=1500
QORVI_LIFETIME_BLOCK_CHUNK_SIZE=200000
QORVI_RECEIPT_MAX_TRANSACTIONS=80
QORVI_HISTORICAL_PRICE_MAX_EVENTS=40
QORVI_AAVE_RESERVE_CACHE_SECONDS=86400
```

## Secret Manager

Do not put provider or LLM keys in Terraform `app_env_content`, VM metadata, or the checked-out application configuration. Store them as Secret Manager versions:

```bash
gcloud services enable secretmanager.googleapis.com --project=qorvi-493115
printf '%s' '<new-etherscan-key>' | gcloud secrets versions add qorvi-etherscan-api-key --data-file=- --project=qorvi-493115
printf '%s' '<alchemy-key>' | gcloud secrets versions add qorvi-alchemy-api-key --data-file=- --project=qorvi-493115
printf '%s' '<openai-key>' | gcloud secrets versions add qorvi-openai-api-key --data-file=- --project=qorvi-493115
```

The same rule applies to runtime credentials used by the existing graph/API surface (`qorvi-dune-api-key`, `qorvi-helius-api-key`, `qorvi-moralis-api-key`, `qorvi-auth-secret`, `qorvi-clerk-secret-key`, `qorvi-postgres-password`, `qorvi-postgres-url`, and `qorvi-neo4j-password`). The render script loads these names when versions exist.

Create each secret once before adding the first version:

```bash
gcloud secrets create qorvi-etherscan-api-key --replication-policy=automatic --project=qorvi-493115
gcloud secrets create qorvi-alchemy-api-key --replication-policy=automatic --project=qorvi-493115
gcloud secrets create qorvi-openai-api-key --replication-policy=automatic --project=qorvi-493115
```

Before building or restarting containers, render the latest secret versions into the host-only runtime env file:

```bash
cd /opt/qorvi/app
./scripts/gcp-render-wallet-copilot-secrets.sh qorvi-493115 .env.wallet-secrets
docker compose --env-file .env.backend --env-file .env.wallet-secrets -f infra/docker/docker-compose.prod.yml up -d --build web api
```

The renderer writes `.env.wallet-secrets` with mode `0600`. Remove provider/LLM secret lines from `.env.backend` after this path is verified. An Etherscan key previously exposed in text or metadata must be revoked in the Etherscan dashboard and replaced with a new Secret Manager version; relocating the old value is not rotation.

The production VM metadata startup script must also remain secret-free. Use `infra/gcp/qorvi-runtime-startup.sh`; it only starts containers from already provisioned runtime env files and never embeds provider credentials in Compute Engine metadata.

## Build And Start

Build from a clean checkout pinned to a reviewed release commit, or deploy a
versioned container image produced from that commit. Do not run a destructive
reset against a runtime directory containing unreleased work.

```bash
cd /opt/qorvi/app
git fetch origin
git switch --detach <reviewed-release-commit-sha>

docker compose --env-file .env.backend --env-file .env.wallet-secrets -f infra/docker/docker-compose.prod.yml up -d --build postgres redis neo4j api web
```

Before starting a web image that includes Wallet Copilot persistence, apply the migration:

```bash
docker compose --env-file .env.backend --env-file .env.wallet-secrets -f infra/docker/docker-compose.prod.yml exec -T postgres \
  psql -v ON_ERROR_STOP=1 -U "${QORVI_POSTGRES_USER:-qorvi}" -d "${QORVI_POSTGRES_DB:-qorvi}" \
  < infra/migrations/postgres/0027_wallet_copilot_index.sql
docker compose --env-file .env.backend --env-file .env.wallet-secrets -f infra/docker/docker-compose.prod.yml exec -T postgres \
  psql -v ON_ERROR_STOP=1 -U "${QORVI_POSTGRES_USER:-qorvi}" -d "${QORVI_POSTGRES_DB:-qorvi}" \
  < infra/migrations/postgres/0028_wallet_copilot_lifetime_coverage.sql
```

Optional existing worker:

```bash
docker compose --env-file .env.backend --env-file .env.wallet-secrets -f infra/docker/docker-compose.prod.yml --profile worker up -d --build worker-backfill
```

## Wallet Copilot Queue Worker

With GCP single-runtime deployment, the wallet copilot queue can be drained by calling the Next.js worker route from the VM.

Manual drain:

```bash
WORKER_SECRET="$(sed -n 's/^QORVI_WALLET_WORKER_SECRET=//p' /opt/qorvi/app/.env.wallet-secrets)"
curl -sS http://127.0.0.1:3000/api/wallet/analyze/worker?limit=1 \
  -X POST \
  -H "Authorization: Bearer ${WORKER_SECRET}"
```

Recommended cron on the VM:

```cron
* * * * * WORKER_SECRET="$(sed -n 's/^QORVI_WALLET_WORKER_SECRET=//p' /opt/qorvi/app/.env.wallet-secrets)" && test -n "$WORKER_SECRET" && curl -fsS -X POST http://127.0.0.1:3000/api/wallet/analyze/worker?limit=1 -H "Authorization: Bearer ${WORKER_SECRET}" >/dev/null
```

You can also point GCP Cloud Scheduler at:

```text
https://qorvi.app/api/wallet/analyze/worker?limit=1
```

with:

```text
Authorization: Bearer <QORVI_WALLET_WORKER_SECRET>
```

## GCP Cloud DNS

The GCP VM currently uses the reserved external IP `34.87.143.25`. Qorvi's
public DNS should be hosted in GCP Cloud DNS for the GCP-only runtime.

Create or refresh the Cloud DNS zone and records:

```bash
./scripts/gcp-upsert-qorvi-dns.sh qorvi.app 34.87.143.25
```

The script upserts these records in managed zone `qorvi-app`:

- `qorvi.app A 34.87.143.25`
- `api.qorvi.app A 34.87.143.25`
- `www.qorvi.app CNAME qorvi.app`

The registrar nameservers must match the Cloud DNS nameservers:

```text
ns-cloud-d1.googledomains.com
ns-cloud-d2.googledomains.com
ns-cloud-d3.googledomains.com
ns-cloud-d4.googledomains.com
```

Verify the authoritative Cloud DNS response before public delegation has fully
propagated:

```powershell
Resolve-DnsName qorvi.app -Type A -Server ns-cloud-d1.googledomains.com
Resolve-DnsName api.qorvi.app -Type A -Server ns-cloud-d1.googledomains.com
```

Verify public delegation after changing nameservers at the registrar:

```powershell
Resolve-DnsName qorvi.app -Type NS
Resolve-DnsName qorvi.app -Type A
Resolve-DnsName api.qorvi.app -Type A
```

Expected public result:

```text
qorvi.app NS -> ns-cloud-d*.googledomains.com
qorvi.app A -> 34.87.143.25
api.qorvi.app A -> 34.87.143.25
```

## Nginx Sketch

```nginx
server {
    listen 80;
    server_name qorvi.app www.qorvi.app;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 120s;
    }
}

server {
    listen 80;
    server_name api.qorvi.app;

    location / {
        proxy_pass http://127.0.0.1:4000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 120s;
    }
}
```

## Health Check

```bash
curl -sS https://qorvi.app/api/wallet/analyze/health \
  -H "Authorization: Bearer $QORVI_WALLET_WORKER_SECRET"
```

Expected:

```json
{
  "ok": true,
  "storage": "redis",
  "redis_configured": true,
  "redis_url_configured": true,
  "upstash_redis_configured": false,
  "execution_mode": "worker",
  "postgres_configured": true,
  "index_store_ok": true,
  "worker_secret_configured": true,
  "error": null
}
```

## Smoke Test

Before DNS delegation has propagated, use `curl --resolve` to test the exact
production hosts against the GCP VM:

```bash
curl -I --resolve qorvi.app:443:34.87.143.25 https://qorvi.app
curl -I --resolve www.qorvi.app:443:34.87.143.25 https://www.qorvi.app
curl -I --resolve api.qorvi.app:443:34.87.143.25 https://api.qorvi.app
```

```bash
curl -sS https://qorvi.app/api/wallet/analyze/jobs \
  -H "content-type: application/json" \
  -d '{"address":"0xF00a90FB0129d61DD09194BF70759CD5D36E3d2D","days":7}'
```

Poll the returned `status_url`, then ask a copilot question:

```bash
curl -sS https://qorvi.app/api/wallet/chat \
  -H "content-type: application/json" \
  -d '{"address":"0xF00a90FB0129d61DD09194BF70759CD5D36E3d2D","days":7,"question":"What is this wallet doing?"}'
```

Expected beta behavior:

- A cache hit does not consume a new analysis quota.
- New anonymous analyses return `quota_exceeded` after three requests per IP per UTC day.
- Etherscan collection returns `provider_budget_exceeded` after the configured global daily provider budget.
- Chat without an existing analysis snapshot returns `analysis_required` rather than fetching provider data.

## Notes

- Do not keep Terraform state, `terraform.tfvars`, VM metadata startup scripts, or runtime env files with secret values in the repository workspace. Keep emergency local copies outside the repo with operator-only filesystem permissions.
- `NEXT_PUBLIC_*` values are browser-visible. Do not put secrets there.
- The current production target is GCP-only.
