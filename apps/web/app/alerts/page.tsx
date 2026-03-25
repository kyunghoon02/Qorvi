import { loadAlertCenterPreview } from "../../lib/api-boundary";
import { buildClerkRequestHeaders } from "../../lib/clerk-server-auth";
import {
  buildTrackedWalletAlertFlash,
  normalizeTrackedWalletAlertQueryState,
} from "./alert-center-flash";

import { AlertCenterScreen } from "./alert-center-screen";

type AlertCenterPageProps = {
  searchParams?:
    | Promise<{
        severity?: string | string[];
        signalType?: string | string[];
        status?: string | string[];
        cursor?: string | string[];
        tracked?: string | string[];
        watchlistId?: string | string[];
        ruleId?: string | string[];
        wallet?: string | string[];
      }>
    | {
        severity?: string | string[];
        signalType?: string | string[];
        status?: string | string[];
        cursor?: string | string[];
        tracked?: string | string[];
        watchlistId?: string | string[];
        ruleId?: string | string[];
        wallet?: string | string[];
      };
};

function normalizeSeverityFilter(
  raw: string | string[] | undefined,
): "all" | "low" | "medium" | "high" | "critical" {
  const value = Array.isArray(raw) ? raw[0] : raw;
  if (
    value === "low" ||
    value === "medium" ||
    value === "high" ||
    value === "critical"
  ) {
    return value;
  }
  return "all";
}

function normalizeSignalTypeFilter(
  raw: string | string[] | undefined,
): "all" | "cluster_score" | "shadow_exit" | "first_connection" {
  const value = Array.isArray(raw) ? raw[0] : raw;
  if (
    value === "cluster_score" ||
    value === "shadow_exit" ||
    value === "first_connection"
  ) {
    return value;
  }
  return "all";
}

function normalizeStatusFilter(
  raw: string | string[] | undefined,
): "all" | "unread" {
  const value = Array.isArray(raw) ? raw[0] : raw;
  if (value === "unread") {
    return value;
  }
  return "all";
}

function normalizeCursor(
  raw: string | string[] | undefined,
): string | undefined {
  const value = Array.isArray(raw) ? raw[0] : raw;
  const trimmed = value?.trim();
  return trimmed ? trimmed : undefined;
}

export default async function AlertCenterPage({
  searchParams,
}: AlertCenterPageProps) {
  const requestHeaders = await buildClerkRequestHeaders();
  const resolvedSearchParams = searchParams
    ? await Promise.resolve(searchParams)
    : undefined;

  const preview = await loadAlertCenterPreview({
    severity: normalizeSeverityFilter(resolvedSearchParams?.severity),
    signalType: normalizeSignalTypeFilter(resolvedSearchParams?.signalType),
    status: normalizeStatusFilter(resolvedSearchParams?.status),
    cursor: normalizeCursor(resolvedSearchParams?.cursor),
    ...(requestHeaders ? { requestHeaders } : {}),
  });
  const trackedWalletState = normalizeTrackedWalletAlertQueryState({
    tracked: resolvedSearchParams?.tracked,
    wallet: resolvedSearchParams?.wallet,
    watchlistId: resolvedSearchParams?.watchlistId,
    ruleId: resolvedSearchParams?.ruleId,
  });
  const flash = buildTrackedWalletAlertFlash(trackedWalletState);

  return <AlertCenterScreen preview={preview} flash={flash} />;
}
