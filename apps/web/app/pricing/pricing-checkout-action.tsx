"use client";

import { useState } from "react";

import { Badge } from "@qorvi/ui";

import {
  type BillingPlanId,
  createBillingCheckoutSession,
  pricingPageRoute,
} from "../../lib/account-billing";
import { useClerkRequestHeaders } from "../../lib/clerk-client-auth";

type PricingCheckoutActionProps = {
  planId: BillingPlanId;
  planLabel: string;
  checkoutRoute: string;
  currentPlan: BillingPlanId;
};

export function PricingCheckoutAction({
  planId,
  planLabel,
  checkoutRoute,
  currentPlan,
}: PricingCheckoutActionProps) {
  const [statusMessage, setStatusMessage] = useState("");
  const [checkoutUrl, setCheckoutUrl] = useState<string | undefined>();
  const [isPending, setIsPending] = useState(false);
  const getRequestHeaders = useClerkRequestHeaders();
  const isCurrentPlan = currentPlan === planId;

  async function handleCheckout() {
    if (isCurrentPlan || isPending) {
      return;
    }

    setIsPending(true);
    setStatusMessage(`Creating a checkout session for ${planLabel}...`);
    setCheckoutUrl(undefined);

    const origin = window.location.origin;
    const requestHeaders = await getRequestHeaders();
    const result = await createBillingCheckoutSession({
      tier: planId,
      successUrl: `${origin}/account?checkout=success&plan=${planId}`,
      cancelUrl: `${origin}${pricingPageRoute}?checkout=cancel&plan=${planId}`,
      ...(requestHeaders ? { requestHeaders } : {}),
    });

    setIsPending(false);
    setStatusMessage(result.message);
    setCheckoutUrl(result.ok ? result.redirectUrl : undefined);
  }

  return (
    <div className="pricing-plan-actions">
      {isCurrentPlan ? (
        <Badge tone="teal">Current plan</Badge>
      ) : (
        <button
          type="button"
          className="search-cta"
          onClick={() => void handleCheckout()}
          disabled={isPending}
        >
          {isPending ? "Creating checkout..." : "Create checkout session"}
        </button>
      )}

      <span className="detail-route-copy">{checkoutRoute}</span>

      {statusMessage ? (
        <p className="pricing-plan-message" aria-live="polite">
          {statusMessage}
        </p>
      ) : null}

      {checkoutUrl ? (
        <a className="search-cta" href={checkoutUrl} rel="noreferrer">
          Continue to checkout
        </a>
      ) : null}
    </div>
  );
}
