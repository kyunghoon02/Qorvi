package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/auth"
	"github.com/whalegraph/whalegraph/apps/api/internal/service"
	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestHealthRoute(t *testing.T) {
	t.Parallel()

	srv := New()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[HealthPayload]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatalf("expected success response")
	}

	if body.Data.Status != "ok" || body.Data.Service != "api" {
		t.Fatalf("unexpected health payload: %+v", body.Data)
	}
}

func TestWalletSummaryRoute(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		Wallets:       service.NewWalletSummaryService(&testWalletSummaryRepository{summary: walletSummaryFixture()}, nil),
		Graphs:        service.NewWalletGraphService(&testWalletGraphRepository{graph: walletGraphFixture()}),
		Clusters:      service.NewClusterDetailService(&testClusterDetailRepository{detail: clusterDetailFixture()}),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/summary", nil)
	req.Header.Set("X-Whalegraph-Plan", "pro")

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[service.WalletSummary]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatalf("expected success response")
	}

	if body.Data.Chain != "evm" {
		t.Fatalf("expected chain evm, got %s", body.Data.Chain)
	}

	if body.Data.Address != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected address: %s", body.Data.Address)
	}

	if body.Meta.Tier != "pro" {
		t.Fatalf("expected pro tier, got %s", body.Meta.Tier)
	}

	if len(body.Data.Scores) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(body.Data.Scores))
	}
	if body.Data.Scores[0].Name != "cluster_score" {
		t.Fatalf("expected cluster score, got %s", body.Data.Scores[0].Name)
	}
}

func TestWalletSummaryRouteRejectsUnsupportedChain(t *testing.T) {
	t.Parallel()

	srv := New()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/bitcoin/0x1234567890abcdef1234567890abcdef12345678/summary", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var body Envelope[any]
	decode(t, rr.Body.Bytes(), &body)

	if body.Error == nil || body.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("expected invalid argument error, got %#v", body.Error)
	}
}

func TestBillingCheckoutSessionRouteCreatesPlaceholderSession(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{ClerkVerifier: auth.NewHeaderClerkVerifier()})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/billing/checkout-sessions", bytes.NewReader([]byte(`{
		"tier":"pro",
		"successUrl":"http://localhost:3000/account?checkout=success",
		"cancelUrl":"http://localhost:3000/account?checkout=cancel",
		"customerEmail":"ops@whalegraph.test"
	}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}

	var body Envelope[service.BillingCheckoutResponse]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatal("expected success response")
	}
	if body.Data.CheckoutSession.Provider != "stripe" {
		t.Fatalf("expected stripe session, got %q", body.Data.CheckoutSession.Provider)
	}
	if body.Data.Plan.Tier != "pro" {
		t.Fatalf("expected pro plan, got %q", body.Data.Plan.Tier)
	}
}

func TestBillingPlansRouteReturnsCatalog(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{ClerkVerifier: auth.NewHeaderClerkVerifier()})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/billing/plans", nil)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[service.BillingPlansResponse]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatal("expected success response")
	}
	if len(body.Data.Plans) < 3 {
		t.Fatalf("expected at least 3 plans, got %d", len(body.Data.Plans))
	}
	if body.Data.Plans[0].CheckoutSessionPath != "/v1/billing/checkout-sessions" {
		t.Fatalf("unexpected checkout path %q", body.Data.Plans[0].CheckoutSessionPath)
	}
}

func TestStripeBillingWebhookReconcilesAccountPlan(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{ClerkVerifier: auth.NewHeaderClerkVerifier()})

	webhookRR := httptest.NewRecorder()
	webhookReq := httptest.NewRequest(http.MethodPost, "/v1/webhooks/billing/stripe", bytes.NewReader([]byte(`{
		"type":"checkout.session.completed",
		"subscriptionId":"sub_123",
		"customerId":"cus_456",
		"principalUserId":"user_123",
		"planTier":"team",
		"status":"active"
	}`)))
	webhookReq.Header.Set("Content-Type", "application/json")

	srv.Handler().ServeHTTP(webhookRR, webhookReq)

	if webhookRR.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", webhookRR.Code)
	}

	accountRR := httptest.NewRecorder()
	accountReq := httptest.NewRequest(http.MethodGet, "/v1/account", nil)
	accountReq.Header.Set("X-Clerk-User-Id", "user_123")
	accountReq.Header.Set("X-Clerk-Session-Id", "session_123")
	accountReq.Header.Set("X-Clerk-Role", "user")
	accountReq.Header.Set("X-Whalegraph-Plan", "free")

	srv.Handler().ServeHTTP(accountRR, accountReq)

	if accountRR.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", accountRR.Code)
	}

	var accountBody Envelope[service.AccountResponse]
	decode(t, accountRR.Body.Bytes(), &accountBody)

	if accountBody.Data.Plan.Tier != "team" {
		t.Fatalf("expected reconciled team plan, got %q", accountBody.Data.Plan.Tier)
	}
	if accountBody.Data.Access.Plan != "team" {
		t.Fatalf("expected access plan team, got %q", accountBody.Data.Access.Plan)
	}
}

func TestWalletGraphRouteDefaultsToOneHop(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		Graphs:        service.NewWalletGraphService(&testWalletGraphRepository{graph: walletGraphFixture()}),
		Clusters:      service.NewClusterDetailService(&testClusterDetailRepository{detail: clusterDetailFixture()}),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/graph", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[domain.WalletGraph]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatalf("expected success response")
	}
	if body.Data.DepthRequested != 1 {
		t.Fatalf("expected default depth 1, got %d", body.Data.DepthRequested)
	}
	if body.Data.DepthResolved != 1 {
		t.Fatalf("expected resolved depth 1, got %d", body.Data.DepthResolved)
	}
	if body.Data.NeighborhoodSummary == nil {
		t.Fatal("expected neighborhood summary")
	}
}

func TestWalletGraphRouteBlocksFreeTierTwoHop(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		Graphs:        service.NewWalletGraphService(&testWalletGraphRepository{graph: walletGraphFixture()}),
		Clusters:      service.NewClusterDetailService(&testClusterDetailRepository{detail: clusterDetailFixture()}),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/graph?depth=2", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rr.Code)
	}

	var body Envelope[any]
	decode(t, rr.Body.Bytes(), &body)

	if body.Error == nil || body.Error.Code != "FORBIDDEN" {
		t.Fatalf("expected forbidden error, got %#v", body.Error)
	}
}

func TestWalletGraphRouteAllowsProTwoHopRequest(t *testing.T) {
	t.Parallel()

	repo := &testWalletGraphRepository{graph: walletGraphFixture()}
	srv := NewWithDependencies(Dependencies{
		Graphs:        service.NewWalletGraphService(repo),
		Clusters:      service.NewClusterDetailService(&testClusterDetailRepository{detail: clusterDetailFixture()}),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/graph?depth=2", nil)
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[domain.WalletGraph]
	decode(t, rr.Body.Bytes(), &body)

	if body.Data.DepthRequested != 2 {
		t.Fatalf("expected requested depth 2, got %d", body.Data.DepthRequested)
	}
	if !body.Data.DensityCapped {
		t.Fatal("expected density cap flag for the placeholder 1-hop response")
	}
	if len(body.Data.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(body.Data.Nodes))
	}
	if body.Data.Nodes[0].Kind != domain.WalletGraphNodeWallet || body.Data.Nodes[0].Label != "Seed Whale" {
		t.Fatalf("unexpected root node: %#v", body.Data.Nodes[0])
	}
	if body.Data.Nodes[1].Kind != domain.WalletGraphNodeCluster {
		t.Fatalf("expected cluster node second, got %#v", body.Data.Nodes[1])
	}
	if len(body.Data.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(body.Data.Edges))
	}
	if body.Data.NeighborhoodSummary == nil {
		t.Fatal("expected neighborhood summary")
	}
	if body.Data.NeighborhoodSummary.ClusterNodeCount != 1 {
		t.Fatalf("unexpected neighborhood summary %#v", body.Data.NeighborhoodSummary)
	}
	if body.Data.Edges[0].Kind != domain.WalletGraphEdgeMemberOf || body.Data.Edges[0].Weight != 82 {
		t.Fatalf("unexpected structural edge: %#v", body.Data.Edges[0])
	}
	if body.Data.Edges[1].Kind != domain.WalletGraphEdgeFundedBy {
		t.Fatalf("expected funded-by edge second, got %#v", body.Data.Edges[1])
	}
	if !repo.called {
		t.Fatal("expected graph repository to be invoked")
	}
}

func TestShadowExitFeedRoute(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		ShadowExits:   service.NewShadowExitFeedService(&testShadowExitFeedRepository{page: shadowExitFeedFixture()}),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/signals/shadow-exits?limit=10", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[service.ShadowExitFeedResponse]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatal("expected success response")
	}
	if len(body.Data.Items) != 1 {
		t.Fatalf("expected one shadow exit item, got %d", len(body.Data.Items))
	}
	if body.Data.WindowLabel != "Last 24 hours" {
		t.Fatalf("unexpected window label %q", body.Data.WindowLabel)
	}
	if body.Data.Items[0].WalletRoute != "/wallets/solana/So11111111111111111111111111111111111111112" {
		t.Fatalf("unexpected wallet route %q", body.Data.Items[0].WalletRoute)
	}
	if body.Data.Items[0].Explanation == "" {
		t.Fatal("expected explanation")
	}
	if body.Data.Items[0].Score != 34 {
		t.Fatalf("unexpected score %d", body.Data.Items[0].Score)
	}
	if body.Data.Items[0].Rating != "medium" {
		t.Fatalf("unexpected rating %q", body.Data.Items[0].Rating)
	}
}

func TestFirstConnectionFeedRoute(t *testing.T) {
	t.Parallel()

	repo := &testFirstConnectionFeedRepository{page: firstConnectionFeedFixture()}
	srv := NewWithDependencies(Dependencies{
		FirstConnections: service.NewFirstConnectionFeedService(repo),
		ClerkVerifier:    auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/signals/first-connections?limit=10", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[service.FirstConnectionFeedResponse]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatal("expected success response")
	}
	if len(body.Data.Items) != 1 {
		t.Fatalf("expected one first connection item, got %d", len(body.Data.Items))
	}
	if body.Data.WindowLabel != "Hot feed baseline" {
		t.Fatalf("unexpected window label %q", body.Data.WindowLabel)
	}
	if body.Data.Items[0].WalletRoute != "/wallets/evm/0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected wallet route %q", body.Data.Items[0].WalletRoute)
	}
	if body.Data.Items[0].Explanation == "" {
		t.Fatal("expected explanation")
	}
	if body.Data.Items[0].Score != 72 {
		t.Fatalf("unexpected score %d", body.Data.Items[0].Score)
	}
	if body.Data.Items[0].Rating != "high" {
		t.Fatalf("unexpected rating %q", body.Data.Items[0].Rating)
	}
	if repo.sort != "latest" {
		t.Fatalf("unexpected sort %q", repo.sort)
	}
}

func TestFirstConnectionFeedRouteSortScore(t *testing.T) {
	t.Parallel()

	repo := &testFirstConnectionFeedRepository{page: firstConnectionFeedFixture()}
	srv := NewWithDependencies(Dependencies{
		FirstConnections: service.NewFirstConnectionFeedService(repo),
		ClerkVerifier:    auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/signals/first-connections?limit=10&sort=score", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[service.FirstConnectionFeedResponse]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatal("expected success response")
	}
	if len(body.Data.Items) != 1 {
		t.Fatalf("expected one first connection item, got %d", len(body.Data.Items))
	}
	if body.Data.Items[0].Score != 72 {
		t.Fatalf("unexpected score %d", body.Data.Items[0].Score)
	}
	if repo.sort != "score" {
		t.Fatalf("unexpected sort %q", repo.sort)
	}
}

func TestFirstConnectionFeedRouteRejectsInvalidSort(t *testing.T) {
	t.Parallel()

	repo := &testFirstConnectionFeedRepository{page: firstConnectionFeedFixture()}
	srv := NewWithDependencies(Dependencies{
		FirstConnections: service.NewFirstConnectionFeedService(repo),
		ClerkVerifier:    auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/signals/first-connections?sort=unknown", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestSearchRouteClassifiesWalletLikeInput(t *testing.T) {
	t.Parallel()

	srv := New()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=0x1234567890abcdef1234567890abcdef12345678", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[service.SearchResponse]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatal("expected success response")
	}
	if body.Data.InputKind != "evm_address" {
		t.Fatalf("expected evm address kind, got %s", body.Data.InputKind)
	}
	if len(body.Data.Results) != 1 {
		t.Fatalf("expected one search result, got %d", len(body.Data.Results))
	}

	result := body.Data.Results[0]
	if result.WalletRoute != "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/summary" {
		t.Fatalf("unexpected wallet route: %s", result.WalletRoute)
	}
	if !result.Navigation {
		t.Fatal("expected navigation to be enabled")
	}
}

func TestClusterDetailRoute(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		Clusters:      service.NewClusterDetailService(&testClusterDetailRepository{detail: clusterDetailFixture()}),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/clusters/cluster_seed_whales", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[service.ClusterDetail]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatal("expected success response")
	}
	if body.Data.ID != "cluster_seed_whales" {
		t.Fatalf("unexpected cluster id %q", body.Data.ID)
	}
	if len(body.Data.Members) != 2 || len(body.Data.CommonActions) != 1 {
		t.Fatalf("unexpected cluster detail %#v", body.Data)
	}
}

func TestClusterDetailRouteReturnsNotFound(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		Clusters:      service.NewClusterDetailService(&testClusterDetailRepository{err: service.ErrClusterDetailNotFound}),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/clusters/cluster_missing", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestSearchRouteEnrichesWalletLikeInputFromLookup(t *testing.T) {
	t.Parallel()

	walletService := service.NewWalletSummaryService(&testWalletSummaryRepository{summary: walletSummaryFixture()}, nil)
	srv := NewWithDependencies(Dependencies{
		Search:        service.NewSearchService(walletService),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=0x1234567890abcdef1234567890abcdef12345678", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[service.SearchResponse]
	decode(t, rr.Body.Bytes(), &body)

	if body.Data.Results[0].Label != "Seed Whale" {
		t.Fatalf("expected enriched label, got %s", body.Data.Results[0].Label)
	}
	if body.Data.Results[0].Chain != "evm" {
		t.Fatalf("expected enriched chain, got %s", body.Data.Results[0].Chain)
	}
	if body.Data.Results[0].ChainLabel != "EVM" {
		t.Fatalf("expected enriched chain label, got %s", body.Data.Results[0].ChainLabel)
	}
	if body.Data.Explanation != "Found wallet summary for Seed Whale." {
		t.Fatalf("unexpected explanation: %s", body.Data.Explanation)
	}
}

func TestSearchRouteQueuesManualRefreshWhenRequested(t *testing.T) {
	t.Parallel()

	lookup := &fakeSearchWalletSummaryLookup{
		summary: service.WalletSummary{
			Chain:       "evm",
			Address:     "0x1234567890abcdef1234567890abcdef12345678",
			DisplayName: "Indexed Whale",
			Indexing: service.WalletIndexingState{
				Status:        "ready",
				LastIndexedAt: "2026-03-22T01:00:00Z",
			},
		},
	}
	queue := &fakeSearchWalletBackfillQueueStore{}
	searchService := service.NewSearchServiceWithBackfillQueue(lookup, queue)
	searchService.Now = func() time.Time {
		return time.Date(2026, time.March, 22, 4, 5, 6, 0, time.UTC)
	}

	srv := NewWithDependencies(Dependencies{
		Search:        searchService,
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/search?q=0x1234567890abcdef1234567890abcdef12345678&refresh=manual",
		nil,
	)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("expected 1 queued manual refresh job, got %d", len(queue.jobs))
	}
	if queue.jobs[0].Source != "search_manual_refresh" {
		t.Fatalf("unexpected queued source %q", queue.jobs[0].Source)
	}
}

func TestParseWalletRoutePath(t *testing.T) {
	t.Parallel()

	chain, address, resource, ok := parseWalletRoutePath("/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/summary")
	if !ok {
		t.Fatal("expected valid wallet route path")
	}
	if chain != "evm" {
		t.Fatalf("expected evm chain, got %s", chain)
	}
	if address != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected address %s", address)
	}
	if resource != "summary" {
		t.Fatalf("expected summary resource, got %s", resource)
	}

	if _, _, _, ok := parseWalletRoutePath("/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/profile"); ok {
		t.Fatal("expected unsupported resource to be rejected")
	}
}

func TestSearchRouteExplainsUnknownInput(t *testing.T) {
	t.Parallel()

	srv := New()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=not-a-wallet", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body Envelope[service.SearchResponse]
	decode(t, rr.Body.Bytes(), &body)

	if body.Data.InputKind != "unknown" {
		t.Fatalf("expected unknown input kind, got %s", body.Data.InputKind)
	}
	if body.Data.Results[0].WalletRoute != "" {
		t.Fatalf("expected no wallet route, got %s", body.Data.Results[0].WalletRoute)
	}
	if body.Data.Results[0].Explanation == "" {
		t.Fatal("expected explanation for unknown input")
	}
}

func TestSearchRouteRejectsEmptyQuery(t *testing.T) {
	t.Parallel()

	srv := New()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/search", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var body Envelope[any]
	decode(t, rr.Body.Bytes(), &body)

	if body.Error == nil || body.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("expected invalid argument error, got %#v", body.Error)
	}
}

func TestAdminStatusRouteRequiresRole(t *testing.T) {
	t.Parallel()

	srv := New()

	forbidden := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)
	srv.Handler().ServeHTTP(forbidden, req)

	if forbidden.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", forbidden.Code)
	}

	forbiddenRole := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	srv.Handler().ServeHTTP(forbiddenRole, req)

	if forbiddenRole.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", forbiddenRole.Code)
	}

	allowed := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/admin/status", nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "admin")
	srv.Handler().ServeHTTP(allowed, req)

	if allowed.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", allowed.Code)
	}

	var body Envelope[AdminStatusPayload]
	decode(t, allowed.Body.Bytes(), &body)

	if !body.Success {
		t.Fatalf("expected success response")
	}

	if body.Data.Scope != "admin" {
		t.Fatalf("expected admin scope, got %s", body.Data.Scope)
	}
}

func TestAlchemyWebhookRouteAcceptsAddressActivityPayload(t *testing.T) {
	t.Parallel()

	ingest := &fakeWebhookIngestService{
		alchemyResult: WebhookIngestResult{AcceptedCount: 2, EventKind: "address_activity"},
	}
	srv := NewWithDependencies(Dependencies{WebhookIngest: ingest})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/webhooks/providers/alchemy/address-activity",
		bytes.NewBufferString(`{"webhookId":"wh_test","id":"evt_test","createdAt":"2026-03-20T00:00:00Z","type":"ADDRESS_ACTIVITY","event":{"network":"ETH_MAINNET","activity":[{"blockNum":"0xdf34a3","hash":"0xabc","fromAddress":"0x503828976d22510aad0201ac7ec88293211d23da","toAddress":"0xbe3f4b43db5eb49d1f48f53443b9abce45da3b79","category":"token","asset":"USDC"},{"blockNum":"0xdf34a4","hash":"0xdef","fromAddress":"0x503828976d22510aad0201ac7ec88293211d23da","toAddress":"0xbe3f4b43db5eb49d1f48f53443b9abce45da3b79","category":"token","asset":"USDC"}]}}`),
	)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}

	var body Envelope[ProviderWebhookAcceptancePayload]
	decode(t, rr.Body.Bytes(), &body)

	if !body.Success {
		t.Fatal("expected success response")
	}
	if body.Data.Provider != "alchemy" {
		t.Fatalf("unexpected provider %q", body.Data.Provider)
	}
	if body.Data.AcceptedCount != 2 {
		t.Fatalf("expected accepted count 2, got %d", body.Data.AcceptedCount)
	}
	if !ingest.alchemyCalled {
		t.Fatal("expected alchemy ingest service to be called")
	}
}

func TestHeliusWebhookRouteAcceptsBatchPayload(t *testing.T) {
	t.Parallel()

	ingest := &fakeWebhookIngestService{
		providerResult: WebhookIngestResult{AcceptedCount: 3, EventKind: "webhook_batch"},
	}
	srv := NewWithDependencies(Dependencies{WebhookIngest: ingest})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/providers/helius/webhooks/address-activity",
		bytes.NewBufferString(`[{"signature":"abc"},{"signature":"def"},{"signature":"ghi"}]`),
	)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}

	var body Envelope[ProviderWebhookAcceptancePayload]
	decode(t, rr.Body.Bytes(), &body)

	if body.Data.AcceptedCount != 3 {
		t.Fatalf("expected accepted count 3, got %d", body.Data.AcceptedCount)
	}
	if body.Data.EventKind != "webhook_batch" {
		t.Fatalf("unexpected event kind %q", body.Data.EventKind)
	}
	if !ingest.providerCalled {
		t.Fatal("expected provider ingest service to be called")
	}
}

func TestWebhookRouteRejectsUnsupportedProvider(t *testing.T) {
	t.Parallel()

	srv := New()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/providers/unknown/webhooks/address-activity",
		bytes.NewBufferString(`{}`),
	)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestWebhookRouteRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	srv := New()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/providers/alchemy/webhooks/address-activity",
		bytes.NewBufferString(`{"event":{"activity":[]}}`),
	)

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func decode[T any](t *testing.T, raw []byte, dst *T) {
	t.Helper()

	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

type testWalletSummaryRepository struct {
	summary domain.WalletSummary
}

func (r *testWalletSummaryRepository) FindWalletSummary(_ context.Context, _ string, _ string) (domain.WalletSummary, error) {
	return r.summary, nil
}

type testWalletGraphRepository struct {
	graph  domain.WalletGraph
	called bool
}

func (r *testWalletGraphRepository) FindWalletGraph(_ context.Context, _ string, _ string, depth int) (domain.WalletGraph, error) {
	r.called = true
	graph := r.graph
	graph.DepthRequested = depth
	if depth > 1 {
		graph.DepthResolved = 1
		graph.DensityCapped = true
	}

	return graph, nil
}

type testClusterDetailRepository struct {
	detail domain.ClusterDetail
	err    error
	called bool
}

func (r *testClusterDetailRepository) FindClusterDetail(_ context.Context, _ string) (domain.ClusterDetail, error) {
	r.called = true
	return r.detail, r.err
}

type fakeWebhookIngestService struct {
	alchemyResult  WebhookIngestResult
	providerResult WebhookIngestResult

	alchemyCalled  bool
	providerCalled bool
}

func (f *fakeWebhookIngestService) IngestAlchemyAddressActivity(_ context.Context, _ AlchemyAddressActivityWebhook) (WebhookIngestResult, error) {
	f.alchemyCalled = true
	if f.alchemyResult == (WebhookIngestResult{}) {
		return WebhookIngestResult{AcceptedCount: 1, EventKind: "address_activity"}, nil
	}

	return f.alchemyResult, nil
}

func (f *fakeWebhookIngestService) IngestProviderWebhook(_ context.Context, provider string, _ json.RawMessage) (WebhookIngestResult, error) {
	f.providerCalled = true
	if f.providerResult == (WebhookIngestResult{}) {
		switch provider {
		case "helius":
			return WebhookIngestResult{AcceptedCount: 1, EventKind: "webhook_event"}, nil
		default:
			return WebhookIngestResult{AcceptedCount: 1, EventKind: "address_activity"}, nil
		}
	}

	return f.providerResult, nil
}

type fakeSearchWalletSummaryLookup struct {
	summary service.WalletSummary
	err     error
}

func (f *fakeSearchWalletSummaryLookup) GetWalletSummary(_ context.Context, _, _ string) (service.WalletSummary, error) {
	return f.summary, f.err
}

type fakeSearchWalletBackfillQueueStore struct {
	jobs []db.WalletBackfillJob
}

func (f *fakeSearchWalletBackfillQueueStore) EnqueueWalletBackfill(_ context.Context, job db.WalletBackfillJob) error {
	f.jobs = append(f.jobs, job)
	return nil
}

func (f *fakeSearchWalletBackfillQueueStore) DequeueWalletBackfill(_ context.Context, _ string) (db.WalletBackfillJob, bool, error) {
	return db.WalletBackfillJob{}, false, nil
}

func walletSummaryFixture() domain.WalletSummary {
	clusterID := "cluster_seed_whales"
	latest := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)

	return domain.WalletSummary{
		Chain:            domain.ChainEVM,
		Address:          "0x1234567890abcdef1234567890abcdef12345678",
		DisplayName:      "Seed Whale",
		ClusterID:        &clusterID,
		Counterparties:   18,
		LatestActivityAt: latest.Format(time.RFC3339),
		Tags:             []string{"wallet-summary", "evm"},
		Scores: []domain.Score{
			{
				Name:   domain.ScoreCluster,
				Value:  82,
				Rating: domain.RatingHigh,
				Evidence: []domain.Evidence{
					{
						Kind:       domain.EvidenceClusterOverlap,
						Label:      "cluster overlap",
						Source:     "cluster-engine",
						Confidence: 0.9,
						ObservedAt: latest.Format(time.RFC3339),
						Metadata:   map[string]any{"overlapping_wallets": 6},
					},
				},
			},
			{
				Name:   domain.ScoreShadowExit,
				Value:  34,
				Rating: domain.RatingMedium,
				Evidence: []domain.Evidence{
					{
						Kind:       domain.EvidenceBridge,
						Label:      "bridge movement",
						Source:     "shadow-exit-engine",
						Confidence: 0.58,
						ObservedAt: latest.Format(time.RFC3339),
						Metadata:   map[string]any{"bridge_transfers": 1},
					},
				},
			},
			{
				Name:   domain.ScoreAlpha,
				Value:  21,
				Rating: domain.RatingLow,
				Evidence: []domain.Evidence{
					{
						Kind:       domain.EvidenceLabel,
						Label:      "first connection",
						Source:     "alpha-engine",
						Confidence: 0.42,
						ObservedAt: latest.Format(time.RFC3339),
						Metadata:   map[string]any{"hot_feed_mentions": 1},
					},
				},
			},
		},
	}
}

func walletGraphFixture() domain.WalletGraph {
	return domain.WalletGraph{
		Chain:          domain.ChainEVM,
		Address:        "0x1234567890abcdef1234567890abcdef12345678",
		DepthRequested: 1,
		DepthResolved:  1,
		DensityCapped:  false,
		Nodes: []domain.WalletGraphNode{
			{
				ID:      "wallet:evm:0x1234567890abcdef1234567890abcdef12345678",
				Kind:    domain.WalletGraphNodeWallet,
				Chain:   domain.ChainEVM,
				Address: "0x1234567890abcdef1234567890abcdef12345678",
				Label:   "Seed Whale",
			},
			{
				ID:    "cluster:cluster_seed_whales",
				Kind:  domain.WalletGraphNodeCluster,
				Label: "cluster_seed_whales",
			},
			{
				ID:    "entity:entity_seed_whale",
				Kind:  domain.WalletGraphNodeEntity,
				Label: "entity_seed_whale",
			},
		},
		Edges: []domain.WalletGraphEdge{
			{
				SourceID:   "wallet:evm:0x1234567890abcdef1234567890abcdef12345678",
				TargetID:   "cluster:cluster_seed_whales",
				Kind:       domain.WalletGraphEdgeMemberOf,
				Weight:     82,
				ObservedAt: "2026-03-19T01:02:03Z",
			},
			{
				SourceID:   "wallet:evm:0x1234567890abcdef1234567890abcdef12345678",
				TargetID:   "entity:entity_seed_whale",
				Kind:       domain.WalletGraphEdgeFundedBy,
				Weight:     13,
				ObservedAt: "2026-03-19T01:02:03Z",
			},
		},
	}
}

func clusterDetailFixture() domain.ClusterDetail {
	return domain.ClusterDetail{
		ID:             "cluster_seed_whales",
		Label:          "cluster_seed_whales",
		ClusterType:    "whale",
		Score:          82,
		Classification: domain.ClusterClassificationStrong,
		MemberCount:    2,
		Members: []domain.ClusterMember{
			{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678", Label: "Seed Whale"},
			{Chain: domain.ChainEVM, Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", Label: "Counterparty Seed"},
		},
		CommonActions: []domain.ClusterCommonAction{
			{
				Kind:              "shared_counterparty",
				Label:             "Bridge wallet",
				Chain:             domain.ChainEVM,
				Address:           "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
				SharedMemberCount: 2,
				InteractionCount:  11,
				ObservedAt:        "2026-03-19T01:02:03Z",
			},
		},
		Evidence: []domain.Evidence{
			{
				Kind:       domain.EvidenceClusterOverlap,
				Label:      "cluster member overlap",
				Source:     "cluster-detail",
				Confidence: 0.88,
				ObservedAt: "2026-03-19T01:02:03Z",
				Metadata:   map[string]any{"member_count": 2, "score": 82},
			},
		},
	}
}

func shadowExitFeedFixture() domain.ShadowExitFeedPage {
	latest := time.Date(2026, time.March, 20, 3, 4, 5, 0, time.UTC)
	cursor := "2026-03-20T03:04:05Z|wallet_1"

	return domain.ShadowExitFeedPage{
		Items: []domain.ShadowExitFeedItem{
			{
				WalletID:       "wallet_1",
				Chain:          domain.ChainSolana,
				Address:        "So11111111111111111111111111111111111111112",
				Label:          "Seed Whale",
				WalletRoute:    "/wallets/solana/So11111111111111111111111111111111111111112",
				Recommendation: "Potential exit-like reshuffling; review recent counterparties and bridge activity.",
				ObservedAt:     latest.Format(time.RFC3339),
				Score: domain.Score{
					Name:   domain.ScoreShadowExit,
					Value:  34,
					Rating: domain.RatingMedium,
					Evidence: []domain.Evidence{
						{
							Kind:       domain.EvidenceBridge,
							Label:      "bridge movement",
							Source:     "shadow-exit-snapshot",
							Confidence: 1,
							ObservedAt: latest.Format(time.RFC3339),
							Metadata:   map[string]any{"bridge_transfers": 1},
						},
					},
				},
			},
		},
		NextCursor: &cursor,
		HasMore:    true,
	}
}

func firstConnectionFeedFixture() domain.FirstConnectionFeedPage {
	latest := time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
	cursor := db.EncodeFirstConnectionFeedCursor(latest, "wallet_1")

	return domain.FirstConnectionFeedPage{
		Items: []domain.FirstConnectionFeedItem{
			{
				WalletID:       "wallet_1",
				Chain:          domain.ChainEVM,
				Address:        "0x1234567890abcdef1234567890abcdef12345678",
				Label:          "Seed Whale",
				WalletRoute:    "/wallets/evm/0x1234567890abcdef1234567890abcdef12345678",
				Recommendation: "Elevated first-connection activity; review recent counterparties and activity.",
				ObservedAt:     latest.Format(time.RFC3339),
				Score: domain.Score{
					Name:   domain.ScoreAlpha,
					Value:  72,
					Rating: domain.RatingHigh,
					Evidence: []domain.Evidence{
						{
							Kind:       domain.EvidenceTransfer,
							Label:      "first connection discovery signal",
							Source:     "first-connection-snapshot",
							Confidence: 1,
							ObservedAt: latest.Format(time.RFC3339),
							Metadata:   map[string]any{"new_common_entries": 2},
						},
					},
				},
			},
		},
		NextCursor: &cursor,
		HasMore:    true,
	}
}

type testShadowExitFeedRepository struct {
	page   domain.ShadowExitFeedPage
	called bool
}

func (r *testShadowExitFeedRepository) FindShadowExitFeed(_ context.Context, _ string, _ int) (domain.ShadowExitFeedPage, error) {
	r.called = true
	return r.page, nil
}

type testFirstConnectionFeedRepository struct {
	page   domain.FirstConnectionFeedPage
	called bool
	sort   string
}

func (r *testFirstConnectionFeedRepository) FindFirstConnectionFeed(_ context.Context, _ string, _ int, sort string) (domain.FirstConnectionFeedPage, error) {
	r.called = true
	r.sort = sort
	return r.page, nil
}
