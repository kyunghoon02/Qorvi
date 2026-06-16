import { NextResponse } from "next/server";

import { isWalletProviderError } from "../../../../lib/wallet-copilot/errors";
import { createWalletAnalysisJob } from "../../../../lib/wallet-copilot/jobs";
import { consumePublicQuota } from "../../../../lib/wallet-copilot/quota";

export async function POST(request: Request) {
  try {
    const body = (await request.json()) as {
      address?: unknown;
      days?: unknown;
    };
    const address = typeof body.address === "string" ? body.address : "";
    const result = await createWalletAnalysisJob({
      address,
      daysInput: body.days,
      consumeQuota: () => consumePublicQuota(request, "analysis"),
    });
    return NextResponse.json(result, {
      status: result.status === "succeeded" ? 200 : 202,
    });
  } catch (error) {
    if (isWalletProviderError(error)) {
      return NextResponse.json(
        { error: error.message, code: error.code },
        { status: error.status },
      );
    }

    const message =
      error instanceof Error ? error.message : "Wallet analysis failed.";
    return NextResponse.json(
      { error: message, code: "invalid_request" },
      { status: 400 },
    );
  }
}
