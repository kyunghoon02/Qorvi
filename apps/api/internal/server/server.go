package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/auth"
	"github.com/whalegraph/whalegraph/apps/api/internal/repository"
	"github.com/whalegraph/whalegraph/apps/api/internal/service"
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

type Server struct {
	mux     *http.ServeMux
	wallets *service.WalletSummaryService
	graphs  *service.WalletGraphService
	search  *service.SearchService
}

type Dependencies struct {
	Wallets       *service.WalletSummaryService
	Graphs        *service.WalletGraphService
	Search        *service.SearchService
	ClerkVerifier auth.ClerkVerifier
}

func New() *Server {
	return NewWithDependencies(Dependencies{})
}

func NewWithDependencies(deps Dependencies) *Server {
	if deps.Wallets == nil {
		deps.Wallets = service.NewWalletSummaryService(
			repository.NewQueryBackedWalletSummaryRepository(notFoundWalletSummaryLoader{}),
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
	if deps.ClerkVerifier == nil {
		deps.ClerkVerifier = auth.NewHeaderClerkVerifier()
	}

	mux := http.NewServeMux()
	s := &Server{
		mux:     mux,
		wallets: deps.Wallets,
		graphs:  deps.Graphs,
		search:  deps.Search,
	}

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /v1/search", s.handleSearch)
	mux.HandleFunc("GET /v1/wallets/", s.handleWalletRoute)
	mux.Handle(
		"GET /v1/admin/status",
		auth.RequireClerkRole(deps.ClerkVerifier, apiAuthResponder{}, "admin", "operator")(http.HandlerFunc(s.handleAdminStatus)),
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

	writeJSON(w, http.StatusOK, Envelope[service.SearchResponse]{
		Success: true,
		Data:    s.search.Search(r.Context(), query),
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
