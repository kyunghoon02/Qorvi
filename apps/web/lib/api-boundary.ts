import {
  mergeRequestHeaders,
  readClientForwardedAuthHeaders,
} from "./request-headers";

export type WalletSummaryScorePreview = {
  name: string;
  value: number;
  rating: "low" | "medium" | "high";
  tone: "teal" | "amber" | "violet" | "emerald";
  clusterBreakdown?: WalletSummaryClusterScoreBreakdownPreview;
};

export type WalletSummaryClusterScoreBreakdownPreview = {
  peerWalletOverlap: number;
  sharedEntityLinks: number;
  bidirectionalPeerFlows: number;
  contradictionPenalty: number;
  suppressionDiscount: number;
  samplingApplied: boolean;
  sourceDensityCapped: boolean;
  sourceNodeCount: number;
  sourceEdgeCount: number;
  analysisNodeCount: number;
  analysisEdgeCount: number;
  contradictionReasons: string[];
  suppressionReasons: string[];
};

export type WalletSummaryCounterpartyPreview = {
  chain: "evm" | "solana";
  chainLabel: string;
  address: string;
  entityKey?: string;
  entityType?: string;
  entityLabel?: string;
  interactionCount: number;
  inboundCount: number;
  outboundCount: number;
  inboundAmount: string;
  outboundAmount: string;
  primaryToken: string;
  tokenBreakdowns: WalletSummaryCounterpartyTokenPreview[];
  directionLabel: string;
  firstSeenAt: string;
  latestActivityAt: string;
};

export type WalletSummaryCounterpartyTokenPreview = {
  symbol: string;
  inboundAmount: string;
  outboundAmount: string;
};

export type WalletSummaryRecentFlowPreview = {
  incomingTxCount7d: number;
  outgoingTxCount7d: number;
  incomingTxCount30d: number;
  outgoingTxCount30d: number;
  netDirection7d: string;
  netDirection30d: string;
};

export type WalletSummaryEnrichmentPreview = {
  provider: string;
  netWorthUsd: string;
  nativeBalance: string;
  nativeBalanceFormatted: string;
  activeChains: string[];
  activeChainCount: number;
  holdings: WalletSummaryHoldingPreview[];
  holdingCount: number;
  source: string;
  updatedAt: string;
};

export type WalletSummaryLatestSignalPreview = {
  name: string;
  value: number;
  rating: "low" | "medium" | "high";
  label: string;
  source: string;
  observedAt: string;
};

export type WalletSummaryHoldingPreview = {
  symbol: string;
  tokenAddress: string;
  balance: string;
  balanceFormatted: string;
  valueUsd: string;
  portfolioPercentage: number;
  isNative: boolean;
};

export type WalletSummaryIndexingPreview = {
  status: "ready" | "indexing";
  lastIndexedAt: string;
  coverageStartAt: string;
  coverageEndAt: string;
  coverageWindowDays: number;
};

export type WalletSummaryPreview = {
  mode: "unavailable" | "live";
  source: "boundary-unavailable" | "live-api";
  route: string;
  chain: "EVM" | "SOLANA";
  chainLabel: string;
  address: string;
  label: string;
  clusterId?: string;
  counterparties: number;
  statusMessage: string;
  topCounterparties: WalletSummaryCounterpartyPreview[];
  recentFlow: WalletSummaryRecentFlowPreview;
  enrichment?: WalletSummaryEnrichmentPreview;
  indexing: WalletSummaryIndexingPreview;
  latestSignals: WalletSummaryLatestSignalPreview[];
  scores: WalletSummaryScorePreview[];
};

export type FindingPreview = {
  id: string;
  type: string;
  subjectType: string;
  chain?: string;
  address?: string;
  key?: string;
  label?: string;
  summary: string;
  importanceReason: string[];
  observedFacts: string[];
  inferredInterpretations: string[];
  confidence: number;
  importanceScore: number;
  observedAt: string;
  coverageStartAt?: string;
  coverageEndAt?: string;
  coverageWindowDays: number;
  evidence: Array<{
    type: string;
    value?: string;
    confidence?: number;
    observedAt?: string;
    metadata?: Record<string, unknown>;
  }>;
  nextWatch: Array<{
    subjectType: string;
    chain?: string;
    address?: string;
    token?: string;
    label?: string;
    metadata?: Record<string, unknown>;
  }>;
};

export type FindingsFeedPreview = {
  mode: "unavailable" | "live";
  source: "boundary-unavailable" | "live-api";
  route: string;
  generatedAt: string;
  statusMessage: string;
  items: FindingPreview[];
  nextCursor?: string;
  hasMore: boolean;
};

export type WalletLabelPreview = {
  key: string;
  name: string;
  class: "verified" | "inferred" | "behavioral";
  entityType: string;
  source: string;
  confidence: number;
  evidenceSummary: string;
  observedAt: string;
};

export type WalletBriefPreview = {
  mode: "unavailable" | "live";
  source: "boundary-unavailable" | "live-api";
  route: string;
  chain: "evm" | "solana";
  address: string;
  displayName: string;
  statusMessage: string;
  aiSummary: string;
  keyFindings: FindingPreview[];
  verifiedLabels: WalletLabelPreview[];
  probableLabels: WalletLabelPreview[];
  behavioralLabels: WalletLabelPreview[];
  topCounterparties: WalletSummaryCounterpartyPreview[];
  recentFlow: WalletSummaryRecentFlowPreview;
  enrichment?: WalletSummaryEnrichmentPreview;
  indexing: WalletSummaryIndexingPreview;
  latestSignals: WalletSummaryLatestSignalPreview[];
  scores: WalletSummaryScorePreview[];
};

export type AnalystWalletExplanationPreview = {
  chain: "evm" | "solana";
  address: string;
  source: string;
  cached: boolean;
  model?: string;
  promptVersion: string;
  summary: string;
  evidence?: string[];
  inference?: string[];
  unknowns?: string[];
  disconfirmers?: string[];
  whyItMatters: string[];
  confidenceNote: string;
  watchNext: string[];
  cooldownSecondsRemaining?: number;
  queued?: boolean;
};

export type AnalystWalletAnalyzeEvidenceRefPreview = {
  kind: string;
  key?: string;
  label?: string;
  route?: string;
  metadata?: Record<string, unknown>;
};

export type AnalystWalletAnalyzeRecentTurnInput = {
  question?: string;
  headline?: string;
  toolTrace?: string[];
  evidenceRefs?: AnalystWalletAnalyzeEvidenceRefPreview[];
};

export type AnalystWalletAnalyzePreview = {
  chain: "evm" | "solana";
  address: string;
  question: string;
  contextReused: boolean;
  recentTurnCount: number;
  headline: string;
  conclusion: string[];
  confidence: "low" | "medium" | "high";
  observedFacts: string[];
  inferredInterpretations: string[];
  alternativeExplanations: string[];
  nextSteps: string[];
  toolTrace: string[];
  evidenceRefs: AnalystWalletAnalyzeEvidenceRefPreview[];
};

export type AnalystEntityAnalyzeRecentTurnInput =
  AnalystWalletAnalyzeRecentTurnInput;

export type AnalystEntityAnalyzePreview = {
  entityKey: string;
  displayName: string;
  question: string;
  contextReused: boolean;
  recentTurnCount: number;
  headline: string;
  conclusion: string[];
  confidence: "low" | "medium" | "high";
  observedFacts: string[];
  inferredInterpretations: string[];
  alternativeExplanations: string[];
  nextSteps: string[];
  toolTrace: string[];
  evidenceRefs: AnalystWalletAnalyzeEvidenceRefPreview[];
};

export type EntityInterpretationMemberPreview = {
  chain: "evm" | "solana";
  address: string;
  displayName: string;
  latestActivityAt?: string;
  verifiedLabels: WalletLabelPreview[];
  probableLabels: WalletLabelPreview[];
  behavioralLabels: WalletLabelPreview[];
};

export type EntityInterpretationPreview = {
  mode: "unavailable" | "live";
  source: "boundary-unavailable" | "live-api";
  route: string;
  entityKey: string;
  entityType: string;
  displayName: string;
  walletCount: number;
  latestActivityAt?: string;
  statusMessage: string;
  members: EntityInterpretationMemberPreview[];
  findings: FindingPreview[];
};

export function shouldPollIndexedWalletSummary(
  preview: WalletSummaryPreview,
): boolean {
  return preview.indexing.status === "indexing";
}

const walletSummaryStaleRefreshAfterMs = 30 * 60 * 1000;

export function shouldQueueWalletSummaryStaleRefresh(
  preview: WalletSummaryPreview,
  now = Date.now(),
): boolean {
  if (preview.mode !== "live") {
    return false;
  }
  if (preview.indexing.status === "indexing") {
    return false;
  }
  const lastIndexedAt = Date.parse(preview.indexing.lastIndexedAt);
  if (Number.isNaN(lastIndexedAt)) {
    return false;
  }

  return now - lastIndexedAt >= walletSummaryStaleRefreshAfterMs;
}

export type ClusterDetailMemberPreview = {
  chain: "evm" | "solana";
  address: string;
  label: string;
  interactionCount: number;
  latestActivityAt?: string;
  role?: string;
};

export type ClusterDetailEvidencePreview = {
  kind: string;
  label: string;
  source: string;
  confidence: number;
  observedAt: string;
  metadata?: Record<string, unknown>;
};

export type ClusterDetailActionPreview = {
  label: string;
  description: string;
  href?: string;
};

export type ClusterDetailPreview = {
  mode: "unavailable" | "live";
  source: "boundary-unavailable" | "live-api";
  route: string;
  clusterId: string;
  label: string;
  clusterType: string;
  classification: "strong" | "weak" | "emerging";
  score: number;
  memberCount: number;
  members: ClusterDetailMemberPreview[];
  commonActions: ClusterDetailActionPreview[];
  evidence: ClusterDetailEvidencePreview[];
  statusMessage: string;
};

export type ShadowExitFeedPreview = {
  mode: "unavailable" | "live";
  source: "boundary-unavailable" | "live-api";
  route: string;
  windowLabel: string;
  itemCount: number;
  highPriorityCount: number;
  latestObservedAt: string;
  statusMessage: string;
  items: ShadowExitFeedPreviewItem[];
};

export type ShadowExitFeedPreviewItem = {
  walletId: string;
  chain: "evm" | "solana";
  chainLabel: string;
  address: string;
  label: string;
  clusterId?: string;
  score: number;
  rating: "low" | "medium" | "high";
  scoreTone: "teal" | "amber" | "violet" | "emerald";
  reviewLabel: string;
  observedAt: string;
  explanation: string;
  walletHref: string;
  clusterHref?: string;
  evidence: ClusterDetailEvidencePreview[];
};

export type FirstConnectionFeedPreview = {
  mode: "unavailable" | "live";
  source: "boundary-unavailable" | "live-api";
  route: string;
  sort: "latest" | "score";
  windowLabel: string;
  itemCount: number;
  highPriorityCount: number;
  latestObservedAt: string;
  statusMessage: string;
  items: FirstConnectionFeedPreviewItem[];
};

export type FirstConnectionFeedPreviewItem = {
  walletId: string;
  chain: "evm" | "solana";
  chainLabel: string;
  address: string;
  label: string;
  clusterId?: string;
  score: number;
  rating: "low" | "medium" | "high";
  scoreTone: "teal" | "amber" | "violet" | "emerald";
  reviewLabel: string;
  observedAt: string;
  explanation: string;
  walletHref: string;
  clusterHref?: string;
  evidence: ClusterDetailEvidencePreview[];
};

export type WalletGraphPreview = {
  mode: "unavailable" | "live";
  source: "boundary-unavailable" | "live-api" | "summary-derived";
  route: string;
  chain: "EVM" | "SOLANA";
  address: string;
  depthRequested: number;
  depthResolved: number;
  densityCapped: boolean;
  statusMessage: string;
  snapshot?: WalletGraphSnapshotPreview;
  neighborhoodSummary: WalletGraphNeighborhoodSummaryPreview;
  nodes: WalletGraphPreviewNode[];
  edges: WalletGraphPreviewEdge[];
};

export type WalletGraphSnapshotPreview = {
  key: string;
  source: string;
  generatedAt: string;
  maxAgeSeconds: number;
};

export type WalletGraphNeighborhoodSummaryPreview = {
  neighborNodeCount: number;
  walletNodeCount: number;
  clusterNodeCount: number;
  entityNodeCount: number;
  interactionEdgeCount: number;
  totalInteractionWeight: number;
  latestObservedAt?: string;
};

export type WalletGraphPreviewNode = {
  id: string;
  kind: "wallet" | "cluster" | "entity";
  label: string;
  chain?: "evm" | "solana";
  address?: string;
};

export type WalletGraphPreviewEdge = {
  sourceId: string;
  targetId: string;
  kind: "member_of" | "interacted_with" | "funded_by" | "entity_linked";
  family: "base" | "derived";
  directionality?: "linked" | "sent" | "received" | "mixed";
  observedAt?: string;
  weight?: number;
  counterpartyCount?: number;
  evidence?: {
    source: string;
    confidence: "low" | "medium" | "high";
    summary: string;
    lastTxHash?: string;
    lastDirection?: string;
    lastProvider?: string;
  };
  tokenFlow?: {
    primaryToken?: string;
    inboundCount?: number;
    outboundCount?: number;
    inboundAmount?: string;
    outboundAmount?: string;
    breakdowns?: Array<{
      symbol: string;
      inboundAmount?: string;
      outboundAmount?: string;
    }>;
  };
};

export type SearchPreview = {
  mode: "unavailable" | "live";
  source: "boundary-unavailable" | "live-api";
  route: string;
  query: string;
  inputKind: string;
  kindLabel: string;
  chainLabel: string | undefined;
  title: string;
  explanation: string;
  walletRoute?: string;
  navigation: boolean;
};

export type AlertCenterPreview = {
  mode: "unavailable" | "live";
  source: "boundary-unavailable" | "live-api";
  inboxRoute: string;
  rulesRoute: string;
  channelsRoute: string;
  activeSeverityFilter: "all" | "low" | "medium" | "high" | "critical";
  activeSignalFilter:
    | "all"
    | "cluster_score"
    | "shadow_exit"
    | "first_connection";
  activeStatusFilter: "all" | "unread";
  statusMessage: string;
  nextCursor?: string | undefined;
  hasMore: boolean;
  unreadCount: number;
  inbox: AlertCenterInboxItemPreview[];
  rules: AlertCenterRulePreview[];
  channels: AlertCenterChannelPreview[];
};

export type AlertCenterInboxItemPreview = {
  id: string;
  alertRuleId: string;
  signalType: "cluster_score" | "shadow_exit" | "first_connection";
  severity: "low" | "medium" | "high" | "critical";
  observedAt: string;
  createdAt: string;
  isRead: boolean;
  readAt?: string;
  title: string;
  explanation: string;
  scoreValue?: number;
};

export type AlertCenterRulePreview = {
  id: string;
  name: string;
  ruleType: string;
  isEnabled: boolean;
  cooldownSeconds: number;
  eventCount: number;
  lastTriggeredAt?: string;
  watchlistId: string;
  signalTypes: string[];
  minimumSeverity: "low" | "medium" | "high" | "critical";
  renotifyOnSeverityIncrease: boolean;
  tags: string[];
  snoozeUntil?: string;
};

export type AlertCenterChannelPreview = {
  id: string;
  label: string;
  channelType: "email" | "discord_webhook" | "telegram";
  target: string;
  isEnabled: boolean;
  metadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
};

export type AdminConsoleLabelPreview = {
  id: string;
  name: string;
  description: string;
  color: string;
  createdBy: string;
  updatedAt: string;
};

export type AdminConsoleSuppressionPreview = {
  id: string;
  scope: string;
  target: string;
  reason: string;
  createdBy: string;
  active: boolean;
  updatedAt: string;
  expiresAt?: string | undefined;
};

export type AdminConsoleQuotaPreview = {
  provider: string;
  status: "healthy" | "warning" | "critical" | "exhausted";
  limit: number;
  used: number;
  reserved: number;
  windowLabel: string;
  lastCheckedAt: string;
};

export type AdminConsoleObservabilityProviderPreview = {
  provider: string;
  status: "healthy" | "warning" | "critical" | "unavailable";
  used24h: number;
  error24h: number;
  avgLatencyMs: number;
  lastSeenAt?: string | undefined;
};

export type AdminConsoleObservabilityIngestPreview = {
  lastBackfillAt?: string | undefined;
  lastWebhookAt?: string | undefined;
  freshnessSeconds: number;
  lagStatus: "healthy" | "warning" | "critical" | "unavailable";
};

export type AdminConsoleObservabilityAlertDeliveryPreview = {
  attempts24h: number;
  delivered24h: number;
  failed24h: number;
  retryableCount: number;
  lastFailureAt?: string | undefined;
};

export type AdminConsoleObservabilityWalletTrackingPreview = {
  candidateCount: number;
  trackedCount: number;
  labeledCount: number;
  scoredCount: number;
  staleCount: number;
  suppressedCount: number;
};

export type AdminConsoleObservabilityTrackingSubscriptionsPreview = {
  pendingCount: number;
  activeCount: number;
  erroredCount: number;
  pausedCount: number;
  lastEventAt?: string | undefined;
};

export type AdminConsoleObservabilityQueueDepthPreview = {
  defaultDepth: number;
  priorityDepth: number;
};

export type AdminConsoleObservabilityBackfillHealthPreview = {
  jobs24h: number;
  activities24h: number;
  transactions24h: number;
  expansions24h: number;
  lastSuccessAt?: string | undefined;
};

export type AdminConsoleObservabilityStaleRefreshPreview = {
  attempts24h: number;
  succeeded24h: number;
  productive24h: number;
  lastHitAt?: string | undefined;
};

export type AdminConsoleObservabilityRunPreview = {
  jobName: string;
  lastStatus: string;
  lastStartedAt: string;
  lastFinishedAt?: string | undefined;
  lastSuccessAt?: string | undefined;
  minutesSinceSuccess: number;
  lastError?: string | undefined;
};

export type AdminConsoleObservabilityFailurePreview = {
  source: string;
  kind: string;
  occurredAt: string;
  summary: string;
  details: Record<string, unknown>;
};

export type AdminConsoleDomesticPrelistingCandidatePreview = {
  chain: string;
  tokenAddress: string;
  tokenSymbol: string;
  normalizedAssetKey: string;
  transferCount7d: number;
  transferCount24h: number;
  activeWalletCount: number;
  trackedWalletCount: number;
  distinctCounterpartyCount: number;
  totalAmount: string;
  largestTransferAmount: string;
  latestObservedAt: string;
  listedOnUpbit: boolean;
  listedOnBithumb: boolean;
};

export type AdminConsoleObservabilityPreview = {
  providerUsage: AdminConsoleObservabilityProviderPreview[];
  ingest: AdminConsoleObservabilityIngestPreview;
  alertDelivery: AdminConsoleObservabilityAlertDeliveryPreview;
  walletTracking: AdminConsoleObservabilityWalletTrackingPreview;
  trackingSubscriptions: AdminConsoleObservabilityTrackingSubscriptionsPreview;
  queueDepth: AdminConsoleObservabilityQueueDepthPreview;
  backfillHealth: AdminConsoleObservabilityBackfillHealthPreview;
  staleRefresh: AdminConsoleObservabilityStaleRefreshPreview;
  recentRuns: AdminConsoleObservabilityRunPreview[];
  recentFailures: AdminConsoleObservabilityFailurePreview[];
};

export type AdminBacktestCheckPreview = {
  key: string;
  label: string;
  description: string;
  status: "ready" | "missing" | "not_configured";
  configured: boolean;
  path?: string | undefined;
};

export type AdminBacktestRunResultPreview = {
  key: string;
  label: string;
  status: "succeeded" | "failed";
  summary: string;
  executedAt: string;
  details: Record<string, unknown>;
};

export type AdminBacktestOpsPreview = {
  route: string;
  statusMessage: string;
  checks: AdminBacktestCheckPreview[];
  latestResult?: AdminBacktestRunResultPreview | undefined;
};

export type AdminConsoleCuratedListPreview = {
  id: string;
  name: string;
  notes: string;
  tags: string[];
  itemCount: number;
  items: Array<{
    id: string;
    itemType: string;
    itemKey: string;
    tags: string[];
    notes?: string | undefined;
    updatedAt: string;
  }>;
  updatedAt: string;
};

export type AdminConsoleAuditEntryPreview = {
  actor: string;
  action: string;
  targetType: string;
  targetKey: string;
  note?: string | undefined;
  createdAt: string;
};

export type AdminConsolePreview = {
  mode: "unavailable" | "live";
  source: "boundary-unavailable" | "live-api";
  labelsRoute: string;
  suppressionsRoute: string;
  quotasRoute: string;
  curatedListsRoute: string;
  auditLogsRoute: string;
  domesticPrelistingRoute: string;
  statusMessage: string;
  observabilityRoute: string;
  labels: AdminConsoleLabelPreview[];
  suppressions: AdminConsoleSuppressionPreview[];
  quotas: AdminConsoleQuotaPreview[];
  curatedLists: AdminConsoleCuratedListPreview[];
  auditLogs: AdminConsoleAuditEntryPreview[];
  domesticPrelisting: AdminConsoleDomesticPrelistingCandidatePreview[];
  observability: AdminConsoleObservabilityPreview;
  backtestOps: AdminBacktestOpsPreview;
};

type WalletSummaryApiScore = {
  name: string;
  value: number;
  rating: "low" | "medium" | "high";
  evidence?: Array<{
    kind: string;
    label: string;
    source: string;
    confidence: number;
    observedAt: string;
    metadata?: Record<string, unknown>;
  }>;
};

type WalletSummaryApiResponse = {
  chain: "evm" | "solana";
  address: string;
  displayName: string;
  clusterId?: string | null;
  counterparties: number;
  latestActivityAt: string;
  topCounterparties?: Array<{
    chain: "evm" | "solana";
    address: string;
    entityKey?: string;
    entityType?: string;
    entityLabel?: string;
    interactionCount: number;
    inboundCount?: number;
    outboundCount?: number;
    inboundAmount?: string;
    outboundAmount?: string;
    primaryToken?: string;
    tokenBreakdowns?: Array<{
      symbol: string;
      inboundAmount?: string;
      outboundAmount?: string;
    }>;
    directionLabel?: string;
    firstSeenAt?: string;
    latestActivityAt: string;
  }>;
  recentFlow?: {
    incomingTxCount7d: number;
    outgoingTxCount7d: number;
    incomingTxCount30d: number;
    outgoingTxCount30d: number;
    netDirection7d: string;
    netDirection30d: string;
  };
  enrichment?: {
    provider: string;
    netWorthUsd: string;
    nativeBalance: string;
    nativeBalanceFormatted: string;
    activeChains: string[];
    activeChainCount: number;
    holdings?: Array<{
      symbol: string;
      tokenAddress?: string;
      balance?: string;
      balanceFormatted?: string;
      valueUsd?: string;
      portfolioPercentage?: number;
      isNative?: boolean;
    }>;
    holdingCount?: number;
    source: string;
    updatedAt: string;
  };
  indexing?: {
    status: "ready" | "indexing";
    lastIndexedAt?: string;
    coverageStartAt?: string;
    coverageEndAt?: string;
    coverageWindowDays?: number;
  };
  latestSignals?: Array<{
    name: string;
    value: number;
    rating: "low" | "medium" | "high";
    label: string;
    source: string;
    observedAt: string;
  }>;
  tags: string[];
  scores: WalletSummaryApiScore[];
};

type WalletSummaryEnvelope = {
  success: boolean;
  data: WalletSummaryApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type WalletGraphApiResponse = {
  chain: "evm" | "solana";
  address: string;
  depthRequested: number;
  depthResolved: number;
  densityCapped: boolean;
  snapshot?: {
    key: string;
    source: string;
    generatedAt: string;
    maxAgeSeconds: number;
  };
  neighborhoodSummary?: {
    neighborNodeCount: number;
    walletNodeCount: number;
    clusterNodeCount: number;
    entityNodeCount: number;
    interactionEdgeCount: number;
    totalInteractionWeight: number;
    latestObservedAt?: string;
  };
  nodes?: Array<{
    id: string;
    kind: "wallet" | "cluster" | "entity";
    label: string;
    chain?: "evm" | "solana";
    address?: string;
  }>;
  edges?: Array<{
    sourceId: string;
    targetId: string;
    kind: "member_of" | "interacted_with" | "funded_by" | "entity_linked";
    family?: "base" | "derived";
    directionality?: "linked" | "sent" | "received" | "mixed";
    observedAt?: string;
    weight?: number;
    counterpartyCount?: number;
    evidence?: {
      source: string;
      confidence: "low" | "medium" | "high";
      summary: string;
      lastTxHash?: string;
      lastDirection?: string;
      lastProvider?: string;
    };
    tokenFlow?: {
      primaryToken?: string;
      inboundCount?: number;
      outboundCount?: number;
      inboundAmount?: string;
      outboundAmount?: string;
      breakdowns?: Array<{
        symbol: string;
        inboundAmount?: string;
        outboundAmount?: string;
      }>;
    };
  }>;
};

type WalletGraphEnvelope = {
  success: boolean;
  data: WalletGraphApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type ClusterDetailApiResponse = {
  id: string;
  label: string;
  clusterType: string;
  score: number;
  classification: "strong" | "weak" | "emerging";
  memberCount: number;
  members?: ClusterDetailMemberPreview[];
  commonActions?: ClusterDetailActionPreview[];
  evidence?: ClusterDetailEvidencePreview[];
};

type ClusterDetailEnvelope = {
  success: boolean;
  data: ClusterDetailApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type ShadowExitFeedApiItem = {
  walletId: string;
  chain: "evm" | "solana";
  address: string;
  label: string;
  clusterId?: string | null;
  score: number;
  rating: "low" | "medium" | "high";
  observedAt: string;
  explanation: string;
  evidence?: ClusterDetailEvidencePreview[];
};

type ShadowExitFeedApiResponse = {
  windowLabel: string;
  generatedAt: string;
  items: ShadowExitFeedApiItem[];
};

type ShadowExitFeedEnvelope = {
  success: boolean;
  data: ShadowExitFeedApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type FirstConnectionFeedApiItem = {
  walletId: string;
  chain: "evm" | "solana";
  address: string;
  label: string;
  clusterId?: string | null;
  score: number;
  rating: "low" | "medium" | "high";
  observedAt: string;
  explanation: string;
  evidence?: ClusterDetailEvidencePreview[];
};

type FirstConnectionFeedApiResponse = {
  sort?: "latest" | "score";
  windowLabel: string;
  generatedAt: string;
  items: FirstConnectionFeedApiItem[];
};

type FirstConnectionFeedEnvelope = {
  success: boolean;
  data: FirstConnectionFeedApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type SearchApiResult = {
  type: string;
  kind: string;
  kindLabel?: string;
  label: string;
  chain?: string;
  chainLabel?: string;
  walletRoute?: string;
  explanation: string;
  confidence: number;
  navigation: boolean;
};

type SearchApiResponse = {
  query: string;
  inputKind: string;
  explanation: string;
  results: SearchApiResult[];
};

type SearchEnvelope = {
  success: boolean;
  data: SearchApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type FindingsFeedApiResponse = {
  generatedAt: string;
  items: FindingPreview[];
  nextCursor?: string;
  hasMore: boolean;
};

type FindingsFeedEnvelope = {
  success: boolean;
  data: FindingsFeedApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

export type DiscoverFeaturedWalletSeedPreview = {
  chain: "evm" | "solana";
  address: string;
  displayName: string;
  description: string;
  category: string;
  tags: string[];
  provider?: string;
  confidence?: number;
  observedAt?: string;
};

export type DiscoverDomesticPrelistingCandidatePreview = {
  chain: "evm" | "solana";
  tokenAddress: string;
  tokenSymbol: string;
  normalizedAssetKey: string;
  transferCount7d: number;
  transferCount24h: number;
  activeWalletCount: number;
  trackedWalletCount: number;
  distinctCounterpartyCount: number;
  totalAmount: string;
  largestTransferAmount: string;
  latestObservedAt: string;
  representativeWalletChain?: "evm" | "solana";
  representativeWallet?: string;
  representativeLabel?: string;
};

type DiscoverFeaturedWalletApiResponse = {
  items: DiscoverFeaturedWalletSeedPreview[];
};

type DiscoverDomesticPrelistingApiResponse = {
  items: DiscoverDomesticPrelistingCandidatePreview[];
};

type DiscoverFeaturedWalletEnvelope = {
  success: boolean;
  data: DiscoverFeaturedWalletApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type DiscoverDomesticPrelistingEnvelope = {
  success: boolean;
  data: DiscoverDomesticPrelistingApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type WalletBriefApiResponse = {
  chain: "evm" | "solana";
  address: string;
  displayName: string;
  aiSummary: string;
  keyFindings?: FindingPreview[];
  verifiedLabels?: WalletLabelPreview[];
  probableLabels?: WalletLabelPreview[];
  behavioralLabels?: WalletLabelPreview[];
  topCounterparties?: Array<{
    chain: "evm" | "solana";
    address: string;
    entityKey?: string;
    entityType?: string;
    entityLabel?: string;
    interactionCount: number;
    inboundCount?: number;
    outboundCount?: number;
    inboundAmount?: string;
    outboundAmount?: string;
    primaryToken?: string;
    tokenBreakdowns?: Array<{
      symbol: string;
      inboundAmount?: string;
      outboundAmount?: string;
    }>;
    directionLabel?: string;
    firstSeenAt?: string;
    latestActivityAt: string;
  }>;
  recentFlow?: {
    incomingTxCount7d: number;
    outgoingTxCount7d: number;
    incomingTxCount30d: number;
    outgoingTxCount30d: number;
    netDirection7d: string;
    netDirection30d: string;
  };
  enrichment?: {
    provider: string;
    netWorthUsd: string;
    nativeBalance: string;
    nativeBalanceFormatted: string;
    activeChains: string[];
    activeChainCount: number;
    holdings?: Array<{
      symbol: string;
      tokenAddress?: string;
      balance?: string;
      balanceFormatted?: string;
      valueUsd?: string;
      portfolioPercentage?: number;
      isNative?: boolean;
    }>;
    holdingCount?: number;
    source: string;
    updatedAt: string;
  };
  indexing?: {
    status: "ready" | "indexing";
    lastIndexedAt?: string;
    coverageStartAt?: string;
    coverageEndAt?: string;
    coverageWindowDays?: number;
  };
  latestSignals?: Array<{
    name: string;
    value: number;
    rating: "low" | "medium" | "high";
    label: string;
    source: string;
    observedAt: string;
  }>;
  scores?: WalletSummaryApiScore[];
};

type WalletBriefEnvelope = {
  success: boolean;
  data: WalletBriefApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AnalystWalletExplanationEnvelope = {
  success: boolean;
  data: AnalystWalletExplanationPreview | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AnalystWalletAnalyzeEnvelope = {
  success: boolean;
  data: AnalystWalletAnalyzePreview | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AnalystEntityAnalyzeEnvelope = {
  success: boolean;
  data: AnalystEntityAnalyzePreview | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type EntityInterpretationApiResponse = {
  entityKey: string;
  entityType: string;
  displayName: string;
  walletCount: number;
  latestActivityAt?: string;
  members?: EntityInterpretationMemberPreview[];
  findings?: FindingPreview[];
};

type EntityInterpretationEnvelope = {
  success: boolean;
  data: EntityInterpretationApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AlertInboxApiItem = {
  id: string;
  alertRuleId: string;
  signalType: "cluster_score" | "shadow_exit" | "first_connection";
  severity: "low" | "medium" | "high" | "critical";
  payload?: Record<string, unknown>;
  observedAt: string;
  isRead?: boolean;
  readAt?: string;
  createdAt: string;
};

type AlertInboxApiResponse = {
  items: AlertInboxApiItem[];
  nextCursor?: string | undefined;
  hasMore?: boolean;
  unreadCount?: number;
};

type AlertRuleApiSummary = {
  id: string;
  name: string;
  ruleType: string;
  isEnabled: boolean;
  cooldownSeconds: number;
  eventCount: number;
  lastTriggeredAt?: string;
  definition: {
    watchlistId?: string;
    signalTypes?: string[];
    minimumSeverity?: "low" | "medium" | "high" | "critical";
    renotifyOnSeverityIncrease?: boolean;
    snoozeUntil?: string;
  };
  tags?: string[];
};

type AlertRuleApiDetail = AlertRuleApiSummary & {
  notes?: string;
};

type AlertRuleCollectionApiResponse = {
  items: AlertRuleApiSummary[];
};

type AlertDeliveryChannelApiItem = {
  id: string;
  label: string;
  channelType: "email" | "discord_webhook" | "telegram";
  target: string;
  metadata?: Record<string, unknown>;
  isEnabled: boolean;
  createdAt: string;
  updatedAt: string;
};

type AlertDeliveryChannelCollectionApiResponse = {
  items: AlertDeliveryChannelApiItem[];
};

type AlertInboxEnvelope = {
  success: boolean;
  data: AlertInboxApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AlertRuleCollectionEnvelope = {
  success: boolean;
  data: AlertRuleCollectionApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AlertDeliveryChannelCollectionEnvelope = {
  success: boolean;
  data: AlertDeliveryChannelCollectionApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AlertInboxMutationEnvelope = {
  success: boolean;
  data?: {
    event: AlertInboxApiItem;
  } | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AlertRuleDetailEnvelope = {
  success: boolean;
  data?: AlertRuleApiDetail | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type WatchlistApiItem = {
  id: string;
  itemType: string;
  chain: string;
  address: string;
  tags?: string[];
  note?: string;
  createdAt: string;
  updatedAt: string;
};

type WatchlistApiSummary = {
  id: string;
  name: string;
  itemCount: number;
  createdAt: string;
  updatedAt: string;
};

type WatchlistApiDetail = WatchlistApiSummary & {
  items: WatchlistApiItem[];
};

type WatchlistCollectionEnvelope = {
  success: boolean;
  data?: {
    items: WatchlistApiSummary[];
  } | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type WatchlistDetailEnvelope = {
  success: boolean;
  data?: WatchlistApiDetail | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AdminLabelApiItem = {
  id: string;
  name: string;
  description: string;
  color: string;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
};

type AdminSuppressionApiItem = {
  id: string;
  scope: string;
  target: string;
  reason: string;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
  expiresAt?: string;
  active: boolean;
};

type AdminQuotaApiItem = {
  provider: string;
  status: "healthy" | "warning" | "critical" | "exhausted";
  limit: number;
  used: number;
  reserved: number;
  windowStart: string;
  windowEnd: string;
  lastCheckedAt: string;
};

type AdminCuratedListApiItem = {
  id: string;
  name: string;
  notes?: string;
  tags?: string[];
  itemCount: number;
  items?: Array<{
    id: string;
    itemType: string;
    itemKey: string;
    tags?: string[];
    notes?: string;
    createdAt: string;
    updatedAt: string;
  }>;
  createdAt: string;
  updatedAt: string;
};

type AdminAuditLogApiItem = {
  actor: string;
  action: string;
  targetType: string;
  targetKey: string;
  note?: string;
  createdAt: string;
};

type AdminLabelCollectionApiResponse = {
  items: AdminLabelApiItem[];
};

type AdminSuppressionCollectionApiResponse = {
  items: AdminSuppressionApiItem[];
};

type AdminQuotaCollectionApiResponse = {
  items: AdminQuotaApiItem[];
};

type AdminObservabilityApiResponse = {
  providerUsage?: Array<{
    provider: string;
    status: "healthy" | "warning" | "critical" | "unavailable";
    used24h: number;
    error24h: number;
    avgLatencyMs: number;
    lastSeenAt?: string;
  }>;
  ingest?: {
    lastBackfillAt?: string;
    lastWebhookAt?: string;
    freshnessSeconds: number;
    lagStatus: "healthy" | "warning" | "critical" | "unavailable";
  };
  alertDelivery?: {
    attempts24h: number;
    delivered24h: number;
    failed24h: number;
    retryableCount: number;
    lastFailureAt?: string;
  };
  walletTracking?: {
    candidateCount: number;
    trackedCount: number;
    labeledCount: number;
    scoredCount: number;
    staleCount: number;
    suppressedCount: number;
  };
  trackingSubscriptions?: {
    pendingCount: number;
    activeCount: number;
    erroredCount: number;
    pausedCount: number;
    lastEventAt?: string;
  };
  queueDepth?: {
    defaultDepth: number;
    priorityDepth: number;
  };
  backfillHealth?: {
    jobs24h: number;
    activities24h: number;
    transactions24h: number;
    expansions24h: number;
    lastSuccessAt?: string;
  };
  staleRefresh?: {
    attempts24h: number;
    succeeded24h: number;
    productive24h: number;
    lastHitAt?: string;
  };
  recentRuns?: Array<{
    jobName: string;
    lastStatus: string;
    lastStartedAt: string;
    lastFinishedAt?: string;
    lastSuccessAt?: string;
    minutesSinceSuccess: number;
    lastError?: string;
  }>;
  recentFailures?: Array<{
    source: string;
    kind: string;
    occurredAt: string;
    summary: string;
    details?: Record<string, unknown>;
  }>;
};

type AdminDomesticPrelistingApiItem = {
  chain: string;
  tokenAddress: string;
  tokenSymbol: string;
  normalizedAssetKey: string;
  transferCount7d: number;
  transferCount24h: number;
  activeWalletCount: number;
  trackedWalletCount: number;
  distinctCounterpartyCount: number;
  totalAmount: string;
  largestTransferAmount: string;
  latestObservedAt: string;
  listedOnUpbit: boolean;
  listedOnBithumb: boolean;
};

type AdminDomesticPrelistingCollectionApiResponse = {
  items: AdminDomesticPrelistingApiItem[];
};

type AdminCuratedListCollectionApiResponse = {
  items: AdminCuratedListApiItem[];
};

type AdminAuditLogCollectionApiResponse = {
  items: AdminAuditLogApiItem[];
};

type AdminLabelCollectionEnvelope = {
  success: boolean;
  data: AdminLabelCollectionApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AdminSuppressionCollectionEnvelope = {
  success: boolean;
  data: AdminSuppressionCollectionApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AdminSuppressionEnvelope = {
  success: boolean;
  data?: AdminSuppressionApiItem | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AdminQuotaCollectionEnvelope = {
  success: boolean;
  data: AdminQuotaCollectionApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AdminObservabilityEnvelope = {
  success: boolean;
  data: AdminObservabilityApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AdminDomesticPrelistingEnvelope = {
  success: boolean;
  data: AdminDomesticPrelistingCollectionApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AdminMutationEnvelope = {
  success: boolean;
  data?: {
    deleted?: boolean;
  } | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AdminCuratedListCollectionEnvelope = {
  success: boolean;
  data: AdminCuratedListCollectionApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AdminAuditLogCollectionEnvelope = {
  success: boolean;
  data: AdminAuditLogCollectionApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AdminBacktestOpsEnvelope = {
  success: boolean;
  data: {
    statusMessage: string;
    checks: Array<{
      key: string;
      label: string;
      description: string;
      status: "ready" | "missing" | "not_configured";
      configured: boolean;
      path?: string;
    }>;
  } | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type AdminBacktestRunEnvelope = {
  success: boolean;
  data?: AdminBacktestRunResultPreview | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

export type WalletSummaryRequest = {
  chain: "evm" | "solana";
  address: string;
};

export type WalletGraphRequest = {
  chain: "evm" | "solana";
  address: string;
  depthRequested: number;
};

export type WalletDetailRequest = WalletSummaryRequest;

export type WalletBriefRequest = WalletSummaryRequest;

export type FindingsFeedRequest = {
  cursor?: string;
  types?: string[];
};

export type EntityInterpretationRequest = {
  entityKey: string;
};

export type ClusterDetailRequest = {
  clusterId: string;
};

type LoadWalletSummaryPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: WalletSummaryPreview;
  request?: WalletSummaryRequest;
  requestHeaders?: HeadersInit;
};

type LoadWalletGraphPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: WalletGraphPreview;
  request?: WalletGraphRequest;
  requestHeaders?: HeadersInit;
};

type LoadWalletBriefPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: WalletBriefPreview;
  request?: WalletBriefRequest;
  requestHeaders?: HeadersInit;
};

type ExplainAnalystWalletOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  request?: WalletBriefRequest;
  requestHeaders?: HeadersInit;
  question?: string;
  forceRefresh?: boolean;
  async?: boolean;
};

type AnalyzeAnalystWalletOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  request?: WalletBriefRequest;
  requestHeaders?: HeadersInit;
  question?: string;
  recentTurns?: AnalystWalletAnalyzeRecentTurnInput[];
};

type AnalyzeAnalystEntityOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  request?: EntityInterpretationRequest;
  requestHeaders?: HeadersInit;
  question?: string;
  recentTurns?: AnalystEntityAnalyzeRecentTurnInput[];
};

type LoadFindingsFeedPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: FindingsFeedPreview;
  request?: FindingsFeedRequest;
  requestHeaders?: HeadersInit;
};

type LoadSearchPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: SearchPreview;
  query: string;
  refreshMode?: "manual";
  requestHeaders?: HeadersInit;
};

type LoadClusterDetailPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: ClusterDetailPreview;
  request?: ClusterDetailRequest;
};

type LoadEntityInterpretationPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: EntityInterpretationPreview;
  request?: EntityInterpretationRequest;
  requestHeaders?: HeadersInit;
};

type LoadFirstConnectionFeedPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: FirstConnectionFeedPreview;
  sort?: "latest" | "score";
};

type LoadAlertCenterPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: AlertCenterPreview;
  severity?: AlertCenterPreview["activeSeverityFilter"];
  signalType?: AlertCenterPreview["activeSignalFilter"];
  status?: AlertCenterPreview["activeStatusFilter"];
  cursor?: string | undefined;
  requestHeaders?: HeadersInit | undefined;
};

type UpdateAlertInboxEventOptions = {
  eventId: string;
  isRead: boolean;
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
};

type UpdateAlertRuleMutationOptions = {
  ruleId: string;
  action: "toggle-enabled" | "toggle-snooze";
  currentRule: AlertCenterRulePreview;
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
};

export type AlertCenterMutationResult = {
  ok: boolean;
  message: string;
  event?: AlertCenterInboxItemPreview;
  rule?: AlertCenterRulePreview;
};

type TrackWalletAlertRuleOptions = {
  chain: "evm" | "solana";
  address: string;
  label: string;
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  requestHeaders?: HeadersInit;
};

export type TrackWalletAlertRuleResult = {
  ok: boolean;
  message: string;
  watchlistId?: string;
  ruleId?: string;
  status?: number;
  nextHref?: string;
};

type LoadAdminConsolePreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: AdminConsolePreview;
  requestHeaders?: HeadersInit;
};

type RunAdminBacktestOperationOptions = {
  checkKey: string;
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  requestHeaders?: HeadersInit;
};

type CreateAdminSuppressionOptions = {
  scope: string;
  target: string;
  reason: string;
  expiresAt?: string;
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
};

type DeleteAdminSuppressionOptions = {
  suppressionId: string;
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
};

export type AdminConsoleMutationResult = {
  ok: boolean;
  message: string;
  suppression?: AdminConsoleSuppressionPreview;
  deletedSuppressionId?: string;
};

export const walletSummaryRoute = "GET /v1/wallets/:chain/:address/summary";
export const walletBriefRoute = "GET /v1/wallets/:chain/:address/brief";
export const analystWalletExplainRoute =
  "POST /v1/analyst/wallets/:chain/:address/explain";
export const analystWalletAnalyzeRoute =
  "POST /v1/analyst/wallets/:chain/:address/analyze";
export const analystEntityAnalyzeRoute = "POST /v1/analyst/entity/:id/analyze";
export const walletGraphRoute = "GET /v1/wallets/:chain/:address/graph";
export const clusterDetailRoute = "GET /v1/clusters/:clusterId";
export const findingsFeedRoute = "GET /v1/findings";
export const discoverFeaturedWalletsRoute = "GET /v1/discover/featured-wallets";
export const discoverDomesticPrelistingRoute =
  "GET /v1/discover/domestic-prelisting-candidates";
export const entityInterpretationRoute = "GET /v1/entity/:id";
export const analystWalletBriefRoute =
  "GET /v1/analyst/wallets/:chain/:address/brief";
export const analystFindingsRoute = "GET /v1/analyst/findings";
export const analystEntityInterpretationRoute = "GET /v1/analyst/entity/:id";
export const shadowExitFeedRoute = "GET /v1/signals/shadow-exits";
export const firstConnectionFeedRoute = "GET /v1/signals/first-connections";
export const searchRoute = "GET /v1/search";
export const alertInboxRoute = "GET /v1/alerts";
export const alertRulesCollectionRoute = "GET /v1/alert-rules";
export const alertDeliveryChannelsRoute = "GET /v1/alert-delivery-channels";
export const watchlistsRoute = "GET /v1/watchlists";
export const adminLabelsRoute = "GET /v1/admin/labels";
export const adminSuppressionsRoute = "GET /v1/admin/suppressions";
export const adminProviderQuotasRoute = "GET /v1/admin/provider-quotas";
export const adminObservabilityRoute = "GET /v1/admin/observability";
export const adminDomesticPrelistingRoute =
  "GET /v1/admin/domestic-prelisting-candidates";
export const adminCuratedListsRoute = "GET /v1/admin/curated-lists";
export const adminAuditLogsRoute = "GET /v1/admin/audit-logs";
export const adminBacktestsRoute = "GET /v1/admin/backtests";
export const adminBacktestRunRoute = "POST /v1/admin/backtests/:checkKey/run";

const walletSummaryRoutePattern =
  /^\/v1\/wallets\/(evm|solana)\/([^/]+)\/summary$/;

const walletSummaryRequest: WalletSummaryRequest = {
  chain: "evm",
  address: "0x8f1d9c72be9f2a8ec6d3b9ac1e5d7c4289a1031f",
};

const walletGraphRequest: WalletGraphRequest = {
  chain: "evm",
  address: "0x8f1d9c72be9f2a8ec6d3b9ac1e5d7c4289a1031f",
  depthRequested: 1,
};

const walletBriefRequest: WalletBriefRequest = walletSummaryRequest;

const findingsFeedRequest: FindingsFeedRequest = {};

const entityInterpretationRequest: EntityInterpretationRequest = {
  entityKey: "curated:exchange:binance",
};

const clusterDetailRequest: ClusterDetailRequest = {
  clusterId: "cluster_seed_whales",
};

function getApiBaseUrl(apiBaseUrl?: string): string | undefined {
  const trimmed = apiBaseUrl?.trim();
  if (trimmed) {
    return trimmed;
  }

  if (typeof window !== "undefined") {
    return undefined;
  }

  const envBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL?.trim();
  return envBaseUrl ? envBaseUrl : undefined;
}

export function buildWalletDetailHref(request: WalletDetailRequest): string {
  return `/wallets/${request.chain}/${encodeURIComponent(request.address)}`;
}

export function buildClusterDetailHref(request: ClusterDetailRequest): string {
  return `/clusters/${encodeURIComponent(request.clusterId)}`;
}

export function buildEntityDetailHref(entityKey: string): string {
  return `/entity/${encodeURIComponent(entityKey)}`;
}

export function buildProductSearchHref(query: string): string {
  return `/?q=${encodeURIComponent(query)}`;
}

export function shouldPersistSearchPreviewToUrl(
  preview: SearchPreview,
): boolean {
  return (
    preview.navigation &&
    (preview.inputKind === "evm_address" ||
      preview.inputKind === "solana_address")
  );
}

export function resolveWalletSummaryRequestFromRoute(
  route: string,
): WalletSummaryRequest | null {
  const match = route.match(walletSummaryRoutePattern);

  if (!match) {
    return null;
  }

  const address = match[2];

  if (!address) {
    return null;
  }

  let decodedAddress = address;

  try {
    decodedAddress = decodeURIComponent(address);
  } catch {
    return null;
  }

  return {
    chain: match[1] as WalletSummaryRequest["chain"],
    address: decodedAddress,
  };
}

export function resolveWalletDetailHrefFromSummaryRoute(
  route: string,
): string | null {
  const request = resolveWalletSummaryRequestFromRoute(route);

  if (!request) {
    return null;
  }

  return buildWalletDetailHref(request);
}

function buildWalletSummaryUrl(
  request: WalletSummaryRequest,
  apiBaseUrl?: string,
): string {
  const path = `/v1/wallets/${request.chain}/${request.address}/summary`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return path;
  }

  return new URL(path, resolvedBaseUrl).toString();
}

function buildWalletBriefUrl(
  request: WalletSummaryRequest,
  apiBaseUrl?: string,
): string {
  const path = `/v1/wallets/${request.chain}/${request.address}/brief`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return path;
  }

  return new URL(path, resolvedBaseUrl).toString();
}

function buildAnalystWalletBriefUrl(
  request: WalletSummaryRequest,
  apiBaseUrl?: string,
): string {
  const path = `/v1/analyst/wallets/${request.chain}/${request.address}/brief`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return path;
  }

  return new URL(path, resolvedBaseUrl).toString();
}

function buildAnalystWalletExplainUrl(
  request: WalletSummaryRequest,
  apiBaseUrl?: string,
): string {
  const path = `/v1/analyst/wallets/${request.chain}/${request.address}/explain`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return path;
  }

  return new URL(path, resolvedBaseUrl).toString();
}

function buildAnalystWalletAnalyzeUrl(
  request: WalletSummaryRequest,
  apiBaseUrl?: string,
): string {
  const path = `/v1/analyst/wallets/${request.chain}/${request.address}/analyze`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return path;
  }

  return new URL(path, resolvedBaseUrl).toString();
}

function buildAnalystEntityAnalyzeUrl(
  request: EntityInterpretationRequest,
  apiBaseUrl?: string,
): string {
  const path = `/v1/analyst/entity/${encodeURIComponent(request.entityKey)}/analyze`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return path;
  }

  return new URL(path, resolvedBaseUrl).toString();
}

function buildWalletGraphUrl(
  request: WalletGraphRequest,
  apiBaseUrl?: string,
): string {
  const path = `/v1/wallets/${request.chain}/${request.address}/graph`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return `${path}?depth=${request.depthRequested}`;
  }

  const url = new URL(path, resolvedBaseUrl);
  url.searchParams.set("depth", String(request.depthRequested));
  return url.toString();
}

function buildFindingsFeedUrl(
  apiBaseUrl?: string,
  cursor?: string,
  types?: string[],
): string {
  const path = "/v1/findings";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  const params = new URLSearchParams();
  if (cursor?.trim()) {
    params.set("cursor", cursor.trim());
  }
  for (const type of types ?? []) {
    const trimmed = type.trim();
    if (trimmed) {
      params.append("type", trimmed);
    }
  }

  if (!resolvedBaseUrl) {
    return params.size > 0 ? `${path}?${params.toString()}` : path;
  }

  const url = new URL(path, resolvedBaseUrl);
  params.forEach((value, key) => url.searchParams.append(key, value));
  return url.toString();
}

function buildAnalystFindingsUrl(
  apiBaseUrl?: string,
  cursor?: string,
  types?: string[],
): string {
  const path = "/v1/analyst/findings";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  const params = new URLSearchParams();
  if (cursor?.trim()) {
    params.set("cursor", cursor.trim());
  }
  for (const type of types ?? []) {
    const trimmed = type.trim();
    if (trimmed) {
      params.append("type", trimmed);
    }
  }

  if (!resolvedBaseUrl) {
    return params.size > 0 ? `${path}?${params.toString()}` : path;
  }

  const url = new URL(path, resolvedBaseUrl);
  params.forEach((value, key) => url.searchParams.append(key, value));
  return url.toString();
}

function buildEntityInterpretationUrl(
  entityKey: string,
  apiBaseUrl?: string,
): string {
  const path = `/v1/entity/${encodeURIComponent(entityKey)}`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAnalystEntityInterpretationUrl(
  entityKey: string,
  apiBaseUrl?: string,
): string {
  const path = `/v1/analyst/entity/${encodeURIComponent(entityKey)}`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildSearchUrl(
  query: string,
  apiBaseUrl?: string,
  refreshMode?: "manual",
): string {
  const path = "/v1/search";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    const params = new URLSearchParams({ q: query });
    if (refreshMode === "manual") {
      params.set("refresh", refreshMode);
    }
    return `${path}?${params.toString()}`;
  }

  const url = new URL(path, resolvedBaseUrl);
  url.searchParams.set("q", query);
  if (refreshMode === "manual") {
    url.searchParams.set("refresh", refreshMode);
  }
  return url.toString();
}

function buildClusterDetailUrl(
  request: ClusterDetailRequest,
  apiBaseUrl?: string,
): string {
  const path = `/v1/clusters/${request.clusterId}`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return path;
  }

  return new URL(path, resolvedBaseUrl).toString();
}

function buildShadowExitFeedUrl(apiBaseUrl?: string): string {
  const path = "/v1/signals/shadow-exits";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return path;
  }

  return new URL(path, resolvedBaseUrl).toString();
}

function buildDiscoverFeaturedWalletsUrl(apiBaseUrl?: string): string {
  const path = "/v1/discover/featured-wallets";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return path;
  }

  return new URL(path, resolvedBaseUrl).toString();
}

function buildDiscoverDomesticPrelistingUrl(apiBaseUrl?: string): string {
  const path = "/v1/discover/domestic-prelisting-candidates";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return path;
  }

  return new URL(path, resolvedBaseUrl).toString();
}

function buildFirstConnectionFeedUrl(
  sort: "latest" | "score" = "latest",
  apiBaseUrl?: string,
): string {
  const path = "/v1/signals/first-connections";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return `${path}?sort=${sort}`;
  }

  const url = new URL(path, resolvedBaseUrl);
  url.searchParams.set("sort", sort);
  return url.toString();
}

function buildAlertInboxUrl(
  severity: AlertCenterPreview["activeSeverityFilter"] = "all",
  signalType: AlertCenterPreview["activeSignalFilter"] = "all",
  status: AlertCenterPreview["activeStatusFilter"] = "all",
  cursor?: string,
  apiBaseUrl?: string,
): string {
  const path = "/v1/alerts";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  const params = new URLSearchParams();
  if (severity !== "all") {
    params.set("severity", severity);
  }
  if (signalType !== "all") {
    params.set("signalType", signalType);
  }
  if (status !== "all") {
    params.set("status", status);
  }
  if (cursor?.trim()) {
    params.set("cursor", cursor.trim());
  }

  if (!resolvedBaseUrl) {
    const query = params.toString();
    return query ? `${path}?${query}` : path;
  }

  const url = new URL(path, resolvedBaseUrl);
  params.forEach((value, key) => {
    url.searchParams.set(key, value);
  });
  return url.toString();
}

function buildAlertInboxEventUrl(eventId: string, apiBaseUrl?: string): string {
  const path = `/v1/alerts/${encodeURIComponent(eventId)}`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAlertRulesUrl(apiBaseUrl?: string): string {
  const path = "/v1/alert-rules";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAlertRuleDetailUrl(ruleID: string, apiBaseUrl?: string): string {
  const path = `/v1/alert-rules/${encodeURIComponent(ruleID)}`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildWatchlistsUrl(apiBaseUrl?: string): string {
  const path = "/v1/watchlists";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildWatchlistDetailUrl(
  watchlistId: string,
  apiBaseUrl?: string,
): string {
  const path = `/v1/watchlists/${encodeURIComponent(watchlistId)}`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildWatchlistItemsUrl(
  watchlistId: string,
  apiBaseUrl?: string,
): string {
  const path = `/v1/watchlists/${encodeURIComponent(watchlistId)}/items`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAdminLabelsUrl(apiBaseUrl?: string): string {
  const path = "/v1/admin/labels";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAdminSuppressionsUrl(apiBaseUrl?: string): string {
  const path = "/v1/admin/suppressions";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAdminSuppressionDetailUrl(
  suppressionID: string,
  apiBaseUrl?: string,
): string {
  const path = `/v1/admin/suppressions/${encodeURIComponent(suppressionID)}`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAdminProviderQuotasUrl(apiBaseUrl?: string): string {
  const path = "/v1/admin/provider-quotas";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAdminObservabilityUrl(apiBaseUrl?: string): string {
  const path = "/v1/admin/observability";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAdminDomesticPrelistingUrl(apiBaseUrl?: string): string {
  const path = "/v1/admin/domestic-prelisting-candidates";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAdminCuratedListsUrl(apiBaseUrl?: string): string {
  const path = "/v1/admin/curated-lists";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAdminAuditLogsUrl(apiBaseUrl?: string): string {
  const path = "/v1/admin/audit-logs";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAdminBacktestsUrl(apiBaseUrl?: string): string {
  const path = "/v1/admin/backtests";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAdminBacktestRunUrl(
  checkKey: string,
  apiBaseUrl?: string,
): string {
  const path = `/v1/admin/backtests/${encodeURIComponent(checkKey)}/run`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function buildAlertDeliveryChannelsUrl(apiBaseUrl?: string): string {
  const path = "/v1/alert-delivery-channels";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function mapEvidenceTone(
  score: WalletSummaryApiScore,
): WalletSummaryScorePreview["tone"] {
  if (score.name === "cluster_score") {
    return "emerald";
  }

  if (score.name === "shadow_exit_risk") {
    return "amber";
  }

  return score.rating === "high" ? "violet" : "teal";
}

function mapWalletSummaryResponse(
  response: WalletSummaryApiResponse,
  source: WalletSummaryPreview["source"],
): WalletSummaryPreview {
  return {
    mode: "live",
    source,
    route: walletSummaryRoute,
    chain: response.chain === "evm" ? "EVM" : "SOLANA",
    chainLabel: formatChainLabel(response.chain),
    address: response.address,
    label: response.displayName,
    ...(response.clusterId ? { clusterId: response.clusterId } : {}),
    counterparties: response.counterparties ?? 0,
    statusMessage:
      "Live backend data loaded from GET /v1/wallets/:chain/:address/summary.",
    topCounterparties: (response.topCounterparties ?? []).map(
      (counterparty) => ({
        chain: counterparty.chain,
        chainLabel: formatChainLabel(counterparty.chain),
        address: counterparty.address,
        ...(counterparty.entityKey
          ? { entityKey: counterparty.entityKey }
          : {}),
        ...(counterparty.entityType
          ? { entityType: counterparty.entityType }
          : {}),
        ...(counterparty.entityLabel
          ? { entityLabel: counterparty.entityLabel }
          : {}),
        interactionCount: counterparty.interactionCount,
        inboundCount: counterparty.inboundCount ?? 0,
        outboundCount: counterparty.outboundCount ?? 0,
        inboundAmount: counterparty.inboundAmount ?? "0",
        outboundAmount: counterparty.outboundAmount ?? "0",
        primaryToken: counterparty.primaryToken ?? "",
        tokenBreakdowns: (counterparty.tokenBreakdowns ?? []).map((token) => ({
          symbol: token.symbol,
          inboundAmount: token.inboundAmount ?? "0",
          outboundAmount: token.outboundAmount ?? "0",
        })),
        directionLabel: counterparty.directionLabel ?? "mixed",
        firstSeenAt: counterparty.firstSeenAt ?? "",
        latestActivityAt: counterparty.latestActivityAt,
      }),
    ),
    recentFlow: {
      incomingTxCount7d: response.recentFlow?.incomingTxCount7d ?? 0,
      outgoingTxCount7d: response.recentFlow?.outgoingTxCount7d ?? 0,
      incomingTxCount30d: response.recentFlow?.incomingTxCount30d ?? 0,
      outgoingTxCount30d: response.recentFlow?.outgoingTxCount30d ?? 0,
      netDirection7d: response.recentFlow?.netDirection7d ?? "balanced",
      netDirection30d: response.recentFlow?.netDirection30d ?? "balanced",
    },
    ...(response.enrichment
      ? {
          enrichment: {
            provider: response.enrichment.provider,
            netWorthUsd: response.enrichment.netWorthUsd,
            nativeBalance: response.enrichment.nativeBalance,
            nativeBalanceFormatted: response.enrichment.nativeBalanceFormatted,
            activeChains: response.enrichment.activeChains ?? [],
            activeChainCount: response.enrichment.activeChainCount ?? 0,
            holdings: (response.enrichment.holdings ?? []).map((holding) => ({
              symbol: holding.symbol,
              tokenAddress: holding.tokenAddress ?? "",
              balance: holding.balance ?? "",
              balanceFormatted: holding.balanceFormatted ?? "",
              valueUsd: holding.valueUsd ?? "",
              portfolioPercentage: holding.portfolioPercentage ?? 0,
              isNative: holding.isNative ?? false,
            })),
            holdingCount:
              response.enrichment.holdingCount ??
              response.enrichment.holdings?.length ??
              0,
            source: response.enrichment.source,
            updatedAt: response.enrichment.updatedAt,
          },
        }
      : {}),
    indexing: {
      status: response.indexing?.status ?? "ready",
      lastIndexedAt: response.indexing?.lastIndexedAt ?? "",
      coverageStartAt: response.indexing?.coverageStartAt ?? "",
      coverageEndAt: response.indexing?.coverageEndAt ?? "",
      coverageWindowDays: response.indexing?.coverageWindowDays ?? 0,
    },
    latestSignals: (response.latestSignals ?? []).map((signal) => ({
      name: signal.name,
      value: signal.value,
      rating: signal.rating,
      label: signal.label,
      source: signal.source,
      observedAt: signal.observedAt,
    })),
    scores: response.scores.map((score) => ({
      name: score.name,
      value: score.value,
      rating: score.rating,
      tone: mapEvidenceTone(score),
      ...(() => {
        if (score.name !== "cluster_score") {
          return {};
        }
        const clusterBreakdown = deriveClusterScoreBreakdown(score);
        return clusterBreakdown ? { clusterBreakdown } : {};
      })(),
    })),
  };
}

function deriveClusterScoreBreakdown(
  score: WalletSummaryApiScore,
): WalletSummaryClusterScoreBreakdownPreview | undefined {
  const evidence = score.evidence ?? [];
  if (evidence.length === 0) {
    return undefined;
  }

  let peerWalletOverlap = 0;
  let sharedEntityLinks = 0;
  let bidirectionalPeerFlows = 0;
  let contradictionPenalty = 0;
  let suppressionDiscount = 0;
  let sourceNodeCount = 0;
  let sourceEdgeCount = 0;
  let analysisNodeCount = 0;
  let analysisEdgeCount = 0;
  let samplingApplied = false;
  let sourceDensityCapped = false;
  const contradictionReasons = new Set<string>();
  const suppressionReasons = new Set<string>();

  for (const item of evidence) {
    const metadata = item.metadata ?? {};
    peerWalletOverlap = Math.max(
      peerWalletOverlap,
      readMetadataNumber(metadata.wallet_peer_overlap),
      readMetadataNumber(metadata.overlapping_wallets),
    );
    sharedEntityLinks = Math.max(
      sharedEntityLinks,
      readMetadataNumber(metadata.shared_entity_neighbors),
      readMetadataNumber(metadata.shared_counterparties),
    );
    bidirectionalPeerFlows = Math.max(
      bidirectionalPeerFlows,
      readMetadataNumber(metadata.bidirectional_flow_peers),
      readMetadataNumber(metadata.mutual_transfer_count),
    );
    contradictionPenalty = Math.max(
      contradictionPenalty,
      readMetadataNumber(metadata.contradiction_penalty),
      readMetadataNumber(metadata.route_contradiction_penalty),
    );
    suppressionDiscount = Math.max(
      suppressionDiscount,
      readMetadataNumber(metadata.suppression_discount),
    );
    sourceNodeCount = Math.max(
      sourceNodeCount,
      readMetadataNumber(metadata.graph_node_count),
      readMetadataNumber(metadata.source_graph_node_count),
    );
    sourceEdgeCount = Math.max(
      sourceEdgeCount,
      readMetadataNumber(metadata.graph_edge_count),
      readMetadataNumber(metadata.source_graph_edge_count),
    );
    analysisNodeCount = Math.max(
      analysisNodeCount,
      readMetadataNumber(metadata.analysis_graph_node_count),
    );
    analysisEdgeCount = Math.max(
      analysisEdgeCount,
      readMetadataNumber(metadata.analysis_graph_edge_count),
    );
    samplingApplied =
      samplingApplied ||
      readMetadataBoolean(metadata.analysis_graph_sampling_applied);
    sourceDensityCapped =
      sourceDensityCapped ||
      readMetadataBoolean(metadata.source_density_capped);

    for (const reason of readMetadataStringArray(
      metadata.contradiction_reasons,
    )) {
      contradictionReasons.add(reason);
    }
    for (const reason of readMetadataStringArray(
      metadata.suppression_reasons,
    )) {
      suppressionReasons.add(reason);
    }
  }

  if (
    peerWalletOverlap === 0 &&
    sharedEntityLinks === 0 &&
    bidirectionalPeerFlows === 0 &&
    contradictionPenalty === 0 &&
    suppressionDiscount === 0 &&
    !samplingApplied &&
    !sourceDensityCapped &&
    sourceNodeCount === 0 &&
    sourceEdgeCount === 0 &&
    analysisNodeCount === 0 &&
    analysisEdgeCount === 0 &&
    contradictionReasons.size === 0 &&
    suppressionReasons.size === 0
  ) {
    return undefined;
  }

  return {
    peerWalletOverlap,
    sharedEntityLinks,
    bidirectionalPeerFlows,
    contradictionPenalty,
    suppressionDiscount,
    samplingApplied,
    sourceDensityCapped,
    sourceNodeCount,
    sourceEdgeCount,
    analysisNodeCount,
    analysisEdgeCount,
    contradictionReasons: [...contradictionReasons],
    suppressionReasons: [...suppressionReasons],
  };
}

function readMetadataNumber(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string") {
    const parsed = Number.parseFloat(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function readMetadataBoolean(value: unknown): boolean {
  if (typeof value === "boolean") {
    return value;
  }
  if (typeof value === "string") {
    return value === "true";
  }
  return false;
}

function readMetadataStringArray(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value
      .map((entry) => (typeof entry === "string" ? entry.trim() : ""))
      .filter((entry) => entry.length > 0);
  }
  if (typeof value === "string" && value.trim().length > 0) {
    return [value.trim()];
  }
  return [];
}

function cloneFindingPreview(item: FindingPreview): FindingPreview {
  return {
    ...item,
    importanceReason: [...item.importanceReason],
    observedFacts: [...item.observedFacts],
    inferredInterpretations: [...item.inferredInterpretations],
    evidence: item.evidence.map((part) => ({
      ...part,
      ...(part.metadata ? { metadata: { ...part.metadata } } : {}),
    })),
    nextWatch: item.nextWatch.map((part) => ({
      ...part,
      ...(part.metadata ? { metadata: { ...part.metadata } } : {}),
    })),
  };
}

function cloneWalletLabelPreview(item: WalletLabelPreview): WalletLabelPreview {
  return {
    ...item,
  };
}

function mapWalletBriefResponse(
  response: WalletBriefApiResponse,
  source: WalletBriefPreview["source"],
): WalletBriefPreview {
  return {
    mode: "live",
    source,
    route: walletBriefRoute,
    chain: response.chain,
    address: response.address,
    displayName: response.displayName,
    statusMessage:
      "Live backend data loaded from GET /v1/wallets/:chain/:address/brief.",
    aiSummary: response.aiSummary,
    keyFindings: (response.keyFindings ?? []).map(cloneFindingPreview),
    verifiedLabels: (response.verifiedLabels ?? []).map(
      cloneWalletLabelPreview,
    ),
    probableLabels: (response.probableLabels ?? []).map(
      cloneWalletLabelPreview,
    ),
    behavioralLabels: (response.behavioralLabels ?? []).map(
      cloneWalletLabelPreview,
    ),
    topCounterparties: (response.topCounterparties ?? []).map(
      (counterparty) => ({
        chain: counterparty.chain,
        chainLabel: formatChainLabel(counterparty.chain),
        address: counterparty.address,
        ...(counterparty.entityKey
          ? { entityKey: counterparty.entityKey }
          : {}),
        ...(counterparty.entityType
          ? { entityType: counterparty.entityType }
          : {}),
        ...(counterparty.entityLabel
          ? { entityLabel: counterparty.entityLabel }
          : {}),
        interactionCount: counterparty.interactionCount,
        inboundCount: counterparty.inboundCount ?? 0,
        outboundCount: counterparty.outboundCount ?? 0,
        inboundAmount: counterparty.inboundAmount ?? "0",
        outboundAmount: counterparty.outboundAmount ?? "0",
        primaryToken: counterparty.primaryToken ?? "",
        tokenBreakdowns: (counterparty.tokenBreakdowns ?? []).map((token) => ({
          symbol: token.symbol,
          inboundAmount: token.inboundAmount ?? "0",
          outboundAmount: token.outboundAmount ?? "0",
        })),
        directionLabel: counterparty.directionLabel ?? "mixed",
        firstSeenAt: counterparty.firstSeenAt ?? "",
        latestActivityAt: counterparty.latestActivityAt,
      }),
    ),
    recentFlow: {
      incomingTxCount7d: response.recentFlow?.incomingTxCount7d ?? 0,
      outgoingTxCount7d: response.recentFlow?.outgoingTxCount7d ?? 0,
      incomingTxCount30d: response.recentFlow?.incomingTxCount30d ?? 0,
      outgoingTxCount30d: response.recentFlow?.outgoingTxCount30d ?? 0,
      netDirection7d: response.recentFlow?.netDirection7d ?? "balanced",
      netDirection30d: response.recentFlow?.netDirection30d ?? "balanced",
    },
    ...(response.enrichment
      ? {
          enrichment: {
            provider: response.enrichment.provider,
            netWorthUsd: response.enrichment.netWorthUsd,
            nativeBalance: response.enrichment.nativeBalance,
            nativeBalanceFormatted: response.enrichment.nativeBalanceFormatted,
            activeChains: response.enrichment.activeChains ?? [],
            activeChainCount: response.enrichment.activeChainCount ?? 0,
            holdings: (response.enrichment.holdings ?? []).map((holding) => ({
              symbol: holding.symbol,
              tokenAddress: holding.tokenAddress ?? "",
              balance: holding.balance ?? "",
              balanceFormatted: holding.balanceFormatted ?? "",
              valueUsd: holding.valueUsd ?? "",
              portfolioPercentage: holding.portfolioPercentage ?? 0,
              isNative: holding.isNative ?? false,
            })),
            holdingCount:
              response.enrichment.holdingCount ??
              response.enrichment.holdings?.length ??
              0,
            source: response.enrichment.source,
            updatedAt: response.enrichment.updatedAt,
          },
        }
      : {}),
    indexing: {
      status: response.indexing?.status ?? "ready",
      lastIndexedAt: response.indexing?.lastIndexedAt ?? "",
      coverageStartAt: response.indexing?.coverageStartAt ?? "",
      coverageEndAt: response.indexing?.coverageEndAt ?? "",
      coverageWindowDays: response.indexing?.coverageWindowDays ?? 0,
    },
    latestSignals: (response.latestSignals ?? []).map((signal) => ({
      name: signal.name,
      value: signal.value,
      rating: signal.rating,
      label: signal.label,
      source: signal.source,
      observedAt: signal.observedAt,
    })),
    scores: (response.scores ?? []).map((score) => ({
      name: score.name,
      value: score.value,
      rating: score.rating,
      tone: mapEvidenceTone(score),
    })),
  };
}

function mapEntityInterpretationResponse(
  response: EntityInterpretationApiResponse,
  source: EntityInterpretationPreview["source"],
): EntityInterpretationPreview {
  return {
    mode: "live",
    source,
    route: entityInterpretationRoute,
    entityKey: response.entityKey,
    entityType: response.entityType,
    displayName: response.displayName,
    walletCount: response.walletCount,
    statusMessage: "Live backend data loaded from GET /v1/entity/:id.",
    ...(response.latestActivityAt
      ? { latestActivityAt: response.latestActivityAt }
      : {}),
    members: (response.members ?? []).map((member) => ({
      chain: member.chain,
      address: member.address,
      displayName: member.displayName,
      ...(member.latestActivityAt
        ? { latestActivityAt: member.latestActivityAt }
        : {}),
      verifiedLabels: (member.verifiedLabels ?? []).map(
        cloneWalletLabelPreview,
      ),
      probableLabels: (member.probableLabels ?? []).map(
        cloneWalletLabelPreview,
      ),
      behavioralLabels: (member.behavioralLabels ?? []).map(
        cloneWalletLabelPreview,
      ),
    })),
    findings: (response.findings ?? []).map(cloneFindingPreview),
  };
}

function mapClusterDetailResponse(
  response: ClusterDetailApiResponse,
  source: ClusterDetailPreview["source"],
): ClusterDetailPreview {
  return {
    mode: "live",
    source,
    route: clusterDetailRoute,
    clusterId: response.id,
    label: response.label,
    clusterType: response.clusterType,
    classification: response.classification,
    score: response.score,
    memberCount: response.memberCount,
    members: response.members ?? [],
    commonActions: response.commonActions ?? [],
    evidence: response.evidence ?? [],
    statusMessage: "Live backend data loaded from GET /v1/clusters/:clusterId.",
  };
}

function mapShadowExitRatingTone(
  rating: ShadowExitFeedApiItem["rating"],
): ShadowExitFeedPreviewItem["scoreTone"] {
  if (rating === "high") {
    return "amber";
  }

  if (rating === "medium") {
    return "violet";
  }

  return "teal";
}

function formatShadowExitReviewLabel(
  rating: ShadowExitFeedApiItem["rating"],
): string {
  if (rating === "high") {
    return "closer review";
  }

  if (rating === "medium") {
    return "monitor";
  }

  return "lighter watch";
}

function buildShadowExitWalletHref(item: ShadowExitFeedApiItem): string {
  return buildWalletDetailHref({
    chain: item.chain,
    address: item.address,
  });
}

function mapShadowExitFeedResponse(
  response: ShadowExitFeedApiResponse,
  source: ShadowExitFeedPreview["source"],
): ShadowExitFeedPreview {
  const items = response.items.map((item) => ({
    walletId: item.walletId,
    chain: item.chain,
    chainLabel: formatChainLabel(item.chain),
    address: item.address,
    label: item.label,
    ...(item.clusterId ? { clusterId: item.clusterId } : {}),
    score: item.score,
    rating: item.rating,
    scoreTone: mapShadowExitRatingTone(item.rating),
    reviewLabel: formatShadowExitReviewLabel(item.rating),
    observedAt: item.observedAt,
    explanation: item.explanation,
    walletHref: buildShadowExitWalletHref(item),
    ...(item.clusterId
      ? {
          clusterHref: buildClusterDetailHref({ clusterId: item.clusterId }),
        }
      : {}),
    evidence: item.evidence ?? [],
  }));

  return {
    mode: "live",
    source,
    route: shadowExitFeedRoute,
    windowLabel: response.windowLabel,
    itemCount: items.length,
    highPriorityCount: items.filter((item) => item.rating === "high").length,
    latestObservedAt:
      items[0]?.observedAt ?? response.generatedAt ?? "2026-03-20T00:00:00Z",
    statusMessage:
      "Live backend data loaded from GET /v1/signals/shadow-exits.",
    items,
  };
}

function mapFirstConnectionRatingTone(
  rating: FirstConnectionFeedApiItem["rating"],
): FirstConnectionFeedPreviewItem["scoreTone"] {
  if (rating === "high") {
    return "amber";
  }

  if (rating === "medium") {
    return "violet";
  }

  return "teal";
}

function formatFirstConnectionReviewLabel(
  rating: FirstConnectionFeedApiItem["rating"],
): string {
  if (rating === "high") {
    return "fresh connection";
  }

  if (rating === "medium") {
    return "monitor";
  }

  return "light watch";
}

function buildFirstConnectionWalletHref(
  item: FirstConnectionFeedApiItem,
): string {
  return buildWalletDetailHref({
    chain: item.chain,
    address: item.address,
  });
}

function mapFirstConnectionFeedResponse(
  response: FirstConnectionFeedApiResponse,
  source: FirstConnectionFeedPreview["source"],
): FirstConnectionFeedPreview {
  const items = response.items.map((item) => ({
    walletId: item.walletId,
    chain: item.chain,
    chainLabel: formatChainLabel(item.chain),
    address: item.address,
    label: item.label,
    ...(item.clusterId ? { clusterId: item.clusterId } : {}),
    score: item.score,
    rating: item.rating,
    scoreTone: mapFirstConnectionRatingTone(item.rating),
    reviewLabel: formatFirstConnectionReviewLabel(item.rating),
    observedAt: item.observedAt,
    explanation: item.explanation,
    walletHref: buildFirstConnectionWalletHref(item),
    ...(item.clusterId
      ? {
          clusterHref: buildClusterDetailHref({ clusterId: item.clusterId }),
        }
      : {}),
    evidence: item.evidence ?? [],
  }));

  return {
    mode: "live",
    source,
    route: firstConnectionFeedRoute,
    sort: response.sort ?? "latest",
    windowLabel: response.windowLabel,
    itemCount: items.length,
    highPriorityCount: items.filter((item) => item.rating === "high").length,
    latestObservedAt:
      items[0]?.observedAt ?? response.generatedAt ?? "2026-03-20T00:00:00Z",
    statusMessage:
      "Live backend data loaded from GET /v1/signals/first-connections.",
    items,
  };
}

function mapFindingsFeedResponse(
  response: FindingsFeedApiResponse,
  source: FindingsFeedPreview["source"],
): FindingsFeedPreview {
  return {
    mode: "live",
    source,
    route: findingsFeedRoute,
    generatedAt: response.generatedAt,
    statusMessage: "Live backend data loaded from GET /v1/findings.",
    items: (response.items ?? []).map(cloneFindingPreview),
    ...(response.nextCursor ? { nextCursor: response.nextCursor } : {}),
    hasMore: response.hasMore,
  };
}

function mapWalletGraphResponse(
  response: WalletGraphApiResponse,
  source: WalletGraphPreview["source"],
): WalletGraphPreview {
  const nodes = response.nodes ?? [];
  const edges = (response.edges ?? []).map((edge) => ({
    ...edge,
    family: edge.family ?? walletGraphEdgeFamilyForKind(edge.kind),
    directionality:
      edge.directionality ?? walletGraphEdgeDirectionalityFor(edge),
  }));

  return {
    mode: "live",
    source,
    route: walletGraphRoute,
    chain: response.chain === "evm" ? "EVM" : "SOLANA",
    address: response.address,
    depthRequested: response.depthRequested,
    depthResolved: response.depthResolved,
    densityCapped: response.densityCapped,
    statusMessage:
      "Live backend data loaded from GET /v1/wallets/:chain/:address/graph.",
    ...(response.snapshot ? { snapshot: response.snapshot } : {}),
    neighborhoodSummary:
      response.neighborhoodSummary ??
      buildWalletGraphNeighborhoodSummaryPreview({
        nodes,
        edges,
      }),
    nodes,
    edges,
  };
}

function mapSearchResponse(
  response: SearchApiResponse,
  source: SearchPreview["source"],
): SearchPreview {
  const primary = response.results[0];

  return {
    mode: "live",
    source,
    route: searchRoute,
    query: response.query,
    inputKind: response.inputKind,
    kindLabel:
      primary?.kindLabel ??
      formatSearchKindLabel(primary?.kind ?? response.inputKind),
    chainLabel: primary?.chainLabel ?? formatSearchChainLabel(primary?.chain),
    title: primary?.label ?? response.query,
    explanation: primary?.explanation ?? response.explanation,
    ...(primary?.walletRoute ? { walletRoute: primary.walletRoute } : {}),
    navigation: Boolean(primary?.navigation && primary?.walletRoute),
  };
}

function formatAlertSignalTypeLabel(
  signalType: AlertCenterInboxItemPreview["signalType"],
): string {
  if (signalType === "cluster_score") {
    return "Cluster score";
  }
  if (signalType === "shadow_exit") {
    return "Shadow exit";
  }
  return "First connection";
}

function formatAlertSeverityLabel(
  severity: AlertCenterInboxItemPreview["severity"],
): string {
  if (severity === "critical") {
    return "Critical";
  }
  if (severity === "high") {
    return "High";
  }
  if (severity === "medium") {
    return "Medium";
  }
  return "Low";
}

function buildAlertInboxExplanation(item: AlertInboxApiItem): string {
  const scoreValue = Number(item.payload?.score_value ?? 0);
  const parts = [
    `${formatAlertSignalTypeLabel(item.signalType)} review candidate`,
    `${formatAlertSeverityLabel(item.severity)} priority`,
  ];
  if (Number.isFinite(scoreValue) && scoreValue > 0) {
    parts.push(`score ${scoreValue}`);
  }
  return parts.join(" · ");
}

function mapAlertInboxItem(
  item: AlertInboxApiItem,
): AlertCenterInboxItemPreview {
  return {
    id: item.id,
    alertRuleId: item.alertRuleId,
    signalType: item.signalType,
    severity: item.severity,
    observedAt: item.observedAt,
    createdAt: item.createdAt,
    isRead: item.isRead ?? false,
    ...(item.readAt ? { readAt: item.readAt } : {}),
    title: formatAlertSignalTypeLabel(item.signalType),
    explanation: buildAlertInboxExplanation(item),
    ...(typeof item.payload?.score_value === "number"
      ? { scoreValue: item.payload.score_value }
      : {}),
  };
}

function mapAlertRuleSummary(
  item: AlertRuleApiSummary,
): AlertCenterRulePreview {
  return {
    id: item.id,
    name: item.name,
    ruleType: item.ruleType,
    isEnabled: item.isEnabled,
    cooldownSeconds: item.cooldownSeconds,
    eventCount: item.eventCount,
    ...(item.lastTriggeredAt ? { lastTriggeredAt: item.lastTriggeredAt } : {}),
    watchlistId: item.definition.watchlistId ?? "",
    signalTypes: item.definition.signalTypes ?? [],
    minimumSeverity: item.definition.minimumSeverity ?? "medium",
    renotifyOnSeverityIncrease:
      item.definition.renotifyOnSeverityIncrease ?? false,
    tags: item.tags ?? [],
    ...(item.definition.snoozeUntil
      ? { snoozeUntil: item.definition.snoozeUntil }
      : {}),
  };
}

function mapAdminSuppressionItem(
  item: AdminSuppressionApiItem,
): AdminConsoleSuppressionPreview {
  return {
    id: item.id,
    scope: item.scope,
    target: item.target,
    reason: item.reason,
    createdBy: item.createdBy,
    active: item.active,
    updatedAt: item.updatedAt,
    expiresAt: item.expiresAt,
  };
}

function mapAlertCenterResponse(input: {
  severity: AlertCenterPreview["activeSeverityFilter"];
  signalType: AlertCenterPreview["activeSignalFilter"];
  status: AlertCenterPreview["activeStatusFilter"];
  inbox: AlertInboxApiResponse;
  rules: AlertRuleCollectionApiResponse;
  channels: AlertDeliveryChannelCollectionApiResponse;
}): AlertCenterPreview {
  return {
    mode: "live",
    source: "live-api",
    inboxRoute: alertInboxRoute,
    rulesRoute: alertRulesCollectionRoute,
    channelsRoute: alertDeliveryChannelsRoute,
    activeSeverityFilter: input.severity,
    activeSignalFilter: input.signalType,
    activeStatusFilter: input.status,
    statusMessage:
      "Live backend data loaded from GET /v1/alerts, GET /v1/alert-rules, and GET /v1/alert-delivery-channels.",
    nextCursor: input.inbox.nextCursor,
    hasMore: input.inbox.hasMore ?? false,
    unreadCount: input.inbox.unreadCount ?? 0,
    inbox: input.inbox.items.map(mapAlertInboxItem),
    rules: input.rules.items.map(mapAlertRuleSummary),
    channels: input.channels.items.map((item) => ({
      id: item.id,
      label: item.label,
      channelType: item.channelType,
      target: item.target,
      isEnabled: item.isEnabled,
      metadata: item.metadata ?? {},
      createdAt: item.createdAt,
      updatedAt: item.updatedAt,
    })),
  };
}

function mapAdminConsoleResponse(input: {
  labels: AdminLabelCollectionApiResponse;
  suppressions: AdminSuppressionCollectionApiResponse;
  quotas: AdminQuotaCollectionApiResponse;
  observability: AdminObservabilityApiResponse;
  domesticPrelisting: AdminDomesticPrelistingCollectionApiResponse;
  curatedLists: AdminCuratedListCollectionApiResponse;
  auditLogs: AdminAuditLogCollectionApiResponse;
  backtests: {
    statusMessage: string;
    checks: Array<{
      key: string;
      label: string;
      description: string;
      status: "ready" | "missing" | "not_configured";
      configured: boolean;
      path?: string;
    }>;
  };
}): AdminConsolePreview {
  return {
    mode: "live",
    source: "live-api",
    labelsRoute: adminLabelsRoute,
    suppressionsRoute: adminSuppressionsRoute,
    quotasRoute: adminProviderQuotasRoute,
    observabilityRoute: adminObservabilityRoute,
    domesticPrelistingRoute: adminDomesticPrelistingRoute,
    curatedListsRoute: adminCuratedListsRoute,
    auditLogsRoute: adminAuditLogsRoute,
    backtestOps: {
      route: adminBacktestsRoute,
      statusMessage: input.backtests.statusMessage,
      checks: (input.backtests.checks ?? []).map((item) => ({
        key: item.key,
        label: item.label,
        description: item.description,
        status: item.status,
        configured: item.configured,
        ...(item.path ? { path: item.path } : {}),
      })),
    },
    statusMessage:
      "Admin console is using live backend data for labels, suppressions, quota pressure, observability health, domestic prelisting candidates, curated lists, audit logs, and backtest operations.",
    labels: (input.labels.items ?? []).map((item) => ({
      id: item.id,
      name: item.name,
      description: item.description,
      color: item.color,
      createdBy: item.createdBy,
      updatedAt: item.updatedAt,
    })),
    suppressions: (input.suppressions.items ?? []).map(mapAdminSuppressionItem),
    domesticPrelisting: (input.domesticPrelisting.items ?? []).map((item) => ({
      chain: item.chain,
      tokenAddress: item.tokenAddress,
      tokenSymbol: item.tokenSymbol,
      normalizedAssetKey: item.normalizedAssetKey,
      transferCount7d: item.transferCount7d,
      transferCount24h: item.transferCount24h,
      activeWalletCount: item.activeWalletCount,
      trackedWalletCount: item.trackedWalletCount,
      distinctCounterpartyCount: item.distinctCounterpartyCount,
      totalAmount: item.totalAmount,
      largestTransferAmount: item.largestTransferAmount,
      latestObservedAt: item.latestObservedAt,
      listedOnUpbit: item.listedOnUpbit,
      listedOnBithumb: item.listedOnBithumb,
    })),
    quotas: (input.quotas.items ?? []).map((item) => ({
      provider: item.provider,
      status: item.status,
      limit: item.limit,
      used: item.used,
      reserved: item.reserved,
      windowLabel: `${item.windowStart} -> ${item.windowEnd}`,
      lastCheckedAt: item.lastCheckedAt,
    })),
    observability: {
      providerUsage: (input.observability.providerUsage ?? []).map((item) => ({
        provider: item.provider,
        status: item.status,
        used24h: item.used24h,
        error24h: item.error24h,
        avgLatencyMs: item.avgLatencyMs,
        ...(item.lastSeenAt ? { lastSeenAt: item.lastSeenAt } : {}),
      })),
      ingest: {
        lastBackfillAt: input.observability.ingest?.lastBackfillAt,
        lastWebhookAt: input.observability.ingest?.lastWebhookAt,
        freshnessSeconds: input.observability.ingest?.freshnessSeconds ?? 0,
        lagStatus: input.observability.ingest?.lagStatus ?? "unavailable",
      },
      alertDelivery: {
        attempts24h: input.observability.alertDelivery?.attempts24h ?? 0,
        delivered24h: input.observability.alertDelivery?.delivered24h ?? 0,
        failed24h: input.observability.alertDelivery?.failed24h ?? 0,
        retryableCount: input.observability.alertDelivery?.retryableCount ?? 0,
        ...(input.observability.alertDelivery?.lastFailureAt
          ? { lastFailureAt: input.observability.alertDelivery.lastFailureAt }
          : {}),
      },
      walletTracking: {
        candidateCount: input.observability.walletTracking?.candidateCount ?? 0,
        trackedCount: input.observability.walletTracking?.trackedCount ?? 0,
        labeledCount: input.observability.walletTracking?.labeledCount ?? 0,
        scoredCount: input.observability.walletTracking?.scoredCount ?? 0,
        staleCount: input.observability.walletTracking?.staleCount ?? 0,
        suppressedCount:
          input.observability.walletTracking?.suppressedCount ?? 0,
      },
      trackingSubscriptions: {
        pendingCount:
          input.observability.trackingSubscriptions?.pendingCount ?? 0,
        activeCount:
          input.observability.trackingSubscriptions?.activeCount ?? 0,
        erroredCount:
          input.observability.trackingSubscriptions?.erroredCount ?? 0,
        pausedCount:
          input.observability.trackingSubscriptions?.pausedCount ?? 0,
        ...(input.observability.trackingSubscriptions?.lastEventAt
          ? {
              lastEventAt:
                input.observability.trackingSubscriptions.lastEventAt,
            }
          : {}),
      },
      queueDepth: {
        defaultDepth: input.observability.queueDepth?.defaultDepth ?? 0,
        priorityDepth: input.observability.queueDepth?.priorityDepth ?? 0,
      },
      backfillHealth: {
        jobs24h: input.observability.backfillHealth?.jobs24h ?? 0,
        activities24h: input.observability.backfillHealth?.activities24h ?? 0,
        transactions24h:
          input.observability.backfillHealth?.transactions24h ?? 0,
        expansions24h: input.observability.backfillHealth?.expansions24h ?? 0,
        ...(input.observability.backfillHealth?.lastSuccessAt
          ? {
              lastSuccessAt: input.observability.backfillHealth.lastSuccessAt,
            }
          : {}),
      },
      staleRefresh: {
        attempts24h: input.observability.staleRefresh?.attempts24h ?? 0,
        succeeded24h: input.observability.staleRefresh?.succeeded24h ?? 0,
        productive24h: input.observability.staleRefresh?.productive24h ?? 0,
        ...(input.observability.staleRefresh?.lastHitAt
          ? { lastHitAt: input.observability.staleRefresh.lastHitAt }
          : {}),
      },
      recentRuns: (input.observability.recentRuns ?? []).map((item) => ({
        jobName: item.jobName,
        lastStatus: item.lastStatus,
        lastStartedAt: item.lastStartedAt,
        ...(item.lastFinishedAt ? { lastFinishedAt: item.lastFinishedAt } : {}),
        ...(item.lastSuccessAt ? { lastSuccessAt: item.lastSuccessAt } : {}),
        minutesSinceSuccess: item.minutesSinceSuccess,
        ...(item.lastError ? { lastError: item.lastError } : {}),
      })),
      recentFailures: (input.observability.recentFailures ?? []).map(
        (item) => ({
          source: item.source,
          kind: item.kind,
          occurredAt: item.occurredAt,
          summary: item.summary,
          details: item.details ?? {},
        }),
      ),
    },
    curatedLists: (input.curatedLists.items ?? []).map((item) => ({
      id: item.id,
      name: item.name,
      notes: item.notes ?? "",
      tags: item.tags ?? [],
      itemCount: item.itemCount,
      items: (item.items ?? []).map((curatedItem) => ({
        id: curatedItem.id,
        itemType: curatedItem.itemType,
        itemKey: curatedItem.itemKey,
        tags: curatedItem.tags ?? [],
        ...(curatedItem.notes ? { notes: curatedItem.notes } : {}),
        updatedAt: curatedItem.updatedAt,
      })),
      updatedAt: item.updatedAt,
    })),
    auditLogs: (input.auditLogs.items ?? [])
      .slice()
      .sort((left, right) => right.createdAt.localeCompare(left.createdAt))
      .map((item) => ({
        actor: item.actor,
        action: item.action,
        targetType: item.targetType,
        targetKey: item.targetKey,
        ...(item.note ? { note: item.note } : {}),
        createdAt: item.createdAt,
      })),
  };
}

function createUnavailableWalletSummaryPreview(
  request: WalletSummaryRequest = walletSummaryRequest,
): WalletSummaryPreview {
  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    route: walletSummaryRoute,
    chain: request.chain === "evm" ? "EVM" : "SOLANA",
    chainLabel: formatChainLabel(request.chain),
    address: request.address,
    label: request.address
      ? compactAddress(request.address)
      : "Search a wallet",
    counterparties: 0,
    statusMessage:
      "Live wallet summary is not available yet. Background indexing or API recovery may still be in progress.",
    topCounterparties: [],
    recentFlow: {
      incomingTxCount7d: 0,
      outgoingTxCount7d: 0,
      incomingTxCount30d: 0,
      outgoingTxCount30d: 0,
      netDirection7d: "balanced",
      netDirection30d: "balanced",
    },
    indexing: {
      status: "indexing",
      lastIndexedAt: "",
      coverageStartAt: "",
      coverageEndAt: "",
      coverageWindowDays: 0,
    },
    latestSignals: [],
    scores: [],
  };
}

function createUnavailableWalletBriefPreview(
  request: WalletBriefRequest = walletBriefRequest,
): WalletBriefPreview {
  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    route: walletBriefRoute,
    chain: request.chain,
    address: request.address,
    displayName: request.address
      ? compactAddress(request.address)
      : "Search a wallet",
    statusMessage:
      "Live wallet brief is unavailable until the wallet brief API responds.",
    aiSummary:
      "Live wallet brief is unavailable until the wallet brief API responds.",
    keyFindings: [],
    verifiedLabels: [],
    probableLabels: [],
    behavioralLabels: [],
    topCounterparties: [],
    recentFlow: {
      incomingTxCount7d: 0,
      outgoingTxCount7d: 0,
      incomingTxCount30d: 0,
      outgoingTxCount30d: 0,
      netDirection7d: "balanced",
      netDirection30d: "balanced",
    },
    indexing: {
      status: "indexing",
      lastIndexedAt: "",
      coverageStartAt: "",
      coverageEndAt: "",
      coverageWindowDays: 0,
    },
    latestSignals: [],
    scores: [],
  };
}

function createUnavailableAnalystWalletBriefPreview(
  request: WalletBriefRequest = walletBriefRequest,
): WalletBriefPreview {
  return {
    ...createUnavailableWalletBriefPreview(request),
    route: analystWalletBriefRoute,
  };
}

function createUnavailableFindingsFeedPreview(
  request: FindingsFeedRequest = findingsFeedRequest,
): FindingsFeedPreview {
  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    route: findingsFeedRoute,
    generatedAt: "",
    statusMessage:
      "Live findings feed is unavailable until the findings API responds.",
    items: [],
    ...(request.cursor ? { nextCursor: request.cursor } : {}),
    hasMore: false,
  };
}

function createUnavailableAnalystFindingsPreview(
  request: FindingsFeedRequest = findingsFeedRequest,
): FindingsFeedPreview {
  return {
    ...createUnavailableFindingsFeedPreview(request),
    route: analystFindingsRoute,
  };
}

function createUnavailableEntityInterpretationPreview(
  request: EntityInterpretationRequest = entityInterpretationRequest,
): EntityInterpretationPreview {
  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    route: entityInterpretationRoute,
    entityKey: request.entityKey,
    entityType: "unknown",
    displayName: request.entityKey,
    walletCount: 0,
    statusMessage:
      "Live entity interpretation is unavailable until the entity API responds.",
    members: [],
    findings: [],
  };
}

function createUnavailableAnalystEntityInterpretationPreview(
  request: EntityInterpretationRequest = entityInterpretationRequest,
): EntityInterpretationPreview {
  return {
    ...createUnavailableEntityInterpretationPreview(request),
    route: analystEntityInterpretationRoute,
  };
}

function createUnavailableClusterDetailPreview(
  request: ClusterDetailRequest = clusterDetailRequest,
): ClusterDetailPreview {
  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    route: clusterDetailRoute,
    clusterId: request.clusterId,
    label: request.clusterId,
    clusterType: "unavailable",
    classification: "emerging",
    score: 0,
    memberCount: 0,
    members: [],
    commonActions: [],
    evidence: [],
    statusMessage:
      "Live cluster detail is unavailable until the cluster API responds.",
  };
}

function createUnavailableShadowExitFeedPreview(): ShadowExitFeedPreview {
  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    route: shadowExitFeedRoute,
    windowLabel: "Last 24 hours",
    itemCount: 0,
    highPriorityCount: 0,
    latestObservedAt: "",
    statusMessage:
      "Live shadow exit feed is unavailable until signal data is ready.",
    items: [],
  };
}

function createUnavailableFirstConnectionFeedPreview(): FirstConnectionFeedPreview {
  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    route: firstConnectionFeedRoute,
    sort: "latest",
    windowLabel: "Last 24 hours",
    itemCount: 0,
    highPriorityCount: 0,
    latestObservedAt: "",
    statusMessage:
      "Live first-connection feed is unavailable until signal data is ready.",
    items: [],
  };
}

function createUnavailableWalletGraphPreview(
  request: WalletGraphRequest = walletGraphRequest,
): WalletGraphPreview {
  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    route: walletGraphRoute,
    chain: request.chain === "evm" ? "EVM" : "SOLANA",
    address: request.address,
    depthRequested: request.depthRequested,
    depthResolved: 0,
    densityCapped: false,
    statusMessage:
      "Live relationship data is not available yet. The graph will appear after indexing or when the graph API responds.",
    nodes: [],
    edges: [],
    neighborhoodSummary: buildWalletGraphNeighborhoodSummaryPreview({
      nodes: [],
      edges: [],
    }),
  };
}

function buildWalletGraphNeighborhoodSummaryPreview({
  nodes,
  edges,
}: {
  nodes: WalletGraphPreviewNode[];
  edges: WalletGraphPreviewEdge[];
}): WalletGraphNeighborhoodSummaryPreview {
  let latestObservedAt: string | undefined;
  let totalInteractionWeight = 0;
  let interactionEdgeCount = 0;

  for (const edge of edges) {
    if (edge.kind === "interacted_with") {
      interactionEdgeCount += 1;
      totalInteractionWeight += edge.weight ?? edge.counterpartyCount ?? 0;
    }

    const observedAt = edge.observedAt;
    if (observedAt && (!latestObservedAt || observedAt > latestObservedAt)) {
      latestObservedAt = observedAt;
    }
  }

  return {
    neighborNodeCount: Math.max(nodes.length - 1, 0),
    walletNodeCount: nodes.filter((node) => node.kind === "wallet").length,
    clusterNodeCount: nodes.filter((node) => node.kind === "cluster").length,
    entityNodeCount: nodes.filter((node) => node.kind === "entity").length,
    interactionEdgeCount,
    totalInteractionWeight,
    ...(latestObservedAt ? { latestObservedAt } : {}),
  };
}

function walletGraphEdgeFamilyForKind(
  kind: WalletGraphPreviewEdge["kind"],
): WalletGraphPreviewEdge["family"] {
  if (kind === "interacted_with") {
    return "base";
  }

  return "derived";
}

function walletGraphEdgeDirectionalityFor({
  kind,
  tokenFlow,
  evidence,
}: Pick<
  WalletGraphPreviewEdge,
  "kind" | "tokenFlow" | "evidence"
>): NonNullable<WalletGraphPreviewEdge["directionality"]> {
  if (kind === "funded_by") {
    return "received";
  }

  if (kind !== "interacted_with") {
    return "linked";
  }

  const inboundCount = tokenFlow?.inboundCount ?? 0;
  const outboundCount = tokenFlow?.outboundCount ?? 0;
  if (inboundCount > 0 && outboundCount > 0) {
    return "mixed";
  }
  if (outboundCount > 0) {
    return "sent";
  }
  if (inboundCount > 0) {
    return "received";
  }
  if (evidence?.lastDirection === "outbound") {
    return "sent";
  }
  if (evidence?.lastDirection === "inbound") {
    return "received";
  }

  return "mixed";
}

function createUnavailableSearchPreview(query: string): SearchPreview {
  const trimmed = query.trim();
  const classification = classifySearchQuery(trimmed);

  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    route: searchRoute,
    query: trimmed,
    inputKind: classification.inputKind,
    kindLabel: classification.kindLabel,
    chainLabel: classification.chainLabel,
    title: classification.title,
    explanation: classification.explanation,
    ...(classification.walletRoute
      ? { walletRoute: classification.walletRoute }
      : {}),
    navigation: classification.navigation,
  };
}

function createUnavailableAlertCenterPreview(
  severity: AlertCenterPreview["activeSeverityFilter"] = "all",
  signalType: AlertCenterPreview["activeSignalFilter"] = "all",
  status: AlertCenterPreview["activeStatusFilter"] = "all",
  cursor?: string,
): AlertCenterPreview {
  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    inboxRoute: alertInboxRoute,
    rulesRoute: alertRulesCollectionRoute,
    channelsRoute: alertDeliveryChannelsRoute,
    activeSeverityFilter: severity,
    activeSignalFilter: signalType,
    activeStatusFilter: status,
    statusMessage:
      "Live alert inbox and delivery channels are unavailable until the alert APIs respond.",
    nextCursor: cursor,
    hasMore: false,
    unreadCount: 0,
    inbox: [],
    rules: [],
    channels: [],
  };
}

function createUnavailableAdminConsolePreview(): AdminConsolePreview {
  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    labelsRoute: adminLabelsRoute,
    suppressionsRoute: adminSuppressionsRoute,
    quotasRoute: adminProviderQuotasRoute,
    observabilityRoute: adminObservabilityRoute,
    domesticPrelistingRoute: adminDomesticPrelistingRoute,
    curatedListsRoute: adminCuratedListsRoute,
    auditLogsRoute: adminAuditLogsRoute,
    backtestOps: {
      route: adminBacktestsRoute,
      statusMessage:
        "Backtest operations are unavailable until the admin backtest APIs respond.",
      checks: [],
    },
    statusMessage:
      "Live admin data is unavailable until the admin APIs respond.",
    labels: [],
    suppressions: [],
    quotas: [],
    domesticPrelisting: [],
    curatedLists: [],
    auditLogs: [],
    observability: {
      providerUsage: [],
      ingest: {
        freshnessSeconds: 0,
        lagStatus: "unavailable",
      },
      alertDelivery: {
        attempts24h: 0,
        delivered24h: 0,
        failed24h: 0,
        retryableCount: 0,
      },
      walletTracking: {
        candidateCount: 0,
        trackedCount: 0,
        labeledCount: 0,
        scoredCount: 0,
        staleCount: 0,
        suppressedCount: 0,
      },
      trackingSubscriptions: {
        pendingCount: 0,
        activeCount: 0,
        erroredCount: 0,
        pausedCount: 0,
      },
      queueDepth: {
        defaultDepth: 0,
        priorityDepth: 0,
      },
      backfillHealth: {
        jobs24h: 0,
        activities24h: 0,
        transactions24h: 0,
        expansions24h: 0,
      },
      staleRefresh: {
        attempts24h: 0,
        succeeded24h: 0,
        productive24h: 0,
      },
      recentRuns: [],
      recentFailures: [],
    },
  };
}

export function getWalletSummaryPreview(
  request: WalletSummaryRequest = walletSummaryRequest,
): WalletSummaryPreview {
  return createUnavailableWalletSummaryPreview(request);
}

export function getWalletBriefPreview(
  request: WalletBriefRequest = walletBriefRequest,
): WalletBriefPreview {
  return createUnavailableWalletBriefPreview(request);
}

export function getAnalystWalletBriefPreview(
  request: WalletBriefRequest = walletBriefRequest,
): WalletBriefPreview {
  return createUnavailableAnalystWalletBriefPreview(request);
}

export function getFindingsFeedPreview(
  request: FindingsFeedRequest = findingsFeedRequest,
): FindingsFeedPreview {
  return createUnavailableFindingsFeedPreview(request);
}

export function getAnalystFindingsPreview(
  request: FindingsFeedRequest = findingsFeedRequest,
): FindingsFeedPreview {
  return createUnavailableAnalystFindingsPreview(request);
}

export function getEntityInterpretationPreview(
  request: EntityInterpretationRequest = entityInterpretationRequest,
): EntityInterpretationPreview {
  return createUnavailableEntityInterpretationPreview(request);
}

export function getAnalystEntityInterpretationPreview(
  request: EntityInterpretationRequest = entityInterpretationRequest,
): EntityInterpretationPreview {
  return createUnavailableAnalystEntityInterpretationPreview(request);
}

export function getClusterDetailPreview(
  request: ClusterDetailRequest = clusterDetailRequest,
): ClusterDetailPreview {
  return createUnavailableClusterDetailPreview(request);
}

export function getShadowExitFeedPreview(): ShadowExitFeedPreview {
  return createUnavailableShadowExitFeedPreview();
}

export function getFirstConnectionFeedPreview(): FirstConnectionFeedPreview {
  return createUnavailableFirstConnectionFeedPreview();
}

export function getWalletGraphPreview(
  request: WalletGraphRequest = walletGraphRequest,
): WalletGraphPreview {
  return createUnavailableWalletGraphPreview(request);
}

export function deriveWalletGraphPreviewFromSummary({
  request,
  summary,
  fallback,
}: {
  request: WalletGraphRequest;
  summary: WalletSummaryPreview;
  fallback?: WalletGraphPreview;
}): WalletGraphPreview {
  const clusterNodeId = summary.clusterId ? `cluster:${summary.clusterId}` : "";
  const nodes: WalletGraphPreviewNode[] = [
    {
      id: "wallet_root",
      kind: "wallet",
      label: summary.label,
      chain: request.chain,
      address: request.address,
    },
  ];
  const edges: WalletGraphPreviewEdge[] = [];
  const seenEntityNodes = new Set<string>();

  if (clusterNodeId) {
    nodes.push({
      id: clusterNodeId,
      kind: "cluster",
      label: summary.clusterId ?? "cluster",
    });
    edges.push({
      sourceId: "wallet_root",
      targetId: clusterNodeId,
      kind: "member_of",
      family: "derived",
      directionality: "linked",
    });
  }

  for (const counterparty of summary.topCounterparties) {
    const nodeId = `wallet:${counterparty.chain}:${counterparty.address}`;
    nodes.push({
      id: nodeId,
      kind: "wallet",
      label: compactAddress(counterparty.address),
      chain: counterparty.chain,
      address: counterparty.address,
    });
    edges.push({
      sourceId: "wallet_root",
      targetId: nodeId,
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
        source: "summary-derived",
        confidence:
          counterparty.interactionCount >= 8
            ? "high"
            : counterparty.interactionCount >= 3
              ? "medium"
              : "low",
        summary:
          counterparty.directionLabel === "inbound"
            ? `Summary-derived funding signal across ${counterparty.interactionCount} transfers.`
            : `Summary-derived transfer activity across ${counterparty.interactionCount} transfers.`,
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

    if (counterparty.entityLabel && counterparty.entityKey) {
      const entityNodeId = `entity:${counterparty.entityKey}`;
      if (!seenEntityNodes.has(entityNodeId)) {
        nodes.push({
          id: entityNodeId,
          kind: "entity",
          label: counterparty.entityLabel,
        });
        seenEntityNodes.add(entityNodeId);
      }
      edges.push({
        sourceId: nodeId,
        targetId: entityNodeId,
        kind: "entity_linked",
        family: "derived",
        directionality: "linked",
        evidence: {
          source: counterparty.entityKey.startsWith("heuristic:")
            ? "provider-heuristic-identity"
            : counterparty.entityKey.startsWith("curated:")
              ? "curated-identity-index"
              : "summary-derived",
          confidence: "medium",
          summary: `Summary-derived entity linkage to ${counterparty.entityLabel}.`,
        },
      });
    }
  }

  const latestObservedAt = summary.topCounterparties
    .map((item) => item.latestActivityAt)
    .filter(Boolean)
    .sort()
    .at(-1);

  return {
    mode: "live",
    source: "summary-derived",
    route: walletGraphRoute,
    chain: summary.chain,
    address: request.address,
    depthRequested: request.depthRequested,
    depthResolved: 1,
    densityCapped: false,
    statusMessage:
      "Graph preview derived from live wallet summary counterparties while the neighborhood graph is unavailable.",
    ...(fallback?.snapshot ? { snapshot: fallback.snapshot } : {}),
    neighborhoodSummary: {
      neighborNodeCount: Math.max(nodes.length - 1, 0),
      walletNodeCount: 1 + summary.topCounterparties.length,
      clusterNodeCount: clusterNodeId ? 1 : 0,
      entityNodeCount: seenEntityNodes.size,
      interactionEdgeCount: edges.filter(
        (edge) => edge.kind !== "entity_linked",
      ).length,
      totalInteractionWeight: summary.topCounterparties.reduce(
        (sum, item) => sum + item.interactionCount,
        0,
      ),
      ...(latestObservedAt ? { latestObservedAt } : {}),
    },
    nodes,
    edges,
  };
}

export function getSearchPreview(query = ""): SearchPreview {
  return createUnavailableSearchPreview(query);
}

export function getAlertCenterPreview(
  options: {
    severity?: AlertCenterPreview["activeSeverityFilter"];
    signalType?: AlertCenterPreview["activeSignalFilter"];
    status?: AlertCenterPreview["activeStatusFilter"];
    cursor?: string;
  } = {},
): AlertCenterPreview {
  return createUnavailableAlertCenterPreview(
    options.severity ?? "all",
    options.signalType ?? "all",
    options.status ?? "all",
    options.cursor,
  );
}

export function getAdminConsolePreview(): AdminConsolePreview {
  return createUnavailableAdminConsolePreview();
}

export async function loadWalletSummaryPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  request = walletSummaryRequest,
  requestHeaders,
}: LoadWalletSummaryPreviewOptions = {}): Promise<WalletSummaryPreview> {
  const nextFallback =
    fallback ?? createUnavailableWalletSummaryPreview(request);
  const endpoint = buildWalletSummaryUrl(request, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as WalletSummaryEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapWalletSummaryResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function loadWalletBriefPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  request = walletBriefRequest,
  requestHeaders,
}: LoadWalletBriefPreviewOptions = {}): Promise<WalletBriefPreview> {
  const nextFallback = fallback ?? createUnavailableWalletBriefPreview(request);
  const endpoint = buildWalletBriefUrl(request, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as WalletBriefEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapWalletBriefResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function loadAnalystWalletBriefPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  request = walletBriefRequest,
  requestHeaders,
}: LoadWalletBriefPreviewOptions = {}): Promise<WalletBriefPreview> {
  const nextFallback =
    fallback ?? createUnavailableAnalystWalletBriefPreview(request);
  const endpoint = buildAnalystWalletBriefUrl(request, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as WalletBriefEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapWalletBriefResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function explainAnalystWallet({
  apiBaseUrl,
  fetchImpl = fetch,
  request = walletBriefRequest,
  requestHeaders,
  question,
  forceRefresh,
  async,
}: ExplainAnalystWalletOptions = {}): Promise<AnalystWalletExplanationPreview> {
  const endpoint = buildAnalystWalletExplainUrl(request, apiBaseUrl);
  const response = await fetchImpl(endpoint, {
    method: "POST",
    headers: mergeRequestHeaders(
      {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      requestHeaders,
    ),
    body: JSON.stringify({
      ...(question ? { question } : {}),
      ...(forceRefresh ? { forceRefresh } : {}),
      ...(async ? { async } : {}),
    }),
  });

  if (!response.ok) {
    throw Object.assign(new Error("wallet explain request failed"), {
      status: response.status,
    });
  }

  const payload = (await response.json()) as AnalystWalletExplanationEnvelope;
  if (!payload.success || !payload.data) {
    throw Object.assign(new Error("wallet explain request failed"), {
      status: response.status,
    });
  }

  return payload.data;
}

export async function analyzeAnalystWallet({
  apiBaseUrl,
  fetchImpl = fetch,
  request = walletBriefRequest,
  requestHeaders,
  question,
  recentTurns,
}: AnalyzeAnalystWalletOptions = {}): Promise<AnalystWalletAnalyzePreview> {
  const endpoint = buildAnalystWalletAnalyzeUrl(request, apiBaseUrl);
  const response = await fetchImpl(endpoint, {
    method: "POST",
    headers: mergeRequestHeaders(
      {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      requestHeaders,
    ),
    body: JSON.stringify({
      ...(question ? { question } : {}),
      ...(recentTurns && recentTurns.length > 0 ? { recentTurns } : {}),
    }),
  });

  if (!response.ok) {
    throw Object.assign(new Error("wallet analyze request failed"), {
      status: response.status,
    });
  }

  const payload = (await response.json()) as AnalystWalletAnalyzeEnvelope;
  if (!payload.success || !payload.data) {
    throw Object.assign(new Error("wallet analyze request failed"), {
      status: response.status,
    });
  }

  return payload.data;
}

export async function analyzeAnalystEntity({
  apiBaseUrl,
  fetchImpl = fetch,
  request = entityInterpretationRequest,
  requestHeaders,
  question,
  recentTurns,
}: AnalyzeAnalystEntityOptions = {}): Promise<AnalystEntityAnalyzePreview> {
  const endpoint = buildAnalystEntityAnalyzeUrl(request, apiBaseUrl);
  const response = await fetchImpl(endpoint, {
    method: "POST",
    headers: mergeRequestHeaders(
      {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      requestHeaders,
    ),
    body: JSON.stringify({
      ...(question ? { question } : {}),
      ...(recentTurns && recentTurns.length > 0 ? { recentTurns } : {}),
    }),
  });

  if (!response.ok) {
    throw Object.assign(new Error("entity analyze request failed"), {
      status: response.status,
    });
  }

  const payload = (await response.json()) as AnalystEntityAnalyzeEnvelope;
  if (!payload.success || !payload.data) {
    throw Object.assign(new Error("entity analyze request failed"), {
      status: response.status,
    });
  }

  return payload.data;
}

export async function loadFindingsFeedPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  request = findingsFeedRequest,
  requestHeaders,
}: LoadFindingsFeedPreviewOptions = {}): Promise<FindingsFeedPreview> {
  const nextFallback =
    fallback ?? createUnavailableFindingsFeedPreview(request);
  const endpoint = buildFindingsFeedUrl(
    apiBaseUrl,
    request.cursor,
    request.types,
  );

  try {
    const response = await fetchImpl(endpoint, {
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as FindingsFeedEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapFindingsFeedResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function loadAnalystFindingsPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  request = findingsFeedRequest,
  requestHeaders,
}: LoadFindingsFeedPreviewOptions = {}): Promise<FindingsFeedPreview> {
  const nextFallback =
    fallback ?? createUnavailableAnalystFindingsPreview(request);
  const endpoint = buildAnalystFindingsUrl(
    apiBaseUrl,
    request.cursor,
    request.types,
  );

  try {
    const response = await fetchImpl(endpoint, {
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as FindingsFeedEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapFindingsFeedResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function loadWalletGraphPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  request = walletGraphRequest,
  requestHeaders,
}: LoadWalletGraphPreviewOptions = {}): Promise<WalletGraphPreview> {
  const nextFallback = fallback ?? createUnavailableWalletGraphPreview(request);
  const endpoint = buildWalletGraphUrl(request, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
    });

    if (response.status === 403 && request.depthRequested > 1) {
      return loadWalletGraphPreview({
        ...(apiBaseUrl ? { apiBaseUrl } : {}),
        fetchImpl,
        fallback: nextFallback,
        ...(requestHeaders ? { requestHeaders } : {}),
        request: {
          ...request,
          depthRequested: 1,
        },
      });
    }

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as WalletGraphEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapWalletGraphResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function loadClusterDetailPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  request = clusterDetailRequest,
}: LoadClusterDetailPreviewOptions = {}): Promise<ClusterDetailPreview> {
  const nextFallback =
    fallback ?? createUnavailableClusterDetailPreview(request);
  const endpoint = buildClusterDetailUrl(request, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: {
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as ClusterDetailEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapClusterDetailResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function loadEntityInterpretationPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  request = entityInterpretationRequest,
  requestHeaders,
}: LoadEntityInterpretationPreviewOptions = {}): Promise<EntityInterpretationPreview> {
  const nextFallback =
    fallback ?? createUnavailableEntityInterpretationPreview(request);
  const endpoint = buildEntityInterpretationUrl(request.entityKey, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as EntityInterpretationEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapEntityInterpretationResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function loadAnalystEntityInterpretationPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  request = entityInterpretationRequest,
  requestHeaders,
}: LoadEntityInterpretationPreviewOptions = {}): Promise<EntityInterpretationPreview> {
  const nextFallback =
    fallback ?? createUnavailableAnalystEntityInterpretationPreview(request);
  const endpoint = buildAnalystEntityInterpretationUrl(
    request.entityKey,
    apiBaseUrl,
  );

  try {
    const response = await fetchImpl(endpoint, {
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as EntityInterpretationEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapEntityInterpretationResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function loadShadowExitFeedPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
}: {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: ShadowExitFeedPreview;
} = {}): Promise<ShadowExitFeedPreview> {
  const nextFallback = fallback ?? createUnavailableShadowExitFeedPreview();
  const endpoint = buildShadowExitFeedUrl(apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: {
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as ShadowExitFeedEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapShadowExitFeedResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function loadDiscoverFeaturedWalletSeedsPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  requestHeaders,
}: {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  requestHeaders?: HeadersInit;
} = {}): Promise<DiscoverFeaturedWalletSeedPreview[]> {
  const endpoint = buildDiscoverFeaturedWalletsUrl(apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
    });

    if (!response.ok) {
      return [];
    }

    const payload = (await response.json()) as DiscoverFeaturedWalletEnvelope;
    if (!payload.success || !payload.data) {
      return [];
    }

    return Array.isArray(payload.data.items) ? payload.data.items : [];
  } catch {
    return [];
  }
}

export async function loadDiscoverDomesticPrelistingPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  requestHeaders,
}: {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  requestHeaders?: HeadersInit;
} = {}): Promise<DiscoverDomesticPrelistingCandidatePreview[]> {
  const endpoint = buildDiscoverDomesticPrelistingUrl(apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
    });

    if (!response.ok) {
      return [];
    }

    const payload = (await response.json()) as DiscoverDomesticPrelistingEnvelope;
    if (!payload.success || !payload.data) {
      return [];
    }

    return Array.isArray(payload.data.items) ? payload.data.items : [];
  } catch {
    return [];
  }
}

export async function loadFirstConnectionFeedPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  sort = "latest",
}: LoadFirstConnectionFeedPreviewOptions = {}): Promise<FirstConnectionFeedPreview> {
  const nextFallback =
    fallback ?? createUnavailableFirstConnectionFeedPreview();
  const endpoint = buildFirstConnectionFeedUrl(sort, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: {
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as FirstConnectionFeedEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapFirstConnectionFeedResponse(
      {
        ...payload.data,
        sort,
      },
      "live-api",
    );
  } catch {
    return {
      ...nextFallback,
      sort,
    };
  }
}

export async function loadSearchPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  query,
  refreshMode,
  requestHeaders,
}: LoadSearchPreviewOptions): Promise<SearchPreview> {
  const trimmed = query.trim();
  const nextFallback = fallback ?? createUnavailableSearchPreview(trimmed);
  if (!trimmed) {
    return nextFallback;
  }

  const endpoint = buildSearchUrl(trimmed, apiBaseUrl, refreshMode);

  try {
    const response = await fetchImpl(endpoint, {
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as SearchEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapSearchResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function loadAlertCenterPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  severity = "all",
  signalType = "all",
  status = "all",
  cursor,
  requestHeaders,
}: LoadAlertCenterPreviewOptions = {}): Promise<AlertCenterPreview> {
  const nextFallback =
    fallback ??
    createUnavailableAlertCenterPreview(severity, signalType, status, cursor);
  const inboxEndpoint = buildAlertInboxUrl(
    severity,
    signalType,
    status,
    cursor,
    apiBaseUrl,
  );
  const rulesEndpoint = buildAlertRulesUrl(apiBaseUrl);
  const channelsEndpoint = buildAlertDeliveryChannelsUrl(apiBaseUrl);

  try {
    const [inboxResponse, rulesResponse, channelsResponse] = await Promise.all([
      fetchImpl(inboxEndpoint, {
        headers: mergeRequestHeaders(
          { Accept: "application/json" },
          requestHeaders,
        ),
      }),
      fetchImpl(rulesEndpoint, {
        headers: mergeRequestHeaders(
          { Accept: "application/json" },
          requestHeaders,
        ),
      }),
      fetchImpl(channelsEndpoint, {
        headers: mergeRequestHeaders(
          { Accept: "application/json" },
          requestHeaders,
        ),
      }),
    ]);

    if (!inboxResponse.ok || !rulesResponse.ok || !channelsResponse.ok) {
      return nextFallback;
    }

    const [inboxPayload, rulesPayload, channelsPayload] = (await Promise.all([
      inboxResponse.json(),
      rulesResponse.json(),
      channelsResponse.json(),
    ])) as [
      AlertInboxEnvelope,
      AlertRuleCollectionEnvelope,
      AlertDeliveryChannelCollectionEnvelope,
    ];

    if (
      !inboxPayload.success ||
      !inboxPayload.data ||
      !rulesPayload.success ||
      !rulesPayload.data ||
      !channelsPayload.success ||
      !channelsPayload.data
    ) {
      return nextFallback;
    }

    return mapAlertCenterResponse({
      severity,
      signalType,
      status,
      inbox: inboxPayload.data,
      rules: rulesPayload.data,
      channels: channelsPayload.data,
    });
  } catch {
    return nextFallback;
  }
}

function buildAlertRuleMutationRequest(detail: AlertRuleApiDetail): {
  name: string;
  ruleType: string;
  isEnabled: boolean;
  cooldownSeconds: number;
  definition: {
    watchlistId: string;
    signalTypes: string[];
    minimumSeverity: "low" | "medium" | "high" | "critical";
    renotifyOnSeverityIncrease: boolean;
    snoozeUntil?: string;
  };
  notes: string;
  tags: string[];
} {
  return {
    name: detail.name,
    ruleType: detail.ruleType,
    isEnabled: detail.isEnabled,
    cooldownSeconds: detail.cooldownSeconds,
    definition: {
      watchlistId: detail.definition.watchlistId ?? "",
      signalTypes: detail.definition.signalTypes ?? [],
      minimumSeverity: detail.definition.minimumSeverity ?? "medium",
      renotifyOnSeverityIncrease:
        detail.definition.renotifyOnSeverityIncrease ?? false,
      ...(detail.definition.snoozeUntil
        ? { snoozeUntil: detail.definition.snoozeUntil }
        : {}),
    },
    notes: detail.notes ?? "",
    tags: detail.tags ?? [],
  };
}

function buildRuleSnoozeUntil(currentRule: AlertCenterRulePreview): string {
  const currentSnooze = currentRule.snoozeUntil?.trim();
  if (currentSnooze) {
    const parsed = Date.parse(currentSnooze);
    if (Number.isFinite(parsed) && parsed > Date.now()) {
      return "";
    }
  }
  return new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
}

export async function updateAlertInboxEvent({
  eventId,
  isRead,
  apiBaseUrl,
  fetchImpl = fetch,
}: UpdateAlertInboxEventOptions): Promise<AlertCenterMutationResult> {
  const endpoint = buildAlertInboxEventUrl(eventId, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      method: "PATCH",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ isRead }),
    });

    if (!response.ok) {
      return {
        ok: false,
        message: "Unable to update this alert event right now.",
      };
    }

    const payload = (await response.json()) as AlertInboxMutationEnvelope;
    if (!payload.success || !payload.data?.event) {
      return {
        ok: false,
        message: "Unable to update this alert event right now.",
      };
    }

    return {
      ok: true,
      message: isRead ? "Alert marked as read." : "Alert marked as unread.",
      event: mapAlertInboxItem(payload.data.event),
    };
  } catch {
    return {
      ok: false,
      message: "Unable to update this alert event right now.",
    };
  }
}

export async function updateAlertRuleMutation({
  ruleId,
  action,
  currentRule,
  apiBaseUrl,
  fetchImpl = fetch,
}: UpdateAlertRuleMutationOptions): Promise<AlertCenterMutationResult> {
  const endpoint = buildAlertRuleDetailUrl(ruleId, apiBaseUrl);

  try {
    const detailResponse = await fetchImpl(endpoint, {
      headers: {
        Accept: "application/json",
      },
    });
    if (!detailResponse.ok) {
      return {
        ok: false,
        message: "Unable to update this alert rule right now.",
      };
    }

    const detailPayload =
      (await detailResponse.json()) as AlertRuleDetailEnvelope;
    if (!detailPayload.success || !detailPayload.data) {
      return {
        ok: false,
        message: "Unable to update this alert rule right now.",
      };
    }

    const nextRequest = buildAlertRuleMutationRequest(detailPayload.data);
    if (action === "toggle-enabled") {
      nextRequest.isEnabled = !currentRule.isEnabled;
    } else {
      const snoozeUntil = buildRuleSnoozeUntil(currentRule);
      if (snoozeUntil) {
        nextRequest.definition.snoozeUntil = snoozeUntil;
      } else {
        nextRequest.definition = {
          watchlistId: nextRequest.definition.watchlistId,
          signalTypes: nextRequest.definition.signalTypes,
          minimumSeverity: nextRequest.definition.minimumSeverity,
          renotifyOnSeverityIncrease:
            nextRequest.definition.renotifyOnSeverityIncrease,
        };
      }
    }

    const patchResponse = await fetchImpl(endpoint, {
      method: "PATCH",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(nextRequest),
    });

    if (!patchResponse.ok) {
      return {
        ok: false,
        message: "Unable to update this alert rule right now.",
      };
    }

    const patchPayload =
      (await patchResponse.json()) as AlertRuleDetailEnvelope;
    if (!patchPayload.success || !patchPayload.data) {
      return {
        ok: false,
        message: "Unable to update this alert rule right now.",
      };
    }

    const nextRule = mapAlertRuleSummary(patchPayload.data);
    return {
      ok: true,
      message:
        action === "toggle-enabled"
          ? nextRule.isEnabled
            ? "Alert rule resumed."
            : "Alert rule muted."
          : nextRule.snoozeUntil
            ? "Alert rule snoozed for 24 hours."
            : "Alert rule snooze cleared.",
      rule: nextRule,
    };
  } catch {
    return {
      ok: false,
      message: "Unable to update this alert rule right now.",
    };
  }
}

const trackedWalletWatchlistName = "Tracked wallets";
const trackedWalletSignalTypes = [
  "cluster_score",
  "shadow_exit",
  "first_connection",
] as const;

function normalizeTrackWalletAddress(address: string): string {
  return address.trim().toLowerCase();
}

function buildTrackedWalletRuleName(label: string): string {
  const trimmed = label.trim();
  return trimmed ? `${trimmed} signal watch` : "Tracked wallet signal watch";
}

async function listWatchlists(
  apiBaseUrl: string | undefined,
  fetchImpl: typeof fetch,
  requestHeaders?: HeadersInit,
): Promise<WatchlistApiSummary[]> {
  const response = await fetchImpl(buildWatchlistsUrl(apiBaseUrl), {
    headers: mergeRequestHeaders(
      { Accept: "application/json" },
      requestHeaders,
    ),
  });
  if (!response.ok) {
    throw Object.assign(new Error("watchlist request failed"), {
      status: response.status,
    });
  }

  const payload = (await response.json()) as WatchlistCollectionEnvelope;
  if (!payload.success || !payload.data) {
    throw Object.assign(new Error("watchlist request failed"), {
      status: response.status,
    });
  }

  return payload.data.items;
}

async function getWatchlistDetail(
  watchlistId: string,
  apiBaseUrl: string | undefined,
  fetchImpl: typeof fetch,
  requestHeaders?: HeadersInit,
): Promise<WatchlistApiDetail> {
  const response = await fetchImpl(
    buildWatchlistDetailUrl(watchlistId, apiBaseUrl),
    {
      headers: mergeRequestHeaders(
        { Accept: "application/json" },
        requestHeaders,
      ),
    },
  );
  if (!response.ok) {
    throw Object.assign(new Error("watchlist detail request failed"), {
      status: response.status,
    });
  }

  const payload = (await response.json()) as WatchlistDetailEnvelope;
  if (!payload.success || !payload.data) {
    throw Object.assign(new Error("watchlist detail request failed"), {
      status: response.status,
    });
  }

  return payload.data;
}

async function createWatchlist(
  name: string,
  apiBaseUrl: string | undefined,
  fetchImpl: typeof fetch,
  requestHeaders?: HeadersInit,
): Promise<WatchlistApiDetail> {
  const response = await fetchImpl(buildWatchlistsUrl(apiBaseUrl), {
    method: "POST",
    headers: mergeRequestHeaders(
      {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      requestHeaders,
    ),
    body: JSON.stringify({ name }),
  });
  if (!response.ok) {
    throw Object.assign(new Error("watchlist create failed"), {
      status: response.status,
    });
  }

  const payload = (await response.json()) as WatchlistDetailEnvelope;
  if (!payload.success || !payload.data) {
    throw Object.assign(new Error("watchlist create failed"), {
      status: response.status,
    });
  }

  return payload.data;
}

async function createAlertRule(
  watchlistId: string,
  label: string,
  apiBaseUrl: string | undefined,
  fetchImpl: typeof fetch,
  requestHeaders?: HeadersInit,
): Promise<AlertRuleApiDetail> {
  const response = await fetchImpl(buildAlertRulesUrl(apiBaseUrl), {
    method: "POST",
    headers: mergeRequestHeaders(
      {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      requestHeaders,
    ),
    body: JSON.stringify({
      name: buildTrackedWalletRuleName(label),
      ruleType: "watchlist_signal",
      isEnabled: true,
      cooldownSeconds: 3600,
      definition: {
        watchlistId,
        signalTypes: [...trackedWalletSignalTypes],
        minimumSeverity: "medium",
        renotifyOnSeverityIncrease: true,
      },
      notes: "Created from wallet detail tracking.",
      tags: ["tracked-wallet"],
    }),
  });
  if (!response.ok) {
    throw Object.assign(new Error("alert rule create failed"), {
      status: response.status,
    });
  }

  const payload = (await response.json()) as AlertRuleDetailEnvelope;
  if (!payload.success || !payload.data) {
    throw Object.assign(new Error("alert rule create failed"), {
      status: response.status,
    });
  }

  return payload.data;
}

async function listAlertRules(
  apiBaseUrl: string | undefined,
  fetchImpl: typeof fetch,
  requestHeaders?: HeadersInit,
): Promise<AlertRuleApiSummary[]> {
  const response = await fetchImpl(buildAlertRulesUrl(apiBaseUrl), {
    headers: mergeRequestHeaders(
      { Accept: "application/json" },
      requestHeaders,
    ),
  });
  if (!response.ok) {
    throw Object.assign(new Error("alert rules request failed"), {
      status: response.status,
    });
  }

  const payload = (await response.json()) as AlertRuleCollectionEnvelope;
  if (!payload.success || !payload.data) {
    throw Object.assign(new Error("alert rules request failed"), {
      status: response.status,
    });
  }

  return payload.data.items;
}

export async function trackWalletAlertRule({
  chain,
  address,
  label,
  apiBaseUrl,
  fetchImpl = fetch,
  requestHeaders,
}: TrackWalletAlertRuleOptions): Promise<TrackWalletAlertRuleResult> {
  const normalizedAddress = normalizeTrackWalletAddress(address);
  const effectiveRequestHeaders =
    requestHeaders ?? readClientForwardedAuthHeaders();

  try {
    const watchlists = await listWatchlists(
      apiBaseUrl,
      fetchImpl,
      effectiveRequestHeaders,
    );
    let trackedWatchlist =
      watchlists.find(
        (watchlist) =>
          watchlist.name.trim().toLowerCase() ===
          trackedWalletWatchlistName.toLowerCase(),
      ) ?? null;

    if (!trackedWatchlist) {
      trackedWatchlist = await createWatchlist(
        trackedWalletWatchlistName,
        apiBaseUrl,
        fetchImpl,
        effectiveRequestHeaders,
      );
    }

    const watchlistDetail = await getWatchlistDetail(
      trackedWatchlist.id,
      apiBaseUrl,
      fetchImpl,
      effectiveRequestHeaders,
    );
    const itemAlreadyTracked = watchlistDetail.items.some(
      (item) =>
        item.chain.toLowerCase() === chain &&
        normalizeTrackWalletAddress(item.address) === normalizedAddress,
    );

    if (!itemAlreadyTracked) {
      const addItemResponse = await fetchImpl(
        buildWatchlistItemsUrl(watchlistDetail.id, apiBaseUrl),
        {
          method: "POST",
          headers: mergeRequestHeaders(
            {
              Accept: "application/json",
              "Content-Type": "application/json",
            },
            effectiveRequestHeaders,
          ),
          body: JSON.stringify({
            chain,
            address,
            tags: ["tracked-wallet"],
            note: "Added from wallet detail.",
          }),
        },
      );
      if (!addItemResponse.ok && addItemResponse.status !== 409) {
        return {
          ok: false,
          message: "Unable to add this wallet to the tracked watchlist.",
          status: addItemResponse.status,
        };
      }
    }

    const existingRules = await listAlertRules(
      apiBaseUrl,
      fetchImpl,
      effectiveRequestHeaders,
    );
    const existingRule =
      existingRules.find((rule) => {
        const signalTypes = [...(rule.definition.signalTypes ?? [])].sort();
        return (
          rule.definition.watchlistId === watchlistDetail.id &&
          signalTypes.join("|") ===
            [...trackedWalletSignalTypes].sort().join("|")
        );
      }) ?? null;

    const nextRule =
      existingRule ??
      (await createAlertRule(
        watchlistDetail.id,
        label,
        apiBaseUrl,
        fetchImpl,
        effectiveRequestHeaders,
      ));

    const params = new URLSearchParams();
    params.set("tracked", "success");
    params.set("watchlistId", watchlistDetail.id);
    params.set("ruleId", nextRule.id);
    params.set("wallet", address);

    return {
      ok: true,
      message: "Wallet tracking is active.",
      watchlistId: watchlistDetail.id,
      ruleId: nextRule.id,
      nextHref: `/alerts?${params.toString()}`,
    };
  } catch (error) {
    const status =
      typeof error === "object" &&
      error !== null &&
      "status" in error &&
      typeof (error as { status?: unknown }).status === "number"
        ? (error as { status: number }).status
        : undefined;

    if (status === 401) {
      return {
        ok: false,
        message: "Sign in is required before tracking a wallet.",
        status,
      };
    }

    if (status === 403) {
      return {
        ok: false,
        message: "Wallet tracking is temporarily unavailable right now.",
        status,
      };
    }

    return {
      ok: false,
      message: "Unable to start wallet tracking right now.",
      ...(status ? { status } : {}),
    };
  }
}

export async function loadAdminConsolePreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  requestHeaders,
}: LoadAdminConsolePreviewOptions = {}): Promise<AdminConsolePreview> {
  const fallbackPreview = fallback ?? createUnavailableAdminConsolePreview();
  const labelsEndpoint = buildAdminLabelsUrl(apiBaseUrl);
  const suppressionsEndpoint = buildAdminSuppressionsUrl(apiBaseUrl);
  const quotasEndpoint = buildAdminProviderQuotasUrl(apiBaseUrl);
  const observabilityEndpoint = buildAdminObservabilityUrl(apiBaseUrl);
  const domesticPrelistingEndpoint =
    buildAdminDomesticPrelistingUrl(apiBaseUrl);
  const curatedListsEndpoint = buildAdminCuratedListsUrl(apiBaseUrl);
  const auditLogsEndpoint = buildAdminAuditLogsUrl(apiBaseUrl);
  const backtestsEndpoint = buildAdminBacktestsUrl(apiBaseUrl);

  try {
    const [
      labelsResponse,
      suppressionsResponse,
      quotasResponse,
      observabilityResponse,
      domesticPrelistingResponse,
      curatedListsResponse,
      auditLogsResponse,
      backtestsResponse,
    ] = await Promise.all([
      fetchImpl(labelsEndpoint, {
        method: "GET",
        cache: "no-store",
        headers: mergeRequestHeaders({}, requestHeaders),
      }),
      fetchImpl(suppressionsEndpoint, {
        method: "GET",
        cache: "no-store",
        headers: mergeRequestHeaders({}, requestHeaders),
      }),
      fetchImpl(quotasEndpoint, {
        method: "GET",
        cache: "no-store",
        headers: mergeRequestHeaders({}, requestHeaders),
      }),
      fetchImpl(observabilityEndpoint, {
        method: "GET",
        cache: "no-store",
        headers: mergeRequestHeaders({}, requestHeaders),
      }),
      fetchImpl(domesticPrelistingEndpoint, {
        method: "GET",
        cache: "no-store",
        headers: mergeRequestHeaders({}, requestHeaders),
      }),
      fetchImpl(curatedListsEndpoint, {
        method: "GET",
        cache: "no-store",
        headers: mergeRequestHeaders({}, requestHeaders),
      }),
      fetchImpl(auditLogsEndpoint, {
        method: "GET",
        cache: "no-store",
        headers: mergeRequestHeaders({}, requestHeaders),
      }),
      fetchImpl(backtestsEndpoint, {
        method: "GET",
        cache: "no-store",
        headers: mergeRequestHeaders({}, requestHeaders),
      }),
    ]);

    if (
      !labelsResponse.ok ||
      !suppressionsResponse.ok ||
      !quotasResponse.ok ||
      !observabilityResponse.ok ||
      !domesticPrelistingResponse.ok ||
      !curatedListsResponse.ok ||
      !auditLogsResponse.ok ||
      !backtestsResponse.ok
    ) {
      return fallbackPreview;
    }

    const [
      labelsEnvelope,
      suppressionsEnvelope,
      quotasEnvelope,
      observabilityEnvelope,
      domesticPrelistingEnvelope,
      curatedListsEnvelope,
      auditLogsEnvelope,
      backtestsEnvelope,
    ] = (await Promise.all([
      labelsResponse.json(),
      suppressionsResponse.json(),
      quotasResponse.json(),
      observabilityResponse.json(),
      domesticPrelistingResponse.json(),
      curatedListsResponse.json(),
      auditLogsResponse.json(),
      backtestsResponse.json(),
    ])) as [
      AdminLabelCollectionEnvelope,
      AdminSuppressionCollectionEnvelope,
      AdminQuotaCollectionEnvelope,
      AdminObservabilityEnvelope,
      AdminDomesticPrelistingEnvelope,
      AdminCuratedListCollectionEnvelope,
      AdminAuditLogCollectionEnvelope,
      AdminBacktestOpsEnvelope,
    ];

    if (
      !labelsEnvelope.success ||
      !labelsEnvelope.data ||
      !suppressionsEnvelope.success ||
      !suppressionsEnvelope.data ||
      !quotasEnvelope.success ||
      !quotasEnvelope.data ||
      !observabilityEnvelope.success ||
      !observabilityEnvelope.data ||
      !domesticPrelistingEnvelope.success ||
      !domesticPrelistingEnvelope.data ||
      !curatedListsEnvelope.success ||
      !curatedListsEnvelope.data ||
      !auditLogsEnvelope.success ||
      !auditLogsEnvelope.data ||
      !backtestsEnvelope.success ||
      !backtestsEnvelope.data
    ) {
      return fallbackPreview;
    }

    return mapAdminConsoleResponse({
      labels: labelsEnvelope.data,
      suppressions: suppressionsEnvelope.data,
      quotas: quotasEnvelope.data,
      observability: observabilityEnvelope.data,
      domesticPrelisting: domesticPrelistingEnvelope.data,
      curatedLists: curatedListsEnvelope.data,
      auditLogs: auditLogsEnvelope.data,
      backtests: backtestsEnvelope.data,
    });
  } catch {
    return fallbackPreview;
  }
}

export async function createAdminSuppression({
  scope,
  target,
  reason,
  expiresAt,
  apiBaseUrl,
  fetchImpl = fetch,
}: CreateAdminSuppressionOptions): Promise<AdminConsoleMutationResult> {
  const endpoint = buildAdminSuppressionsUrl(apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        scope,
        target,
        reason,
        expiresAt: expiresAt?.trim() ?? "",
      }),
    });

    if (!response.ok) {
      return {
        ok: false,
        message: "Unable to create this suppression right now.",
      };
    }

    const payload = (await response.json()) as AdminSuppressionEnvelope;
    if (!payload.success || !payload.data) {
      return {
        ok: false,
        message: "Unable to create this suppression right now.",
      };
    }

    return {
      ok: true,
      message: "Suppression created.",
      suppression: mapAdminSuppressionItem(payload.data),
    };
  } catch {
    return {
      ok: false,
      message: "Unable to create this suppression right now.",
    };
  }
}

export async function deleteAdminSuppression({
  suppressionId,
  apiBaseUrl,
  fetchImpl = fetch,
}: DeleteAdminSuppressionOptions): Promise<AdminConsoleMutationResult> {
  const endpoint = buildAdminSuppressionDetailUrl(suppressionId, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      method: "DELETE",
      headers: {
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      return {
        ok: false,
        message: "Unable to remove this suppression right now.",
      };
    }

    const payload = (await response.json()) as AdminMutationEnvelope;
    if (!payload.success) {
      return {
        ok: false,
        message: "Unable to remove this suppression right now.",
      };
    }

    return {
      ok: true,
      message: "Suppression removed.",
      deletedSuppressionId: suppressionId,
    };
  } catch {
    return {
      ok: false,
      message: "Unable to remove this suppression right now.",
    };
  }
}

export async function runAdminBacktestOperation({
  checkKey,
  apiBaseUrl,
  fetchImpl = fetch,
  requestHeaders,
}: RunAdminBacktestOperationOptions): Promise<{
  ok: boolean;
  message: string;
  result?: AdminBacktestRunResultPreview;
}> {
  const endpoint = buildAdminBacktestRunUrl(checkKey, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      method: "POST",
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
    });

    if (!response.ok) {
      return {
        ok: false,
        message: "Unable to run this backtest operation right now.",
      };
    }

    const payload = (await response.json()) as AdminBacktestRunEnvelope;
    if (!payload.success || !payload.data) {
      return {
        ok: false,
        message: "Unable to run this backtest operation right now.",
      };
    }

    return {
      ok: true,
      message: payload.data.summary,
      result: payload.data,
    };
  } catch {
    return {
      ok: false,
      message: "Unable to run this backtest operation right now.",
    };
  }
}

function classifySearchQuery(query: string): {
  inputKind: string;
  kindLabel: string;
  chainLabel: string | undefined;
  title: string;
  explanation: string;
  walletRoute?: string;
  navigation: boolean;
} {
  if (isEVMAddress(query)) {
    return {
      inputKind: "evm_address",
      kindLabel: "EVM wallet address",
      chainLabel: "EVM",
      title: `EVM wallet ${query}`,
      explanation:
        "Address format recognized locally. Live wallet intelligence will load when the search API responds.",
      walletRoute: `/v1/wallets/evm/${query}/summary`,
      navigation: true,
    };
  }

  if (isSolanaAddress(query)) {
    return {
      inputKind: "solana_address",
      kindLabel: "Solana wallet address",
      chainLabel: "Solana",
      title: `Solana wallet ${query}`,
      explanation:
        "Address format recognized locally. Live wallet intelligence will load when the search API responds.",
      walletRoute: `/v1/wallets/solana/${query}/summary`,
      navigation: true,
    };
  }

  if (isENSLike(query)) {
    return {
      inputKind: "ens_name",
      kindLabel: "ENS-like name",
      chainLabel: undefined,
      title: query || "ENS-like query",
      explanation:
        "ENS-like input recognized locally. Resolve it to a wallet address before opening detail.",
      navigation: false,
    };
  }

  return {
    inputKind: "unknown",
    kindLabel: "Unknown input",
    chainLabel: undefined,
    title: "Unresolved query",
    explanation:
      "Enter an EVM address, Solana address, or ENS-like name to load live intelligence.",
    navigation: false,
  };
}

function isEVMAddress(query: string): boolean {
  return /^0x[0-9a-fA-F]{40}$/.test(query);
}

function isSolanaAddress(query: string): boolean {
  if (query.length < 32 || query.length > 44) {
    return false;
  }

  return /^[1-9A-HJ-NP-Za-km-z]+$/.test(query);
}

function isENSLike(query: string): boolean {
  const lowered = query.toLowerCase();

  if (!lowered.endsWith(".eth")) {
    return false;
  }

  const labels = lowered.split(".");

  if (labels.length < 2) {
    return false;
  }

  return labels.every((label) => {
    if (!label) {
      return false;
    }

    if (label.startsWith("-") || label.endsWith("-")) {
      return false;
    }

    return /^[a-z0-9-]+$/.test(label);
  });
}

function formatChainLabel(chain: "evm" | "solana"): string {
  if (chain === "evm") {
    return "EVM";
  }

  return "Solana";
}

function compactAddress(address: string): string {
  const trimmed = address.trim();
  if (trimmed.length <= 14) {
    return trimmed;
  }

  return `${trimmed.slice(0, 8)}...${trimmed.slice(-6)}`;
}

function formatSearchChainLabel(chain?: string): string | undefined {
  if (chain === "evm" || chain === "solana") {
    return formatChainLabel(chain);
  }

  return undefined;
}

function formatSearchKindLabel(kind?: string): string {
  if (!kind) {
    return "Unknown input";
  }

  if (kind === "evm_address") {
    return "EVM wallet address";
  }

  if (kind === "solana_address") {
    return "Solana wallet address";
  }

  if (kind === "ens_name") {
    return "ENS-like name";
  }

  return kind.replaceAll("_", " ");
}
