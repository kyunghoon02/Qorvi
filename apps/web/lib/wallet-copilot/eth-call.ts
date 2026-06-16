import { WalletProviderError } from "./errors";
import { throttleEtherscanRequest } from "./etherscan-throttle";
import { fetchWithTimeout } from "./http";
import {
  isRateLimitMessage,
  resolveEthereumRpcMaxRetries,
  throttleEthereumRpcRequest,
  waitForEthereumRpcRetry,
} from "./rpc-throttle";

const etherscanBaseUrl = "https://api.etherscan.io/v2/api";

export async function ethCall({
  to,
  data,
}: {
  to: string;
  data: string;
}): Promise<string> {
  const rpcUrl = getEthereumRpcUrl();
  if (rpcUrl) {
    return ethCallViaRpc({ rpcUrl, to, data });
  }
  return ethCallViaEtherscan({ to, data });
}

async function ethCallViaRpc({
  rpcUrl,
  to,
  data,
  attempt = 0,
}: {
  rpcUrl: string;
  to: string;
  data: string;
  attempt?: number;
}): Promise<string> {
  await throttleEthereumRpcRequest();
  const response = await fetchWithTimeout(rpcUrl, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      jsonrpc: "2.0",
      id: 1,
      method: "eth_call",
      params: [{ to, data }, "latest"],
    }),
    next: { revalidate: 30 },
  });
  if (!response.ok) {
    if (response.status === 429 && attempt < resolveEthereumRpcMaxRetries()) {
      await waitForEthereumRpcRetry(attempt);
      return ethCallViaRpc({ rpcUrl, to, data, attempt: attempt + 1 });
    }
    throw new WalletProviderError({
      code: response.status === 429 ? "rate_limited" : "provider_unavailable",
      message: `Ethereum RPC HTTP ${response.status}`,
      status: response.status === 429 ? 429 : 503,
    });
  }
  const payload = (await response.json()) as {
    result?: string;
    error?: { message?: string };
  };
  if (!payload.result) {
    if (
      isRateLimitMessage(payload.error?.message) &&
      attempt < resolveEthereumRpcMaxRetries()
    ) {
      await waitForEthereumRpcRetry(attempt);
      return ethCallViaRpc({ rpcUrl, to, data, attempt: attempt + 1 });
    }
    throw new WalletProviderError({
      code: "provider_error",
      message: payload.error?.message ?? "Ethereum RPC eth_call failed.",
    });
  }
  return payload.result;
}

async function ethCallViaEtherscan({
  to,
  data,
  attempt = 0,
}: {
  to: string;
  data: string;
  attempt?: number;
}): Promise<string> {
  const apiKey = process.env.ETHERSCAN_API_KEY?.trim();
  if (!apiKey) {
    throw new WalletProviderError({
      code: "missing_api_key",
      message: "ETHERSCAN_API_KEY is required for contract position reads.",
    });
  }
  await throttleEtherscanRequest();
  const params = new URLSearchParams({
    chainid: "1",
    module: "proxy",
    action: "eth_call",
    to,
    data,
    tag: "latest",
    apikey: apiKey,
  });
  const response = await fetchWithTimeout(
    `${etherscanBaseUrl}?${params.toString()}`,
    {
      next: { revalidate: 30 },
    },
  );
  if (!response.ok) {
    if (response.status === 429 && attempt === 0) {
      await waitForRetryWindow();
      return ethCallViaEtherscan({ to, data, attempt: attempt + 1 });
    }
    throw new WalletProviderError({
      code: response.status === 429 ? "rate_limited" : "provider_unavailable",
      message: `Etherscan proxy HTTP ${response.status}`,
      status: response.status === 429 ? 429 : 503,
    });
  }
  const payload = (await response.json()) as {
    result?: string;
    error?: { message?: string };
  };
  if (!payload.result) {
    const message =
      payload.error?.message ?? "Etherscan proxy eth_call failed.";
    if (message.toLowerCase().includes("rate limit") && attempt === 0) {
      await waitForRetryWindow();
      return ethCallViaEtherscan({ to, data, attempt: attempt + 1 });
    }
    throw new WalletProviderError({ code: "provider_error", message });
  }
  return payload.result;
}

export function encodeAddress(address: string): string {
  return address.toLowerCase().replace(/^0x/, "").padStart(64, "0");
}

export function encodeUint(value: bigint | number | string): string {
  return BigInt(value).toString(16).padStart(64, "0");
}

export function encodeInt(value: bigint | number | string): string {
  const bigintValue = BigInt(value);
  const encoded = bigintValue >= 0n ? bigintValue : (1n << 256n) + bigintValue;
  return encoded.toString(16).padStart(64, "0");
}

export function decodeWords(data: string): string[] {
  const clean = data.replace(/^0x/, "");
  const words: string[] = [];
  for (let index = 0; index < clean.length; index += 64) {
    words.push(clean.slice(index, index + 64).padStart(64, "0"));
  }
  return words;
}

export function wordToAddress(word: string): string {
  return `0x${word.slice(-40)}`.toLowerCase();
}

export function wordToBigInt(word: string): bigint {
  return BigInt(`0x${word || "0"}`);
}

export function wordToSignedNumber(word: string): number {
  const value = wordToBigInt(word);
  const signBit = 1n << 255n;
  const signed = value >= signBit ? value - (1n << 256n) : value;
  return Number(signed);
}

export function decodeStringResult(data: string): string | null {
  const words = decodeWords(data);
  if (words.length === 1) {
    return hexToAscii(words[0] ?? "");
  }
  const length = Number(wordToBigInt(words[1] ?? "0"));
  if (!Number.isFinite(length) || length <= 0) {
    return null;
  }
  const hex = data.replace(/^0x/, "").slice(128, 128 + length * 2);
  return hexToAscii(hex);
}

export function formatUnits(raw: bigint | string, decimals: number): string {
  const value = typeof raw === "bigint" ? raw : BigInt(raw || "0");
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
    .slice(0, 8);
  return `${whole.toString()}.${fractionText}`;
}

export function addDecimalStrings(left: string, right: string): string {
  const result =
    Number.parseFloat(left || "0") + Number.parseFloat(right || "0");
  if (!Number.isFinite(result)) {
    return left;
  }
  return result.toLocaleString("en-US", {
    maximumFractionDigits: 8,
    useGrouping: false,
  });
}

function getEthereumRpcUrl(): string | null {
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

function hexToAscii(hex: string): string | null {
  const bytes: number[] = [];
  for (let index = 0; index < hex.length; index += 2) {
    const byte = Number.parseInt(hex.slice(index, index + 2), 16);
    if (byte > 0) {
      bytes.push(byte);
    }
  }
  if (bytes.length === 0) {
    return null;
  }
  return new TextDecoder().decode(Uint8Array.from(bytes)).trim() || null;
}

async function waitForRetryWindow(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 1250));
}
