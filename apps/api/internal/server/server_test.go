package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/auth"
	"github.com/whalegraph/whalegraph/apps/api/internal/service"
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
		Wallets:       service.NewWalletSummaryService(&testWalletSummaryRepository{summary: walletSummaryFixture()}),
		Graphs:        service.NewWalletGraphService(&testWalletGraphRepository{graph: walletGraphFixture()}),
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

func TestWalletGraphRouteDefaultsToOneHop(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		Graphs:        service.NewWalletGraphService(&testWalletGraphRepository{graph: walletGraphFixture()}),
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
}

func TestWalletGraphRouteBlocksFreeTierTwoHop(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		Graphs:        service.NewWalletGraphService(&testWalletGraphRepository{graph: walletGraphFixture()}),
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

func TestSearchRouteEnrichesWalletLikeInputFromLookup(t *testing.T) {
	t.Parallel()

	walletService := service.NewWalletSummaryService(&testWalletSummaryRepository{summary: walletSummaryFixture()})
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
