import { fetchWithTimeout } from "./http";
import {
  getHistoricalPricePoint,
  persistHistoricalPricePoints,
} from "./index-repository";
import { normalizeAddress } from "./labels";
import { isEnabledEnv, readPositiveNumberEnv } from "./provider-utils";
import type {
  HistoricalPricePoint,
  ProviderDataset,
  ProviderERC20Transfer,
  ProviderTransaction,
} from "./types";

const coinGeckoBaseUrl = "https://api.coingecko.com/api/v3";
const stablecoinSymbols = new Set([
  "USDC",
  "USDT",
  "DAI",
  "USDS",
  "USDE",
  "FRAX",
]);

export async function attachHistoricalPrices(
  dataset: ProviderDataset,
): Promise<ProviderDataset> {
  if (
    isEnabledEnv("QORVI_INSTANT_ANALYSIS") ||
    isEnabledEnv("QORVI_SKIP_HISTORICAL_PRICES")
  ) {
    return {
      ...dataset,
      historicalPrices: [],
      historicalPriceCoverage: "unavailable",
    };
  }

  const events = uniquePriceEvents(
    dataset.transactions,
    dataset.erc20Transfers,
  );
  const maxEvents = readPositiveNumberEnv(
    "QORVI_HISTORICAL_PRICE_MAX_EVENTS",
    40,
  );
  const selected = events.slice(0, maxEvents);
  const points: HistoricalPricePoint[] = [];
  for (const event of selected) {
    points.push(await getOrFetchHistoricalPrice(event));
  }
  await persistHistoricalPricePoints(points);
  const truncated = events.length > selected.length;
  return {
    ...dataset,
    historicalPrices: points,
    historicalPriceCoverage:
      points.length === 0
        ? "unavailable"
        : truncated || points.some((point) => !point.available)
          ? "partial"
          : "complete",
  };
}

export function historicalPriceBucket(timestamp: string): string {
  const date = new Date(timestamp);
  date.setUTCMinutes(0, 0, 0);
  return date.toISOString();
}

async function getOrFetchHistoricalPrice(event: {
  token_address: string;
  token_symbol: string;
  timestamp: string;
}): Promise<HistoricalPricePoint> {
  const assetAddress = normalizeAddress(event.token_address);
  const timestamp = historicalPriceBucket(event.timestamp);
  const cached = await getHistoricalPricePoint(assetAddress, timestamp);
  if (cached) {
    return cached;
  }
  if (stablecoinSymbols.has(event.token_symbol.toUpperCase())) {
    return {
      asset_address: assetAddress,
      timestamp,
      provider: "stablecoin_parity",
      value_usd: 1,
      available: true,
    };
  }
  const from = Math.floor(new Date(timestamp).getTime() / 1000);
  const to = from + 3600;
  try {
    const endpoint =
      assetAddress === "eth"
        ? `${coinGeckoBaseUrl}/coins/ethereum/market_chart/range?vs_currency=usd&from=${from}&to=${to}`
        : `${coinGeckoBaseUrl}/coins/ethereum/contract/${assetAddress}/market_chart/range?vs_currency=usd&from=${from}&to=${to}`;
    const response = await fetchWithTimeout(endpoint, {
      next: { revalidate: 86400 },
    });
    if (response.ok) {
      const payload = (await response.json()) as {
        prices?: Array<[number, number]>;
      };
      const price = payload.prices?.[0]?.[1];
      if (typeof price === "number" && Number.isFinite(price)) {
        return {
          asset_address: assetAddress,
          timestamp,
          provider: "coingecko",
          value_usd: price,
          available: true,
        };
      }
    }
  } catch {
    // Missing historical valuation is persisted as unavailable coverage.
  }
  return {
    asset_address: assetAddress,
    timestamp,
    provider: "coingecko",
    value_usd: null,
    available: false,
  };
}

function uniquePriceEvents(
  transactions: ProviderTransaction[],
  transfers: ProviderERC20Transfer[],
) {
  const events = new Map<
    string,
    {
      token_address: string;
      token_symbol: string;
      timestamp: string;
    }
  >();
  for (const transaction of transactions) {
    if (Number.parseFloat(transaction.value_eth) <= 0) {
      continue;
    }
    const timestamp = historicalPriceBucket(transaction.timestamp);
    events.set(`eth:${timestamp}`, {
      token_address: "eth",
      token_symbol: "ETH",
      timestamp,
    });
  }
  for (const transfer of transfers) {
    const timestamp = historicalPriceBucket(transfer.timestamp);
    const key = `${normalizeAddress(transfer.token_address)}:${timestamp}`;
    if (!events.has(key)) {
      events.set(key, {
        token_address: transfer.token_address,
        token_symbol: transfer.token_symbol,
        timestamp,
      });
    }
  }
  return [...events.values()];
}
