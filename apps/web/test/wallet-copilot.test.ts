import assert from "node:assert/strict";
import { createServer } from "node:net";
import test from "node:test";

import { decodeReceiptActions } from "../lib/wallet-copilot/action-decoder";
import {
  answerWalletQuestion,
  classifyUserQuestion,
  normalizeDays,
} from "../lib/wallet-copilot/agent";
import {
  buildWalletAnalysisCacheKey,
  clearWalletAnalysisCache,
  getOrSetWalletAnalysisCache,
  setWalletAnalysisCache,
} from "../lib/wallet-copilot/cache";
import {
  buildWalletAnalysis,
  detectBridgeMovements,
} from "../lib/wallet-copilot/classifier";
import {
  calculateUniswapV3Amounts,
  decodeAaveReserveTokens,
  getSqrtRatioAtTick,
  selectCurveCandidatePools,
} from "../lib/wallet-copilot/defi-position-readers";
import { WalletProviderError } from "../lib/wallet-copilot/errors";
import { ethCall, formatUnits } from "../lib/wallet-copilot/eth-call";
import { historicalPriceBucket } from "../lib/wallet-copilot/historical-prices";
import { verifyGroundedIdentifiers } from "../lib/wallet-copilot/llm";
import { buildPnlSummary } from "../lib/wallet-copilot/pnl";
import {
  consumeProviderBudget,
  consumePublicQuota,
} from "../lib/wallet-copilot/quota";
import {
  getWalletCopilotStorage,
  resetWalletCopilotStorageForTests,
} from "../lib/wallet-copilot/storage";
import type {
  AnalyzeWalletResponse,
  ProviderDataset,
  WalletIndexCoverage,
} from "../lib/wallet-copilot/types";

test("normalizeDays only accepts supported wallet analysis windows", () => {
  assert.equal(normalizeDays(7), 7);
  assert.equal(normalizeDays(30), 30);
  assert.equal(normalizeDays(90), 90);
  assert.equal(normalizeDays(365), 30);
  assert.equal(normalizeDays("7"), 7);
});

test("classifyUserQuestion routes supported copilot questions", () => {
  assert.equal(
    classifyUserQuestion("Did this wallet interact with DeFi protocols?"),
    "defi_activity",
  );
  assert.equal(
    classifyUserQuestion("What is this wallet holding now?"),
    "portfolio",
  );
  assert.equal(
    classifyUserQuestion("What tokens did this wallet receive the most?"),
    "token_flow",
  );
  assert.equal(
    classifyUserQuestion("Summarize the latest 10 transactions."),
    "latest_transactions",
  );
  assert.equal(
    classifyUserQuestion("Explain this transaction in simple terms."),
    "transaction_explanation",
  );
  assert.equal(
    classifyUserQuestion("Is this wallet more like a trader or holder?"),
    "behavior_profile",
  );
  assert.equal(classifyUserQuestion("Who owns this wallet?"), "unknown");
  assert.equal(
    classifyUserQuestion("Did it bridge assets to Base?"),
    "bridge_activity",
  );
  assert.equal(
    classifyUserQuestion("Show on-chain performance."),
    "onchain_performance",
  );
});

test("anonymous beta quota rejects analysis above the configured daily limit", async () => {
  resetWalletCopilotStorageForTests();
  const previous = process.env.QORVI_ANON_ANALYSIS_DAILY_LIMIT;
  process.env.QORVI_ANON_ANALYSIS_DAILY_LIMIT = "2";
  const request = new Request("https://qorvi.test/api/wallet/analyze/jobs", {
    headers: { "x-forwarded-for": "203.0.113.20" },
  });

  assert.equal((await consumePublicQuota(request, "analysis")).remaining, 1);
  assert.equal((await consumePublicQuota(request, "analysis")).remaining, 0);
  await assert.rejects(
    () => consumePublicQuota(request, "analysis"),
    (error) =>
      error instanceof WalletProviderError && error.code === "quota_exceeded",
  );

  process.env.QORVI_ANON_ANALYSIS_DAILY_LIMIT = previous ?? "";
  resetWalletCopilotStorageForTests();
});

test("provider budget returns a structured hard-stop error", async () => {
  resetWalletCopilotStorageForTests();
  const previous = process.env.QORVI_PROVIDER_DAILY_BUDGET;
  process.env.QORVI_PROVIDER_DAILY_BUDGET = "1";
  await consumeProviderBudget();
  await assert.rejects(
    () => consumeProviderBudget(),
    (error) =>
      error instanceof WalletProviderError &&
      error.code === "provider_budget_exceeded",
  );
  process.env.QORVI_PROVIDER_DAILY_BUDGET = previous ?? "";
  resetWalletCopilotStorageForTests();
});

test("Ethereum RPC eth_call retries a 429 response", async () => {
  const previousRpcUrl = process.env.ETHEREUM_RPC_URL;
  const previousRateLimit = process.env.QORVI_ETH_RPC_RATE_LIMIT_MS;
  const previousRetryDelay = process.env.QORVI_ETH_RPC_RETRY_MS;
  const previousMaxRetries = process.env.QORVI_ETH_RPC_MAX_RETRIES;
  const originalFetch = globalThis.fetch;
  process.env.ETHEREUM_RPC_URL = "https://rpc.test";
  process.env.QORVI_ETH_RPC_RATE_LIMIT_MS = "0";
  process.env.QORVI_ETH_RPC_RETRY_MS = "0";
  process.env.QORVI_ETH_RPC_MAX_RETRIES = "1";
  let calls = 0;
  globalThis.fetch = (async () => {
    calls += 1;
    if (calls === 1) {
      return new Response("rate limited", { status: 429 });
    }
    return new Response(JSON.stringify({ result: "0x1234" }), {
      status: 200,
      headers: { "content-type": "application/json" },
    });
  }) as typeof fetch;

  const result = await ethCall({
    to: "0x0000000000000000000000000000000000000001",
    data: "0x",
  });

  assert.equal(result, "0x1234");
  assert.equal(calls, 2);
  globalThis.fetch = originalFetch;
  process.env.ETHEREUM_RPC_URL = previousRpcUrl ?? "";
  process.env.QORVI_ETH_RPC_RATE_LIMIT_MS = previousRateLimit ?? "";
  process.env.QORVI_ETH_RPC_RETRY_MS = previousRetryDelay ?? "";
  process.env.QORVI_ETH_RPC_MAX_RETRIES = previousMaxRetries ?? "";
});

test("bridge movements require an exact Ethereum allowlist contract match", () => {
  const wallet = "0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d";
  const movements = detectBridgeMovements(wallet, [
    {
      hash: `0x${"b".repeat(64)}`,
      from: wallet,
      to: "0x3154cf16ccdb4c6d922629664174b904d80f2c35",
      value_eth: "1.25",
      timestamp: "2026-05-01T00:00:00.000Z",
      input: "0x",
    },
    {
      hash: `0x${"c".repeat(64)}`,
      from: wallet,
      to: "0x0000000000000000000000000000000000000002",
      value_eth: "2",
      timestamp: "2026-05-01T00:00:00.000Z",
      input: "0x",
    },
  ]);

  assert.equal(movements.length, 1);
  assert.equal(movements[0]?.bridge, "Base Standard Bridge");
  assert.equal(movements[0]?.destination_chain_hint, "Base");
});

test("bridge allowlist reports Stargate routes without inventing destination chain", () => {
  const wallet = "0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d";
  const movements = detectBridgeMovements(wallet, [
    {
      hash: `0x${"d".repeat(64)}`,
      from: wallet,
      to: "0x8731d54e9d02c286767d56ac03e8037c07e01e98",
      value_eth: "0.25",
      timestamp: "2026-05-01T00:00:00.000Z",
      input: "0x",
    },
  ]);

  assert.equal(movements[0]?.bridge, "Stargate Router");
  assert.equal(
    movements[0]?.destination_chain_hint,
    "Destination encoded by Stargate route",
  );
});

test("bridge allowlist covers canonical OP Stack portals and Across Ethereum routes", () => {
  const wallet = "0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d";
  const movements = detectBridgeMovements(wallet, [
    {
      hash: `0x${"e".repeat(64)}`,
      from: wallet,
      to: "0xbEb5Fc579115071764c7423A4f12eDde41f106Ed",
      value_eth: "0.4",
      timestamp: "2026-05-01T00:00:00.000Z",
      input: "0x",
    },
    {
      hash: `0x${"f".repeat(64)}`,
      from: wallet,
      to: "0x49048044D57e1C92A77f79988d21Fa8fAF74E97e",
      value_eth: "0.5",
      timestamp: "2026-05-01T00:00:00.000Z",
      input: "0x",
    },
    {
      hash: `0x${"a".repeat(64)}`,
      from: wallet,
      to: "0x5c7BCd6E7De5423a257D81B442095A1a6ced35C5",
      value_eth: "0.6",
      timestamp: "2026-05-01T00:00:00.000Z",
      input: "0x",
    },
  ]);

  assert.deepEqual(
    movements.map((movement) => [
      movement.bridge,
      movement.destination_chain_hint,
    ]),
    [
      ["Optimism Portal", "OP Mainnet"],
      ["Base OptimismPortal", "Base"],
      ["Across Ethereum SpokePool", "Destination encoded by Across route"],
    ],
  );
});

test("Aave receipt log decoder emits high-confidence supply evidence", () => {
  const wallet = "0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d";
  const token = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48";
  const padded = (value: string) => value.replace(/^0x/, "").padStart(64, "0");
  const actions = decodeReceiptActions({
    wallet,
    receipts: [
      {
        transaction_hash: `0x${"9".repeat(64)}`,
        block_number: 1,
        status: "success",
        source: "ethereum_rpc",
        logs: [
          {
            address: "0x87870bca3f3fd6335c3f4ce8392d69350b4fa4e2",
            topics: [
              "0x2b627736bca15cd5381dcf80b0bf11fd197d01a037c52b927a881a10fb73ba61",
              `0x${padded(token)}`,
            ],
            data: `0x${padded(wallet)}${padded("0x0f4240")}${padded("0x0")}`,
            log_index: 0,
            transaction_hash: `0x${"9".repeat(64)}`,
            block_number: 1,
          },
        ],
      },
    ],
    transfers: [
      {
        hash: `0x${"9".repeat(64)}`,
        from: wallet,
        to: "0x87870bca3f3fd6335c3f4ce8392d69350b4fa4e2",
        token_symbol: "USDC",
        token_name: "USD Coin",
        token_address: token,
        value: "1",
        decimals: 6,
        timestamp: "2026-05-01T00:00:00.000Z",
      },
    ],
  });

  assert.equal(actions[0]?.protocol, "Aave V3");
  assert.equal(actions[0]?.action_type, "supply");
  assert.equal(actions[0]?.amount, "1");
  assert.equal(actions[0]?.confidence, "high");
  assert.equal(actions[0]?.decoder_source, "receipt_log");
});

test("Aave receipt decoder does not report an amount without transfer evidence", () => {
  const actions = decodeReceiptActions({
    wallet: "0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d",
    receipts: [
      {
        transaction_hash: `0x${"7".repeat(64)}`,
        block_number: 1,
        status: "success",
        source: "ethereum_rpc",
        logs: [
          {
            address: "0x87870bca3f3fd6335c3f4ce8392d69350b4fa4e2",
            topics: [
              "0x2b627736bca15cd5381dcf80b0bf11fd197d01a037c52b927a881a10fb73ba61",
              `0x${"a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48".padStart(64, "0")}`,
            ],
            data: "0x",
            log_index: 0,
            transaction_hash: `0x${"7".repeat(64)}`,
            block_number: 1,
          },
        ],
      },
    ],
    transfers: [],
  });

  assert.equal(actions.length, 0);
});

test("Curve pool receipt log decoder emits liquidity actions from transfer evidence", () => {
  const wallet = "0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d";
  const txHash = `0x${"8".repeat(64)}`;
  const padded = (value: string) => value.replace(/^0x/, "").padStart(64, "0");
  const actions = decodeReceiptActions({
    wallet,
    receipts: [
      {
        transaction_hash: txHash,
        block_number: 1,
        status: "success",
        source: "ethereum_rpc",
        logs: [
          {
            address: "0xdc24316b9ae028f1497c275eb9192a3ea0f67022",
            topics: [
              "0x189c623b666b1b45b83d7178f39b8c087cb09774317ca2f53c2d3c3726f222a2",
              `0x${padded(wallet)}`,
            ],
            data: "0x",
            log_index: 0,
            transaction_hash: txHash,
            block_number: 1,
          },
        ],
      },
    ],
    transfers: [
      {
        hash: txHash,
        from: wallet,
        to: "0xdc24316b9ae028f1497c275eb9192a3ea0f67022",
        token_symbol: "ETH",
        token_name: "Ether",
        token_address: "0x0000000000000000000000000000000000000000",
        value: "2",
        decimals: 18,
        timestamp: "2026-05-01T00:00:00.000Z",
      },
    ],
  });

  assert.equal(actions[0]?.protocol, "Curve Pool");
  assert.equal(actions[0]?.action_type, "add_liquidity");
  assert.equal(actions[0]?.direction, "outbound");
  assert.equal(actions[0]?.decoder_source, "receipt_log");
});

test("Aave reserve discovery ABI decoder extracts dynamic symbol tuples", () => {
  const word = (hex: string) => hex.replace(/^0x/, "").padStart(64, "0");
  const usdcAscii = Buffer.from("USDC").toString("hex").padEnd(64, "0");
  const encoded = `0x${word("20")}${word("1")}${word("20")}${word("40")}${word(
    "a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
  )}${word("4")}${usdcAscii}`;
  const assets = decodeAaveReserveTokens(encoded);

  assert.equal(assets[0]?.symbol, "USDC");
  assert.equal(
    assets[0]?.address,
    "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
  );
});

test("historical price points use deterministic UTC hour cache buckets", () => {
  assert.equal(
    historicalPriceBucket("2026-05-26T03:42:21.000Z"),
    "2026-05-26T03:00:00.000Z",
  );
});

test("grounding verifier rejects invented transaction identifiers", () => {
  const actualHash = `0x${"1".repeat(64)}`;
  assert.equal(
    verifyGroundedIdentifiers(
      `Evidence hash: ${actualHash}.`,
      { tx_hash: actualHash },
      [],
    ),
    true,
  );
  assert.equal(
    verifyGroundedIdentifiers(
      `Evidence hash: 0x${"2".repeat(64)}.`,
      { tx_hash: actualHash },
      [],
    ),
    false,
  );
});

test("wallet analysis cache reuses fresh entries and normalizes keys", async () => {
  clearWalletAnalysisCache();
  const key = buildWalletAnalysisCacheKey({
    address: "0xF00a90FB0129d61DD09194BF70759CD5D36E3d2D",
    days: 30,
  });
  let loadCount = 0;
  const load = async () => {
    loadCount += 1;
    return { ok: true, loadCount };
  };

  const first = await getOrSetWalletAnalysisCache({
    key,
    load,
    now: 1_000,
    ttlMs: 10_000,
  });
  const second = await getOrSetWalletAnalysisCache({
    key,
    load,
    now: 2_000,
    ttlMs: 10_000,
  });

  assert.equal(key, "auto:0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d:30");
  assert.deepEqual(first, second);
  assert.equal(loadCount, 1);
});

test("wallet analysis cache dedupes inflight loads", async () => {
  clearWalletAnalysisCache();
  let loadCount = 0;
  const key = "wallet:30";
  const load = async () => {
    loadCount += 1;
    await new Promise((resolve) => setTimeout(resolve, 5));
    return { value: loadCount };
  };

  const [first, second] = await Promise.all([
    getOrSetWalletAnalysisCache({ key, load, ttlMs: 10_000 }),
    getOrSetWalletAnalysisCache({ key, load, ttlMs: 10_000 }),
  ]);

  assert.deepEqual(first, second);
  assert.equal(loadCount, 1);
});

test("chat requires an existing analysis snapshot and never starts collection", async () => {
  clearWalletAnalysisCache();
  resetWalletCopilotStorageForTests();
  await assert.rejects(
    () =>
      answerWalletQuestion({
        address: "0xF00a90FB0129d61DD09194BF70759CD5D36E3d2D",
        days: 30,
        question: "What did this wallet do recently?",
      }),
    (error) =>
      error instanceof WalletProviderError &&
      error.code === "analysis_required",
  );
});

test("wallet analysis jobs return cached analysis immediately when available", async () => {
  clearWalletAnalysisCache();
  resetWalletCopilotStorageForTests();
  const address = "0xF00a90FB0129d61DD09194BF70759CD5D36E3d2D";
  const key = buildWalletAnalysisCacheKey({ address, days: 30 });
  const cachedAnalysis = {
    address,
    period_days: 30,
    generated_at: "2026-01-01T00:00:00.000Z",
    provider: "etherscan",
    data_mode: "live",
    data_notice: "test",
    summary: {
      total_transactions: 0,
      erc20_transfer_count: 0,
      unique_counterparties: 0,
      most_active_tokens: [],
      main_activity_types: [],
      risk_level: "low",
    },
    analysis: {
      token_flows: [],
      swaps: [],
      protocol_interactions: [],
      defi_actions: [],
      portfolio: {},
      defi_positions: {},
      cex_transfer_hints: [],
      risk_flags: [],
      behavior_profile: { labels: [], rationale: [] },
    },
    ai_report: "Cached report",
    evidence: [],
  } as unknown as AnalyzeWalletResponse;
  await setWalletAnalysisCache(key, cachedAnalysis, 10_000);

  const { createWalletAnalysisJob } = await import(
    "../lib/wallet-copilot/jobs"
  );
  const job = await createWalletAnalysisJob({
    address,
    daysInput: 30,
  });

  assert.equal(job.status, "succeeded");
  assert.equal(job.cached, true);
  assert.equal(job.result?.ai_report, "Cached report");
});

test("partial lifetime cached analyses resume indexing without charging quota", async () => {
  clearWalletAnalysisCache();
  resetWalletCopilotStorageForTests();
  const previousExecutionMode =
    process.env.QORVI_WALLET_ANALYSIS_EXECUTION_MODE;
  process.env.QORVI_WALLET_ANALYSIS_EXECUTION_MODE = "worker";
  const address = "0xF00a90FB0129d61DD09194BF70759CD5D36E3d2D";
  const key = buildWalletAnalysisCacheKey({ address, days: 30 });
  const cachedAnalysis = {
    address,
    period_days: 30,
    index_coverage: {
      scope: "lifetime",
      stage: "backfilling",
      completeness: "partial",
      lifetime_start_block: 1,
      lifetime_end_block: 500_000,
      indexed_start_block: 300_000,
      indexed_end_block: 500_000,
      unsupported_event_count: 0,
      limitation: "Indexing remains.",
    },
    performance_status: "partial",
  } as unknown as AnalyzeWalletResponse;
  await setWalletAnalysisCache(key, cachedAnalysis, 10_000);
  let quotaConsumed = false;
  const { createWalletAnalysisJob, resetWalletAnalysisJobsForTests } =
    await import("../lib/wallet-copilot/jobs");
  resetWalletAnalysisJobsForTests();

  const job = await createWalletAnalysisJob({
    address,
    daysInput: 30,
    consumeQuota: async () => {
      quotaConsumed = true;
      throw new Error("Cached continuation must not spend quota.");
    },
  });

  assert.equal(job.status, "queued");
  assert.equal(job.cached, true);
  assert.equal(job.result?.index_coverage.completeness, "partial");
  assert.equal(quotaConsumed, false);

  process.env.QORVI_WALLET_ANALYSIS_EXECUTION_MODE =
    previousExecutionMode ?? "";
  resetWalletAnalysisJobsForTests();
  resetWalletCopilotStorageForTests();
});

test("wallet analysis worker mode queues jobs and processes them separately", async () => {
  clearWalletAnalysisCache();
  resetWalletCopilotStorageForTests();
  const previousExecutionMode =
    process.env.QORVI_WALLET_ANALYSIS_EXECUTION_MODE;
  const previousKey = process.env.ETHERSCAN_API_KEY;
  const previousProvider = process.env.QORVI_WALLET_PROVIDER;
  process.env.QORVI_WALLET_ANALYSIS_EXECUTION_MODE = "worker";
  process.env.QORVI_WALLET_PROVIDER = "etherscan";
  process.env.ETHERSCAN_API_KEY = "";

  const {
    createWalletAnalysisJob,
    getWalletAnalysisJob,
    processWalletAnalysisQueue,
    resetWalletAnalysisJobsForTests,
  } = await import("../lib/wallet-copilot/jobs");
  resetWalletAnalysisJobsForTests();

  const created = await createWalletAnalysisJob({
    address: "0xF00a90FB0129d61DD09194BF70759CD5D36E3d2D",
    daysInput: 30,
  });
  const queued = await getWalletAnalysisJob(created.job_id);

  assert.equal(created.status, "queued");
  assert.equal(queued?.status, "queued");
  assert.equal(queued?.execution_mode, "worker");

  const workerResult = await processWalletAnalysisQueue({ limit: 1 });
  const completed = await getWalletAnalysisJob(created.job_id);

  assert.equal(workerResult.processed, 1);
  assert.equal(workerResult.failed, 1);
  assert.equal(completed?.status, "failed");
  assert.equal(completed?.error?.code, "missing_api_key");

  if (previousExecutionMode) {
    process.env.QORVI_WALLET_ANALYSIS_EXECUTION_MODE = previousExecutionMode;
  } else {
    process.env.QORVI_WALLET_ANALYSIS_EXECUTION_MODE = "";
  }
  if (previousKey) {
    process.env.ETHERSCAN_API_KEY = previousKey;
  } else {
    process.env.ETHERSCAN_API_KEY = "";
  }
  process.env.QORVI_WALLET_PROVIDER = previousProvider ?? "";
});

test("wallet copilot storage can use REDIS_URL for GCP Redis", async () => {
  const previousRedisUrl = process.env.REDIS_URL;
  const previousUpstashUrl = process.env.UPSTASH_REDIS_REST_URL;
  const previousUpstashToken = process.env.UPSTASH_REDIS_REST_TOKEN;
  const store = new Map<string, string | string[]>();
  const server = createServer((socket) => {
    socket.on("data", (chunk) => {
      const command = parseRespCommand(chunk);
      const [name = "", key = "", ...args] = command;
      switch (name.toUpperCase()) {
        case "GET": {
          const value = store.get(key);
          socket.write(
            typeof value === "string" ? bulkString(value) : "$-1\r\n",
          );
          return;
        }
        case "SET": {
          store.set(key, args[0] ?? "");
          socket.write("+OK\r\n");
          return;
        }
        case "DEL": {
          const deleted = store.delete(key) ? 1 : 0;
          socket.write(`:${deleted}\r\n`);
          return;
        }
        case "RPUSH": {
          const current = store.get(key);
          const list = Array.isArray(current) ? current : [];
          list.push(args[0] ?? "");
          store.set(key, list);
          socket.write(`:${list.length}\r\n`);
          return;
        }
        case "LPOP": {
          const current = store.get(key);
          if (!Array.isArray(current) || current.length === 0) {
            socket.write("$-1\r\n");
            return;
          }
          const value = current.shift() ?? "";
          store.set(key, current);
          socket.write(bulkString(value));
          return;
        }
        case "EXPIRE":
          socket.write(":1\r\n");
          return;
        case "INCR": {
          const current = Number(store.get(key) ?? "0");
          const next = current + 1;
          store.set(key, String(next));
          socket.write(`:${next}\r\n`);
          return;
        }
        default:
          socket.write(`-ERR unsupported command ${name}\r\n`);
      }
    });
  });

  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  assert.ok(address && typeof address === "object");

  process.env.REDIS_URL = `redis://127.0.0.1:${address.port}/0`;
  process.env.UPSTASH_REDIS_REST_URL = "";
  process.env.UPSTASH_REDIS_REST_TOKEN = "";
  resetWalletCopilotStorageForTests();

  const storage = getWalletCopilotStorage();
  assert.equal(storage.kind, "redis");
  await storage.setJson("qorvi:test", { ok: true }, { ttlSeconds: 60 });
  assert.deepEqual(await storage.getJson("qorvi:test"), { ok: true });
  await storage.pushJson("qorvi:queue", "job-1", { ttlSeconds: 60 });
  assert.equal(await storage.popJson("qorvi:queue"), "job-1");

  await new Promise<void>((resolve) => server.close(() => resolve()));
  process.env.REDIS_URL = previousRedisUrl ?? "";
  process.env.UPSTASH_REDIS_REST_URL = previousUpstashUrl ?? "";
  process.env.UPSTASH_REDIS_REST_TOKEN = previousUpstashToken ?? "";
  resetWalletCopilotStorageForTests();
});

test("contract read helpers format token units deterministically", () => {
  assert.equal(formatUnits(1_000_000n, 6), "1");
  assert.equal(formatUnits(1_234_567_890_000_000_000n, 18), "1.23456789");
});

test("Uniswap V3 math returns deterministic tick and liquidity amounts", () => {
  assert.equal(getSqrtRatioAtTick(0), 79_228_162_514_264_337_593_543_950_336n);
  const amounts = calculateUniswapV3Amounts({
    sqrtPriceX96: getSqrtRatioAtTick(0),
    sqrtRatioAX96: getSqrtRatioAtTick(-60),
    sqrtRatioBX96: getSqrtRatioAtTick(60),
    liquidity: 1_000_000n,
    currentTick: 0,
    tickLower: -60,
    tickUpper: 60,
  });
  assert.ok(amounts.amount0Raw > 0n);
  assert.ok(amounts.amount1Raw > 0n);
});

test("Curve pool selector prioritizes direct wallet candidates before TVL ranked pools", () => {
  const pools = [
    {
      name: "Big pool",
      address: "0x0000000000000000000000000000000000000001",
      lpTokenAddress: "0x0000000000000000000000000000000000000011",
      gaugeAddress: "0x0000000000000000000000000000000000000111",
      usdTotal: 1_000_000,
    },
    {
      name: "Wallet pool",
      address: "0x0000000000000000000000000000000000000002",
      lpTokenAddress: "0x0000000000000000000000000000000000000022",
      gaugeAddress: "0x0000000000000000000000000000000000000222",
      usdTotal: 10,
    },
  ];

  const selected = selectCurveCandidatePools({
    pools,
    candidateAddresses: ["0x0000000000000000000000000000000000000022"],
    maxPools: 1,
  });

  assert.equal(selected.length, 1);
  assert.equal(selected[0]?.name, "Wallet pool");
});

test("PnL summary tracks FIFO unrealized PnL for detected swaps", () => {
  const summary = buildPnlSummary({
    swaps: [
      {
        protocol: "Uniswap",
        sent_token_symbol: "USDC",
        sent_token_address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
        sent_amount: "100",
        received_token_symbol: "ABC",
        received_token_address: "0x00000000000000000000000000000000000000ab",
        received_amount: "10",
        tx_hash: `0x${"1".repeat(64)}`,
        timestamp: "2026-01-01T00:00:00.000Z",
      },
    ],
    dataset: {
      tokenBalances: [
        {
          token_symbol: "ABC",
          token_address: "0x00000000000000000000000000000000000000ab",
          balance: "10",
          decimals: 18,
        },
      ],
      tokenPricesUsd: {
        "0x00000000000000000000000000000000000000ab": 12,
      },
    } as never,
  });

  assert.equal(summary.status, "available");
  assert.equal(summary.unrealized_pnl_usd, 20);
  assert.equal(summary.total_pnl_usd, 20);
});

test("complete lifetime performance values externally received ETH at event time", () => {
  const wallet = "0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d";
  const dataset = buildLifetimePerformanceDataset(wallet);
  const analysis = buildWalletAnalysis(
    wallet,
    30,
    dataset,
    buildCompleteLifetimeCoverage(),
    dataset,
  );

  assert.equal(analysis.analysis.onchain_performance.status, "complete");
  assert.equal(
    analysis.analysis.onchain_performance.external_inflows_usd,
    2000,
  );
  assert.equal(analysis.analysis.onchain_performance.unpriced_event_count, 0);
});

test("bridge movements keep lifetime performance partial without destination value", () => {
  const wallet = "0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d";
  const dataset = buildLifetimePerformanceDataset(wallet);
  dataset.transactions.push({
    hash: `0x${"2".repeat(64)}`,
    from: wallet,
    to: "0x3154cf16ccdb4c6d922629664174b904d80f2c35",
    value_eth: "0.1",
    timestamp: "2026-01-01T00:00:00.000Z",
    input: "0x1234",
  });
  const analysis = buildWalletAnalysis(
    wallet,
    30,
    dataset,
    buildCompleteLifetimeCoverage(),
    dataset,
  );

  assert.equal(analysis.analysis.onchain_performance.status, "partial");
  assert.equal(analysis.analysis.onchain_performance.bridge_movement_count, 1);
  assert.equal(
    analysis.analysis.onchain_performance.unsupported_event_count,
    1,
  );
});

test("unpriced external lifetime movement prevents complete performance", () => {
  const wallet = "0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d";
  const dataset = buildLifetimePerformanceDataset(wallet);
  const [firstTransaction] = dataset.transactions;
  assert.ok(firstTransaction);
  firstTransaction.timestamp = "2026-01-01T01:00:00.000Z";
  const analysis = buildWalletAnalysis(
    wallet,
    30,
    dataset,
    buildCompleteLifetimeCoverage(),
    dataset,
  );

  assert.equal(analysis.analysis.onchain_performance.status, "partial");
  assert.equal(analysis.analysis.onchain_performance.unpriced_event_count, 1);
});

test("selected-window data cannot produce complete lifetime performance", () => {
  const wallet = "0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d";
  const dataset = buildLifetimePerformanceDataset(wallet);
  dataset.performanceLedgerScope = "selected_window";
  const analysis = buildWalletAnalysis(
    wallet,
    30,
    dataset,
    {
      ...buildCompleteLifetimeCoverage(),
      scope: "selected_window",
    },
    dataset,
  );

  assert.equal(analysis.analysis.onchain_performance.status, "partial");
});

function buildCompleteLifetimeCoverage(): WalletIndexCoverage {
  return {
    scope: "lifetime",
    stage: "succeeded",
    completeness: "complete",
    lifetime_start_block: 1,
    lifetime_end_block: 2,
    indexed_start_block: 1,
    indexed_end_block: 2,
    receipt_log_coverage: "complete",
    historical_price_coverage: "complete",
    unsupported_event_count: 0,
    limitation: null,
  };
}

function buildLifetimePerformanceDataset(wallet: string): ProviderDataset {
  return {
    provider: "etherscan",
    data_mode: "live",
    data_notice: "test",
    transactions: [
      {
        hash: `0x${"1".repeat(64)}`,
        from: "0x0000000000000000000000000000000000000001",
        to: wallet,
        value_eth: "1",
        timestamp: "2026-01-01T00:00:00.000Z",
        input: "0x",
      },
    ],
    erc20Transfers: [],
    receipts: [],
    receiptCoverage: "complete",
    historicalPrices: [
      {
        asset_address: "eth",
        timestamp: "2026-01-01T00:00:00.000Z",
        provider: "coingecko",
        value_usd: 2000,
        available: true,
      },
    ],
    historicalPriceCoverage: "complete",
    performanceLedgerScope: "lifetime",
    nativeBalanceEth: "1",
    tokenBalances: [],
    tokenPricesUsd: { eth: 2000 },
    defiPositions: {
      current_positions_status: "unavailable",
      lp_positions_status: "unavailable",
      total_supplied_usd: null,
      total_borrowed_usd: null,
      total_lp_value_usd: null,
      aave_positions: [],
      uniswap_v3_positions: [],
      curve_positions: [],
      explanation: "test",
      sources: [],
      errors: [],
    },
  };
}

test("DeFi position readers expose an explicit disabled state for tests", async () => {
  const previous = process.env.QORVI_DEFI_POSITION_READERS;
  process.env.QORVI_DEFI_POSITION_READERS = "0";
  const { readLiveDefiPositions } = await import(
    "../lib/wallet-copilot/defi-position-readers"
  );
  const result = await readLiveDefiPositions(
    "0xF00a90FB0129d61DD09194BF70759CD5D36E3d2D",
  );

  assert.equal(result.current_positions_status, "unavailable");
  assert.equal(result.lp_positions_status, "unavailable");
  assert.equal(result.aave_positions.length, 0);

  if (previous) {
    process.env.QORVI_DEFI_POSITION_READERS = previous;
  } else {
    process.env.QORVI_DEFI_POSITION_READERS = "";
  }
});

test("getWalletDataset requires Etherscan API key", async () => {
  const previousKey = process.env.ETHERSCAN_API_KEY;
  const previousProvider = process.env.QORVI_WALLET_PROVIDER;
  process.env.QORVI_WALLET_PROVIDER = "etherscan";
  process.env.ETHERSCAN_API_KEY = "";
  const { getWalletDataset } = await import("../lib/wallet-copilot/provider");

  await assert.rejects(
    () => getWalletDataset("0xF00a90FB0129d61DD09194BF70759CD5D36E3d2D", 30),
    (error) =>
      error instanceof WalletProviderError &&
      error.code === "missing_api_key" &&
      error.status === 401,
  );

  if (previousKey) {
    process.env.ETHERSCAN_API_KEY = previousKey;
  } else {
    process.env.ETHERSCAN_API_KEY = "";
  }
  process.env.QORVI_WALLET_PROVIDER = previousProvider ?? "";
});

test("getWalletDataset uses timestamp block range and paginates Etherscan rows", async () => {
  const previousKey = process.env.ETHERSCAN_API_KEY;
  const previousProvider = process.env.QORVI_WALLET_PROVIDER;
  const previousRateLimit = process.env.QORVI_ETHERSCAN_RATE_LIMIT_MS;
  const previousPositionReaders = process.env.QORVI_DEFI_POSITION_READERS;
  process.env.QORVI_WALLET_PROVIDER = "etherscan";
  process.env.ETHERSCAN_API_KEY = "test-key";
  process.env.QORVI_ETHERSCAN_RATE_LIMIT_MS = "0";
  process.env.QORVI_DEFI_POSITION_READERS = "0";
  const calls: URLSearchParams[] = [];
  const timestamp = Math.floor(Date.now() / 1000).toString();
  const wallet = "0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d";
  const txPage = Array.from({ length: 1000 }, (_, index) => ({
    hash: `0x${(index + 1).toString(16).padStart(64, "0")}`,
    from: wallet,
    to: "0x0000000000000000000000000000000000000001",
    value: "0",
    timeStamp: timestamp,
    input: "0x",
    functionName: "",
    isError: "0",
    blockNumber: "150",
  }));
  const transfer = {
    hash: `0x${"a".repeat(64)}`,
    from: "0x0000000000000000000000000000000000000001",
    to: wallet,
    contractAddress: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
    tokenSymbol: "USDC",
    tokenName: "USD Coin",
    tokenDecimal: "6",
    value: "1000000",
    timeStamp: timestamp,
    blockNumber: "151",
  };
  const previousFetch = globalThis.fetch;
  globalThis.fetch = (async (input: RequestInfo | URL) => {
    const url = new URL(input.toString());
    const params = url.searchParams;
    calls.push(new URLSearchParams(params));
    const action = params.get("action");
    const page = params.get("page");
    if (action === "getblocknobytime") {
      const closest = params.get("closest");
      return jsonResponse({
        status: "1",
        message: "OK",
        result: closest === "after" ? "100" : "200",
      });
    }
    if (action === "txlist" && page === "1") {
      return jsonResponse({ status: "1", message: "OK", result: txPage });
    }
    if (action === "txlist" && page === "2") {
      return jsonResponse({ status: "1", message: "OK", result: [txPage[0]] });
    }
    if (action === "tokentx") {
      return jsonResponse({
        status: "1",
        message: "OK",
        result: page === "1" ? [transfer] : [],
      });
    }
    if (action === "balance") {
      return jsonResponse({
        status: "1",
        message: "OK",
        result: "1000000000000000000",
      });
    }
    if (action === "tokenbalance") {
      return jsonResponse({
        status: "1",
        message: "OK",
        result: "1000000",
      });
    }
    return jsonResponse({
      status: "0",
      message: "NOTOK",
      result: "unexpected action",
    });
  }) as typeof fetch;

  const { getWalletDataset } = await import("../lib/wallet-copilot/provider");
  const dataset = await getWalletDataset(wallet, 30);

  assert.equal(dataset.provider, "etherscan");
  assert.equal(dataset.data_mode, "live");
  assert.equal(dataset.transactions.length, 1001);
  assert.equal(dataset.erc20Transfers.length, 1);
  assert.equal(calls[0]?.get("action"), "getblocknobytime");
  assert.equal(calls[0]?.get("closest"), "after");
  assert.equal(calls[1]?.get("closest"), "before");
  const txCalls = calls.filter((call) => call.get("action") === "txlist");
  const tokenCalls = calls.filter((call) => call.get("action") === "tokentx");
  assert.equal(txCalls[0]?.get("startblock"), "100");
  assert.equal(txCalls[0]?.get("endblock"), "200");
  assert.equal(txCalls[0]?.get("page"), "1");
  assert.equal(txCalls[1]?.get("page"), "2");
  assert.equal(tokenCalls[0]?.get("startblock"), "100");
  assert.equal(dataset.transactions[0]?.block_number, 150);
  assert.equal(dataset.erc20Transfers[0]?.block_number, 151);

  globalThis.fetch = previousFetch;
  if (previousKey) {
    process.env.ETHERSCAN_API_KEY = previousKey;
  } else {
    process.env.ETHERSCAN_API_KEY = "";
  }
  process.env.QORVI_WALLET_PROVIDER = previousProvider ?? "";
  if (previousRateLimit) {
    process.env.QORVI_ETHERSCAN_RATE_LIMIT_MS = previousRateLimit;
  } else {
    process.env.QORVI_ETHERSCAN_RATE_LIMIT_MS = "";
  }
  if (previousPositionReaders) {
    process.env.QORVI_DEFI_POSITION_READERS = previousPositionReaders;
  } else {
    process.env.QORVI_DEFI_POSITION_READERS = "";
  }
});

test("getWalletDataset can use Alchemy live provider fallback", async () => {
  const previousProvider = process.env.QORVI_WALLET_PROVIDER;
  const previousAlchemyKey = process.env.ALCHEMY_API_KEY;
  const previousAlchemyUrl = process.env.ALCHEMY_BASE_URL;
  const previousPositionReaders = process.env.QORVI_DEFI_POSITION_READERS;
  process.env.QORVI_WALLET_PROVIDER = "alchemy";
  process.env.ALCHEMY_API_KEY = "test-alchemy-key";
  process.env.ALCHEMY_BASE_URL = "https://eth-mainnet.g.alchemy.com";
  process.env.QORVI_DEFI_POSITION_READERS = "0";

  const wallet = "0xf00a90fb0129d61dd09194bf70759cd5d36e3d2d";
  const requestedUrls: string[] = [];
  const previousFetch = globalThis.fetch;
  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    requestedUrls.push(input.toString());
    const body = JSON.parse(String(init?.body ?? "{}")) as {
      method?: string;
      params?: unknown[];
    };
    if (body.method === "alchemy_getAssetTransfers") {
      const params = body.params?.[0] as { fromAddress?: string };
      const direction = params.fromAddress ? "outbound" : "inbound";
      return jsonResponse({
        jsonrpc: "2.0",
        id: 1,
        result: {
          transfers:
            direction === "outbound"
              ? [
                  {
                    blockNum: "0x1",
                    hash: `0x${"1".repeat(64)}`,
                    from: wallet,
                    to: "0x0000000000000000000000000000000000000001",
                    value: 0.5,
                    asset: "ETH",
                    category: "external",
                    metadata: { blockTimestamp: new Date().toISOString() },
                    rawContract: {
                      value: "0x6f05b59d3b20000",
                      address: null,
                      decimal: "0x12",
                    },
                  },
                ]
              : [
                  {
                    blockNum: "0x2",
                    hash: `0x${"2".repeat(64)}`,
                    from: "0x0000000000000000000000000000000000000002",
                    to: wallet,
                    value: 1,
                    asset: "USDC",
                    category: "erc20",
                    metadata: { blockTimestamp: new Date().toISOString() },
                    rawContract: {
                      value: "0xf4240",
                      address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
                      decimal: "0x6",
                    },
                  },
                ],
        },
      });
    }
    if (body.method === "eth_getBalance") {
      return jsonResponse({
        jsonrpc: "2.0",
        id: 1,
        result: "0xde0b6b3a7640000",
      });
    }
    if (body.method === "alchemy_getTokenBalances") {
      return jsonResponse({
        jsonrpc: "2.0",
        id: 1,
        result: {
          address: wallet,
          tokenBalances: [
            {
              contractAddress: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
              tokenBalance: "0x1e8480",
            },
          ],
        },
      });
    }
    return jsonResponse({ ethereum: { usd: 3000 } });
  }) as typeof fetch;

  const { getWalletDataset } = await import("../lib/wallet-copilot/provider");
  const dataset = await getWalletDataset(wallet, 7);

  assert.equal(dataset.provider, "alchemy");
  assert.equal(dataset.transactions.length, 1);
  assert.equal(dataset.transactions[0]?.value_eth, "0.5");
  assert.equal(dataset.erc20Transfers.length, 1);
  assert.equal(dataset.erc20Transfers[0]?.token_symbol, "USDC");
  assert.equal(dataset.erc20Transfers[0]?.value, "1");
  assert.equal(dataset.nativeBalanceEth, "1");
  assert.equal(dataset.tokenBalances[0]?.balance, "2");
  assert.ok(
    requestedUrls.some((url) =>
      url.startsWith("https://eth-mainnet.g.alchemy.com/v2/test-alchemy-key"),
    ),
  );

  globalThis.fetch = previousFetch;
  process.env.QORVI_WALLET_PROVIDER = previousProvider ?? "";
  process.env.ALCHEMY_API_KEY = previousAlchemyKey ?? "";
  process.env.ALCHEMY_BASE_URL = previousAlchemyUrl ?? "";
  process.env.QORVI_DEFI_POSITION_READERS = previousPositionReaders ?? "";
});

function jsonResponse(payload: unknown): Response {
  return new Response(JSON.stringify(payload), {
    headers: { "content-type": "application/json" },
  });
}

function parseRespCommand(buffer: Buffer): string[] {
  const raw = buffer.toString("utf8");
  const parts = raw.split("\r\n");
  const values: string[] = [];
  for (let index = 1; index < parts.length; index += 2) {
    if (!parts[index]?.startsWith("$")) {
      continue;
    }
    const value = parts[index + 1];
    if (value !== undefined) {
      values.push(value);
    }
  }
  return values;
}

function bulkString(value: string): string {
  return `$${Buffer.byteLength(value)}\r\n${value}\r\n`;
}
