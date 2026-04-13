"use client";

import { useSearchParams } from "next/navigation";
import {
  Fragment,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { Badge, type Tone } from "@qorvi/ui";

import {
  type AnalystMemoryTurn,
  buildWalletAnalystMemoryScopeKey,
  readAnalystMemory,
  writeAnalystMemory,
} from "../../../../lib/analyst-memory";
import type {
  AnalystWalletAnalyzeEvidenceRefPreview,
  AnalystWalletAnalyzePreview,
  AnalystWalletAnalyzeRecentTurnInput,
  ClusterDetailPreview,
  WalletBriefPreview,
  WalletDetailRequest,
  WalletGraphPreview,
  WalletGraphPreviewEdge,
  WalletGraphPreviewNode,
  WalletSummaryClusterScoreBreakdownPreview,
  WalletSummaryPreview,
} from "../../../../lib/api-boundary";
import {
  analyzeAnalystWallet,
  buildClusterDetailHref,
  buildProductSearchHref,
  buildWalletDetailHref,
  deriveWalletGraphPreviewFromSummary,
  loadAnalystWalletBriefPreview,
  loadClusterDetailPreview,
  loadSearchPreview,
  loadWalletGraphPreview,
  loadWalletSummaryPreview,
  shouldPollIndexedWalletSummary,
  trackWalletAlertRule,
} from "../../../../lib/api-boundary";
import { useClerkRequestHeaders } from "../../../../lib/clerk-client-auth";
import { useTranslation } from "../../../../lib/i18n/provider";
import { persistClientForwardedAuthHeaders } from "../../../../lib/request-headers";
import { LanguageSwitcher } from "../../../components/language-switcher";
import { PageShell } from "../../../components/page-shell";
import {
  type GraphEntityAssignmentPresentation,
  buildCounterpartyEntityAssignment,
  buildFallbackEntityAssignment,
  buildGraphEntityAssignmentIndex,
  buildSelectedGraphNodeHref,
  buildSelectedGraphNodeHrefLabel,
  buildWalletGraphAvailabilityPresentation,
  buildWalletSummaryAvailabilityPresentation,
  describeGraphRelationshipDirection,
  formatEntityAssignmentSource,
  formatGraphSnapshotSource,
  mergeEntityAssignments,
  toneForEntityAssignmentSource,
} from "./wallet-graph-presenter";
import { WalletGraphVisual } from "./wallet-graph-visual";
import {
  buildWalletGraphEdgeKey,
  getWalletGraphEdgeFamilyLabel,
  getWalletGraphEdgeKindLabel,
} from "./wallet-graph-visual-model";

export { buildGraphEntityAssignmentIndex };

const scoreToneByName: Record<string, Tone> = {
  cluster_score: "emerald",
  shadow_exit_risk: "amber",
};

export type WalletDetailViewModel = {
  title: string;
  chainLabel: string;
  address: string;
  aiBrief: WalletBriefViewModel;
  summaryRoute: string;
  summaryStatus: string;
  summaryModeLabel: string;
  summarySourceLabel: string;
  clusterDetailHref: string | null;
  graphRoute: string;
  graphStatus: string;
  graphModeLabel: string;
  graphSourceLabel: string;
  graphSnapshotSourceLabel: string;
  graphSnapshotGeneratedAt: string | null;
  backHref: string;
  summaryScores: Array<{
    name: string;
    value: number;
    rating: string;
    tone: Tone;
    clusterBreakdown?: WalletSummaryClusterScoreBreakdownPreview;
  }>;
  latestSignals: WalletLatestSignalViewModel[];
  indexing: WalletIndexingViewModel;
  enrichment: WalletEnrichmentViewModel | null;
  relatedAddresses: WalletRelatedAddressViewModel[];
  relatedAddressCountAvailable: number;
  relatedAddressCountShown: number;
  relatedAddressCountLabel: string;
  recentFlow: WalletRecentFlowViewModel;
  graphNodeCount: number;
  graphEdgeCount: number;
  graphNodes: WalletGraphNodeViewModel[];
  graphEdges: WalletGraphEdgeViewModel[];
  graphRelationships: WalletGraphRelationshipViewModel[];
};

export type WalletBriefViewModel = {
  headline: string;
  summary: string;
  keyFindings: string[];
  evidence: string[];
  nextWatch: string[];
};

export type WalletGraphNodeViewModel = WalletGraphPreviewNode & {
  tone: Tone;
  kindLabel: string;
  isPrimary: boolean;
};

export type WalletGraphEntityLinkViewModel = {
  id: string;
  label: string;
  kindLabel: string;
  tone: Tone;
  href: string | null;
  sourceLabel?: string;
  sourceTone?: Tone;
};

export type WalletGraphEntityContextViewModel = {
  label: string;
  helperCopy: string;
  links: WalletGraphEntityLinkViewModel[];
};

export type WalletGraphEntityAssignmentViewModel =
  GraphEntityAssignmentPresentation;

export type WalletGraphEdgeViewModel = WalletGraphPreviewEdge & {
  sourceLabel: string;
  targetLabel: string;
  kindLabel: string;
};

export type WalletGraphRelationshipViewModel = {
  key: string;
  sourceLabel: string;
  targetLabel: string;
  kindLabel: string;
  directionLabel: string;
  family: WalletGraphPreviewEdge["family"];
  familyLabel: string;
  confidence: string;
  evidenceSummary: string;
  evidenceSource: string;
  lastTxHash: string;
  lastDirection: string;
  lastProvider: string;
  observedAt?: string | undefined;
  weight: number;
  primaryToken: string;
  inboundCount: number;
  outboundCount: number;
  inboundAmount: string;
  outboundAmount: string;
  tokenBreakdowns: WalletRelatedAddressTokenBreakdownViewModel[];
  href: string | null;
};

export type WalletRelatedAddressViewModel = {
  chainLabel: string;
  address: string;
  entityKey: string;
  entityType: string;
  entityLabel: string;
  interactionCount: number;
  inboundCount: number;
  outboundCount: number;
  inboundAmount: string;
  outboundAmount: string;
  primaryToken: string;
  tokenBreakdowns: WalletRelatedAddressTokenBreakdownViewModel[];
  tokenBreakdownCount: number;
  directionLabel: string;
  firstSeenAt: string;
  latestActivityAt: string;
  href: string;
};

export type WalletRelatedAddressTokenBreakdownViewModel = {
  symbol: string;
  inboundAmount: string;
  outboundAmount: string;
};

export type WalletRecentFlowViewModel = {
  incomingTxCount7d: number;
  outgoingTxCount7d: number;
  incomingTxCount30d: number;
  outgoingTxCount30d: number;
  netDirection7d: string;
  netDirection30d: string;
};

export type WalletEnrichmentViewModel = {
  provider: string;
  netWorthUsd: string;
  nativeBalanceFormatted: string;
  activeChains: string[];
  activeChainCount: number;
  holdings: WalletHoldingViewModel[];
  holdingCount: number;
  source: string;
  updatedAt: string;
};

export type WalletHoldingViewModel = {
  symbol: string;
  tokenAddress: string;
  balanceFormatted: string;
  valueUsd: string;
  portfolioPercentage: number;
  isNative: boolean;
};

export type WalletIndexingViewModel = {
  status: "ready" | "indexing";
  statusLabel: string;
  actionLabel: string;
  helperCopy: string;
  lastIndexedAt: string;
  coverageStartAt: string;
  coverageEndAt: string;
  coverageWindowLabel: string;
};

export type WalletLatestSignalViewModel = {
  name: string;
  label: string;
  rating: string;
  value: number;
  source: string;
  observedAt: string;
};

export type WalletRelatedAddressDirectionFilter =
  | "all"
  | "inbound"
  | "outbound"
  | "mixed";

export type WalletRelatedAddressSortKey =
  | "interaction"
  | "latest_activity"
  | "first_seen"
  | "total_volume"
  | "outbound_volume"
  | "inbound_volume";

const MAX_GRAPH_NODE_BUDGET = 120;
const DEFAULT_WALLET_GRAPH_DEPTH = 3;

export type WalletGraphExpansionState = {
  canExpand: boolean;
  expansionKey: string | null;
  reason: string;
  budgetLabel: string;
  hopsUsed: number;
  hopBudget: number;
  nodeCount: number;
  nodeBudget: number;
};

export function buildWalletDetailViewModel({
  request,
  summary,
  graph,
  brief,
  t,
}: {
  request: WalletDetailRequest;
  summary: WalletSummaryPreview;
  graph: WalletGraphPreview;
  brief?: WalletBriefPreview;
  t: (key: string) => string;
}): WalletDetailViewModel {
  const summaryAvailability =
    buildWalletSummaryAvailabilityPresentation(summary);
  const graphAvailability = buildWalletGraphAvailabilityPresentation(graph);
  const aiBrief = buildWalletBriefViewModel(summary, brief);

  return {
    title: summary.label,
    chainLabel: summary.chainLabel,
    address: request.address,
    aiBrief,
    summaryRoute: summary.route,
    summaryStatus: summary.statusMessage,
    summaryModeLabel: summaryAvailability.modeLabel,
    summarySourceLabel: summaryAvailability.sourceLabel,
    clusterDetailHref: summary.clusterId
      ? buildClusterDetailHref({ clusterId: summary.clusterId })
      : null,
    graphRoute: graph.route,
    graphStatus: graph.statusMessage,
    graphModeLabel: graphAvailability.modeLabel,
    graphSourceLabel: graphAvailability.sourceLabel,
    graphSnapshotSourceLabel: formatGraphSnapshotSource(graph.snapshot?.source),
    graphSnapshotGeneratedAt: graph.snapshot?.generatedAt ?? null,
    backHref: "/",
    summaryScores: summary.scores.map((score) => ({
      name: score.name,
      value: score.value,
      rating: score.rating,
      tone: scoreToneByName[score.name] ?? score.tone,
      ...(score.clusterBreakdown
        ? { clusterBreakdown: score.clusterBreakdown }
        : {}),
    })),
    latestSignals: summary.latestSignals.map((signal) => ({
      name: signal.name,
      label: signal.label,
      rating: signal.rating,
      value: signal.value,
      source: signal.source,
      observedAt: signal.observedAt,
    })),
    indexing: {
      status: summary.indexing.status,
      statusLabel:
        summary.indexing.status === "indexing"
          ? t("walletDetail.labels.indexing")
          : t("walletDetail.labels.coverageReady"),
      actionLabel:
        summary.indexing.status === "indexing"
          ? t("walletDetail.labels.continueIndexing")
          : t("walletDetail.labels.expandCoverage"),
      helperCopy:
        summary.indexing.status === "indexing"
          ? "Fresh counterparties and flows are still being collected. This panel refreshes automatically."
          : "The current coverage window is indexed and ready to inspect.",
      lastIndexedAt: summary.indexing.lastIndexedAt,
      coverageStartAt: summary.indexing.coverageStartAt,
      coverageEndAt: summary.indexing.coverageEndAt,
      coverageWindowLabel: formatCoverageWindow(summary.indexing),
    },
    enrichment: summary.enrichment
      ? {
          provider: summary.enrichment.provider,
          netWorthUsd: summary.enrichment.netWorthUsd,
          nativeBalanceFormatted: summary.enrichment.nativeBalanceFormatted,
          activeChains: [...summary.enrichment.activeChains],
          activeChainCount: summary.enrichment.activeChainCount,
          holdings: summary.enrichment.holdings.map((holding) => ({
            symbol: holding.symbol,
            tokenAddress: holding.tokenAddress,
            balanceFormatted: holding.balanceFormatted,
            valueUsd: holding.valueUsd,
            portfolioPercentage: holding.portfolioPercentage,
            isNative: holding.isNative,
          })),
          holdingCount: summary.enrichment.holdingCount,
          source: summary.enrichment.source,
          updatedAt: summary.enrichment.updatedAt,
        }
      : null,
    relatedAddresses: summary.topCounterparties.map((counterparty) => ({
      chainLabel: counterparty.chainLabel,
      address: counterparty.address,
      entityKey: counterparty.entityKey ?? "",
      entityType: counterparty.entityType ?? "",
      entityLabel: counterparty.entityLabel ?? "",
      interactionCount: counterparty.interactionCount,
      inboundCount: counterparty.inboundCount,
      outboundCount: counterparty.outboundCount,
      inboundAmount: counterparty.inboundAmount,
      outboundAmount: counterparty.outboundAmount,
      primaryToken: counterparty.primaryToken,
      tokenBreakdowns: counterparty.tokenBreakdowns.map((token) => ({
        symbol: token.symbol,
        inboundAmount: token.inboundAmount,
        outboundAmount: token.outboundAmount,
      })),
      tokenBreakdownCount: counterparty.tokenBreakdowns.length,
      directionLabel: counterparty.directionLabel,
      firstSeenAt: counterparty.firstSeenAt,
      latestActivityAt: counterparty.latestActivityAt,
      href: buildWalletDetailHref({
        chain: counterparty.chain,
        address: counterparty.address,
      }),
    })),
    relatedAddressCountAvailable: summary.counterparties,
    relatedAddressCountShown: summary.topCounterparties.length,
    relatedAddressCountLabel: formatRelatedAddressCoverageLabel(
      summary.topCounterparties.length,
      summary.counterparties,
    ),
    recentFlow: {
      incomingTxCount7d: summary.recentFlow.incomingTxCount7d,
      outgoingTxCount7d: summary.recentFlow.outgoingTxCount7d,
      incomingTxCount30d: summary.recentFlow.incomingTxCount30d,
      outgoingTxCount30d: summary.recentFlow.outgoingTxCount30d,
      netDirection7d: summary.recentFlow.netDirection7d,
      netDirection30d: summary.recentFlow.netDirection30d,
    },
    graphNodeCount: graph.nodes.length,
    graphEdgeCount: graph.edges.length,
    graphNodes: graph.nodes.map((node, index) => ({
      ...node,
      kindLabel: formatGraphKind(node.kind),
      tone: graphToneByKind[node.kind] ?? "teal",
      isPrimary: index === 0 || node.kind === "wallet",
    })),
    graphEdges: graph.edges.map((edge) => ({
      ...edge,
      sourceLabel: resolveGraphNodeLabel(graph.nodes, edge.sourceId),
      targetLabel: resolveGraphNodeLabel(graph.nodes, edge.targetId),
      kindLabel: formatGraphKind(edge.kind),
    })),
    graphRelationships: buildGraphRelationships(graph),
  };
}

function buildWalletBriefViewModel(
  summary: WalletSummaryPreview,
  brief?: WalletBriefPreview,
): WalletBriefViewModel {
  if (brief && brief.mode === "live") {
    const evidence = brief.keyFindings
      .flatMap((finding) => finding.observedFacts)
      .filter(Boolean)
      .slice(0, 3);
    const nextWatch = brief.keyFindings
      .flatMap((finding) =>
        finding.nextWatch.map((item) => {
          if (item.label) {
            return item.label;
          }
          if (item.token) {
            return item.token;
          }
          if (item.address) {
            return compactAddress(item.address);
          }
          return item.subjectType;
        }),
      )
      .filter(Boolean)
      .slice(0, 3);

    return {
      headline: `${brief.displayName} AI brief`,
      summary: brief.aiSummary,
      keyFindings: brief.keyFindings
        .map((finding) => finding.summary)
        .slice(0, 4),
      evidence:
        evidence.length > 0
          ? evidence
          : ["Evidence is still being assembled for this wallet."],
      nextWatch:
        nextWatch.length > 0
          ? nextWatch
          : ["Watch the top counterparties and linked entities next."],
    };
  }

  const primarySignal = summary.latestSignals[0];
  const primaryCounterparty = summary.topCounterparties[0];
  const leadingScore = summary.scores[0];
  const clusterBreakdown = findPrimaryClusterBreakdown(summary);

  const headline = primarySignal?.label
    ? `${summary.label} is showing ${primarySignal.label}.`
    : primaryCounterparty?.entityLabel
      ? `${summary.label} is tied closely to ${primaryCounterparty.entityLabel}.`
      : `${summary.label} has indexed activity ready for review.`;

  const summaryLine = [
    summary.counterparties > 0
      ? `${summary.counterparties} indexed counterparties`
      : "No indexed counterparties yet",
    summary.indexing.coverageWindowDays > 0
      ? `${summary.indexing.coverageWindowDays}d coverage`
      : "coverage warming up",
    summary.recentFlow.netDirection7d !== "balanced"
      ? `${summary.recentFlow.netDirection7d} flow in the last 7d`
      : "balanced recent flow",
  ].join(" · ");

  const keyFindings = [
    ...summary.latestSignals
      .slice(0, 3)
      .map(
        (signal) =>
          `${formatSignalLabel(signal.name)}: ${signal.label} (${signal.rating})`,
      ),
    ...(leadingScore
      ? [`${formatScoreLabel(leadingScore.name)} ${leadingScore.rating}`]
      : []),
  ].slice(0, 4);

  const evidence = [
    ...(clusterBreakdown
      ? [
          `Cluster cohort evidence: ${clusterBreakdown.peerWalletOverlap} peer overlaps, ${clusterBreakdown.sharedEntityLinks} shared entity links, ${clusterBreakdown.bidirectionalPeerFlows} bidirectional peer flows.`,
        ]
      : []),
    primaryCounterparty
      ? `Top counterparty ${compactAddress(primaryCounterparty.address)} with ${primaryCounterparty.interactionCount} hits.`
      : "No top counterparty evidence is available yet.",
    summary.indexing.lastIndexedAt
      ? `Last indexed ${formatRelativeTime(summary.indexing.lastIndexedAt)}.`
      : "Coverage is still warming up.",
  ];

  const nextWatch = [
    ...(clusterBreakdown?.samplingApplied ||
    clusterBreakdown?.sourceDensityCapped
      ? ["Review the sampled cohort before treating the cluster as conclusive."]
      : []),
    ...(primaryCounterparty?.entityLabel
      ? [`Follow ${primaryCounterparty.entityLabel} linked flow.`]
      : []),
    ...(primarySignal?.label
      ? [`Watch for continuation of ${primarySignal.label}.`]
      : []),
  ];

  return {
    headline,
    summary: summaryLine,
    keyFindings,
    evidence,
    nextWatch,
  };
}

const graphToneByKind: Record<string, Tone> = {
  wallet: "emerald",
  cluster: "violet",
  entity: "amber",
  counterparty: "amber",
  exchange: "teal",
};

function formatGraphKind(kind: string): string {
  return kind.replaceAll("_", " ");
}

function resolveGraphNodeLabel(
  nodes: WalletGraphPreviewNode[],
  nodeId: string,
): string {
  return nodes.find((node) => node.id === nodeId)?.label ?? nodeId;
}

function buildGraphRelationships(
  graph: WalletGraphPreview,
): WalletGraphRelationshipViewModel[] {
  const ranked = graph.edges
    .map((edge) => {
      const sourceNode = graph.nodes.find((node) => node.id === edge.sourceId);
      const targetNode = graph.nodes.find((node) => node.id === edge.targetId);
      const href = targetNode ? buildSelectedGraphNodeHref(targetNode) : null;
      const tokenFlow = edge.tokenFlow;
      const evidence = edge.evidence;

      return {
        key: buildWalletGraphEdgeKey(edge),
        sourceLabel: sourceNode?.label ?? edge.sourceId,
        targetLabel: targetNode?.label ?? edge.targetId,
        kindLabel: getWalletGraphEdgeKindLabel(edge.kind),
        directionLabel: describeGraphRelationshipDirection(edge),
        family: edge.family,
        familyLabel: getWalletGraphEdgeFamilyLabel(edge.family),
        confidence: evidence?.confidence ?? deriveRelationshipConfidence(edge),
        evidenceSummary:
          evidence?.summary ??
          buildRelationshipEvidenceSummary(edge, tokenFlow),
        evidenceSource: evidence?.source ?? "graph",
        lastTxHash: evidence?.lastTxHash ?? "",
        lastDirection: evidence?.lastDirection ?? "",
        lastProvider: evidence?.lastProvider ?? "",
        observedAt: edge.observedAt,
        weight: edge.weight ?? edge.counterpartyCount ?? 0,
        primaryToken: tokenFlow?.primaryToken ?? "",
        inboundCount: tokenFlow?.inboundCount ?? 0,
        outboundCount: tokenFlow?.outboundCount ?? 0,
        inboundAmount: tokenFlow?.inboundAmount ?? "",
        outboundAmount: tokenFlow?.outboundAmount ?? "",
        tokenBreakdowns: (tokenFlow?.breakdowns ?? []).map((item) => ({
          symbol: item.symbol,
          inboundAmount: item.inboundAmount ?? "",
          outboundAmount: item.outboundAmount ?? "",
        })),
        href,
      };
    })
    .sort((left, right) => {
      if (right.weight !== left.weight) {
        return right.weight - left.weight;
      }
      return (
        parseObservedAt(right.observedAt ?? "") -
        parseObservedAt(left.observedAt ?? "")
      );
    });

  return ranked;
}

export function filterAndSortRelatedAddresses(
  items: WalletRelatedAddressViewModel[],
  {
    directionFilter,
    sortKey,
    tokenFilter,
  }: {
    directionFilter: WalletRelatedAddressDirectionFilter;
    sortKey: WalletRelatedAddressSortKey;
    tokenFilter: string;
  },
): WalletRelatedAddressViewModel[] {
  const filtered = items.filter((item) => {
    if (directionFilter === "all") {
      return matchesTokenFilter(item, tokenFilter);
    }

    return (
      normalizeDirectionLabel(item.directionLabel) === directionFilter &&
      matchesTokenFilter(item, tokenFilter)
    );
  });

  const ranked = [...filtered];
  ranked.sort((left, right) => {
    if (sortKey === "latest_activity") {
      return (
        parseObservedAt(right.latestActivityAt) -
        parseObservedAt(left.latestActivityAt)
      );
    }

    if (sortKey === "first_seen") {
      return (
        parseObservedAt(left.firstSeenAt) - parseObservedAt(right.firstSeenAt)
      );
    }

    if (sortKey === "total_volume") {
      return totalCounterpartyVolume(right) - totalCounterpartyVolume(left);
    }

    if (sortKey === "outbound_volume") {
      return (
        parseNumericAmount(right.outboundAmount) -
        parseNumericAmount(left.outboundAmount)
      );
    }

    if (sortKey === "inbound_volume") {
      return (
        parseNumericAmount(right.inboundAmount) -
        parseNumericAmount(left.inboundAmount)
      );
    }

    return right.interactionCount - left.interactionCount;
  });

  return ranked;
}

export function WalletDetailScreen({
  request,
  summary,
  brief,
  graph,
  requestHeaders,
}: {
  request: WalletDetailRequest;
  summary: WalletSummaryPreview;
  brief?: WalletBriefPreview;
  graph: WalletGraphPreview;
  requestHeaders?: HeadersInit;
}) {
  const { t } = useTranslation();
  const searchParams = useSearchParams();
  const analystSeedQuestion = searchParams.get("ask")?.trim() ?? "";
  const consumedAnalystSeedQuestionRef = useRef<string | null>(null);
  const [summaryPreviewState, setSummaryPreviewState] = useState(summary);
  const [briefPreviewState, setBriefPreviewState] = useState<
    WalletBriefPreview | undefined
  >(brief);
  const [graphPreviewState, setGraphPreviewState] = useState(graph);
  const [directionFilter, setDirectionFilter] =
    useState<WalletRelatedAddressDirectionFilter>("all");
  const [sortKey, setSortKey] =
    useState<WalletRelatedAddressSortKey>("interaction");
  const [tokenFilter, setTokenFilter] = useState<string>("all");
  const [expandedRelatedAddressKeys, setExpandedRelatedAddressKeys] = useState<
    string[]
  >([]);
  const [selectedGraphNodeId, setSelectedGraphNodeId] = useState<string | null>(
    graph.nodes[0]?.id ?? null,
  );
  const [selectedGraphRelationshipKey, setSelectedGraphRelationshipKey] =
    useState<string | null>(null);
  const [copiedRelatedAddressKey, setCopiedRelatedAddressKey] = useState<
    string | null
  >(null);
  const [expandedGraphNeighborhoodKeys, setExpandedGraphNeighborhoodKeys] =
    useState<string[]>([]);
  const [isExpandingGraph, setIsExpandingGraph] = useState(false);
  const [isRefreshingWallet, setIsRefreshingWallet] = useState(false);
  const [isTrackingWallet, setIsTrackingWallet] = useState(false);
  const [trackWalletMessage, setTrackWalletMessage] = useState("");
  const [analystQuestion, setAnalystQuestion] = useState("");
  const [isAnalyzingWallet, setIsAnalyzingWallet] = useState(false);
  const [walletAnalysisTurns, setWalletAnalysisTurns] = useState<
    AnalystWalletAnalyzePreview[]
  >([]);
  const [walletAnalysisError, setWalletAnalysisError] = useState("");
  const getClerkRequestHeaders = useClerkRequestHeaders();
  const graphSectionRef = useRef<HTMLElement | null>(null);
  const viewModel = buildWalletDetailViewModel({
    request,
    summary: summaryPreviewState,
    graph: graphPreviewState,
    ...(briefPreviewState ? { brief: briefPreviewState } : {}),
    t,
  });
  const analystMemoryScopeKey = useMemo(
    () => buildWalletAnalystMemoryScopeKey(request.chain, request.address),
    [request.address, request.chain],
  );

  useEffect(() => {
    setSummaryPreviewState(summary);
    setBriefPreviewState(brief);
    setGraphPreviewState(graph);
    setSelectedGraphNodeId(graph.nodes[0]?.id ?? null);
    setSelectedGraphRelationshipKey(
      graph.edges[0] ? buildWalletGraphEdgeKey(graph.edges[0]) : null,
    );
    setExpandedGraphNeighborhoodKeys([]);
    setTrackWalletMessage("");
    setAnalystQuestion("");
    setWalletAnalysisError("");
    setIsTrackingWallet(false);
  }, [summary, brief, graph]);

  useEffect(() => {
    const memory = readAnalystMemory(analystMemoryScopeKey);
    if (memory.length === 0) {
      setWalletAnalysisTurns([]);
      return;
    }
    setWalletAnalysisTurns(
      memory.map((turn) => ({
        chain: request.chain,
        address: request.address,
        question: turn.question,
        contextReused: true,
        recentTurnCount: 0,
        headline: turn.headline,
        conclusion: [],
        confidence: "medium",
        observedFacts: [],
        inferredInterpretations: [],
        alternativeExplanations: [],
        nextSteps: [],
        toolTrace: turn.toolTrace,
        evidenceRefs: turn.evidenceRefs,
      })),
    );
  }, [analystMemoryScopeKey, request.address, request.chain]);

  useEffect(() => {
    const memory: AnalystMemoryTurn[] = walletAnalysisTurns.map((turn) => ({
      question: turn.question,
      headline: turn.headline,
      toolTrace: turn.toolTrace,
      evidenceRefs: turn.evidenceRefs,
      createdAt: new Date().toISOString(),
    }));
    writeAnalystMemory(analystMemoryScopeKey, memory);
  }, [analystMemoryScopeKey, walletAnalysisTurns]);

  useEffect(() => {
    persistClientForwardedAuthHeaders(requestHeaders);
  }, [requestHeaders]);

  useEffect(() => {
    if (!analystSeedQuestion) {
      return;
    }
    setAnalystQuestion((current) => current || analystSeedQuestion);
  }, [analystSeedQuestion]);

  useEffect(() => {
    const body = document.body;
    const html = document.documentElement;

    const hasVisibleModal = () =>
      Array.from(
        document.querySelectorAll(
          [
            '[role="dialog"][aria-modal="true"]',
            '[aria-modal="true"]',
            "[data-clerk-modal]",
          ].join(", "),
        ),
      ).some((element) => {
        const style = window.getComputedStyle(element);
        return (
          style.display !== "none" &&
          style.visibility !== "hidden" &&
          style.opacity !== "0"
        );
      });

    const clearScrollLockStyles = () => {
      const targets = [body, html];
      for (const target of targets) {
        target.style.removeProperty("overflow");
        target.style.removeProperty("overflow-x");
        target.style.removeProperty("overflow-y");
        target.style.removeProperty("pointer-events");
        target.style.removeProperty("touch-action");
        target.style.removeProperty("overscroll-behavior");
        target.style.removeProperty("position");
        target.style.removeProperty("top");
        target.style.removeProperty("left");
        target.style.removeProperty("right");
        target.style.removeProperty("width");
        target.style.removeProperty("height");
        target.style.removeProperty("padding-right");
      }
      body.removeAttribute("data-scroll-locked");
    };

    const syncScrollLock = () => {
      if (!hasVisibleModal()) {
        clearScrollLockStyles();
      }
    };

    syncScrollLock();

    const observer = new MutationObserver(() => {
      syncScrollLock();
    });

    observer.observe(body, {
      attributes: true,
      attributeFilter: ["style", "data-scroll-locked"],
      childList: true,
      subtree: true,
    });
    observer.observe(html, {
      attributes: true,
      attributeFilter: ["style"],
    });

    const frame = window.requestAnimationFrame(() => {
      syncScrollLock();
    });

    return () => {
      window.cancelAnimationFrame(frame);
      observer.disconnect();
    };
  }, []);

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
      if (triggerRefreshQueue) {
        await loadSearchPreview({
          query: request.address,
          refreshMode: "manual",
          ...(requestHeaders ? { requestHeaders } : {}),
        });
      }

      const nextSummary = await loadWalletSummaryPreview(
        summaryFallback
          ? {
              request,
              fallback: summaryFallback,
              ...(requestHeaders ? { requestHeaders } : {}),
            }
          : {
              request,
              ...(requestHeaders ? { requestHeaders } : {}),
            },
      );
      if (!canCommit()) {
        return;
      }
      setSummaryPreviewState(nextSummary);

      const nextBrief = await loadAnalystWalletBriefPreview(
        briefPreviewState
          ? {
              request,
              fallback: briefPreviewState,
              ...(requestHeaders ? { requestHeaders } : {}),
            }
          : {
              request,
              ...(requestHeaders ? { requestHeaders } : {}),
            },
      );
      if (!canCommit()) {
        return;
      }
      setBriefPreviewState(nextBrief);

      const loadedGraph = await loadWalletGraphPreview(
        graphFallback
          ? {
              request: {
                ...request,
                depthRequested: DEFAULT_WALLET_GRAPH_DEPTH,
              },
              fallback: graphFallback,
              ...(requestHeaders ? { requestHeaders } : {}),
            }
          : {
              request: {
                ...request,
                depthRequested: DEFAULT_WALLET_GRAPH_DEPTH,
              },
              ...(requestHeaders ? { requestHeaders } : {}),
            },
      );
      if (!canCommit()) {
        return;
      }

      const nextGraph =
        loadedGraph.mode === "unavailable" &&
        nextSummary.topCounterparties.length > 0
          ? deriveWalletGraphPreviewFromSummary({
              request: {
                ...request,
                depthRequested: DEFAULT_WALLET_GRAPH_DEPTH,
              },
              summary: nextSummary,
              fallback: loadedGraph,
            })
          : loadedGraph;
      if (!canCommit()) {
        return;
      }
      setGraphPreviewState(nextGraph);
    },
    [briefPreviewState, request, requestHeaders],
  );

  const queuedUnavailableRefresh = useRef(false);

  useEffect(() => {
    if (queuedUnavailableRefresh.current) {
      return;
    }
    if (summaryPreviewState.mode !== "unavailable") {
      return;
    }

    queuedUnavailableRefresh.current = true;
    let active = true;

    void refreshWalletArtifacts({
      triggerRefreshQueue: true,
      summaryFallback: summaryPreviewState,
      graphFallback: graphPreviewState,
      canCommit: () => active,
    });

    return () => {
      active = false;
    };
  }, [graphPreviewState, refreshWalletArtifacts, summaryPreviewState]);

  useEffect(() => {
    if (
      !shouldPollIndexedWalletSummary(summaryPreviewState) &&
      summaryPreviewState.mode !== "unavailable"
    ) {
      return;
    }

    let active = true;
    const interval = window.setInterval(() => {
      void (async () => {
        await refreshWalletArtifacts({
          summaryFallback: summaryPreviewState,
          graphFallback: graphPreviewState,
          canCommit: () => active,
        });
      })();
    }, 5000);

    return () => {
      active = false;
      window.clearInterval(interval);
    };
  }, [summaryPreviewState, graphPreviewState, refreshWalletArtifacts]);
  const availableTokens = buildRelatedAddressTokenFilters(
    viewModel.relatedAddresses,
  );
  const graphAddressIndex = useMemo(
    () => buildGraphAddressIndex(viewModel.graphNodes),
    [viewModel.graphNodes],
  );
  const graphEntityAssignmentIndex = useMemo(
    () =>
      buildGraphEntityAssignmentIndex(
        viewModel.graphNodes,
        viewModel.graphEdges,
      ),
    [viewModel.graphEdges, viewModel.graphNodes],
  );
  const visibleRelatedAddresses = filterAndSortRelatedAddresses(
    viewModel.relatedAddresses,
    {
      directionFilter,
      sortKey,
      tokenFilter,
    },
  );
  const heroTitle = looksLikeWalletAddress(viewModel.title)
    ? `${viewModel.chainLabel} wallet`
    : viewModel.title;
  const graphAvailability = useMemo(
    () => buildWalletGraphAvailabilityPresentation(graphPreviewState),
    [graphPreviewState],
  );
  const graphEmptyState = useMemo(() => {
    const hasRenderedGraph =
      graphPreviewState.nodes.length > 1 || graphPreviewState.edges.length > 0;
    if (hasRenderedGraph) {
      return null;
    }

    const summaryUnavailable = summaryPreviewState.mode === "unavailable";
    const indexingStatus = summaryPreviewState.indexing.status;
    const hasCounterparties = summaryPreviewState.topCounterparties.length > 0;

    if (summaryUnavailable || indexingStatus === "indexing") {
      return {
        title: "Queued for indexing",
        summary:
          "We queued this wallet for background backfill. Summary data may appear before relationship edges are materialized.",
        helper:
          "Keep this page open for a few seconds or refresh once to load the first graph snapshot.",
      };
    }

    if (!hasCounterparties) {
      return {
        title: "No indexed interactions yet",
        summary:
          "This wallet is indexed, but there are no counterparties in the current coverage window yet.",
        helper:
          "Try again after more activity is ingested or search a wallet with existing onchain interaction history.",
      };
    }

    return {
      title: "Graph is still warming up",
      summary:
        "We have summary coverage for this wallet, but the relationship graph has not materialized yet.",
      helper:
        "Use refresh now to request another graph load after the latest backfill finishes.",
    };
  }, [graphPreviewState, summaryPreviewState]);
  const analystRecentTurns = useMemo<AnalystWalletAnalyzeRecentTurnInput[]>(
    () =>
      walletAnalysisTurns.slice(-3).map((turn) => ({
        question: turn.question,
        headline: turn.headline,
        toolTrace: turn.toolTrace,
        evidenceRefs: turn.evidenceRefs,
      })),
    [walletAnalysisTurns],
  );
  const graphSourceCopy = graphAvailability.statusCopy;
  const selectedGraphNode =
    viewModel.graphNodes.find((node) => node.id === selectedGraphNodeId) ??
    viewModel.graphNodes[0] ??
    null;
  const selectedGraphRelationship =
    viewModel.graphRelationships.find(
      (relationship) => relationship.key === selectedGraphRelationshipKey,
    ) ??
    viewModel.graphRelationships[0] ??
    null;
  const selectedGraphNodeHref = selectedGraphNode
    ? buildSelectedGraphNodeHref(selectedGraphNode)
    : null;
  const selectedGraphEntityAssignments = useMemo(() => {
    if (!selectedGraphNode) {
      return [];
    }

    const graphAssignments =
      graphEntityAssignmentIndex.get(selectedGraphNode.id) ?? [];
    if (
      selectedGraphNode.kind !== "wallet" ||
      !selectedGraphNode.chain ||
      !selectedGraphNode.address
    ) {
      return graphAssignments;
    }

    const selectedGraphNodeAddress = selectedGraphNode.address;
    const summaryCounterparty =
      viewModel.relatedAddresses.find(
        (counterparty) =>
          counterparty.chainLabel.toLowerCase() ===
            selectedGraphNode.chain?.toLowerCase() &&
          counterparty.address.toLowerCase() ===
            selectedGraphNodeAddress.toLowerCase(),
      ) ?? null;
    const fallbackAssignments = summaryCounterparty
      ? [
          buildFallbackEntityAssignment(
            summaryCounterparty.entityKey,
            summaryCounterparty.entityLabel,
          ),
        ].filter(
          (assignment): assignment is GraphEntityAssignmentPresentation =>
            Boolean(assignment),
        )
      : [];

    return mergeEntityAssignments(graphAssignments, fallbackAssignments);
  }, [
    graphEntityAssignmentIndex,
    selectedGraphNode,
    viewModel.relatedAddresses,
  ]);
  const selectedGraphEntityContext = useMemo(
    () =>
      resolveSelectedGraphEntityContext({
        selectedNode: selectedGraphNode,
        graphNodes: viewModel.graphNodes,
        graphEdges: viewModel.graphEdges,
        relatedAddresses: viewModel.relatedAddresses,
      }),
    [
      selectedGraphNode,
      viewModel.graphEdges,
      viewModel.graphNodes,
      viewModel.relatedAddresses,
    ],
  );
  const isRelatedAddressExpanded = (
    counterparty: WalletRelatedAddressViewModel,
  ) =>
    expandedRelatedAddressKeys.includes(buildRelatedAddressKey(counterparty));
  const graphExpansionState = resolveGraphExpansionState({
    selectedNode: selectedGraphNode,
    expandedGraphNeighborhoodKeys,
    graphNodeCount: viewModel.graphNodeCount,
    graphNodes: viewModel.graphNodes,
    relatedAddresses: viewModel.relatedAddresses,
  });
  const expandableGraphNodeIds = useMemo(
    () =>
      resolveExpandableGraphNodeIds({
        graphNodes: viewModel.graphNodes,
        expandedGraphNeighborhoodKeys,
        graphNodeCount: viewModel.graphNodeCount,
        relatedAddresses: viewModel.relatedAddresses,
      }),
    [
      expandedGraphNeighborhoodKeys,
      viewModel.graphNodeCount,
      viewModel.graphNodes,
      viewModel.relatedAddresses,
    ],
  );

  useEffect(() => {
    setSelectedGraphRelationshipKey((current) => {
      if (!viewModel.graphRelationships.length) {
        return null;
      }
      if (
        current &&
        viewModel.graphRelationships.some(
          (relationship) => relationship.key === current,
        )
      ) {
        return current;
      }
      return viewModel.graphRelationships[0]?.key ?? null;
    });
  }, [viewModel.graphRelationships]);
  const handleCopyRelatedAddress = async (
    counterparty: WalletRelatedAddressViewModel,
  ) => {
    const rowKey = buildRelatedAddressKey(counterparty);
    try {
      if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(counterparty.address);
      }
      setCopiedRelatedAddressKey(rowKey);
      window.setTimeout(() => {
        setCopiedRelatedAddressKey((current) =>
          current === rowKey ? null : current,
        );
      }, 1600);
    } catch {
      setCopiedRelatedAddressKey(null);
    }
  };
  const handleFocusRelatedAddressInGraph = (
    counterparty: WalletRelatedAddressViewModel,
  ) => {
    const nodeId = resolveGraphNodeIdForAddress(
      counterparty,
      graphAddressIndex,
    );
    if (!nodeId) {
      return;
    }

    setSelectedGraphNodeId(nodeId);
    graphSectionRef.current?.scrollIntoView({
      behavior: "smooth",
      block: "start",
    });
  };
  const handleExpandSelectedGraphNode = async () => {
    if (
      !graphExpansionState.canExpand ||
      !graphExpansionState.expansionKey ||
      !selectedGraphNode
    ) {
      return;
    }

    setIsExpandingGraph(true);
    try {
      const nextGraph = await expandGraphNode({
        node: selectedGraphNode,
        graphNodes: viewModel.graphNodes,
        relatedAddresses: viewModel.relatedAddresses,
        rootRequest: request,
        ...(requestHeaders ? { requestHeaders } : {}),
      });

      if (
        nextGraph.mode === "unavailable" &&
        nextGraph.source === "boundary-unavailable"
      ) {
        return;
      }

      setGraphPreviewState((current) =>
        mergeWalletGraphPreviews(current, nextGraph),
      );
      const expansionKey = graphExpansionState.expansionKey;
      if (expansionKey) {
        setExpandedGraphNeighborhoodKeys((current) => [
          ...current,
          expansionKey,
        ]);
      }
    } finally {
      setIsExpandingGraph(false);
    }
  };
  const handleExpandGraphNode = async (nodeId: string) => {
    const node =
      viewModel.graphNodes.find((graphNode) => graphNode.id === nodeId) ?? null;
    if (!node) {
      return;
    }

    setSelectedGraphNodeId(nodeId);
    const nextExpansionState = resolveGraphExpansionState({
      selectedNode: node,
      expandedGraphNeighborhoodKeys,
      graphNodeCount: viewModel.graphNodeCount,
      graphNodes: viewModel.graphNodes,
      relatedAddresses: viewModel.relatedAddresses,
    });
    if (!nextExpansionState.canExpand || !nextExpansionState.expansionKey) {
      return;
    }

    setIsExpandingGraph(true);
    try {
      const nextGraph = await expandGraphNode({
        node,
        graphNodes: viewModel.graphNodes,
        relatedAddresses: viewModel.relatedAddresses,
        rootRequest: request,
        ...(requestHeaders ? { requestHeaders } : {}),
      });

      if (
        nextGraph.mode === "unavailable" &&
        nextGraph.source === "boundary-unavailable"
      ) {
        return;
      }

      setGraphPreviewState((current) =>
        mergeWalletGraphPreviews(current, nextGraph),
      );
      setExpandedGraphNeighborhoodKeys((current) => [
        ...current,
        nextExpansionState.expansionKey as string,
      ]);
    } finally {
      setIsExpandingGraph(false);
    }
  };
  const handleCopyRelatedAddressSummary = async (
    counterparty: WalletRelatedAddressViewModel,
  ) => {
    const rowKey = buildRelatedAddressKey(counterparty);
    try {
      if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(
          buildCounterpartySummaryCopy(counterparty),
        );
      }
      setCopiedRelatedAddressKey(`${rowKey}:summary`);
      window.setTimeout(() => {
        setCopiedRelatedAddressKey((current) =>
          current === `${rowKey}:summary` ? null : current,
        );
      }, 1600);
    } catch {
      setCopiedRelatedAddressKey(null);
    }
  };
  const handleAnalyzeWallet = useCallback(
    async (nextQuestion?: string) => {
      const question = (nextQuestion ?? analystQuestion).trim();
      if (!question) {
        setWalletAnalysisError("Enter a wallet question first.");
        return;
      }

      setIsAnalyzingWallet(true);
      setWalletAnalysisError("");
      try {
        const authHeaders = requestHeaders ?? (await getClerkRequestHeaders());
        const result = await analyzeAnalystWallet({
          request,
          question,
          recentTurns: analystRecentTurns,
          ...(authHeaders ? { requestHeaders: authHeaders } : {}),
        });
        setWalletAnalysisTurns((current) => [...current, result].slice(-4));
        setAnalystQuestion("");
      } catch (error) {
        setWalletAnalysisError(
          error instanceof Error
            ? error.message
            : "wallet analyze request failed",
        );
      } finally {
        setIsAnalyzingWallet(false);
      }
    },
    [
      analystQuestion,
      analystRecentTurns,
      getClerkRequestHeaders,
      request,
      requestHeaders,
    ],
  );

  useEffect(() => {
    if (!analystSeedQuestion) {
      return;
    }
    if (consumedAnalystSeedQuestionRef.current === analystSeedQuestion) {
      return;
    }
    if (isAnalyzingWallet) {
      return;
    }
    consumedAnalystSeedQuestionRef.current = analystSeedQuestion;
    void handleAnalyzeWallet(analystSeedQuestion);
  }, [analystSeedQuestion, handleAnalyzeWallet, isAnalyzingWallet]);
  const relationshipMapSection = (
    <article
      ref={graphSectionRef}
      className="preview-card detail-card boundary-card"
    >
      <div className="preview-header">
        <div>
          <h2>{summaryPreviewState.label}</h2>
          <span className="preview-kicker">
            {t("walletDetail.headers.graphInvestigation")}
          </span>
        </div>
        <div className="preview-state">
          <span className="detail-state-copy">
            {graphAvailability.stateLabel}
          </span>
        </div>
      </div>

      <div className="preview-status">
        <p>{graphSourceCopy}</p>
        {viewModel.graphSnapshotGeneratedAt ? (
          <span className="detail-route-copy">
            {viewModel.graphSnapshotSourceLabel} ·{" "}
            {viewModel.graphSnapshotGeneratedAt}
          </span>
        ) : null}
      </div>
      <div className="preview-identity">
        <div>
          <span>Hop expansion</span>
          <strong>{graphExpansionState.hopsUsed} expanded</strong>
        </div>
        <div>
          <span>Visible nodes</span>
          <strong>
            {graphExpansionState.nodeCount} / {graphExpansionState.nodeBudget}
          </strong>
        </div>
        <div>
          <span>Density capped</span>
          <strong>{graphPreviewState.densityCapped ? "true" : "false"}</strong>
        </div>
      </div>

      {selectedGraphNode ? (
        <div className="detail-graph-actions">
          <button
            className="search-cta detail-graph-action"
            disabled={isExpandingGraph || !graphExpansionState.canExpand}
            onClick={() => {
              void handleExpandSelectedGraphNode();
            }}
            type="button"
          >
            {isExpandingGraph
              ? "Expanding..."
              : graphExpansionState.canExpand
                ? "Expand next hop"
                : "Expand unavailable"}
          </button>
          <span className="detail-graph-action-copy">
            {graphExpansionState.reason}
          </span>
        </div>
      ) : null}

      <div className="graph-preview-strip detail-graph-stage">
        <WalletGraphVisual
          densityCapped={graphPreviewState.densityCapped}
          edges={graphPreviewState.edges}
          neighborhoodSummary={graphPreviewState.neighborhoodSummary}
          nodes={graphPreviewState.nodes}
          expandableNodeIds={expandableGraphNodeIds}
          expandingNodeId={isExpandingGraph ? selectedGraphNodeId : null}
          onExpandNode={(nodeId) => {
            void handleExpandGraphNode(nodeId);
          }}
          onSelectedEdgeIdChange={setSelectedGraphRelationshipKey}
          onSelectedNodeIdChange={setSelectedGraphNodeId}
          selectedEdgeId={selectedGraphRelationshipKey}
          selectedNodeId={selectedGraphNodeId}
          variant="hero"
        />

        <div className="detail-map-metrics">
          <article className="detail-map-metric">
            <span>Visible nodes</span>
            <strong>{viewModel.graphNodeCount}</strong>
          </article>
          <article className="detail-map-metric">
            <span>Visible edges</span>
            <strong>{viewModel.graphEdgeCount}</strong>
          </article>
          <article className="detail-map-metric">
            <span>Top relationship load</span>
            <strong>{viewModel.graphRelationships[0]?.weight ?? 0}</strong>
          </article>
          <article className="detail-map-metric">
            <span>Hop expansion</span>
            <strong>{graphExpansionState.hopsUsed} expanded</strong>
          </article>
        </div>
      </div>

      {selectedGraphNode ? (
        <div
          className="detail-node-inspector"
          data-node-kind={selectedGraphNode.kind}
        >
          <div className="detail-node-inspector-head">
            <div>
              <strong>{selectedGraphNode.label}</strong>
            </div>
            <div className="detail-node-inspector-actions">
              <Badge tone={selectedGraphNode.tone}>
                {selectedGraphNode.kindLabel}
              </Badge>
              {selectedGraphNodeHref ? (
                <a className="detail-inline-link" href={selectedGraphNodeHref}>
                  {buildSelectedGraphNodeHrefLabel(selectedGraphNode)}
                </a>
              ) : null}
            </div>
          </div>
          <div className="detail-node-inspector-grid">
            <article className="detail-node-inspector-card">
              <span>Identity</span>
              <strong>
                {selectedGraphNode.address
                  ? compactAddress(selectedGraphNode.address)
                  : selectedGraphNode.id}
              </strong>
            </article>
            <article className="detail-node-inspector-card">
              <span>Expansion</span>
              <strong>
                {graphExpansionState.canExpand ? "available" : "blocked"}
              </strong>
            </article>
            <article className="detail-node-inspector-card">
              <span>Rule</span>
              <strong>{graphExpansionState.reason}</strong>
            </article>
            <article className="detail-node-inspector-card">
              <span>Budget</span>
              <strong>{graphExpansionState.budgetLabel}</strong>
            </article>
          </div>
          {selectedGraphEntityAssignments.length > 0 ? (
            <div
              className="detail-entity-linkage"
              data-node-kind={selectedGraphNode.kind}
            >
              <div className="detail-entity-linkage-head">
                <div>
                  <span className="preview-kicker">Entity assignments</span>
                  <strong>
                    {selectedGraphEntityAssignments.length} visible label
                    {selectedGraphEntityAssignments.length === 1 ? "" : "s"}
                  </strong>
                </div>
              </div>
              <p className="detail-entity-linkage-copy">
                Provider or heuristic entity assignments attached to the
                selected wallet in the current neighborhood.
              </p>
              <div className="detail-entity-linkage-strip">
                {selectedGraphEntityAssignments.map((assignment) => (
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
          {selectedGraphEntityContext ? (
            <div
              className="detail-entity-linkage"
              data-node-kind={selectedGraphNode.kind}
            >
              <div className="detail-entity-linkage-head">
                <div>
                  <span className="preview-kicker">
                    {selectedGraphEntityContext.label}
                  </span>
                  <strong>
                    {selectedGraphEntityContext.links.length} visible link
                    {selectedGraphEntityContext.links.length === 1 ? "" : "s"}
                  </strong>
                </div>
              </div>
              <p className="detail-entity-linkage-copy">
                {selectedGraphEntityContext.helperCopy}
              </p>
              {selectedGraphEntityContext.links.length > 0 ? (
                <div className="detail-entity-linkage-strip">
                  {selectedGraphEntityContext.links.map((link) =>
                    link.href ? (
                      <a
                        key={link.id}
                        className="detail-entity-link"
                        href={link.href}
                      >
                        <span>{link.label}</span>
                        <Badge tone={link.tone}>{link.kindLabel}</Badge>
                        {link.sourceLabel ? (
                          <Badge tone={link.sourceTone ?? "teal"}>
                            {link.sourceLabel}
                          </Badge>
                        ) : null}
                      </a>
                    ) : (
                      <div key={link.id} className="detail-entity-link">
                        <span>{link.label}</span>
                        <Badge tone={link.tone}>{link.kindLabel}</Badge>
                        {link.sourceLabel ? (
                          <Badge tone={link.sourceTone ?? "teal"}>
                            {link.sourceLabel}
                          </Badge>
                        ) : null}
                      </div>
                    ),
                  )}
                </div>
              ) : null}
            </div>
          ) : null}
        </div>
      ) : null}

      {selectedGraphRelationship ? (
        <div className="detail-relationship-inspector">
          <div className="detail-relationship-inspector-head">
            <div>
              <span className="preview-kicker">Selected relationship</span>
              <strong>
                {selectedGraphRelationship.sourceLabel} →{" "}
                {selectedGraphRelationship.targetLabel}
              </strong>
            </div>
            <div className="detail-relationship-inspector-actions">
              <Badge tone="teal">{selectedGraphRelationship.kindLabel}</Badge>
              <Badge tone="amber">
                {selectedGraphRelationship.directionLabel}
              </Badge>
              <Badge
                tone={
                  selectedGraphRelationship.family === "derived"
                    ? "violet"
                    : "teal"
                }
              >
                {selectedGraphRelationship.familyLabel}
              </Badge>
              <Badge
                tone={toneForConfidence(selectedGraphRelationship.confidence)}
              >
                {selectedGraphRelationship.confidence}
              </Badge>
            </div>
          </div>
          <p className="detail-relationship-summary">
            {selectedGraphRelationship.evidenceSummary}
          </p>
          <div className="detail-relationship-inspector-grid">
            <article className="detail-node-inspector-card">
              <span>Observed</span>
              <strong>
                {selectedGraphRelationship.observedAt || "Unavailable"}
              </strong>
            </article>
            <article className="detail-node-inspector-card">
              <span>Weight</span>
              <strong>{selectedGraphRelationship.weight} hits</strong>
            </article>
            <article className="detail-node-inspector-card">
              <span>Flow type</span>
              <strong>{selectedGraphRelationship.directionLabel}</strong>
            </article>
            <article className="detail-node-inspector-card">
              <span>Primary token</span>
              <strong>
                {selectedGraphRelationship.primaryToken || "Unavailable"}
              </strong>
            </article>
            <article className="detail-node-inspector-card">
              <span>Evidence source</span>
              <strong>{selectedGraphRelationship.evidenceSource}</strong>
            </article>
          </div>
          <div className="detail-relationship-flow-strip">
            <article className="detail-relationship-flow-card">
              <span>Inbound</span>
              <strong>{selectedGraphRelationship.inboundAmount || "0"}</strong>
              <small>{selectedGraphRelationship.inboundCount} transfers</small>
            </article>
            <article className="detail-relationship-flow-card">
              <span>Outbound</span>
              <strong>{selectedGraphRelationship.outboundAmount || "0"}</strong>
              <small>{selectedGraphRelationship.outboundCount} transfers</small>
            </article>
            <article className="detail-relationship-flow-card">
              <span>Latest provider</span>
              <strong>
                {selectedGraphRelationship.lastProvider || "Unavailable"}
              </strong>
              <small>
                {selectedGraphRelationship.lastTxHash
                  ? compactAddress(selectedGraphRelationship.lastTxHash)
                  : "No tx hash"}
              </small>
            </article>
          </div>
          {selectedGraphRelationship.tokenBreakdowns.length > 0 ? (
            <div className="detail-relationship-token-breakdowns">
              {selectedGraphRelationship.tokenBreakdowns
                .slice(0, 3)
                .map((token) => (
                  <article
                    key={`${selectedGraphRelationship.key}:${token.symbol}`}
                    className="detail-relationship-token-breakdown"
                  >
                    <strong>{token.symbol}</strong>
                    <span>
                      IN {token.inboundAmount || "0"} · OUT{" "}
                      {token.outboundAmount || "0"}
                    </span>
                  </article>
                ))}
            </div>
          ) : null}
        </div>
      ) : null}

      {viewModel.graphRelationships.length > 0 ? (
        <div className="detail-relationship-list">
          <div className="detail-relationship-list-head">
            <div>
              <strong>{viewModel.graphRelationships.length} edges</strong>
            </div>
            <span className="detail-relationship-count">
              {graphPreviewState.mode === "live"
                ? "Live graph"
                : "Waiting for graph"}
            </span>
          </div>

          <div className="detail-relationship-list-body">
            {viewModel.graphRelationships.slice(0, 6).map((relationship) => (
              <button
                key={relationship.key}
                type="button"
                className={`detail-relationship-item ${
                  relationship.key === selectedGraphRelationshipKey
                    ? "detail-relationship-item-active"
                    : ""
                }`}
                onClick={() =>
                  setSelectedGraphRelationshipKey(relationship.key)
                }
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    setSelectedGraphRelationshipKey(relationship.key);
                  }
                }}
              >
                <div>
                  <span className="detail-relationship-path">
                    {relationship.sourceLabel} → {relationship.targetLabel}
                  </span>
                  <span className="detail-relationship-meta">
                    {relationship.kindLabel} · {relationship.directionLabel} ·{" "}
                    {relationship.familyLabel}
                    {relationship.observedAt
                      ? ` · ${relationship.observedAt}`
                      : ""}
                  </span>
                  <span className="detail-relationship-meta">
                    {relationship.primaryToken
                      ? `${relationship.primaryToken} · IN ${relationship.inboundAmount || "0"} · OUT ${relationship.outboundAmount || "0"}`
                      : relationship.evidenceSummary}
                  </span>
                </div>
                <div className="detail-relationship-actions">
                  <Badge tone="teal">{relationship.weight} hits</Badge>
                  {relationship.href ? (
                    <a className="detail-inline-link" href={relationship.href}>
                      Open
                    </a>
                  ) : null}
                </div>
              </button>
            ))}
          </div>
        </div>
      ) : null}
    </article>
  );

  return (
    <PageShell>
      <div className="detail-shell detail-shell--redesigned">
        {/* 1. Header (Hero) */}
        <section className="detail-hero detail-hero--redesigned">
          <div className="detail-hero-copy">
            <div className="detail-hero-title-row">
              <h1>{heroTitle}</h1>
              {viewModel.aiBrief.keyFindings[0] ? (
                <Badge
                  tone={
                    (summaryPreviewState.scores[0]?.value ?? 60) >= 70
                      ? "emerald"
                      : "amber"
                  }
                >
                  {formatPercent(
                    (summaryPreviewState.scores[0]?.value ?? 60) / 100,
                  )}{" "}
                  confidence
                </Badge>
              ) : null}
            </div>
            <p className="detail-hero-summary">
              {viewModel.aiBrief.summary || summaryPreviewState.label}
            </p>
          </div>

          <div className="detail-identity detail-identity--compact">
            <div>
              <span>Chain</span>
              <strong>{viewModel.chainLabel}</strong>
            </div>
            <div>
              <span>Wallet address</span>
              <strong>{viewModel.address}</strong>
            </div>
          </div>

          <div className="detail-actions">
            <LanguageSwitcher />
            <button
              className="search-cta"
              disabled={isTrackingWallet}
              onClick={() => {
                void (async () => {
                  setIsTrackingWallet(true);
                  setTrackWalletMessage("");
                  try {
                    const authHeaders =
                      requestHeaders ?? (await getClerkRequestHeaders());
                    const result = await trackWalletAlertRule({
                      chain: request.chain,
                      address: request.address,
                      label: summaryPreviewState.label,
                      ...(authHeaders ? { requestHeaders: authHeaders } : {}),
                    });
                    if (result.nextHref) {
                      window.location.assign(result.nextHref);
                      return;
                    }
                    setTrackWalletMessage(result.message);
                  } finally {
                    setIsTrackingWallet(false);
                  }
                })();
              }}
              type="button"
            >
              {isTrackingWallet ? "Tracking..." : "Track wallet"}
            </button>
            <a className="search-cta" href="#graph-canvas">
              Open graph
            </a>
            <a className="search-cta" href="#evidence-timeline">
              View evidence
            </a>
          </div>

          {trackWalletMessage ? (
            <p className="detail-route-copy" aria-live="polite">
              {trackWalletMessage}
            </p>
          ) : null}
          <article className="preview-card detail-card boundary-card">
            <div className="preview-header">
              <div>
                <h2>Interactive analyst</h2>
                <span className="preview-kicker">AI-powered wallet intelligence</span>
              </div>
              <div className="preview-state">
                <Badge tone="violet">
                  {walletAnalysisTurns.length} Turn
                  {walletAnalysisTurns.length === 1 ? "" : "s"}
                </Badge>
              </div>
            </div>

            <div className="analyst-chat-container">
              {walletAnalysisTurns.length > 0 && (
                <div className="analyst-history">
                  {[...walletAnalysisTurns].map((turn, index) => (
                    <Fragment key={`${turn.question}:${index}`}>
                      <div className="analyst-message-user">
                        <strong>You</strong>
                        <div className="analyst-message-content">{turn.question}</div>
                      </div>

                      <div className="analyst-message-ai">
                        <strong>Analyst</strong>
                        <div className="analyst-message-content">
                          <p style={{ fontWeight: 600, fontSize: "1.05rem", marginBottom: "8px" }}>{turn.headline}</p>
                          {turn.conclusion.length > 0 && (
                            <p style={{ marginBottom: "12px" }}>{turn.conclusion.join(" ")}</p>
                          )}

                          <div className="analyst-message-findings">
                            {turn.observedFacts.length > 0 && (
                              <div className="analyst-finding-row">
                                <span>Facts</span>
                                <div>{turn.observedFacts.join(" · ")}</div>
                              </div>
                            )}
                            {turn.alternativeExplanations.length > 0 && (
                              <div className="analyst-finding-row">
                                <span>Alternatives</span>
                                <div>{turn.alternativeExplanations.join(" · ")}</div>
                              </div>
                            )}
                            {turn.toolTrace.length > 0 && (
                              <div className="analyst-finding-row">
                                <span>Analysis</span>
                                <div style={{ opacity: 0.7, fontSize: "0.8rem" }}>
                                  Used: {turn.toolTrace.join(", ")}
                                </div>
                              </div>
                            )}
                          </div>

                          {turn.evidenceRefs.length > 0 && (
                            <div className="detail-inline-evidence">
                              {turn.evidenceRefs
                                .map(describeAnalystEvidenceRef)
                                .filter(Boolean)
                                .map((item) => (
                                  <Badge key={item} tone="teal">
                                    {item}
                                  </Badge>
                                ))}
                            </div>
                          )}

                          {turn.nextSteps.length > 0 && (
                            <div className="analyst-next-steps">
                              {turn.nextSteps.map((step) => (
                                <button
                                  key={step}
                                  className="analyst-suggestion-pill"
                                  onClick={() => void handleAnalyzeWallet(step)}
                                  type="button"
                                >
                                  {step}
                                </button>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    </Fragment>
                  ))}
                </div>
              )}

              {isAnalyzingWallet && (
                <div className="analyst-message-ai analyst-loading-state">
                  <div className="analyst-shimmer" />
                  <span>Analyst is processing data...</span>
                </div>
              )}

              <div className="analyst-input-wrapper">
                <input
                  className="analyst-chat-input"
                  onChange={(event) => setAnalystQuestion(event.currentTarget.value)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter") {
                      event.preventDefault();
                      void handleAnalyzeWallet();
                    }
                  }}
                  placeholder="Ask the analyst anything about this wallet..."
                  type="text"
                  value={analystQuestion}
                />
                <button
                  className="analyst-send-button"
                  disabled={isAnalyzingWallet || !analystQuestion.trim()}
                  onClick={() => void handleAnalyzeWallet()}
                  type="button"
                  aria-label="Send question"
                >
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                    <line x1="22" y1="2" x2="11" y2="13"></line>
                    <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
                  </svg>
                </button>
              </div>

              {walletAnalysisError && (
                <p className="detail-route-copy" style={{ color: "var(--amber)", marginTop: "-10px" }} aria-live="polite">
                  {walletAnalysisError}
                </p>
              )}
            </div>
          </article>
        </section>

        {/* 2. AI Wallet Brief (Key Findings) */}
        {briefPreviewState?.mode === "live" &&
        briefPreviewState.keyFindings.length > 0 ? (
          <section className="detail-finding-cards" aria-label="Key Findings">
            <h2>{t("walletDetail.headers.aiBrief")}</h2>
            <div className="detail-signal-list">
              {briefPreviewState.keyFindings.slice(0, 4).map((finding) => (
                <article
                  key={finding.id}
                  className="detail-signal-item detail-finding-card"
                >
                  <div>
                    <strong>{finding.summary}</strong>
                    <span>{finding.observedFacts.slice(0, 2).join(" · ")}</span>
                    <div className="detail-finding-actions">
                      <a href="#graph-canvas" className="detail-inline-link">
                        Drill down in graph
                      </a>
                      <button
                        className="detail-inline-link"
                        onClick={() => {
                          void handleAnalyzeWallet(
                            `Explain the ${finding.type.replaceAll("_", " ")} finding for this wallet.`,
                          );
                        }}
                        type="button"
                      >
                        Explain with AI
                      </button>
                    </div>
                  </div>
                  <Badge
                    tone={finding.importanceScore >= 0.7 ? "emerald" : "amber"}
                  >
                    {formatPercent(finding.confidence)} confidence
                  </Badge>
                </article>
              ))}
            </div>
          </section>
        ) : null}

        {/* 3. Graph Canvas (Main Section) */}
        <div id="graph-canvas" className="detail-graph-container">
          {graphEmptyState ? (
            <section className="detail-graph-empty-state" aria-live="polite">
              <div>
                <span className="preview-kicker">
                  {graphAvailability.modeLabel}
                </span>
                <h3>{graphEmptyState.title}</h3>
                <p>{graphEmptyState.summary}</p>
                <p className="detail-route-copy">{graphEmptyState.helper}</p>
              </div>
              <div className="detail-actions">
                <button
                  className="search-cta"
                  disabled={isRefreshingWallet}
                  onClick={() => {
                    void (async () => {
                      setIsRefreshingWallet(true);
                      try {
                        await refreshWalletArtifacts({
                          triggerRefreshQueue: true,
                          summaryFallback: summaryPreviewState,
                          graphFallback: graphPreviewState,
                        });
                      } finally {
                        setIsRefreshingWallet(false);
                      }
                    })();
                  }}
                  type="button"
                >
                  {isRefreshingWallet ? "Refreshing..." : "Refresh now"}
                </button>
              </div>
            </section>
          ) : null}
          {relationshipMapSection}
        </div>

        {/* 4. Evidence Timeline */}
        <section id="evidence-timeline" className="detail-timeline-section">
          <h2>Evidence Timeline</h2>
          <div className="detail-timeline-list">
            {viewModel.aiBrief.evidence.map((evidenceItem) => {
              return (
                <article
                  key={`evidence-${evidenceItem}`}
                  className="detail-timeline-item"
                >
                  <div className="timeline-marker" />
                  <div className="timeline-content">
                    <strong>Evidence Observation</strong>
                    <span>{evidenceItem}</span>
                  </div>
                </article>
              );
            })}
            {viewModel.latestSignals.map((signal) => (
              <article
                key={`${signal.name}-${signal.observedAt}-${signal.label}`}
                className="detail-timeline-item"
              >
                <div className="timeline-marker" />
                <div className="timeline-content">
                  <strong>{signal.label}</strong>
                  <span>
                    {formatScoreLabel(signal.name)} ·{" "}
                    {formatRelativeTime(signal.observedAt)}
                  </span>
                </div>
                <Badge tone={scoreToneByName[signal.name] ?? "teal"}>
                  {signal.rating}
                </Badge>
              </article>
            ))}
          </div>
        </section>

        {/* 5. Next Watch / Historical Analogs */}
        {viewModel.aiBrief.nextWatch.length > 0 ? (
          <section className="detail-next-watch-section">
            <h2>Next watch & analogs</h2>
            <div className="detail-enrichment-list">
              {viewModel.aiBrief.nextWatch.map((item) => (
                <span key={item} className="detail-enrichment-item">
                  {item}
                </span>
              ))}
            </div>
          </section>
        ) : null}

        <section className="detail-stacked-grid">
          {/* 6. Counterparties (Related Addresses) */}
          <article className="preview-card detail-card detail-counterparties-section">
            <div className="preview-header">
              <div>
                <h2>{t("walletDetail.headers.relatedAddresses")}</h2>
                <span className="preview-kicker">Counterparties</span>
              </div>
              <div className="preview-state">
                <span className="detail-state-copy">
                  {viewModel.relatedAddressCountLabel}
                </span>
              </div>
            </div>

            <div className="related-address-toolbar">
              <div
                className="related-address-filters"
                aria-label="Direction filter"
              >
                {(
                  [
                    ["all", "All"],
                    ["outbound", "Outbound"],
                    ["inbound", "Inbound"],
                    ["mixed", "Mixed"],
                  ] as const
                ).map(([value, label]) => (
                  <button
                    key={value}
                    className={`related-address-filter ${directionFilter === value ? "related-address-filter-active" : ""}`}
                    onClick={() => {
                      setDirectionFilter(value);
                    }}
                    type="button"
                  >
                    {label}
                  </button>
                ))}
              </div>
              <label className="related-address-sort">
                <span>Token</span>
                <select
                  value={tokenFilter}
                  onChange={(event) => {
                    setTokenFilter(event.currentTarget.value);
                  }}
                >
                  <option value="all">All tokens</option>
                  {availableTokens.map((token) => (
                    <option key={token} value={token}>
                      {token}
                    </option>
                  ))}
                </select>
              </label>
              <label className="related-address-sort">
                <span>Sort</span>
                <select
                  value={sortKey}
                  onChange={(event) => {
                    setSortKey(
                      event.currentTarget.value as WalletRelatedAddressSortKey,
                    );
                  }}
                >
                  <option value="interaction">Most interactions</option>
                  <option value="total_volume">Highest total volume</option>
                  <option value="outbound_volume">Highest outbound</option>
                  <option value="inbound_volume">Highest inbound</option>
                  <option value="latest_activity">Latest activity</option>
                  <option value="first_seen">First seen</option>
                </select>
              </label>
            </div>
            <div className="related-address-table-shell">
              <table
                className="related-address-table"
                aria-label="Related addresses preview"
              >
                <thead>
                  <tr>
                    <th>Counterparty</th>
                    <th>Direction</th>
                    <th>In / out</th>
                    <th>Amount</th>
                    <th>First seen</th>
                    <th>Latest activity</th>
                    <th>Interactions</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {visibleRelatedAddresses.map((counterparty) => {
                    const rowKey = buildRelatedAddressKey(counterparty);
                    const expanded = isRelatedAddressExpanded(counterparty);
                    const graphNodeId = resolveGraphNodeIdForAddress(
                      counterparty,
                      graphAddressIndex,
                    );
                    const entityAssignments = graphNodeId
                      ? (graphEntityAssignmentIndex.get(graphNodeId) ?? [])
                      : [];
                    const summaryEntityAssignment =
                      buildCounterpartyEntityAssignment(counterparty);
                    const primaryEntityAssignment =
                      entityAssignments[0] ?? summaryEntityAssignment;

                    return (
                      <Fragment key={rowKey}>
                        <tr key={rowKey}>
                          <td>
                            <div className="related-address-cell">
                              <strong>{counterparty.address}</strong>
                              <span>{counterparty.chainLabel}</span>
                              {primaryEntityAssignment ? (
                                <div className="related-address-entity-meta">
                                  {primaryEntityAssignment.entityHref ? (
                                    <a
                                      className="detail-inline-link"
                                      href={primaryEntityAssignment.entityHref}
                                    >
                                      {primaryEntityAssignment.entityLabel}
                                    </a>
                                  ) : (
                                    <strong>
                                      {primaryEntityAssignment.entityLabel}
                                    </strong>
                                  )}
                                  <Badge
                                    tone={primaryEntityAssignment.sourceTone}
                                  >
                                    {primaryEntityAssignment.sourceLabel}
                                  </Badge>
                                </div>
                              ) : null}
                            </div>
                          </td>
                          <td>
                            <Badge
                              tone={flowToneByDirection(
                                counterparty.directionLabel,
                              )}
                            >
                              {counterparty.directionLabel}
                            </Badge>
                          </td>
                          <td>
                            <strong>
                              {counterparty.inboundCount} /{" "}
                              {counterparty.outboundCount}
                            </strong>
                          </td>
                          <td>
                            <div className="related-address-cell">
                              <strong>
                                {formatCounterpartyAmount(counterparty)}
                              </strong>
                              <span>
                                {formatCounterpartyTokenSummary(counterparty)}
                              </span>
                            </div>
                          </td>
                          <td>{counterparty.firstSeenAt || "n/a"}</td>
                          <td>{counterparty.latestActivityAt}</td>
                          <td>
                            <strong>{counterparty.interactionCount}</strong>
                          </td>
                          <td>
                            <div className="related-address-actions">
                              <button
                                className="detail-inline-button"
                                onClick={() => {
                                  setExpandedRelatedAddressKeys((current) =>
                                    toggleExpandedRelatedAddress(
                                      current,
                                      rowKey,
                                    ),
                                  );
                                }}
                                type="button"
                              >
                                {expanded ? "Hide tokens" : "Token breakdown"}
                              </button>
                              {graphNodeId ? (
                                <button
                                  className="detail-inline-button"
                                  onClick={() => {
                                    handleFocusRelatedAddressInGraph(
                                      counterparty,
                                    );
                                  }}
                                  type="button"
                                >
                                  Focus in graph
                                </button>
                              ) : null}
                              <button
                                className="detail-inline-button"
                                onClick={() => {
                                  void handleCopyRelatedAddress(counterparty);
                                }}
                                type="button"
                              >
                                {copiedRelatedAddressKey === rowKey
                                  ? "Copied"
                                  : "Copy address"}
                              </button>
                              <a
                                className="detail-inline-link"
                                href={counterparty.href}
                              >
                                Open
                              </a>
                            </div>
                          </td>
                        </tr>
                        {expanded ? (
                          <tr className="related-address-expanded-row">
                            <td colSpan={8}>
                              <div className="related-address-expanded-shell">
                                <div className="related-address-expanded-head">
                                  <div className="related-address-expanded-title">
                                    <span>Token breakdown</span>
                                    <strong>
                                      {counterparty.tokenBreakdownCount} tokens
                                    </strong>
                                  </div>
                                  <div className="related-address-expanded-actions">
                                    <a
                                      className="detail-inline-link"
                                      href={buildProductSearchHref(
                                        counterparty.address,
                                      )}
                                    >
                                      Open in search
                                    </a>
                                    <button
                                      className="detail-inline-button"
                                      onClick={() => {
                                        void handleCopyRelatedAddressSummary(
                                          counterparty,
                                        );
                                      }}
                                      type="button"
                                    >
                                      {copiedRelatedAddressKey ===
                                      `${rowKey}:summary`
                                        ? "Summary copied"
                                        : "Copy summary"}
                                    </button>
                                  </div>
                                </div>
                                <div className="related-address-breakdown-list">
                                  {counterparty.tokenBreakdowns.map((token) => (
                                    <article
                                      key={`${rowKey}:${token.symbol}`}
                                      className="related-address-breakdown-card"
                                    >
                                      <div className="related-address-breakdown-top">
                                        <strong>{token.symbol}</strong>
                                        <span>
                                          {formatTokenBreakdownTotal(token)}
                                        </span>
                                      </div>
                                      <div className="related-address-breakdown-flow">
                                        <span>
                                          IN{" "}
                                          {normalizeAmount(token.inboundAmount)}
                                        </span>
                                        <span>
                                          OUT{" "}
                                          {normalizeAmount(
                                            token.outboundAmount,
                                          )}
                                        </span>
                                      </div>
                                    </article>
                                  ))}
                                </div>
                              </div>
                            </td>
                          </tr>
                        ) : null}
                      </Fragment>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </article>

          {/* 7. Secondary Info (Tabs/Accordion) */}
          <details className="detail-accordion secondary-info-accordion">
            <summary>Enrichment, Labels & Flow (2nd-tier Data)</summary>
            <div className="detail-accordion-content">
              {viewModel.enrichment ? (
                <div className="detail-enrichment-grid">
                  <article className="detail-enrichment-card">
                    <span>Net worth</span>
                    <strong>
                      {formatNetWorthUsd(viewModel.enrichment.netWorthUsd)}
                    </strong>
                    <p>
                      {formatEnrichmentProvider(viewModel.enrichment.provider)}{" "}
                      · {formatRelativeTime(viewModel.enrichment.updatedAt)}
                    </p>
                  </article>
                  <article className="detail-enrichment-card">
                    <span>Native balance</span>
                    <strong>
                      {formatEnrichmentValue(
                        viewModel.enrichment.nativeBalanceFormatted,
                      )}
                    </strong>
                    <p>{formatEnrichmentSource(viewModel.enrichment.source)}</p>
                  </article>
                  <article className="detail-enrichment-card detail-enrichment-card--wide">
                    <span>Top holdings</span>
                    <strong>{viewModel.enrichment.holdingCount}</strong>
                    <div className="detail-holdings-list">
                      {viewModel.enrichment.holdings
                        .slice(0, 4)
                        .map((holding) => (
                          <div
                            key={`${holding.symbol}:${holding.tokenAddress}`}
                            className="detail-holding-item"
                          >
                            <div>
                              <strong>{holding.symbol || "Token"}</strong>
                              <span>
                                {holding.balanceFormatted || "Unavailable"}
                                {holding.isNative ? " · native" : ""}
                              </span>
                            </div>
                            <div>
                              <strong>
                                {formatHoldingUsdValue(holding.valueUsd)}
                              </strong>
                              <span>
                                {formatHoldingAllocation(
                                  holding.portfolioPercentage,
                                )}
                              </span>
                            </div>
                          </div>
                        ))}
                    </div>
                  </article>
                </div>
              ) : null}

              <div
                className="detail-flow-grid"
                aria-label="Recent flow summary"
              >
                <article className="detail-flow-card">
                  <span>7d in / out</span>
                  <strong>
                    {viewModel.recentFlow.incomingTxCount7d} /{" "}
                    {viewModel.recentFlow.outgoingTxCount7d}
                  </strong>
                  <Badge
                    tone={flowToneByDirection(
                      viewModel.recentFlow.netDirection7d,
                    )}
                  >
                    {viewModel.recentFlow.netDirection7d}
                  </Badge>
                </article>
                <article className="detail-flow-card">
                  <span>30d in / out</span>
                  <strong>
                    {viewModel.recentFlow.incomingTxCount30d} /{" "}
                    {viewModel.recentFlow.outgoingTxCount30d}
                  </strong>
                  <Badge
                    tone={flowToneByDirection(
                      viewModel.recentFlow.netDirection30d,
                    )}
                  >
                    {viewModel.recentFlow.netDirection30d}
                  </Badge>
                </article>
              </div>
            </div>
          </details>

          {/* 8. Raw Details (Debug) */}
          <details className="detail-accordion raw-details-accordion">
            <summary>Raw Details & Status</summary>
            <div className="detail-accordion-content">
              <div className="preview-identity">
                <div>
                  <span>Address</span>
                  <strong>{compactAddress(summaryPreviewState.address)}</strong>
                </div>
                <div>
                  <span>Updated</span>
                  <strong>
                    {viewModel.indexing.lastIndexedAt
                      ? formatRelativeTime(viewModel.indexing.lastIndexedAt)
                      : "Warming up"}
                  </strong>
                </div>
              </div>

              <div className="detail-indexing-grid">
                <article className="detail-flow-card">
                  <span>Status</span>
                  <strong>{viewModel.indexing.statusLabel}</strong>
                  <p>{viewModel.indexing.helperCopy}</p>
                </article>
                <article className="detail-flow-card">
                  <span>Coverage</span>
                  <strong>{viewModel.indexing.coverageWindowLabel}</strong>
                  <p>{renderCoverageRange(viewModel.indexing)}</p>
                </article>
              </div>

              <div className="preview-scores">
                {viewModel.summaryScores.map((score) => (
                  <article key={score.name} className="score-row">
                    <div className="score-row-copy">
                      <span>{score.name}</span>
                      <strong>{score.value}</strong>
                      {score.clusterBreakdown ? (
                        <div className="score-breakdown">
                          <div className="score-breakdown-grid">
                            <div>
                              <span>Peer overlap</span>
                              <strong>
                                {score.clusterBreakdown.peerWalletOverlap}
                              </strong>
                            </div>
                            <div>
                              <span>Shared entities</span>
                              <strong>
                                {score.clusterBreakdown.sharedEntityLinks}
                              </strong>
                            </div>
                            <div>
                              <span>Bidirectional flow</span>
                              <strong>
                                {score.clusterBreakdown.bidirectionalPeerFlows}
                              </strong>
                            </div>
                          </div>
                          {renderClusterScoreBreakdownNote(
                            score.clusterBreakdown,
                          )}
                        </div>
                      ) : null}
                    </div>
                    <Badge tone={scoreToneByName[score.name] ?? score.tone}>
                      {score.rating}
                    </Badge>
                  </article>
                ))}
              </div>
            </div>
          </details>
        </section>
      </div>
    </PageShell>
  );
}

function flowToneByDirection(direction: string): Tone {
  if (direction === "outbound") {
    return "amber";
  }

  if (direction === "inbound") {
    return "teal";
  }

  return "violet";
}

function renderClusterScoreBreakdownNote(
  breakdown: WalletSummaryClusterScoreBreakdownPreview,
): string {
  const notes: string[] = [];

  if (breakdown.samplingApplied || breakdown.sourceDensityCapped) {
    const source =
      breakdown.sourceNodeCount > 0 || breakdown.sourceEdgeCount > 0
        ? `${breakdown.sourceNodeCount} nodes / ${breakdown.sourceEdgeCount} edges`
        : "dense source graph";
    const analysis =
      breakdown.analysisNodeCount > 0 || breakdown.analysisEdgeCount > 0
        ? `${breakdown.analysisNodeCount} nodes / ${breakdown.analysisEdgeCount} edges`
        : "analysis graph";
    notes.push(`Sampled from ${source} into ${analysis}.`);
  }
  if (breakdown.contradictionPenalty > 0) {
    notes.push(
      `Contradiction penalty ${breakdown.contradictionPenalty} applied.`,
    );
  }
  if (breakdown.suppressionDiscount > 0) {
    notes.push(
      `Suppression discount ${breakdown.suppressionDiscount} applied.`,
    );
  }
  if (breakdown.contradictionReasons[0]) {
    notes.push(
      `Caution: ${formatReasonLabel(breakdown.contradictionReasons[0])}.`,
    );
  } else if (breakdown.suppressionReasons[0]) {
    notes.push(
      `Caution: ${formatReasonLabel(breakdown.suppressionReasons[0])}.`,
    );
  }

  return notes.join(" ");
}

function findPrimaryClusterBreakdown(
  summary: WalletSummaryPreview,
): WalletSummaryClusterScoreBreakdownPreview | undefined {
  for (const score of summary.scores) {
    if (score.name === "cluster_score" && score.clusterBreakdown) {
      return score.clusterBreakdown;
    }
  }
  return undefined;
}

function formatReasonLabel(reason: string): string {
  return reason.replaceAll("_", " ");
}

function describeAnalystEvidenceRef(
  ref: AnalystWalletAnalyzeEvidenceRefPreview,
): string {
  if (ref.kind === "cluster_context") {
    const peerOverlap = readEvidenceRefNumber(ref.metadata?.peerWalletOverlap);
    const sharedEntities = readEvidenceRefNumber(
      ref.metadata?.sharedEntityLinks,
    );
    const bidirectionalFlow = readEvidenceRefNumber(
      ref.metadata?.bidirectionalPeerFlow,
    );
    const contradictionPenalty = readEvidenceRefNumber(
      ref.metadata?.contradictionPenalty,
    );
    const suppressionDiscount = readEvidenceRefNumber(
      ref.metadata?.suppressionDiscount,
    );
    const notes: string[] = [];
    if (peerOverlap > 0 || sharedEntities > 0 || bidirectionalFlow > 0) {
      notes.push(
        `Cluster cohort: ${peerOverlap} peer overlaps, ${sharedEntities} shared entities, ${bidirectionalFlow} bidirectional flows`,
      );
    }
    if (
      readEvidenceRefBoolean(ref.metadata?.samplingApplied) ||
      readEvidenceRefBoolean(ref.metadata?.sourceDensityCapped)
    ) {
      notes.push("sampled from a denser graph");
    }
    if (contradictionPenalty > 0) {
      notes.push(`contradiction penalty ${contradictionPenalty}`);
    }
    if (suppressionDiscount > 0) {
      notes.push(`suppression discount ${suppressionDiscount}`);
    }
    return notes.join(" · ");
  }

  if (ref.label?.trim()) {
    return `${ref.kind.replaceAll("_", " ")}: ${ref.label.trim()}`;
  }
  if (ref.key?.trim()) {
    return `${ref.kind.replaceAll("_", " ")}: ${ref.key.trim()}`;
  }
  return ref.kind.replaceAll("_", " ");
}

function readEvidenceRefNumber(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string") {
    const parsed = Number.parseFloat(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function readEvidenceRefBoolean(value: unknown): boolean {
  if (typeof value === "boolean") {
    return value;
  }
  if (typeof value === "string") {
    return value === "true";
  }
  return false;
}

function toneForConfidence(confidence: string): Tone {
  if (confidence === "high") {
    return "emerald";
  }
  if (confidence === "medium") {
    return "amber";
  }
  return "violet";
}

function deriveRelationshipConfidence(edge: WalletGraphPreviewEdge): string {
  const weight = edge.weight ?? edge.counterpartyCount ?? 0;
  if (edge.family === "derived") {
    return weight >= 3 ? "high" : weight >= 1 ? "medium" : "low";
  }
  if (weight >= 5) {
    return "high";
  }
  if (weight >= 2) {
    return "medium";
  }
  return "low";
}

function buildRelationshipEvidenceSummary(
  edge: WalletGraphPreviewEdge,
  tokenFlow: WalletGraphPreviewEdge["tokenFlow"] | null | undefined,
): string {
  const weight = edge.weight ?? edge.counterpartyCount ?? 0;
  const primaryToken = tokenFlow?.primaryToken ?? "";
  const inboundCount = tokenFlow?.inboundCount ?? 0;
  const outboundCount = tokenFlow?.outboundCount ?? 0;
  if (edge.kind === "funded_by") {
    return primaryToken
      ? `Observed inbound funding across ${weight} transfers, led by ${primaryToken}.`
      : `Observed inbound funding across ${weight} transfers.`;
  }
  if (edge.kind === "interacted_with") {
    if (inboundCount > 0 && outboundCount > 0) {
      return primaryToken
        ? `Observed transfer activity in both directions (IN ${inboundCount} · OUT ${outboundCount}), led by ${primaryToken}.`
        : `Observed transfer activity in both directions (IN ${inboundCount} · OUT ${outboundCount}).`;
    }
    if (outboundCount > 0) {
      return primaryToken
        ? `Observed ${outboundCount} outbound transfers to this counterparty, led by ${primaryToken}.`
        : `Observed ${outboundCount} outbound transfers to this counterparty.`;
    }
    if (inboundCount > 0) {
      return primaryToken
        ? `Observed ${inboundCount} inbound transfers from this counterparty, led by ${primaryToken}.`
        : `Observed ${inboundCount} inbound transfers from this counterparty.`;
    }
    return primaryToken
      ? `Observed ${weight} direct transfers, with ${primaryToken} as the leading token.`
      : `Observed ${weight} direct transfers between these wallets.`;
  }
  return "Derived graph relationship.";
}

function normalizeDirectionLabel(
  direction: string,
): WalletRelatedAddressDirectionFilter {
  if (direction === "inbound" || direction === "outbound") {
    return direction;
  }

  return "mixed";
}

function parseObservedAt(value: string): number {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return 0;
  }

  return parsed;
}

function buildRelatedAddressKey(
  counterparty: WalletRelatedAddressViewModel,
): string {
  return `${counterparty.chainLabel}:${counterparty.address}`;
}

function buildGraphAddressIndex(
  nodes: WalletGraphNodeViewModel[],
): Map<string, string> {
  const index = new Map<string, string>();
  for (const node of nodes) {
    if (!node.address || !node.chain) {
      continue;
    }

    index.set(`${node.chain}:${node.address.toLowerCase()}`, node.id);
  }

  return index;
}

function resolveGraphNodeIdForAddress(
  counterparty: WalletRelatedAddressViewModel,
  graphAddressIndex: Map<string, string>,
): string | null {
  return (
    graphAddressIndex.get(
      `${toCounterpartyChainKey(counterparty.chainLabel)}:${counterparty.address.toLowerCase()}`,
    ) ?? null
  );
}

function toCounterpartyChainKey(chainLabel: string): "evm" | "solana" {
  return chainLabel === "SOL" || chainLabel === "SOLANA" ? "solana" : "evm";
}

function toggleExpandedRelatedAddress(
  current: string[],
  key: string,
): string[] {
  if (current.includes(key)) {
    return current.filter((entry) => entry !== key);
  }

  return [...current, key];
}

function parseNumericAmount(value: string): number {
  const parsed = Number(value.trim());
  if (Number.isNaN(parsed)) {
    return 0;
  }

  return parsed;
}

function totalCounterpartyVolume(
  counterparty: WalletRelatedAddressViewModel,
): number {
  return (
    parseNumericAmount(counterparty.inboundAmount) +
    parseNumericAmount(counterparty.outboundAmount)
  );
}

function matchesTokenFilter(
  counterparty: WalletRelatedAddressViewModel,
  tokenFilter: string,
): boolean {
  if (tokenFilter === "all") {
    return true;
  }

  return counterparty.tokenBreakdowns.some(
    (token) => token.symbol === tokenFilter,
  );
}

function buildRelatedAddressTokenFilters(
  items: WalletRelatedAddressViewModel[],
): string[] {
  const tokens = new Set<string>();
  for (const item of items) {
    for (const token of item.tokenBreakdowns) {
      const normalized = token.symbol.trim();
      if (normalized) {
        tokens.add(normalized);
      }
    }
  }

  return [...tokens].sort((left, right) => left.localeCompare(right));
}

function looksLikeWalletAddress(value: string): boolean {
  const trimmed = value.trim();
  return trimmed.startsWith("0x") || trimmed.length > 36;
}

function compactAddress(address: string): string {
  const trimmed = address.trim();
  if (trimmed.length <= 20) {
    return trimmed;
  }

  return `${trimmed.slice(0, 10)}...${trimmed.slice(-8)}`;
}

function formatNetWorthUsd(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "Unavailable";
  }

  return `$${trimmed}`;
}

function formatEnrichmentValue(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "Unavailable";
  }

  return trimmed;
}

function formatEnrichmentProvider(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "Enrichment";
  }

  return trimmed.charAt(0).toUpperCase() + trimmed.slice(1);
}

function formatEnrichmentSource(value: string): string {
  if (value === "cache") {
    return "Cached snapshot";
  }
  if (value === "live") {
    return "Live lookup";
  }

  return formatEnrichmentValue(value);
}

function formatHoldingUsdValue(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "Unavailable";
  }

  return `$${trimmed}`;
}

function formatHoldingAllocation(value: number): string {
  if (!Number.isFinite(value) || value <= 0) {
    return "Allocation pending";
  }

  return `${value.toFixed(1)}% of wallet`;
}

function formatObservedAt(value: string): string {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return "Pending";
  }

  return new Date(parsed).toISOString().slice(0, 10);
}

function formatScoreLabel(name: string): string {
  if (name === "cluster_score") {
    return "Cluster score";
  }
  if (name === "shadow_exit_risk") {
    return "Shadow exit risk";
  }
  if (name === "alpha_score") {
    return "Alpha score";
  }

  return name.replaceAll("_", " ");
}

function formatSignalLabel(name: string): string {
  return formatScoreLabel(name);
}

function formatRelativeTime(value: string): string {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return "Warming up";
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

  return formatObservedAt(value);
}

function formatCoverageWindow(indexing: {
  status: "ready" | "indexing";
  coverageWindowDays: number;
}): string {
  if (indexing.status === "indexing" || indexing.coverageWindowDays <= 0) {
    return "Warming up";
  }

  return `${indexing.coverageWindowDays} days`;
}

function renderCoverageRange(indexing: WalletIndexingViewModel): string {
  if (!indexing.coverageStartAt || !indexing.coverageEndAt) {
    return "Historical coverage is still being filled.";
  }

  return `Observed range ${formatObservedAt(indexing.coverageStartAt)} -> ${formatObservedAt(indexing.coverageEndAt)}`;
}

function formatRelatedAddressCoverageLabel(
  shownCount: number,
  indexedCount: number,
): string {
  if (indexedCount > 0) {
    return `Showing ${shownCount} of ${indexedCount} indexed`;
  }

  return `Showing ${shownCount} retrieved counterparties`;
}

function formatCounterpartyAmount(
  counterparty: WalletRelatedAddressViewModel,
): string {
  const token = counterparty.primaryToken.trim();
  const inbound = normalizeAmount(counterparty.inboundAmount);
  const outbound = normalizeAmount(counterparty.outboundAmount);

  if (token) {
    return `IN ${inbound} / OUT ${outbound} ${token}`;
  }

  return `IN ${inbound} / OUT ${outbound}`;
}

function formatCounterpartyTokenSummary(
  counterparty: WalletRelatedAddressViewModel,
): string {
  const primaryToken = counterparty.primaryToken.trim();
  if (!primaryToken) {
    return "token n/a";
  }

  if (counterparty.tokenBreakdownCount <= 1) {
    return primaryToken;
  }

  return `${primaryToken} +${counterparty.tokenBreakdownCount - 1}`;
}

function formatTokenBreakdownTotal(
  token: WalletRelatedAddressTokenBreakdownViewModel,
): string {
  const total =
    parseNumericAmount(token.inboundAmount) +
    parseNumericAmount(token.outboundAmount);

  return `${total.toFixed(6)} total`;
}

function buildCounterpartySummaryCopy(
  counterparty: WalletRelatedAddressViewModel,
): string {
  return [
    `Address: ${counterparty.address}`,
    `Direction: ${counterparty.directionLabel}`,
    `Interactions: ${counterparty.interactionCount}`,
    `In/Out count: ${counterparty.inboundCount}/${counterparty.outboundCount}`,
    `Amount: ${formatCounterpartyAmount(counterparty)}`,
    `First seen: ${counterparty.firstSeenAt || "n/a"}`,
    `Latest activity: ${counterparty.latestActivityAt}`,
  ].join(" | ");
}

export function mergeWalletGraphPreviews(
  current: WalletGraphPreview,
  expansion: WalletGraphPreview,
): WalletGraphPreview {
  const mergedNodes = dedupeGraphNodes([...current.nodes, ...expansion.nodes]);
  const mergedEdges = dedupeGraphEdges([...current.edges, ...expansion.edges]);

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
    neighborhoodSummary: buildMergedNeighborhoodSummary(
      mergedNodes,
      mergedEdges,
      current,
      expansion,
    ),
    nodes: mergedNodes,
    edges: mergedEdges,
  };
}

function dedupeGraphNodes(
  nodes: WalletGraphPreviewNode[],
): WalletGraphPreviewNode[] {
  const next = new Map<string, WalletGraphPreviewNode>();
  for (const node of nodes) {
    if (!next.has(node.id)) {
      next.set(node.id, node);
    }
  }

  return [...next.values()];
}

function dedupeGraphEdges(
  edges: WalletGraphPreviewEdge[],
): WalletGraphPreviewEdge[] {
  const next = new Map<string, WalletGraphPreviewEdge>();
  for (const edge of edges) {
    const key = `${edge.sourceId}:${edge.targetId}:${edge.kind}`;
    const current = next.get(key);
    if (!current) {
      next.set(key, edge);
      continue;
    }

    const maxWeight = Math.max(current.weight ?? 0, edge.weight ?? 0);
    const maxCounterpartyCount = Math.max(
      current.counterpartyCount ?? 0,
      edge.counterpartyCount ?? 0,
    );
    const observedAt = current.observedAt ?? edge.observedAt;

    next.set(key, {
      ...current,
      ...(maxWeight > 0 ? { weight: maxWeight } : {}),
      ...(maxCounterpartyCount > 0
        ? { counterpartyCount: maxCounterpartyCount }
        : {}),
      ...(observedAt ? { observedAt } : {}),
    });
  }

  return [...next.values()];
}

function buildMergedNeighborhoodSummary(
  nodes: WalletGraphPreviewNode[],
  edges: WalletGraphPreviewEdge[],
  current: WalletGraphPreview,
  expansion: WalletGraphPreview,
) {
  const latestObservedAt = [
    current.neighborhoodSummary.latestObservedAt,
    expansion.neighborhoodSummary.latestObservedAt,
    ...edges.map((edge) => edge.observedAt),
  ]
    .filter((value): value is string => Boolean(value))
    .sort()
    .at(-1);

  const baseSummary = {
    neighborNodeCount: Math.max(nodes.length - 1, 0),
    walletNodeCount: nodes.filter((node) => node.kind === "wallet").length,
    clusterNodeCount: nodes.filter((node) => node.kind === "cluster").length,
    entityNodeCount: nodes.filter((node) => node.kind === "entity").length,
    interactionEdgeCount: edges.length,
    totalInteractionWeight: edges.reduce(
      (sum, edge) => sum + (edge.weight ?? edge.counterpartyCount ?? 1),
      0,
    ),
  };

  return latestObservedAt ? { ...baseSummary, latestObservedAt } : baseSummary;
}

export function resolveGraphExpansionState({
  selectedNode,
  expandedGraphNeighborhoodKeys,
  graphNodeCount,
  graphNodes = [],
  relatedAddresses = [],
}: {
  selectedNode: WalletGraphNodeViewModel | null;
  expandedGraphNeighborhoodKeys: string[];
  graphNodeCount: number;
  graphNodes?: WalletGraphNodeViewModel[];
  relatedAddresses?: WalletRelatedAddressViewModel[];
}): WalletGraphExpansionState {
  const hopsUsed = expandedGraphNeighborhoodKeys.length;
  const budgetLabel = `${hopsUsed} hops expanded · ${graphNodeCount}/${MAX_GRAPH_NODE_BUDGET} nodes visible`;

  if (!selectedNode) {
    return {
      canExpand: false,
      expansionKey: null,
      reason: "Select a wallet node to expand.",
      budgetLabel,
      hopsUsed,
      hopBudget: hopsUsed,
      nodeCount: graphNodeCount,
      nodeBudget: MAX_GRAPH_NODE_BUDGET,
    };
  }

  const expansionKey = resolveGraphExpansionKey(selectedNode);
  if (!expansionKey) {
    return {
      canExpand: false,
      expansionKey: null,
      reason: "This node cannot be expanded.",
      budgetLabel,
      hopsUsed,
      hopBudget: hopsUsed,
      nodeCount: graphNodeCount,
      nodeBudget: MAX_GRAPH_NODE_BUDGET,
    };
  }

  if (expandedGraphNeighborhoodKeys.includes(expansionKey)) {
    return {
      canExpand: false,
      expansionKey,
      reason: describeExpandedGraphNodeReason(selectedNode.kind),
      budgetLabel,
      hopsUsed,
      hopBudget: hopsUsed,
      nodeCount: graphNodeCount,
      nodeBudget: MAX_GRAPH_NODE_BUDGET,
    };
  }

  if (graphNodeCount >= MAX_GRAPH_NODE_BUDGET) {
    return {
      canExpand: false,
      expansionKey,
      reason: "Visible node budget reached.",
      budgetLabel,
      hopsUsed,
      hopBudget: hopsUsed,
      nodeCount: graphNodeCount,
      nodeBudget: MAX_GRAPH_NODE_BUDGET,
    };
  }

  if (
    selectedNode.kind === "entity" &&
    !hasExpandableEntityWallets({
      selectedNode,
      graphNodes,
      relatedAddresses,
    })
  ) {
    return {
      canExpand: false,
      expansionKey,
      reason: "No additional indexed wallets are linked to this entity.",
      budgetLabel,
      hopsUsed,
      hopBudget: hopsUsed,
      nodeCount: graphNodeCount,
      nodeBudget: MAX_GRAPH_NODE_BUDGET,
    };
  }

  return {
    canExpand: true,
    expansionKey,
    reason: describeGraphExpansionReason(selectedNode.kind),
    budgetLabel,
    hopsUsed,
    hopBudget: hopsUsed,
    nodeCount: graphNodeCount,
    nodeBudget: MAX_GRAPH_NODE_BUDGET,
  };
}

export function resolveExpandableGraphNodeIds({
  graphNodes,
  expandedGraphNeighborhoodKeys,
  graphNodeCount,
  relatedAddresses = [],
}: {
  graphNodes: WalletGraphNodeViewModel[];
  expandedGraphNeighborhoodKeys: string[];
  graphNodeCount: number;
  relatedAddresses?: WalletRelatedAddressViewModel[];
}): string[] {
  return graphNodes
    .filter(
      (node) =>
        resolveGraphExpansionState({
          selectedNode: node,
          expandedGraphNeighborhoodKeys,
          graphNodeCount,
          graphNodes,
          relatedAddresses,
        }).canExpand,
    )
    .map((node) => node.id);
}

async function expandGraphNode({
  node,
  graphNodes,
  relatedAddresses,
  rootRequest,
  requestHeaders,
}: {
  node: WalletGraphNodeViewModel;
  graphNodes: WalletGraphNodeViewModel[];
  relatedAddresses: WalletRelatedAddressViewModel[];
  rootRequest: WalletDetailRequest;
  requestHeaders?: HeadersInit;
}): Promise<WalletGraphPreview> {
  if (node.kind === "cluster") {
    const cluster = await loadClusterDetailPreview({
      request: { clusterId: resolveClusterNodeId(node.id) },
    });

    return buildClusterExpansionGraphPreview({
      cluster,
      selectedNode: node,
      graphNodes,
      rootRequest,
    });
  }

  if (node.kind === "entity") {
    return buildEntityExpansionGraphPreview({
      selectedNode: node,
      graphNodes,
      relatedAddresses,
      rootRequest,
    });
  }

  if (!node.chain || !node.address) {
    return createUnavailableExpansionGraphPreview(rootRequest);
  }

  const requestedGraph = await loadWalletGraphPreview({
    request: {
      chain: node.chain,
      address: node.address,
      depthRequested: DEFAULT_WALLET_GRAPH_DEPTH,
    },
    ...(requestHeaders ? { requestHeaders } : {}),
  });

  if (requestedGraph.mode === "live") {
    return requestedGraph;
  }

  const summary = await loadWalletSummaryPreview({
    request: {
      chain: node.chain,
      address: node.address,
    },
    ...(requestHeaders ? { requestHeaders } : {}),
  });

  if (summary.mode !== "live") {
    return createUnavailableExpansionGraphPreview(rootRequest);
  }

  return rebaseExpandedGraphRootNode(
    deriveWalletGraphPreviewFromSummary({
      request: {
        chain: node.chain,
        address: node.address,
        depthRequested: DEFAULT_WALLET_GRAPH_DEPTH,
      },
      summary,
      fallback: requestedGraph,
    }),
    node.id,
  );
}

function buildClusterExpansionGraphPreview({
  cluster,
  selectedNode,
  graphNodes,
  rootRequest,
}: {
  cluster: ClusterDetailPreview;
  selectedNode: WalletGraphNodeViewModel;
  graphNodes: WalletGraphNodeViewModel[];
  rootRequest: WalletDetailRequest;
}): WalletGraphPreview {
  if (cluster.mode === "unavailable") {
    return createUnavailableExpansionGraphPreview(rootRequest);
  }

  const clusterNodeId = selectedNode.id;
  const nodes: WalletGraphPreviewNode[] = [];
  const edges: WalletGraphPreviewEdge[] = [];

  for (const member of cluster.members) {
    const existingWalletNode =
      graphNodes.find(
        (graphNode) =>
          graphNode.kind === "wallet" &&
          graphNode.chain === member.chain &&
          graphNode.address?.toLowerCase() === member.address.toLowerCase(),
      ) ?? null;
    const memberNodeId =
      existingWalletNode?.id ??
      `wallet:${member.chain}:${member.address.toLowerCase()}`;

    nodes.push({
      id: memberNodeId,
      kind: "wallet",
      chain: member.chain,
      address: member.address,
      label: member.label,
    });
    edges.push({
      sourceId: memberNodeId,
      targetId: clusterNodeId,
      kind: "member_of",
      family: "derived",
      directionality: "linked",
      ...(member.latestActivityAt
        ? { observedAt: member.latestActivityAt }
        : {}),
      weight: member.interactionCount,
      counterpartyCount: member.interactionCount,
      evidence: {
        source: "cluster-detail-members",
        confidence: cluster.classification === "strong" ? "high" : "medium",
        summary: `${member.label} is listed as a ${member.role ?? "member"} of ${cluster.label}.`,
      },
    });
  }

  return {
    mode: "live",
    source: "live-api",
    route: cluster.route,
    chain: rootRequest.chain === "evm" ? "EVM" : "SOLANA",
    address: rootRequest.address,
    depthRequested: 1,
    depthResolved: 1,
    densityCapped: false,
    statusMessage: `Expanded cluster members from ${cluster.label}.`,
    neighborhoodSummary: buildPreviewNeighborhoodSummary(nodes, edges),
    nodes,
    edges,
  };
}

function buildEntityExpansionGraphPreview({
  selectedNode,
  graphNodes,
  relatedAddresses,
  rootRequest,
}: {
  selectedNode: WalletGraphNodeViewModel;
  graphNodes: WalletGraphNodeViewModel[];
  relatedAddresses: WalletRelatedAddressViewModel[];
  rootRequest: WalletDetailRequest;
}): WalletGraphPreview {
  const rootNode = graphNodes.find((node) => node.isPrimary) ?? null;
  const entityWallets = resolveExpandableEntityWallets({
    selectedNode,
    graphNodes,
    relatedAddresses,
  });

  if (
    !rootNode ||
    !rootNode.chain ||
    !rootNode.address ||
    !entityWallets.length
  ) {
    return createUnavailableExpansionGraphPreview(rootRequest);
  }

  const nodes: WalletGraphPreviewNode[] = [];
  const edges: WalletGraphPreviewEdge[] = [];

  for (const counterparty of entityWallets) {
    const existingWalletNode =
      graphNodes.find(
        (graphNode) =>
          graphNode.kind === "wallet" &&
          graphNode.chain?.toLowerCase() ===
            counterparty.chainLabel.toLowerCase() &&
          graphNode.address?.toLowerCase() ===
            counterparty.address.toLowerCase(),
      ) ?? null;
    const counterpartyChain =
      counterparty.chainLabel.toLowerCase() === "solana" ? "solana" : "evm";
    const counterpartyNodeId =
      existingWalletNode?.id ??
      `wallet:${counterpartyChain}:${counterparty.address.toLowerCase()}`;

    nodes.push({
      id: counterpartyNodeId,
      kind: "wallet",
      chain: counterpartyChain,
      address: counterparty.address,
      label: counterparty.entityLabel || counterparty.address,
    });
    edges.push({
      sourceId: rootNode.id,
      targetId: counterpartyNodeId,
      kind:
        counterparty.directionLabel === "inbound"
          ? "funded_by"
          : "interacted_with",
      family: counterparty.directionLabel === "inbound" ? "derived" : "base",
      directionality:
        counterparty.directionLabel === "inbound"
          ? "received"
          : counterparty.directionLabel === "outbound"
            ? "sent"
            : "mixed",
      observedAt: counterparty.latestActivityAt,
      weight: counterparty.interactionCount,
      counterpartyCount: counterparty.interactionCount,
      evidence: {
        source: "entity-summary-expansion",
        confidence:
          counterparty.interactionCount >= 8
            ? "high"
            : counterparty.interactionCount >= 3
              ? "medium"
              : "low",
        summary: `${counterparty.address} shares the ${selectedNode.label} entity assignment in indexed counterparties.`,
      },
      tokenFlow: {
        primaryToken: counterparty.primaryToken,
        inboundCount: counterparty.inboundCount,
        outboundCount: counterparty.outboundCount,
        inboundAmount: counterparty.inboundAmount,
        outboundAmount: counterparty.outboundAmount,
        breakdowns: counterparty.tokenBreakdowns.map((token) => ({
          symbol: token.symbol,
          inboundAmount: token.inboundAmount,
          outboundAmount: token.outboundAmount,
        })),
      },
    });
    edges.push({
      sourceId: counterpartyNodeId,
      targetId: selectedNode.id,
      kind: "entity_linked",
      family: "derived",
      directionality: "linked",
      evidence: {
        source: "entity-summary-expansion",
        confidence: "medium",
        summary: `Indexed counterparty assigned to ${selectedNode.label}.`,
      },
    });
  }

  return {
    mode: "live",
    source: "summary-derived",
    route:
      rootRequest.chain === "evm"
        ? "GET /v1/wallets/:chain/:address/graph"
        : "GET /v1/wallets/:chain/:address/graph",
    chain: rootRequest.chain === "evm" ? "EVM" : "SOLANA",
    address: rootRequest.address,
    depthRequested: 1,
    depthResolved: 1,
    densityCapped: false,
    statusMessage: `Expanded indexed wallets linked to ${selectedNode.label}.`,
    neighborhoodSummary: buildPreviewNeighborhoodSummary(nodes, edges),
    nodes,
    edges,
  };
}

function createUnavailableExpansionGraphPreview(
  request: WalletDetailRequest,
): WalletGraphPreview {
  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    route: "GET /v1/wallets/:chain/:address/graph",
    chain: request.chain === "evm" ? "EVM" : "SOLANA",
    address: request.address,
    depthRequested: 1,
    depthResolved: 0,
    densityCapped: false,
    statusMessage: "Expansion data is unavailable.",
    neighborhoodSummary: {
      neighborNodeCount: 0,
      walletNodeCount: 0,
      clusterNodeCount: 0,
      entityNodeCount: 0,
      interactionEdgeCount: 0,
      totalInteractionWeight: 0,
    },
    nodes: [],
    edges: [],
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

function buildPreviewNeighborhoodSummary(
  nodes: WalletGraphPreviewNode[],
  edges: WalletGraphPreviewEdge[],
): WalletGraphPreview["neighborhoodSummary"] {
  const latestObservedAt = edges
    .map((edge) => edge.observedAt)
    .filter((value): value is string => Boolean(value))
    .sort()
    .at(-1);

  return {
    neighborNodeCount: Math.max(nodes.length, 0),
    walletNodeCount: nodes.filter((node) => node.kind === "wallet").length,
    clusterNodeCount: nodes.filter((node) => node.kind === "cluster").length,
    entityNodeCount: nodes.filter((node) => node.kind === "entity").length,
    interactionEdgeCount: edges.filter((edge) => edge.kind !== "entity_linked")
      .length,
    totalInteractionWeight: edges.reduce(
      (sum, edge) => sum + (edge.weight ?? edge.counterpartyCount ?? 1),
      0,
    ),
    ...(latestObservedAt ? { latestObservedAt } : {}),
  };
}

function resolveGraphExpansionKey(
  selectedNode: WalletGraphNodeViewModel,
): string | null {
  if (
    selectedNode.kind === "wallet" &&
    selectedNode.chain &&
    selectedNode.address
  ) {
    return `${selectedNode.chain}:${selectedNode.address.toLowerCase()}`;
  }

  if (selectedNode.kind === "cluster") {
    return `cluster:${resolveClusterNodeId(selectedNode.id)}`;
  }

  if (selectedNode.kind === "entity") {
    return `entity:${resolveEntityNodeKey(selectedNode)}`;
  }

  return null;
}

function describeGraphExpansionReason(
  kind: WalletGraphNodeViewModel["kind"],
): string {
  if (kind === "cluster") {
    return "Show cluster members around this node.";
  }

  if (kind === "entity") {
    return "Show indexed wallets linked to this entity.";
  }

  return "Expand the next hop from this wallet.";
}

function describeExpandedGraphNodeReason(
  kind: WalletGraphNodeViewModel["kind"],
): string {
  if (kind === "cluster") {
    return "This cluster already has its member expansion loaded.";
  }

  if (kind === "entity") {
    return "This entity already has its linked wallets loaded.";
  }

  return "This wallet already has its next hop loaded.";
}

function resolveClusterNodeId(nodeId: string): string {
  return nodeId.startsWith("cluster:")
    ? nodeId.slice("cluster:".length)
    : nodeId;
}

function resolveEntityNodeKey(
  node: Pick<WalletGraphNodeViewModel, "id" | "label">,
): string {
  return node.id.startsWith("entity:")
    ? node.id.slice("entity:".length)
    : node.label.toLowerCase();
}

function hasExpandableEntityWallets({
  selectedNode,
  graphNodes,
  relatedAddresses,
}: {
  selectedNode: WalletGraphNodeViewModel;
  graphNodes: WalletGraphNodeViewModel[];
  relatedAddresses: WalletRelatedAddressViewModel[];
}): boolean {
  return (
    resolveExpandableEntityWallets({
      selectedNode,
      graphNodes,
      relatedAddresses,
    }).length > 0
  );
}

function resolveExpandableEntityWallets({
  selectedNode,
  graphNodes,
  relatedAddresses,
}: {
  selectedNode: WalletGraphNodeViewModel;
  graphNodes: WalletGraphNodeViewModel[];
  relatedAddresses: WalletRelatedAddressViewModel[];
}): WalletRelatedAddressViewModel[] {
  if (selectedNode.kind !== "entity") {
    return [];
  }

  const entityKey = resolveEntityNodeKey(selectedNode);
  const visibleWalletKeys = new Set(
    graphNodes
      .filter(
        (
          node,
        ): node is WalletGraphNodeViewModel & {
          chain: "evm" | "solana";
          address: string;
        } =>
          node.kind === "wallet" &&
          Boolean(node.chain) &&
          Boolean(node.address),
      )
      .map((node) => `${node.chain}:${node.address.toLowerCase()}`),
  );

  return relatedAddresses.filter((counterparty) => {
    const counterpartyChain =
      counterparty.chainLabel.toLowerCase() === "solana" ? "solana" : "evm";
    const walletKey = `${counterpartyChain}:${counterparty.address.toLowerCase()}`;
    const matchesEntity =
      counterparty.entityKey.toLowerCase() === entityKey.toLowerCase() ||
      counterparty.entityLabel.toLowerCase() ===
        selectedNode.label.toLowerCase();

    return matchesEntity && !visibleWalletKeys.has(walletKey);
  });
}

export function resolveSelectedGraphEntityContext({
  selectedNode,
  graphNodes,
  graphEdges,
  relatedAddresses = [],
}: {
  selectedNode: WalletGraphNodeViewModel | null;
  graphNodes: WalletGraphNodeViewModel[];
  graphEdges: WalletGraphEdgeViewModel[];
  relatedAddresses?: WalletRelatedAddressViewModel[];
}): WalletGraphEntityContextViewModel | null {
  if (!selectedNode) {
    return null;
  }

  const entityAssignmentIndex = buildGraphEntityAssignmentIndex(
    graphNodes,
    graphEdges,
  );

  if (selectedNode.kind !== "entity") {
    const linkedEntities = entityAssignmentIndex.get(selectedNode.id) ?? [];
    if (linkedEntities.length) {
      return {
        label: "Linked entities",
        helperCopy:
          "Named entities directly linked to the selected node in the current neighborhood.",
        links: linkedEntities.map((assignment) => ({
          id: assignment.entityNodeId,
          label: assignment.entityLabel,
          kindLabel: "entity",
          tone: "amber",
          href: assignment.entityHref,
          sourceLabel: assignment.sourceLabel,
          sourceTone: assignment.sourceTone,
        })),
      };
    }

    if (
      selectedNode.kind !== "wallet" ||
      !selectedNode.chain ||
      !selectedNode.address
    ) {
      return null;
    }

    const selectedNodeAddress = selectedNode.address;
    const summaryCounterparty =
      relatedAddresses.find(
        (counterparty) =>
          counterparty.chainLabel.toLowerCase() ===
            selectedNode.chain?.toLowerCase() &&
          counterparty.address.toLowerCase() ===
            selectedNodeAddress.toLowerCase(),
      ) ?? null;
    const fallbackAssignment =
      summaryCounterparty &&
      buildFallbackEntityAssignment(
        summaryCounterparty.entityKey,
        summaryCounterparty.entityLabel,
      );
    if (!fallbackAssignment) {
      return null;
    }

    return {
      label: "Linked entities",
      helperCopy:
        "Named entity context derived from the indexed wallet summary while the visible neighborhood warms up.",
      links: [
        {
          id: fallbackAssignment.entityNodeId,
          label: fallbackAssignment.entityLabel,
          kindLabel: "entity",
          tone: "amber",
          href: fallbackAssignment.entityHref,
          sourceLabel: fallbackAssignment.sourceLabel,
          sourceTone: fallbackAssignment.sourceTone,
        },
      ],
    };
  }

  const linkedNodes: WalletGraphEntityLinkViewModel[] = [];
  for (const edge of graphEdges) {
    if (
      edge.kind !== "entity_linked" ||
      (edge.sourceId !== selectedNode.id && edge.targetId !== selectedNode.id)
    ) {
      continue;
    }

    const linkedNodeId =
      edge.sourceId === selectedNode.id ? edge.targetId : edge.sourceId;
    const linkedNode = graphNodes.find((node) => node.id === linkedNodeId);
    if (!linkedNode) {
      continue;
    }

    const source = edge.evidence?.source ?? "entity-linked";
    linkedNodes.push({
      id: linkedNode.id,
      label: linkedNode.label,
      kindLabel: linkedNode.kindLabel,
      tone: linkedNode.tone,
      href: buildSelectedGraphNodeHref(linkedNode),
      sourceLabel: formatEntityAssignmentSource(source),
      sourceTone: toneForEntityAssignmentSource(source),
    });
  }
  linkedNodes.sort((left, right) => left.label.localeCompare(right.label));

  if (!linkedNodes.length) {
    return null;
  }

  return {
    label: "Entity linkage",
    helperCopy:
      "Indexed entity label connected to visible wallets or clusters in this neighborhood.",
    links: linkedNodes,
  };
}

function normalizeAmount(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "0";
  }

  return trimmed;
}

function formatPercent(value: number): string {
  const normalized = value > 1 ? value : value * 100;
  return `${Math.round(normalized)}%`;
}
