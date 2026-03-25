import {
  type APIRequestContext,
  type Page,
  expect,
  test,
} from "@playwright/test";

const apiBaseUrl = "http://127.0.0.1:4000";
const seededWalletAddress = "0x1234567890abcdef1234567890abcdef12345678";
const authHeaders = {
  "X-Clerk-User-Id": "user_123",
  "X-Clerk-Session-Id": "session_123",
  "X-Clerk-Role": "user",
};
const persistedAuthHeaderStorageKey = "flowintel.forwarded-auth-headers";

async function seedBrowserAuth(page: Page) {
  await page.addInitScript((headers) => {
    const normalizedEntries = Object.entries(headers).map(([key, value]) => [
      key.toLowerCase(),
      value,
    ]);
    window.sessionStorage.setItem(
      "flowintel.forwarded-auth-headers",
      JSON.stringify(normalizedEntries),
    );

    const originalFetch = window.fetch.bind(window);
    window.fetch = (input, init) => {
      const nextHeaders = new Headers(init?.headers ?? {});
      for (const [key, value] of normalizedEntries) {
        if (!nextHeaders.has(key)) {
          nextHeaders.set(key, value);
        }
      }

      return originalFetch(input, {
        ...init,
        headers: nextHeaders,
      });
    };
  }, authHeaders);
}

async function reconcileTeamPlan({
  request,
  subscriptionId,
  customerId,
}: {
  request: APIRequestContext;
  subscriptionId: string;
  customerId: string;
}) {
  const response = await request.post(
    `${apiBaseUrl}/v1/webhooks/billing/stripe`,
    {
      data: {
        type: "checkout.session.completed",
        subscriptionId,
        customerId,
        principalUserId: authHeaders["X-Clerk-User-Id"],
        planTier: "team",
        status: "active",
      },
    },
  );

  await expect(response).toBeOK();
}

async function trackWalletViaApi({
  request,
  chain,
  address,
  label,
}: {
  request: APIRequestContext;
  chain: "evm" | "solana";
  address: string;
  label: string;
}) {
  const listWatchlistsResponse = await request.get(
    `${apiBaseUrl}/v1/watchlists`,
    {
      headers: authHeaders,
    },
  );
  await expect(listWatchlistsResponse).toBeOK();
  const listWatchlistsPayload = await listWatchlistsResponse.json();
  let watchlist =
    listWatchlistsPayload.data.items.find(
      (item: { name: string }) => item.name === "Tracked wallets",
    ) ?? null;

  if (!watchlist) {
    const createWatchlistResponse = await request.post(
      `${apiBaseUrl}/v1/watchlists`,
      {
        headers: {
          ...authHeaders,
          "Content-Type": "application/json",
        },
        data: { name: "Tracked wallets" },
      },
    );
    await expect(createWatchlistResponse).toBeOK();
    const createWatchlistPayload = await createWatchlistResponse.json();
    watchlist = createWatchlistPayload.data;
  }

  const watchlistDetailResponse = await request.get(
    `${apiBaseUrl}/v1/watchlists/${watchlist.id}`,
    {
      headers: authHeaders,
    },
  );
  await expect(watchlistDetailResponse).toBeOK();
  const watchlistDetailPayload = await watchlistDetailResponse.json();
  const alreadyTracked = watchlistDetailPayload.data.items.some(
    (item: { chain: string; address: string }) =>
      item.chain.toLowerCase() === chain &&
      item.address.toLowerCase() === address.toLowerCase(),
  );

  if (!alreadyTracked) {
    const addItemResponse = await request.post(
      `${apiBaseUrl}/v1/watchlists/${watchlist.id}/items`,
      {
        headers: {
          ...authHeaders,
          "Content-Type": "application/json",
        },
        data: {
          chain,
          address,
          tags: ["tracked-wallet"],
          note: "Added from wallet detail.",
        },
      },
    );
    expect([201, 409]).toContain(addItemResponse.status());
  }

  const listRulesResponse = await request.get(`${apiBaseUrl}/v1/alert-rules`, {
    headers: authHeaders,
  });
  await expect(listRulesResponse).toBeOK();
  const listRulesPayload = await listRulesResponse.json();
  let rule =
    listRulesPayload.data.items.find(
      (item: {
        definition?: { watchlistId?: string; signalTypes?: string[] };
      }) =>
        item.definition?.watchlistId === watchlist.id &&
        (item.definition?.signalTypes ?? []).sort().join("|") ===
          ["cluster_score", "shadow_exit", "first_connection"].sort().join("|"),
    ) ?? null;

  if (!rule) {
    const createRuleResponse = await request.post(
      `${apiBaseUrl}/v1/alert-rules`,
      {
        headers: {
          ...authHeaders,
          "Content-Type": "application/json",
        },
        data: {
          name: `${label} signal watch`,
          ruleType: "watchlist_signal",
          isEnabled: true,
          cooldownSeconds: 3600,
          definition: {
            watchlistId: watchlist.id,
            signalTypes: ["cluster_score", "shadow_exit", "first_connection"],
            minimumSeverity: "medium",
            renotifyOnSeverityIncrease: true,
          },
          notes: "Created from wallet detail tracking.",
          tags: ["tracked-wallet"],
        },
      },
    );
    await expect(createRuleResponse).toBeOK();
    const createRulePayload = await createRuleResponse.json();
    rule = createRulePayload.data;
  }

  const params = new URLSearchParams({
    tracked: "success",
    watchlistId: watchlist.id,
    ruleId: rule.id,
    wallet: address,
  });

  return `/alerts?${params.toString()}`;
}

test.describe("WG-042 beta flow", () => {
  test.use({
    extraHTTPHeaders: authHeaders,
  });

  test("searches a wallet and lands on tracked alerts", async ({
    page,
    request,
  }) => {
    await seedBrowserAuth(page);
    await reconcileTeamPlan({
      request,
      subscriptionId: "sub_e2e_track_wallet",
      customerId: "cus_e2e_track_wallet",
    });

    await page.goto("/");

    const searchInput = page.getByPlaceholder("EVM or Solana address").first();
    await searchInput.fill(seededWalletAddress);
    await page.getByRole("button", { name: "Search" }).first().click();

    await expect(page.getByRole("link", { name: "Open detail" })).toBeVisible();
    await page.getByRole("link", { name: "Open detail" }).click();

    await expect(page).toHaveURL(
      new RegExp(`/wallets/evm/${seededWalletAddress}$`),
    );
    await expect(
      page.getByRole("button", { name: "Track wallet" }),
    ).toBeVisible();

    const alertsHref = await trackWalletViaApi({
      request,
      chain: "evm",
      address: seededWalletAddress,
      label: "Seed Whale",
    });
    await page.goto(alertsHref);

    await expect(page).toHaveURL(/\/alerts\?tracked=success/);
    await expect(page.getByText("Wallet tracking active")).toBeVisible();
    await expect(page.getByText(/Watchlist .* and rule /i)).toBeVisible();
  });

  test("creates checkout intent, reconciles billing, and shows upgraded account", async ({
    page,
    request,
  }) => {
    await page.addInitScript((storageKey) => {
      window.sessionStorage.removeItem(storageKey);
    }, persistedAuthHeaderStorageKey);
    const checkoutResponse = await request.post(
      `${apiBaseUrl}/v1/billing/checkout-sessions`,
      {
        headers: {
          ...authHeaders,
          "Content-Type": "application/json",
        },
        data: {
          tier: "team",
          successUrl:
            "http://127.0.0.1:3000/account?checkout=success&plan=team",
          cancelUrl: "http://127.0.0.1:3000/account?checkout=cancel&plan=team",
        },
      },
    );
    await expect(checkoutResponse).toBeOK();

    const checkoutPayload = await checkoutResponse.json();
    expect(checkoutPayload.success).toBeTruthy();
    expect(checkoutPayload.data.checkoutSession.provider).toBe("stripe");
    expect(checkoutPayload.data.plan.tier).toBe("team");

    await reconcileTeamPlan({
      request,
      subscriptionId: "sub_e2e_account_plan",
      customerId: "cus_e2e_account_plan",
    });

    await page.goto("/account");

    await expect(
      page.getByRole("heading", { name: "Account & billing" }),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: /Current plan: Team/i }),
    ).toBeVisible();
  });
});
