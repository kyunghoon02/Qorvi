import React from "react";

import { Badge, Pill, type Tone } from "@flowintel/ui";

import type {
  BillingAccountPreview,
  BillingCheckoutQueryState,
  BillingPlanId,
} from "../../lib/account-billing";
import {
  describeBillingCheckoutQueryState,
  pricingPageRoute,
} from "../../lib/account-billing";

import { BillingCheckoutAction } from "./billing-checkout-action";

const toneByPlan: Record<BillingPlanId, Tone> = {
  free: "amber",
  pro: "violet",
  team: "teal",
};

export type AccountBillingViewModel = {
  title: string;
  explanation: string;
  statusMessage: string;
  route: string;
  checkoutFlash:
    | {
        tone: Tone;
        title: string;
        message: string;
      }
    | undefined;
  currentPlanLabel: string;
  currentRoleLabel: string;
  billingCycleLabel: string;
  renewalLabel: string;
  checkoutIntent: BillingAccountPreview["checkoutIntent"] & {
    tone: Tone;
    routeLabel: string;
  };
  entitlements: Array<
    BillingAccountPreview["entitlements"][number] & {
      tone: Tone;
      availabilityLabel: string;
      accessHint: string;
    }
  >;
  plans: Array<
    BillingAccountPreview["plans"][number] & {
      tone: Tone;
    }
  >;
};

export function buildAccountBillingViewModel({
  preview,
  checkoutState,
}: {
  preview: BillingAccountPreview;
  checkoutState?: BillingCheckoutQueryState;
}): AccountBillingViewModel {
  const checkoutFlash = describeBillingCheckoutQueryState({
    checkoutState: checkoutState ?? { status: "idle", plan: undefined },
    preview,
    surface: "account",
  });

  return {
    title: "Account & billing",
    explanation:
      "Review the current plan, understand which features are unlocked, and continue checkout when you want to upgrade access.",
    statusMessage: preview.statusMessage,
    route: preview.route,
    checkoutFlash: checkoutFlash
      ? {
          ...checkoutFlash,
          tone: checkoutFlash.tone,
        }
      : undefined,
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
    entitlements: preview.entitlements.map((item) => ({
      ...item,
      tone: toneByPlan[item.requiredPlan],
      availabilityLabel: item.available ? "available" : "locked",
      accessHint: item.available
        ? "Included in the current access context."
        : item.accessReason === "role_required"
          ? "Requires an admin or operator role."
          : `Requires ${item.requiredPlan.toUpperCase()} or higher.`,
    })),
    plans: preview.plans.map((item) => ({
      ...item,
      tone: toneByPlan[item.id],
    })),
  };
}

export function AccountBillingScreen({
  preview,
  checkoutState,
}: {
  preview: BillingAccountPreview;
  checkoutState?: BillingCheckoutQueryState;
}) {
  const viewModel = buildAccountBillingViewModel({
    preview,
    ...(checkoutState ? { checkoutState } : {}),
  });

  return (
    <main className="page-shell detail-shell">
      <section className="detail-hero alert-center-hero">
        <div className="eyebrow-row">
          <Pill tone="amber">Account</Pill>
          <Pill tone="violet">billing plan</Pill>
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
          <a className="search-cta" href={viewModel.checkoutIntent.ctaHref}>
            {viewModel.checkoutIntent.ctaLabel}
          </a>
          <a className="search-cta" href="/">
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

          <div className="detail-actions">
            {viewModel.checkoutIntent.state === "complete" ? (
              <a className="search-cta" href={viewModel.checkoutIntent.ctaHref}>
                {viewModel.checkoutIntent.ctaLabel}
              </a>
            ) : (
              <BillingCheckoutAction
                ctaLabel={viewModel.checkoutIntent.ctaLabel}
                fallbackHref={pricingPageRoute}
                tier={viewModel.checkoutIntent.recommendedPlan}
              />
            )}
            <span className="detail-route-copy">
              {viewModel.checkoutIntent.routeLabel}
            </span>
          </div>

          <div className="cluster-member-meta">
            <Pill tone={viewModel.checkoutIntent.tone}>
              {viewModel.checkoutIntent.recommendedPlanLabel}
            </Pill>
            <span>
              {viewModel.checkoutIntent.state === "complete"
                ? "No upgrade required"
                : "Upgrade baseline available"}
            </span>
          </div>
        </article>

        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Plan snapshot</span>
              <h2>{viewModel.currentPlanLabel}</h2>
            </div>
            <div className="preview-state">
              <Badge tone="amber">
                {preview.currentPlan === "free"
                  ? "baseline"
                  : `${preview.currentPlanLabel} active`}
              </Badge>
            </div>
          </div>

          <div className="preview-status">
            <span className="preview-kicker">Data status</span>
            <p>{viewModel.statusMessage}</p>
          </div>

          <div className="alert-inbox-list">
            {viewModel.entitlements.map((entitlement) => (
              <article key={entitlement.id} className="alert-inbox-item">
                <div className="alert-inbox-topline">
                  <strong>{entitlement.label}</strong>
                  <Badge tone={entitlement.available ? "teal" : "violet"}>
                    {entitlement.availabilityLabel}
                  </Badge>
                </div>
                <p>{entitlement.description}</p>
                <div className="cluster-member-meta">
                  <Pill tone={entitlement.tone}>
                    {entitlement.requiredPlan.toUpperCase()}
                  </Pill>
                  <span>{entitlement.value}</span>
                </div>
              </article>
            ))}
          </div>
        </article>

        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Gating</span>
              <h2>Feature access</h2>
            </div>
          </div>

          <div className="alert-inbox-list">
            {viewModel.entitlements.map((entitlement) => (
              <article key={entitlement.id} className="alert-inbox-item">
                <div className="alert-inbox-topline">
                  <strong>{entitlement.value}</strong>
                  <Badge tone={entitlement.available ? "teal" : "amber"}>
                    {entitlement.available ? "unlocked" : "upgrade required"}
                  </Badge>
                </div>
                <p>
                  {entitlement.available
                    ? `Your ${viewModel.currentPlanLabel} plan includes this capability.`
                    : entitlement.accessHint}
                </p>
              </article>
            ))}
          </div>
        </article>

        <article className="preview-card detail-card">
          <div id="plan-matrix" />
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Plan matrix</span>
              <h2>Free / Pro / Team</h2>
            </div>
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
                  {plan.features.map((feature) => (
                    <Pill key={`${plan.id}-${feature}`} tone={plan.tone}>
                      {feature}
                    </Pill>
                  ))}
                </div>
              </article>
            ))}
          </div>
        </article>
      </section>
    </main>
  );
}
