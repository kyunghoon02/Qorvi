import { loadAdminConsolePreview } from "../../lib/api-boundary";
import { buildClerkRequestHeaders } from "../../lib/clerk-server-auth";

import { AdminConsoleScreen } from "./admin-console-screen";

export default async function AdminConsolePage() {
  const requestHeaders = await buildClerkRequestHeaders();
  const preview = await loadAdminConsolePreview(
    requestHeaders ? { requestHeaders } : undefined,
  );
  return <AdminConsoleScreen preview={preview} />;
}
