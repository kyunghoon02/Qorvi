import assert from "node:assert/strict";
import test from "node:test";

import { filterSprintPanels, sprintPanels } from "../lib/sprint0.js";

test("filterSprintPanels returns all panels when query is empty", () => {
  const panels = filterSprintPanels("");

  assert.equal(panels.length, sprintPanels.length);
});

test("filterSprintPanels narrows to the wallet contract panel", () => {
  const panels = filterSprintPanels("wallet");

  assert.equal(panels.length, 3);
  assert.ok(panels.some((panel) => panel.id === "scope-freeze"));
  assert.ok(panels.some((panel) => panel.id === "contract-baseline"));
  assert.ok(panels.some((panel) => panel.id === "first-slice"));
});
