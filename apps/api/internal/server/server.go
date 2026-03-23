package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/auth"
	"github.com/whalegraph/whalegraph/apps/api/internal/repository"
	"github.com/whalegraph/whalegraph/apps/api/internal/service"
	"github.com/whalegraph/whalegraph/packages/billing"
	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type Envelope[T any] struct {
	Success bool         `json:"success"`
	Data    T            `json:"data"`
	Error   *APIError    `json:"error"`
	Meta    ResponseMeta `json:"meta"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ResponseMeta struct {
	RequestID string        `json:"requestId"`
	Timestamp string        `json:"timestamp"`
	Chain     string        `json:"chain,omitempty"`
	Tier      string        `json:"tier,omitempty"`
	Freshness FreshnessMeta `json:"freshness"`
}

type FreshnessMeta struct {
	GeneratedAt   string `json:"generatedAt"`
	Source        string `json:"source"`
	MaxAgeSeconds int    `json:"maxAgeSeconds"`
}

type HealthPayload struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type AdminStatusPayload struct {
	Status string `json:"status"`
	Scope  string `json:"scope"`
}

type ProviderWebhookAcceptancePayload struct {
	Provider      string `json:"provider"`
	EventKind     string `json:"eventKind"`
	AcceptedCount int    `json:"acceptedCount"`
	EventCount    int    `json:"eventCount"`
	Accepted      bool   `json:"accepted"`
}

type Server struct {
	mux              *http.ServeMux
	wallets          *service.WalletSummaryService
	graphs           *service.WalletGraphService
	clusters         *service.ClusterDetailService
	shadowExits      *service.ShadowExitFeedService
	firstConnections *service.FirstConnectionFeedService
	alertRules       *service.AlertRuleService
	alertDelivery    *service.AlertDeliveryService
	watchlists       *service.WatchlistService
	adminConsole     *service.AdminConsoleService
	account          *service.AccountService
	billing          *service.BillingService
	search           *service.SearchService
	webhookIngest    WebhookIngestService
}

type Dependencies struct {
	Wallets          *service.WalletSummaryService
	Graphs           *service.WalletGraphService
	Clusters         *service.ClusterDetailService
	ShadowExits      *service.ShadowExitFeedService
	FirstConnections *service.FirstConnectionFeedService
	AlertRules       *service.AlertRuleService
	AlertDelivery    *service.AlertDeliveryService
	Watchlists       *service.WatchlistService
	AdminConsole     *service.AdminConsoleService
	Account          *service.AccountService
	Billing          *service.BillingService
	Search           *service.SearchService
	WebhookIngest    WebhookIngestService
	ClerkVerifier    auth.ClerkVerifier
}

func New() *Server {
	return NewWithDependencies(Dependencies{})
}

func NewWithDependencies(deps Dependencies) *Server {
	if deps.Wallets == nil {
		deps.Wallets = service.NewWalletSummaryService(
			repository.NewQueryBackedWalletSummaryRepository(notFoundWalletSummaryLoader{}),
			nil,
		)
	}
	if deps.Graphs == nil {
		deps.Graphs = service.NewWalletGraphService(
			repository.NewQueryBackedWalletGraphRepository(notFoundWalletGraphLoader{}),
		)
	}
	if deps.Search == nil {
		deps.Search = service.NewSearchService(deps.Wallets)
	}
	if deps.Clusters == nil {
		deps.Clusters = service.NewClusterDetailService(
			repository.NewQueryBackedClusterDetailRepository(notFoundClusterDetailLoader{}),
		)
	}
	if deps.ShadowExits == nil {
		deps.ShadowExits = service.NewShadowExitFeedService(
			repository.NewQueryBackedShadowExitFeedRepository(notFoundShadowExitFeedLoader{}),
		)
	}
	if deps.FirstConnections == nil {
		deps.FirstConnections = service.NewFirstConnectionFeedService(
			repository.NewQueryBackedFirstConnectionFeedRepository(notFoundFirstConnectionFeedLoader{}),
		)
	}
	if deps.AlertRules == nil {
		deps.AlertRules = service.NewAlertRuleService(repository.NewInMemoryAlertRuleRepository())
	}
	if deps.AlertDelivery == nil {
		deps.AlertDelivery = service.NewAlertDeliveryService(repository.NewInMemoryAlertDeliveryRepository())
	}
	if deps.Watchlists == nil {
		deps.Watchlists = service.NewWatchlistService(repository.NewInMemoryWatchlistRepository())
	}
	if deps.AdminConsole == nil {
		deps.AdminConsole = service.NewAdminConsoleService(repository.NewInMemoryAdminConsoleRepository())
	}
	if deps.Account == nil {
		deps.Account = service.NewAccountService()
	}
	if deps.Billing == nil {
		deps.Billing = service.NewBillingService(repository.NewInMemoryBillingRepository(), billing.StripeConfig{})
	}
	if deps.WebhookIngest == nil {
		deps.WebhookIngest = newCountingWebhookIngestService()
	}
	if deps.ClerkVerifier == nil {
		deps.ClerkVerifier = auth.NewHeaderClerkVerifier()
	}

	mux := http.NewServeMux()
	s := &Server{
		mux:              mux,
		wallets:          deps.Wallets,
		graphs:           deps.Graphs,
		clusters:         deps.Clusters,
		shadowExits:      deps.ShadowExits,
		firstConnections: deps.FirstConnections,
		alertRules:       deps.AlertRules,
		alertDelivery:    deps.AlertDelivery,
		watchlists:       deps.Watchlists,
		adminConsole:     deps.AdminConsole,
		account:          deps.Account,
		billing:          deps.Billing,
		search:           deps.Search,
		webhookIngest:    deps.WebhookIngest,
	}

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /v1/search", s.handleSearch)
	mux.HandleFunc("GET /v1/billing/plans", s.handleBillingPlans)
	mux.HandleFunc("GET /v1/clusters/", s.handleClusterRoute)
	mux.HandleFunc("GET /v1/signals/shadow-exits", s.handleShadowExitFeed)
	mux.HandleFunc("GET /v1/signals/first-connections", s.handleFirstConnectionFeed)
	mux.Handle("POST /v1/billing/checkout-sessions", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleBillingCheckoutSession)))
	mux.HandleFunc("POST /v1/webhooks/billing/stripe", s.handleStripeBillingWebhook)
	mux.Handle("GET /v1/alerts", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAlertInbox)))
	mux.Handle("PATCH /v1/alerts/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAlertInboxRoute)))
	mux.Handle("GET /v1/alert-rules", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAlertRuleCollection)))
	mux.Handle("POST /v1/alert-rules", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAlertRuleCollection)))
	mux.Handle("GET /v1/alert-rules/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAlertRuleRoute)))
	mux.Handle("PATCH /v1/alert-rules/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAlertRuleRoute)))
	mux.Handle("DELETE /v1/alert-rules/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAlertRuleRoute)))
	mux.Handle("GET /v1/alert-delivery-channels", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAlertDeliveryChannelCollection)))
	mux.Handle("POST /v1/alert-delivery-channels", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAlertDeliveryChannelCollection)))
	mux.Handle("GET /v1/alert-delivery-channels/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAlertDeliveryChannelRoute)))
	mux.Handle("PATCH /v1/alert-delivery-channels/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAlertDeliveryChannelRoute)))
	mux.Handle("DELETE /v1/alert-delivery-channels/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAlertDeliveryChannelRoute)))
	mux.Handle("GET /v1/watchlists", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleWatchlistCollection)))
	mux.Handle("POST /v1/watchlists", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleWatchlistCollection)))
	mux.Handle("GET /v1/watchlists/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleWatchlistRoute)))
	mux.Handle("POST /v1/watchlists/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleWatchlistRoute)))
	mux.Handle("PATCH /v1/watchlists/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleWatchlistRoute)))
	mux.Handle("DELETE /v1/watchlists/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleWatchlistRoute)))
	mux.HandleFunc("POST /v1/webhooks/providers/alchemy/address-activity", s.handleAlchemyAddressActivityWebhook)
	mux.HandleFunc("POST /v1/providers/", s.handleProviderRoute)
	mux.HandleFunc("GET /v1/wallets/", s.handleWalletRoute)
	mux.Handle(
		"GET /v1/admin/status",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin", "operator")(http.HandlerFunc(s.handleAdminStatus)),
	)
	mux.Handle(
		"GET /v1/admin/labels",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin", "operator")(http.HandlerFunc(s.handleAdminLabels)),
	)
	mux.Handle(
		"POST /v1/admin/labels",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin")(http.HandlerFunc(s.handleAdminLabels)),
	)
	mux.Handle(
		"DELETE /v1/admin/labels/",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin")(http.HandlerFunc(s.handleAdminLabelRoute)),
	)
	mux.Handle(
		"GET /v1/admin/suppressions",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin", "operator")(http.HandlerFunc(s.handleAdminSuppressions)),
	)
	mux.Handle(
		"POST /v1/admin/suppressions",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin")(http.HandlerFunc(s.handleAdminSuppressions)),
	)
	mux.Handle(
		"DELETE /v1/admin/suppressions/",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin")(http.HandlerFunc(s.handleAdminSuppressionRoute)),
	)
	mux.Handle(
		"GET /v1/admin/provider-quotas",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin", "operator")(http.HandlerFunc(s.handleAdminProviderQuotas)),
	)
	mux.Handle(
		"GET /v1/admin/observability",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin", "operator")(http.HandlerFunc(s.handleAdminObservability)),
	)
	mux.Handle(
		"GET /v1/admin/curated-lists",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin", "operator")(http.HandlerFunc(s.handleAdminCuratedLists)),
	)
	mux.Handle(
		"POST /v1/admin/curated-lists",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin")(http.HandlerFunc(s.handleAdminCuratedLists)),
	)
	mux.Handle(
		"DELETE /v1/admin/curated-lists/",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin")(http.HandlerFunc(s.handleAdminCuratedListRoute)),
	)
	mux.Handle(
		"POST /v1/admin/curated-lists/",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin")(http.HandlerFunc(s.handleAdminCuratedListRoute)),
	)
	mux.Handle(
		"GET /v1/admin/audit-logs",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin", "operator")(http.HandlerFunc(s.handleAdminAuditLogs)),
	)
	mux.Handle(
		"GET /v1/account",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAccount)),
	)
	mux.Handle(
		"GET /v1/account/entitlements",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAccount)),
	)

	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Envelope[HealthPayload]{
		Success: true,
		Data: HealthPayload{
			Status:  "ok",
			Service: "api",
		},
		Meta: newMeta("", "", freshness("snapshot", 60)),
	})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("query"))
	}
	if query == "" {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "search query is required", "", tierFromHeader(r)))
		return
	}

	options := service.SearchOptions{
		ManualRefresh: strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("refresh")), "manual"),
	}

	writeJSON(w, http.StatusOK, Envelope[service.SearchResponse]{
		Success: true,
		Data:    s.search.SearchWithOptions(r.Context(), query, options),
		Meta:    newMeta("", tierFromHeader(r), freshness("snapshot", 60)),
	})
}

func (s *Server) handleWalletRoute(w http.ResponseWriter, r *http.Request) {
	chain, address, resource, ok := parseWalletRoutePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "wallet route not found", "", ""))
		return
	}

	switch resource {
	case "summary":
		s.handleWalletSummary(w, r, chain, address)
	case "graph":
		s.handleWalletGraph(w, r, chain, address)
	default:
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "wallet route not found", chain, tierFromHeader(r)))
	}
}

func (s *Server) handleProviderRoute(w http.ResponseWriter, r *http.Request) {
	provider, resource, ok := parseProviderRoutePath(r.URL.Path)
	if !ok || resource != "webhooks/address-activity" {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "provider route not found", "", ""))
		return
	}

	s.handleProviderAddressActivityWebhook(w, r, provider)
}

func (s *Server) handleClusterRoute(w http.ResponseWriter, r *http.Request) {
	clusterID, ok := parseClusterRoutePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "cluster route not found", "", tierFromHeader(r)))
		return
	}

	if strings.TrimSpace(clusterID) == "" {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid cluster detail request", "", tierFromHeader(r)))
		return
	}

	detail, err := s.clusters.GetClusterDetail(r.Context(), clusterID)
	if err != nil {
		if errors.Is(err, service.ErrClusterDetailNotFound) {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", err.Error(), "", tierFromHeader(r)))
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "cluster detail lookup failed", "", tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.ClusterDetail]{
		Success: true,
		Data:    detail,
		Meta:    newMeta("", tierFromHeader(r), freshness("snapshot", 300)),
	})
}

func (s *Server) handleShadowExitFeed(w http.ResponseWriter, r *http.Request) {
	limit, err := parseShadowExitFeedLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid shadow exit feed limit", "", tierFromHeader(r)))
		return
	}

	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	if _, err := db.BuildShadowExitFeedQuery(limit, cursor); err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid shadow exit feed cursor", "", tierFromHeader(r)))
		return
	}

	feed, err := s.shadowExits.ListShadowExitFeed(r.Context(), cursor, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "shadow exit feed lookup failed", "", tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.ShadowExitFeedResponse]{
		Success: true,
		Data:    feed,
		Meta:    newMeta("", tierFromHeader(r), freshness("snapshot", 300)),
	})
}

func (s *Server) handleFirstConnectionFeed(w http.ResponseWriter, r *http.Request) {
	limit, err := parseFirstConnectionFeedLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid first connection feed limit", "", tierFromHeader(r)))
		return
	}

	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	sort := strings.TrimSpace(r.URL.Query().Get("sort"))
	query, err := db.BuildFirstConnectionFeedQuery(limit, cursor, sort)
	if err != nil {
		message := "invalid first connection feed cursor"
		if strings.Contains(strings.ToLower(err.Error()), "sort") {
			message = "invalid first connection feed sort"
		}
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", message, "", tierFromHeader(r)))
		return
	}

	feed, err := s.firstConnections.ListFirstConnectionFeedSorted(r.Context(), cursor, query.Limit, string(query.Sort))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "first connection feed lookup failed", "", tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.FirstConnectionFeedResponse]{
		Success: true,
		Data:    feed,
		Meta:    newMeta("", tierFromHeader(r), freshness("snapshot", 300)),
	})
}

func (s *Server) handleWalletSummary(w http.ResponseWriter, r *http.Request, chain string, address string) {
	if !domain.IsSupportedChain(domain.Chain(chain)) || len(address) < 16 {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid wallet summary request", chain, tierFromHeader(r)))
		return
	}

	summary, err := s.wallets.GetWalletSummary(r.Context(), chain, address)
	if err != nil {
		if errors.Is(err, service.ErrWalletSummaryNotFound) {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", err.Error(), chain, tierFromHeader(r)))
			return
		}

		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "wallet summary lookup failed", chain, tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.WalletSummary]{
		Success: true,
		Data:    summary,
		Meta:    newMeta(chain, tierFromHeader(r), freshness("snapshot", 300)),
	})
}

func (s *Server) handleWalletGraph(w http.ResponseWriter, r *http.Request, chain string, address string) {
	if !domain.IsSupportedChain(domain.Chain(chain)) || len(address) < 16 {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid wallet graph request", chain, tierFromHeader(r)))
		return
	}

	depth, err := parseWalletGraphDepth(r.URL.Query().Get("depth"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid wallet graph depth", chain, tierFromHeader(r)))
		return
	}

	graph, err := s.graphs.GetWalletGraph(r.Context(), chain, address, depth, tierFromHeader(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWalletGraphNotFound):
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", err.Error(), chain, tierFromHeader(r)))
		case errors.Is(err, service.ErrWalletGraphDepthNotAllowed):
			writeJSON(w, http.StatusForbidden, errorEnvelope("FORBIDDEN", "2-hop graph requires a higher plan", chain, tierFromHeader(r)))
		default:
			writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "wallet graph lookup failed", chain, tierFromHeader(r)))
		}
		return
	}

	writeJSON(w, http.StatusOK, Envelope[domain.WalletGraph]{
		Success: true,
		Data:    graph,
		Meta:    newMeta(chain, tierFromHeader(r), freshness("snapshot", 300)),
	})
}

func (s *Server) handleAdminStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Envelope[AdminStatusPayload]{
		Success: true,
		Data: AdminStatusPayload{
			Status: "ok",
			Scope:  "admin",
		},
		Meta: newMeta("", "team", freshness("live", 30)),
	})
}

func (s *Server) handleProviderAddressActivityWebhook(w http.ResponseWriter, r *http.Request, provider string) {
	if !isSupportedWebhookProvider(provider) {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "unsupported webhook provider", "", ""))
		return
	}

	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid webhook payload", "", ""))
		return
	}

	result, err := s.webhookIngest.IngestProviderWebhook(r.Context(), provider, json.RawMessage(raw))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", err.Error(), "", ""))
		return
	}

	writeJSON(w, http.StatusAccepted, Envelope[ProviderWebhookAcceptancePayload]{
		Success: true,
		Data: ProviderWebhookAcceptancePayload{
			Provider:      provider,
			EventKind:     result.EventKind,
			AcceptedCount: result.AcceptedCount,
			EventCount:    result.AcceptedCount,
			Accepted:      true,
		},
		Meta: newMeta("", "system", freshness("live", 0)),
	})
}

func parseWalletRoutePath(path string) (string, string, string, bool) {
	rest, ok := strings.CutPrefix(path, "/v1/wallets/")
	if !ok {
		return "", "", "", false
	}

	chain, rest, ok := strings.Cut(rest, "/")
	if !ok {
		return "", "", "", false
	}

	address, resource, ok := strings.Cut(rest, "/")
	if !ok || chain == "" || address == "" {
		return "", "", "", false
	}

	if resource != "summary" && resource != "graph" {
		return "", "", "", false
	}

	return chain, address, resource, true
}

func parseProviderRoutePath(path string) (string, string, bool) {
	rest, ok := strings.CutPrefix(path, "/v1/providers/")
	if !ok {
		return "", "", false
	}

	provider, resource, ok := strings.Cut(rest, "/")
	if !ok || provider == "" || resource == "" {
		return "", "", false
	}

	return provider, resource, true
}

func parseClusterRoutePath(path string) (string, bool) {
	clusterID, ok := strings.CutPrefix(path, "/v1/clusters/")
	if !ok {
		return "", false
	}
	clusterID = strings.TrimSpace(clusterID)
	if clusterID == "" || strings.Contains(clusterID, "/") {
		return "", false
	}
	return clusterID, true
}

func parseWalletGraphDepth(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 1, nil
	}

	depth, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || depth <= 0 {
		return 0, errors.New("wallet graph depth must be positive")
	}

	return depth, nil
}

func parseShadowExitFeedLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 20, nil
	}

	limit, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || limit <= 0 {
		return 0, errors.New("shadow exit feed limit must be positive")
	}
	if limit > 50 {
		limit = 50
	}

	return limit, nil
}

func parseFirstConnectionFeedLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 20, nil
	}

	limit, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || limit <= 0 {
		return 0, errors.New("first connection feed limit must be positive")
	}
	if limit > 50 {
		limit = 50
	}

	return limit, nil
}

func isSupportedWebhookProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "alchemy", "helius":
		return true
	default:
		return false
	}
}

func tierFromHeader(r *http.Request) string {
	switch strings.ToLower(strings.TrimSpace(r.Header.Get("X-Whalegraph-Plan"))) {
	case "pro", "team":
		return strings.ToLower(strings.TrimSpace(r.Header.Get("X-Whalegraph-Plan")))
	default:
		return "free"
	}
}

func freshness(source string, maxAgeSeconds int) FreshnessMeta {
	return FreshnessMeta{
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Source:        source,
		MaxAgeSeconds: maxAgeSeconds,
	}
}

func newMeta(chain string, tier string, f FreshnessMeta) ResponseMeta {
	return ResponseMeta{
		RequestID: randomID(),
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Chain:     chain,
		Tier:      tier,
		Freshness: f,
	}
}

func errorEnvelope(code string, message string, chain string, tier string) Envelope[any] {
	return Envelope[any]{
		Success: false,
		Data:    nil,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
		Meta: newMeta(chain, tier, freshness("snapshot", 0)),
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func randomID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "req_unavailable"
	}

	return hex.EncodeToString(buf[:])
}

type apiAuthResponder struct{}

func (apiAuthResponder) Unauthorized(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
}

func (apiAuthResponder) Forbidden(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusForbidden, errorEnvelope("FORBIDDEN", "you do not have access to this resource", "", tierFromHeader(r)))
}

type notFoundWalletSummaryLoader struct{}

func (notFoundWalletSummaryLoader) LoadWalletSummaryInputs(context.Context, db.WalletRef) (db.WalletSummaryInputs, error) {
	return db.WalletSummaryInputs{}, db.ErrWalletSummaryNotFound
}

type notFoundWalletGraphLoader struct{}

func (notFoundWalletGraphLoader) LoadWalletGraph(context.Context, db.WalletGraphQuery) (domain.WalletGraph, error) {
	return domain.WalletGraph{}, db.ErrWalletGraphNotFound
}

type notFoundClusterDetailLoader struct{}

func (notFoundClusterDetailLoader) LoadClusterDetail(context.Context, string) (domain.ClusterDetail, error) {
	return domain.ClusterDetail{}, db.ErrClusterDetailNotFound
}

type notFoundShadowExitFeedLoader struct{}

func (notFoundShadowExitFeedLoader) LoadShadowExitFeed(context.Context, db.ShadowExitFeedQuery) (domain.ShadowExitFeedPage, error) {
	return domain.ShadowExitFeedPage{}, nil
}

type notFoundFirstConnectionFeedLoader struct{}

func (notFoundFirstConnectionFeedLoader) LoadFirstConnectionFeed(context.Context, db.FirstConnectionFeedQuery) (domain.FirstConnectionFeedPage, error) {
	return domain.FirstConnectionFeedPage{}, nil
}
