import assert from "node:assert/strict";
import test from "node:test";

import { resolveClerkRole } from "../lib/clerk-role";

test("resolveClerkRole prefers direct role claims", () => {
  assert.equal(resolveClerkRole({ rol: "admin" }), "admin");
  assert.equal(resolveClerkRole({ org_role: "operator" }), "operator");
});

test("resolveClerkRole falls back to metadata role", () => {
  assert.equal(
    resolveClerkRole({
      public_metadata: {
        role: "user",
      },
    }),
    "user",
  );
});
