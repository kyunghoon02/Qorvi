import { headers } from "next/headers";

import { buildForwardedAuthHeaders } from "../lib/request-headers";

import { HomeScreen } from "./home-screen";

export default function Page() {
  const requestHeaders = buildForwardedAuthHeaders(headers());

  return <HomeScreen {...(requestHeaders ? { requestHeaders } : {})} />;
}
