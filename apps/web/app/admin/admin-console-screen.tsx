"use client";

import { useEffect, useState } from "react";

import { Badge, Pill, type Tone } from "@whalegraph/ui";

import {
  type AdminConsolePreview,
  createAdminSuppression,
  deleteAdminSuppression,
} from "../../lib/api-boundary";

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

  useEffect(() => {
    setCurrentPreview(preview);
    setMutationMessage("");
    setPendingKey("");
  }, [preview]);

  const viewModel = buildAdminConsoleViewModel({ preview: currentPreview });

  async function handleCreateSuppression(): Promise<void> {
    setPendingKey("suppression:create");
    setMutationMessage("");
    const result = await createAdminSuppression({
      scope: suppressionScope,
      target: suppressionTarget,
      reason: suppressionReason,
      expiresAt: suppressionExpiresAt,
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
    const result = await deleteAdminSuppression({
      suppressionId: suppressionID,
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

  return (
    <main className="page-shell detail-shell">
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
            <span>Labels</span>
            <strong>{viewModel.labels.length}</strong>
          </div>
          <div>
            <span>Active suppressions</span>
            <strong>
              {viewModel.suppressions.filter((item) => item.active).length}
            </strong>
          </div>
          <div>
            <span>Providers tracked</span>
            <strong>{viewModel.quotas.length}</strong>
          </div>
          <div>
            <span>Recent failures</span>
            <strong>{viewModel.observability.recentFailures.length}</strong>
          </div>
          <div>
            <span>Curated lists</span>
            <strong>{viewModel.curatedLists.length}</strong>
          </div>
          <div>
            <span>Audit entries</span>
            <strong>{viewModel.auditLogs.length}</strong>
          </div>
        </div>

        <div className="detail-actions">
          <a className="search-cta" href="/">
            Back to search
          </a>
          <span className="detail-route-copy">{viewModel.labelsRoute}</span>
        </div>
        {mutationMessage ? (
          <p className="detail-route-copy" aria-live="polite">
            {mutationMessage}
          </p>
        ) : null}
      </section>

      <section className="detail-grid alert-center-grid">
        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Label editor</span>
              <h2>{viewModel.labelsRoute}</h2>
            </div>
          </div>
          <div className="preview-status">
            <span className="preview-kicker">Data status</span>
            <p>{viewModel.statusMessage}</p>
          </div>
          <div className="alert-inbox-list">
            {viewModel.labels.map((item) => (
              <article key={item.id} className="alert-inbox-item">
                <div className="alert-inbox-topline">
                  <strong>{item.name}</strong>
                  <Badge tone="teal">{item.color}</Badge>
                </div>
                <p>{item.description}</p>
              </article>
            ))}
          </div>
        </article>

        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Suppressions</span>
              <h2>{viewModel.suppressionsRoute}</h2>
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
                onChange={(event) => setSuppressionScope(event.target.value)}
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
                onChange={(event) => setSuppressionTarget(event.target.value)}
                placeholder="wallet address, cluster id, entity key..."
                type="text"
              />
            </label>
            <label className="detail-route-copy">
              Reason
              <input
                value={suppressionReason}
                onChange={(event) => setSuppressionReason(event.target.value)}
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
            {viewModel.suppressions.map((item) => (
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
                    disabled={pendingKey === `suppression:${item.id}:delete`}
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
            {viewModel.quotas.map((item) => (
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
          </div>
        </article>

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
            <span className="preview-kicker">Ingest freshness</span>
            <p>{viewModel.observability.ingest.activityLabel}</p>
            <p>{viewModel.observability.ingest.freshnessLabel}</p>
          </div>
          <div className="alert-inbox-list">
            {viewModel.observability.providerUsage.map((item) => (
              <article key={item.provider} className="alert-inbox-item">
                <div className="alert-inbox-topline">
                  <strong>{item.provider}</strong>
                  <Badge tone={item.tone}>{item.status}</Badge>
                </div>
                <p>
                  {item.used24h} calls · {item.avgLatencyMs}ms avg latency
                </p>
                <p>{item.errorRateLabel}</p>
                <p>{item.lastSeenLabel}</p>
              </article>
            ))}
            <article className="alert-inbox-item">
              <div className="alert-inbox-topline">
                <strong>Alert delivery</strong>
                <Badge tone={viewModel.observability.alertDelivery.healthTone}>
                  {viewModel.observability.alertDelivery.retryableCount}{" "}
                  retryable
                </Badge>
              </div>
              <p>{viewModel.observability.alertDelivery.deliveryRateLabel}</p>
              <p>{viewModel.observability.alertDelivery.lastFailureLabel}</p>
            </article>
            {viewModel.observability.recentRuns.map((item) => (
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
            {viewModel.observability.recentFailures.map((item) => (
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
          </div>
        </article>

        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Curated lists</span>
              <h2>{viewModel.curatedListsRoute}</h2>
            </div>
            <div className="preview-state">
              <Badge tone="teal">{viewModel.curatedLists.length} lists</Badge>
            </div>
          </div>
          <div className="alert-inbox-list">
            {viewModel.curatedLists.map((item) => (
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
          </div>
        </article>

        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Audit logs</span>
              <h2>{viewModel.auditLogsRoute}</h2>
            </div>
            <div className="preview-state">
              <Badge tone="violet">{viewModel.auditLogs.length} entries</Badge>
            </div>
          </div>
          <div className="alert-inbox-list">
            {viewModel.auditLogs.map((item) => (
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
          </div>
        </article>
      </section>
    </main>
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
