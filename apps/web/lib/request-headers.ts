const forwardedAuthHeaderNames = [
  "x-clerk-user-id",
  "x-clerk-session-id",
  "x-clerk-role",
  "x-flowintel-plan",
] as const;
const bearerAuthorizationHeaderName = "authorization";
const clientForwardedAuthHeadersStorageKey =
  "flowintel.forwarded-auth-headers";

export type ForwardedAuthHeaderInput = {
  bearerToken: string | undefined;
  userId: string | undefined;
  sessionId: string | undefined;
  role: string | undefined;
  plan: string | undefined;
};

export function createForwardedAuthHeaders({
  bearerToken,
  userId,
  sessionId,
  role,
  plan,
}: ForwardedAuthHeaderInput): HeadersInit | undefined {
  const nextHeaders = new Headers();

  if (bearerToken?.trim()) {
    nextHeaders.set(
      bearerAuthorizationHeaderName,
      `Bearer ${bearerToken.trim()}`,
    );
  }
  if (userId?.trim()) {
    nextHeaders.set("x-clerk-user-id", userId.trim());
  }
  if (sessionId?.trim()) {
    nextHeaders.set("x-clerk-session-id", sessionId.trim());
  }
  if (role?.trim()) {
    nextHeaders.set("x-clerk-role", role.trim());
  }
  if (plan?.trim()) {
    nextHeaders.set("x-flowintel-plan", plan.trim());
  }

  return Array.from(nextHeaders.keys()).length > 0 ? nextHeaders : undefined;
}

export function buildForwardedAuthHeaders(
  requestHeaders: Pick<Headers, "get">,
): HeadersInit | undefined {
  return createForwardedAuthHeaders({
    bearerToken: normalizeBearerToken(
      requestHeaders.get(bearerAuthorizationHeaderName),
    ),
    userId: requestHeaders.get("x-clerk-user-id") ?? undefined,
    sessionId: requestHeaders.get("x-clerk-session-id") ?? undefined,
    role: requestHeaders.get("x-clerk-role") ?? undefined,
    plan: requestHeaders.get("x-flowintel-plan") ?? undefined,
  });
}

export function mergeRequestHeaders(
  baseHeaders: HeadersInit,
  requestHeaders?: HeadersInit,
): Headers {
  const mergedHeaders = new Headers(baseHeaders);
  if (!requestHeaders) {
    return mergedHeaders;
  }

  const nextHeaders = new Headers(requestHeaders);
  nextHeaders.forEach((value, key) => {
    mergedHeaders.set(key, value);
  });
  return mergedHeaders;
}

export function persistClientForwardedAuthHeaders(
  requestHeaders?: HeadersInit,
): void {
  if (typeof window === "undefined") {
    return;
  }

  if (!requestHeaders) {
    window.sessionStorage.removeItem(clientForwardedAuthHeadersStorageKey);
    return;
  }

  const nextHeaders = new Headers(requestHeaders);
  const persistedEntries = Array.from(nextHeaders.entries()).filter(
    ([key, value]) =>
      forwardedAuthHeaderNames.includes(
        key as (typeof forwardedAuthHeaderNames)[number],
      ) && value.trim() !== "",
  );

  if (persistedEntries.length === 0) {
    window.sessionStorage.removeItem(clientForwardedAuthHeadersStorageKey);
    return;
  }

  window.sessionStorage.setItem(
    clientForwardedAuthHeadersStorageKey,
    JSON.stringify(persistedEntries),
  );
}

export function readClientForwardedAuthHeaders(): HeadersInit | undefined {
  if (typeof window === "undefined") {
    return undefined;
  }

  const persistedValue = window.sessionStorage.getItem(
    clientForwardedAuthHeadersStorageKey,
  );
  if (!persistedValue) {
    return undefined;
  }

  try {
    const parsed = JSON.parse(persistedValue) as Array<[string, string]>;
    if (!Array.isArray(parsed) || parsed.length === 0) {
      return undefined;
    }
    return new Headers(parsed);
  } catch {
    return undefined;
  }
}

function normalizeBearerToken(
  authorizationHeader: string | null,
): string | undefined {
  const value = authorizationHeader?.trim();
  if (!value) {
    return undefined;
  }

  if (value.toLowerCase().startsWith("bearer ")) {
    const token = value.slice("bearer ".length).trim();
    return token || undefined;
  }

  return value;
}
