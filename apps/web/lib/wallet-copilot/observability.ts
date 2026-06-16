import { createHash } from "node:crypto";

type WalletCopilotLogLevel = "info" | "warn" | "error";

type WalletCopilotLogPayload = Record<
  string,
  string | number | boolean | null | undefined
>;

export function logWalletCopilotEvent({
  event,
  level = "info",
  address,
  ...payload
}: {
  event: string;
  level?: WalletCopilotLogLevel;
  address?: string;
} & WalletCopilotLogPayload): void {
  const record = {
    scope: "wallet_copilot",
    event,
    timestamp: new Date().toISOString(),
    address_hash: address ? hashAddress(address) : undefined,
    ...payload,
  };

  const line = JSON.stringify(record);
  if (level === "error") {
    console.error(line);
    return;
  }
  if (level === "warn") {
    console.warn(line);
    return;
  }
  console.info(line);
}

function hashAddress(address: string): string {
  return createHash("sha256")
    .update(address.trim().toLowerCase())
    .digest("hex")
    .slice(0, 16);
}
