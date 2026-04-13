import { auth, currentUser } from "@clerk/nextjs/server";
import { notFound } from "next/navigation";

import { loadAdminConsolePreview } from "../../lib/api-boundary";
import { resolveClerkRole } from "../../lib/clerk-role";
import { buildClerkRequestHeaders } from "../../lib/clerk-server-auth";

import { AdminConsoleScreen } from "./admin-console-screen";

export default async function AdminConsolePage() {
  const authState = await auth();
  const user = await currentUser();
  const role =
    resolveClerkRole(authState.sessionClaims) ?? resolveClerkRole(user);
  if (role !== "admin") {
    notFound();
  }
  if (!isAllowedAdminViewer(authState.userId, authState.sessionClaims, user)) {
    notFound();
  }
  const requestHeaders = await buildClerkRequestHeaders();
  const preview = await loadAdminConsolePreview(
    requestHeaders ? { requestHeaders } : undefined,
  );
  return <AdminConsoleScreen preview={preview} />;
}

function isAllowedAdminViewer(
  userId: string | null,
  sessionClaims: unknown,
  user: unknown,
): boolean {
  const allowedUserIds = parseAllowlist(
    process.env.QORVI_ADMIN_ALLOWLIST_USER_IDS,
  );
  const allowedEmails = parseAllowlist(
    process.env.QORVI_ADMIN_ALLOWLIST_EMAILS,
  );
  if (allowedUserIds.size === 0 && allowedEmails.size === 0) {
    return true;
  }
  const normalizedUserId = userId?.trim().toLowerCase() ?? "";
  if (normalizedUserId && allowedUserIds.has(normalizedUserId)) {
    return true;
  }
  const email =
    readAdminViewerEmail(sessionClaims) ?? readAdminViewerEmail(user);
  return email ? allowedEmails.has(email.toLowerCase()) : false;
}

function parseAllowlist(rawValue: string | undefined): Set<string> {
  return new Set(
    (rawValue ?? "")
      .split(",")
      .map((value) => value.trim().toLowerCase())
      .filter(Boolean),
  );
}

function readAdminViewerEmail(sessionClaims: unknown): string | undefined {
  if (!sessionClaims || typeof sessionClaims !== "object") {
    return undefined;
  }
  const claims = sessionClaims as Record<string, unknown>;
  for (const candidate of [
    claims.email,
    claims.email_address,
    claims.emailAddress,
  ]) {
    if (typeof candidate === "string" && candidate.trim()) {
      return candidate.trim();
    }
  }
  const primaryEmailAddress = claims.primaryEmailAddress;
  if (primaryEmailAddress && typeof primaryEmailAddress === "object") {
    for (const candidate of [
      (primaryEmailAddress as Record<string, unknown>).emailAddress,
      (primaryEmailAddress as Record<string, unknown>).email_address,
    ]) {
      if (typeof candidate === "string" && candidate.trim()) {
        return candidate.trim();
      }
    }
  }
  return undefined;
}
