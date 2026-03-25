import assert from "node:assert/strict";
import test from "node:test";

import {
  buildForwardedAuthHeaders,
  createForwardedAuthHeaders,
} from "../lib/request-headers";

test("createForwardedAuthHeaders builds bearer and clerk identity headers", () => {
  const headers = new Headers(
    createForwardedAuthHeaders({
      bearerToken: "token_123",
      userId: "user_123",
      sessionId: "sess_123",
      role: "operator",
      plan: "team",
    }),
  );

  assert.equal(headers.get("authorization"), "Bearer token_123");
  assert.equal(headers.get("x-clerk-user-id"), "user_123");
  assert.equal(headers.get("x-clerk-session-id"), "sess_123");
  assert.equal(headers.get("x-clerk-role"), "operator");
  assert.equal(headers.get("x-flowintel-plan"), "team");
});

test("buildForwardedAuthHeaders normalizes incoming bearer headers", () => {
  const headers = new Headers(
    buildForwardedAuthHeaders({
      get(name) {
        if (name === "authorization") {
          return "Bearer token_abc";
        }
        if (name === "x-clerk-user-id") {
          return "user_abc";
        }
        return null;
      },
    }),
  );

  assert.equal(headers.get("authorization"), "Bearer token_abc");
  assert.equal(headers.get("x-clerk-user-id"), "user_abc");
});
