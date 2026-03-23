import React from "react";

import { Badge, Pill, type Tone } from "@whalegraph/ui";

import type {
  BillingAccountPreview,
  BillingCheckoutQueryState,
  BillingPlanId,
} from "../../lib/account-billing";
import {
  billingCheckoutRoute,
  describeBillingCheckoutQueryState,
  pricingPageRoute,
} from "../../lib/account-billing";

import { PricingCheckoutAction } from "./pricing-checkout-action";

const toneByPlan: Record<BillingPlanId, Tone> = {
  free: "amber",
  pro: "violet",
  team: "teal",
};

export type PricingViewModel = {
  title: string;
  explanation: string;
  route: string;
  currentPlan: BillingPlanId;
  checkoutFlash:
    | {
        tone: Tone;
        title: string;
        message: string;
      }
    | undefined;
  statusMessage: string;
  currentPlanLabel: string;
  currentRoleLabel: string;
  billingCycleLabel: string;
  renewalLabel: string;
  checkoutIntent: BillingAccountPreview["checkoutIntent"] & {
    tone: Tone;
    routeLabel: string;
  };
  plans: Array<
    BillingAccountPreview["plans"][number] & {
      tone: Tone;
      isRecommended: boolean;
      canCheckout: boolean;
      upgradeLabel: string;
    }
  >;
};

export function buildPricingViewModel({
  preview,
  checkoutState,
}: {
  preview: BillingAccountPreview;
  checkoutState?: BillingCheckoutQueryState;
}): PricingViewModel {
  const checkoutFlash = describeBillingCheckoutQueryState({
    checkoutState: checkoutState ?? { status: "idle", plan: undefined },
    preview,
    surface: "pricing",
  });

  return {
    title: "Pricing and checkout",
    explanation:
      "Compare plans, review the current access snapshot, and continue to checkout when you are ready.",
    route: pricingPageRoute,
    currentPlan: preview.currentPlan,
    checkoutFlash: checkoutFlash
      ? {
          ...checkoutFlash,
          tone: checkoutFlash.tone,
        }
      : undefined,
    statusMessage: preview.statusMessage,
    currentPlanLabel: preview.currentPlanLabel,
    currentRoleLabel: preview.currentRoleLabel,
    billingCycleLabel: preview.billingCycleLabel,
    renewalLabel: preview.renewalLabel,
    checkoutIntent: {
      ...preview.checkoutIntent,
      tone:
        preview.checkoutIntent.recommendedPlan === "free"
          ? toneByPlan.free
          : toneByPlan[preview.checkoutIntent.recommendedPlan],
      routeLabel: preview.checkoutIntent.checkoutRoute,
    },
    plans: preview.plans.map((item) => ({
      ...item,
      tone: toneByPlan[item.id],
      isRecommended: item.id === preview.checkoutIntent.recommendedPlan,
      canCheckout: !item.current && item.id !== "free",
      upgradeLabel:
        item.id === "free"
          ? "Free baseline"
          : item.current
            ? "Current plan"
            : `Upgrade to ${item.name}`,
    })),
  };
}

export function PricingScreen({
  preview,
  checkoutState,
}: {
  preview: BillingAccountPreview;
  checkoutState?: BillingCheckoutQueryState;
}) {
  const viewModel = buildPricingViewModel({
    preview,
    ...(checkoutState ? { checkoutState } : {}),
  });

  return (
    <main className="page-shell detail-shell">
      <section className="detail-hero alert-center-hero">
        <div className="eyebrow-row">
          <Pill tone="amber">Billing</Pill>
          <Pill tone="violet">pricing</Pill>
        </div>

        <div className="detail-hero-copy">
          <h1>{viewModel.title}</h1>
          <p>{viewModel.explanation}</p>
        </div>

        <div className="detail-identity">
          <div>
            <span>Current plan</span>
            <strong>{viewModel.currentPlanLabel}</strong>
          </div>
          <div>
            <span>Billing cycle</span>
            <strong>{viewModel.billingCycleLabel}</strong>
          </div>
          <div>
            <span>Role</span>
            <strong>{viewModel.currentRoleLabel}</strong>
          </div>
          <div>
            <span>Renewal</span>
            <strong>{viewModel.renewalLabel}</strong>
          </div>
        </div>

        <div className="detail-actions">
          <a className="search-cta" href="/account">
            Back to account
          </a>
          <a className="search-cta" href="/#wallet-search">
            Back to search
          </a>
          <span className="detail-route-copy">{viewModel.route}</span>
        </div>

        {viewModel.checkoutFlash ? (
          <div className="preview-status detail-status-inline">
            <span className="preview-kicker">
              {viewModel.checkoutFlash.title}
            </span>
            <div className="cluster-member-meta">
              <Badge tone={viewModel.checkoutFlash.tone}>
                {viewModel.checkoutFlash.tone === "teal"
                  ? "success"
                  : "canceled"}
              </Badge>
              <span>{viewModel.checkoutFlash.message}</span>
            </div>
          </div>
        ) : null}
      </section>

      <section className="detail-grid alert-center-grid">
        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Checkout state</span>
              <h2>{viewModel.checkoutIntent.title}</h2>
            </div>
            <div className="preview-state">
              <Badge tone={preview.mode === "live" ? "teal" : "amber"}>
                {preview.mode === "live" ? "live snapshot" : "unavailable"}
              </Badge>
            </div>
          </div>

          <div className="preview-status">
            <span className="preview-kicker">Upgrade intent</span>
            <p>{viewModel.checkoutIntent.description}</p>
          </div>

          <div className="cluster-member-meta">
            <Pill tone={viewModel.checkoutIntent.tone}>
              {viewModel.checkoutIntent.recommendedPlanLabel}
            </Pill>
            <span>{viewModel.checkoutIntent.routeLabel}</span>
          </div>

          <div className="pricing-plan-message">
            <Badge tone="violet">{viewModel.checkoutIntent.state}</Badge>
            <span>{viewModel.checkoutIntent.ctaHref}</span>
          </div>
        </article>

        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Data status</span>
              <h2>{viewModel.currentPlanLabel}</h2>
            </div>
            <div className="preview-state">
              <Badge tone={preview.mode === "live" ? "teal" : "amber"}>
                {preview.mode === "live" ? "live data" : "unavailable"}
              </Badge>
            </div>
          </div>

          <div className="preview-status">
            <span className="preview-kicker">Session state</span>
            <p>{viewModel.statusMessage}</p>
          </div>

          <div className="alert-inbox-list">
            {viewModel.plans.map((plan) => (
              <article key={plan.id} className="alert-inbox-item">
                <div className="alert-inbox-topline">
                  <strong>
                    {plan.name} {plan.current ? "(current)" : ""}
                  </strong>
                  <Badge tone={plan.current ? "teal" : plan.tone}>
                    {plan.priceLabel}
                  </Badge>
                </div>
                <p>{plan.summary}</p>
                <div className="cluster-member-meta">
                  <Pill tone={plan.tone}>
                    {plan.isRecommended
                      ? "recommended"
                      : (plan.features[0] ?? "")}
                  </Pill>
                  <span>{plan.features.join(" • ")}</span>
                </div>
              </article>
            ))}
          </div>
        </article>
      </section>

      <section className="panel-section boundary-card" id="plans">
        <div className="section-header">
          <div>
            <span className="preview-kicker">Plan cards</span>
            <h2>Review a plan, then create a checkout session</h2>
          </div>
          <Badge tone="violet">{billingCheckoutRoute}</Badge>
        </div>

        <div className="pricing-grid">
          {viewModel.plans.map((plan) => (
            <article
              key={plan.id}
              className="preview-card detail-card pricing-plan-card"
            >
              <div className="preview-header">
                <div>
                  <span className="preview-kicker">Plan</span>
                  <h2>
                    {plan.name} {plan.current ? "(current)" : ""}
                  </h2>
                </div>
                <div className="preview-state">
                  <Badge tone={plan.current ? "teal" : plan.tone}>
                    {plan.current
                      ? "current"
                      : plan.isRecommended
                        ? "recommended"
                        : "available"}
                  </Badge>
                  <Pill tone={plan.tone}>{plan.priceLabel}</Pill>
                </div>
              </div>

              <div className="preview-status">
                <span className="preview-kicker">Summary</span>
                <p>{plan.summary}</p>
              </div>

              <div className="cluster-member-meta">
                <Pill tone={plan.tone}>{plan.upgradeLabel}</Pill>
                <span>
                  {plan.canCheckout
                    ? "Checkout preview available"
                    : "No checkout action required"}
                </span>
              </div>

              <div className="cluster-member-meta">
                {plan.features.map((feature) => (
                  <Pill key={`${plan.id}-${feature}`} tone={plan.tone}>
                    {feature}
                  </Pill>
                ))}
              </div>

              <PricingCheckoutAction
                planId={plan.id}
                planLabel={plan.name}
                checkoutRoute={billingCheckoutRoute}
                currentPlan={preview.currentPlan}
              />
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}
