export function resolveClerkRole(sessionClaims: unknown): string | undefined {
  if (!sessionClaims || typeof sessionClaims !== "object") {
    return undefined;
  }

  const claims = sessionClaims as Record<string, unknown>;
  const directRole = normalizeRole(
    firstNonEmptyString([
      claims.rol,
      claims.role,
      claims.org_role,
      claims.orgRole,
    ]),
  );
  if (directRole) {
    return directRole;
  }

  for (const key of [
    "public_metadata",
    "publicMetadata",
    "unsafe_metadata",
    "unsafeMetadata",
  ]) {
    const metadataRole = normalizeRole(readNestedRole(claims[key]));
    if (metadataRole) {
      return metadataRole;
    }
  }

  return undefined;
}

function normalizeRole(value: string | undefined): string | undefined {
  if (!value) {
    return undefined;
  }

  const normalized = value.trim().toLowerCase().replace(/^org:/, "");
  if (
    normalized === "admin" ||
    normalized === "operator" ||
    normalized === "user"
  ) {
    return normalized;
  }
  return undefined;
}

function readNestedRole(value: unknown): string | undefined {
  if (!value || typeof value !== "object") {
    return undefined;
  }

  const candidate = firstNonEmptyString([
    (value as Record<string, unknown>).role,
    (value as Record<string, unknown>).rol,
  ]);
  return candidate?.trim();
}

function firstNonEmptyString(values: unknown[]): string | undefined {
  for (const value of values) {
    if (typeof value === "string" && value.trim()) {
      return value.trim();
    }
  }

  return undefined;
}
