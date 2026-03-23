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
    viewModel.observability.recentRuns[0]?.successLabel,
    "1m since success",
  );
  assert.equal(
    viewModel.observability.recentFailures[0]?.title,
    "provider · alchemy",
  );
});
