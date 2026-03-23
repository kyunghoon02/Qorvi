import { loadAdminConsolePreview } from "../../lib/api-boundary";

import { AdminConsoleScreen } from "./admin-console-screen";

export default async function AdminConsolePage() {
  const preview = await loadAdminConsolePreview();
  return <AdminConsoleScreen preview={preview} />;
}
