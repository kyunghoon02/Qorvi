import assert from "node:assert/strict";
import test from "node:test";

import { buildPricingViewModel } from "../app/pricing/pricing-screen";
import {
  buildPricingPlansHref,
  createBillingCheckoutSession,
  describeBillingCheckoutQueryState,
  getBillingAccountPreview,
  normalizeBillingCheckoutQueryState,
  pricingPageRoute,
} from "../lib/account-billing";

test("buildPricingViewModel keeps the upgrade intent aligned with the pricing page", () => {
  const viewModel = buildPricingViewModel({
    preview: getBillingAccountPreview(),
  });

  assert.equal(viewModel.title, "Pricing and checkout");
  assert.equal(viewModel.checkoutIntent.ctaHref, buildPricingPlansHref());
  assert.equal(
    viewModel.checkoutIntent.routeLabel,
    "POST /v1/billing/checkout-sessions",
  );
  assert.equal(viewModel.plans.length, 3);
  assert.equal(viewModel.plans[1]?.isRecommended, true);
  assert.equal(viewModel.plans[0]?.current, false);
});

test("buildPricingViewModel exposes cancel flash state from query params", () => {
  const viewModel = buildPricingViewModel({
    preview: getBillingAccountPreview(),
    checkoutState: { status: "cancel", plan: "pro" },
  });

  assert.equal(viewModel.checkoutFlash?.tone, "amber");
  assert.match(viewModel.checkoutFlash?.message ?? "", /start a new checkout/i);
});

test("normalizeBillingCheckoutQueryState keeps unknown pricing state idle", () => {
  const state = normalizeBillingCheckoutQueryState({
    checkout: "noop",
    plan: "enterprise",
  });

  assert.deepEqual(state, { status: "idle", plan: undefined });
});

test("describeBillingCheckoutQueryState explains pricing success state", () => {
  const flash = describeBillingCheckoutQueryState({
    checkoutState: { status: "success", plan: "team" },
    preview: getBillingAccountPreview(),
    surface: "pricing",
  });

  assert.equal(flash?.tone, "teal");
  assert.match(flash?.message ?? "", /Open account to confirm the Team plan/i);
});

test("createBillingCheckoutSession returns a redirect when checkout url is present", async () => {
  let requestedUrl = "";
  let requestedBody = "";

  const result = await createBillingCheckoutSession({
    tier: "pro",
    successUrl: "https://example.test/account?checkout=success",
    cancelUrl: "https://example.test/pricing?checkout=cancel",
    apiBaseUrl: "https://api.example",
    fetchImpl: async (input, init) => {
      requestedUrl = String(input);
      requestedBody = String(init?.body ?? "");

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            checkoutSession: {
              url: "https://checkout.example/session_123",
            },
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

  assert.equal(
    requestedUrl,
    `${"https://api.example"}/v1/billing/checkout-sessions`,
  );
  assert.match(requestedBody, /"tier":"pro"/);
  assert.match(
    requestedBody,
    /"successUrl":"https:\/\/example\.test\/account\?checkout=success"/,
  );
  assert.match(
    requestedBody,
    /"cancelUrl":"https:\/\/example\.test\/pricing\?checkout=cancel"/,
  );
  assert.equal(result.ok, true);
  assert.equal(result.status, "redirect");
  assert.equal(result.redirectUrl, "https://checkout.example/session_123");
});

test("createBillingCheckoutSession falls back when auth is unavailable", async () => {
  const result = await createBillingCheckoutSession({
    tier: "team",
    successUrl: "https://example.test/account?checkout=success",
    cancelUrl: "https://example.test/pricing?checkout=cancel",
    apiBaseUrl: "https://api.example",
    fetchImpl: async () =>
      new Response("", {
        status: 401,
      }),
  });

  assert.equal(result.ok, false);
  assert.equal(result.status, "unavailable");
  assert.match(result.message, /sign in/i);
});

test("pricing page route stays aligned with the web surface", () => {
  assert.equal(pricingPageRoute, "/pricing");
});
