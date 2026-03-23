import assert from "node:assert/strict";
import test from "node:test";

import {
  buildCounterpartyEntityAssignment,
  buildWalletGraphAvailabilityPresentation,
  buildWalletSummaryAvailabilityPresentation,
  formatGraphSnapshotSource,
  isSummaryDerivedGraph,
} from "../app/wallets/[chain]/[address]/wallet-graph-presenter";

test("buildWalletGraphAvailabilityPresentation distinguishes live, summary-derived, and unavailable graph states", () => {
  assert.deepEqual(
    buildWalletGraphAvailabilityPresentation({
      mode: "live",
      source: "live-api",
      snapshot: {
        key: "k",
        source: "postgres-wallet-graph-snapshot",
        generatedAt: "2026-03-23T00:00:00Z",
        maxAgeSeconds: 300,
      },
    }),
    {
      stateLabel: "Live relationship map",
      modeLabel: "live data",
      sourceLabel: "live graph",
      statusCopy: "Live neighborhood loaded from the graph store.",
      snapshotSourceLabel: "Graph snapshot",
    },
  );

  assert.deepEqual(
    buildWalletGraphAvailabilityPresentation({
      mode: "unavailable",
      source: "summary-derived",
    }),
    {
      stateLabel: "Map from current summary",
      modeLabel: "derived context",
      sourceLabel: "summary-derived",
      statusCopy:
        "Relationship map derived from wallet summary counterparties while the canonical neighborhood warms up.",
      snapshotSourceLabel: "No snapshot",
    },
  );

  assert.deepEqual(
    buildWalletGraphAvailabilityPresentation({
      mode: "unavailable",
      source: "boundary-unavailable",
    }),
    {
      stateLabel: "Relationship map unavailable",
      modeLabel: "waiting for live data",
      sourceLabel: "boundary unavailable",
      statusCopy:
        "Relationship data is still loading or temporarily unavailable.",
      snapshotSourceLabel: "No snapshot",
    },
  );
});

test("buildWalletSummaryAvailabilityPresentation distinguishes live and unavailable summary states", () => {
  assert.deepEqual(
    buildWalletSummaryAvailabilityPresentation({
      mode: "live",
      source: "live-api",
    }),
    {
      modeLabel: "live data",
      sourceLabel: "live summary",
    },
  );

  assert.deepEqual(
    buildWalletSummaryAvailabilityPresentation({
      mode: "unavailable",
      source: "boundary-unavailable",
    }),
    {
      modeLabel: "waiting for live data",
      sourceLabel: "boundary unavailable",
    },
  );
});

test("buildCounterpartyEntityAssignment maps heuristic and curated sources into stable entity badges", () => {
  const heuristic = buildCounterpartyEntityAssignment({
    entityKey: "heuristic:evm:opensea",
    entityLabel: "OpenSea",
  });
  assert.deepEqual(heuristic, {
    entityNodeId: "entity:heuristic:evm:opensea",
    entityLabel: "OpenSea",
    entityHref: "/?q=OpenSea",
    source: "provider-heuristic-identity",
    sourceLabel: "Heuristic",
    sourceTone: "amber",
  });

  const curated = buildCounterpartyEntityAssignment({
    entityKey: "curated:seaport",
    entityLabel: "Seaport",
  });
  assert.deepEqual(curated, {
    entityNodeId: "entity:curated:seaport",
    entityLabel: "Seaport",
    entityHref: "/?q=Seaport",
    source: "curated-identity-index",
    sourceLabel: "Curated",
    sourceTone: "emerald",
  });
});

test("summary-derived graph detection and snapshot source formatting stay stable", () => {
  assert.equal(isSummaryDerivedGraph({ source: "summary-derived" }), true);
  assert.equal(isSummaryDerivedGraph({ source: "live-api" }), false);
  assert.equal(
    formatGraphSnapshotSource("postgres-wallet-graph-snapshot"),
    "Graph snapshot",
  );
  assert.equal(formatGraphSnapshotSource(undefined), "No snapshot");
});
