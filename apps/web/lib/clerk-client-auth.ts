"use client";

import { useAuth, useUser } from "@clerk/clerk-react";
import { useCallback } from "react";

import { resolveClerkRole } from "./clerk-role";
import { createForwardedAuthHeaders } from "./request-headers";

export function useClerkRequestHeaders(): () => Promise<
  HeadersInit | undefined
> {
  const { userId, sessionId, sessionClaims, getToken } = useAuth();
  const { user } = useUser();

  return useCallback(async () => {
    const token = await getToken();
    const role = resolveClerkRole(sessionClaims) ?? resolveClerkRole(user);

    return createForwardedAuthHeaders({
      bearerToken: token ?? undefined,
      userId: userId ?? undefined,
      sessionId: sessionId ?? undefined,
      role,
      plan: undefined,
    });
  }, [getToken, sessionClaims, sessionId, user, userId]);
}
