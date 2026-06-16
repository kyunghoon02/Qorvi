import { NextResponse } from "next/server";

import { getWalletAnalysisJob } from "../../../../../../lib/wallet-copilot/jobs";

export async function GET(
  _request: Request,
  { params }: { params: { jobId: string } },
) {
  const job = await getWalletAnalysisJob(params.jobId);
  if (!job) {
    return NextResponse.json(
      {
        error: "Wallet analysis job was not found or has expired.",
        code: "job_not_found",
      },
      { status: 404 },
    );
  }

  return NextResponse.json(job);
}
