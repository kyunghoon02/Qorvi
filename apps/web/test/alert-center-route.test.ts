import assert from "node:assert/strict";
import test from "node:test";

import {
  buildTrackedWalletAlertFlash,
  normalizeTrackedWalletAlertQueryState,
} from "../app/alerts/alert-center-flash";
import {
  buildAlertCenterHref,
  buildAlertCenterViewModel,
} from "../app/alerts/alert-center-screen";
import { getAlertCenterPreview } from "../lib/api-boundary";

test("buildAlertCenterHref omits default filters and preserves selected filters", () => {
  assert.equal(
    buildAlertCenterHref({ severity: "all", signalType: "all" }),
    "/alerts",
  );
  assert.equal(
    buildAlertCenterHref({
      severity: "critical",
      signalType: "cluster_score",
    }),
    "/alerts?severity=critical&signalType=cluster_score",
  );
  assert.equal(
    buildAlertCenterHref({
      severity: "high",
      signalType: "shadow_exit",
      status: "unread",
      cursor: "cursor_123",
    }),
    "/alerts?severity=high&signalType=shadow_exit&status=unread&cursor=cursor_123",
  );
});

test("buildAlertCenterViewModel carries inbox, rules, and channels", () => {
  const viewModel = buildAlertCenterViewModel({
    preview: getAlertCenterPreview({
      severity: "high",
      signalType: "shadow_exit",
    }),
  });

  assert.equal(viewModel.title, "Alert center");
  assert.equal(viewModel.activeSeverityFilter, "high");
  assert.equal(viewModel.activeSignalFilter, "shadow_exit");
  assert.equal(viewModel.activeStatusFilter, "all");
  assert.equal(viewModel.inboxRoute, "GET /v1/alerts");
  assert.equal(viewModel.rulesRoute, "GET /v1/alert-rules");
  assert.equal(viewModel.channelsRoute, "GET /v1/alert-delivery-channels");
  assert.equal(viewModel.inbox.length, 0);
  assert.equal(viewModel.rules.length, 0);
  assert.equal(viewModel.channels.length, 0);
  assert.equal(viewModel.unreadCount, 0);
  assert.ok(viewModel.severityFilters.find((item) => item.active)?.href);
  assert.ok(viewModel.signalFilters.find((item) => item.active)?.href);
  assert.ok(viewModel.statusFilters.find((item) => item.active)?.href);
});

test("tracked wallet query state normalizes success flash parameters", () => {
  const state = normalizeTrackedWalletAlertQueryState({
    tracked: ["success"],
    wallet: "0xabc123",
    watchlistId: ["watch_123"],
    ruleId: "rule_123",
  });

  assert.deepEqual(state, {
    status: "success",
    wallet: "0xabc123",
    watchlistId: "watch_123",
    ruleId: "rule_123",
  });
});

test("tracked wallet flash explains the new watchlist and rule", () => {
  const flash = buildTrackedWalletAlertFlash({
    status: "success",
    wallet: "0xabc123",
    watchlistId: "watch_123",
    ruleId: "rule_123",
  });

  assert.equal(flash?.tone, "teal");
  assert.match(flash?.message ?? "", /0xabc123 is now tracked/i);
  assert.match(flash?.message ?? "", /watch_123/i);
  assert.match(flash?.message ?? "", /rule_123/i);
});
