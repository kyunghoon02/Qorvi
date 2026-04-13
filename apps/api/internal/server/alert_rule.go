package server

import (
	"net/http"
	"strings"

	"github.com/qorvi/qorvi/apps/api/internal/auth"
	"github.com/qorvi/qorvi/apps/api/internal/service"
	"github.com/qorvi/qorvi/packages/domain"
)

func (s *Server) handleAlertRuleCollection(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}

	tier := s.resolveAuthorizedTier(r, principal.UserID)

	switch r.Method {
	case http.MethodGet:
		page, err := s.alertRules.ListAlertRules(r.Context(), principal.UserID, tier)
		if err != nil {
			writeAlertRuleError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusOK, Envelope[service.AlertRuleCollection]{
			Success: true,
			Data:    page,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	case http.MethodPost:
		var req service.CreateAlertRuleRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid alert rule payload", "", string(tier)))
			return
		}

		detail, err := s.alertRules.CreateAlertRule(r.Context(), principal.UserID, tier, req)
		if err != nil {
			writeAlertRuleError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusCreated, Envelope[service.AlertRuleDetail]{
			Success: true,
			Data:    detail,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", string(tier)))
	}
}

func (s *Server) handleAlertRuleRoute(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}

	tier := s.resolveAuthorizedTier(r, principal.UserID)
	ruleID, resource, ok := parseAlertRuleRoutePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "alert rule route not found", "", string(tier)))
		return
	}

	switch resource {
	case "":
		s.handleAlertRuleDetailRoute(w, r, principal.UserID, tier, ruleID)
	case "events":
		s.handleAlertRuleEventsRoute(w, r, principal.UserID, tier, ruleID)
	default:
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "alert rule route not found", "", string(tier)))
	}
}

func (s *Server) handleAlertRuleDetailRoute(w http.ResponseWriter, r *http.Request, ownerUserID string, tier domain.PlanTier, ruleID string) {
	switch r.Method {
	case http.MethodGet:
		detail, err := s.alertRules.GetAlertRule(r.Context(), ownerUserID, tier, ruleID)
		if err != nil {
			writeAlertRuleError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusOK, Envelope[service.AlertRuleDetail]{
			Success: true,
			Data:    detail,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	case http.MethodPatch:
		var req service.UpdateAlertRuleRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid alert rule payload", "", string(tier)))
			return
		}

		detail, err := s.alertRules.UpdateAlertRule(r.Context(), ownerUserID, tier, ruleID, req)
		if err != nil {
			writeAlertRuleError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusOK, Envelope[service.AlertRuleDetail]{
			Success: true,
			Data:    detail,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	case http.MethodDelete:
		if err := s.alertRules.DeleteAlertRule(r.Context(), ownerUserID, tier, ruleID); err != nil {
			writeAlertRuleError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusOK, Envelope[service.AlertRuleMutationResult]{
			Success: true,
			Data:    service.AlertRuleMutationResult{Deleted: true},
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", string(tier)))
	}
}

func (s *Server) handleAlertRuleEventsRoute(w http.ResponseWriter, r *http.Request, ownerUserID string, tier domain.PlanTier, ruleID string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", string(tier)))
		return
	}

	events, err := s.alertRules.ListAlertEvents(r.Context(), ownerUserID, tier, ruleID)
	if err != nil {
		writeAlertRuleError(w, err, tier)
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.AlertEventCollection]{
		Success: true,
		Data:    events,
		Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
	})
}

func parseAlertRuleRoutePath(path string) (string, string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSuffix(path, "/"), "/v1/alert-rules/")
	if !ok || rest == "" {
		return "", "", false
	}

	parts := strings.Split(rest, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return "", "", false
	}
	if len(parts) == 1 {
		return parts[0], "", true
	}
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		return parts[0], parts[1], true
	}
	return "", "", false
}

func writeAlertRuleError(w http.ResponseWriter, err error, tier domain.PlanTier) {
	switch {
	case err == nil:
		return
	case err == service.ErrAlertRuleForbidden:
		writeJSON(w, http.StatusForbidden, errorEnvelope("FORBIDDEN", "alert feature is not available for this plan", "", string(tier)))
	case err == service.ErrAlertRuleNotFound:
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "alert rule not found", "", string(tier)))
	case err == service.ErrAlertRuleLimitExceeded:
		writeJSON(w, http.StatusConflict, errorEnvelope("LIMIT_EXCEEDED", "alert rule limit exceeded", "", string(tier)))
	case err == service.ErrAlertRuleConflict:
		writeJSON(w, http.StatusConflict, errorEnvelope("CONFLICT", "alert rule conflict", "", string(tier)))
	case err == service.ErrAlertRuleInvalidRequest:
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid alert rule request", "", string(tier)))
	default:
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "internal server error", "", string(tier)))
	}
}
