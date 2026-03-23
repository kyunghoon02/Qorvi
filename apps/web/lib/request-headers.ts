const forwardedAuthHeaderNames = [
  "x-clerk-user-id",
  "x-clerk-session-id",
  "x-clerk-role",
  "x-whalegraph-plan",
] as const;
const clientForwardedAuthHeadersStorageKey =
  "whalegraph.forwarded-auth-headers";

export function buildForwardedAuthHeaders(
  requestHeaders: Pick<Headers, "get">,
): HeadersInit | undefined {
  const nextHeaders = new Headers();

  for (const headerName of forwardedAuthHeaderNames) {
    const value = requestHeaders.get(headerName)?.trim();
    if (!value) {
      continue;
    }

    nextHeaders.set(headerName, value);
  }

  return Array.from(nextHeaders.keys()).length > 0 ? nextHeaders : undefined;
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
    ([, value]) => value.trim() !== "",
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
