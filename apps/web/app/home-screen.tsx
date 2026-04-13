"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Badge } from "@qorvi/ui";

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
  getAnalystFindingsPreview,
  getSearchPreview,
  loadAnalystFindingsPreview,
  loadSearchPreview,
  resolveWalletSummaryRequestFromRoute,
  shouldPollIndexedWalletSummary,
} from "../lib/api-boundary";
import { persistClientForwardedAuthHeaders } from "../lib/request-headers";

import { useTranslation } from "../lib/i18n/provider";
import { AuthButtons } from "./components/auth-buttons";
import { LanguageSwitcher } from "./components/language-switcher";
import { NetworkBackground } from "./components/network-background";
import { describeGraphRelationshipDirection } from "./wallets/[chain]/[address]/wallet-graph-presenter";
import {
  buildWalletGraphEdgeKey,
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
  t: (key: string) => string,
): HomeFindingFeedItem[] {
  const items: HomeFindingFeedItem[] = [];

  for (const score of preview.scores.slice(0, 2)) {
    const title = formatScoreLabel(score.name);
    const clusterBreakdown =
      score.name === "cluster_score" ? score.clusterBreakdown : undefined;
    items.push({
      id: `score:${score.name}`,
      title,
      findingTypeLabel: "Signal interpretation",
      summary: clusterBreakdown
        ? `${title} is elevated with ${clusterBreakdown.peerWalletOverlap} peer overlaps, ${clusterBreakdown.sharedEntityLinks} shared entity links, and ${clusterBreakdown.bidirectionalPeerFlows} bidirectional peer flows in the current coverage window.`
        : score.rating === "high"
          ? `${title} is elevated and worth reviewing first.`
          : `${title} is active in the current indexed coverage window.`,
      evidenceLabel: clusterBreakdown
        ? `Derived from wallet score ${score.value}/100 · ${clusterBreakdown.peerWalletOverlap} peer overlaps · ${clusterBreakdown.sharedEntityLinks} shared entity links`
        : `Derived from wallet score ${score.value}/100`,
      nextWatchLabel: t("home.feedItem.nextWatch"),
      nextWatchHref: walletDetailHref,
      analystEntryLabel: t("home.feedItem.analyzeWallet"),
      analystEntryHref: walletDetailHref,
      importance: score.value / 100,
      confidence: score.value >= 90 ? 0.9 : 0.65,
      subjectLabel: preview.label,
      subjectHref: walletDetailHref,
      subjectTypeLabel: t("home.feedItem.subjectType"),
      badgeTone: score.rating === "high" ? "emerald" : "amber",
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
      nextWatchLabel: t("home.feedItem.nextWatch"),
      nextWatchHref: walletDetailHref,
      analystEntryLabel: t("home.feedItem.analyzeWallet"),
      analystEntryHref: walletDetailHref,
      importance: signal.value / 100,
      confidence: signal.value / 100,
      subjectLabel: preview.label,
      subjectHref: walletDetailHref,
      subjectTypeLabel: t("home.feedItem.subjectType"),
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
      nextWatchLabel: t("home.feedItem.nextWatch"),
      nextWatchHref: topCounterparty.entityLabel
        ? buildProductSearchHref(topCounterparty.entityLabel)
        : buildWalletDetailHref({
            chain: topCounterparty.chain,
            address: topCounterparty.address,
          }),
      analystEntryLabel: t("home.feedItem.analyzeWallet"),
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
      subjectTypeLabel: topCounterparty.entityLabel
        ? "Entity"
        : t("home.feedItem.subjectType"),
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
  const { t } = useTranslation();
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryFromUrl = searchParams.get("q")?.trim() ?? "";
  const lastHydratedUrlQuery = useRef<string | null>(queryFromUrl);
  const queryRef = useRef("");
  const [query, setQuery] = useState("");
  const [searchPreview, setSearchPreview] = useState(() => getSearchPreview());
  const [pendingWalletDetail, setPendingWalletDetail] =
    useState<WalletSummaryRequest | null>(null);
  const [findingsFeedPreview, setFindingsFeedPreview] = useState(() =>
    getAnalystFindingsPreview(),
  );
  const walletRequestForDetail =
    resolveWalletRequestFromSearchPreview(searchPreview);
  const walletDetailHref = walletRequestForDetail
    ? buildWalletDetailHref(walletRequestForDetail)
    : null;
  const findingsFeedItems = useMemo(
    () => buildHomeFindingsFeedItemsFromFeed(findingsFeedPreview),
    [findingsFeedPreview],
  );
  const hasSearchFeedback =
    query.length > 0 && searchPreview.query.length > 0 && !walletDetailHref;

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

  useEffect(() => {
    queryRef.current = query;
  }, [query]);

  const runSearch = useCallback(
    async (nextQuery: string, syncUrl = false) => {
      const trimmed = nextQuery.trim();
      const immediateSearchPreview = getSearchPreview(trimmed);
      const immediateWalletRequest = resolveWalletRequestFromSearchPreview(
        immediateSearchPreview,
      );
      setQuery(trimmed);
      setSearchPreview(immediateSearchPreview);
      setPendingWalletDetail(immediateWalletRequest);

      const nextSearchPreview = await loadSearchPreview({ query: trimmed });
      const nextWalletRequest =
        resolveWalletRequestFromSearchPreview(nextSearchPreview);
      const nextWalletDetailHref = nextWalletRequest
        ? buildWalletDetailHref(nextWalletRequest)
        : null;

      setSearchPreview(nextSearchPreview);
      setPendingWalletDetail(nextWalletRequest);

      if (nextWalletDetailHref) {
        router.push(nextWalletDetailHref);
        return;
      }

      setPendingWalletDetail(null);

      if (syncUrl) {
        lastHydratedUrlQuery.current = trimmed;
        router.replace(trimmed ? buildProductSearchHref(trimmed) : "/", {
          scroll: false,
        });
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
          <nav className="discover-nav">
            <a href="/discover" className="discover-nav-link">
              Discover
            </a>
            <a href="/signals/shadow-exits" className="discover-nav-link">
              Signals
            </a>
            <a href="/alerts" className="discover-nav-link">
              Alerts
            </a>
          </nav>
        </div>
        <div
          style={{
            marginLeft: "auto",
            display: "flex",
            alignItems: "center",
            gap: "12px",
          }}
        >
          <LanguageSwitcher />
          <AuthButtons />
        </div>
      </header>

      <section className="home-fullscreen-body">
        <div className="home-fullscreen-canvas">
          <div className="graph-empty-state home-discover-shell">
            <div className="graph-empty-content home-discover-hero">
              <strong style={{ fontSize: "2.5rem", marginBottom: "8px" }}>
                {t("hero.title")}
              </strong>
              <p style={{ fontSize: "1.2rem", marginBottom: "24px" }}>
                {t("hero.subtitle")}
              </p>
              <div
                className="search-bar"
                style={{ width: "100%", display: "flex", gap: "8px" }}
              >
                <input
                  id="wallet-search-hero"
                  value={query}
                  onChange={(event) => {
                    queryRef.current = event.currentTarget.value;
                    setQuery(event.currentTarget.value);
                  }}
                  onKeyDown={(event) => {
                    if (event.key === "Enter") {
                      event.preventDefault();
                      void runSearch(queryRef.current, true);
                    }
                  }}
                  placeholder={t("hero.searchPlaceholder")}
                  aria-label={t("hero.searchPlaceholder")}
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
                  aria-label="Search wallet address"
                  onClick={() => {
                    void runSearch(queryRef.current, true);
                  }}
                  type="button"
                  style={{
                    borderRadius: "40px",
                    padding: "0 32px",
                    fontSize: "1.1rem",
                    fontWeight: 500,
                  }}
                >
                  {pendingWalletDetail ? "Opening..." : t("hero.searchButton")}
                </button>
              </div>

              {pendingWalletDetail ? (
                <article className="preview-card home-summary-card home-discover-feedback">
                  <div className="preview-header">
                    <div className="home-side-header">
                      <h2
                        style={{
                          fontSize: "1.05rem",
                          fontWeight: 600,
                          margin: 0,
                        }}
                      >
                        Opening wallet detail
                      </h2>
                      <p
                        style={{
                          margin: "4px 0 0",
                          color: "var(--muted)",
                          fontSize: "0.85rem",
                        }}
                      >
                        {pendingWalletDetail.chain === "solana"
                          ? "Solana"
                          : "EVM"}{" "}
                        wallet
                      </p>
                    </div>
                  </div>
                  <p
                    className="home-summary-copy"
                    style={{
                      marginBottom: 10,
                      fontFamily:
                        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, Courier New, monospace",
                    }}
                  >
                    {pendingWalletDetail.address}
                  </p>
                  <p className="home-summary-copy">
                    Preparing the first indexed view, AI brief, and graph
                    evidence for this wallet.
                  </p>
                </article>
              ) : null}

              {hasSearchFeedback ? (
                <article className="preview-card home-summary-card home-discover-feedback">
                  <div className="preview-header">
                    <div className="home-side-header">
                      <h2
                        style={{
                          fontSize: "1.05rem",
                          fontWeight: 600,
                          margin: 0,
                        }}
                      >
                        {searchPreview.title}
                      </h2>
                      <p
                        style={{
                          margin: "4px 0 0",
                          color: "var(--muted)",
                          fontSize: "0.85rem",
                        }}
                      >
                        {searchPreview.kindLabel}
                        {searchPreview.chainLabel
                          ? ` · ${searchPreview.chainLabel}`
                          : ""}
                      </p>
                    </div>
                  </div>
                  <p className="home-summary-copy">
                    {searchPreview.explanation}
                  </p>
                </article>
              ) : null}

              {findingsFeedItems.length > 0 ? (
                <article className="preview-card home-summary-card home-discover-feed">
                  <div className="preview-header">
                    <div className="home-side-header">
                      <h2
                        style={{
                          fontSize: "1.2rem",
                          fontWeight: 600,
                          margin: 0,
                        }}
                      >
                        {t("home.feedTitle")}
                      </h2>
                      <p
                        style={{
                          margin: "4px 0 0",
                          color: "var(--muted)",
                          fontSize: "0.85rem",
                        }}
                      >
                        {t("home.feedSubtitle")}
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
                          <div className="item-scores">
                            <div className="score-ring">
                              <dt>{t("home.feedItem.importance")}</dt>
                              <dd style={{ color: "rgb(250, 250, 250)" }}>
                                {Math.round(item.importance * 100)}%
                              </dd>
                            </div>
                            <div className="score-ring">
                              <dt>{t("home.feedItem.confidence")}</dt>
                              <dd style={{ color: "rgba(255, 255, 255, 0.6)" }}>
                                {Math.round(item.confidence * 100)}%
                              </dd>
                            </div>
                          </div>
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
              ) : null}
            </div>
          </div>
        </div>
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
  if (item.subjectType === "wallet" && item.chain && item.address) {
    const detailHref = buildWalletDetailHref({
      chain: item.chain as "evm" | "solana",
      address: item.address,
    });
    const question = encodeURIComponent(
      `Explain the ${item.type.replaceAll("_", " ")} finding for this wallet.`,
    );
    return `${detailHref}?ask=${question}`;
  }

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
