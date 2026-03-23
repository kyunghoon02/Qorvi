import assert from "node:assert/strict";
import test from "node:test";

import {
  getClusterDetailPreview,
} from "../lib/api-boundary";
import {
  resolveClusterDetailRequestFromParams,
} from "../app/clusters/[clusterId]/cluster-detail-route";
import {
  buildClusterDetailViewModel,
} from "../app/clusters/[clusterId]/cluster-detail-screen";

test("resolveClusterDetailRequestFromParams validates cluster route params", () => {
  assert.deepEqual(resolveClusterDetailRequestFromParams("cluster_seed_whales"), {
    clusterId: "cluster_seed_whales",
  });
  assert.deepEqual(resolveClusterDetailRequestFromParams("cluster%20live"), {
    clusterId: "cluster live",
  });
  assert.equal(resolveClusterDetailRequestFromParams(""), null);
});

test("buildClusterDetailViewModel carries cluster copy and evidence", () => {
  const request = {
    clusterId: "cluster_seed_whales",
  };

  const viewModel = buildClusterDetailViewModel({
    cluster: getClusterDetailPreview(request),
  });

  assert.equal(viewModel.title, "cluster_seed_whales");
  assert.equal(viewModel.clusterTypeLabel, "unavailable");
  assert.equal(viewModel.classificationLabel, "Emerging cluster");
  assert.equal(viewModel.scoreLabel, "0");
  assert.equal(viewModel.memberCount, 0);
  assert.equal(viewModel.members.length, 0);
  assert.equal(viewModel.commonActions.length, 0);
  assert.equal(viewModel.evidence.length, 0);
  assert.equal(viewModel.backHref, "/");
});
