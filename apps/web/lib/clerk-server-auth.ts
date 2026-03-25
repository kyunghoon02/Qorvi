import { auth } from "@clerk/nextjs/server";

import { resolveClerkRole } from "./clerk-role";
import { createForwardedAuthHeaders } from "./request-headers";

export async function buildClerkRequestHeaders(): Promise<
  HeadersInit | undefined
> {
  const authState = await auth();
  const token = await authState.getToken();

  return createForwardedAuthHeaders({
    bearerToken: token ?? undefined,
    userId: authState.userId ?? undefined,
    sessionId: authState.sessionId ?? undefined,
    role: resolveClerkRole(authState.sessionClaims),
    plan: undefined,
  });
}
