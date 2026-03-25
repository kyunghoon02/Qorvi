import {
  loadBillingAccountPreview,
  normalizeBillingCheckoutQueryState,
} from "../../lib/account-billing";
import { buildClerkRequestHeaders } from "../../lib/clerk-server-auth";

import { AccountBillingScreen } from "./account-screen";

type AccountPageProps = {
  searchParams?:
    | Promise<{ checkout?: string | string[]; plan?: string | string[] }>
    | { checkout?: string | string[]; plan?: string | string[] };
};

export default async function AccountPage({ searchParams }: AccountPageProps) {
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

  return (
    <AccountBillingScreen preview={preview} checkoutState={checkoutState} />
  );
}
