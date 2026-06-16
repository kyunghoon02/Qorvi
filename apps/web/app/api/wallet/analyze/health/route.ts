import { NextResponse } from "next/server";

import { checkWalletIndexRepositoryHealth } from "../../../../../lib/wallet-copilot/index-repository";
import { resolveWalletAnalysisExecutionMode } from "../../../../../lib/wallet-copilot/jobs";
import { checkWalletCopilotStorageHealth } from "../../../../../lib/wallet-copilot/storage";
import { authorizeWalletWorkerRequest } from "../../../../../lib/wallet-copilot/worker-auth";

export async function GET(request: Request) {
  const auth = authorizeWalletWorkerRequest(request);
  if (!auth.ok) {
    return NextResponse.json(
      { error: auth.message, code: auth.code },
      { status: auth.status },
    );
  }

  const [storage, indexStore] = await Promise.all([
    checkWalletCopilotStorageHealth(),
    checkWalletIndexRepositoryHealth(),
  ]);
  const status = storage.ok && indexStore.ok ? 200 : 503;
  return NextResponse.json(
    {
      ok: storage.ok && indexStore.ok,
      storage: storage.storage,
      redis_configured: storage.redis_configured,
      redis_url_configured: storage.redis_url_configured,
      upstash_redis_configured: storage.upstash_redis_configured,
      execution_mode: resolveWalletAnalysisExecutionMode(),
      postgres_configured: indexStore.postgres_configured,
      index_store_ok: indexStore.ok,
      worker_secret_configured: Boolean(
        process.env.QORVI_WALLET_WORKER_SECRET?.trim() ||
          process.env.CRON_SECRET?.trim(),
      ),
      error: storage.error ?? indexStore.error,
    },
    { status },
  );
}
