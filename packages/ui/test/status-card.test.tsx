import assert from "node:assert/strict";
import test from "node:test";

import { renderToStaticMarkup } from "react-dom/server";

import { MetricCard, StatusCard } from "../src/index.js";

test("StatusCard renders key copy", () => {
  const html = renderToStaticMarkup(
    <StatusCard
      eyebrow="WG-002"
      title="Wallet summary contract"
      summary="The boundary is visible from the product shell."
      tone="teal"
      badgeLabel="route bound"
      bullets={["Envelope is fixed", "Evidence is mandatory"]}
      tags={["api", "contract"]}
      footer="Ready for integration"
    />,
  );

  assert.ok(html.includes("Wallet summary contract"));
  assert.ok(html.includes("Evidence is mandatory"));
});

test("MetricCard renders the metric label and hint", () => {
  const html = renderToStaticMarkup(
    <MetricCard
      label="Scope"
      value="Frozen"
      hint="Must / Should / Later set"
      tone="amber"
    />,
  );

  assert.ok(html.includes("Scope"));
  assert.ok(html.includes("Frozen"));
  assert.ok(html.includes("Must / Should / Later set"));
});
