export type BillingPlanId = "free" | "pro" | "team";

export type BillingEntitlementAccessReason =
  | "granted"
  | "plan_disabled"
  | "role_required";

export type BillingEntitlementPreview = {
  id: string;
  label: string;
  description: string;
  requiredPlan: BillingPlanId;
  available: boolean;
  accessReason: BillingEntitlementAccessReason;
  value: string;
};

export type BillingPlanPreview = {
  id: BillingPlanId;
  name: string;
  priceLabel: string;
  summary: string;
  features: string[];
  current: boolean;
};

export type BillingCheckoutMutationResult = {
  ok: boolean;
  redirectUrl?: string;
  status: "redirect" | "unavailable";
  message: string;
};

export type BillingCheckoutSurface = "account" | "pricing";

export type BillingCheckoutQueryState = {
  status: "idle" | "success" | "cancel";
  plan: BillingPlanId | undefined;
};

export type BillingCheckoutFlash = {
  tone: "teal" | "amber";
  title: string;
  message: string;
};

export type BillingAccountPreview = {
  mode: "unavailable" | "live";
  source: "boundary-unavailable" | "live-api";
  route: string;
  statusMessage: string;
  currentPlan: BillingPlanId;
  currentPlanLabel: string;
  currentRoleLabel: string;
  billingCycleLabel: string;
  renewalLabel: string;
  checkoutIntent: BillingCheckoutIntentPreview;
  entitlements: BillingEntitlementPreview[];
  plans: BillingPlanPreview[];
};

export type BillingCheckoutIntentPreview = {
  state: "preview" | "ready" | "complete";
  title: string;
  description: string;
  recommendedPlan: BillingPlanId;
  recommendedPlanLabel: string;
  checkoutRoute: string;
  ctaLabel: string;
  ctaHref: string;
};

type LoadBillingAccountPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: BillingAccountPreview;
  requestHeaders?: HeadersInit | undefined;
};

type CreateBillingCheckoutSessionOptions = {
  tier: BillingPlanId;
  successUrl: string;
  cancelUrl: string;
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
};

type BillingAccountApiEnvelope = {
  success: boolean;
  data?: BillingAccountApiResponse;
};

type BillingAccountApiResponse = {
  principal: {
    userId: string;
    sessionId: string;
    role: string;
    email?: string;
  };
  access: {
    role: string;
    plan: BillingPlanId;
  };
  plan: {
    tier: BillingPlanId;
    name: string;
    currency: string;
    monthlyPriceCents: number;
    stripePriceId: string;
    enabledFeatureCount: number;
    disabledFeatureCount: number;
  };
  entitlements: Array<{
    feature: string;
    enabled: boolean;
    accessGranted: boolean;
    accessReason: BillingEntitlementAccessReason;
    maxGraphDepth: number;
    maxFreshnessSeconds: number;
    maxRequestsPerMinute: number;
  }>;
  issuedAt: string;
};

const planRank: Record<BillingPlanId, number> = {
  free: 0,
  pro: 1,
  team: 2,
};

const featureMetadata: Record<
  string,
  {
    label: string;
    description: string;
    requiredPlan: BillingPlanId;
    value: (input: {
      maxGraphDepth: number;
      maxFreshnessSeconds: number;
      maxRequestsPerMinute: number;
    }) => string;
  }
> = {
  search: {
    label: "Search and profile lookup",
    description: "Address and entity lookup with cached serving freshness.",
    requiredPlan: "free",
    value: ({ maxRequestsPerMinute }) => `${maxRequestsPerMinute} req/min`,
  },
  wallet_summary: {
    label: "Wallet summary",
    description: "Summary cards, counterparties, and latest signal snapshots.",
    requiredPlan: "free",
    value: ({ maxRequestsPerMinute }) => `${maxRequestsPerMinute} req/min`,
  },
  graph: {
    label: "Graph exploration",
    description: "Relationship graph depth for wallet investigation.",
    requiredPlan: "pro",
    value: ({ maxGraphDepth }) => `${Math.max(maxGraphDepth, 1)}-hop graph`,
  },
  cluster: {
    label: "Cluster intelligence",
    description: "Cluster score snapshots and cluster detail review.",
    requiredPlan: "pro",
    value: ({ maxRequestsPerMinute }) => `${maxRequestsPerMinute} req/min`,
  },
  shadow_exit: {
    label: "Shadow exit feed",
    description: "Review candidate treasury exits and bridge escape patterns.",
    requiredPlan: "pro",
    value: ({ maxFreshnessSeconds }) =>
      maxFreshnessSeconds > 0
        ? `${maxFreshnessSeconds}s freshness`
        : "Pro signal feed",
  },
  first_connection: {
    label: "First connection feed",
    description:
      "Surface novel counterparties within the recent lookback window.",
    requiredPlan: "pro",
    value: ({ maxFreshnessSeconds }) =>
      maxFreshnessSeconds > 0
        ? `${maxFreshnessSeconds}s freshness`
        : "Pro signal feed",
  },
  alerts: {
    label: "Alert rules",
    description: "Create alert rules, inbox events, and delivery channels.",
    requiredPlan: "pro",
    value: ({ maxRequestsPerMinute }) => `${maxRequestsPerMinute} req/min`,
  },
  watchlist: {
    label: "Watchlists",
    description: "Track curated wallets and trigger downstream indexing.",
    requiredPlan: "pro",
    value: ({ maxRequestsPerMinute }) => `${maxRequestsPerMinute} req/min`,
  },
  admin_console: {
    label: "Admin console",
    description: "Labels, suppressions, curated lists, quotas, and audit logs.",
    requiredPlan: "team",
    value: () => "operator role required",
  },
  billing_console: {
    label: "Billing console",
    description: "Plan controls and billing management surface.",
    requiredPlan: "team",
    value: () => "billing workflow baseline",
  },
};

const staticPlans: Array<Omit<BillingPlanPreview, "current">> = [
  {
    id: "free",
    name: "Free",
    priceLabel: "$0",
    summary: "Baseline exploration plan for a single operator.",
    features: ["1-hop graph", "cached wallet summary", "search preview"],
  },
  {
    id: "pro",
    name: "Pro",
    priceLabel: "$49",
    summary: "Adds deeper graph traversal, watchlists, and alerts.",
    features: ["2-hop graph", "watchlists", "alerts + signal feeds"],
  },
  {
    id: "team",
    name: "Team",
    priceLabel: "$199",
    summary: "For internal operators who also need admin controls.",
    features: ["admin console", "Telegram delivery", "higher quotas"],
  },
];

export const accountEntitlementsRoute = "GET /v1/account/entitlements";
export const billingCheckoutRoute = "POST /v1/billing/checkout-sessions";
export const pricingPageRoute = "/pricing";

export function buildPricingPlansHref(): string {
  return `${pricingPageRoute}#plans`;
}

export function normalizeBillingCheckoutQueryState({
  checkout,
  plan,
}: {
  checkout: string | string[] | undefined;
  plan: string | string[] | undefined;
}): BillingCheckoutQueryState {
  const checkoutValue = Array.isArray(checkout) ? checkout[0] : checkout;
  const planValue = Array.isArray(plan) ? plan[0] : plan;

  return {
    status:
      checkoutValue === "success"
        ? "success"
        : checkoutValue === "cancel"
          ? "cancel"
          : "idle",
    plan:
      planValue === "free" || planValue === "pro" || planValue === "team"
        ? planValue
        : undefined,
  };
}

export function describeBillingCheckoutQueryState({
  checkoutState,
  preview,
  surface,
}: {
  checkoutState: BillingCheckoutQueryState;
  preview: BillingAccountPreview;
  surface: BillingCheckoutSurface;
}): BillingCheckoutFlash | undefined {
  if (checkoutState.status === "idle") {
    return undefined;
  }

  const planLabel =
    preview.plans.find((plan) => plan.id === checkoutState.plan)?.name ??
    preview.checkoutIntent.recommendedPlanLabel;

  if (checkoutState.status === "success") {
    return {
      tone: "teal",
      title:
        surface === "account"
          ? "Checkout returned"
          : "Checkout returned to pricing",
      message:
        surface === "account"
          ? `Stripe checkout returned successfully. Refresh this account snapshot to confirm the ${planLabel} plan once billing reconciliation completes.`
          : `Stripe checkout returned successfully. Open account to confirm the ${planLabel} plan once billing reconciliation completes.`,
    };
  }

  return {
    tone: "amber",
    title: "Checkout canceled",
    message:
      surface === "account"
        ? "Checkout was canceled before completion. Your current account snapshot has not changed."
        : "Checkout was canceled before completion. Review the plan cards and start a new checkout when ready.",
  };
}

export function getBillingAccountPreview(): BillingAccountPreview {
  const currentPlan: BillingPlanId = "free";

  return {
    mode: "unavailable",
    source: "boundary-unavailable",
    route: accountEntitlementsRoute,
    statusMessage:
      "Account entitlements are unavailable until the billing APIs respond.",
    currentPlan,
    currentPlanLabel: "Unavailable",
    currentRoleLabel: "Unavailable",
    billingCycleLabel: "Unavailable",
    renewalLabel: "Billing status unavailable.",
    checkoutIntent: buildCheckoutIntentPreview(currentPlan),
    entitlements: [],
    plans: getBillingPlanMatrix(currentPlan).map((plan) => ({
      ...plan,
      current: false,
    })),
  };
}

export function getBillingPlanMatrix(
  currentPlan: BillingPlanId,
): BillingPlanPreview[] {
  return staticPlans.map((plan) => ({
    ...plan,
    current: plan.id === currentPlan,
  }));
}

export async function loadBillingAccountPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  requestHeaders,
}: LoadBillingAccountPreviewOptions = {}): Promise<BillingAccountPreview> {
  const nextFallback = fallback ?? getBillingAccountPreview();
  const endpoint = buildAccountEntitlementsUrl(apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: mergeRequestHeaders(
        {
          Accept: "application/json",
        },
        requestHeaders,
      ),
      cache: "no-store",
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as BillingAccountApiEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapBillingAccountResponse(payload.data);
  } catch {
    return nextFallback;
  }
}

export function isPlanAtLeast(
  currentPlan: BillingPlanId,
  requiredPlan: BillingPlanId,
): boolean {
  return planRank[currentPlan] >= planRank[requiredPlan];
}

const DEFAULT_QORVI_API_BASE_URL = "https://api.qorvi.app";

function getApiBaseUrl(apiBaseUrl?: string): string | undefined {
  const trimmed = apiBaseUrl?.trim();
  if (trimmed) {
    return trimmed;
  }

  const envBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL?.trim();
  return envBaseUrl ? envBaseUrl : DEFAULT_QORVI_API_BASE_URL;
}

function buildAccountEntitlementsUrl(apiBaseUrl?: string): string {
  const path = "/v1/account/entitlements";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}

function mapBillingAccountResponse(
  response: BillingAccountApiResponse,
): BillingAccountPreview {
  const currentPlan = response.plan.tier;
  const role = response.access.role.trim();
  const roleLabel =
    role.length > 0
      ? `${role.charAt(0).toUpperCase()}${role.slice(1)}`
      : "User";

  return {
    mode: "live",
    source: "live-api",
    route: accountEntitlementsRoute,
    statusMessage:
      "Live entitlement snapshot loaded from GET /v1/account/entitlements.",
    currentPlan,
    currentPlanLabel: response.plan.name,
    currentRoleLabel: roleLabel,
    billingCycleLabel: "Monthly",
    renewalLabel:
      response.plan.monthlyPriceCents > 0
        ? `Recurring ${formatPriceLabel(response.plan.monthlyPriceCents, response.plan.currency)} plan.`
        : "No renewal while on the Free plan.",
    checkoutIntent: buildCheckoutIntentPreview(currentPlan),
    entitlements: response.entitlements.map((item) =>
      mapEntitlementPreview(item, currentPlan),
    ),
    plans: getBillingPlanMatrix(currentPlan),
  };
}

function mapEntitlementPreview(
  item: BillingAccountApiResponse["entitlements"][number],
  currentPlan: BillingPlanId,
): BillingEntitlementPreview {
  const metadata = featureMetadata[item.feature] ?? {
    label: item.feature,
    description: "Feature gate exported from the account entitlement snapshot.",
    requiredPlan: currentPlan,
    value: ({ maxRequestsPerMinute }: { maxRequestsPerMinute: number }) =>
      `${maxRequestsPerMinute} req/min`,
  };

  return {
    id: item.feature,
    label: metadata.label,
    description: metadata.description,
    requiredPlan: metadata.requiredPlan,
    available: item.accessGranted,
    accessReason: item.accessReason,
    value: metadata.value({
      maxGraphDepth: item.maxGraphDepth,
      maxFreshnessSeconds: item.maxFreshnessSeconds,
      maxRequestsPerMinute: item.maxRequestsPerMinute,
    }),
  };
}

function formatPriceLabel(cents: number, currency: string): string {
  if (cents <= 0) {
    return "$0";
  }

  const normalizedCurrency = currency.trim().toUpperCase() || "USD";
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: normalizedCurrency,
    maximumFractionDigits: 0,
  }).format(cents / 100);
}

function buildCheckoutIntentPreview(
  currentPlan: BillingPlanId,
): BillingCheckoutIntentPreview {
  if (currentPlan === "free") {
    return {
      state: "preview",
      title: "Upgrade intent: Pro",
      description:
        "Review the plan cards on /pricing, then create a live checkout session for the smallest plan that unlocks 2-hop graph exploration, alerts, and watchlists.",
      recommendedPlan: "pro",
      recommendedPlanLabel: "Pro",
      checkoutRoute: billingCheckoutRoute,
      ctaLabel: "Review pricing",
      ctaHref: buildPricingPlansHref(),
    };
  }

  if (currentPlan === "pro") {
    return {
      state: "ready",
      title: "Upgrade intent: Team",
      description:
        "Review the plan cards on /pricing, then create a live checkout session for the Team plan that adds operator controls, admin console access, and higher launch quotas.",
      recommendedPlan: "team",
      recommendedPlanLabel: "Team",
      checkoutRoute: billingCheckoutRoute,
      ctaLabel: "Review pricing",
      ctaHref: buildPricingPlansHref(),
    };
  }

  return {
    state: "complete",
    title: "Current plan: Team",
    description:
      "This is the top plan in the current launch matrix. Use /pricing to confirm current access and operator surface.",
    recommendedPlan: "team",
    recommendedPlanLabel: "Team",
    checkoutRoute: billingCheckoutRoute,
    ctaLabel: "Review pricing",
    ctaHref: buildPricingPlansHref(),
  };
}

export async function createBillingCheckoutSession({
  tier,
  successUrl,
  cancelUrl,
  apiBaseUrl,
  fetchImpl = fetch,
}: CreateBillingCheckoutSessionOptions): Promise<BillingCheckoutMutationResult> {
  const endpoint = buildBillingCheckoutUrl(apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify({
        tier,
        successUrl,
        cancelUrl,
      }),
    });

    if (!response.ok) {
      return {
        ok: false,
        status: "unavailable",
        message:
          response.status === 401
            ? "Sign in with an active session to continue checkout."
            : "Checkout is temporarily unavailable. Review the pricing options and try again shortly.",
      };
    }

    const payload = (await response.json()) as {
      success?: boolean;
      data?: {
        checkoutSession?: {
          url?: string;
        };
        checkoutUrl?: string;
        url?: string;
      };
    };
    const redirectUrl =
      payload.data?.checkoutSession?.url?.trim() ??
      payload.data?.checkoutUrl?.trim() ??
      payload.data?.url?.trim();
    if (!payload.success || !redirectUrl) {
      return {
        ok: false,
        status: "unavailable",
        message:
          "Checkout could not be started from this response. Review pricing and try again.",
      };
    }

    return {
      ok: true,
      status: "redirect",
      redirectUrl,
      message: "Redirecting to Stripe checkout preview.",
    };
  } catch {
    return {
      ok: false,
      status: "unavailable",
      message: "Checkout is unavailable in the current environment.",
    };
  }
}

function buildBillingCheckoutUrl(apiBaseUrl?: string): string {
  const path = "/v1/billing/checkout-sessions";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);
  if (!resolvedBaseUrl) {
    return path;
  }
  return new URL(path, resolvedBaseUrl).toString();
}
import { mergeRequestHeaders } from "./request-headers";
