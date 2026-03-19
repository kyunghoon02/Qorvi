import { walletSummaryRoute } from "./api-boundary.js";

export type Tone = "amber" | "teal" | "violet" | "emerald";

export type SprintMetric = {
  label: string;
  value: string;
  hint: string;
  tone: Tone;
};

export type SprintPanel = {
  id: string;
  eyebrow: string;
  title: string;
  summary: string;
  badgeLabel: string;
  bullets: string[];
  tags: string[];
  footer: string;
};

export const sprintMetrics: SprintMetric[] = [
  {
    label: "Scope",
    value: "Frozen",
    hint: "Must / Should / Later set",
    tone: "amber",
  },
  {
    label: "Contract",
    value: "Bounded",
    hint: "Envelope + evidence shape",
    tone: "teal",
  },
  {
    label: "Frontend",
    value: "Online",
    hint: "Next.js scaffold only",
    tone: "violet",
  },
  {
    label: "First slice",
    value: "Wallet summary",
    hint: "API-boundary first",
    tone: "emerald",
  },
];

export const sprintPanels: SprintPanel[] = [
  {
    id: "scope-freeze",
    eyebrow: "WG-001",
    title: "Beta scope is fixed for Sprint 0",
    summary:
      "The product shell is limited to the capabilities needed to unlock the first vertical slice.",
    badgeLabel: "must / should / later",
    bullets: [
      "Search, wallet intelligence, ops, and commercial boundaries are retained.",
      "Advanced collaboration and export features stay out of the first slice.",
      "The product view can evolve without blocking the backend language decision.",
    ],
    tags: ["scope", "beta", "priority"],
    footer: "Goal: keep the first release narrow enough to ship.",
  },
  {
    id: "foundation-stack",
    eyebrow: "WG-003",
    title: "Frontend stack is wired around Next.js",
    summary:
      "The web app is intentionally isolated from backend implementation details and only consumes typed API boundaries.",
    badgeLabel: "next.js + local ui",
    bullets: [
      "Workspace packages are limited to shared UI primitives and page-local scaffolding.",
      "Backend integration now uses a live API path with graceful fallback when the backend is unavailable.",
      "The shell still uses local state for search and panel filtering.",
    ],
    tags: ["nextjs", "ui", "boundary", "infra"],
    footer: "Goal: move the product surface now, not after the backend lands.",
  },
  {
    id: "contract-baseline",
    eyebrow: "WG-002",
    title: "The wallet summary contract is visible in the UI",
    summary:
      "The UI shows the route, identity block, and score rails that the backend will later satisfy.",
    badgeLabel: "wallet summary route",
    bullets: [
      `Route preview: \`${walletSummaryRoute.replace("GET ", "")}\`.`,
      "The contract shape includes source, label, and score metadata.",
      "The panel is designed to survive backend language changes.",
    ],
    tags: ["api", "contract", "summary"],
    footer: "Goal: make the boundary explicit before integration.",
  },
  {
    id: "first-slice",
    eyebrow: "Next",
    title: "The first slice remains wallet summary",
    summary:
      "When the backend arrives, this screen becomes the first consumer of real data without changing the layout.",
    badgeLabel: "ready for api",
    bullets: [
      "Search interaction already works against local panel data.",
      "The preview card mirrors the future API response boundary.",
      "The remaining work is deepening the live summary fields, not changing the layout.",
    ],
    tags: ["slice", "mock", "search"],
    footer: "Goal: keep the UI ready for integration, not blocked by it.",
  },
];

export const quickQueries = ["wallet", "contract", "infra", "slice"];

export function filterSprintPanels(query: string): SprintPanel[] {
  const normalized = query.trim().toLowerCase();

  if (normalized.length === 0) {
    return sprintPanels;
  }

  return sprintPanels.filter((panel) => {
    const haystack = [
      panel.eyebrow,
      panel.title,
      panel.summary,
      panel.badgeLabel,
      panel.footer,
      panel.id,
      ...panel.tags,
      ...panel.bullets,
    ]
      .join(" ")
      .toLowerCase();

    return haystack.includes(normalized);
  });
}
