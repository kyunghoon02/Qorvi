import { fetchWithTimeout } from "./http";
import { normalizeAddress } from "./labels";
import type {
  DefiPositionSummary,
  ProviderERC20Transfer,
  ProviderPriceMap,
  ProviderTransaction,
} from "./types";

const coinGeckoBaseUrl = "https://api.coingecko.com/api/v3";

export const maxBalanceTokens = 12;

export type TokenBalanceTarget = {
  token_symbol: string;
  token_address: string;
  decimals: number;
};

export function readPositiveNumberEnv(
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

export function isEnabledEnv(name: string): boolean {
  const raw = process.env[name]?.trim().toLowerCase();
  return raw === "1" || raw === "true" || raw === "yes";
}

export function emptyDefiPositions(explanation: string): DefiPositionSummary {
  return {
    current_positions_status: "unavailable",
    lp_positions_status: "unavailable",
    total_supplied_usd: null,
    total_borrowed_usd: null,
    total_lp_value_usd: null,
    aave_positions: [],
    uniswap_v3_positions: [],
    curve_positions: [],
    explanation,
    detected_actions: [],
    sources: [],
    errors: [],
  };
}

export function buildTokenBalanceTargets(
  transfers: ProviderERC20Transfer[],
): TokenBalanceTarget[] {
  const counts = new Map<string, TokenBalanceTarget & { count: number }>();
  for (const transfer of transfers) {
    const current =
      counts.get(transfer.token_address) ??
      ({
        token_symbol: transfer.token_symbol,
        token_address: transfer.token_address,
        decimals: transfer.decimals,
        count: 0,
      } satisfies TokenBalanceTarget & { count: number });
    current.count += 1;
    counts.set(transfer.token_address, current);
  }
  return [...counts.values()]
    .sort((left, right) => right.count - left.count)
    .slice(0, maxBalanceTokens)
    .map(({ token_symbol, token_address, decimals }) => ({
      token_symbol,
      token_address,
      decimals,
    }));
}

export async function fetchUsdPrices({
  tokenAddresses,
  includeEth,
}: {
  tokenAddresses: string[];
  includeEth: boolean;
}): Promise<ProviderPriceMap> {
  const prices: ProviderPriceMap = {};
  try {
    const uniqueAddresses = [...new Set(tokenAddresses.map(normalizeAddress))];
    for (const address of uniqueAddresses) {
      const params = new URLSearchParams({
        contract_addresses: address,
        vs_currencies: "usd",
      });
      const response = await fetchWithTimeout(
        `${coinGeckoBaseUrl}/simple/token_price/ethereum?${params.toString()}`,
        { next: { revalidate: 60 } },
      );
      if (response.ok) {
        const payload = (await response.json()) as Record<
          string,
          { usd?: number }
        >;
        for (const [address, value] of Object.entries(payload)) {
          if (typeof value.usd === "number") {
            prices[normalizeAddress(address)] = value.usd;
          }
        }
      }
    }

    if (includeEth) {
      const response = await fetchWithTimeout(
        `${coinGeckoBaseUrl}/simple/price?ids=ethereum&vs_currencies=usd`,
        { next: { revalidate: 60 } },
      );
      if (response.ok) {
        const payload = (await response.json()) as {
          ethereum?: { usd?: number };
        };
        if (typeof payload.ethereum?.usd === "number") {
          prices.eth = payload.ethereum.usd;
        }
      }
    }
  } catch {
    return prices;
  }
  return prices;
}

export function dedupeProviderTransactions(
  transactions: ProviderTransaction[],
): ProviderTransaction[] {
  const seen = new Set<string>();
  const deduped: ProviderTransaction[] = [];
  for (const tx of transactions) {
    if (seen.has(tx.hash)) {
      continue;
    }
    seen.add(tx.hash);
    deduped.push(tx);
  }
  return deduped.sort(
    (left, right) => Date.parse(left.timestamp) - Date.parse(right.timestamp),
  );
}

export function formatProviderUnits(
  raw: bigint | string,
  decimals: number,
): string {
  const value =
    typeof raw === "bigint"
      ? raw
      : raw.startsWith("0x")
        ? BigInt(raw)
        : BigInt(raw.replace(/[^0-9]/g, "") || "0");
  const divisor = 10n ** BigInt(Math.max(decimals, 0));
  const whole = value / divisor;
  const fraction = value % divisor;
  if (fraction === 0n || decimals === 0) {
    return whole.toString();
  }
  const fractionText = fraction
    .toString()
    .padStart(decimals, "0")
    .replace(/0+$/, "")
    .slice(0, 6);
  return `${whole.toString()}.${fractionText}`;
}

export function resolveEthereumRpcUrl(): string | null {
  const direct =
    process.env.ETHEREUM_RPC_URL?.trim() ||
    process.env.ALCHEMY_ETHEREUM_RPC_URL?.trim();
  if (direct) {
    return direct;
  }
  const alchemyKey = process.env.ALCHEMY_API_KEY?.trim();
  const alchemyBaseUrl = process.env.ALCHEMY_BASE_URL?.trim();
  if (alchemyBaseUrl) {
    if (alchemyBaseUrl.includes("/v2/") || !alchemyKey) {
      return alchemyBaseUrl;
    }
    return `${alchemyBaseUrl.replace(/\/$/, "")}/v2/${alchemyKey}`;
  }
  if (alchemyKey) {
    return `https://eth-mainnet.g.alchemy.com/v2/${alchemyKey}`;
  }
  return null;
}

export const resolveAlchemyRpcUrl = resolveEthereumRpcUrl;
