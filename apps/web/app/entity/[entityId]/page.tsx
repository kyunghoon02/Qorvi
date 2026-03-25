import { headers } from "next/headers";

import {
  buildWalletDetailHref,
  loadAnalystEntityInterpretationPreview,
} from "../../../lib/api-boundary";
import { buildForwardedAuthHeaders } from "../../../lib/request-headers";

import { EntityScreen } from "./entity-screen";

export default async function EntityPage({
  params,
}: Readonly<{
  params: {
    entityId: string;
  };
}>) {
  let entityId = params.entityId;
  try {
    entityId = decodeURIComponent(params.entityId);
  } catch {
    entityId = params.entityId;
  }

  const requestHeaders = buildForwardedAuthHeaders(await headers());
  const entity = await loadAnalystEntityInterpretationPreview({
    request: { entityKey: entityId },
    ...(requestHeaders ? { requestHeaders } : {}),
  });

  return (
    <EntityScreen
      entity={entity}
      backHref="/"
      walletHrefBuilder={buildWalletDetailHref}
    />
  );
}
