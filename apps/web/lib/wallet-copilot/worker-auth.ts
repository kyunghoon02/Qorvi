export type WalletWorkerAuthResult =
  | { ok: true }
  | { ok: false; code: string; message: string; status: number };

export function authorizeWalletWorkerRequest(
  request: Request,
): WalletWorkerAuthResult {
  const secret = resolveWalletWorkerSecret();
  if (!secret) {
    if (process.env.NODE_ENV === "production") {
      return {
        ok: false,
        code: "worker_secret_missing",
        message:
          "QORVI_WALLET_WORKER_SECRET or CRON_SECRET is required in production.",
        status: 503,
      };
    }
    return { ok: true };
  }

  const authorization = request.headers.get("authorization") ?? "";
  const bearerToken = authorization.startsWith("Bearer ")
    ? authorization.slice("Bearer ".length).trim()
    : "";
  const headerToken = request.headers.get("x-qorvi-worker-secret")?.trim();
  if (bearerToken === secret || headerToken === secret) {
    return { ok: true };
  }

  return {
    ok: false,
    code: "unauthorized",
    message: "Unauthorized wallet analysis worker request.",
    status: 401,
  };
}

function resolveWalletWorkerSecret(): string {
  return (
    process.env.QORVI_WALLET_WORKER_SECRET?.trim() ??
    process.env.CRON_SECRET?.trim() ??
    ""
  );
}
