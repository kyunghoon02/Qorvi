import { headers } from "next/headers";

import {
  deriveWalletGraphPreviewFromSummary,
  loadAnalystWalletBriefPreview,
  loadWalletGraphPreview,
  loadWalletSummaryPreview,
} from "../../../../lib/api-boundary";
import { buildForwardedAuthHeaders } from "../../../../lib/request-headers";

import {
  resolveFlowLensContextFromSearchParams,
  resolveWalletDetailRequestFromParams,
} from "./wallet-detail-route";
import { WalletDetailScreen } from "./wallet-detail-screen";

function InvalidWalletRoute() {
  return (
    <main className="page-shell detail-shell">
      <section className="empty-state">
        <h3>Wallet route not available</h3>
        <p>The chain must be `evm` or `solana`, and the address must exist.</p>
      </section>
    </main>
  );
}

export default async function WalletDetailPage({
  params,
  searchParams,
}: Readonly<{
  params: {
    chain: string;
    address: string;
  };
  searchParams?: Record<string, string | string[] | undefined>;
}>) {
  const requestHeaders = buildForwardedAuthHeaders(await headers());
  const request = resolveWalletDetailRequestFromParams(
    params.chain,
    params.address,
  );
  const flowLensContext = resolveFlowLensContextFromSearchParams(searchParams);

  if (!request) {
    return <InvalidWalletRoute />;
  }

  const [summary, brief, loadedGraph] = await Promise.all([
    loadWalletSummaryPreview({ request }),
    loadAnalystWalletBriefPreview({
      request,
      ...(requestHeaders ? { requestHeaders } : {}),
    }),
    loadWalletGraphPreview({
      request: {
        ...request,
        depthRequested: 1,
      },
    }),
  ]);
  const graph =
    loadedGraph.mode === "unavailable" && summary.topCounterparties.length > 0
      ? deriveWalletGraphPreviewFromSummary({
          request: {
            ...request,
            depthRequested: 1,
          },
          summary,
          fallback: loadedGraph,
        })
      : loadedGraph;

  return (
    <WalletDetailScreen
      request={request}
      summary={summary}
      brief={brief}
      graph={graph}
      flowLensContext={flowLensContext}
      {...(requestHeaders ? { requestHeaders } : {})}
    />
  );
}
