"use client";

import { useAuth } from "@clerk/clerk-react";
import { useCallback } from "react";

import { resolveClerkRole } from "./clerk-role";
import { createForwardedAuthHeaders } from "./request-headers";

export function useClerkRequestHeaders(): () => Promise<
  HeadersInit | undefined
> {
  const { userId, sessionId, sessionClaims, getToken } = useAuth();

  return useCallback(async () => {
    const token = await getToken();

    return createForwardedAuthHeaders({
      bearerToken: token ?? undefined,
      userId: userId ?? undefined,
      sessionId: sessionId ?? undefined,
      role: resolveClerkRole(sessionClaims),
      plan: undefined,
    });
  }, [getToken, sessionClaims, sessionId, userId]);
}
