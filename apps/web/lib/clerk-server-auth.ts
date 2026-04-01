import { auth, currentUser } from "@clerk/nextjs/server";

import { resolveClerkRole } from "./clerk-role";
import { createForwardedAuthHeaders } from "./request-headers";

export async function buildClerkRequestHeaders(): Promise<
  HeadersInit | undefined
> {
  const authState = await auth();
  const user = await currentUser();
  const token = await authState.getToken();
  const role =
    resolveClerkRole(authState.sessionClaims) ?? resolveClerkRole(user);

  return createForwardedAuthHeaders({
    bearerToken: token ?? undefined,
    userId: authState.userId ?? undefined,
    sessionId: authState.sessionId ?? undefined,
    role,
    plan: undefined,
  });
}
