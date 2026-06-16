import { WalletProviderError } from "./errors";
import { fetchWithTimeout } from "./http";
import { getKnownLabel, normalizeAddress } from "./labels";
import {
  isEnabledEnv,
  readPositiveNumberEnv,
  resolveEthereumRpcUrl,
} from "./provider-utils";
import {
  isRateLimitMessage,
  resolveEthereumRpcMaxRetries,
  throttleEthereumRpcRequest,
  waitForEthereumRpcRetry,
} from "./rpc-throttle";
import type {
  ProviderLog,
  ProviderReceipt,
  ProviderTransaction,
} from "./types";

type RpcReceipt = {
  transactionHash: string;
  blockNumber?: string;
  status?: string;
  logs?: Array<{
    address: string;
    topics?: string[];
    data?: string;
    logIndex?: string;
    blockNumber?: string;
    transactionHash?: string;
  }>;
};

export async function readTransactionReceipts(
  transactions: ProviderTransaction[],
): Promise<{
  receipts: ProviderReceipt[];
  coverage: "complete" | "partial" | "unavailable";
}> {
  if (
    isEnabledEnv("QORVI_INSTANT_ANALYSIS") ||
    isEnabledEnv("QORVI_SKIP_RECEIPTS")
  ) {
    return { receipts: [], coverage: "unavailable" };
  }

  const rpcUrl = resolveEthereumRpcUrl();
  if (!rpcUrl) {
    return { receipts: [], coverage: "unavailable" };
  }

  const candidates = transactions.filter(
    (tx) => Boolean(getKnownLabel(tx.to)) || (tx.input && tx.input !== "0x"),
  );
  const maxReceipts = readPositiveNumberEnv(
    "QORVI_RECEIPT_MAX_TRANSACTIONS",
    80,
  );
  const selected = candidates.slice(0, maxReceipts);
  const receipts: ProviderReceipt[] = [];
  let failed = false;
  for (const tx of selected) {
    try {
      const receipt = await fetchReceipt(rpcUrl, tx.hash);
      if (receipt) {
        receipts.push(receipt);
      }
    } catch {
      failed = true;
    }
  }
  const truncated = candidates.length > selected.length;

  return {
    receipts,
    coverage: failed || truncated ? "partial" : "complete",
  };
}

async function fetchReceipt(
  rpcUrl: string,
  txHash: string,
  attempt = 0,
): Promise<ProviderReceipt | null> {
  await throttleEthereumRpcRequest();
  const response = await fetchWithTimeout(rpcUrl, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      jsonrpc: "2.0",
      id: 1,
      method: "eth_getTransactionReceipt",
      params: [txHash],
    }),
    cache: "no-store",
  });
  if (!response.ok) {
    if (response.status === 429 && attempt < resolveEthereumRpcMaxRetries()) {
      await waitForEthereumRpcRetry(attempt);
      return fetchReceipt(rpcUrl, txHash, attempt + 1);
    }
    throw new WalletProviderError({
      code: response.status === 429 ? "rate_limited" : "provider_unavailable",
      message: `Ethereum RPC receipt HTTP ${response.status}`,
      status: response.status === 429 ? 429 : 503,
    });
  }
  const payload = (await response.json()) as {
    result?: RpcReceipt | null;
    error?: { message?: string };
  };
  if (payload.error) {
    if (
      isRateLimitMessage(payload.error.message) &&
      attempt < resolveEthereumRpcMaxRetries()
    ) {
      await waitForEthereumRpcRetry(attempt);
      return fetchReceipt(rpcUrl, txHash, attempt + 1);
    }
    throw new WalletProviderError({
      code: "provider_error",
      message: payload.error.message ?? "Ethereum RPC receipt read failed.",
    });
  }
  if (!payload.result) {
    return null;
  }
  return toProviderReceipt(payload.result);
}

function toProviderReceipt(receipt: RpcReceipt): ProviderReceipt {
  const status =
    receipt.status === "0x1"
      ? "success"
      : receipt.status === "0x0"
        ? "failed"
        : "unknown";
  return {
    transaction_hash: receipt.transactionHash,
    block_number: parseHexNumber(receipt.blockNumber),
    status,
    source: "ethereum_rpc",
    logs: (receipt.logs ?? []).map(toProviderLog),
  };
}

function toProviderLog(
  log: NonNullable<RpcReceipt["logs"]>[number],
): ProviderLog {
  return {
    address: normalizeAddress(log.address),
    topics: (log.topics ?? []).map((topic) => topic.toLowerCase()),
    data: log.data ?? "0x",
    log_index: parseHexNumber(log.logIndex) ?? 0,
    transaction_hash: log.transactionHash ?? "",
    block_number: parseHexNumber(log.blockNumber),
  };
}

function parseHexNumber(value?: string): number | null {
  if (!value) {
    return null;
  }
  const parsed = Number.parseInt(value, value.startsWith("0x") ? 16 : 10);
  return Number.isFinite(parsed) ? parsed : null;
}
