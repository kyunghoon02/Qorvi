import assert from "node:assert/strict";
import test from "node:test";

import { buildAdminConsoleViewModel } from "../app/admin/admin-console-screen";
import { getAdminConsolePreview } from "../lib/api-boundary";

test("buildAdminConsoleViewModel carries labels suppressions and quota state", () => {
  const viewModel = buildAdminConsoleViewModel({
    preview: getAdminConsolePreview(),
  });

  assert.equal(viewModel.title, "Admin console");
  assert.equal(viewModel.labels.length, 0);
  assert.equal(viewModel.suppressions.length, 0);
  assert.equal(viewModel.quotas.length, 0);
  assert.equal(viewModel.observability.providerUsage.length, 0);
  assert.equal(viewModel.observability.ingest.lagStatus, "unavailable");
  assert.equal(viewModel.curatedLists.length, 0);
  assert.equal(viewModel.auditLogs.length, 0);
});

test("buildAdminConsoleViewModel derives quota pressure labels", () => {
  const viewModel = buildAdminConsoleViewModel({
    preview: {
      ...getAdminConsolePreview(),
      quotas: [
        {
          provider: "alchemy",
          status: "warning",
          limit: 5000,
          used: 3200,
          reserved: 400,
          windowLabel: "2026-03-22T00:00:00Z -> 2026-03-23T00:00:00Z",
          lastCheckedAt: "2026-03-22T12:00:00Z",
        },
      ],
    },
  });

  assert.equal(viewModel.quotas[0]?.usagePercent, 64);
  assert.equal(viewModel.quotas[0]?.reservedPercent, 8);
  assert.equal(viewModel.quotas[0]?.headroomLabel, "1800 remaining");
});

test("buildAdminConsoleViewModel derives observability labels", () => {
  const viewModel = buildAdminConsoleViewModel({
    preview: {
      ...getAdminConsolePreview(),
      observability: {
        providerUsage: [
          {
            provider: "alchemy",
            status: "warning",
            used24h: 3200,
            error24h: 32,
            avgLatencyMs: 210,
            lastSeenAt: "2026-03-22T12:00:00Z",
          },
        ],
        ingest: {
          lagStatus: "healthy",
          freshnessSeconds: 120,
          lastBackfillAt: "2026-03-22T11:59:00Z",
          lastWebhookAt: "2026-03-22T11:58:30Z",
        },
        alertDelivery: {
          attempts24h: 12,
          delivered24h: 11,
          failed24h: 1,
          retryableCount: 1,
          lastFailureAt: "2026-03-22T11:50:00Z",
        },
        walletTracking: {
          candidateCount: 14,
          trackedCount: 10,
          labeledCount: 6,
          scoredCount: 4,
          staleCount: 2,
          suppressedCount: 1,
        },
        trackingSubscriptions: {
          pendingCount: 3,
          activeCount: 7,
          erroredCount: 1,
          pausedCount: 0,
          lastEventAt: "2026-03-22T11:59:30Z",
        },
        queueDepth: {
          defaultDepth: 12,
          priorityDepth: 2,
        },
        backfillHealth: {
          jobs24h: 18,
          activities24h: 2200,
          transactions24h: 980,
          expansions24h: 14,
          lastSuccessAt: "2026-03-22T11:58:00Z",
        },
        staleRefresh: {
          attempts24h: 5,
          succeeded24h: 5,
          productive24h: 3,
          lastHitAt: "2026-03-22T11:40:00Z",
        },
        recentRuns: [
          {
            jobName: "wallet-backfill-drain-batch",
            lastStatus: "succeeded",
            lastStartedAt: "2026-03-22T11:58:00Z",
            lastSuccessAt: "2026-03-22T11:59:00Z",
            minutesSinceSuccess: 1,
          },
        ],
        recentFailures: [
          {
            source: "provider",
            kind: "alchemy",
            occurredAt: "2026-03-22T11:50:00Z",
            summary: "transfers.backfill returned 500",
            details: {},
          },
        ],
      },
    },
  });

  assert.equal(
    viewModel.observability.providerUsage[0]?.errorRateLabel,
    "1% error rate",
  );
  assert.equal(viewModel.observability.ingest.freshnessLabel, "120s freshness");
  assert.equal(
    viewModel.observability.alertDelivery.deliveryRateLabel,
    "92% delivered",
  );
  assert.equal(
    viewModel.observability.walletTracking.trackedCoverageLabel,
    "10/34 wallets are labeled or scored",
  );
  assert.equal(
    viewModel.observability.trackingSubscriptions.activeRatioLabel,
    "7/11 subscriptions are active",
  );
  assert.equal(viewModel.observability.queueDepth.backlogLabel, "12 jobs in default queue");
  assert.equal(
    viewModel.observability.backfillHealth.throughputLabel,
    "980 transactions and 2200 activities processed in the last 24 hours",
  );
  assert.equal(
    viewModel.observability.staleRefresh.hitRateLabel,
    "3/5 stale refresh attempts were productive (60% hit rate)",
  );
  assert.equal(
    viewModel.observability.recentRuns[0]?.successLabel,
    "1m since success",
  );
  assert.equal(
    viewModel.observability.recentFailures[0]?.title,
    "provider · alchemy",
  );
});
