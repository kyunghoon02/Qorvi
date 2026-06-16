import { createHash } from "node:crypto";

import { WalletProviderError } from "./errors";
import { logWalletCopilotEvent } from "./observability";
import { getWalletCopilotStorage } from "./storage";
import type { QuotaState } from "./types";

type PublicQuotaKind = "analysis" | "chat";

export async function consumePublicQuota(
  request: Request,
  kind: PublicQuotaKind,
): Promise<QuotaState> {
  const identity = await resolveQuotaIdentity(request);
  const limit = quotaLimit(identity.scope, kind);
  const { day, ttlSeconds, resetAt } = utcDailyWindow();
  const key = `qorvi:quota:v1:${kind}:${identity.scope}:${identity.key}:${day}`;
  const used = await getWalletCopilotStorage().increment(key, { ttlSeconds });
  const state: QuotaState = {
    scope: identity.scope,
    kind,
    limit,
    used,
    remaining: Math.max(0, limit - used),
    reset_at: resetAt,
  };

  if (used > limit) {
    logWalletCopilotEvent({
      event: "quota_rejection",
      level: "warn",
      quota_kind: kind,
      quota_scope: identity.scope,
      used,
      limit,
    });
    throw new WalletProviderError({
      code: "quota_exceeded",
      message: `${kind === "analysis" ? "Wallet analysis" : "Chat"} daily quota exceeded. Try again after ${resetAt}.`,
    });
  }
  return state;
}

export async function consumeProviderBudget(): Promise<void> {
  const limit = readPositiveIntegerEnv("QORVI_PROVIDER_DAILY_BUDGET", 90_000);
  const { day, ttlSeconds, resetAt } = utcDailyWindow();
  const used = await getWalletCopilotStorage().increment(
    `qorvi:provider-budget:v1:etherscan:${day}`,
    { ttlSeconds },
  );
  if (used > limit) {
    logWalletCopilotEvent({
      event: "provider_budget_rejection",
      level: "error",
      provider: "etherscan",
      used,
      limit,
    });
    throw new WalletProviderError({
      code: "provider_budget_exceeded",
      message: `Etherscan provider daily budget exhausted. Requests resume after ${resetAt}.`,
    });
  }
}

async function resolveQuotaIdentity(
  request: Request,
): Promise<{ scope: "anonymous" | "authenticated"; key: string }> {
  if (process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY?.trim()) {
    try {
      const { auth } = await import("@clerk/nextjs/server");
      const authState = await auth();
      if (authState.userId) {
        return {
          scope: "authenticated",
          key: hashIdentity(authState.userId),
        };
      }
    } catch {
      // Authentication unavailable is treated as anonymous public access.
    }
  }

  const forwarded = request.headers.get("x-forwarded-for")?.split(",")[0];
  const ip =
    forwarded?.trim() ||
    request.headers.get("x-real-ip")?.trim() ||
    "unknown-ip";
  return { scope: "anonymous", key: hashIdentity(ip) };
}

function quotaLimit(
  scope: "anonymous" | "authenticated",
  kind: PublicQuotaKind,
): number {
  if (kind === "analysis") {
    return scope === "authenticated"
      ? readPositiveIntegerEnv("QORVI_AUTH_ANALYSIS_DAILY_LIMIT", 10)
      : readPositiveIntegerEnv("QORVI_ANON_ANALYSIS_DAILY_LIMIT", 3);
  }
  return scope === "authenticated"
    ? readPositiveIntegerEnv("QORVI_AUTH_CHAT_DAILY_LIMIT", 50)
    : readPositiveIntegerEnv("QORVI_ANON_CHAT_DAILY_LIMIT", 30);
}

function utcDailyWindow(): {
  day: string;
  ttlSeconds: number;
  resetAt: string;
} {
  const now = new Date();
  const next = new Date(now);
  next.setUTCHours(24, 0, 0, 0);
  return {
    day: now.toISOString().slice(0, 10),
    ttlSeconds: Math.max(
      60,
      Math.ceil((next.getTime() - now.getTime()) / 1000),
    ),
    resetAt: next.toISOString(),
  };
}

function hashIdentity(identity: string): string {
  return createHash("sha256").update(identity).digest("hex").slice(0, 24);
}

function readPositiveIntegerEnv(name: string, fallback: number): number {
  const value = Number(process.env[name]);
  return Number.isInteger(value) && value > 0 ? value : fallback;
}
