import { loadClusterDetailPreview } from "../../../lib/api-boundary";

import { resolveClusterDetailRequestFromParams } from "./cluster-detail-route";
import { ClusterDetailScreen } from "./cluster-detail-screen";

function InvalidClusterRoute() {
  return (
    <main className="page-shell detail-shell">
      <section className="empty-state">
        <h3>Cluster route not available</h3>
        <p>The cluster id must exist and cannot be empty.</p>
      </section>
    </main>
  );
}

export default async function ClusterDetailPage({
  params,
}: Readonly<{
  params: {
    clusterId: string;
  };
}>) {
  const request = resolveClusterDetailRequestFromParams(params.clusterId);

  if (!request) {
    return <InvalidClusterRoute />;
  }

  const cluster = await loadClusterDetailPreview({ request });

  return <ClusterDetailScreen cluster={cluster} />;
}
