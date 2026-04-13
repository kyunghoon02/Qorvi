"use client";

import { type ReactNode, useEffect, useState } from "react";

import { Badge, Pill, type Tone } from "@qorvi/ui";

import {
  type AdminConsolePreview,
  createAdminSuppression,
  deleteAdminSuppression,
  runAdminBacktestOperation,
} from "../../lib/api-boundary";
import { useClerkRequestHeaders } from "../../lib/clerk-client-auth";
import { PageShell } from "../components/page-shell";

const toneByQuotaStatus: Record<
  AdminConsolePreview["quotas"][number]["status"],
  Tone
> = {
  healthy: "teal",
  warning: "amber",
  critical: "violet",
  exhausted: "amber",
};

const toneByObservabilityStatus: Record<
  | AdminConsolePreview["observability"]["providerUsage"][number]["status"]
  | AdminConsolePreview["observability"]["ingest"]["lagStatus"],
  Tone
> = {
  healthy: "teal",
  warning: "amber",
  critical: "violet",
  unavailable: "violet",
};

export type AdminConsoleViewModel = {
  title: string;
  explanation: string;
  statusMessage: string;
  labelsRoute: string;
  suppressionsRoute: string;
  quotasRoute: string;
  observabilityRoute: string;
  curatedListsRoute: string;
  auditLogsRoute: string;
  labels: AdminConsolePreview["labels"];
  suppressions: AdminConsolePreview["suppressions"];
  quotas: Array<
    AdminConsolePreview["quotas"][number] & {
      tone: Tone;
      usageLabel: string;
      remaining: number;
      usagePercent: number;
      reservedPercent: number;
      headroomLabel: string;
    }
  >;
  observability: {
    providerUsage: Array<
      AdminConsolePreview["observability"]["providerUsage"][number] & {
        tone: Tone;
        lastSeenLabel: string;
        errorRateLabel: string;
      }
    >;
    ingest: AdminConsolePreview["observability"]["ingest"] & {
      tone: Tone;
      freshnessLabel: string;
      activityLabel: string;
    };
    alertDelivery: AdminConsolePreview["observability"]["alertDelivery"] & {
      healthTone: Tone;
      deliveryRateLabel: string;
      lastFailureLabel: string;
    };
    walletTracking: AdminConsolePreview["observability"]["walletTracking"] & {
      trackedCoverageLabel: string;
      staleLabel: string;
      suppressedLabel: string;
    };
    trackingSubscriptions: AdminConsolePreview["observability"]["trackingSubscriptions"] & {
      healthTone: Tone;
      activeRatioLabel: string;
      pendingLabel: string;
      lastEventLabel: string;
    };
    queueDepth: AdminConsolePreview["observability"]["queueDepth"] & {
      backlogTone: Tone;
      backlogLabel: string;
      priorityLabel: string;
    };
    backfillHealth: AdminConsolePreview["observability"]["backfillHealth"] & {
      throughputLabel: string;
      expansionLabel: string;
      lastSuccessLabel: string;
    };
    staleRefresh: AdminConsolePreview["observability"]["staleRefresh"] & {
      healthTone: Tone;
      hitRateLabel: string;
      lastHitLabel: string;
    };
    recentRuns: Array<
      AdminConsolePreview["observability"]["recentRuns"][number] & {
        tone: Tone;
        successLabel: string;
      }
    >;
    recentFailures: Array<
      AdminConsolePreview["observability"]["recentFailures"][number] & {
        title: string;
      }
    >;
  };
  curatedLists: Array<
    AdminConsolePreview["curatedLists"][number] & {
      tagLabel: string;
      firstItemLabel: string;
    }
  >;
  auditLogs: Array<
    AdminConsolePreview["auditLogs"][number] & {
      actionTone: Tone;
      actionLabel: string;
      targetLabel: string;
    }
  >;
};

export function buildAdminConsoleViewModel({
  preview,
}: {
  preview: AdminConsolePreview;
}): AdminConsoleViewModel {
  return {
    title: "Admin console",
    explanation:
      "Operators can review labels, suppressions, provider health, delivery failures, curated lists, and audit history without reaching into the database directly.",
    statusMessage: preview.statusMessage,
    labelsRoute: preview.labelsRoute,
    suppressionsRoute: preview.suppressionsRoute,
    quotasRoute: preview.quotasRoute,
    observabilityRoute: preview.observabilityRoute,
    curatedListsRoute: preview.curatedListsRoute,
    auditLogsRoute: preview.auditLogsRoute,
    labels: preview.labels,
    suppressions: preview.suppressions,
    quotas: preview.quotas.map((item) => ({
      ...item,
      tone: toneByQuotaStatus[item.status],
      usageLabel: `${item.used}/${item.limit}`,
      remaining: Math.max(item.limit - item.used, 0),
      usagePercent:
        item.limit > 0 ? Math.min((item.used / item.limit) * 100, 100) : 0,
      reservedPercent:
        item.limit > 0 ? Math.min((item.reserved / item.limit) * 100, 100) : 0,
      headroomLabel:
        item.limit > 0
          ? `${Math.max(item.limit - item.used, 0)} remaining`
          : "No quota limit",
    })),
    observability: {
      providerUsage: preview.observability.providerUsage.map((item) => ({
        ...item,
        tone: toneByObservabilityStatus[item.status],
        lastSeenLabel: item.lastSeenAt
          ? `Last seen ${formatRelativeTimestamp(item.lastSeenAt)}`
          : "No recent provider activity",
        errorRateLabel:
          item.used24h > 0
            ? `${Math.round((item.error24h / item.used24h) * 100)}% error rate`
            : "No recent calls",
      })),
      ingest: {
        ...preview.observability.ingest,
        tone: toneByObservabilityStatus[preview.observability.ingest.lagStatus],
        freshnessLabel:
          preview.observability.ingest.lagStatus === "unavailable"
            ? "No ingest heartbeat yet"
            : `${Math.max(preview.observability.ingest.freshnessSeconds, 0)}s freshness`,
        activityLabel: buildIngestActivityLabel(preview.observability.ingest),
      },
      alertDelivery: {
        ...preview.observability.alertDelivery,
        healthTone:
          preview.observability.alertDelivery.failed24h > 0 ||
          preview.observability.alertDelivery.retryableCount > 0
            ? "amber"
            : "teal",
        deliveryRateLabel:
          preview.observability.alertDelivery.attempts24h > 0
            ? `${Math.round(
                (preview.observability.alertDelivery.delivered24h /
                  preview.observability.alertDelivery.attempts24h) *
                  100,
              )}% delivered`
            : "No recent delivery attempts",
        lastFailureLabel: preview.observability.alertDelivery.lastFailureAt
          ? `Last failure ${formatRelativeTimestamp(preview.observability.alertDelivery.lastFailureAt)}`
          : "No recent delivery failures",
      },
      walletTracking: {
        ...preview.observability.walletTracking,
        trackedCoverageLabel: buildTrackedCoverageLabel(
          preview.observability.walletTracking,
        ),
        staleLabel: buildStaleTrackingLabel(
          preview.observability.walletTracking,
        ),
        suppressedLabel:
          preview.observability.walletTracking.suppressedCount > 0
            ? `${preview.observability.walletTracking.suppressedCount} currently suppressed`
            : "No active tracking suppressions",
      },
      trackingSubscriptions: {
        ...preview.observability.trackingSubscriptions,
        healthTone:
          preview.observability.trackingSubscriptions.erroredCount > 0
            ? "violet"
            : preview.observability.trackingSubscriptions.pendingCount > 0
              ? "amber"
              : "teal",
        activeRatioLabel: buildTrackingSubscriptionRatioLabel(
          preview.observability.trackingSubscriptions,
        ),
        pendingLabel: buildTrackingSubscriptionPendingLabel(
          preview.observability.trackingSubscriptions,
        ),
        lastEventLabel: preview.observability.trackingSubscriptions.lastEventAt
          ? `Last subscription event ${formatRelativeTimestamp(preview.observability.trackingSubscriptions.lastEventAt)}`
          : "No recent subscription events",
      },
      queueDepth: {
        ...preview.observability.queueDepth,
        backlogTone:
          preview.observability.queueDepth.priorityDepth > 0
            ? "amber"
            : preview.observability.queueDepth.defaultDepth > 25
              ? "amber"
              : "teal",
        backlogLabel: buildQueueBacklogLabel(preview.observability.queueDepth),
        priorityLabel: buildPriorityQueueLabel(
          preview.observability.queueDepth,
        ),
      },
      backfillHealth: {
        ...preview.observability.backfillHealth,
        throughputLabel: buildBackfillThroughputLabel(
          preview.observability.backfillHealth,
        ),
        expansionLabel: buildBackfillExpansionLabel(
          preview.observability.backfillHealth,
        ),
        lastSuccessLabel: preview.observability.backfillHealth.lastSuccessAt
          ? `Last successful drain ${formatRelativeTimestamp(preview.observability.backfillHealth.lastSuccessAt)}`
          : "No successful backfill drain recorded yet",
      },
      staleRefresh: {
        ...preview.observability.staleRefresh,
        healthTone:
          preview.observability.staleRefresh.attempts24h > 0 &&
          preview.observability.staleRefresh.productive24h === 0
            ? "violet"
            : preview.observability.staleRefresh.attempts24h > 0
              ? "teal"
              : "amber",
        hitRateLabel: buildStaleRefreshHitRateLabel(
          preview.observability.staleRefresh,
        ),
        lastHitLabel: preview.observability.staleRefresh.lastHitAt
          ? `Last productive stale refresh ${formatRelativeTimestamp(preview.observability.staleRefresh.lastHitAt)}`
          : "No productive stale refresh recorded yet",
      },
      recentRuns: preview.observability.recentRuns.map((item) => ({
        ...item,
        tone:
          item.lastStatus === "succeeded"
            ? "teal"
            : item.lastStatus === "failed"
              ? "violet"
              : "amber",
        successLabel: item.lastSuccessAt
          ? `${item.minutesSinceSuccess}m since success`
          : "No successful run yet",
      })),
      recentFailures: preview.observability.recentFailures.map((item) => ({
        ...item,
        title: `${item.source} · ${item.kind}`,
      })),
    },
    curatedLists: preview.curatedLists.map((item) => ({
      ...item,
      tagLabel: item.tags.length > 0 ? item.tags.join(", ") : "untagged",
      firstItemLabel:
        item.items[0] != null
          ? `${item.items[0].itemType}: ${item.items[0].itemKey}`
          : "No curated items yet",
    })),
    auditLogs: preview.auditLogs.map((item) => ({
      ...item,
      actionTone: toneByAuditAction(item.action),
      actionLabel: formatAuditActionLabel(item.action),
      targetLabel: `${item.targetType}: ${item.targetKey}`,
    })),
  };
}

export function AdminConsoleScreen({
  preview,
}: {
  preview: AdminConsolePreview;
}) {
  const [currentPreview, setCurrentPreview] = useState(preview);
  const [mutationMessage, setMutationMessage] = useState("");
  const [pendingKey, setPendingKey] = useState("");
  const [suppressionScope, setSuppressionScope] = useState("wallet");
  const [suppressionTarget, setSuppressionTarget] = useState("");
  const [suppressionReason, setSuppressionReason] = useState("");
  const [suppressionExpiresAt, setSuppressionExpiresAt] = useState("");
  const [latestBacktestResult, setLatestBacktestResult] = useState(
    preview.backtestOps.latestResult,
  );
  const getRequestHeaders = useClerkRequestHeaders();

  useEffect(() => {
    setCurrentPreview(preview);
    setMutationMessage("");
    setPendingKey("");
    setLatestBacktestResult(preview.backtestOps.latestResult);
  }, [preview]);

  const viewModel = buildAdminConsoleViewModel({ preview: currentPreview });
  const configuredCheckCount = currentPreview.backtestOps.checks.filter(
    (item) => item.configured,
  ).length;
  const activeSuppressionCount = viewModel.suppressions.filter(
    (item) => item.active,
  ).length;
  const totalQueueDepth =
    viewModel.observability.queueDepth.defaultDepth +
    viewModel.observability.queueDepth.priorityDepth;
  const totalSubscriptions =
    viewModel.observability.trackingSubscriptions.pendingCount +
    viewModel.observability.trackingSubscriptions.activeCount +
    viewModel.observability.trackingSubscriptions.erroredCount +
    viewModel.observability.trackingSubscriptions.pausedCount;
  const labelsPreview = viewModel.labels.slice(0, 4);
  const suppressionsPreview = viewModel.suppressions.slice(0, 4);
  const quotasPreview = viewModel.quotas.slice(0, 4);
  const providerUsagePreview = viewModel.observability.providerUsage.slice(
    0,
    4,
  );
  const recentRunsPreview = viewModel.observability.recentRuns.slice(0, 4);
  const recentFailuresPreview = viewModel.observability.recentFailures.slice(
    0,
    4,
  );
  const curatedListsPreview = viewModel.curatedLists.slice(0, 4);
  const auditLogsPreview = viewModel.auditLogs.slice(0, 5);
  const backtestChecksPreview = currentPreview.backtestOps.checks.slice(0, 5);

  async function handleCreateSuppression(): Promise<void> {
    setPendingKey("suppression:create");
    setMutationMessage("");
    const requestHeaders = await getRequestHeaders();
    const result = await createAdminSuppression({
      scope: suppressionScope,
      target: suppressionTarget,
      reason: suppressionReason,
      expiresAt: suppressionExpiresAt,
      ...(requestHeaders ? { requestHeaders } : {}),
    });
    setPendingKey("");
    setMutationMessage(result.message);
    if (result.ok && result.suppression) {
      const nextSuppression = result.suppression;
      setCurrentPreview((existing) => ({
        ...existing,
        suppressions: [nextSuppression, ...existing.suppressions],
      }));
      setSuppressionTarget("");
      setSuppressionReason("");
      setSuppressionExpiresAt("");
    }
  }

  async function handleDeleteSuppression(suppressionID: string): Promise<void> {
    setPendingKey(`suppression:${suppressionID}:delete`);
    setMutationMessage("");
    const requestHeaders = await getRequestHeaders();
    const result = await deleteAdminSuppression({
      suppressionId: suppressionID,
      ...(requestHeaders ? { requestHeaders } : {}),
    });
    setPendingKey("");
    setMutationMessage(result.message);
    if (result.ok) {
      setCurrentPreview((existing) => ({
        ...existing,
        suppressions: existing.suppressions.filter(
          (item) => item.id !== suppressionID,
        ),
      }));
    }
  }

  async function handleRunBacktestCheck(checkKey: string): Promise<void> {
    setPendingKey(`backtest:${checkKey}`);
    setMutationMessage("");
    const requestHeaders = await getRequestHeaders();
    const result = await runAdminBacktestOperation({
      checkKey,
      ...(requestHeaders ? { requestHeaders } : {}),
    });
    setPendingKey("");
    setMutationMessage(result.message);
    if (result.ok && result.result) {
      setLatestBacktestResult(result.result);
    }
  }

  return (
    <PageShell>
      <div className="detail-shell">
        <section className="detail-hero alert-center-hero">
          <div className="eyebrow-row">
            <Pill tone="amber">Admin</Pill>
            <Pill tone="violet">ops console</Pill>
          </div>

          <div className="detail-hero-copy">
            <h1>{viewModel.title}</h1>
            <p>{viewModel.explanation}</p>
          </div>

          <div className="detail-identity">
            <div>
              <span>Tracked wallets</span>
              <strong>
                {viewModel.observability.walletTracking.trackedCount}
              </strong>
            </div>
            <div>
              <span>Queue backlog</span>
              <strong>{totalQueueDepth}</strong>
            </div>
            <div>
              <span>Subscriptions</span>
              <strong>{totalSubscriptions}</strong>
            </div>
            <div>
              <span>Stale refresh hit rate</span>
              <strong>
                {buildCompactStaleRefreshRate(
                  viewModel.observability.staleRefresh,
                )}
              </strong>
            </div>
            <div>
              <span>Configured checks</span>
              <strong>{configuredCheckCount}</strong>
            </div>
            <div>
              <span>Active suppressions</span>
              <strong>{activeSuppressionCount}</strong>
            </div>
            <div>
              <span>Recent failures</span>
              <strong>{viewModel.observability.recentFailures.length}</strong>
            </div>
          </div>

          <div className="detail-actions">
            <a className="search-cta" href="/">
              Back to home
            </a>
            <span className="detail-route-copy">{viewModel.labelsRoute}</span>
          </div>
          {mutationMessage ? (
            <p className="detail-route-copy" aria-live="polite">
              {mutationMessage}
            </p>
          ) : null}
        </section>

        <section className="admin-console-snapshot-grid">
          <article className="preview-card detail-card admin-console-stat-card">
            <span className="preview-kicker">Core health</span>
            <strong>{viewModel.observability.ingest.freshnessLabel}</strong>
            <p>{viewModel.observability.ingest.activityLabel}</p>
          </article>
          <article className="preview-card detail-card admin-console-stat-card">
            <span className="preview-kicker">Queue</span>
            <strong>{totalQueueDepth} jobs queued</strong>
            <p>{viewModel.observability.queueDepth.priorityLabel}</p>
          </article>
          <article className="preview-card detail-card admin-console-stat-card">
            <span className="preview-kicker">Backfill throughput</span>
            <strong>
              {viewModel.observability.backfillHealth.jobs24h} jobs / 24h
            </strong>
            <p>{viewModel.observability.backfillHealth.throughputLabel}</p>
          </article>
          <article className="preview-card detail-card admin-console-stat-card">
            <span className="preview-kicker">Subscriptions</span>
            <strong>
              {viewModel.observability.trackingSubscriptions.activeRatioLabel}
            </strong>
            <p>{viewModel.observability.trackingSubscriptions.pendingLabel}</p>
          </article>
        </section>

        <section className="admin-console-layout">
          <div className="admin-console-main">
            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">Observability</span>
                  <h2>{viewModel.observabilityRoute}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone={viewModel.observability.ingest.tone}>
                    {viewModel.observability.ingest.lagStatus}
                  </Badge>
                </div>
              </div>
              <div className="preview-status">
                <span className="preview-kicker">Operator snapshot</span>
                <p>{viewModel.observability.ingest.activityLabel}</p>
                <p>{viewModel.observability.ingest.freshnessLabel}</p>
              </div>
              <div className="admin-console-observability-grid">
                <article className="alert-inbox-item">
                  <div className="alert-inbox-topline">
                    <strong>Wallet tracking</strong>
                    <Badge tone="teal">
                      {viewModel.observability.walletTracking.trackedCount}{" "}
                      tracked
                    </Badge>
                  </div>
                  <p>
                    {
                      viewModel.observability.walletTracking
                        .trackedCoverageLabel
                    }
                  </p>
                  <p>{viewModel.observability.walletTracking.staleLabel}</p>
                  <p>
                    {viewModel.observability.walletTracking.suppressedLabel}
                  </p>
                </article>
                <article className="alert-inbox-item">
                  <div className="alert-inbox-topline">
                    <strong>Tracking subscriptions</strong>
                    <Badge
                      tone={
                        viewModel.observability.trackingSubscriptions.healthTone
                      }
                    >
                      {
                        viewModel.observability.trackingSubscriptions
                          .activeCount
                      }{" "}
                      active
                    </Badge>
                  </div>
                  <p>
                    {
                      viewModel.observability.trackingSubscriptions
                        .activeRatioLabel
                    }
                  </p>
                  <p>
                    {viewModel.observability.trackingSubscriptions.pendingLabel}
                  </p>
                  <p>
                    {
                      viewModel.observability.trackingSubscriptions
                        .lastEventLabel
                    }
                  </p>
                </article>
                <article className="alert-inbox-item">
                  <div className="alert-inbox-topline">
                    <strong>Queue depth</strong>
                    <Badge
                      tone={viewModel.observability.queueDepth.backlogTone}
                    >
                      {totalQueueDepth} queued
                    </Badge>
                  </div>
                  <p>{viewModel.observability.queueDepth.backlogLabel}</p>
                  <p>{viewModel.observability.queueDepth.priorityLabel}</p>
                </article>
                <article className="alert-inbox-item">
                  <div className="alert-inbox-topline">
                    <strong>Backfill throughput</strong>
                    <Badge tone="teal">
                      {viewModel.observability.backfillHealth.jobs24h} jobs /
                      24h
                    </Badge>
                  </div>
                  <p>
                    {viewModel.observability.backfillHealth.throughputLabel}
                  </p>
                  <p>{viewModel.observability.backfillHealth.expansionLabel}</p>
                  <p>
                    {viewModel.observability.backfillHealth.lastSuccessLabel}
                  </p>
                </article>
                <article className="alert-inbox-item">
                  <div className="alert-inbox-topline">
                    <strong>Stale refresh</strong>
                    <Badge
                      tone={viewModel.observability.staleRefresh.healthTone}
                    >
                      {viewModel.observability.staleRefresh.productive24h}{" "}
                      productive
                    </Badge>
                  </div>
                  <p>{viewModel.observability.staleRefresh.hitRateLabel}</p>
                  <p>{viewModel.observability.staleRefresh.lastHitLabel}</p>
                </article>
                <article className="alert-inbox-item">
                  <div className="alert-inbox-topline">
                    <strong>Alert delivery</strong>
                    <Badge
                      tone={viewModel.observability.alertDelivery.healthTone}
                    >
                      {viewModel.observability.alertDelivery.retryableCount}{" "}
                      retryable
                    </Badge>
                  </div>
                  <p>
                    {viewModel.observability.alertDelivery.deliveryRateLabel}
                  </p>
                  <p>
                    {viewModel.observability.alertDelivery.lastFailureLabel}
                  </p>
                </article>
              </div>

              <div className="admin-console-section-split">
                <div className="admin-console-subsection">
                  <div className="section-header">
                    <div>
                      <span className="preview-kicker">Provider health</span>
                      <h2>Provider usage</h2>
                    </div>
                    <Badge tone="teal">
                      {viewModel.observability.providerUsage.length} providers
                    </Badge>
                  </div>
                  <div className="alert-inbox-list">
                    {providerUsagePreview.map((item) => (
                      <article key={item.provider} className="alert-inbox-item">
                        <div className="alert-inbox-topline">
                          <strong>{item.provider}</strong>
                          <Badge tone={item.tone}>{item.status}</Badge>
                        </div>
                        <p>
                          {item.used24h} calls · {item.avgLatencyMs}ms avg
                          latency
                        </p>
                        <p>{item.errorRateLabel}</p>
                        <p>{item.lastSeenLabel}</p>
                      </article>
                    ))}
                    {renderPreviewOverflowNote(
                      viewModel.observability.providerUsage.length,
                      providerUsagePreview.length,
                      "provider entries",
                    )}
                  </div>
                </div>

                <div className="admin-console-subsection">
                  <div className="section-header">
                    <div>
                      <span className="preview-kicker">Execution health</span>
                      <h2>Recent runs</h2>
                    </div>
                    <Badge tone="amber">
                      {viewModel.observability.recentRuns.length} recent
                    </Badge>
                  </div>
                  <div className="alert-inbox-list">
                    {recentRunsPreview.map((item) => (
                      <article
                        key={`${item.jobName}:${item.lastStartedAt}`}
                        className="alert-inbox-item"
                      >
                        <div className="alert-inbox-topline">
                          <strong>{item.jobName}</strong>
                          <Badge tone={item.tone}>{item.lastStatus}</Badge>
                        </div>
                        <p>{item.successLabel}</p>
                        {item.lastError ? <p>{item.lastError}</p> : null}
                      </article>
                    ))}
                    {renderPreviewOverflowNote(
                      viewModel.observability.recentRuns.length,
                      recentRunsPreview.length,
                      "run entries",
                    )}
                  </div>
                </div>
              </div>

              <div className="admin-console-subsection">
                <div className="section-header">
                  <div>
                    <span className="preview-kicker">Attention queue</span>
                    <h2>Recent failures</h2>
                  </div>
                  <Badge tone="violet">
                    {viewModel.observability.recentFailures.length} failures
                  </Badge>
                </div>
                <div className="alert-inbox-list">
                  {recentFailuresPreview.map((item) => (
                    <article
                      key={`${item.source}:${item.kind}:${item.occurredAt}`}
                      className="alert-inbox-item"
                    >
                      <div className="alert-inbox-topline">
                        <strong>{item.title}</strong>
                        <Badge tone="violet">
                          {formatRelativeTimestamp(item.occurredAt)}
                        </Badge>
                      </div>
                      <p>{item.summary}</p>
                    </article>
                  ))}
                  {renderPreviewOverflowNote(
                    viewModel.observability.recentFailures.length,
                    recentFailuresPreview.length,
                    "failure entries",
                  )}
                </div>
              </div>
            </article>

            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">Backtest Ops</span>
                  <h2>{currentPreview.backtestOps.route}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone="amber">{configuredCheckCount} configured</Badge>
                </div>
              </div>
              <div className="preview-status">
                <span className="preview-kicker">Manual validation</span>
                <p>{currentPreview.backtestOps.statusMessage}</p>
              </div>
              <div className="alert-inbox-list">
                {backtestChecksPreview.map((item) => (
                  <article key={item.key} className="alert-inbox-item">
                    <div className="alert-inbox-topline">
                      <strong>{item.label}</strong>
                      <Badge
                        tone={
                          item.status === "ready"
                            ? "teal"
                            : item.status === "missing"
                              ? "amber"
                              : "violet"
                        }
                      >
                        {item.status}
                      </Badge>
                    </div>
                    <p>{item.description}</p>
                    {item.path ? (
                      <p className="detail-route-copy">{item.path}</p>
                    ) : null}
                    <div className="detail-actions">
                      <button
                        className="search-cta"
                        disabled={
                          pendingKey === `backtest:${item.key}` ||
                          !item.configured
                        }
                        onClick={() => void handleRunBacktestCheck(item.key)}
                        type="button"
                      >
                        {pendingKey === `backtest:${item.key}`
                          ? "Running..."
                          : "Run check"}
                      </button>
                    </div>
                  </article>
                ))}
                {renderPreviewOverflowNote(
                  currentPreview.backtestOps.checks.length,
                  backtestChecksPreview.length,
                  "checks",
                )}
                {latestBacktestResult ? (
                  <article className="alert-inbox-item">
                    <div className="alert-inbox-topline">
                      <strong>{latestBacktestResult.label}</strong>
                      <Badge
                        tone={
                          latestBacktestResult.status === "succeeded"
                            ? "teal"
                            : "violet"
                        }
                      >
                        {latestBacktestResult.status}
                      </Badge>
                    </div>
                    <p>{latestBacktestResult.summary}</p>
                    <div className="cluster-member-meta">
                      <span>{latestBacktestResult.executedAt}</span>
                    </div>
                  </article>
                ) : null}
              </div>
            </article>
          </div>

          <div className="admin-console-side">
            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">Suppressions</span>
                  <h2>{viewModel.suppressionsRoute}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone="amber">{activeSuppressionCount} active</Badge>
                </div>
              </div>
              <div className="preview-status">
                <span className="preview-kicker">Human override</span>
                <p>
                  Add a suppression to override downstream alerts or temporarily
                  mute a target without touching the database.
                </p>
              </div>
              <div className="cluster-action-list">
                <label className="detail-route-copy">
                  Scope
                  <select
                    value={suppressionScope}
                    onChange={(event) =>
                      setSuppressionScope(event.target.value)
                    }
                  >
                    <option value="wallet">wallet</option>
                    <option value="cluster">cluster</option>
                    <option value="entity">entity</option>
                    <option value="alert_rule">alert_rule</option>
                  </select>
                </label>
                <label className="detail-route-copy">
                  Target
                  <input
                    value={suppressionTarget}
                    onChange={(event) =>
                      setSuppressionTarget(event.target.value)
                    }
                    placeholder="wallet address, cluster id, entity key..."
                    type="text"
                  />
                </label>
                <label className="detail-route-copy">
                  Reason
                  <input
                    value={suppressionReason}
                    onChange={(event) =>
                      setSuppressionReason(event.target.value)
                    }
                    placeholder="Explain why this override is needed"
                    type="text"
                  />
                </label>
                <label className="detail-route-copy">
                  Expires at
                  <input
                    value={suppressionExpiresAt}
                    onChange={(event) =>
                      setSuppressionExpiresAt(event.target.value)
                    }
                    placeholder="2026-03-24T00:00:00Z"
                    type="text"
                  />
                </label>
                <button
                  className="search-cta"
                  disabled={
                    pendingKey === "suppression:create" ||
                    suppressionTarget.trim() === "" ||
                    suppressionReason.trim() === ""
                  }
                  onClick={() => void handleCreateSuppression()}
                  type="button"
                >
                  {pendingKey === "suppression:create"
                    ? "Saving..."
                    : "Add suppression"}
                </button>
              </div>
              <div className="alert-inbox-list">
                {suppressionsPreview.map((item) => (
                  <article key={item.id} className="alert-inbox-item">
                    <div className="alert-inbox-topline">
                      <strong>{item.scope}</strong>
                      <Badge tone={item.active ? "amber" : "teal"}>
                        {item.active ? "Active" : "Inactive"}
                      </Badge>
                    </div>
                    <p>{item.target}</p>
                    <p>{item.reason}</p>
                    <div className="cluster-member-meta">
                      <span>{item.createdBy}</span>
                      <span>{item.updatedAt}</span>
                      {item.expiresAt ? <span>{item.expiresAt}</span> : null}
                    </div>
                    <div className="detail-actions">
                      <button
                        className="search-cta"
                        disabled={
                          pendingKey === `suppression:${item.id}:delete`
                        }
                        onClick={() => void handleDeleteSuppression(item.id)}
                        type="button"
                      >
                        {pendingKey === `suppression:${item.id}:delete`
                          ? "Removing..."
                          : "Remove"}
                      </button>
                    </div>
                  </article>
                ))}
                {renderPreviewOverflowNote(
                  viewModel.suppressions.length,
                  suppressionsPreview.length,
                  "suppressions",
                )}
              </div>
            </article>

            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">Provider quota</span>
                  <h2>{viewModel.quotasRoute}</h2>
                </div>
              </div>
              <div className="alert-inbox-list">
                {quotasPreview.map((item) => (
                  <article key={item.provider} className="alert-inbox-item">
                    <div className="alert-inbox-topline">
                      <strong>{item.provider}</strong>
                      <Badge tone={item.tone}>{item.status}</Badge>
                    </div>
                    <p>{item.usageLabel}</p>
                    <p>
                      {item.usagePercent.toFixed(0)}% used ·{" "}
                      {item.reservedPercent.toFixed(0)}% reserved
                    </p>
                    <p>{item.headroomLabel}</p>
                    <p>{item.windowLabel}</p>
                    <p>Last checked {item.lastCheckedAt}</p>
                  </article>
                ))}
                {renderPreviewOverflowNote(
                  viewModel.quotas.length,
                  quotasPreview.length,
                  "quota entries",
                )}
              </div>
            </article>

            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">Labels</span>
                  <h2>{viewModel.labelsRoute}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone="teal">{viewModel.labels.length} labels</Badge>
                </div>
              </div>
              <div className="preview-status">
                <span className="preview-kicker">Data status</span>
                <p>{viewModel.statusMessage}</p>
              </div>
              <div className="alert-inbox-list">
                {labelsPreview.map((item) => (
                  <article key={item.id} className="alert-inbox-item">
                    <div className="alert-inbox-topline">
                      <strong>{item.name}</strong>
                      <Badge tone="teal">{item.color}</Badge>
                    </div>
                    <p>{item.description}</p>
                  </article>
                ))}
                {renderPreviewOverflowNote(
                  viewModel.labels.length,
                  labelsPreview.length,
                  "labels",
                )}
              </div>
            </article>

            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">Curated lists</span>
                  <h2>{viewModel.curatedListsRoute}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone="teal">
                    {viewModel.curatedLists.length} lists
                  </Badge>
                </div>
              </div>
              <div className="alert-inbox-list">
                {curatedListsPreview.map((item) => (
                  <article key={item.id} className="alert-inbox-item">
                    <div className="alert-inbox-topline">
                      <strong>{item.name}</strong>
                      <Badge tone="teal">{item.tagLabel}</Badge>
                    </div>
                    <p>{item.notes || "No notes attached yet."}</p>
                    <div className="cluster-member-meta">
                      <Pill tone="amber">{item.itemCount} items</Pill>
                      <span>{item.firstItemLabel}</span>
                      <span>{item.updatedAt}</span>
                    </div>
                  </article>
                ))}
                {renderPreviewOverflowNote(
                  viewModel.curatedLists.length,
                  curatedListsPreview.length,
                  "curated lists",
                )}
              </div>
            </article>

            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">Audit logs</span>
                  <h2>{viewModel.auditLogsRoute}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone="violet">
                    {viewModel.auditLogs.length} entries
                  </Badge>
                </div>
              </div>
              <div className="alert-inbox-list">
                {auditLogsPreview.map((item) => (
                  <article
                    key={`${item.actor}:${item.createdAt}:${item.action}`}
                    className="alert-inbox-item"
                  >
                    <div className="alert-inbox-topline">
                      <strong>{item.targetLabel}</strong>
                      <Badge tone={item.actionTone}>{item.actionLabel}</Badge>
                    </div>
                    <p>{item.note || "No audit note attached."}</p>
                    <div className="cluster-member-meta">
                      <span>{item.actor}</span>
                      <span>{item.createdAt}</span>
                    </div>
                  </article>
                ))}
                {renderPreviewOverflowNote(
                  viewModel.auditLogs.length,
                  auditLogsPreview.length,
                  "audit entries",
                )}
              </div>
            </article>
          </div>
        </section>
      </div>
    </PageShell>
  );
}

function formatAuditActionLabel(action: string): string {
  return action
    .replace(/_/g, " ")
    .split(" ")
    .map((part) => capitalizeWord(part))
    .join(" ");
}

function toneByAuditAction(action: string): Tone {
  if (
    action.includes("create") ||
    action.includes("add") ||
    action.includes("upsert")
  ) {
    return "teal";
  }

  if (action.includes("update")) {
    return "amber";
  }

  if (
    action.includes("delete") ||
    action.includes("remove") ||
    action.includes("suppress")
  ) {
    return "violet";
  }

  return "emerald";
}

function capitalizeWord(value: string): string {
  if (!value) {
    return value;
  }

  return value.charAt(0).toUpperCase() + value.slice(1);
}

function buildIngestActivityLabel(
  ingest: AdminConsolePreview["observability"]["ingest"],
): string {
  const parts: string[] = [];
  if (ingest.lastBackfillAt) {
    parts.push(`Backfill ${formatRelativeTimestamp(ingest.lastBackfillAt)}`);
  }
  if (ingest.lastWebhookAt) {
    parts.push(`Webhook ${formatRelativeTimestamp(ingest.lastWebhookAt)}`);
  }
  if (parts.length === 0) {
    return "No recent ingest activity";
  }
  return parts.join(" · ");
}

function buildTrackedCoverageLabel(
  walletTracking: AdminConsolePreview["observability"]["walletTracking"],
): string {
  const totalTrackedSurface =
    walletTracking.candidateCount +
    walletTracking.trackedCount +
    walletTracking.labeledCount +
    walletTracking.scoredCount;
  if (totalTrackedSurface === 0) {
    return "No tracked wallet workload yet";
  }
  const matureTracked =
    walletTracking.labeledCount + walletTracking.scoredCount;
  return `${matureTracked}/${totalTrackedSurface} wallets are labeled or scored`;
}

function buildStaleTrackingLabel(
  walletTracking: AdminConsolePreview["observability"]["walletTracking"],
): string {
  if (walletTracking.staleCount <= 0) {
    return "No stale tracked wallets queued for refresh";
  }
  return `${walletTracking.staleCount} tracked wallets are stale and should refresh soon`;
}

function buildTrackingSubscriptionRatioLabel(
  subscriptions: AdminConsolePreview["observability"]["trackingSubscriptions"],
): string {
  const total =
    subscriptions.pendingCount +
    subscriptions.activeCount +
    subscriptions.erroredCount +
    subscriptions.pausedCount;
  if (total <= 0) {
    return "No tracking subscriptions registered yet";
  }
  return `${subscriptions.activeCount}/${total} subscriptions are active`;
}

function buildTrackingSubscriptionPendingLabel(
  subscriptions: AdminConsolePreview["observability"]["trackingSubscriptions"],
): string {
  if (subscriptions.erroredCount > 0) {
    return `${subscriptions.erroredCount} subscriptions need repair`;
  }
  if (subscriptions.pendingCount > 0) {
    return `${subscriptions.pendingCount} subscriptions still pending activation`;
  }
  if (subscriptions.pausedCount > 0) {
    return `${subscriptions.pausedCount} subscriptions are paused`;
  }
  return "No pending subscription backlog";
}

function buildQueueBacklogLabel(
  queueDepth: AdminConsolePreview["observability"]["queueDepth"],
): string {
  return `${queueDepth.defaultDepth} jobs in default queue`;
}

function buildPriorityQueueLabel(
  queueDepth: AdminConsolePreview["observability"]["queueDepth"],
): string {
  if (queueDepth.priorityDepth > 0) {
    return `${queueDepth.priorityDepth} jobs waiting in priority queue`;
  }
  return "No priority queue backlog";
}

function buildBackfillThroughputLabel(
  backfillHealth: AdminConsolePreview["observability"]["backfillHealth"],
): string {
  if (backfillHealth.jobs24h <= 0) {
    return "No successful backfill drain jobs in the last 24 hours";
  }
  return `${backfillHealth.transactions24h} transactions and ${backfillHealth.activities24h} activities processed in the last 24 hours`;
}

function buildBackfillExpansionLabel(
  backfillHealth: AdminConsolePreview["observability"]["backfillHealth"],
): string {
  if (backfillHealth.expansions24h <= 0) {
    return "No expansion jobs were enqueued from backfill in the last 24 hours";
  }
  return `${backfillHealth.expansions24h} expansion jobs were enqueued from successful drains`;
}

function buildStaleRefreshHitRateLabel(
  staleRefresh: AdminConsolePreview["observability"]["staleRefresh"],
): string {
  if (staleRefresh.attempts24h <= 0) {
    return "No stale refresh attempts recorded in the last 24 hours";
  }
  const hitRate = Math.round(
    (staleRefresh.productive24h / staleRefresh.attempts24h) * 100,
  );
  return `${staleRefresh.productive24h}/${staleRefresh.attempts24h} stale refresh attempts were productive (${hitRate}% hit rate)`;
}

function buildCompactStaleRefreshRate(
  staleRefresh: AdminConsolePreview["observability"]["staleRefresh"],
): string {
  if (staleRefresh.attempts24h <= 0) {
    return "No runs";
  }
  const hitRate = Math.round(
    (staleRefresh.productive24h / staleRefresh.attempts24h) * 100,
  );
  return `${hitRate}%`;
}

function renderPreviewOverflowNote(
  totalCount: number,
  previewCount: number,
  label: string,
): ReactNode {
  if (totalCount <= previewCount) {
    return null;
  }

  return (
    <p className="admin-console-preview-note">
      Showing {previewCount} of {totalCount} {label}.
    </p>
  );
}

function formatRelativeTimestamp(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  const deltaMs = Date.now() - date.getTime();
  const deltaMinutes = Math.max(Math.round(deltaMs / 60000), 0);
  if (deltaMinutes < 1) {
    return "just now";
  }
  if (deltaMinutes < 60) {
    return `${deltaMinutes}m ago`;
  }
  const deltaHours = Math.round(deltaMinutes / 60);
  if (deltaHours < 24) {
    return `${deltaHours}h ago`;
  }
  return `${Math.round(deltaHours / 24)}d ago`;
}
