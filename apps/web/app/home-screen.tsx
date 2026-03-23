"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Badge } from "@whalegraph/ui";

import {
  type SearchPreview,
  type WalletGraphPreview,
  type WalletGraphPreviewNode,
  type WalletSummaryPreview,
  type WalletSummaryRequest,
  buildProductSearchHref,
  buildWalletDetailHref,
  deriveWalletGraphPreviewFromSummary,
  getSearchPreview,
  getWalletGraphPreview,
  getWalletSummaryPreview,
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
  formatGraphKind,
  getWalletGraphEdgeFamilyLabel,
  getWalletGraphEdgeKindLabel,
} from "./wallets/[chain]/[address]/wallet-graph-visual-model";

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
  const [walletRequest, setWalletRequest] =
    useState<WalletSummaryRequest | null>(null);
  const [preview, setPreview] = useState(() =>
    getWalletSummaryPreview({ chain: "evm", address: "" }),
  );
  const [graphPreview, setGraphPreview] = useState(() =>
    getWalletGraphPreview({ chain: "evm", address: "", depthRequested: 2 }),
  );
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const selectedNode = useMemo(() => {
    return (
      graphPreview.nodes.find((node) => node.id === selectedNodeId) ?? null
    );
  }, [graphPreview.nodes, selectedNodeId]);
  const activeNode = selectedNode ?? graphPreview.nodes[0] ?? null;
  const [isRefreshingWalletPreview, setIsRefreshingWalletPreview] =
    useState(false);
  const walletRequestForDetail =
    resolveWalletRequestFromSearchPreview(searchPreview);
  const walletDetailHref = walletRequestForDetail
    ? buildWalletDetailHref(walletRequestForDetail)
    : null;

  useEffect(() => {
    persistClientForwardedAuthHeaders(requestHeaders);
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
              depthRequested: 2,
            },
            fallback: graphFallback,
          }
          : {
            request: {
              ...walletRequest,
              depthRequested: 2,
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
              depthRequested: 2,
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
      return;
    }

    setSelectedNodeId((current) => {
      if (current && graphPreview.nodes.some((node) => node.id === current)) {
        return current;
      }
      return graphPreview.nodes[0]?.id ?? null;
    });
  }, [graphPreview.nodes]);

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
            Wallet graph
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
        <div className="home-fullscreen-canvas">
          <div
            className="preview-header home-fullscreen-canvas-overlay"
            style={{
              background: "transparent",
              boxShadow: "none",
              border: "none",
            }}
          >

          </div>

          {hasWalletPreview ? (
            <WalletGraphVisual
              densityCapped={graphPreview.densityCapped}
              nodes={graphPreview.nodes}
              edges={graphPreview.edges}
              neighborhoodSummary={graphPreview.neighborhoodSummary}
              variant="hero"
              selectedNodeId={selectedNodeId}
              onSelectedNodeIdChange={setSelectedNodeId}
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
                          ? "Refreshing..."
                          : "Refresh Intelligence"}
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
                          {preview.topCounterparties.length} visible
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
