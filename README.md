# Qorvi

Qorvi is an AI-native wallet analysis product for Ethereum. It combines
deterministic on-chain data collection with an LLM explanation layer so a user
can inspect a wallet, read an evidence-backed report, and ask follow-up
questions without treating the model as the source of truth.

## Live Links

Checked on 2026-06-16:

- App: [https://qorvi.app](https://qorvi.app)
- Deployment runbook: [docs/deploy/gcp-wallet-copilot.md](/C:/Github/Qorvi/docs/deploy/gcp-wallet-copilot.md)

## What It Does

- Analyzes an Ethereum wallet over a `7d`, `30d`, or `90d` window.
- Collects native transfers, ERC-20 transfers, contract interactions, and
  receipt/log evidence.
- Classifies swaps, DeFi activity, bridge usage, CEX transfer hints, and
  behavior patterns.
- Reads current holdings plus supported Aave V3, Uniswap V3, and Curve
  positions.
- Persists checkpoints, evidence, historical prices, decoded actions, and
  reports in PostgreSQL.
- Supports follow-up chat using deterministic internal tools over an existing
  analysis snapshot.
- Uses live provider data only. If providers are unavailable, Qorvi returns an
  error rather than mock output.

## Product Flow

1. User enters an Ethereum wallet address.
2. Qorvi runs deterministic indexing and classification.
3. Qorvi generates an evidence-backed report.
4. User asks follow-up questions against the stored snapshot.
5. The LLM explains normalized tool output; it does not invent facts.

## Architecture

```text
User
-> Next.js UI
-> Wallet analysis API
-> Redis queue / quota / cache
-> Provider adapters
-> Classification and risk engine
-> PostgreSQL evidence store
-> LLM summarization
-> Structured wallet report
```

## Key Code Paths

- `apps/web/app/copilot/page.tsx`: wallet copilot route.
- `apps/web/app/copilot-screen.tsx`: main copilot UI.
- `apps/web/app/api/wallet/analyze/*`: analysis job, worker, cron, and health
  routes.
- `apps/web/app/api/wallet/chat/route.ts`: follow-up chat endpoint.
- `apps/web/lib/wallet-copilot/provider.ts`: provider orchestration.
- `apps/web/lib/wallet-copilot/classifier.ts`: transaction and behavior
  classification.
- `apps/web/lib/wallet-copilot/index-repository.ts`: PostgreSQL persistence.
- `apps/web/lib/wallet-copilot/jobs.ts`: async job lifecycle.
- `apps/web/lib/wallet-copilot/llm.ts`: report and answer generation.
- `apps/web/test/wallet-copilot.test.ts`: copilot test suite.

## Supported Questions

- What did this wallet do recently?
- Did this wallet interact with DeFi protocols?
- What swaps or Aave actions were detected?
- What is this wallet holding now?
- Did this wallet use an allowlisted bridge?
- What is the current on-chain performance coverage?
- Did this wallet send funds to a CEX?
- What tokens did this wallet receive the most?
- Is this wallet more like a trader, holder, or DeFi user?

## Runtime Model

- Chain support: Ethereum mainnet only.
- Default provider: Etherscan V2.
- Fallback provider: Alchemy when configured.
- Queue and quota state: Redis or Upstash Redis REST.
- Evidence and snapshots: PostgreSQL.
- Production execution mode: worker queue mode.
- Development fallback: local file storage for cache/job state.

## Core Environment Variables

Provider and RPC:

- `ETHERSCAN_API_KEY`
- `ALCHEMY_API_KEY`
- `ALCHEMY_BASE_URL`
- `ALCHEMY_ETHEREUM_RPC_URL`
- `ETHEREUM_RPC_URL`
- `QORVI_WALLET_PROVIDER`

Runtime and storage:

- `REDIS_URL`
- `POSTGRES_URL`
- `QORVI_WALLET_ANALYSIS_EXECUTION_MODE`
- `QORVI_WALLET_WORKER_SECRET`
- `CRON_SECRET`

LLM:

- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`

Limits and tuning:

- `QORVI_PROVIDER_DAILY_BUDGET`
- `QORVI_ANON_ANALYSIS_DAILY_LIMIT`
- `QORVI_AUTH_ANALYSIS_DAILY_LIMIT`
- `QORVI_ANON_CHAT_DAILY_LIMIT`
- `QORVI_AUTH_CHAT_DAILY_LIMIT`
- `QORVI_ETHERSCAN_RATE_LIMIT_MS`
- `QORVI_PROVIDER_TIMEOUT_MS`

For the full production setup, use
[docs/deploy/gcp-wallet-copilot.md](/C:/Github/Qorvi/docs/deploy/gcp-wallet-copilot.md).

## Local Development

Install dependencies:

```bash
corepack pnpm install
```

Create `apps/web/.env.local` with at least:

```bash
ETHERSCAN_API_KEY=your_etherscan_api_key
```

Run the web app:

```bash
corepack pnpm dev:web
```

Open `http://localhost:3000`.

Validation:

```bash
corepack pnpm --filter @qorvi/web typecheck
corepack pnpm --filter @qorvi/web test
```

## Deployment

- Wallet copilot backend lives in `apps/web/app/api/wallet/*`.
- Production target is the GCP single-runtime deployment documented in
  [docs/deploy/gcp-wallet-copilot.md](/C:/Github/Qorvi/docs/deploy/gcp-wallet-copilot.md).
- Production compose file lives at
  [infra/docker/docker-compose.prod.yml](/C:/Github/Qorvi/infra/docker/docker-compose.prod.yml).
- PostgreSQL migrations for copilot persistence live in
  [infra/migrations/postgres/0027_wallet_copilot_index.sql](/C:/Github/Qorvi/infra/migrations/postgres/0027_wallet_copilot_index.sql)
  and
  [infra/migrations/postgres/0028_wallet_copilot_lifetime_coverage.sql](/C:/Github/Qorvi/infra/migrations/postgres/0028_wallet_copilot_lifetime_coverage.sql).
- Keep Terraform state, `tfvars`, and secret env files out of git.

## Limitations

- Ethereum mainnet only.
- Protocol and CEX labels are heuristic, not authoritative.
- Risk scoring is not a security audit.
- Missing pricing or incomplete indexing yields partial coverage.
- Destination-chain balances after bridge activity are not included.
- Alchemy fallback can miss zero-value contract calls that Etherscan `txlist`
  would expose.
- Local file-backed cache/job state is not suitable for multi-instance
  production.
- The LLM must not invent transaction hashes, amounts, labels, counterparties,
  or risk flags.
