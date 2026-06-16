import { NextResponse } from "next/server";

import { isWalletProviderError } from "../../../../../lib/wallet-copilot/errors";
import { processWalletAnalysisQueue } from "../../../../../lib/wallet-copilot/jobs";
import { logWalletCopilotEvent } from "../../../../../lib/wallet-copilot/observability";
import { authorizeWalletWorkerRequest } from "../../../../../lib/wallet-copilot/worker-auth";

export async function GET(request: Request) {
  const auth = authorizeWalletWorkerRequest(request);
  if (!auth.ok) {
    return NextResponse.json(
      { error: auth.message, code: auth.code },
      { status: auth.status },
    );
  }

  const url = new URL(request.url);
  const queryLimit = url.searchParams.get("limit");
  const limit = Number(
    queryLimit ?? process.env.QORVI_WALLET_WORKER_CRON_LIMIT ?? 1,
  );
  try {
    const result = await processWalletAnalysisQueue({ limit });
    logWalletCopilotEvent({
      event: "analysis_worker_cron_invoked",
      processed: result.processed,
      succeeded: result.succeeded,
      failed: result.failed,
      empty: result.empty,
    });

    return NextResponse.json(result);
  } catch (error) {
    if (isWalletProviderError(error)) {
      return NextResponse.json(
        { error: error.message, code: error.code },
        { status: error.status },
      );
    }
    return NextResponse.json(
      {
        error:
          error instanceof Error
            ? error.message
            : "Wallet analysis cron failed.",
        code: "worker_failed",
      },
      { status: 500 },
    );
  }
}
