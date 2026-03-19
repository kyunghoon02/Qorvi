import {
  type WalletDetailRequest,
  loadWalletGraphPreview,
  loadWalletSummaryPreview,
} from "../../../../lib/api-boundary.js";

import { WalletDetailScreen } from "./wallet-detail-screen.js";

function safeDecodeURIComponent(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

export function resolveWalletDetailRequestFromParams(
  chain: string,
  address: string,
): WalletDetailRequest | null {
  if (chain !== "evm" && chain !== "solana") {
    return null;
  }

  const decodedAddress = safeDecodeURIComponent(address).trim();

  if (!decodedAddress) {
    return null;
  }

  return {
    chain,
    address: decodedAddress,
  };
}

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
}: Readonly<{
  params: {
    chain: string;
    address: string;
  };
}>) {
  const request = resolveWalletDetailRequestFromParams(
    params.chain,
    params.address,
  );

  if (!request) {
    return <InvalidWalletRoute />;
  }

  const [summary, graph] = await Promise.all([
    loadWalletSummaryPreview({ request }),
    loadWalletGraphPreview({
      request: {
        ...request,
        depthRequested: 2,
      },
    }),
  ]);

  return (
    <WalletDetailScreen request={request} summary={summary} graph={graph} />
  );
}
