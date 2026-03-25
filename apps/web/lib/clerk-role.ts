export function resolveClerkRole(sessionClaims: unknown): string | undefined {
  if (!sessionClaims || typeof sessionClaims !== "object") {
    return undefined;
  }

  const claims = sessionClaims as Record<string, unknown>;
  const directRole = firstNonEmptyString([
    claims.rol,
    claims.role,
    claims.org_role,
  ]);
  if (directRole) {
    return directRole;
  }

  const metadataRole = readNestedRole(claims.public_metadata);
  if (metadataRole) {
    return metadataRole;
  }

  return readNestedRole(claims.unsafeMetadata);
}

function readNestedRole(value: unknown): string | undefined {
  if (!value || typeof value !== "object") {
    return undefined;
  }

  const candidate = (value as Record<string, unknown>).role;
  return typeof candidate === "string" && candidate.trim()
    ? candidate.trim()
    : undefined;
}

function firstNonEmptyString(values: unknown[]): string | undefined {
  for (const value of values) {
    if (typeof value === "string" && value.trim()) {
      return value.trim();
    }
  }

  return undefined;
}
