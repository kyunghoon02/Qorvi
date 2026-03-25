import type { Tone } from "@flowintel/ui";

export type TrackedWalletAlertQueryState = {
  status: "idle" | "success";
  wallet?: string;
  watchlistId?: string;
  ruleId?: string;
};

export function normalizeTrackedWalletAlertQueryState({
  tracked,
  wallet,
  watchlistId,
  ruleId,
}: {
  tracked: string | string[] | undefined;
  wallet: string | string[] | undefined;
  watchlistId: string | string[] | undefined;
  ruleId: string | string[] | undefined;
}): TrackedWalletAlertQueryState {
  const trackedValue = Array.isArray(tracked) ? tracked[0] : tracked;
  const walletValue = Array.isArray(wallet) ? wallet[0] : wallet;
  const watchlistValue = Array.isArray(watchlistId)
    ? watchlistId[0]
    : watchlistId;
  const ruleValue = Array.isArray(ruleId) ? ruleId[0] : ruleId;

  return {
    status: trackedValue === "success" ? "success" : "idle",
    ...(walletValue?.trim() ? { wallet: walletValue.trim() } : {}),
    ...(watchlistValue?.trim() ? { watchlistId: watchlistValue.trim() } : {}),
    ...(ruleValue?.trim() ? { ruleId: ruleValue.trim() } : {}),
  };
}

export function buildTrackedWalletAlertFlash(
  state: TrackedWalletAlertQueryState,
):
  | {
      tone: Tone;
      title: string;
      message: string;
    }
  | undefined {
  if (state.status !== "success") {
    return undefined;
  }

  const walletLabel = state.wallet ? `${state.wallet} is now tracked.` : "";
  const watchlistLabel = state.watchlistId
    ? `Watchlist ${state.watchlistId}`
    : "Tracked wallets";
  const ruleLabel = state.ruleId
    ? `rule ${state.ruleId}`
    : "the default signal rule";

  return {
    tone: "teal",
    title: "Wallet tracking active",
    message: `${walletLabel} ${watchlistLabel} and ${ruleLabel} are ready in this alert center.`,
  };
}
