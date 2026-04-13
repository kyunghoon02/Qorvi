package server

import (
	"net/http"
	"strings"

	"github.com/qorvi/qorvi/apps/api/internal/auth"
	"github.com/qorvi/qorvi/apps/api/internal/service"
)

func (s *Server) handleAdminBacktests(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}
	if !s.ensureAdminPrincipalAccess(w, principal) {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", "admin"))
		return
	}
	preview, err := s.adminBacktests.Preview(r.Context(), principal.Role)
	if err != nil {
		writeAdminConsoleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope[service.AdminBacktestOpsPreview]{
		Success: true,
		Data:    preview,
		Meta:    newMeta("", "admin", freshness("snapshot", 120)),
	})
}

func (s *Server) handleAdminBacktestRoute(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}
	if !s.ensureAdminPrincipalAccess(w, principal) {
		return
	}
	key, suffix, _, ok := parseNestedAdminResourcePath(r.URL.Path, "/v1/admin/backtests/")
	if !ok || suffix != "run" || strings.TrimSpace(key) == "" {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "admin backtest route not found", "", "admin"))
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", "admin"))
		return
	}
	result, err := s.adminBacktests.Run(r.Context(), principal.Role, key)
	if err != nil {
		if err == service.ErrAdminBacktestOpNotFound {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "admin backtest operation not found", "", "admin"))
			return
		}
		writeAdminConsoleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope[service.AdminBacktestRunResult]{
		Success: true,
		Data:    result,
		Meta:    newMeta("", "admin", freshness("snapshot", 30)),
	})
}
