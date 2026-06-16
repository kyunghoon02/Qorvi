import { readLiveDefiPositions } from "./defi-position-readers";
import { WalletProviderError } from "./errors";
import { fetchWithTimeout } from "./http";
import { normalizeAddress } from "./labels";
import {
  type TokenBalanceTarget,
  buildTokenBalanceTargets,
  dedupeProviderTransactions,
  emptyDefiPositions,
  fetchUsdPrices,
  formatProviderUnits,
  isEnabledEnv,
  maxBalanceTokens,
  readPositiveNumberEnv,
  resolveAlchemyRpcUrl,
} from "./provider-utils";
import { readTransactionReceipts } from "./receipt-reader";
import type {
  ProviderDataset,
  ProviderERC20Transfer,
  ProviderTokenBalance,
  ProviderTransaction,
} from "./types";

const alchemyTransferPageSize = "0x3e8";
const alchemyMaxPages = readPositiveNumberEnv("QORVI_ALCHEMY_MAX_PAGES", 8);

type AlchemyRpcEnvelope<T> = {
  jsonrpc: "2.0";
  id: number;
  result?: T;
  error?: {
    code?: number;
    message?: string;
  };
};

type AlchemyAssetTransfer = {
  blockNum: string;
  hash: string;
  from: string | null;
  to: string | null;
  value: number | string | null;
  asset: string | null;
  category: "external" | "internal" | "erc20" | "erc721" | "erc1155";
  metadata?: {
    blockTimestamp?: string;
  };
  rawContract?: {
    value?: string | null;
    address?: string | null;
    decimal?: string | null;
  };
};

type AlchemyAssetTransfersResult = {
  transfers: AlchemyAssetTransfer[];
  pageKey?: string;
};

type AlchemyTokenBalancesResult = {
  address: string;
  tokenBalances: Array<{
    contractAddress: string;
    tokenBalance: string | null;
  }>;
};

export async function getAlchemyWalletDataset(
  address: string,
  days: number,
): Promise<ProviderDataset> {
  const rpcUrl = resolveAlchemyRpcUrl();
  if (!rpcUrl) {
    throw new WalletProviderError({
      code: "missing_api_key",
      message:
        "ALCHEMY_API_KEY, ALCHEMY_BASE_URL, or ALCHEMY_ETHEREUM_RPC_URL is required for Alchemy live fallback.",
    });
  }

  const cutoff = Date.now() - days * 24 * 60 * 60 * 1000;
  const [outboundTransfers, inboundTransfers] = await Promise.all([
    fetchAlchemyAssetTransfers({
      rpcUrl,
      address,
      direction: "from",
      cutoff,
    }),
    fetchAlchemyAssetTransfers({
      rpcUrl,
      address,
      direction: "to",
      cutoff,
    }),
  ]);
  const allTransfers = dedupeAlchemyTransfers([
    ...outboundTransfers,
    ...inboundTransfers,
  ]).filter((transfer) => alchemyTransferTimestampMs(transfer) >= cutoff);

  const normalizedTransactions = dedupeProviderTransactions(
    allTransfers
      .filter((transfer) => transfer.category === "external")
      .map(toAlchemyProviderTransaction),
  );
  const normalizedTransfers = allTransfers
    .filter((transfer) => transfer.category === "erc20")
    .map(toAlchemyProviderTransfer)
    .filter(Boolean) as ProviderERC20Transfer[];
  const receiptResult = await readTransactionReceipts(normalizedTransactions);
  if (isEnabledEnv("QORVI_INSTANT_ANALYSIS")) {
    return {
      provider: "alchemy",
      data_mode: "live",
      data_notice:
        "Live Ethereum mainnet transfer activity loaded from Alchemy in instant mode. Current balances, DeFi positions, receipt logs, and historical prices are deferred for speed.",
      transactions: normalizedTransactions,
      erc20Transfers: normalizedTransfers,
      receipts: receiptResult.receipts,
      receiptCoverage: receiptResult.coverage,
      historicalPrices: [],
      historicalPriceCoverage: "unavailable",
      performanceLedgerScope: "selected_window",
      nativeBalanceEth: "0",
      tokenBalances: [],
      tokenPricesUsd: {},
      defiPositions: emptyDefiPositions(
        "Current DeFi position reads are deferred in instant analysis mode.",
      ),
    };
  }

  const tokenBalanceTargets = buildTokenBalanceTargets(normalizedTransfers);
  const nativeBalanceEth = await fetchAlchemyNativeBalance({ rpcUrl, address });
  const tokenBalances = await fetchAlchemyTokenBalances({
    rpcUrl,
    address,
    tokens: tokenBalanceTargets,
  });
  const tokenPricesUsd = await fetchUsdPrices({
    tokenAddresses: tokenBalanceTargets.map((token) => token.token_address),
    includeEth: true,
  });
  const defiPositions = await readLiveDefiPositions(address, {
    curveCandidateAddresses: [
      ...tokenBalanceTargets.map((token) => token.token_address),
      ...normalizedTransfers.map((transfer) => transfer.token_address),
    ],
  });

  return {
    provider: "alchemy",
    data_mode: "live",
    data_notice:
      "Live Ethereum mainnet data loaded from Alchemy Transfers and Token APIs. This fallback is partial for zero-value contract calls because Alchemy transfer history is transfer-centric. Labels and risk classification are heuristic.",
    transactions: normalizedTransactions,
    erc20Transfers: normalizedTransfers,
    receipts: receiptResult.receipts,
    receiptCoverage: receiptResult.coverage,
    historicalPrices: [],
    historicalPriceCoverage: "unavailable",
    performanceLedgerScope: "selected_window",
    nativeBalanceEth,
    tokenBalances,
    tokenPricesUsd,
    defiPositions,
  };
}

async function fetchAlchemyAssetTransfers({
  rpcUrl,
  address,
  direction,
  cutoff,
}: {
  rpcUrl: string;
  address: string;
  direction: "from" | "to";
  cutoff: number;
}): Promise<AlchemyAssetTransfer[]> {
  const transfers: AlchemyAssetTransfer[] = [];
  let pageKey: string | undefined;

  for (let page = 1; page <= alchemyMaxPages; page += 1) {
    const params: Record<string, unknown> = {
      fromBlock: "0x0",
      toBlock: "latest",
      category: ["external", "erc20"],
      withMetadata: true,
      excludeZeroValue: false,
      maxCount: alchemyTransferPageSize,
      order: "desc",
    };
    if (direction === "from") {
      params.fromAddress = address;
    } else {
      params.toAddress = address;
    }
    if (pageKey) {
      params.pageKey = pageKey;
    }

    const result = await fetchAlchemyRpc<AlchemyAssetTransfersResult>(rpcUrl, {
      method: "alchemy_getAssetTransfers",
      params: [params],
    });
    transfers.push(...result.transfers);

    const oldestTimestamp = Math.min(
      ...result.transfers
        .map(alchemyTransferTimestampMs)
        .filter(Number.isFinite),
    );
    if (!result.pageKey || result.transfers.length === 0) {
      break;
    }
    if (Number.isFinite(oldestTimestamp) && oldestTimestamp < cutoff) {
      break;
    }
    pageKey = result.pageKey;
  }

  return transfers;
}

async function fetchAlchemyNativeBalance({
  rpcUrl,
  address,
}: {
  rpcUrl: string;
  address: string;
}): Promise<string> {
  const rawBalance = await fetchAlchemyRpc<string>(rpcUrl, {
    method: "eth_getBalance",
    params: [address, "latest"],
  });
  return formatProviderUnits(BigInt(rawBalance), 18);
}

async function fetchAlchemyTokenBalances({
  rpcUrl,
  address,
  tokens,
}: {
  rpcUrl: string;
  address: string;
  tokens: TokenBalanceTarget[];
}): Promise<ProviderTokenBalance[]> {
  const selectedTokens = tokens.slice(0, maxBalanceTokens);
  if (selectedTokens.length === 0) {
    return [];
  }

  const result = await fetchAlchemyRpc<AlchemyTokenBalancesResult>(rpcUrl, {
    method: "alchemy_getTokenBalances",
    params: [address, selectedTokens.map((token) => token.token_address)],
  });
  const tokenByAddress = new Map(
    selectedTokens.map((token) => [
      normalizeAddress(token.token_address),
      token,
    ]),
  );

  return result.tokenBalances.flatMap((balance) => {
    const token = tokenByAddress.get(normalizeAddress(balance.contractAddress));
    if (!token || !balance.tokenBalance) {
      return [];
    }
    return [
      {
        token_symbol: token.token_symbol,
        token_address: token.token_address,
        decimals: token.decimals,
        balance: formatProviderUnits(
          BigInt(balance.tokenBalance),
          token.decimals,
        ),
      },
    ];
  });
}

async function fetchAlchemyRpc<T>(
  rpcUrl: string,
  {
    method,
    params,
  }: {
    method: string;
    params: unknown[];
  },
): Promise<T> {
  const response = await fetchWithTimeout(rpcUrl, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      jsonrpc: "2.0",
      id: 1,
      method,
      params,
    }),
    next: { revalidate: 30 },
  });

  if (!response.ok) {
    throw new WalletProviderError({
      code: response.status === 429 ? "rate_limited" : "provider_unavailable",
      message: `Alchemy HTTP ${response.status}`,
      status: response.status === 429 ? 429 : 503,
    });
  }

  const payload = (await response.json()) as AlchemyRpcEnvelope<T>;
  if (payload.error || payload.result === undefined) {
    throw new WalletProviderError({
      code: "provider_error",
      message: payload.error?.message ?? `Alchemy ${method} failed.`,
    });
  }
  return payload.result;
}

function toAlchemyProviderTransaction(
  transfer: AlchemyAssetTransfer,
): ProviderTransaction {
  return {
    hash: transfer.hash,
    from: normalizeAddress(transfer.from ?? ""),
    to: normalizeAddress(transfer.to ?? ""),
    value_eth: decimalValueToString(transfer.value),
    timestamp: alchemyTransferTimestamp(transfer),
    input: "0x",
    is_error: false,
    block_number: Number.parseInt(transfer.blockNum, 16),
  };
}

function toAlchemyProviderTransfer(
  transfer: AlchemyAssetTransfer,
): ProviderERC20Transfer | null {
  const tokenAddress = transfer.rawContract?.address;
  if (!tokenAddress) {
    return null;
  }
  const decimals = parseAlchemyDecimal(transfer.rawContract?.decimal);
  return {
    hash: transfer.hash,
    from: normalizeAddress(transfer.from ?? ""),
    to: normalizeAddress(transfer.to ?? ""),
    token_symbol: transfer.asset || "UNKNOWN",
    token_name: transfer.asset || "Unknown token",
    token_address: normalizeAddress(tokenAddress),
    value:
      transfer.rawContract?.value && transfer.rawContract.value !== "0x"
        ? formatProviderUnits(BigInt(transfer.rawContract.value), decimals)
        : decimalValueToString(transfer.value),
    decimals,
    timestamp: alchemyTransferTimestamp(transfer),
    block_number: Number.parseInt(transfer.blockNum, 16),
  };
}

function dedupeAlchemyTransfers(
  transfers: AlchemyAssetTransfer[],
): AlchemyAssetTransfer[] {
  const seen = new Set<string>();
  const deduped: AlchemyAssetTransfer[] = [];
  for (const transfer of transfers) {
    const key = `${transfer.hash}:${transfer.category}:${transfer.from}:${transfer.to}:${transfer.rawContract?.address}:${transfer.rawContract?.value}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    deduped.push(transfer);
  }
  return deduped;
}

function alchemyTransferTimestamp(transfer: AlchemyAssetTransfer): string {
  const timestamp = transfer.metadata?.blockTimestamp;
  if (timestamp && !Number.isNaN(Date.parse(timestamp))) {
    return new Date(timestamp).toISOString();
  }
  return new Date(0).toISOString();
}

function alchemyTransferTimestampMs(transfer: AlchemyAssetTransfer): number {
  return Date.parse(alchemyTransferTimestamp(transfer));
}

function parseAlchemyDecimal(value: string | null | undefined): number {
  if (!value) {
    return 18;
  }
  if (value.startsWith("0x")) {
    return Number.parseInt(value, 16);
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 18;
}

function decimalValueToString(value: number | string | null): string {
  if (value === null) {
    return "0";
  }
  if (typeof value === "number") {
    return Number.isFinite(value) ? value.toString() : "0";
  }
  return value;
}
