import assert from "node:assert/strict";
import test from "node:test";

import {
  getShadowExitFeedPreview,
} from "../lib/api-boundary";
import {
  buildShadowExitFeedViewModel,
} from "../app/signals/shadow-exits/shadow-exit-feed-screen";

test("buildShadowExitFeedViewModel carries review copy and links", () => {
  const viewModel = buildShadowExitFeedViewModel({
    feed: getShadowExitFeedPreview(),
  });

  assert.equal(viewModel.title, "Shadow exit review feed");
  assert.match(viewModel.explanation, /review candidates/i);
  assert.equal(viewModel.feedRoute, "GET /v1/signals/shadow-exits");
  assert.equal(viewModel.windowLabel, "Last 24 hours");
  assert.equal(viewModel.itemCount, 0);
  assert.equal(viewModel.highPriorityCount, 0);
  assert.equal(viewModel.backHref, "/");
  assert.equal(viewModel.items.length, 0);
});
