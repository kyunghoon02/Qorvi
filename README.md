# Qorvi AI Wallet Copilot

Qorvi is evolving from wallet search and on-chain flow visualization into an AI-native wallet analysis assistant: "ChatGPT for understanding an on-chain wallet."

The product lets a user enter an Ethereum wallet address, select a 7d/30d/90d window, run deterministic analysis, read an AI-generated wallet behavior report, and ask follow-up questions that are routed to internal tools.

## What It Does

- Analyzes Ethereum wallet transactions and ERC-20 transfers for the selected report window and persists new analysis evidence in PostgreSQL.
- Classifies activity into native transfers, token flows, swaps, DeFi interactions, CEX transfer hints, contract interactions, and unknowns.
- Infers clear token swap summaries such as token sent, token received, amount, protocol label, and evidence transaction.
- Shows current ETH and active ERC-20 holdings with live USD value when pricing data is available.
- Reads current Aave V3 supply/borrow positions, Uniswap V3 LP NFTs, and Curve LP/gauge balances from live Ethereum data.
- Detects only allowlisted Ethereum bridge contracts, including canonical Optimism/Base/Arbitrum L1 contracts, Across Ethereum SpokePool, and official Stargate Ethereum entry contracts.
- Labels value attribution as `On-chain Performance`; incomplete indexing or pricing is surfaced as `partial`, never presented as complete PnL.
- Generates deterministic metrics such as transaction count, ERC-20 transfer count, unique counterparties, active tokens, activity types, and heuristic risk level.
- Produces an AI-readable report with evidence-backed sections.
- Supports follow-up chat through deterministic internal tools.
- Requires live Ethereum provider data. Qorvi uses Etherscan by default and can fall back to Alchemy when configured; if no live provider is available, Qorvi returns an error instead of using mock data.

## Why This Is AI x Crypto

Qorvi combines on-chain data tools with an LLM summarization layer. The LLM is not the source of truth. It only explains structured data returned by provider, classification, risk, and agent tool modules. This mirrors an AI wallet/DeFi copilot architecture where an agent selects tools, receives deterministic results, and explains those results in plain English.

## Architecture

```text
User
-> Next.js UI
-> Wallet Analysis Job API
-> Redis Admission Control / Job Queue / Provider Budget
-> Agent Tool Router
-> On-chain Data Providers
-> Classification / Risk Engine
-> PostgreSQL Evidence / Snapshot Store
-> LLM Report Generator
-> Structured Wallet Report
```

Implementation modules:

- `apps/web/app/copilot-screen.tsx`: main wallet copilot interface.
- `apps/web/app/api/wallet/analyze/route.ts`: `POST /api/wallet/analyze`.
- `apps/web/app/api/wallet/analyze/jobs/route.ts`: `POST /api/wallet/analyze/jobs` creates an async analysis job.
- `apps/web/app/api/wallet/analyze/jobs/[jobId]/route.ts`: `GET /api/wallet/analyze/jobs/:jobId` returns queued/running/succeeded/failed status.
- `apps/web/app/api/wallet/analyze/worker/route.ts`: `POST /api/wallet/analyze/worker` processes queued analysis jobs for a separate worker or cron runner.
- `apps/web/app/api/wallet/analyze/cron/route.ts`: `GET /api/wallet/analyze/cron` is a scheduler-compatible queue processor.
- `apps/web/app/api/wallet/analyze/health/route.ts`: `GET /api/wallet/analyze/health` verifies storage configuration and runtime mode.
- `apps/web/app/api/wallet/chat/route.ts`: `POST /api/wallet/chat`.
- `apps/web/lib/wallet-copilot/storage.ts`: lazy storage adapter. Uses Upstash Redis REST when configured, local file storage in development, and process memory only as a last-resort production fallback.
- `apps/web/lib/wallet-copilot/quota.ts`: Redis-backed public quota and global Etherscan request budget guard.
- `apps/web/lib/wallet-copilot/index-repository.ts`: PostgreSQL persistence for block checkpoints, raw transactions/transfers, receipts/logs, historical prices, Aave reserve snapshots, evidence, decoded actions, bridge movements, and analysis snapshots.
- `apps/web/lib/wallet-copilot/lifetime-backfill.ts`: resumable Ethereum block-range lifetime backfill pass that advances persisted checkpoints without discarding completed ranges.
- `apps/web/lib/wallet-copilot/receipt-reader.ts` and `action-decoder.ts`: RPC receipt/log enrichment and evidence-backed Aave V3, Uniswap V3 LP, and Curve gauge event decoding.
- `apps/web/lib/wallet-copilot/historical-prices.ts`: event-hour historical USD price adapter with PostgreSQL-backed cache.
- `apps/web/lib/wallet-copilot/jobs.ts`: wallet analysis job model, status persistence, cache reuse, and in-process job runner.
- `apps/web/lib/wallet-copilot/provider.ts`: live-provider orchestrator. `auto` mode prefers Etherscan and falls back to Alchemy for recoverable provider failures when Alchemy is configured.
- `apps/web/lib/wallet-copilot/etherscan-provider.ts`: Etherscan V2 block-range, pagination, transfer, balance, and price-backed dataset reader.
- `apps/web/lib/wallet-copilot/alchemy-provider.ts`: Alchemy transfer, token balance, native balance, and price-backed dataset reader.
- `apps/web/lib/wallet-copilot/provider-utils.ts`: shared provider helpers for pricing, token target selection, amount formatting, env parsing, and deduplication.
- `apps/web/lib/wallet-copilot/defi-position-readers.ts`: Aave V3, Uniswap V3, and Curve live position readers.
- `apps/web/lib/wallet-copilot/eth-call.ts`: Ethereum `eth_call` helper using `ETHEREUM_RPC_URL`, `ALCHEMY_API_KEY`, or Etherscan proxy fallback.
- `apps/web/lib/wallet-copilot/classifier.ts`: transaction classification, token flows, swap inference, protocol/CEX hints, portfolio summary, DeFi position summary, risk flags.
- `apps/web/lib/wallet-copilot/agent.ts`: intent router and deterministic tool executor.
- `apps/web/lib/wallet-copilot/llm.ts`: report and answer generation with deterministic fallback and grounding verification.

## Agent Tool Flow

1. User asks a follow-up question.
2. Chat uses an already analyzed snapshot; it does not initiate a provider re-fetch.
3. The intent router classifies it as `wallet_summary`, `latest_transactions`, `transaction_explanation`, `token_flow`, `defi_activity`, `bridge_activity`, `portfolio`, `onchain_performance`, `behavior_profile`, or `unknown`.
4. The agent selects one deterministic tool:
   - `get_wallet_summary`
   - `get_token_flow_summary`
   - `get_defi_interactions`
   - `get_cex_transfer_hints`
   - `get_portfolio_summary`
   - `get_onchain_performance`
   - `get_bridge_movements`
   - `get_wallet_behavior_profile`
5. The tool returns structured data and evidence ids.
6. OpenAI summarizes only normalized tool output. An identifier-grounding check discards answers containing unsupported hashes or addresses; missing keys or failed checks use deterministic output.

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

## Data Sources

The current product supports Ethereum mainnet.

- Live mode: Etherscan V2 account APIs are used by default. Alchemy Transfers and Token APIs are supported as a live fallback when `ALCHEMY_API_KEY`, `ALCHEMY_BASE_URL`, or `ALCHEMY_ETHEREUM_RPC_URL` is configured.
- Provider mode: `QORVI_WALLET_PROVIDER=auto` tries Etherscan first and falls back to Alchemy for provider errors, timeouts, invalid/missing Etherscan keys, or rate limits. Set `etherscan` or `alchemy` to force one provider.
- Current price mode: CoinGecko simple price APIs are used for ETH and active ERC-20 USD values when available.
- DeFi position mode:
  - Aave V3: discovers Ethereum reserve tokens through the ProtocolDataProvider, snapshots discovered reserves, and reads wallet supply/borrow state.
  - Uniswap V3: reads LP NFT token IDs from the canonical NonfungiblePositionManager, then values principal and uncollected fees from pool `slot0`, tick state, and fee growth.
  - Curve: reads Curve all-pools metadata, prioritizes direct wallet LP/gauge candidates plus top-TVL pools, then reads wallet/gauge LP balances from on-chain `balanceOf` calls.
- Bridge coverage: Qorvi confirms only exact allowlist hits for Optimism, Base, Arbitrum One, Across, and Stargate Ethereum-side contracts. Destination-chain holdings are shown as coverage limitations, not inferred balances.
- No mock mode: missing keys or provider failures return an API error.
- Etherscan free-plan protection: provider calls are serialized at a conservative default maximum of 2 calls/second, and a global hard stop rejects provider work above 90,000 calls/day.
- Block range + pagination: Qorvi resolves timestamp windows with `getblocknobytime`, then paginates `txlist` and `tokentx` with `startblock`, `endblock`, `page`, and `offset`.
- Lifetime indexing: a worker pass advances a PostgreSQL block checkpoint from the latest Ethereum block toward the wallet's first observed activity range, retaining each completed chunk for retry-safe continuation.
- Receipt/log decoding: when an Ethereum RPC endpoint is configured, Qorvi fetches transaction receipts for contract interactions and decodes supported protocol events rather than relying only on counterparty labels.
- Historical event pricing: ERC-20 activity is valued at UTC event-hour price points from CoinGecko when available; stablecoin legs use explicit parity pricing and unavailable points remain unpriced.
- Holdings: Qorvi reads native ETH balance and active ERC-20 token balances discovered from recent transfer activity.
- Wallet analysis jobs: the UI creates an analysis job, polls its status, and renders the result when the job succeeds. Job records store `queued`, `running`, `succeeded`, or `failed` state plus structured errors.
- Durable worker mode: set `QORVI_WALLET_ANALYSIS_EXECUTION_MODE=worker` and configure either `REDIS_URL` for a standard Redis instance or Upstash Redis REST to make web requests enqueue jobs only. A separate worker or scheduled job should call `POST /api/wallet/analyze/worker?limit=1` with `Authorization: Bearer $QORVI_WALLET_WORKER_SECRET`.
- Queue scheduler: `GET /api/wallet/analyze/cron?limit=1` or `POST /api/wallet/analyze/worker?limit=1` processes queued jobs. The current production setup wires this through VM cron on GCP.
- Health check: call `GET /api/wallet/analyze/health` with `Authorization: Bearer $QORVI_WALLET_WORKER_SECRET` to verify Redis/local storage and execution mode without exposing secrets.
- Observability: wallet analysis job lifecycle events are emitted as structured JSON logs with hashed wallet addresses, job ids, durations, result counts, cache hits, and provider errors.
- Wallet analysis cache: repeated `address + days` analyses are cached for 5 minutes by default. If `REDIS_URL` or `UPSTASH_REDIS_REST_URL` plus `UPSTASH_REDIS_REST_TOKEN` are configured, cache and job status are stored in Redis; otherwise local development uses `.qorvi-wallet-copilot-store`.
- Public beta admission control: cache hits do not spend analysis quota. New analysis jobs are limited to 3/day by anonymous IP or 10/day for authenticated users; chat is 30/day or 50/day respectively.
- PostgreSQL storage: apply `infra/migrations/postgres/0027_wallet_copilot_index.sql` and `0028_wallet_copilot_lifetime_coverage.sql`, then set `POSTGRES_URL` to store checkpoints, raw windows, receipt/log evidence, historical prices, reserve snapshots, decoded actions, bridges, and reports.
- LLM mode: `OPENAI_API_KEY` is the primary production configuration; deterministic output is used on model failure or grounding failure.
- Deterministic mode: local report and chat answer generation when LLM keys are missing.

Supported environment variables:

- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `ALCHEMY_API_KEY`
- `ETHERSCAN_API_KEY`
- `COVALENT_API_KEY`
- `NEXT_PUBLIC_APP_URL`
- `ETHEREUM_RPC_URL`
- `ALCHEMY_BASE_URL`
- `ALCHEMY_ETHEREUM_RPC_URL`
- `QORVI_WALLET_PROVIDER`
- `QORVI_ETHERSCAN_MAX_PAGES`
- `QORVI_ALCHEMY_MAX_PAGES`
- `QORVI_ETHERSCAN_RATE_LIMIT_MS`
- `QORVI_ETH_RPC_RATE_LIMIT_MS`
- `QORVI_ETH_RPC_MAX_RETRIES`
- `QORVI_ETH_RPC_RETRY_MS`
- `QORVI_PROVIDER_DAILY_BUDGET`
- `QORVI_ANON_ANALYSIS_DAILY_LIMIT`
- `QORVI_AUTH_ANALYSIS_DAILY_LIMIT`
- `QORVI_ANON_CHAT_DAILY_LIMIT`
- `QORVI_AUTH_CHAT_DAILY_LIMIT`
- `QORVI_DEFI_POSITION_READERS`
- `QORVI_UNISWAP_MAX_POSITIONS`
- `QORVI_CURVE_MAX_POOLS`
- `QORVI_CURVE_POOL_CACHE_SECONDS`
- `QORVI_PROVIDER_TIMEOUT_MS`
- `QORVI_LIFETIME_BLOCK_CHUNK_SIZE`
- `QORVI_RECEIPT_MAX_TRANSACTIONS`
- `QORVI_HISTORICAL_PRICE_MAX_EVENTS`
- `QORVI_AAVE_RESERVE_CACHE_SECONDS`
- `QORVI_WALLET_ANALYSIS_EXECUTION_MODE`
- `QORVI_WALLET_ANALYSIS_CACHE_SECONDS`
- `QORVI_WALLET_ANALYSIS_JOB_TTL_SECONDS`
- `QORVI_WALLET_WORKER_SECRET`
- `QORVI_WALLET_WORKER_CRON_LIMIT`
- `CRON_SECRET`
- `REDIS_URL`
- `POSTGRES_URL`
- `UPSTASH_REDIS_REST_URL`
- `UPSTASH_REDIS_REST_TOKEN`

At least one live wallet provider must be configured: `ETHERSCAN_API_KEY` for Etherscan or an Alchemy RPC URL/key for Alchemy. `ETHEREUM_RPC_URL`, `ALCHEMY_BASE_URL`, `ALCHEMY_ETHEREUM_RPC_URL`, or `ALCHEMY_API_KEY` are recommended for faster contract reads; otherwise Etherscan proxy `eth_call` is used when available. LLM keys are optional because local deterministic report/chat generation is still available.

## Limitations

- Protocol and CEX labels are intentionally small maps in the current version.
- CEX transfer hints are possible labels, not definitive proof of exchange deposits or withdrawals.
- Risk scoring is heuristic and not a security audit.
- Current USD value is based on live token prices when available, but missing price coverage is possible.
- On-chain Performance uses the persisted lifetime ledger after backfill completes and exposes `complete` only when lifetime indexing, receipt/log coverage, event-time prices, current asset pricing, and value attribution are complete. Detected DeFi actions or bridge movements remain `partial` until their event-level contribution or destination-chain value can be fully attributed.
- Destination-chain balances and performance after allowlisted bridge movement are not included.
- Alchemy fallback is transfer-centric. It can recover native/ERC-20 transfer history and balances, but may miss zero-value contract calls that Etherscan `txlist` would expose.
- Inline execution mode still runs analysis from the web process for simple local development. Production deployments should use worker mode with Redis-backed queue state.
- Without Upstash Redis, development cache and job state are local-file based and not suitable for multi-instance production deployments.
- Aave reserve discovery uses the Ethereum V3 ProtocolDataProvider; a small static fallback is used only when live discovery cannot be read.
- Uniswap V3 LP value depends on live pool state and CoinGecko token price coverage; missing token prices produce partial valuation.
- Curve LP value is calculated from Curve all-pools API pool USD total and on-chain LP/gauge balances for selected candidate pools.
- Only Ethereum mainnet EVM addresses are supported.
- The LLM must not invent transaction hashes, amounts, labels, counterparties, or risk flags.

## Deployment Notes

- The wallet copilot backend lives in Next.js API routes under `apps/web/app/api/wallet/*` and is deployed with the Next.js container on the current GCP VM.
- Production defaults to worker queue mode unless `QORVI_WALLET_ANALYSIS_EXECUTION_MODE=inline` is set explicitly. Configure `REDIS_URL` or Upstash Redis REST; production worker mode fails fast without durable Redis-backed job storage.
- GCP single-runtime setup is documented in `docs/deploy/gcp-wallet-copilot.md`.
- Keep Terraform state, `terraform.tfvars`, and `.env*` files out of git. They contain infrastructure config and may contain secrets.

## Future Improvements

- More protocol labels.
- More accurate CEX labels.
- Real-time monitoring.
- Wallet alerts.
- Cross-chain support.
- Wider LP valuation coverage for tokens missing public price data.
- User-specific Curve gauge discovery beyond candidate/top-TVL scanning.
- Event-level performance attribution for Aave debt/yield, LP fees, and destination-chain bridge assets.
- Expanded protocol event coverage beyond Aave V3, Uniswap V3 LP, and Curve gauge events.
- Transaction simulation.
- RAG over protocol docs.
- Integration with vLLM or SGLang for self-hosted model serving.

## Development

```bash
corepack pnpm install
corepack pnpm dev:web
```

Create `apps/web/.env.local` with:

```bash
ETHERSCAN_API_KEY=your_etherscan_api_key
```

Open the app at `http://localhost:3000`.

Validation commands:

```bash
corepack pnpm --filter @qorvi/web typecheck
corepack pnpm --filter @qorvi/web test
```
