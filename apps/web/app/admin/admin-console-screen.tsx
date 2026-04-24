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
  domesticPrelistingRoute: string;
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
  domesticPrelisting: Array<
    AdminConsolePreview["domesticPrelisting"][number] & {
      listingLabel: string;
      activityLabel: string;
      amountLabel: string;
      recencyLabel: string;
    }
  >;
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
    title: "운영 대시보드",
    explanation:
      "운영자는 이 화면에서 라벨, 억제 규칙, 프로바이더 상태, 전송 실패, 큐 상태, 국내 미상장 후보, 큐레이션 목록, 감사 로그를 한 번에 점검할 수 있습니다.",
    statusMessage: preview.statusMessage,
    labelsRoute: preview.labelsRoute,
    suppressionsRoute: preview.suppressionsRoute,
    quotasRoute: preview.quotasRoute,
    observabilityRoute: preview.observabilityRoute,
    domesticPrelistingRoute: preview.domesticPrelistingRoute,
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
          ? `${Math.max(item.limit - item.used, 0)} 남음`
          : "제한 없음",
    })),
    observability: {
      providerUsage: preview.observability.providerUsage.map((item) => ({
        ...item,
        tone: toneByObservabilityStatus[item.status],
        lastSeenLabel: item.lastSeenAt
          ? `마지막 호출 ${formatRelativeTimestamp(item.lastSeenAt)}`
          : "최근 프로바이더 활동 없음",
        errorRateLabel:
          item.used24h > 0
            ? `에러율 ${Math.round((item.error24h / item.used24h) * 100)}%`
            : "최근 호출 없음",
      })),
      ingest: {
        ...preview.observability.ingest,
        tone: toneByObservabilityStatus[preview.observability.ingest.lagStatus],
        freshnessLabel:
          preview.observability.ingest.lagStatus === "unavailable"
            ? "수집 heartbeat 없음"
            : `신선도 ${Math.max(preview.observability.ingest.freshnessSeconds, 0)}초`,
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
              )}% 전달 성공`
            : "최근 전송 시도 없음",
        lastFailureLabel: preview.observability.alertDelivery.lastFailureAt
          ? `마지막 실패 ${formatRelativeTimestamp(preview.observability.alertDelivery.lastFailureAt)}`
          : "최근 전송 실패 없음",
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
            ? `${preview.observability.walletTracking.suppressedCount}개 억제 적용 중`
            : "활성 추적 억제 없음",
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
          ? `마지막 구독 이벤트 ${formatRelativeTimestamp(preview.observability.trackingSubscriptions.lastEventAt)}`
          : "최근 구독 이벤트 없음",
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
          ? `마지막 성공 drain ${formatRelativeTimestamp(preview.observability.backfillHealth.lastSuccessAt)}`
          : "성공한 backfill drain 기록 없음",
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
          ? `마지막 유효 stale refresh ${formatRelativeTimestamp(preview.observability.staleRefresh.lastHitAt)}`
          : "유효 stale refresh 기록 없음",
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
          ? `${item.minutesSinceSuccess}분 전 성공`
          : "성공한 실행 없음",
      })),
      recentFailures: preview.observability.recentFailures.map((item) => ({
        ...item,
        title: `${item.source} · ${item.kind}`,
      })),
    },
    domesticPrelisting: preview.domesticPrelisting.map((item) => ({
      ...item,
      listingLabel: buildDomesticListingLabel(item),
      activityLabel: `${item.activeWalletCount}개 지갑 · 추적 ${item.trackedWalletCount}개 · 24시간 ${item.transferCount24h}건`,
      amountLabel: `7일 누적 ${formatTokenAmount(item.totalAmount)} ${item.tokenSymbol} · 최대 단일 이동 ${formatTokenAmount(item.largestTransferAmount)} ${item.tokenSymbol}`,
      recencyLabel: formatRelativeTimestamp(item.latestObservedAt),
    })),
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
  const domesticPrelistingPreview = viewModel.domesticPrelisting.slice(0, 6);
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
              <span>추적 지갑</span>
              <strong>
                {viewModel.observability.walletTracking.trackedCount}
              </strong>
            </div>
            <div>
              <span>큐 적체</span>
              <strong>{totalQueueDepth}</strong>
            </div>
            <div>
              <span>구독 수</span>
              <strong>{totalSubscriptions}</strong>
            </div>
            <div>
              <span>Stale refresh 적중률</span>
              <strong>
                {buildCompactStaleRefreshRate(
                  viewModel.observability.staleRefresh,
                )}
              </strong>
            </div>
            <div>
              <span>설정된 점검</span>
              <strong>{configuredCheckCount}</strong>
            </div>
            <div>
              <span>활성 억제</span>
              <strong>{activeSuppressionCount}</strong>
            </div>
            <div>
              <span>최근 실패</span>
              <strong>{viewModel.observability.recentFailures.length}</strong>
            </div>
          </div>

          <div className="detail-actions">
            <a className="search-cta" href="/">
              홈으로
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
            <span className="preview-kicker">핵심 상태</span>
            <strong>{viewModel.observability.ingest.freshnessLabel}</strong>
            <p>{viewModel.observability.ingest.activityLabel}</p>
          </article>
          <article className="preview-card detail-card admin-console-stat-card">
            <span className="preview-kicker">큐</span>
            <strong>{totalQueueDepth}건 대기 중</strong>
            <p>{viewModel.observability.queueDepth.priorityLabel}</p>
          </article>
          <article className="preview-card detail-card admin-console-stat-card">
            <span className="preview-kicker">Backfill 처리량</span>
            <strong>
              {viewModel.observability.backfillHealth.jobs24h}건 / 24시간
            </strong>
            <p>{viewModel.observability.backfillHealth.throughputLabel}</p>
          </article>
          <article className="preview-card detail-card admin-console-stat-card">
            <span className="preview-kicker">구독</span>
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
                  <span className="preview-kicker">관측 지표</span>
                  <h2>{viewModel.observabilityRoute}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone={viewModel.observability.ingest.tone}>
                    {viewModel.observability.ingest.lagStatus}
                  </Badge>
                </div>
              </div>
              <div className="preview-status">
                <span className="preview-kicker">운영 스냅샷</span>
                <p>{viewModel.observability.ingest.activityLabel}</p>
                <p>{viewModel.observability.ingest.freshnessLabel}</p>
              </div>
              <div className="admin-console-observability-grid">
                <article className="alert-inbox-item">
                  <div className="alert-inbox-topline">
                    <strong>지갑 추적</strong>
                    <Badge tone="teal">
                      {viewModel.observability.walletTracking.trackedCount}개
                      추적 중
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
                    <strong>추적 구독</strong>
                    <Badge
                      tone={
                        viewModel.observability.trackingSubscriptions.healthTone
                      }
                    >
                      {
                        viewModel.observability.trackingSubscriptions
                          .activeCount
                      }{" "}
                      활성
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
                    <strong>큐 깊이</strong>
                    <Badge
                      tone={viewModel.observability.queueDepth.backlogTone}
                    >
                      {totalQueueDepth}건 대기
                    </Badge>
                  </div>
                  <p>{viewModel.observability.queueDepth.backlogLabel}</p>
                  <p>{viewModel.observability.queueDepth.priorityLabel}</p>
                </article>
                <article className="alert-inbox-item">
                  <div className="alert-inbox-topline">
                    <strong>Backfill 처리량</strong>
                    <Badge tone="teal">
                      {viewModel.observability.backfillHealth.jobs24h}건 /
                      24시간
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
                      {viewModel.observability.staleRefresh.productive24h}건
                      유효
                    </Badge>
                  </div>
                  <p>{viewModel.observability.staleRefresh.hitRateLabel}</p>
                  <p>{viewModel.observability.staleRefresh.lastHitLabel}</p>
                </article>
                <article className="alert-inbox-item">
                  <div className="alert-inbox-topline">
                    <strong>알림 전송</strong>
                    <Badge
                      tone={viewModel.observability.alertDelivery.healthTone}
                    >
                      {viewModel.observability.alertDelivery.retryableCount}건
                      재시도 가능
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
                      <span className="preview-kicker">프로바이더 상태</span>
                      <h2>프로바이더 사용량</h2>
                    </div>
                    <Badge tone="teal">
                      {viewModel.observability.providerUsage.length}개
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
                          호출 {item.used24h}회 · 평균 {item.avgLatencyMs}ms
                        </p>
                        <p>{item.errorRateLabel}</p>
                        <p>{item.lastSeenLabel}</p>
                      </article>
                    ))}
                    {renderPreviewOverflowNote(
                      viewModel.observability.providerUsage.length,
                      providerUsagePreview.length,
                      "프로바이더 항목",
                    )}
                  </div>
                </div>

                <div className="admin-console-subsection">
                  <div className="section-header">
                    <div>
                      <span className="preview-kicker">실행 상태</span>
                      <h2>최근 실행</h2>
                    </div>
                    <Badge tone="amber">
                      최근 {viewModel.observability.recentRuns.length}건
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
                      "실행 항목",
                    )}
                  </div>
                </div>
              </div>

              <div className="admin-console-subsection">
                <div className="section-header">
                  <div>
                    <span className="preview-kicker">주의 필요</span>
                    <h2>최근 실패</h2>
                  </div>
                  <Badge tone="violet">
                    실패 {viewModel.observability.recentFailures.length}건
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
                    "실패 항목",
                  )}
                </div>
              </div>
            </article>

            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">국내 미상장 모니터</span>
                  <h2>{viewModel.domesticPrelistingRoute}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone="amber">
                    {viewModel.domesticPrelisting.length}개 후보
                  </Badge>
                </div>
              </div>
              <div className="preview-status">
                <span className="preview-kicker">후보 기준</span>
                <p>
                  업비트와 빗썸 상장 레지스트리에는 없지만 최근 7일 내 온체인
                  이동이 잡힌 토큰을 추적 지갑 관여도와 활동량 기준으로
                  정렬합니다.
                </p>
              </div>
              <div className="alert-inbox-list">
                {domesticPrelistingPreview.map((item) => (
                  <article
                    key={`${item.chain}:${item.tokenAddress}`}
                    className="alert-inbox-item"
                  >
                    <div className="alert-inbox-topline">
                      <strong>
                        {item.tokenSymbol} · {item.chain}
                      </strong>
                      <Badge tone="amber">{item.listingLabel}</Badge>
                    </div>
                    <p>{item.activityLabel}</p>
                    <p>{item.amountLabel}</p>
                    <div className="cluster-member-meta">
                      <span>{item.recencyLabel}</span>
                      <span>{item.tokenAddress}</span>
                    </div>
                  </article>
                ))}
                {viewModel.domesticPrelisting.length === 0 ? (
                  <p className="admin-console-preview-note">
                    아직 조건에 맞는 국내 미상장 후보가 없습니다.
                  </p>
                ) : null}
                {renderPreviewOverflowNote(
                  viewModel.domesticPrelisting.length,
                  domesticPrelistingPreview.length,
                  "후보",
                )}
              </div>
            </article>

            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">백테스트 작업</span>
                  <h2>{currentPreview.backtestOps.route}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone="amber">{configuredCheckCount}개 설정됨</Badge>
                </div>
              </div>
              <div className="preview-status">
                <span className="preview-kicker">수동 검증</span>
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
                          ? "실행 중..."
                          : "점검 실행"}
                      </button>
                    </div>
                  </article>
                ))}
                {renderPreviewOverflowNote(
                  currentPreview.backtestOps.checks.length,
                  backtestChecksPreview.length,
                  "점검",
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
                  <span className="preview-kicker">억제 규칙</span>
                  <h2>{viewModel.suppressionsRoute}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone="amber">{activeSuppressionCount}개 활성</Badge>
                </div>
              </div>
              <div className="preview-status">
                <span className="preview-kicker">수동 오버라이드</span>
                <p>
                  데이터베이스를 직접 건드리지 않고도 특정 대상의 후속 알림을
                  억제하거나 잠시 음소거할 수 있습니다.
                </p>
              </div>
              <div className="cluster-action-list">
                <label className="detail-route-copy">
                  범위
                  <select
                    value={suppressionScope}
                    onChange={(event) =>
                      setSuppressionScope(event.target.value)
                    }
                  >
                    <option value="wallet">지갑</option>
                    <option value="cluster">클러스터</option>
                    <option value="entity">엔티티</option>
                    <option value="alert_rule">알림 규칙</option>
                  </select>
                </label>
                <label className="detail-route-copy">
                  대상
                  <input
                    value={suppressionTarget}
                    onChange={(event) =>
                      setSuppressionTarget(event.target.value)
                    }
                    placeholder="지갑 주소, 클러스터 ID, 엔티티 키..."
                    type="text"
                  />
                </label>
                <label className="detail-route-copy">
                  사유
                  <input
                    value={suppressionReason}
                    onChange={(event) =>
                      setSuppressionReason(event.target.value)
                    }
                    placeholder="왜 이 오버라이드가 필요한지 입력하세요"
                    type="text"
                  />
                </label>
                <label className="detail-route-copy">
                  만료 시각
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
                    ? "저장 중..."
                    : "억제 추가"}
                </button>
              </div>
              <div className="alert-inbox-list">
                {suppressionsPreview.map((item) => (
                  <article key={item.id} className="alert-inbox-item">
                    <div className="alert-inbox-topline">
                      <strong>{item.scope}</strong>
                      <Badge tone={item.active ? "amber" : "teal"}>
                        {item.active ? "활성" : "비활성"}
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
                          ? "삭제 중..."
                          : "삭제"}
                      </button>
                    </div>
                  </article>
                ))}
                {renderPreviewOverflowNote(
                  viewModel.suppressions.length,
                  suppressionsPreview.length,
                  "억제 규칙",
                )}
              </div>
            </article>

            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">프로바이더 한도</span>
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
                      사용 {item.usagePercent.toFixed(0)}% · 예약{" "}
                      {item.reservedPercent.toFixed(0)}%
                    </p>
                    <p>{item.headroomLabel}</p>
                    <p>{item.windowLabel}</p>
                    <p>마지막 확인 {item.lastCheckedAt}</p>
                  </article>
                ))}
                {renderPreviewOverflowNote(
                  viewModel.quotas.length,
                  quotasPreview.length,
                  "쿼터 항목",
                )}
              </div>
            </article>

            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">라벨</span>
                  <h2>{viewModel.labelsRoute}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone="teal">{viewModel.labels.length}개</Badge>
                </div>
              </div>
              <div className="preview-status">
                <span className="preview-kicker">데이터 상태</span>
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
                  "라벨",
                )}
              </div>
            </article>

            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">큐레이션 목록</span>
                  <h2>{viewModel.curatedListsRoute}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone="teal">{viewModel.curatedLists.length}개</Badge>
                </div>
              </div>
              <div className="alert-inbox-list">
                {curatedListsPreview.map((item) => (
                  <article key={item.id} className="alert-inbox-item">
                    <div className="alert-inbox-topline">
                      <strong>{item.name}</strong>
                      <Badge tone="teal">{item.tagLabel}</Badge>
                    </div>
                    <p>{item.notes || "아직 메모가 없습니다."}</p>
                    <div className="cluster-member-meta">
                      <Pill tone="amber">{item.itemCount}개 항목</Pill>
                      <span>{item.firstItemLabel}</span>
                      <span>{item.updatedAt}</span>
                    </div>
                  </article>
                ))}
                {renderPreviewOverflowNote(
                  viewModel.curatedLists.length,
                  curatedListsPreview.length,
                  "큐레이션 목록",
                )}
              </div>
            </article>

            <article className="preview-card detail-card">
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">감사 로그</span>
                  <h2>{viewModel.auditLogsRoute}</h2>
                </div>
                <div className="preview-state">
                  <Badge tone="violet">{viewModel.auditLogs.length}건</Badge>
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
                    <p>{item.note || "감사 메모가 없습니다."}</p>
                    <div className="cluster-member-meta">
                      <span>{item.actor}</span>
                      <span>{item.createdAt}</span>
                    </div>
                  </article>
                ))}
                {renderPreviewOverflowNote(
                  viewModel.auditLogs.length,
                  auditLogsPreview.length,
                  "감사 로그",
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
    return "최근 수집 활동 없음";
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
    return "아직 추적 지갑 워크로드가 없습니다";
  }
  const matureTracked =
    walletTracking.labeledCount + walletTracking.scoredCount;
  return `${matureTracked}/${totalTrackedSurface}개 지갑이 라벨링 또는 점수화되었습니다`;
}

function buildStaleTrackingLabel(
  walletTracking: AdminConsolePreview["observability"]["walletTracking"],
): string {
  if (walletTracking.staleCount <= 0) {
    return "새로고침 대기 중인 stale 추적 지갑이 없습니다";
  }
  return `${walletTracking.staleCount}개 추적 지갑이 stale 상태로 곧 새로고침 대상입니다`;
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
    return "등록된 추적 구독이 없습니다";
  }
  return `${subscriptions.activeCount}/${total}개 구독이 활성 상태입니다`;
}

function buildTrackingSubscriptionPendingLabel(
  subscriptions: AdminConsolePreview["observability"]["trackingSubscriptions"],
): string {
  if (subscriptions.erroredCount > 0) {
    return `${subscriptions.erroredCount}개 구독이 복구 필요 상태입니다`;
  }
  if (subscriptions.pendingCount > 0) {
    return `${subscriptions.pendingCount}개 구독이 아직 활성 대기 중입니다`;
  }
  if (subscriptions.pausedCount > 0) {
    return `${subscriptions.pausedCount}개 구독이 일시중지 상태입니다`;
  }
  return "대기 중인 구독 적체가 없습니다";
}

function buildQueueBacklogLabel(
  queueDepth: AdminConsolePreview["observability"]["queueDepth"],
): string {
  return `기본 큐에 ${queueDepth.defaultDepth}건이 대기 중입니다`;
}

function buildPriorityQueueLabel(
  queueDepth: AdminConsolePreview["observability"]["queueDepth"],
): string {
  if (queueDepth.priorityDepth > 0) {
    return `우선순위 큐에 ${queueDepth.priorityDepth}건이 대기 중입니다`;
  }
  return "우선순위 큐 적체가 없습니다";
}

function buildBackfillThroughputLabel(
  backfillHealth: AdminConsolePreview["observability"]["backfillHealth"],
): string {
  if (backfillHealth.jobs24h <= 0) {
    return "지난 24시간 동안 성공한 backfill drain 작업이 없습니다";
  }
  return `지난 24시간 동안 트랜잭션 ${backfillHealth.transactions24h}건, 액티비티 ${backfillHealth.activities24h}건을 처리했습니다`;
}

function buildBackfillExpansionLabel(
  backfillHealth: AdminConsolePreview["observability"]["backfillHealth"],
): string {
  if (backfillHealth.expansions24h <= 0) {
    return "지난 24시간 동안 backfill에서 파생된 확장 작업이 없습니다";
  }
  return `성공한 drain에서 ${backfillHealth.expansions24h}건의 확장 작업이 추가되었습니다`;
}

function buildStaleRefreshHitRateLabel(
  staleRefresh: AdminConsolePreview["observability"]["staleRefresh"],
): string {
  if (staleRefresh.attempts24h <= 0) {
    return "지난 24시간 동안 stale refresh 시도가 없었습니다";
  }
  const hitRate = Math.round(
    (staleRefresh.productive24h / staleRefresh.attempts24h) * 100,
  );
  return `${staleRefresh.attempts24h}건 중 ${staleRefresh.productive24h}건이 유효했습니다 (${hitRate}% 적중률)`;
}

function buildCompactStaleRefreshRate(
  staleRefresh: AdminConsolePreview["observability"]["staleRefresh"],
): string {
  if (staleRefresh.attempts24h <= 0) {
    return "기록 없음";
  }
  const hitRate = Math.round(
    (staleRefresh.productive24h / staleRefresh.attempts24h) * 100,
  );
  return `${hitRate}%`;
}

function buildDomesticListingLabel(
  item: AdminConsolePreview["domesticPrelisting"][number],
): string {
  if (!item.listedOnUpbit && !item.listedOnBithumb) {
    return "업비트/빗썸 미상장";
  }
  if (!item.listedOnUpbit) {
    return "업비트 미상장";
  }
  if (!item.listedOnBithumb) {
    return "빗썸 미상장";
  }
  return "상장 확인 필요";
}

function formatTokenAmount(value: string): string {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return value;
  }
  return new Intl.NumberFormat("ko-KR", {
    maximumFractionDigits: parsed >= 1000 ? 0 : 2,
  }).format(parsed);
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
      총 {totalCount}개 {label} 중 {previewCount}개만 미리 보여주고 있습니다.
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
    return "방금 전";
  }
  if (deltaMinutes < 60) {
    return `${deltaMinutes}분 전`;
  }
  const deltaHours = Math.round(deltaMinutes / 60);
  if (deltaHours < 24) {
    return `${deltaHours}시간 전`;
  }
  return `${Math.round(deltaHours / 24)}일 전`;
}
