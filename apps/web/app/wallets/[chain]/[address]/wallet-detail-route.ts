import type { WalletDetailRequest } from "../../../../lib/api-boundary";

export type FlowLensContext = {
  source: "flowlens";
  exchange: string;
  symbol: string;
  tokenAddress: string;
  flowMinute: string;
  direction: string;
  amount: string;
  approxUsd: string;
  signalScore: string;
  backUrl: string;
};

type SearchParamInput = Record<string, string | string[] | undefined>;

function safeDecodeURIComponent(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function readSearchParam(
  searchParams: SearchParamInput | undefined,
  key: string,
): string {
  const rawValue = searchParams?.[key];
  const value = Array.isArray(rawValue) ? rawValue[0] : rawValue;
  return typeof value === "string" ? safeDecodeURIComponent(value).trim() : "";
}

function sanitizeBackUrl(value: string): string {
  if (!value) {
    return "";
  }

  try {
    const parsed = new URL(value);
    if (!["http:", "https:"].includes(parsed.protocol)) {
      return "";
    }
    return parsed.toString();
  } catch {
    return "";
  }
}

export function resolveWalletDetailRequestFromParams(
  chain: string,
  address: string,
): WalletDetailRequest | null {
  if (chain !== "evm" && chain !== "solana") {
    return null;
  }

  const decodedAddress = safeDecodeURIComponent(address).trim();

  if (!decodedAddress) {
    return null;
  }

  return {
    chain,
    address: decodedAddress,
  };
}

export function resolveFlowLensContextFromSearchParams(
  searchParams: SearchParamInput | undefined,
): FlowLensContext | null {
  if (readSearchParam(searchParams, "source").toLowerCase() !== "flowlens") {
    return null;
  }

  return {
    source: "flowlens",
    exchange: readSearchParam(searchParams, "exchange"),
    symbol: readSearchParam(searchParams, "symbol"),
    tokenAddress: readSearchParam(searchParams, "token_address"),
    flowMinute: readSearchParam(searchParams, "flow_minute"),
    direction: readSearchParam(searchParams, "direction"),
    amount: readSearchParam(searchParams, "amount"),
    approxUsd: readSearchParam(searchParams, "approx_usd"),
    signalScore: readSearchParam(searchParams, "signal_score"),
    backUrl: sanitizeBackUrl(readSearchParam(searchParams, "back_url")),
  };
}
