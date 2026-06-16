import { WalletProviderError } from "./errors";

const defaultTimeoutMs = readPositiveNumberEnv(
  "QORVI_PROVIDER_TIMEOUT_MS",
  12_000,
);

export async function fetchWithTimeout(
  input: string,
  init: RequestInit = {},
): Promise<Response> {
  const timeoutMs = readPositiveNumberEnv(
    "QORVI_PROVIDER_TIMEOUT_MS",
    defaultTimeoutMs,
  );
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(input, {
      ...init,
      signal: init.signal ?? controller.signal,
    });
  } catch (error) {
    if (error instanceof Error && error.name === "AbortError") {
      throw new WalletProviderError({
        code: "provider_timeout",
        message: `Provider request timed out after ${timeoutMs}ms.`,
        status: 504,
      });
    }
    throw error;
  } finally {
    clearTimeout(timeout);
  }
}

export async function fetchJsonWithTimeout<T>(
  input: string,
  init: RequestInit = {},
): Promise<T> {
  const response = await fetchWithTimeout(input, init);
  if (!response.ok) {
    throw new WalletProviderError({
      code: response.status === 429 ? "rate_limited" : "provider_unavailable",
      message: `Provider HTTP ${response.status}`,
      status: response.status === 429 ? 429 : 503,
    });
  }
  return (await response.json()) as T;
}

function readPositiveNumberEnv(name: string, fallback: number): number {
  const raw = process.env[name]?.trim();
  if (!raw) {
    return fallback;
  }
  const value = Number(raw);
  return Number.isFinite(value) && value > 0 ? value : fallback;
}
