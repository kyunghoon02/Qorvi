import { NextResponse } from "next/server";

import { answerWalletQuestion } from "../../../../lib/wallet-copilot/agent";
import { isWalletProviderError } from "../../../../lib/wallet-copilot/errors";
import { consumePublicQuota } from "../../../../lib/wallet-copilot/quota";

export async function POST(request: Request) {
  try {
    const body = (await request.json()) as {
      address?: unknown;
      days?: unknown;
      question?: unknown;
    };
    const address = typeof body.address === "string" ? body.address : "";
    const question = typeof body.question === "string" ? body.question : "";
    if (!question.trim()) {
      return NextResponse.json(
        { error: "Ask a wallet follow-up question.", code: "invalid_request" },
        { status: 400 },
      );
    }
    await consumePublicQuota(request, "chat");
    const result = await answerWalletQuestion({
      address,
      days: body.days,
      question,
    });
    return NextResponse.json(result);
  } catch (error) {
    if (isWalletProviderError(error)) {
      return NextResponse.json(
        { error: error.message, code: error.code },
        { status: error.status },
      );
    }

    const message =
      error instanceof Error ? error.message : "Wallet chat failed.";
    return NextResponse.json(
      { error: message, code: "invalid_request" },
      { status: 400 },
    );
  }
}
