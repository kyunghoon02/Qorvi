package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/qorvi/qorvi/apps/api/internal/auth"
	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/apps/api/internal/service"
	"github.com/qorvi/qorvi/packages/billing"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
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
	mux                *http.ServeMux
	handler            http.Handler
	wallets            *service.WalletSummaryService
	walletBriefs       *service.WalletBriefService
	graphs             *service.WalletGraphService
	discover           *service.DiscoverService
	analystTools       *service.AnalystToolsService
	analystFindings    *service.AnalystFindingDrilldownService
	analystExplain     *service.AnalystFindingExplanationService
	interactiveAnalyst *service.InteractiveAnalystService
	findings           *service.FindingsFeedService
	entities           *service.EntityInterpretationService
	clusters           *service.ClusterDetailService
	shadowExits        *service.ShadowExitFeedService
	firstConnections   *service.FirstConnectionFeedService
	alertRules         *service.AlertRuleService
	alertDelivery      *service.AlertDeliveryService
	watchlists         *service.WatchlistService
	adminConsole       *service.AdminConsoleService
	adminBacktests     *service.AdminBacktestOpsService
	account            *service.AccountService
	billing            *service.BillingService
	search             *service.SearchService
	webhookIngest      WebhookIngestService
	adminAllowlist     adminPrincipalAllowlist
}

type Dependencies struct {
	Wallets             *service.WalletSummaryService
	WalletBriefs        *service.WalletBriefService
	Graphs              *service.WalletGraphService
	Discover            *service.DiscoverService
	AnalystTools        *service.AnalystToolsService
	AnalystFindings     *service.AnalystFindingDrilldownService
	AnalystExplanations *service.AnalystFindingExplanationService
	InteractiveAnalyst  *service.InteractiveAnalystService
	Findings            *service.FindingsFeedService
	Entities            *service.EntityInterpretationService
	Clusters            *service.ClusterDetailService
	ShadowExits         *service.ShadowExitFeedService
	FirstConnections    *service.FirstConnectionFeedService
	AlertRules          *service.AlertRuleService
	AlertDelivery       *service.AlertDeliveryService
	Watchlists          *service.WatchlistService
	AdminConsole        *service.AdminConsoleService
	AdminBacktests      *service.AdminBacktestOpsService
	Account             *service.AccountService
	Billing             *service.BillingService
	Search              *service.SearchService
	WebhookIngest       WebhookIngestService
	ClerkVerifier       auth.ClerkVerifier
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
	if deps.Discover == nil {
		deps.Discover = service.NewDiscoverService(nil)
	}
	if deps.Findings == nil {
		deps.Findings = service.NewFindingsFeedService(
			repository.NewQueryBackedFindingsRepository(nil),
		)
	}
	if deps.WalletBriefs == nil {
		deps.WalletBriefs = service.NewWalletBriefService(
			repository.NewQueryBackedWalletSummaryRepository(notFoundWalletSummaryLoader{}),
			nil,
			repository.NewQueryBackedFindingsRepository(nil),
			repository.NewQueryBackedWalletEntryFeaturesRepository(nil),
		)
	}
	if deps.Entities == nil {
		deps.Entities = service.NewEntityInterpretationService(
			repository.NewQueryBackedEntityInterpretationRepository(nil),
		)
	}
	if deps.Search == nil {
		deps.Search = service.NewSearchService(deps.Wallets)
	}
	if deps.AdminBacktests == nil {
		deps.AdminBacktests = service.NewAdminBacktestOpsService("", "", "")
	}
	if deps.AnalystTools == nil {
		deps.AnalystTools = service.NewAnalystToolsService(
			deps.Wallets,
			deps.WalletBriefs,
			deps.Graphs,
		)
	}
	if deps.AnalystFindings == nil {
		deps.AnalystFindings = service.NewAnalystFindingDrilldownService(
			repository.NewQueryBackedFindingsRepository(nil),
			deps.Wallets,
			repository.NewQueryBackedWalletEntryFeaturesRepository(nil),
		)
	}
	if deps.AnalystExplanations == nil {
		deps.AnalystExplanations = service.NewAnalystFindingExplanationService(
			repository.NewQueryBackedFindingsRepository(nil),
			nil,
			nil,
			nil,
			nil,
		)
	}
	if deps.InteractiveAnalyst == nil {
		deps.InteractiveAnalyst = service.NewInteractiveAnalystService(
			deps.WalletBriefs,
			deps.AnalystTools,
			deps.AnalystFindings,
			deps.Entities,
		)
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
		mux:                mux,
		wallets:            deps.Wallets,
		walletBriefs:       deps.WalletBriefs,
		graphs:             deps.Graphs,
		discover:           deps.Discover,
		analystTools:       deps.AnalystTools,
		analystFindings:    deps.AnalystFindings,
		analystExplain:     deps.AnalystExplanations,
		interactiveAnalyst: deps.InteractiveAnalyst,
		findings:           deps.Findings,
		entities:           deps.Entities,
		clusters:           deps.Clusters,
		shadowExits:        deps.ShadowExits,
		firstConnections:   deps.FirstConnections,
		alertRules:         deps.AlertRules,
		alertDelivery:      deps.AlertDelivery,
		watchlists:         deps.Watchlists,
		adminConsole:       deps.AdminConsole,
		adminBacktests:     deps.AdminBacktests,
		account:            deps.Account,
		billing:            deps.Billing,
		search:             deps.Search,
		webhookIngest:      deps.WebhookIngest,
		adminAllowlist:     loadAdminPrincipalAllowlistFromEnv(),
	}
	s.handler = withCORS(mux, loadCORSAllowedOriginsFromEnv())

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /v1/search", s.handleSearch)
	mux.HandleFunc("GET /v1/findings", s.handleFindingsFeed)
	mux.HandleFunc("GET /v1/discover/featured-wallets", s.handleDiscoverFeaturedWallets)
	mux.HandleFunc("GET /v1/analyst/findings", s.handleAnalystFindingsFeed)
	mux.HandleFunc("GET /v1/analyst/findings/", s.handleAnalystFindingRoute)
	mux.Handle("POST /v1/analyst/findings/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAuthorizedAnalystFindingRoute)))
	mux.HandleFunc("GET /v1/entity/", s.handleEntityRoute)
	mux.HandleFunc("GET /v1/analyst/entity/", s.handleAnalystEntityRoute)
	mux.Handle("POST /v1/analyst/entity/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAuthorizedAnalystEntityRoute)))
	mux.Handle("POST /v1/analyst/wallets/", auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "user", "admin", "operator")(http.HandlerFunc(s.handleAuthorizedAnalystWalletRoute)))
	mux.HandleFunc("GET /v1/analyst/tools/wallets/", s.handleAnalystToolWalletRoute)
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
	mux.HandleFunc("POST /v1/webhooks/providers/helius/address-activity", s.handleHeliusAddressActivityWebhook)
	mux.HandleFunc("POST /v1/providers/", s.handleProviderRoute)
	mux.HandleFunc("GET /v1/wallets/", s.handleWalletRoute)
	mux.HandleFunc("GET /v1/analyst/wallets/", s.handleAnalystWalletRoute)
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
		"GET /v1/admin/backtests",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin")(http.HandlerFunc(s.handleAdminBacktests)),
	)
	mux.Handle(
		"POST /v1/admin/backtests/",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin")(http.HandlerFunc(s.handleAdminBacktestRoute)),
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
	if s == nil {
		return http.NewServeMux()
	}
	if s.handler != nil {
		return s.handler
	}
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
	case "brief":
		s.handleWalletBrief(w, r, chain, address)
	case "graph":
		s.handleWalletGraph(w, r, chain, address)
	default:
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "wallet route not found", chain, tierFromHeader(r)))
	}
}

func (s *Server) handleAnalystWalletRoute(w http.ResponseWriter, r *http.Request) {
	chain, address, resource, ok := parseAnalystWalletRoutePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst wallet route not found", "", ""))
		return
	}

	switch resource {
	case "brief":
		s.handleWalletBrief(w, r, chain, address)
	default:
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst wallet route not found", chain, tierFromHeader(r)))
	}
}

func (s *Server) handleAnalystToolWalletRoute(w http.ResponseWriter, r *http.Request) {
	chain, address, resource, ok := parseAnalystToolWalletRoutePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst tool wallet route not found", "", ""))
		return
	}

	switch resource {
	case "counterparties":
		s.handleAnalystWalletCounterparties(w, r, chain, address)
	case "graph":
		s.handleAnalystWalletGraph(w, r, chain, address)
	case "behavior-patterns":
		s.handleAnalystBehaviorPatterns(w, r, chain, address)
	default:
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst tool wallet route not found", chain, tierFromHeader(r)))
	}
}

func (s *Server) handleFindingsFeed(w http.ResponseWriter, r *http.Request) {
	limit, err := parseFindingsFeedLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid findings feed limit", "", tierFromHeader(r)))
		return
	}

	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	types := parseFindingsTypesFromQuery(r.URL.Query())
	feed, err := s.findings.ListFindings(r.Context(), cursor, limit, types)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "findings feed lookup failed", "", tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.FindingsFeedResponse]{
		Success: true,
		Data:    feed,
		Meta:    newMeta("", tierFromHeader(r), freshness("snapshot", 300)),
	})
}

func (s *Server) handleDiscoverFeaturedWallets(w http.ResponseWriter, r *http.Request) {
	items, err := s.discover.ListFeaturedWallets(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "discover featured wallet lookup failed", "", tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.DiscoverFeaturedWalletResponse]{
		Success: true,
		Data:    items,
		Meta:    newMeta("", tierFromHeader(r), freshness("snapshot", 300)),
	})
}

func (s *Server) handleAnalystFindingsFeed(w http.ResponseWriter, r *http.Request) {
	limit, err := parseFindingsFeedLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid analyst findings feed limit", "", tierFromHeader(r)))
		return
	}

	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	types := parseFindingsTypesFromQuery(r.URL.Query())
	feed, err := s.findings.ListFindings(r.Context(), cursor, limit, types)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "analyst findings feed lookup failed", "", tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.FindingsFeedResponse]{
		Success: true,
		Data:    feed,
		Meta:    newMeta("", tierFromHeader(r), freshness("analyst-snapshot", 300)),
	})
}

func (s *Server) handleAnalystFindingRoute(w http.ResponseWriter, r *http.Request) {
	findingID, resource, ok := parseAnalystFindingRoutePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst finding route not found", "", tierFromHeader(r)))
		return
	}

	switch resource {
	case "":
		s.handleAnalystFindingDetail(w, r, findingID)
	case "evidence-timeline":
		s.handleAnalystFindingEvidenceTimeline(w, r, findingID)
	case "historical-analogs":
		s.handleAnalystFindingHistoricalAnalogs(w, r, findingID)
	default:
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst finding route not found", "", tierFromHeader(r)))
	}
}

func (s *Server) handleAuthorizedAnalystFindingRoute(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", tierFromHeader(r)))
		return
	}

	findingID, resource, ok := parseAnalystFindingRoutePath(r.URL.Path)
	if !ok || resource != "explain" {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst finding route not found", "", tierFromHeader(r)))
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", tierFromHeader(r)))
		return
	}

	s.handleAnalystFindingExplain(w, r, principal, findingID)
}

func (s *Server) handleAuthorizedAnalystWalletRoute(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", tierFromHeader(r)))
		return
	}

	chain, address, resource, ok := parseAnalystWalletRoutePath(r.URL.Path)
	if !ok || (resource != "explain" && resource != "analyze") {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst wallet route not found", "", tierFromHeader(r)))
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", chain, tierFromHeader(r)))
		return
	}

	switch resource {
	case "analyze":
		s.handleInteractiveAnalystWalletAnalyze(w, r, principal, chain, address)
	default:
		s.handleAnalystWalletExplain(w, r, principal, chain, address)
	}
}

func (s *Server) handleAuthorizedAnalystEntityRoute(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", tierFromHeader(r)))
		return
	}

	entityKey, resource, ok := parseAuthorizedAnalystEntityRoutePath(r.URL.Path)
	if !ok || resource != "analyze" {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst entity route not found", "", tierFromHeader(r)))
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", tierFromHeader(r)))
		return
	}

	s.handleInteractiveAnalystEntityAnalyze(w, r, principal, entityKey)
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

func (s *Server) handleEntityRoute(w http.ResponseWriter, r *http.Request) {
	entityKey, ok := parseEntityRoutePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "entity route not found", "", tierFromHeader(r)))
		return
	}

	detail, err := s.entities.GetEntityInterpretation(r.Context(), entityKey)
	if err != nil {
		if errors.Is(err, service.ErrEntityInterpretationNotFound) {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", err.Error(), "", tierFromHeader(r)))
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "entity interpretation lookup failed", "", tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.EntityInterpretation]{
		Success: true,
		Data:    detail,
		Meta:    newMeta("", tierFromHeader(r), freshness("snapshot", 300)),
	})
}

func (s *Server) handleAnalystEntityRoute(w http.ResponseWriter, r *http.Request) {
	entityKey, ok := parseAnalystEntityRoutePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst entity route not found", "", tierFromHeader(r)))
		return
	}

	detail, err := s.entities.GetEntityInterpretation(r.Context(), entityKey)
	if err != nil {
		if errors.Is(err, service.ErrEntityInterpretationNotFound) {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", err.Error(), "", tierFromHeader(r)))
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "analyst entity interpretation lookup failed", "", tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.EntityInterpretation]{
		Success: true,
		Data:    detail,
		Meta:    newMeta("", tierFromHeader(r), freshness("analyst-snapshot", 300)),
	})
}

func (s *Server) handleAnalystWalletCounterparties(w http.ResponseWriter, r *http.Request, chain string, address string) {
	if s.analystTools == nil {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst tools unavailable", chain, tierFromHeader(r)))
		return
	}

	limit, err := parseFindingsFeedLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid analyst counterparty limit", chain, tierFromHeader(r)))
		return
	}
	minInteractions, err := parseOptionalPositiveInt(r.URL.Query().Get("min_interactions"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid analyst counterparty minimum interactions", chain, tierFromHeader(r)))
		return
	}

	payload, err := s.analystTools.GetWalletCounterparties(r.Context(), chain, address, limit, minInteractions)
	if err != nil {
		if errors.Is(err, service.ErrWalletSummaryNotFound) {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", err.Error(), chain, tierFromHeader(r)))
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "analyst counterparties lookup failed", chain, tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.AnalystCounterpartiesResponse]{
		Success: true,
		Data:    payload,
		Meta:    newMeta(chain, tierFromHeader(r), freshness("analyst-tool", 180)),
	})
}

func (s *Server) handleAnalystWalletGraph(w http.ResponseWriter, r *http.Request, chain string, address string) {
	if s.analystTools == nil {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst tools unavailable", chain, tierFromHeader(r)))
		return
	}

	depth, err := parseWalletGraphDepth(r.URL.Query().Get("depth"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid analyst graph depth", chain, tierFromHeader(r)))
		return
	}

	graph, err := s.analystTools.GetWalletGraphEvidence(r.Context(), chain, address, depth, tierFromHeader(r))
	if err != nil {
		if errors.Is(err, service.ErrWalletGraphNotFound) {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", err.Error(), chain, tierFromHeader(r)))
			return
		}
		if errors.Is(err, service.ErrWalletGraphDepthNotAllowed) {
			writeJSON(w, http.StatusForbidden, errorEnvelope("FORBIDDEN", err.Error(), chain, tierFromHeader(r)))
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "analyst graph lookup failed", chain, tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[domain.WalletGraph]{
		Success: true,
		Data:    graph,
		Meta:    newMeta(chain, tierFromHeader(r), freshness("analyst-tool", 180)),
	})
}

func (s *Server) handleAnalystBehaviorPatterns(w http.ResponseWriter, r *http.Request, chain string, address string) {
	if s.analystTools == nil {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst tools unavailable", chain, tierFromHeader(r)))
		return
	}

	payload, err := s.analystTools.DetectBehaviorPatterns(r.Context(), chain, address)
	if err != nil {
		if errors.Is(err, service.ErrWalletSummaryNotFound) {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", err.Error(), chain, tierFromHeader(r)))
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "analyst behavior pattern lookup failed", chain, tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.AnalystBehaviorPatternsResponse]{
		Success: true,
		Data:    payload,
		Meta:    newMeta(chain, tierFromHeader(r), freshness("analyst-tool", 180)),
	})
}

func (s *Server) handleAnalystFindingDetail(w http.ResponseWriter, r *http.Request, findingID string) {
	if s.analystFindings == nil {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst finding drilldown unavailable", "", tierFromHeader(r)))
		return
	}
	payload, err := s.analystFindings.GetFindingDetail(r.Context(), findingID)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL"
		if errors.Is(err, service.ErrFindingNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		writeJSON(w, status, errorEnvelope(code, "analyst finding detail lookup failed", "", tierFromHeader(r)))
		return
	}
	writeJSON(w, http.StatusOK, Envelope[service.AnalystFindingDetail]{
		Success: true,
		Data:    payload,
		Meta:    newMeta("", tierFromHeader(r), freshness("analyst-tool", 180)),
	})
}

func (s *Server) handleAnalystFindingEvidenceTimeline(w http.ResponseWriter, r *http.Request, findingID string) {
	if s.analystFindings == nil {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst finding drilldown unavailable", "", tierFromHeader(r)))
		return
	}
	payload, err := s.analystFindings.GetEvidenceTimeline(r.Context(), findingID)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL"
		if errors.Is(err, service.ErrFindingNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		writeJSON(w, status, errorEnvelope(code, "analyst finding evidence timeline lookup failed", "", tierFromHeader(r)))
		return
	}
	writeJSON(w, http.StatusOK, Envelope[service.AnalystFindingEvidenceTimeline]{
		Success: true,
		Data:    payload,
		Meta:    newMeta(payload.Chain, tierFromHeader(r), freshness("analyst-tool", 180)),
	})
}

func (s *Server) handleAnalystFindingHistoricalAnalogs(w http.ResponseWriter, r *http.Request, findingID string) {
	if s.analystFindings == nil {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst finding drilldown unavailable", "", tierFromHeader(r)))
		return
	}
	limit, err := parseFindingsFeedLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid analyst historical analog limit", "", tierFromHeader(r)))
		return
	}
	payload, err := s.analystFindings.GetHistoricalAnalogs(r.Context(), findingID, limit)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL"
		if errors.Is(err, service.ErrFindingNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		writeJSON(w, status, errorEnvelope(code, "analyst historical analog lookup failed", "", tierFromHeader(r)))
		return
	}
	writeJSON(w, http.StatusOK, Envelope[service.AnalystHistoricalAnalogs]{
		Success: true,
		Data:    payload,
		Meta:    newMeta("", tierFromHeader(r), freshness("analyst-tool", 180)),
	})
}

func (s *Server) handleAnalystFindingExplain(
	w http.ResponseWriter,
	r *http.Request,
	principal auth.ClerkPrincipal,
	findingID string,
) {
	if s.analystExplain == nil {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst explanation service unavailable", "", tierFromHeader(r)))
		return
	}

	var req service.AnalystFindingExplainRequest
	if err := decodeJSONBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid analyst explanation payload", "", tierFromHeader(r)))
		return
	}

	payload, err := s.analystExplain.ExplainFinding(r.Context(), principal, findingID, req)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL"
		switch {
		case errors.Is(err, service.ErrFindingNotFound):
			status = http.StatusNotFound
			code = "NOT_FOUND"
		case errors.Is(err, service.ErrAnalystExplanationInvalidRequest):
			status = http.StatusBadRequest
			code = "INVALID_ARGUMENT"
		case errors.Is(err, service.ErrAnalystExplanationQuotaExceeded):
			status = http.StatusTooManyRequests
			code = "RATE_LIMITED"
		}
		writeJSON(w, status, errorEnvelope(code, "analyst finding explanation failed", "", tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.AnalystFindingExplanation]{
		Success: true,
		Data:    payload,
		Meta:    newMeta("", tierFromHeader(r), freshness("analyst-tool", 180)),
	})
}

func (s *Server) handleAnalystWalletExplain(
	w http.ResponseWriter,
	r *http.Request,
	principal auth.ClerkPrincipal,
	chain string,
	address string,
) {
	if s.analystExplain == nil {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "analyst explanation service unavailable", chain, tierFromHeader(r)))
		return
	}

	var req service.AnalystWalletExplainRequest
	if err := decodeJSONBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid analyst wallet explanation payload", chain, tierFromHeader(r)))
		return
	}

	payload, err := s.analystExplain.ExplainWallet(r.Context(), principal, chain, address, req)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL"
		switch {
		case errors.Is(err, service.ErrWalletSummaryNotFound):
			status = http.StatusNotFound
			code = "NOT_FOUND"
		case errors.Is(err, service.ErrAnalystExplanationInvalidRequest):
			status = http.StatusBadRequest
			code = "INVALID_ARGUMENT"
		case errors.Is(err, service.ErrAnalystExplanationQuotaExceeded):
			status = http.StatusTooManyRequests
			code = "RATE_LIMITED"
		}
		writeJSON(w, status, errorEnvelope(code, "analyst wallet explanation failed", chain, tierFromHeader(r)))
		return
	}

	httpStatus := http.StatusOK
	if payload.Queued {
		httpStatus = http.StatusAccepted
	}
	writeJSON(w, httpStatus, Envelope[service.AnalystWalletExplanation]{
		Success: true,
		Data:    payload,
		Meta:    newMeta(chain, tierFromHeader(r), freshness("analyst-tool", 180)),
	})
}

func (s *Server) handleInteractiveAnalystWalletAnalyze(
	w http.ResponseWriter,
	r *http.Request,
	principal auth.ClerkPrincipal,
	chain string,
	address string,
) {
	if s.interactiveAnalyst == nil {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "interactive analyst unavailable", chain, tierFromHeader(r)))
		return
	}

	var req service.InteractiveAnalystWalletRequest
	if err := decodeJSONBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid interactive analyst payload", chain, tierFromHeader(r)))
		return
	}

	payload, err := s.interactiveAnalyst.AnalyzeWallet(r.Context(), chain, address, req)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL"
		switch {
		case errors.Is(err, service.ErrWalletSummaryNotFound):
			status = http.StatusNotFound
			code = "NOT_FOUND"
		case errors.Is(err, service.ErrInteractiveAnalystInvalidRequest):
			status = http.StatusBadRequest
			code = "INVALID_ARGUMENT"
		}
		writeJSON(w, status, errorEnvelope(code, "interactive analyst wallet analysis failed", chain, tierFromHeader(r)))
		return
	}

	_ = principal
	writeJSON(w, http.StatusOK, Envelope[service.InteractiveAnalystWalletResponse]{
		Success: true,
		Data:    payload,
		Meta:    newMeta(chain, tierFromHeader(r), freshness("analyst-tool", 180)),
	})
}

func (s *Server) handleInteractiveAnalystEntityAnalyze(
	w http.ResponseWriter,
	r *http.Request,
	principal auth.ClerkPrincipal,
	entityKey string,
) {
	if s.interactiveAnalyst == nil {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "interactive analyst unavailable", "", tierFromHeader(r)))
		return
	}

	var req service.InteractiveAnalystEntityRequest
	if err := decodeJSONBody(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid interactive analyst entity payload", "", tierFromHeader(r)))
		return
	}

	payload, err := s.interactiveAnalyst.AnalyzeEntity(r.Context(), entityKey, req)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL"
		switch {
		case errors.Is(err, service.ErrEntityInterpretationNotFound):
			status = http.StatusNotFound
			code = "NOT_FOUND"
		case errors.Is(err, service.ErrInteractiveAnalystInvalidRequest):
			status = http.StatusBadRequest
			code = "INVALID_ARGUMENT"
		}
		writeJSON(w, status, errorEnvelope(code, "interactive analyst entity analysis failed", "", tierFromHeader(r)))
		return
	}

	_ = principal
	writeJSON(w, http.StatusOK, Envelope[service.InteractiveAnalystEntityResponse]{
		Success: true,
		Data:    payload,
		Meta:    newMeta("", tierFromHeader(r), freshness("analyst-tool", 180)),
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

func (s *Server) handleWalletBrief(w http.ResponseWriter, r *http.Request, chain string, address string) {
	if !domain.IsSupportedChain(domain.Chain(chain)) || len(address) < 16 {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid wallet brief request", chain, tierFromHeader(r)))
		return
	}

	brief, err := s.walletBriefs.GetWalletBrief(r.Context(), chain, address)
	if err != nil {
		if errors.Is(err, service.ErrWalletSummaryNotFound) {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", err.Error(), chain, tierFromHeader(r)))
			return
		}

		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "wallet brief lookup failed", chain, tierFromHeader(r)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.WalletBrief]{
		Success: true,
		Data:    brief,
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
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}
	if !s.ensureAdminPrincipalAccess(w, principal) {
		return
	}
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

	if resource != "summary" && resource != "brief" && resource != "graph" {
		return "", "", "", false
	}

	return chain, address, resource, true
}

func parseAnalystWalletRoutePath(path string) (string, string, string, bool) {
	rest, ok := strings.CutPrefix(path, "/v1/analyst/wallets/")
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

	if resource != "brief" && resource != "explain" && resource != "analyze" {
		return "", "", "", false
	}

	return chain, address, resource, true
}

func parseAnalystToolWalletRoutePath(path string) (string, string, string, bool) {
	rest, ok := strings.CutPrefix(path, "/v1/analyst/tools/wallets/")
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

	switch resource {
	case "counterparties", "graph", "behavior-patterns":
		return chain, address, resource, true
	default:
		return "", "", "", false
	}
}

func parseAnalystFindingRoutePath(path string) (string, string, bool) {
	rest, ok := strings.CutPrefix(path, "/v1/analyst/findings/")
	if !ok {
		return "", "", false
	}
	findingID, resource, hasResource := strings.Cut(strings.TrimSpace(rest), "/")
	if strings.TrimSpace(findingID) == "" {
		return "", "", false
	}
	if !hasResource {
		return strings.TrimSpace(findingID), "", true
	}
	resource = strings.TrimSpace(resource)
	switch resource {
	case "evidence-timeline", "historical-analogs", "explain":
		return strings.TrimSpace(findingID), resource, true
	default:
		return "", "", false
	}
}

func parseFindingsFeedLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 20, nil
	}

	limit, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || limit <= 0 {
		return 0, errors.New("findings feed limit must be positive")
	}
	if limit > 50 {
		limit = 50
	}

	return limit, nil
}

func parseFindingsTypes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseFindingsTypesFromQuery(values url.Values) []string {
	repeated := values["type"]
	if len(repeated) > 0 {
		out := make([]string, 0, len(repeated))
		for _, value := range repeated {
			out = append(out, parseFindingsTypes(value)...)
		}
		if len(out) > 0 {
			return out
		}
	}

	return parseFindingsTypes(values.Get("types"))
}

func parseOptionalPositiveInt(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0, errors.New("value must be zero or positive")
	}
	return value, nil
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

func parseEntityRoutePath(path string) (string, bool) {
	entityKey, ok := strings.CutPrefix(path, "/v1/entity/")
	if !ok {
		return "", false
	}
	entityKey = strings.TrimSpace(entityKey)
	if entityKey == "" || strings.Contains(entityKey, "/") {
		return "", false
	}
	return entityKey, true
}

func parseAnalystEntityRoutePath(path string) (string, bool) {
	entityKey, ok := strings.CutPrefix(path, "/v1/analyst/entity/")
	if !ok {
		return "", false
	}
	entityKey = strings.TrimSpace(entityKey)
	if entityKey == "" || strings.Contains(entityKey, "/") {
		return "", false
	}
	return entityKey, true
}

func parseAuthorizedAnalystEntityRoutePath(path string) (string, string, bool) {
	rest, ok := strings.CutPrefix(path, "/v1/analyst/entity/")
	if !ok {
		return "", "", false
	}

	entityKey, resource, ok := strings.Cut(rest, "/")
	if !ok {
		return "", "", false
	}

	entityKey = strings.TrimSpace(entityKey)
	resource = strings.TrimSpace(resource)
	if entityKey == "" || resource == "" || strings.Contains(entityKey, "/") {
		return "", "", false
	}
	if resource != "analyze" {
		return "", "", false
	}

	return entityKey, resource, true
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
	for _, headerName := range []string{
		"X-Qorvi-Plan",
		"X-Flowintel-Plan",
		"X-Whalegraph-Plan",
	} {
		switch tier := strings.ToLower(strings.TrimSpace(r.Header.Get(headerName))); tier {
		case "pro", "team":
			return tier
		}
	}

	return "free"
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
