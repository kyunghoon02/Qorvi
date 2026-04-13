import { headers } from "next/headers";

import {
  deriveWalletGraphPreviewFromSummary,
  loadAnalystWalletBriefPreview,
  loadSearchPreview,
  loadWalletGraphPreview,
  loadWalletSummaryPreview,
  shouldQueueWalletSummaryStaleRefresh,
} from "../../../../lib/api-boundary";
import { buildForwardedAuthHeaders } from "../../../../lib/request-headers";

import {
  resolveFlowLensContextFromSearchParams,
  resolveWalletDetailRequestFromParams,
} from "./wallet-detail-route";
import { WalletDetailScreen } from "./wallet-detail-screen";

const DEFAULT_WALLET_GRAPH_DEPTH = 1;

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
    loadWalletSummaryPreview({
      request,
      ...(requestHeaders ? { requestHeaders } : {}),
    }),
    loadAnalystWalletBriefPreview({
      request,
      ...(requestHeaders ? { requestHeaders } : {}),
    }),
    loadWalletGraphPreview({
      request: {
        ...request,
        depthRequested: DEFAULT_WALLET_GRAPH_DEPTH,
      },
      ...(requestHeaders ? { requestHeaders } : {}),
    }),
  ]);
  const shouldDeriveGraphFromSummary =
    loadedGraph.mode === "unavailable" && summary.mode === "live";
  const graph =
    shouldDeriveGraphFromSummary
      ? deriveWalletGraphPreviewFromSummary({
          request: {
            ...request,
            depthRequested: DEFAULT_WALLET_GRAPH_DEPTH,
          },
          summary,
          fallback: loadedGraph,
        })
      : loadedGraph;

  if (
    summary.mode === "unavailable" ||
    shouldQueueWalletSummaryStaleRefresh(summary)
  ) {
    await loadSearchPreview({
      query: request.address,
      ...(requestHeaders ? { requestHeaders } : {}),
    });
  }

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
