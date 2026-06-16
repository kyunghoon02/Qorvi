import { readLiveDefiPositions } from "./defi-position-readers";
import { WalletProviderError } from "./errors";
import { throttleEtherscanRequest } from "./etherscan-throttle";
import { fetchWithTimeout } from "./http";
import { normalizeAddress } from "./labels";
import {
  type TokenBalanceTarget,
  buildTokenBalanceTargets,
  emptyDefiPositions,
  fetchUsdPrices,
  formatProviderUnits,
  isEnabledEnv,
  maxBalanceTokens,
  readPositiveNumberEnv,
} from "./provider-utils";
import { readTransactionReceipts } from "./receipt-reader";
import type {
  ProviderDataset,
  ProviderERC20Transfer,
  ProviderTokenBalance,
  ProviderTransaction,
} from "./types";

const etherscanBaseUrl = "https://api.etherscan.io/v2/api";
const etherscanPageSize = 1000;
const etherscanMaxPages = readPositiveNumberEnv(
  "QORVI_ETHERSCAN_MAX_PAGES",
  10,
);

type EtherscanEnvelope<T> = {
  status: string;
  message: string;
  result: T;
};

type EtherscanTransaction = {
  hash: string;
  from: string;
  to: string;
  value: string;
  timeStamp: string;
  input: string;
  functionName?: string;
  isError?: string;
  blockNumber?: string;
};

type EtherscanTokenTransfer = {
  hash: string;
  from: string;
  to: string;
  contractAddress: string;
  tokenSymbol: string;
  tokenName: string;
  tokenDecimal: string;
  value: string;
  timeStamp: string;
  blockNumber?: string;
};

export type EtherscanActivityWindow = Pick<
  ProviderDataset,
  | "provider"
  | "data_mode"
  | "data_notice"
  | "transactions"
  | "erc20Transfers"
  | "receipts"
  | "receiptCoverage"
>;

export async function getEtherscanWalletDataset(
  address: string,
  days: number,
): Promise<ProviderDataset> {
  const apiKey = process.env.ETHERSCAN_API_KEY?.trim();
  if (!apiKey) {
    throw new WalletProviderError({
      code: "missing_api_key",
      message:
        "ETHERSCAN_API_KEY is required because Qorvi is configured for live on-chain data only.",
    });
  }

  const { startBlock, endBlock } = await resolveEtherscanBlockRange(
    days,
    apiKey,
  );
  const activity = await getEtherscanActivityWindow(
    address,
    Number(startBlock),
    Number(endBlock),
  );
  const cutoff = Date.now() - days * 24 * 60 * 60 * 1000;
  const normalizedTransactions = activity.transactions.filter(
    (tx) => Date.parse(tx.timestamp) >= cutoff,
  );
  const normalizedTransfers = activity.erc20Transfers.filter(
    (transfer) => Date.parse(transfer.timestamp) >= cutoff,
  );
  if (isEnabledEnv("QORVI_INSTANT_ANALYSIS")) {
    return {
      provider: "etherscan",
      data_mode: "live",
      data_notice:
        "Live Ethereum mainnet activity loaded from Etherscan V2 in instant mode. Current balances, DeFi positions, receipt logs, and historical prices are deferred for speed.",
      transactions: normalizedTransactions,
      erc20Transfers: normalizedTransfers,
      receipts: activity.receipts,
      receiptCoverage: activity.receiptCoverage,
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
  const nativeBalanceEth = await fetchEtherscanNativeBalance(address, apiKey);
  const tokenBalances = await fetchEtherscanTokenBalances({
    address,
    apiKey,
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
    provider: "etherscan",
    data_mode: "live",
    data_notice:
      "Live Ethereum mainnet data loaded from Etherscan V2. Labels and risk classification are heuristic.",
    transactions: normalizedTransactions,
    erc20Transfers: normalizedTransfers,
    receipts: activity.receipts,
    receiptCoverage: activity.receiptCoverage,
    historicalPrices: [],
    historicalPriceCoverage: "unavailable",
    performanceLedgerScope: "selected_window",
    nativeBalanceEth,
    tokenBalances,
    tokenPricesUsd,
    defiPositions,
  };
}

export async function getEtherscanActivityWindow(
  address: string,
  startBlock: number,
  endBlock: number,
): Promise<EtherscanActivityWindow> {
  const apiKey = process.env.ETHERSCAN_API_KEY?.trim();
  if (!apiKey) {
    throw new WalletProviderError({
      code: "missing_api_key",
      message: "ETHERSCAN_API_KEY is required for lifetime wallet indexing.",
    });
  }
  const [transactions, erc20Transfers] = await Promise.all([
    fetchPaginatedEtherscanAccountAction<EtherscanTransaction[]>(
      "txlist",
      address,
      apiKey,
      String(startBlock),
      String(endBlock),
    ),
    fetchPaginatedEtherscanAccountAction<EtherscanTokenTransfer[]>(
      "tokentx",
      address,
      apiKey,
      String(startBlock),
      String(endBlock),
    ),
  ]);
  const normalizedTransactions = transactions.map(toProviderTransaction);
  const receiptResult = await readTransactionReceipts(normalizedTransactions);
  return {
    provider: "etherscan",
    data_mode: "live",
    data_notice:
      "Live Ethereum mainnet activity window loaded from Etherscan V2 with RPC receipt/log enrichment when configured.",
    transactions: normalizedTransactions,
    erc20Transfers: erc20Transfers.map(toProviderTransfer),
    receipts: receiptResult.receipts,
    receiptCoverage: receiptResult.coverage,
  };
}

export async function resolveEtherscanLatestBlock(): Promise<number> {
  const apiKey = process.env.ETHERSCAN_API_KEY?.trim();
  if (!apiKey) {
    throw new WalletProviderError({
      code: "missing_api_key",
      message: "ETHERSCAN_API_KEY is required for lifetime wallet indexing.",
    });
  }
  return Number(
    await fetchEtherscanBlockByTimestamp(
      Math.floor(Date.now() / 1000),
      "before",
      apiKey,
    ),
  );
}

export async function findEtherscanFirstActivityBlock(
  address: string,
): Promise<number | null> {
  const apiKey = process.env.ETHERSCAN_API_KEY?.trim();
  if (!apiKey) {
    throw new WalletProviderError({
      code: "missing_api_key",
      message: "ETHERSCAN_API_KEY is required for lifetime wallet indexing.",
    });
  }
  const [transactions, transfers] = await Promise.all([
    fetchEtherscanAccountAction<EtherscanTransaction[]>(
      "txlist",
      address,
      apiKey,
      "0",
      "999999999",
      1,
      1,
    ),
    fetchEtherscanAccountAction<EtherscanTokenTransfer[]>(
      "tokentx",
      address,
      apiKey,
      "0",
      "999999999",
      1,
      1,
    ),
  ]);
  const blocks = [...transactions, ...transfers]
    .map((row) => Number(row.blockNumber))
    .filter(Number.isFinite);
  return blocks.length === 0 ? null : Math.min(...blocks);
}

async function fetchEtherscanAccountAction<T>(
  action: "txlist" | "tokentx",
  address: string,
  apiKey: string,
  startBlock: string,
  endBlock: string,
  page: number,
  offset = etherscanPageSize,
): Promise<T> {
  await throttleEtherscanRequest();
  const params = new URLSearchParams({
    chainid: "1",
    module: "account",
    action,
    address,
    startblock: startBlock,
    endblock: endBlock,
    page: page.toString(),
    offset: offset.toString(),
    sort: "asc",
    apikey: apiKey,
  });
  return fetchEtherscan<T>(params);
}

async function fetchPaginatedEtherscanAccountAction<T extends unknown[]>(
  action: "txlist" | "tokentx",
  address: string,
  apiKey: string,
  startBlock: string,
  endBlock: string,
): Promise<T> {
  const rows: unknown[] = [];
  for (let page = 1; page <= etherscanMaxPages; page += 1) {
    const pageRows = await fetchEtherscanAccountAction<T>(
      action,
      address,
      apiKey,
      startBlock,
      endBlock,
      page,
    );
    rows.push(...pageRows);
    if (pageRows.length < etherscanPageSize) {
      break;
    }
  }

  return rows as T;
}

async function resolveEtherscanBlockRange(
  days: number,
  apiKey: string,
): Promise<{ startBlock: string; endBlock: string }> {
  const nowSeconds = Math.floor(Date.now() / 1000);
  const startSeconds = nowSeconds - days * 24 * 60 * 60;
  const startBlock = await fetchEtherscanBlockByTimestamp(
    startSeconds,
    "after",
    apiKey,
  );
  const endBlock = await fetchEtherscanBlockByTimestamp(
    nowSeconds,
    "before",
    apiKey,
  );

  return { startBlock, endBlock };
}

async function fetchEtherscanBlockByTimestamp(
  timestamp: number,
  closest: "before" | "after",
  apiKey: string,
): Promise<string> {
  await throttleEtherscanRequest();
  const params = new URLSearchParams({
    chainid: "1",
    module: "block",
    action: "getblocknobytime",
    timestamp: timestamp.toString(),
    closest,
    apikey: apiKey,
  });
  return fetchEtherscan<string>(params);
}

async function fetchEtherscanNativeBalance(
  address: string,
  apiKey: string,
): Promise<string> {
  await throttleEtherscanRequest();
  const params = new URLSearchParams({
    chainid: "1",
    module: "account",
    action: "balance",
    address,
    tag: "latest",
    apikey: apiKey,
  });
  const rawBalance = await fetchEtherscan<string>(params);
  return formatProviderUnits(rawBalance, 18);
}

async function fetchEtherscanTokenBalances({
  address,
  apiKey,
  tokens,
}: {
  address: string;
  apiKey: string;
  tokens: TokenBalanceTarget[];
}): Promise<ProviderTokenBalance[]> {
  const balances: ProviderTokenBalance[] = [];
  for (const token of tokens.slice(0, maxBalanceTokens)) {
    await throttleEtherscanRequest();
    const params = new URLSearchParams({
      chainid: "1",
      module: "account",
      action: "tokenbalance",
      contractaddress: token.token_address,
      address,
      tag: "latest",
      apikey: apiKey,
    });
    const rawBalance = await fetchEtherscan<string>(params);
    balances.push({
      token_symbol: token.token_symbol,
      token_address: token.token_address,
      decimals: token.decimals,
      balance: formatProviderUnits(rawBalance, token.decimals),
    });
  }
  return balances;
}

async function fetchEtherscan<T>(
  params: URLSearchParams,
  attempt = 0,
): Promise<T> {
  const response = await fetchWithTimeout(
    `${etherscanBaseUrl}?${params.toString()}`,
    {
      next: { revalidate: 60 },
    },
  );

  if (!response.ok) {
    if (response.status === 429 && attempt === 0) {
      await waitForEtherscanRetryWindow();
      return fetchEtherscan<T>(params, attempt + 1);
    }
    throw new WalletProviderError({
      code: response.status === 429 ? "rate_limited" : "provider_unavailable",
      message: `Etherscan HTTP ${response.status}`,
      status: response.status === 429 ? 429 : 503,
    });
  }

  const payload = (await response.json()) as EtherscanEnvelope<T | string>;
  if (payload.status === "0" && payload.message !== "No transactions found") {
    const error = mapEtherscanError(String(payload.result || payload.message));
    if (error.code === "rate_limited" && attempt === 0) {
      await waitForEtherscanRetryWindow();
      return fetchEtherscan<T>(params, attempt + 1);
    }
    throw error;
  }

  if (typeof payload.result === "string") {
    if (
      params.get("action") === "getblocknobytime" ||
      params.get("action") === "balance" ||
      params.get("action") === "tokenbalance"
    ) {
      return payload.result as T;
    }
    return [] as T;
  }
  return payload.result;
}

async function waitForEtherscanRetryWindow(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 1250));
}

function mapEtherscanError(message: string): WalletProviderError {
  const normalized = message.toLowerCase();
  if (normalized.includes("rate limit")) {
    return new WalletProviderError({
      code: "rate_limited",
      message,
    });
  }
  if (
    normalized.includes("invalid api key") ||
    normalized.includes("missing or unsupported chainid")
  ) {
    return new WalletProviderError({
      code: "invalid_api_key",
      message,
    });
  }
  if (normalized.includes("timeout") || normalized.includes("busy")) {
    return new WalletProviderError({
      code: "provider_unavailable",
      message,
    });
  }
  return new WalletProviderError({
    code: "provider_error",
    message,
  });
}

function toProviderTransaction(tx: EtherscanTransaction): ProviderTransaction {
  const normalized: ProviderTransaction = {
    hash: tx.hash,
    from: normalizeAddress(tx.from),
    to: normalizeAddress(tx.to),
    value_eth: formatProviderUnits(tx.value || "0", 18),
    timestamp: new Date(Number(tx.timeStamp) * 1000).toISOString(),
    input: tx.input || "0x",
    is_error: tx.isError === "1",
  };
  if (tx.blockNumber) {
    normalized.block_number = Number(tx.blockNumber);
  }
  if (tx.functionName) {
    normalized.function_name = tx.functionName;
  }
  return normalized;
}

function toProviderTransfer(
  transfer: EtherscanTokenTransfer,
): ProviderERC20Transfer {
  const normalized: ProviderERC20Transfer = {
    hash: transfer.hash,
    from: normalizeAddress(transfer.from),
    to: normalizeAddress(transfer.to),
    token_symbol: transfer.tokenSymbol || "UNKNOWN",
    token_name: transfer.tokenName || "Unknown token",
    token_address: normalizeAddress(transfer.contractAddress),
    value: formatProviderUnits(
      transfer.value || "0",
      Number(transfer.tokenDecimal || 18),
    ),
    decimals: Number(transfer.tokenDecimal || 18),
    timestamp: new Date(Number(transfer.timeStamp) * 1000).toISOString(),
  };
  if (transfer.blockNumber) {
    normalized.block_number = Number(transfer.blockNumber);
  }
  return normalized;
}
