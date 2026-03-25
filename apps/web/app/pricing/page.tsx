import {
  loadBillingAccountPreview,
  normalizeBillingCheckoutQueryState,
} from "../../lib/account-billing";
import { buildClerkRequestHeaders } from "../../lib/clerk-server-auth";

import { PricingScreen } from "./pricing-screen";

type PricingPageProps = {
  searchParams?:
    | Promise<{ checkout?: string | string[]; plan?: string | string[] }>
    | { checkout?: string | string[]; plan?: string | string[] };
};

export default async function PricingPage({ searchParams }: PricingPageProps) {
  const requestHeaders = await buildClerkRequestHeaders();
  const resolvedSearchParams = searchParams
    ? await Promise.resolve(searchParams)
    : undefined;
  const preview = await loadBillingAccountPreview(
    requestHeaders ? { requestHeaders } : undefined,
  );
  const checkoutState = normalizeBillingCheckoutQueryState({
    checkout: resolvedSearchParams?.checkout,
    plan: resolvedSearchParams?.plan,
  });

  return <PricingScreen preview={preview} checkoutState={checkoutState} />;
}
