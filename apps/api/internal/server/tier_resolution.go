package server

import (
	"net/http"
	"strings"

	"github.com/flowintel/flowintel/packages/domain"
)

func (s *Server) resolveAuthorizedTier(r *http.Request, ownerUserID string) domain.PlanTier {
	fallback := domain.PlanTier(tierFromHeader(r))
	if s == nil || s.billing == nil || strings.TrimSpace(ownerUserID) == "" {
		return fallback
	}

	return s.billing.ResolvePlan(r.Context(), ownerUserID, fallback)
}
