package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/whalegraph/whalegraph/apps/api/internal/auth"
	"github.com/whalegraph/whalegraph/apps/api/internal/service"
	"github.com/whalegraph/whalegraph/packages/domain"
)

func (s *Server) handleAlertInbox(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}

	tier := s.resolveAuthorizedTier(r, principal.UserID)
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	inbox, err := s.alertDelivery.ListInboxEvents(r.Context(), principal.UserID, tier, service.AlertInboxQuery{
		Limit:      limit,
		Severity:   r.URL.Query().Get("severity"),
		SignalType: r.URL.Query().Get("signalType"),
		Cursor:     r.URL.Query().Get("cursor"),
		Status:     r.URL.Query().Get("status"),
	})
	if err != nil {
		writeAlertDeliveryError(w, err, tier)
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.AlertInboxCollection]{
		Success: true,
		Data:    inbox,
		Meta:    newMeta("", string(tier), freshness("snapshot", 120)),
	})
}

func (s *Server) handleAlertInboxRoute(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}

	tier := s.resolveAuthorizedTier(r, principal.UserID)
	eventID, ok := parseAlertInboxEventRoutePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "alert inbox route not found", "", string(tier)))
		return
	}

	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", string(tier)))
		return
	}

	var req service.UpdateAlertInboxEventRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid alert inbox payload", "", string(tier)))
		return
	}

	result, err := s.alertDelivery.UpdateInboxEvent(r.Context(), principal.UserID, tier, eventID, req)
	if err != nil {
		writeAlertDeliveryError(w, err, tier)
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.AlertInboxMutationResult]{
		Success: true,
		Data:    result,
		Meta:    newMeta("", string(tier), freshness("snapshot", 120)),
	})
}

func (s *Server) handleAlertDeliveryChannelCollection(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}

	tier := s.resolveAuthorizedTier(r, principal.UserID)
	switch r.Method {
	case http.MethodGet:
		page, err := s.alertDelivery.ListAlertDeliveryChannels(r.Context(), principal.UserID, tier)
		if err != nil {
			writeAlertDeliveryError(w, err, tier)
			return
		}
		writeJSON(w, http.StatusOK, Envelope[service.AlertDeliveryChannelCollection]{
			Success: true,
			Data:    page,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	case http.MethodPost:
		var req service.CreateAlertDeliveryChannelRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid alert delivery payload", "", string(tier)))
			return
		}
		channel, err := s.alertDelivery.CreateAlertDeliveryChannel(r.Context(), principal.UserID, tier, req)
		if err != nil {
			writeAlertDeliveryError(w, err, tier)
			return
		}
		writeJSON(w, http.StatusCreated, Envelope[service.AlertDeliveryChannelSummary]{
			Success: true,
			Data:    channel,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", string(tier)))
	}
}

func (s *Server) handleAlertDeliveryChannelRoute(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}

	tier := s.resolveAuthorizedTier(r, principal.UserID)
	channelID, ok := parseAlertDeliveryChannelRoutePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "alert delivery channel route not found", "", string(tier)))
		return
	}

	switch r.Method {
	case http.MethodGet:
		channel, err := s.alertDelivery.GetAlertDeliveryChannel(r.Context(), principal.UserID, tier, channelID)
		if err != nil {
			writeAlertDeliveryError(w, err, tier)
			return
		}
		writeJSON(w, http.StatusOK, Envelope[service.AlertDeliveryChannelSummary]{
			Success: true,
			Data:    channel,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	case http.MethodPatch:
		var req service.UpdateAlertDeliveryChannelRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid alert delivery payload", "", string(tier)))
			return
		}
		channel, err := s.alertDelivery.UpdateAlertDeliveryChannel(r.Context(), principal.UserID, tier, channelID, req)
		if err != nil {
			writeAlertDeliveryError(w, err, tier)
			return
		}
		writeJSON(w, http.StatusOK, Envelope[service.AlertDeliveryChannelSummary]{
			Success: true,
			Data:    channel,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	case http.MethodDelete:
		if err := s.alertDelivery.DeleteAlertDeliveryChannel(r.Context(), principal.UserID, tier, channelID); err != nil {
			writeAlertDeliveryError(w, err, tier)
			return
		}
		writeJSON(w, http.StatusOK, Envelope[service.AlertDeliveryMutationResult]{
			Success: true,
			Data:    service.AlertDeliveryMutationResult{Deleted: true},
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", string(tier)))
	}
}

func parseAlertDeliveryChannelRoutePath(path string) (string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSuffix(path, "/"), "/v1/alert-delivery-channels/")
	if !ok || strings.TrimSpace(rest) == "" {
		return "", false
	}
	if strings.Contains(rest, "/") {
		return "", false
	}
	return rest, true
}

func parseAlertInboxEventRoutePath(path string) (string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSuffix(path, "/"), "/v1/alerts/")
	if !ok || strings.TrimSpace(rest) == "" {
		return "", false
	}
	if strings.Contains(rest, "/") {
		return "", false
	}
	return rest, true
}

func writeAlertDeliveryError(w http.ResponseWriter, err error, tier domain.PlanTier) {
	switch {
	case err == nil:
		return
	case err == service.ErrAlertDeliveryForbidden:
		writeJSON(w, http.StatusForbidden, errorEnvelope("FORBIDDEN", "alert delivery feature is not available for this plan", "", string(tier)))
	case err == service.ErrAlertDeliveryNotFound:
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "alert delivery channel not found", "", string(tier)))
	case err == service.ErrAlertDeliveryConflict:
		writeJSON(w, http.StatusConflict, errorEnvelope("CONFLICT", "alert delivery channel conflict", "", string(tier)))
	case err == service.ErrAlertDeliveryLimitExceeded:
		writeJSON(w, http.StatusConflict, errorEnvelope("LIMIT_EXCEEDED", "alert delivery channel limit exceeded", "", string(tier)))
	case err == service.ErrAlertDeliveryInvalidRequest:
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid alert delivery request", "", string(tier)))
	default:
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "internal server error", "", string(tier)))
	}
}
