import type { WalletDetailRequest } from "../../../../lib/api-boundary";

function safeDecodeURIComponent(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
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
