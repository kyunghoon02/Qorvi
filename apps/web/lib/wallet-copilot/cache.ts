import { getWalletCopilotStorage } from "./storage";

type CacheEntry<T> = {
  value: T;
  expiresAt: number;
};

const defaultWalletAnalysisCacheTtlMs = 5 * 60 * 1000;
const walletAnalysisCachePrefix = "qorvi:wallet-analysis:v1:";
const walletAnalysisCache = new Map<string, CacheEntry<unknown>>();
const walletAnalysisInflight = new Map<string, Promise<unknown>>();

export function buildWalletAnalysisCacheKey({
  address,
  days,
  provider = resolveWalletProviderCacheSegment(),
}: {
  address: string;
  days: number;
  provider?: string;
}): string {
  return `${provider}:${address.trim().toLowerCase()}:${days}`;
}

export async function getOrSetWalletAnalysisCache<T>({
  key,
  load,
  now = Date.now(),
  ttlMs = resolveWalletAnalysisCacheTtlMs(),
}: {
  key: string;
  load: () => Promise<T>;
  now?: number;
  ttlMs?: number;
}): Promise<T> {
  const cached = await getWalletAnalysisCache<T>(key, now);
  if (cached) {
    return cached;
  }

  const inflight = walletAnalysisInflight.get(key);
  if (inflight) {
    return inflight as Promise<T>;
  }

  const promise = load()
    .then(async (value) => {
      await setWalletAnalysisCache(key, value, ttlMs);
      return value;
    })
    .finally(() => {
      walletAnalysisInflight.delete(key);
    });

  walletAnalysisInflight.set(key, promise);
  return promise;
}

export async function getWalletAnalysisCache<T>(
  key: string,
  now = Date.now(),
): Promise<T | null> {
  const cached = walletAnalysisCache.get(key);
  if (cached && cached.expiresAt > now) {
    return cached.value as T;
  }

  const persisted = await getWalletCopilotStorage().getJson<CacheEntry<T>>(
    `${walletAnalysisCachePrefix}${key}`,
  );
  if (!persisted || persisted.expiresAt <= now) {
    return null;
  }

  walletAnalysisCache.set(key, persisted);
  return persisted.value;
}

export async function setWalletAnalysisCache<T>(
  key: string,
  value: T,
  ttlMs = resolveWalletAnalysisCacheTtlMs(),
): Promise<void> {
  const entry: CacheEntry<T> = {
    value,
    expiresAt: Date.now() + ttlMs,
  };
  walletAnalysisCache.set(key, entry);
  await getWalletCopilotStorage().setJson(
    `${walletAnalysisCachePrefix}${key}`,
    entry,
    { ttlSeconds: Math.ceil(ttlMs / 1000) },
  );
}

export function clearWalletAnalysisCache(): void {
  walletAnalysisCache.clear();
  walletAnalysisInflight.clear();
}

export function resolveWalletAnalysisCacheTtlMs(): number {
  const seconds = Number(
    process.env.QORVI_WALLET_ANALYSIS_CACHE_SECONDS ?? 300,
  );
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return defaultWalletAnalysisCacheTtlMs;
  }
  return seconds * 1000;
}

function resolveWalletProviderCacheSegment(): string {
  const provider = process.env.QORVI_WALLET_PROVIDER?.trim().toLowerCase();
  if (provider === "etherscan" || provider === "alchemy") {
    return provider;
  }
  return "auto";
}
