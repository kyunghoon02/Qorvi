import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  fullyParallel: false,
  retries: 0,
  use: {
    baseURL: "http://127.0.0.1:3000",
    trace: "retain-on-failure",
  },
  webServer: {
    command: "cd ../.. && corepack pnpm dev:stack:no-worker",
    url: "http://127.0.0.1:3000",
    timeout: 180_000,
    reuseExistingServer: true,
  },
});
