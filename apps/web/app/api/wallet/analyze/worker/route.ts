import { NextResponse } from "next/server";

import { isWalletProviderError } from "../../../../../lib/wallet-copilot/errors";
import { processWalletAnalysisQueue } from "../../../../../lib/wallet-copilot/jobs";
import { authorizeWalletWorkerRequest } from "../../../../../lib/wallet-copilot/worker-auth";

export async function POST(request: Request) {
  const auth = authorizeWalletWorkerRequest(request);
  if (!auth.ok) {
    return NextResponse.json(
      { error: auth.message, code: auth.code },
      { status: auth.status },
    );
  }

  const url = new URL(request.url);
  const limit = Number(url.searchParams.get("limit") ?? 1);
  try {
    const result = await processWalletAnalysisQueue({ limit });
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
            : "Wallet analysis worker failed.",
        code: "worker_failed",
      },
      { status: 500 },
    );
  }
}
