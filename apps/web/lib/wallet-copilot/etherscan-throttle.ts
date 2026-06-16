import { consumeProviderBudget } from "./quota";

const etherscanMinIntervalMs = readPositiveNumberEnv(
  "QORVI_ETHERSCAN_RATE_LIMIT_MS",
  500,
  { allowZero: true },
);

let etherscanRateLimitQueue = Promise.resolve();
let lastEtherscanRequestAt = 0;

export async function throttleEtherscanRequest(): Promise<void> {
  await consumeProviderBudget();
  const previous = etherscanRateLimitQueue;
  let release: () => void = () => {};
  etherscanRateLimitQueue = new Promise((resolve) => {
    release = resolve;
  });

  await previous;
  const elapsed = Date.now() - lastEtherscanRequestAt;
  if (elapsed < etherscanMinIntervalMs) {
    await new Promise((resolve) =>
      setTimeout(resolve, etherscanMinIntervalMs - elapsed),
    );
  }
  lastEtherscanRequestAt = Date.now();
  release();
}

function readPositiveNumberEnv(
  name: string,
  fallback: number,
  options: { allowZero?: boolean } = {},
): number {
  const raw = process.env[name]?.trim();
  if (!raw) {
    return fallback;
  }
  const value = Number(raw);
  if (!Number.isFinite(value)) {
    return fallback;
  }
  if (value === 0 && options.allowZero) {
    return value;
  }
  return value > 0 ? value : fallback;
}
