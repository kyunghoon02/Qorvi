import test from "node:test";
import assert from "node:assert/strict";

import {
  shouldHydrateHomeSearchQuery,
  shouldPollHomeWalletPreview,
} from "../app/home-screen";
import { getWalletSummaryPreview } from "../lib/api-boundary";

test("shouldHydrateHomeSearchQuery only hydrates when URL query changes", () => {
  assert.equal(shouldHydrateHomeSearchQuery("0xabc", null), true);
  assert.equal(shouldHydrateHomeSearchQuery("0xabc", ""), true);
  assert.equal(shouldHydrateHomeSearchQuery("0xabc", "0xabc"), false);
  assert.equal(shouldHydrateHomeSearchQuery("", ""), false);
});

test("shouldPollHomeWalletPreview follows indexing status", () => {
  const fallback = getWalletSummaryPreview();
  const ready = {
    ...fallback,
    indexing: {
      ...fallback.indexing,
      status: "ready" as const,
      coverageWindowDays: 30,
    },
  };

  assert.equal(shouldPollHomeWalletPreview(fallback), true);
  assert.equal(shouldPollHomeWalletPreview(ready), false);
});
