import { loadFirstConnectionFeedPreview } from "../../../lib/api-boundary";

import { FirstConnectionFeedScreen } from "./first-connection-feed-screen";

type FirstConnectionFeedPageProps = {
  searchParams?:
    | Promise<{ sort?: string | string[] }>
    | { sort?: string | string[] };
};

function normalizeFirstConnectionSort(
  raw: string | string[] | undefined,
): "latest" | "score" {
  const value = Array.isArray(raw) ? raw[0] : raw;
  return value === "score" ? "score" : "latest";
}

export default async function FirstConnectionFeedPage({
  searchParams,
}: FirstConnectionFeedPageProps) {
  const resolvedSearchParams = searchParams
    ? await Promise.resolve(searchParams)
    : undefined;
  const feed = await loadFirstConnectionFeedPreview({
    sort: normalizeFirstConnectionSort(resolvedSearchParams?.sort),
  });

  return <FirstConnectionFeedScreen feed={feed} />;
}
