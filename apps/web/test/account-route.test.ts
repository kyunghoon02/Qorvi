import assert from "node:assert/strict";
import test from "node:test";

import { buildAccountBillingViewModel } from "../app/account/account-screen";
import {
  accountEntitlementsRoute,
  billingCheckoutRoute,
  buildPricingPlansHref,
  createBillingCheckoutSession,
  describeBillingCheckoutQueryState,
  getBillingAccountPreview,
  isPlanAtLeast,
  loadBillingAccountPreview,
  normalizeBillingCheckoutQueryState,
} from "../lib/account-billing";

test("buildAccountBillingViewModel carries current plan and entitlement gates", () => {
  const viewModel = buildAccountBillingViewModel({
    preview: getBillingAccountPreview(),
  });

  assert.equal(viewModel.title, "Account & billing");
  assert.equal(viewModel.currentPlanLabel, "Unavailable");
  assert.equal(viewModel.currentRoleLabel, "Unavailable");
  assert.equal(viewModel.entitlements.length, 0);
  assert.equal(viewModel.checkoutIntent.routeLabel, billingCheckoutRoute);
  assert.equal(viewModel.checkoutIntent.ctaLabel, "Review pricing");
  assert.equal(viewModel.checkoutIntent.ctaHref, buildPricingPlansHref());
  assert.equal(viewModel.plans.length, 3);
  assert.equal(viewModel.plans[0]?.current, false);
  assert.equal(viewModel.plans[1]?.current, false);
});

test("normalizeBillingCheckoutQueryState maps checkout query params", () => {
  const successState = normalizeBillingCheckoutQueryState({
    checkout: "success",
    plan: "team",
  });
  const cancelState = normalizeBillingCheckoutQueryState({
    checkout: ["cancel"],
    plan: ["pro"],
  });
  const idleState = normalizeBillingCheckoutQueryState({
    checkout: "unknown",
    plan: "invalid",
  });

  assert.deepEqual(successState, { status: "success", plan: "team" });
  assert.deepEqual(cancelState, { status: "cancel", plan: "pro" });
  assert.deepEqual(idleState, { status: "idle", plan: undefined });
});

test("buildAccountBillingViewModel exposes success flash when checkout returns", () => {
  const preview = getBillingAccountPreview();
  const viewModel = buildAccountBillingViewModel({
    preview,
    checkoutState: { status: "success", plan: "pro" },
  });

  assert.equal(viewModel.checkoutFlash?.tone, "teal");
  assert.match(viewModel.checkoutFlash?.message ?? "", /confirm the Pro plan/i);
});

test("describeBillingCheckoutQueryState explains account cancel state", () => {
  const flash = describeBillingCheckoutQueryState({
    checkoutState: { status: "cancel", plan: "pro" },
    preview: getBillingAccountPreview(),
    surface: "account",
  });

  assert.equal(flash?.tone, "amber");
  assert.match(flash?.message ?? "", /canceled before completion/i);
});

test("plan comparison helper treats Pro and Team as upgrades over Free", () => {
  assert.equal(isPlanAtLeast("free", "free"), true);
  assert.equal(isPlanAtLeast("free", "pro"), false);
  assert.equal(isPlanAtLeast("pro", "pro"), true);
  assert.equal(isPlanAtLeast("team", "pro"), true);
  assert.equal(isPlanAtLeast("pro", "team"), false);
});

test("account entitlement route stays aligned with the backend contract", () => {
  assert.equal(accountEntitlementsRoute, "GET /v1/account/entitlements");
});

test("loadBillingAccountPreview maps live entitlement snapshots when available", async () => {
  let requestedUrl = "";

  const preview = await loadBillingAccountPreview({
    fetchImpl: async (input) => {
      requestedUrl = String(input);

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            principal: {
              userId: "user_123",
              sessionId: "session_123",
              role: "operator",
            },
            access: {
              role: "operator",
              plan: "team",
            },
            plan: {
              tier: "team",
              name: "Team",
              currency: "usd",
              monthlyPriceCents: 19900,
              stripePriceId: "price_team",
              enabledFeatureCount: 9,
              disabledFeatureCount: 1,
            },
            entitlements: [
              {
                feature: "graph",
                enabled: true,
                accessGranted: true,
                accessReason: "granted",
                maxGraphDepth: 3,
                maxFreshnessSeconds: 300,
                maxRequestsPerMinute: 120,
              },
              {
                feature: "admin_console",
                enabled: true,
                accessGranted: true,
                accessReason: "granted",
                maxGraphDepth: 3,
                maxFreshnessSeconds: 300,
                maxRequestsPerMinute: 48,
              },
            ],
            issuedAt: "2026-03-21T00:00:00Z",
          },
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json",
          },
        },
      );
    },
  });

  assert.equal(requestedUrl, "/v1/account/entitlements");
  assert.equal(preview.mode, "live");
  assert.equal(preview.currentPlan, "team");
  assert.equal(preview.currentRoleLabel, "Operator");
  assert.equal(preview.checkoutIntent.title, "Current plan: Team");
  assert.equal(preview.entitlements[0]?.value, "3-hop graph");
  assert.equal(preview.entitlements[1]?.available, true);
});

test("createBillingCheckoutSession returns redirect url when API succeeds", async () => {
  const result = await createBillingCheckoutSession({
    tier: "pro",
    successUrl: "http://localhost:3000/account?checkout=success",
    cancelUrl: "http://localhost:3000/account?checkout=cancel",
    fetchImpl: async () =>
      new Response(
        JSON.stringify({
          success: true,
          data: {
            checkoutSession: {
              url: "https://checkout.stripe.test/session/cs_test",
            },
          },
        }),
        {
          status: 201,
          headers: {
            "Content-Type": "application/json",
          },
        },
      ),
  });

  assert.equal(result.ok, true);
  assert.equal(result.status, "redirect");
  assert.equal(
    result.redirectUrl,
    "https://checkout.stripe.test/session/cs_test",
  );
});

test("createBillingCheckoutSession falls back on unauthorized response", async () => {
  const result = await createBillingCheckoutSession({
    tier: "team",
    successUrl: "http://localhost:3000/account?checkout=success",
    cancelUrl: "http://localhost:3000/account?checkout=cancel",
    fetchImpl: async () => new Response("{}", { status: 401 }),
  });

  assert.equal(result.ok, false);
  assert.equal(result.status, "unavailable");
  assert.match(result.message, /sign in/i);
});
