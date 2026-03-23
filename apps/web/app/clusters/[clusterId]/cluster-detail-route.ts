export type ClusterDetailRequest = {
  clusterId: string;
};

export function resolveClusterDetailRequestFromParams(
  clusterId: string,
): ClusterDetailRequest | null {
  const normalized = decodeURIComponent(clusterId).trim();
  if (!normalized) {
    return null;
  }

  return {
    clusterId: normalized,
  };
}
