import { getAlchemyWalletDataset } from "./alchemy-provider";
import { isWalletProviderError } from "./errors";
import { getEtherscanWalletDataset } from "./etherscan-provider";
import { logWalletCopilotEvent } from "./observability";
import { resolveAlchemyRpcUrl } from "./provider-utils";
import type { ProviderDataset } from "./types";

export function isValidEvmAddress(address: string): boolean {
  return /^0x[a-fA-F0-9]{40}$/.test(address.trim());
}

export async function getWalletDataset(
  address: string,
  days: number,
): Promise<ProviderDataset> {
  const providerMode = resolveWalletProviderMode();
  const hasEtherscanKey = Boolean(process.env.ETHERSCAN_API_KEY?.trim());
  const hasAlchemyRpc = Boolean(resolveAlchemyRpcUrl());

  if (providerMode === "alchemy") {
    return getAlchemyWalletDataset(address, days);
  }
  if (providerMode === "etherscan") {
    return getEtherscanWalletDataset(address, days);
  }

  if (!hasEtherscanKey && hasAlchemyRpc) {
    return getAlchemyWalletDataset(address, days);
  }

  try {
    return await getEtherscanWalletDataset(address, days);
  } catch (error) {
    if (hasAlchemyRpc && shouldFallbackToAlchemy(error)) {
      logWalletCopilotEvent({
        event: "provider_fallback",
        level: "warn",
        address,
        from_provider: "etherscan",
        to_provider: "alchemy",
        error_code: isWalletProviderError(error) ? error.code : "unknown",
      });
      return getAlchemyWalletDataset(address, days);
    }
    throw error;
  }
}

function shouldFallbackToAlchemy(error: unknown): boolean {
  if (!isWalletProviderError(error)) {
    return false;
  }
  return [
    "missing_api_key",
    "invalid_api_key",
    "rate_limited",
    "provider_unavailable",
    "provider_timeout",
    "provider_error",
    "invalid_response",
  ].includes(error.code);
}

function resolveWalletProviderMode(): "auto" | "etherscan" | "alchemy" {
  const mode = process.env.QORVI_WALLET_PROVIDER?.trim().toLowerCase();
  if (mode === "etherscan" || mode === "alchemy") {
    return mode;
  }
  return "auto";
}
