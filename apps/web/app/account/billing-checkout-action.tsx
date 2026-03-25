"use client";

import { useState } from "react";

import {
  type BillingPlanId,
  createBillingCheckoutSession,
} from "../../lib/account-billing";
import { useClerkRequestHeaders } from "../../lib/clerk-client-auth";

type BillingCheckoutActionProps = {
  tier: BillingPlanId;
  ctaLabel: string;
  fallbackHref: string;
  fallbackLabel?: string;
};

export function BillingCheckoutAction({
  tier,
  ctaLabel,
  fallbackHref,
  fallbackLabel = "Review pricing instead",
}: BillingCheckoutActionProps) {
  const [status, setStatus] = useState<string>("");
  const [pending, setPending] = useState(false);
  const getRequestHeaders = useClerkRequestHeaders();

  async function handleCheckout() {
    if (typeof window === "undefined") {
      return;
    }

    setPending(true);
    setStatus("");

    const origin = window.location.origin;
    const requestHeaders = await getRequestHeaders();
    const result = await createBillingCheckoutSession({
      tier,
      successUrl: `${origin}/account?checkout=success`,
      cancelUrl: `${origin}/account?checkout=cancel`,
      ...(requestHeaders ? { requestHeaders } : {}),
    });

    if (result.ok && result.redirectUrl) {
      window.location.href = result.redirectUrl;
      return;
    }

    setStatus(result.message);
    setPending(false);
  }

  return (
    <div className="billing-checkout-action">
      <button
        className="search-cta"
        disabled={pending}
        onClick={handleCheckout}
        type="button"
      >
        {pending ? "Preparing checkout..." : ctaLabel}
      </button>
      <a
        className="detail-route-copy billing-fallback-link"
        href={fallbackHref}
      >
        {fallbackLabel}
      </a>
      {status ? <p className="billing-inline-status">{status}</p> : null}
    </div>
  );
}
