"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Badge } from "@flowintel/ui";

import {
  type FindingPreview,
  type FindingsFeedPreview,
  type SearchPreview,
  type WalletGraphPreview,
  type WalletGraphPreviewEdge,
  type WalletGraphPreviewNode,
  type WalletSummaryPreview,
  type WalletSummaryRequest,
  buildEntityDetailHref,
  buildProductSearchHref,
  buildWalletDetailHref,
  deriveWalletGraphPreviewFromSummary,
  getAnalystFindingsPreview,
  getSearchPreview,
  getWalletGraphPreview,
  getWalletSummaryPreview,
  loadAnalystFindingsPreview,
  loadSearchPreview,
  loadWalletGraphPreview,
  loadWalletSummaryPreview,
  resolveWalletSummaryRequestFromRoute,
  shouldPersistSearchPreviewToUrl,
  shouldPollIndexedWalletSummary,
} from "../lib/api-boundary";
import { persistClientForwardedAuthHeaders } from "../lib/request-headers";
import { quickQueries } from "../lib/sprint0";
import { NetworkBackground } from "./components/network-background";
import {
  type GraphEntityAssignmentPresentation,
  buildCounterpartyEntityAssignment,
  buildGraphEntityAssignmentIndex,
  buildSelectedGraphNodeHref,
  buildSelectedGraphNodeHrefLabel,
  buildWalletGraphAvailabilityPresentation,
  describeGraphRelationshipDirection,
} from "./wallets/[chain]/[address]/wallet-graph-presenter";
import { WalletGraphVisual } from "./wallets/[chain]/[address]/wallet-graph-visual";
import {
  buildWalletGraphEdgeKey,
  formatGraphKind,
  getWalletGraphEdgeFamilyLabel,
  getWalletGraphEdgeKindLabel,
} from "./wallets/[chain]/[address]/wallet-graph-visual-model";

const HOME_GRAPH_HOP_BUDGET = 20;
const HOME_GRAPH_NODE_BUDGET = 120;

export function resolveWalletRequestFromSearchPreview(
  preview: SearchPreview,
): WalletSummaryRequest | null {
  if (!preview.navigation || !preview.walletRoute) {
    return null;
  }

  return resolveWalletSummaryRequestFromRoute(preview.walletRoute);
}

export function shouldHydrateHomeSearchQuery(
  urlQuery: string,
  lastHydratedUrlQuery: string | null,
): boolean {
  return urlQuery !== lastHydratedUrlQuery;
}

export function shouldPollHomeWalletPreview(
  preview: WalletSummaryPreview,
): boolean {
  return shouldPollIndexedWalletSummary(preview);
}

export function getHomeCoverageActionLabel(
  preview: WalletSummaryPreview,
): string {
  return preview.indexing.status === "indexing"
    ? "Continue indexing"
    : "Expand coverage";
}

export type HomeFindingFeedItem = {
  id: string;
  title: string;
  findingTypeLabel: string;
  summary: string;
  evidenceLabel: string;
  nextWatchLabel: string;
  nextWatchHref: string | null;
  analystEntryLabel: string;
  analystEntryHref: string | null;
  importance: number;
  confidence: number;
  subjectLabel: string;
  subjectHref: string | null;
  subjectTypeLabel: string;
  badgeTone: "teal" | "amber" | "violet" | "emerald";
};

export function buildHomeFindingsFeedItems(
  preview: WalletSummaryPreview,
  walletDetailHref: string | null,
): HomeFindingFeedItem[] {
  const items: HomeFindingFeedItem[] = [];

  for (const score of preview.scores.slice(0, 2)) {
    const title = formatScoreLabel(score.name);
    items.push({
      id: `score:${score.name}`,
      title,
      findingTypeLabel: "Signal interpretation",
      summary:
        score.rating === "high"
          ? `${title} is elevated and worth reviewing first.`
          : `${title} is active in the current indexed coverage window.`,
      evidenceLabel: `Derived from wallet score ${score.value}/100`,
      nextWatchLabel: "Open wallet brief",
      nextWatchHref: walletDetailHref,
      analystEntryLabel: "Analyze wallet",
      analystEntryHref: walletDetailHref,
      importance: score.value / 100,
      confidence: score.value / 100,
      subjectLabel: preview.label,
      subjectHref: walletDetailHref,
      subjectTypeLabel: "Wallet",
      badgeTone: score.tone,
    });
  }

  for (const signal of preview.latestSignals.slice(0, 2)) {
    items.push({
      id: `signal:${signal.name}:${signal.observedAt}`,
      title: signal.label || formatScoreLabel(signal.name),
      findingTypeLabel: "Signal interpretation",
      summary:
        signal.rating === "high"
          ? `${signal.label || formatScoreLabel(signal.name)} is the latest high-priority signal.`
          : `${signal.label || formatScoreLabel(signal.name)} is still active in the current coverage window.`,
      evidenceLabel: `Source ${signal.source} · observed ${signal.observedAt.slice(0, 10)}`,
      nextWatchLabel: "Open wallet brief",
      nextWatchHref: walletDetailHref,
      analystEntryLabel: "Analyze wallet",
      analystEntryHref: walletDetailHref,
      importance: signal.value / 100,
      confidence: signal.value / 100,
      subjectLabel: preview.label,
      subjectHref: walletDetailHref,
      subjectTypeLabel: "Wallet",
      badgeTone: signal.rating === "high" ? "emerald" : "amber",
    });
  }

  const topCounterparty = preview.topCounterparties[0];
  if (topCounterparty) {
    items.push({
      id: `counterparty:${topCounterparty.chain}:${topCounterparty.address}`,
      title: `${topCounterparty.directionLabel} counterparty`,
      findingTypeLabel: "Counterparty evidence",
      summary: `${compactAddress(topCounterparty.address)} has ${topCounterparty.interactionCount} indexed interactions and ${topCounterparty.primaryToken} is the dominant token.`,
      evidenceLabel: topCounterparty.latestActivityAt
        ? `${topCounterparty.interactionCount} indexed interactions · latest activity ${topCounterparty.latestActivityAt.slice(0, 10)}`
        : `${topCounterparty.interactionCount} indexed interactions`,
      nextWatchLabel: topCounterparty.entityLabel
        ? `Open ${topCounterparty.entityLabel}`
        : `Watch ${compactAddress(topCounterparty.address)}`,
      nextWatchHref: topCounterparty.entityLabel
        ? buildProductSearchHref(topCounterparty.entityLabel)
        : buildWalletDetailHref({
            chain: topCounterparty.chain,
            address: topCounterparty.address,
          }),
      analystEntryLabel: topCounterparty.entityLabel
        ? "Analyze entity"
        : "Analyze wallet",
      analystEntryHref: topCounterparty.entityLabel
        ? buildProductSearchHref(topCounterparty.entityLabel)
        : buildWalletDetailHref({
            chain: topCounterparty.chain,
            address: topCounterparty.address,
          }),
      importance:
        preview.counterparties > 0
          ? topCounterparty.interactionCount / preview.counterparties
          : topCounterparty.interactionCount / 10,
      confidence:
        topCounterparty.interactionCount >= 8
          ? 0.82
          : topCounterparty.interactionCount >= 3
            ? 0.68
            : 0.5,
      subjectLabel:
        topCounterparty.entityLabel || compactAddress(topCounterparty.address),
      subjectHref: topCounterparty.entityLabel
        ? buildProductSearchHref(topCounterparty.entityLabel)
        : buildWalletDetailHref({
            chain: topCounterparty.chain,
            address: topCounterparty.address,
          }),
      subjectTypeLabel: topCounterparty.entityLabel ? "Entity" : "Wallet",
      badgeTone:
        topCounterparty.directionLabel === "inbound" ? "violet" : "teal",
    });
  }

  return items
    .sort((left, right) => right.importance - left.importance)
    .slice(0, 4);
}

export function buildHomeFindingsFeedItemsFromFeed(
  preview: FindingsFeedPreview,
): HomeFindingFeedItem[] {
  return preview.items.slice(0, 6).map((item) => ({
    id: item.id,
    title: formatFindingTypeLabel(item.type),
    findingTypeLabel: formatFindingSubjectTypeLabel(item.subjectType),
    summary: item.summary,
    evidenceLabel:
      item.observedFacts[0] ??
      item.importanceReason[0] ??
      item.evidence[0]?.value ??
      "Evidence bundle available",
    nextWatchLabel: resolveFindingNextWatchLabel(item),
    nextWatchHref: resolveFindingNextWatchHref(item),
    analystEntryLabel: resolveFindingAnalystEntryLabel(item),
    analystEntryHref: resolveFindingAnalystEntryHref(item),
    importance: item.importanceScore,
    confidence: item.confidence,
    subjectLabel: resolveFindingSubjectLabel(item),
    subjectHref: resolveFindingSubjectHref(item),
    subjectTypeLabel: formatFindingSubjectTypeLabel(item.subjectType),
    badgeTone: toneForFindingType(item.type),
  }));
}

export function HomeScreen({
  requestHeaders,
}: {
  requestHeaders?: HeadersInit;
}) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryFromUrl = searchParams.get("q")?.trim() ?? "";
  const lastHydratedUrlQuery = useRef<string | null>(queryFromUrl);
  const [query, setQuery] = useState("");
  const [searchPreview, setSearchPreview] = useState(() => getSearchPreview());
  const [findingsFeedPreview, setFindingsFeedPreview] = useState(() =>
    getAnalystFindingsPreview(),
  );
  const [walletRequest, setWalletRequest] =
    useState<WalletSummaryRequest | null>(null);
  const [preview, setPreview] = useState(() =>
    getWalletSummaryPreview({ chain: "evm", address: "" }),
  );
  const [graphPreview, setGraphPreview] = useState(() =>
    getWalletGraphPreview({ chain: "evm", address: "", depthRequested: 1 }),
  );
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [hoveredNodeId, setHoveredNodeId] = useState<string | null>(null);
  const [expandedGraphNeighborhoodKeys, setExpandedGraphNeighborhoodKeys] =
    useState<string[]>([]);
  const [expandingNodeId, setExpandingNodeId] = useState<string | null>(null);
  const selectedNode = useMemo(() => {
    return (
      graphPreview.nodes.find((node) => node.id === selectedNodeId) ?? null
    );
  }, [graphPreview.nodes, selectedNodeId]);
  const hoveredNode = useMemo(() => {
    return graphPreview.nodes.find((node) => node.id === hoveredNodeId) ?? null;
  }, [graphPreview.nodes, hoveredNodeId]);
  const activeNode =
    hoveredNode ?? selectedNode ?? graphPreview.nodes[0] ?? null;
  const [isRefreshingWalletPreview, setIsRefreshingWalletPreview] =
    useState(false);
  const walletRequestForDetail =
    resolveWalletRequestFromSearchPreview(searchPreview);
  const walletDetailHref = walletRequestForDetail
    ? buildWalletDetailHref(walletRequestForDetail)
    : null;
  const findingsFeedItems = useMemo(() => {
    if (findingsFeedPreview.items.length > 0) {
      return buildHomeFindingsFeedItemsFromFeed(findingsFeedPreview);
    }
    return buildHomeFindingsFeedItems(preview, walletDetailHref);
  }, [findingsFeedPreview, preview, walletDetailHref]);

  useEffect(() => {
    persistClientForwardedAuthHeaders(requestHeaders);
  }, [requestHeaders]);

  useEffect(() => {
    let active = true;
    void (async () => {
      const nextFeed = await loadAnalystFindingsPreview({
        ...(requestHeaders ? { requestHeaders } : {}),
      });
      if (!active) {
        return;
      }
      setFindingsFeedPreview(nextFeed);
    })();

    return () => {
      active = false;
    };
  }, [requestHeaders]);

  const runSearch = useCallback(
    async (nextQuery: string, syncUrl = false) => {
      const trimmed = nextQuery.trim();
      const nextSearchPreview = await loadSearchPreview({ query: trimmed });
      setQuery(trimmed);
      setSearchPreview(nextSearchPreview);
      setWalletRequest(
        resolveWalletRequestFromSearchPreview(nextSearchPreview),
      );
      setExpandedGraphNeighborhoodKeys([]);
      setExpandingNodeId(null);

      if (syncUrl) {
        lastHydratedUrlQuery.current = trimmed;
        router.replace(
          trimmed && shouldPersistSearchPreviewToUrl(nextSearchPreview)
            ? buildProductSearchHref(trimmed)
            : "/",
          {
            scroll: false,
          },
        );
      }
    },
    [router],
  );

  useEffect(() => {
    if (
      !shouldHydrateHomeSearchQuery(queryFromUrl, lastHydratedUrlQuery.current)
    ) {
      return;
    }

    lastHydratedUrlQuery.current = queryFromUrl;
    void runSearch(queryFromUrl);
  }, [queryFromUrl, runSearch]);

  const refreshWalletArtifacts = useCallback(
    async ({
      triggerRefreshQueue = false,
      summaryFallback,
      graphFallback,
      canCommit = () => true,
    }: {
      triggerRefreshQueue?: boolean;
      summaryFallback?: WalletSummaryPreview;
      graphFallback?: WalletGraphPreview;
      canCommit?: () => boolean;
    } = {}) => {
      if (!walletRequest) {
        return;
      }

      if (triggerRefreshQueue) {
        const nextSearchPreview = await loadSearchPreview({
          query: walletRequest.address,
          fallback: searchPreview,
          refreshMode: "manual",
        });
        setSearchPreview(nextSearchPreview);
      }

      const nextSummary = await loadWalletSummaryPreview(
        summaryFallback
          ? {
              request: walletRequest,
              fallback: summaryFallback,
            }
          : { request: walletRequest },
      );
      if (!canCommit()) {
        return;
      }
      setPreview(nextSummary);

      const loadedGraph = await loadWalletGraphPreview(
        graphFallback
          ? {
              request: {
                ...walletRequest,
                depthRequested: 1,
              },
              fallback: graphFallback,
            }
          : {
              request: {
                ...walletRequest,
                depthRequested: 1,
              },
            },
      );
      if (!canCommit()) {
        return;
      }
      setGraphPreview(
        loadedGraph.mode === "unavailable" &&
          nextSummary.topCounterparties.length > 0
          ? deriveWalletGraphPreviewFromSummary({
              request: {
                ...walletRequest,
                depthRequested: 1,
              },
              summary: nextSummary,
              fallback: loadedGraph,
            })
          : loadedGraph,
      );
    },
    [walletRequest, searchPreview],
  );

  useEffect(() => {
    let active = true;
    const syncWalletPreview = async () => {
      await refreshWalletArtifacts({
        canCommit: () => active,
      });
    };

    void syncWalletPreview();

    return () => {
      active = false;
    };
  }, [refreshWalletArtifacts]);

  useEffect(() => {
    if (!graphPreview.nodes.length) {
      setSelectedNodeId(null);
      setHoveredNodeId(null);
      return;
    }

    setSelectedNodeId((current) => {
      if (current && graphPreview.nodes.some((node) => node.id === current)) {
        return current;
      }
      return graphPreview.nodes[0]?.id ?? null;
    });
  }, [graphPreview.nodes]);

  const expandableGraphNodeIds = useMemo(() => {
    if (
      graphPreview.nodes.length >= HOME_GRAPH_NODE_BUDGET ||
      expandedGraphNeighborhoodKeys.length >= HOME_GRAPH_HOP_BUDGET
    ) {
      return [];
    }

    const expandedKeys = new Set(expandedGraphNeighborhoodKeys);

    return graphPreview.nodes
      .filter(
        (node) =>
          node.kind === "wallet" &&
          Boolean(node.chain) &&
          Boolean(node.address) &&
          !expandedKeys.has(buildHomeGraphExpansionKey(node)),
      )
      .map((node) => node.id);
  }, [expandedGraphNeighborhoodKeys, graphPreview.nodes]);

  const handleExpandGraphNode = useCallback(
    async (nodeId: string) => {
      const node =
        graphPreview.nodes.find((graphNode) => graphNode.id === nodeId) ?? null;
      if (
        !node ||
        node.kind !== "wallet" ||
        !node.chain ||
        !node.address ||
        expandedGraphNeighborhoodKeys.length >= HOME_GRAPH_HOP_BUDGET ||
        graphPreview.nodes.length >= HOME_GRAPH_NODE_BUDGET
      ) {
        return;
      }

      const expansionKey = buildHomeGraphExpansionKey(node);
      if (expandedGraphNeighborhoodKeys.includes(expansionKey)) {
        return;
      }

      setSelectedNodeId(nodeId);
      setExpandingNodeId(nodeId);
      try {
        const requestedGraph = await loadWalletGraphPreview({
          request: {
            chain: node.chain,
            address: node.address,
            depthRequested: 1,
          },
        });
        const nextGraph =
          requestedGraph.mode === "unavailable"
            ? rebaseExpandedGraphRootNode(
                deriveWalletGraphPreviewFromSummary({
                  request: {
                    chain: node.chain,
                    address: node.address,
                    depthRequested: 1,
                  },
                  summary: await loadWalletSummaryPreview({
                    request: {
                      chain: node.chain,
                      address: node.address,
                    },
                  }),
                  fallback: requestedGraph,
                }),
                node.id,
              )
            : requestedGraph;

        if (
          nextGraph.mode === "unavailable" &&
          nextGraph.source === "boundary-unavailable"
        ) {
          return;
        }

        setGraphPreview((current) =>
          mergeHomeGraphPreviews(current, nextGraph),
        );
        setExpandedGraphNeighborhoodKeys((current) => [
          ...current,
          expansionKey,
        ]);
      } finally {
        setExpandingNodeId(null);
      }
    },
    [expandedGraphNeighborhoodKeys, graphPreview.nodes],
  );

  useEffect(() => {
    if (!walletRequest || !shouldPollHomeWalletPreview(preview)) {
      return;
    }

    let active = true;
    const interval = window.setInterval(() => {
      void (async () => {
        if (!active) {
          return;
        }
        await refreshWalletArtifacts({
          summaryFallback: preview,
          graphFallback: graphPreview,
          canCommit: () => active,
        });
      })();
    }, 5000);

    return () => {
      active = false;
      window.clearInterval(interval);
    };
  }, [walletRequest, preview, graphPreview, refreshWalletArtifacts]);

  const graphRelationships = buildHomeGraphRelationships(graphPreview);
  const graphEntityAssignmentIndex = useMemo(
    () =>
      buildGraphEntityAssignmentIndex(graphPreview.nodes, graphPreview.edges),
    [graphPreview.edges, graphPreview.nodes],
  );
  const activeNodeEntityAssignments = useMemo(() => {
    if (!activeNode) {
      return [];
    }

    const graphAssignments =
      graphEntityAssignmentIndex.get(activeNode.id) ?? [];
    if (graphAssignments.length) {
      return graphAssignments;
    }
    if (activeNode.kind !== "wallet" || !activeNode.address) {
      return [];
    }

    const summaryCounterparty =
      preview.topCounterparties.find(
        (counterparty) =>
          counterparty.chain.toLowerCase() ===
            activeNode.chain?.toLowerCase() &&
          counterparty.address.toLowerCase() ===
            activeNode.address?.toLowerCase(),
      ) ?? null;
    const fallback = summaryCounterparty
      ? buildCounterpartyEntityAssignment(summaryCounterparty)
      : null;
    return fallback ? [fallback] : [];
  }, [activeNode, graphEntityAssignmentIndex, preview.topCounterparties]);
  const activeNodeRelationships = useMemo(() => {
    if (!activeNode) {
      return [];
    }

    return graphRelationships
      .filter(
        (relationship) =>
          relationship.sourceId === activeNode.id ||
          relationship.targetId === activeNode.id,
      )
      .slice(0, 3);
  }, [activeNode, graphRelationships]);
  const hasWalletPreview = Boolean(walletRequest && preview.address);
  const graphAvailability = useMemo(
    () => buildWalletGraphAvailabilityPresentation(graphPreview),
    [graphPreview],
  );

  return (
    <main className="home-fullscreen-layout">
      <NetworkBackground />
      <header className="home-fullscreen-header">
        <div className="home-fullscreen-brand">
          <h1
            style={{
              fontSize: "1.1rem",
              fontWeight: 600,
              letterSpacing: "-0.01em",
            }}
          >
            Qorvi
          </h1>
          {walletDetailHref ? (
            <a className="search-cta" href={walletDetailHref}>
              Open detail
            </a>
          ) : null}
        </div>

        {hasWalletPreview && (
          <form
            className="search-bar home-fullscreen-search"
            onSubmit={async (event) => {
              event.preventDefault();
              await runSearch(query, true);
            }}
          >
            <input
              id="wallet-search"
              value={query}
              onChange={(event) => setQuery(event.currentTarget.value)}
              placeholder="EVM or Solana address"
              aria-label="Search wallet intelligence"
            />
            <button type="submit">Search</button>
          </form>
        )}
      </header>

      <section className="home-fullscreen-body">
        {hasWalletPreview && findingsFeedItems.length > 0 ? (
          <aside className="home-fullscreen-panel" style={{ marginBottom: 16 }}>
            <article className="preview-card home-summary-card">
              <div className="preview-header">
                <div className="home-side-header">
                  <h2
                    style={{ fontSize: "1.2rem", fontWeight: 600, margin: 0 }}
                  >
                    Findings feed
                  </h2>
                  <p
                    style={{
                      margin: "4px 0 0",
                      color: "var(--muted)",
                      fontSize: "0.85rem",
                    }}
                  >
                    AI findings and signal interpretations from the current
                    indexed coverage.
                  </p>
                </div>
              </div>
              <div
                className="home-counterparty-stack"
                style={{ marginTop: 16 }}
              >
                {findingsFeedItems.map((item) => (
                  <div key={item.id} className="home-counterparty-card">
                    <div>
                      <strong>{item.title}</strong>
                      <span>{item.findingTypeLabel}</span>
                      <span>{item.evidenceLabel}</span>
                      <span>Next watch · {item.nextWatchLabel}</span>
                      <span>{item.summary}</span>
                      <span>
                        {item.subjectTypeLabel} · {item.subjectLabel}
                      </span>
                    </div>
                    <div
                      style={{
                        display: "flex",
                        gap: 8,
                        flexWrap: "wrap",
                        justifyContent: "flex-end",
                      }}
                    >
                      <Badge tone={item.badgeTone}>
                        {Math.round(item.importance * 100)} importance
                      </Badge>
                      <Badge tone="amber">
                        {Math.round(item.confidence * 100)} confidence
                      </Badge>
                      {item.subjectHref ? (
                        <a
                          className="search-cta home-inline-refresh"
                          href={item.subjectHref}
                        >
                          Open
                        </a>
                      ) : null}
                      {item.analystEntryHref ? (
                        <a
                          className="search-cta home-inline-refresh"
                          href={item.analystEntryHref}
                        >
                          {item.analystEntryLabel}
                        </a>
                      ) : null}
                      {item.nextWatchHref ? (
                        <a
                          className="search-cta home-inline-refresh"
                          href={item.nextWatchHref}
                        >
                          Next watch
                        </a>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            </article>
          </aside>
        ) : null}

        <div className="home-fullscreen-canvas">
          <div
            className="preview-header home-fullscreen-canvas-overlay"
            style={{
              background: "transparent",
              boxShadow: "none",
              border: "none",
            }}
          />

          {hasWalletPreview ? (
            <WalletGraphVisual
              densityCapped={graphPreview.densityCapped}
              nodes={graphPreview.nodes}
              edges={graphPreview.edges}
              neighborhoodSummary={graphPreview.neighborhoodSummary}
              variant="hero"
              expandableNodeIds={expandableGraphNodeIds}
              expandingNodeId={expandingNodeId}
              onExpandNode={(nodeId) => {
                void handleExpandGraphNode(nodeId);
              }}
              selectedNodeId={selectedNodeId}
              onSelectedNodeIdChange={setSelectedNodeId}
              onHoveredNodeIdChange={setHoveredNodeId}
            />
          ) : (
            <div className="graph-empty-state">
              <div
                className="graph-empty-content"
                style={{ width: "100%", maxWidth: "640px" }}
              >
                <strong style={{ fontSize: "2.5rem", marginBottom: "8px" }}>
                  Start with a wallet address
                </strong>
                <p style={{ fontSize: "1.2rem", marginBottom: "24px" }}>
                  Search an EVM or Solana wallet to load live summary, related
                  addresses, and the relationship map.
                </p>
                <form
                  className="search-bar"
                  onSubmit={async (event) => {
                    event.preventDefault();
                    await runSearch(query, true);
                  }}
                  style={{ width: "100%", display: "flex", gap: "8px" }}
                >
                  <input
                    id="wallet-search-hero"
                    value={query}
                    onChange={(event) => setQuery(event.currentTarget.value)}
                    placeholder="EVM or Solana address"
                    aria-label="Search wallet intelligence"
                    style={{
                      flex: 1,
                      padding: "20px 28px",
                      fontSize: "1.15rem",
                      borderRadius: "40px",
                      background: "rgba(255, 255, 255, 0.05)",
                      border: "1px solid rgba(255, 255, 255, 0.1)",
                      backdropFilter: "blur(10px)",
                    }}
                  />
                  <button
                    type="submit"
                    style={{
                      borderRadius: "40px",
                      padding: "0 32px",
                      fontSize: "1.1rem",
                      fontWeight: 500,
                    }}
                  >
                    Search
                  </button>
                </form>
              </div>
            </div>
          )}
        </div>

        {hasWalletPreview && activeNode ? (
          <aside className="home-fullscreen-panel">
            <article className="preview-card home-summary-card">
              <div className="preview-header">
                <div className="home-side-header">
                  <h2
                    style={{ fontSize: "1.2rem", fontWeight: 600, margin: 0 }}
                  >
                    {activeNode.label}
                  </h2>
                  <p
                    style={{
                      margin: "4px 0 0",
                      color: "var(--muted)",
                      fontSize: "0.85rem",
                    }}
                  >
                    {formatGraphKind(activeNode.kind)}
                  </p>
                </div>
                <div className="home-summary-actions">
                  <button
                    className="search-cta home-inline-refresh"
                    onClick={() => setSelectedNodeId(null)}
                    type="button"
                  >
                    Reset focus
                  </button>
                </div>
              </div>

              {activeNode.address === preview.address ? (
                <>
                  <div
                    className="home-summary-actions"
                    style={{ marginTop: 16 }}
                  >
                    {walletRequest ? (
                      <button
                        className="search-cta home-inline-refresh"
                        disabled={isRefreshingWalletPreview}
                        onClick={() => {
                          void (async () => {
                            setIsRefreshingWalletPreview(true);
                            try {
                              await refreshWalletArtifacts({
                                triggerRefreshQueue: true,
                                summaryFallback: preview,
                                graphFallback: graphPreview,
                              });
                            } finally {
                              setIsRefreshingWalletPreview(false);
                            }
                          })();
                        }}
                        type="button"
                      >
                        {isRefreshingWalletPreview
                          ? "Expanding..."
                          : getHomeCoverageActionLabel(preview)}
                      </button>
                    ) : null}
                  </div>

                  <p className="home-summary-copy">
                    {preview.indexing.status === "indexing"
                      ? "Background indexing is running."
                      : preview.indexing.lastIndexedAt
                        ? `Updated ${formatRelativeTime(preview.indexing.lastIndexedAt)}`
                        : "Ready"}
                  </p>
                  <p className="home-summary-copy">
                    {preview.indexing.coverageWindowDays > 0
                      ? `${preview.indexing.coverageWindowDays}d indexed`
                      : "Coverage warming up"}
                  </p>

                  <div className="preview-identity home-summary-grid">
                    <div>
                      <span>Chain</span>
                      <strong>{preview.chainLabel}</strong>
                    </div>
                    <div>
                      <span>Address</span>
                      <strong className="home-summary-address-value">
                        {preview.address}
                      </strong>
                    </div>
                    <div>
                      <span>Status</span>
                      <strong>
                        {preview.indexing.status === "indexing"
                          ? "Indexing"
                          : "Ready"}
                      </strong>
                    </div>
                  </div>

                  <div className="preview-scores home-score-grid">
                    {preview.scores.map((score) => (
                      <article key={score.name} className="score-row">
                        <div>
                          <span>{formatScoreLabel(score.name)}</span>
                          <strong>{score.value}</strong>
                        </div>
                        <Badge tone={score.tone}>{score.rating}</Badge>
                      </article>
                    ))}
                  </div>

                  {activeNodeEntityAssignments.length > 0 ? (
                    <div
                      className="detail-entity-linkage"
                      style={{ marginTop: 16 }}
                    >
                      <div className="detail-entity-linkage-head">
                        <div>
                          <span className="preview-kicker">Entity context</span>
                          <strong>
                            {activeNodeEntityAssignments.length} visible label
                            {activeNodeEntityAssignments.length === 1
                              ? ""
                              : "s"}
                          </strong>
                        </div>
                      </div>
                      <div className="detail-entity-linkage-strip">
                        {activeNodeEntityAssignments.map((assignment) => (
                          <div
                            key={`${assignment.entityNodeId}:${assignment.source}`}
                            className="detail-entity-link"
                          >
                            {assignment.entityHref ? (
                              <a
                                className="detail-inline-link"
                                href={assignment.entityHref}
                              >
                                {assignment.entityLabel}
                              </a>
                            ) : (
                              <span>{assignment.entityLabel}</span>
                            )}
                            <Badge tone="amber">entity</Badge>
                            <Badge tone={assignment.sourceTone}>
                              {assignment.sourceLabel}
                            </Badge>
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : null}

                  {preview.topCounterparties.length > 0 ? (
                    <div className="home-counterparty-stack">
                      <div
                        className="home-counterparty-head"
                        style={{ marginBottom: 12 }}
                      >
                        <strong
                          style={{ fontSize: "0.9rem", color: "var(--text)" }}
                        >
                          Top related
                        </strong>
                        <span
                          style={{ color: "var(--muted)", fontSize: "0.85rem" }}
                        >
                          {preview.counterparties > 0
                            ? `${Math.min(preview.topCounterparties.length, 3)} shown of ${preview.counterparties} indexed`
                            : `${preview.topCounterparties.length} visible`}
                        </span>
                      </div>

                      {preview.topCounterparties
                        .slice(0, 3)
                        .map((counterparty) => (
                          <div
                            key={`${counterparty.chain}:${counterparty.address}`}
                            className="home-counterparty-card"
                          >
                            <div>
                              <strong>
                                {compactAddress(counterparty.address)}
                              </strong>
                              <span>{counterparty.directionLabel}</span>
                              {counterparty.entityLabel ? (
                                <span>{counterparty.entityLabel}</span>
                              ) : null}
                            </div>
                            <Badge tone="teal">
                              {counterparty.interactionCount} hits
                            </Badge>
                          </div>
                        ))}
                    </div>
                  ) : null}
                </>
              ) : (
                <>
                  <div
                    className="preview-identity home-summary-grid"
                    style={{ marginTop: 24 }}
                  >
                    {activeNode.chain ? (
                      <div>
                        <span>Chain</span>
                        <strong>{activeNode.chain.toUpperCase()}</strong>
                      </div>
                    ) : null}
                    {activeNode.address ? (
                      <div>
                        <span>Address / Identifier</span>
                        <strong className="home-summary-address-value">
                          {activeNode.address}
                        </strong>
                      </div>
                    ) : null}
                  </div>

                  {activeNodeEntityAssignments.length > 0 ? (
                    <div
                      className="detail-entity-linkage"
                      style={{ marginTop: 16 }}
                    >
                      <div className="detail-entity-linkage-head">
                        <div>
                          <span className="preview-kicker">Entity context</span>
                          <strong>
                            {activeNodeEntityAssignments[0]?.entityLabel}
                          </strong>
                        </div>
                      </div>
                      <div className="detail-entity-linkage-strip">
                        {activeNodeEntityAssignments
                          .slice(0, 2)
                          .map((assignment) => (
                            <div
                              key={`${assignment.entityNodeId}:${assignment.source}`}
                              className="detail-entity-link"
                            >
                              <span>{assignment.entityLabel}</span>
                              <Badge tone={assignment.sourceTone}>
                                {assignment.sourceLabel}
                              </Badge>
                            </div>
                          ))}
                      </div>
                    </div>
                  ) : null}

                  {activeNodeRelationships.length > 0 ? (
                    <div
                      className="home-counterparty-stack"
                      style={{ marginTop: 16 }}
                    >
                      <div
                        className="home-counterparty-head"
                        style={{ marginBottom: 12 }}
                      >
                        <strong
                          style={{ fontSize: "0.9rem", color: "var(--text)" }}
                        >
                          Visible relationships
                        </strong>
                        <span
                          style={{ color: "var(--muted)", fontSize: "0.85rem" }}
                        >
                          {activeNodeRelationships.length} linked
                        </span>
                      </div>
                      {activeNodeRelationships.map((relationship) => (
                        <div
                          key={relationship.key}
                          className="home-counterparty-card"
                        >
                          <div>
                            <strong>
                              {relationship.sourceLabel} →{" "}
                              {relationship.targetLabel}
                            </strong>
                            <span>
                              {relationship.kindLabel} ·{" "}
                              {relationship.directionLabel}
                            </span>
                            <span>{relationship.familyLabel}</span>
                          </div>
                          <Badge tone="teal">{relationship.weight} hits</Badge>
                        </div>
                      ))}
                    </div>
                  ) : null}

                  <div className="home-side-actions">
                    <button
                      className="search-cta"
                      onClick={() => {
                        const href = buildSelectedGraphNodeHref(activeNode);
                        if (href) {
                          router.push(href);
                          return;
                        }
                        void runSearch(
                          activeNode.address ?? activeNode.label,
                          true,
                        );
                      }}
                      style={{ width: "100%", justifyContent: "center" }}
                      type="button"
                    >
                      {buildSelectedGraphNodeHrefLabel(activeNode)}
                    </button>
                  </div>
                </>
              )}
            </article>
          </aside>
        ) : null}
      </section>
    </main>
  );
}

export function buildHomeGraphExpansionKey(
  node: Pick<WalletGraphPreviewNode, "kind" | "chain" | "address" | "id">,
): string {
  if (node.kind === "wallet" && node.chain && node.address) {
    return `${node.chain}:${node.address.toLowerCase()}`;
  }

  return node.id;
}

export function mergeHomeGraphPreviews(
  current: WalletGraphPreview,
  expansion: WalletGraphPreview,
): WalletGraphPreview {
  const nodeMap = new Map<string, WalletGraphPreviewNode>();
  for (const node of [...current.nodes, ...expansion.nodes]) {
    if (!nodeMap.has(node.id)) {
      nodeMap.set(node.id, node);
    }
  }

  const edgeMap = new Map<string, WalletGraphPreviewEdge>();
  for (const edge of [...current.edges, ...expansion.edges]) {
    const key = buildWalletGraphEdgeKey(edge);
    if (!edgeMap.has(key)) {
      edgeMap.set(key, edge);
    }
  }

  const mergedNodes = [...nodeMap.values()];
  const mergedEdges = [...edgeMap.values()];

  return {
    ...current,
    mode:
      current.mode === "live" || expansion.mode === "live"
        ? "live"
        : current.mode,
    source: current.source === "live-api" ? current.source : expansion.source,
    depthResolved: Math.max(current.depthResolved, expansion.depthResolved),
    densityCapped: current.densityCapped || expansion.densityCapped,
    statusMessage:
      expansion.mode === "live"
        ? expansion.statusMessage
        : current.statusMessage,
    neighborhoodSummary: {
      neighborNodeCount: Math.max(mergedNodes.length - 1, 0),
      walletNodeCount: mergedNodes.filter((node) => node.kind === "wallet")
        .length,
      clusterNodeCount: mergedNodes.filter((node) => node.kind === "cluster")
        .length,
      entityNodeCount: mergedNodes.filter((node) => node.kind === "entity")
        .length,
      interactionEdgeCount: mergedEdges.length,
      totalInteractionWeight: mergedEdges.reduce(
        (sum, edge) => sum + (edge.weight ?? edge.counterpartyCount ?? 1),
        0,
      ),
      ...(() => {
        const latestObservedAt = [
          current.neighborhoodSummary.latestObservedAt,
          expansion.neighborhoodSummary.latestObservedAt,
          ...mergedEdges.map((edge) => edge.observedAt),
        ]
          .filter((value): value is string => Boolean(value))
          .sort()
          .at(-1);

        return latestObservedAt ? { latestObservedAt } : {};
      })(),
    },
    nodes: mergedNodes,
    edges: mergedEdges,
  };
}

function rebaseExpandedGraphRootNode(
  graph: WalletGraphPreview,
  nextRootNodeId: string,
): WalletGraphPreview {
  if (!graph.nodes.some((node) => node.id === "wallet_root")) {
    return graph;
  }

  return {
    ...graph,
    nodes: graph.nodes.map((node) =>
      node.id === "wallet_root" ? { ...node, id: nextRootNodeId } : node,
    ),
    edges: graph.edges.map((edge) => ({
      ...edge,
      sourceId:
        edge.sourceId === "wallet_root" ? nextRootNodeId : edge.sourceId,
      targetId:
        edge.targetId === "wallet_root" ? nextRootNodeId : edge.targetId,
    })),
  };
}

type HomeGraphRelationship = {
  key: string;
  sourceId: string;
  targetId: string;
  sourceLabel: string;
  targetLabel: string;
  kindLabel: string;
  directionLabel: string;
  familyLabel: string;
  observedAt?: string | undefined;
  weight: number;
};

function buildHomeGraphRelationships(
  graphPreview: WalletGraphPreview,
): HomeGraphRelationship[] {
  return graphPreview.edges
    .map((edge) => ({
      key: `${edge.sourceId}:${edge.targetId}:${edge.kind}:${edge.observedAt ?? ""}`,
      sourceId: edge.sourceId,
      targetId: edge.targetId,
      sourceLabel:
        graphPreview.nodes.find((node) => node.id === edge.sourceId)?.label ??
        edge.sourceId,
      targetLabel:
        graphPreview.nodes.find((node) => node.id === edge.targetId)?.label ??
        edge.targetId,
      kindLabel: getWalletGraphEdgeKindLabel(edge.kind),
      directionLabel: describeGraphRelationshipDirection(edge),
      familyLabel: getWalletGraphEdgeFamilyLabel(edge.family),
      observedAt: edge.observedAt,
      weight: edge.weight ?? edge.counterpartyCount ?? 0,
    }))
    .sort((left, right) => right.weight - left.weight);
}

function compactAddress(value: string): string {
  if (value.length <= 18) {
    return value;
  }

  return `${value.slice(0, 8)}...${value.slice(-6)}`;
}

function formatRelativeTime(value: string): string {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return "just now";
  }

  const deltaSeconds = Math.max(0, Math.floor((Date.now() - parsed) / 1000));
  if (deltaSeconds < 45) {
    return "just now";
  }
  if (deltaSeconds < 3600) {
    return `${Math.floor(deltaSeconds / 60)}m ago`;
  }
  if (deltaSeconds < 86400) {
    return `${Math.floor(deltaSeconds / 3600)}h ago`;
  }
  if (deltaSeconds < 86400 * 14) {
    return `${Math.floor(deltaSeconds / 86400)}d ago`;
  }

  return new Date(parsed).toISOString().slice(0, 10);
}

function formatScoreLabel(name: string): string {
  return name.replaceAll("_", " ");
}

function formatFindingTypeLabel(type: string): string {
  return type.replaceAll("_", " ");
}

function formatFindingSubjectTypeLabel(type: string): string {
  return type.replaceAll("_", " ");
}

function resolveFindingNextWatchLabel(item: FindingPreview): string {
  const nextWatch = item.nextWatch[0];
  if (!nextWatch) {
    return item.subjectType === "entity"
      ? "Open entity context"
      : item.subjectType === "wallet"
        ? "Open wallet brief"
        : "Open finding context";
  }

  if (nextWatch.subjectType === "wallet") {
    return nextWatch.label?.trim()
      ? `Watch ${nextWatch.label.trim()}`
      : nextWatch.address?.trim()
        ? `Watch ${compactAddress(nextWatch.address.trim())}`
        : "Watch counterparty wallet";
  }

  if (nextWatch.subjectType === "entity") {
    return nextWatch.label?.trim()
      ? `Open ${nextWatch.label.trim()}`
      : "Open entity context";
  }

  if (nextWatch.subjectType === "token") {
    return nextWatch.token?.trim()
      ? `Watch token ${nextWatch.token.trim()}`
      : "Watch token context";
  }

  return nextWatch.label?.trim()
    ? `Watch ${nextWatch.label.trim()}`
    : "Watch next context";
}

function resolveFindingNextWatchHref(item: FindingPreview): string | null {
  const nextWatch = item.nextWatch[0];
  if (!nextWatch) {
    return resolveFindingSubjectHref(item);
  }

  if (
    nextWatch.subjectType === "wallet" &&
    nextWatch.chain &&
    nextWatch.address
  ) {
    return buildWalletDetailHref({
      chain: nextWatch.chain as "evm" | "solana",
      address: nextWatch.address,
    });
  }
  if (nextWatch.subjectType === "token" && nextWatch.token?.trim()) {
    return buildProductSearchHref(nextWatch.token.trim());
  }
  if (nextWatch.label?.trim()) {
    return buildProductSearchHref(nextWatch.label.trim());
  }

  return null;
}

function resolveFindingAnalystEntryLabel(item: FindingPreview): string {
  if (item.subjectType === "wallet") {
    return "Analyze wallet";
  }
  if (item.subjectType === "entity") {
    return "Analyze entity";
  }
  if (item.subjectType === "token") {
    return "Analyze token";
  }
  return "Analyze finding";
}

function resolveFindingAnalystEntryHref(item: FindingPreview): string | null {
  return resolveFindingSubjectHref(item);
}

function resolveFindingSubjectLabel(item: FindingPreview): string {
  if (item.label?.trim()) {
    return item.label.trim();
  }
  if (item.address?.trim()) {
    return compactAddress(item.address);
  }
  if (item.key?.trim()) {
    return item.key.trim();
  }
  return "Finding";
}

function resolveFindingSubjectHref(item: FindingPreview): string | null {
  if (item.subjectType === "wallet" && item.chain && item.address) {
    return buildWalletDetailHref({
      chain: item.chain as "evm" | "solana",
      address: item.address,
    });
  }
  if (item.subjectType === "entity" && item.key) {
    return buildEntityDetailHref(item.key);
  }
  if (item.label?.trim()) {
    return buildProductSearchHref(item.label.trim());
  }
  return null;
}

function toneForFindingType(type: string): HomeFindingFeedItem["badgeTone"] {
  if (type.includes("exit") || type.includes("pressure")) {
    return "amber";
  }
  if (type.includes("convergence") || type.includes("entry")) {
    return "emerald";
  }
  if (type.includes("rotation")) {
    return "violet";
  }
  return "teal";
}
