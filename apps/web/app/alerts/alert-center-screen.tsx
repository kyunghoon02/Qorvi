"use client";

import { useEffect, useState } from "react";

import { Badge, Pill, type Tone } from "@whalegraph/ui";

import {
  type AlertCenterPreview,
  updateAlertInboxEvent,
  updateAlertRuleMutation,
} from "../../lib/api-boundary";

const toneBySeverity: Record<
  AlertCenterPreview["inbox"][number]["severity"],
  Tone
> = {
  critical: "amber",
  high: "amber",
  medium: "violet",
  low: "teal",
};

export type AlertCenterViewModel = {
  title: string;
  explanation: string;
  statusMessage: string;
  flash?:
    | {
        tone: Tone;
        title: string;
        message: string;
      }
    | undefined;
  activeSeverityFilter: AlertCenterPreview["activeSeverityFilter"];
  activeSignalFilter: AlertCenterPreview["activeSignalFilter"];
  activeStatusFilter: AlertCenterPreview["activeStatusFilter"];
  inboxRoute: string;
  rulesRoute: string;
  channelsRoute: string;
  backHref: string;
  severityFilters: Array<{
    key: AlertCenterPreview["activeSeverityFilter"];
    label: string;
    href: string;
    active: boolean;
  }>;
  signalFilters: Array<{
    key: AlertCenterPreview["activeSignalFilter"];
    label: string;
    href: string;
    active: boolean;
  }>;
  statusFilters: Array<{
    key: AlertCenterPreview["activeStatusFilter"];
    label: string;
    href: string;
    active: boolean;
  }>;
  nextPageHref?: string | undefined;
  unreadCount: number;
  inbox: Array<
    AlertCenterPreview["inbox"][number] & {
      severityTone: Tone;
      severityLabel: string;
      readLabel: string;
    }
  >;
  rules: AlertCenterPreview["rules"];
  channels: AlertCenterPreview["channels"];
};

export function buildAlertCenterViewModel({
  preview,
  flash,
}: {
  preview: AlertCenterPreview;
  flash?:
    | {
        tone: Tone;
        title: string;
        message: string;
      }
    | undefined;
}): AlertCenterViewModel {
  const activeSeverityFilter = preview.activeSeverityFilter;
  const activeSignalFilter = preview.activeSignalFilter;
  const activeStatusFilter = preview.activeStatusFilter;
  const severityFilterKeys: AlertCenterPreview["activeSeverityFilter"][] = [
    "all",
    "low",
    "medium",
    "high",
    "critical",
  ];
  const signalFilterOptions: Array<{
    key: AlertCenterPreview["activeSignalFilter"];
    label: string;
  }> = [
    { key: "all", label: "All signals" },
    { key: "cluster_score", label: "Cluster score" },
    { key: "shadow_exit", label: "Shadow exit" },
    { key: "first_connection", label: "First connection" },
  ];
  const statusFilterOptions: Array<{
    key: AlertCenterPreview["activeStatusFilter"];
    label: string;
  }> = [
    { key: "all", label: "All events" },
    { key: "unread", label: "Unread only" },
  ];

  return {
    title: "Alert center",
    explanation:
      "Triggered events, active watchlist rules, and delivery channels are kept in one place so operators can review what fired before muting or escalating.",
    statusMessage: preview.statusMessage,
    flash,
    activeSeverityFilter,
    activeSignalFilter,
    activeStatusFilter,
    inboxRoute: preview.inboxRoute,
    rulesRoute: preview.rulesRoute,
    channelsRoute: preview.channelsRoute,
    backHref: "/",
    severityFilters: severityFilterKeys.map((key) => ({
      key,
      label: key === "all" ? "All severities" : capitalizeWord(key),
      href: buildAlertCenterHref({
        severity: key,
        signalType: activeSignalFilter,
        status: activeStatusFilter,
      }),
      active: activeSeverityFilter === key,
    })),
    signalFilters: signalFilterOptions.map(({ key, label }) => ({
      key,
      label,
      href: buildAlertCenterHref({
        severity: activeSeverityFilter,
        signalType: key,
        status: activeStatusFilter,
      }),
      active: activeSignalFilter === key,
    })),
    statusFilters: statusFilterOptions.map(({ key, label }) => ({
      key,
      label,
      href: buildAlertCenterHref({
        severity: activeSeverityFilter,
        signalType: activeSignalFilter,
        status: key,
      }),
      active: activeStatusFilter === key,
    })),
    nextPageHref: preview.nextCursor
      ? buildAlertCenterHref({
          severity: activeSeverityFilter,
          signalType: activeSignalFilter,
          status: activeStatusFilter,
          cursor: preview.nextCursor,
        })
      : undefined,
    unreadCount: preview.unreadCount,
    inbox: preview.inbox.map((item) => ({
      ...item,
      severityTone: toneBySeverity[item.severity],
      severityLabel:
        item.severity === "critical"
          ? "Critical"
          : item.severity === "high"
            ? "High"
            : item.severity === "medium"
              ? "Medium"
              : "Low",
      readLabel: item.isRead ? "Read" : "Unread",
    })),
    rules: preview.rules,
    channels: preview.channels,
  };
}

export type TrackedWalletAlertQueryState = {
  status: "idle" | "success";
  wallet?: string;
  watchlistId?: string;
  ruleId?: string;
};

export function normalizeTrackedWalletAlertQueryState({
  tracked,
  wallet,
  watchlistId,
  ruleId,
}: {
  tracked: string | string[] | undefined;
  wallet: string | string[] | undefined;
  watchlistId: string | string[] | undefined;
  ruleId: string | string[] | undefined;
}): TrackedWalletAlertQueryState {
  const trackedValue = Array.isArray(tracked) ? tracked[0] : tracked;
  const walletValue = Array.isArray(wallet) ? wallet[0] : wallet;
  const watchlistValue = Array.isArray(watchlistId)
    ? watchlistId[0]
    : watchlistId;
  const ruleValue = Array.isArray(ruleId) ? ruleId[0] : ruleId;

  return {
    status: trackedValue === "success" ? "success" : "idle",
    ...(walletValue?.trim() ? { wallet: walletValue.trim() } : {}),
    ...(watchlistValue?.trim() ? { watchlistId: watchlistValue.trim() } : {}),
    ...(ruleValue?.trim() ? { ruleId: ruleValue.trim() } : {}),
  };
}

export function buildTrackedWalletAlertFlash(
  state: TrackedWalletAlertQueryState,
):
  | {
      tone: Tone;
      title: string;
      message: string;
    }
  | undefined {
  if (state.status !== "success") {
    return undefined;
  }

  const walletLabel = state.wallet ? `${state.wallet} is now tracked.` : "";
  const watchlistLabel = state.watchlistId
    ? `Watchlist ${state.watchlistId}`
    : "Tracked wallets";
  const ruleLabel = state.ruleId
    ? `rule ${state.ruleId}`
    : "the default signal rule";

  return {
    tone: "teal",
    title: "Wallet tracking active",
    message:
      `${walletLabel} ${watchlistLabel} and ${ruleLabel} are ready in this alert center.`
        .trim()
        .replace(/\s+/g, " "),
  };
}

export function buildAlertCenterHref({
  severity = "all",
  signalType = "all",
  status = "all",
  cursor,
}: {
  severity?: AlertCenterPreview["activeSeverityFilter"];
  signalType?: AlertCenterPreview["activeSignalFilter"];
  status?: AlertCenterPreview["activeStatusFilter"];
  cursor?: string;
}): string {
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
  if (cursor) {
    params.set("cursor", cursor);
  }
  const query = params.toString();
  return query ? `/alerts?${query}` : "/alerts";
}

function replaceAlertInboxItem(
  preview: AlertCenterPreview,
  nextItem: AlertCenterPreview["inbox"][number],
): AlertCenterPreview {
  const previousItem = preview.inbox.find((item) => item.id === nextItem.id);
  const nextUnreadCount =
    preview.unreadCount +
    (previousItem?.isRead === true && nextItem.isRead === false
      ? 1
      : previousItem?.isRead === false && nextItem.isRead === true
        ? -1
        : 0);

  return {
    ...preview,
    unreadCount: Math.max(nextUnreadCount, 0),
    inbox: preview.inbox.map((item) =>
      item.id === nextItem.id ? nextItem : item,
    ),
  };
}

function replaceAlertRule(
  preview: AlertCenterPreview,
  nextRule: AlertCenterPreview["rules"][number],
): AlertCenterPreview {
  return {
    ...preview,
    rules: preview.rules.map((rule) =>
      rule.id === nextRule.id ? nextRule : rule,
    ),
  };
}

function isRuleSnoozed(rule: AlertCenterPreview["rules"][number]): boolean {
  if (!rule.snoozeUntil) {
    return false;
  }
  const parsed = Date.parse(rule.snoozeUntil);
  return Number.isFinite(parsed) && parsed > Date.now();
}

function capitalizeWord(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1);
}

export function AlertCenterScreen({
  preview,
  flash,
}: {
  preview: AlertCenterPreview;
  flash?:
    | {
        tone: Tone;
        title: string;
        message: string;
      }
    | undefined;
}) {
  const [currentPreview, setCurrentPreview] = useState(preview);
  const [mutationMessage, setMutationMessage] = useState("");
  const [pendingActionKey, setPendingActionKey] = useState("");

  useEffect(() => {
    setCurrentPreview(preview);
    setMutationMessage("");
    setPendingActionKey("");
  }, [preview]);

  const viewModel = buildAlertCenterViewModel({
    preview: currentPreview,
    flash,
  });

  async function handleInboxToggle(
    item: AlertCenterPreview["inbox"][number],
  ): Promise<void> {
    setPendingActionKey(`event:${item.id}`);
    setMutationMessage("");
    const result = await updateAlertInboxEvent({
      eventId: item.id,
      isRead: !item.isRead,
    });
    setPendingActionKey("");
    setMutationMessage(result.message);
    if (result.ok && result.event) {
      const nextEvent = result.event;
      setCurrentPreview((existing) =>
        replaceAlertInboxItem(existing, nextEvent),
      );
    }
  }

  async function handleRuleMutation(
    rule: AlertCenterPreview["rules"][number],
    action: "toggle-enabled" | "toggle-snooze",
  ): Promise<void> {
    setPendingActionKey(`rule:${rule.id}:${action}`);
    setMutationMessage("");
    const result = await updateAlertRuleMutation({
      ruleId: rule.id,
      action,
      currentRule: rule,
    });
    setPendingActionKey("");
    setMutationMessage(result.message);
    if (result.ok && result.rule) {
      const nextRule = result.rule;
      setCurrentPreview((existing) => replaceAlertRule(existing, nextRule));
    }
  }

  return (
    <main className="page-shell detail-shell">
      <section className="detail-hero alert-center-hero">
        <div className="eyebrow-row">
          <Pill tone="amber">Alerts</Pill>
          <Pill tone="violet">review center</Pill>
        </div>

        <div className="detail-hero-copy">
          <h1>{viewModel.title}</h1>
          <p>{viewModel.explanation}</p>
        </div>

        <div className="detail-identity">
          <div>
            <span>Triggered events</span>
            <strong>{viewModel.inbox.length}</strong>
          </div>
          <div>
            <span>Unread</span>
            <strong>{viewModel.unreadCount}</strong>
          </div>
          <div>
            <span>Active rules</span>
            <strong>
              {viewModel.rules.filter((item) => item.isEnabled).length}
            </strong>
          </div>
          <div>
            <span>Delivery channels</span>
            <strong>{viewModel.channels.length}</strong>
          </div>
        </div>

        <div className="detail-actions">
          <a className="search-cta" href={viewModel.backHref}>
            Back to search
          </a>
          <span className="detail-route-copy">{viewModel.inboxRoute}</span>
        </div>
        {viewModel.flash ? (
          <div className="preview-status detail-status-inline">
            <span className="preview-kicker">{viewModel.flash.title}</span>
            <div className="cluster-member-meta">
              <Badge tone={viewModel.flash.tone}>ready</Badge>
              <span>{viewModel.flash.message}</span>
            </div>
          </div>
        ) : null}
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
              <span className="preview-kicker">Triggered events</span>
              <h2>{viewModel.inboxRoute}</h2>
            </div>
            <div className="preview-state">
              <Badge tone={viewModel.inbox.length > 0 ? "amber" : "teal"}>
                {viewModel.inbox.length} visible
              </Badge>
            </div>
          </div>

          <div className="preview-status">
            <span className="preview-kicker">Data status</span>
            <p>{viewModel.statusMessage}</p>
          </div>

          <div className="alert-filter-group">
            <div className="alert-filter-row">
              {viewModel.severityFilters.map((filter) => (
                <a
                  key={filter.key}
                  className="search-cta"
                  aria-current={filter.active ? "page" : undefined}
                  href={filter.href}
                >
                  {filter.label}
                </a>
              ))}
            </div>
            <div className="alert-filter-row">
              {viewModel.signalFilters.map((filter) => (
                <a
                  key={filter.key}
                  className="search-cta"
                  aria-current={filter.active ? "page" : undefined}
                  href={filter.href}
                >
                  {filter.label}
                </a>
              ))}
            </div>
            <div className="alert-filter-row">
              {viewModel.statusFilters.map((filter) => (
                <a
                  key={filter.key}
                  className="search-cta"
                  aria-current={filter.active ? "page" : undefined}
                  href={filter.href}
                >
                  {filter.label}
                </a>
              ))}
            </div>
          </div>

          <div className="alert-inbox-list">
            {viewModel.inbox.map((item) => (
              <article
                key={item.id}
                className="cluster-member-card shadow-feed-card"
              >
                <div className="cluster-member-card-head">
                  <div>
                    <strong>{item.title}</strong>
                    <span>{item.alertRuleId}</span>
                  </div>
                  <Badge tone={item.severityTone}>{item.severityLabel}</Badge>
                </div>
                <p className="shadow-feed-card-copy">{item.explanation}</p>
                <div className="cluster-member-meta">
                  <span>{item.signalType}</span>
                  <span>{item.observedAt}</span>
                  <Pill tone={item.isRead ? "violet" : "teal"}>
                    {item.readLabel}
                  </Pill>
                  {typeof item.scoreValue === "number" ? (
                    <Pill tone={item.severityTone}>
                      score {item.scoreValue}
                    </Pill>
                  ) : null}
                </div>
                <div className="detail-actions">
                  <button
                    className="search-cta"
                    disabled={pendingActionKey === `event:${item.id}`}
                    onClick={() => void handleInboxToggle(item)}
                    type="button"
                  >
                    {pendingActionKey === `event:${item.id}`
                      ? "Saving..."
                      : item.isRead
                        ? "Mark unread"
                        : "Mark read"}
                  </button>
                </div>
              </article>
            ))}
          </div>
          {viewModel.nextPageHref ? (
            <div className="detail-actions">
              <a className="search-cta" href={viewModel.nextPageHref}>
                Next page
              </a>
            </div>
          ) : null}
        </article>

        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Active rules</span>
              <h2>{viewModel.rulesRoute}</h2>
            </div>
            <div className="preview-state">
              <Badge tone="violet">{viewModel.rules.length} rules</Badge>
            </div>
          </div>

          <div className="alert-rule-list">
            {viewModel.rules.map((rule) => (
              <article
                key={rule.id}
                className="cluster-action-card alert-rule-card"
              >
                <div>
                  <strong>{rule.name}</strong>
                  <p>
                    {rule.ruleType} · min severity {rule.minimumSeverity} ·
                    cooldown {rule.cooldownSeconds}s
                  </p>
                </div>
                <div className="cluster-member-meta">
                  <Pill tone={rule.isEnabled ? "teal" : "violet"}>
                    {rule.isEnabled ? "enabled" : "muted"}
                  </Pill>
                  <span>{rule.eventCount} events</span>
                  {rule.lastTriggeredAt ? (
                    <span>{rule.lastTriggeredAt}</span>
                  ) : null}
                </div>
                <div className="search-meta">
                  {rule.signalTypes.map((signalType) => (
                    <Pill key={`${rule.id}-${signalType}`} tone="violet">
                      {signalType}
                    </Pill>
                  ))}
                  {rule.renotifyOnSeverityIncrease ? (
                    <Pill tone="amber">re-notify on escalation</Pill>
                  ) : null}
                  {rule.snoozeUntil ? (
                    <Pill tone="violet">snoozed until {rule.snoozeUntil}</Pill>
                  ) : null}
                </div>
                <div className="detail-actions">
                  <button
                    className="search-cta"
                    disabled={
                      pendingActionKey === `rule:${rule.id}:toggle-enabled`
                    }
                    onClick={() =>
                      void handleRuleMutation(rule, "toggle-enabled")
                    }
                    type="button"
                  >
                    {pendingActionKey === `rule:${rule.id}:toggle-enabled`
                      ? "Saving..."
                      : rule.isEnabled
                        ? "Mute rule"
                        : "Resume rule"}
                  </button>
                  <button
                    className="search-cta"
                    disabled={
                      pendingActionKey === `rule:${rule.id}:toggle-snooze`
                    }
                    onClick={() =>
                      void handleRuleMutation(rule, "toggle-snooze")
                    }
                    type="button"
                  >
                    {pendingActionKey === `rule:${rule.id}:toggle-snooze`
                      ? "Saving..."
                      : isRuleSnoozed(rule)
                        ? "Clear snooze"
                        : "Snooze 24h"}
                  </button>
                </div>
              </article>
            ))}
          </div>
        </article>

        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Delivery channels</span>
              <h2>{viewModel.channelsRoute}</h2>
            </div>
            <div className="preview-state">
              <Badge tone="teal">
                {viewModel.channels.filter((item) => item.isEnabled).length}{" "}
                enabled
              </Badge>
            </div>
          </div>

          <div className="cluster-action-list">
            {viewModel.channels.map((channel) => (
              <article
                key={channel.id}
                className="cluster-action-card alert-channel-card"
              >
                <div>
                  <strong>{channel.label}</strong>
                  <p>
                    {channel.channelType} · {channel.target}
                  </p>
                </div>
                <div className="cluster-member-meta">
                  <Pill tone={channel.isEnabled ? "teal" : "violet"}>
                    {channel.isEnabled ? "enabled" : "disabled"}
                  </Pill>
                  <span>{channel.updatedAt}</span>
                </div>
              </article>
            ))}
          </div>
        </article>
      </section>
    </main>
  );
}
