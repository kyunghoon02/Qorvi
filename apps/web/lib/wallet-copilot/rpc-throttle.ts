import { readPositiveNumberEnv } from "./provider-utils";

let nextRpcRequestAt = 0;
let rpcThrottleChain: Promise<void> = Promise.resolve();

export async function throttleEthereumRpcRequest(): Promise<void> {
  const minIntervalMs = readPositiveNumberEnv(
    "QORVI_ETH_RPC_RATE_LIMIT_MS",
    350,
    { allowZero: true },
  );
  if (minIntervalMs === 0) {
    return;
  }
  const run = async () => {
    const now = Date.now();
    if (now < nextRpcRequestAt) {
      await sleep(nextRpcRequestAt - now);
    }
    nextRpcRequestAt = Date.now() + minIntervalMs;
  };
  rpcThrottleChain = rpcThrottleChain.then(run, run);
  return rpcThrottleChain;
}

export async function waitForEthereumRpcRetry(attempt: number): Promise<void> {
  const baseDelayMs = readPositiveNumberEnv("QORVI_ETH_RPC_RETRY_MS", 1500, {
    allowZero: true,
  });
  if (baseDelayMs === 0) {
    return;
  }
  await sleep(baseDelayMs * Math.max(attempt + 1, 1));
}

export function resolveEthereumRpcMaxRetries(): number {
  return Math.floor(
    readPositiveNumberEnv("QORVI_ETH_RPC_MAX_RETRIES", 2, {
      allowZero: true,
    }),
  );
}

export function isRateLimitMessage(message: string | undefined): boolean {
  const normalized = message?.toLowerCase() ?? "";
  return (
    normalized.includes("rate limit") ||
    normalized.includes("too many requests") ||
    normalized.includes("429")
  );
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
