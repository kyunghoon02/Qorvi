import assert from "node:assert/strict";
import test from "node:test";

import { buildAdminConsoleViewModel } from "../app/admin/admin-console-screen";
import { getAdminConsolePreview } from "../lib/api-boundary";

test("buildAdminConsoleViewModel carries labels suppressions and quota state", () => {
  const viewModel = buildAdminConsoleViewModel({
    preview: getAdminConsolePreview(),
  });

  assert.equal(viewModel.title, "운영 대시보드");
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
  assert.equal(viewModel.quotas[0]?.headroomLabel, "1800 남음");
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
    "에러율 1%",
  );
  assert.equal(viewModel.observability.ingest.freshnessLabel, "신선도 120초");
  assert.equal(
    viewModel.observability.alertDelivery.deliveryRateLabel,
    "92% 전달 성공",
  );
  assert.equal(
    viewModel.observability.walletTracking.trackedCoverageLabel,
    "10/34개 지갑이 라벨링 또는 점수화되었습니다",
  );
  assert.equal(
    viewModel.observability.trackingSubscriptions.activeRatioLabel,
    "7/11개 구독이 활성 상태입니다",
  );
  assert.equal(
    viewModel.observability.queueDepth.backlogLabel,
    "기본 큐에 12건이 대기 중입니다",
  );
  assert.equal(
    viewModel.observability.backfillHealth.throughputLabel,
    "지난 24시간 동안 트랜잭션 980건, 액티비티 2200건을 처리했습니다",
  );
  assert.equal(
    viewModel.observability.staleRefresh.hitRateLabel,
    "5건 중 3건이 유효했습니다 (60% 적중률)",
  );
  assert.equal(
    viewModel.observability.recentRuns[0]?.successLabel,
    "1분 전 성공",
  );
  assert.equal(
    viewModel.observability.recentFailures[0]?.title,
    "provider · alchemy",
  );
});
