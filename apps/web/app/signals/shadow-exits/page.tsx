import { loadShadowExitFeedPreview } from "../../../lib/api-boundary";

import { ShadowExitFeedScreen } from "./shadow-exit-feed-screen";

export default async function ShadowExitFeedPage() {
  const feed = await loadShadowExitFeedPreview();

  return <ShadowExitFeedScreen feed={feed} />;
}
