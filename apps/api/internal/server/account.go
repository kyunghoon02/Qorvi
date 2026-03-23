package server

import (
	"net/http"
	"strings"

	"github.com/whalegraph/whalegraph/apps/api/internal/auth"
	"github.com/whalegraph/whalegraph/apps/api/internal/service"
	"github.com/whalegraph/whalegraph/packages/domain"
)

func (s *Server) handleAccount(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}

	tier := domain.PlanTier(tierFromHeader(r))
	if s.billing != nil {
		tier = s.billing.ResolvePlan(r.Context(), principal.UserID, tier)
	}
	account, err := s.account.GetAccount(r.Context(), principal, tier)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "account lookup failed", "", string(tier)))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.AccountResponse]{
		Success: true,
		Data:    account,
		Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
	})
}
