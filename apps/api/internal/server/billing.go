package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/whalegraph/whalegraph/apps/api/internal/auth"
	"github.com/whalegraph/whalegraph/apps/api/internal/service"
)

func (s *Server) handleBillingCheckoutSession(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", tierFromHeader(r)))
		return
	}
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		writeJSON(w, http.StatusUnsupportedMediaType, errorEnvelope("INVALID_ARGUMENT", "content type must be application/json", "", tierFromHeader(r)))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	var req service.BillingCheckoutRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid checkout payload", "", tierFromHeader(r)))
		return
	}

	tier := s.resolveAuthorizedTier(r, principal.UserID)
	result, err := s.billing.CreateCheckoutSessionResponse(
		r.Context(),
		principal,
		tier,
		req,
	)
	if err != nil {
		writeBillingError(w, err, string(tier))
		return
	}

	writeJSON(w, http.StatusCreated, Envelope[service.BillingCheckoutResponse]{
		Success: true,
		Data:    result,
		Meta:    newMeta("", string(tier), freshness("live", 0)),
	})
}

func (s *Server) handleBillingPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", tierFromHeader(r)))
		return
	}

	result, err := s.billing.ListPlans(r.Context())
	if err != nil {
		writeBillingError(w, err, tierFromHeader(r))
		return
	}

	writeJSON(w, http.StatusOK, Envelope[service.BillingPlansResponse]{
		Success: true,
		Data:    result,
		Meta:    newMeta("", tierFromHeader(r), freshness("snapshot", 3600)),
	})
}

func (s *Server) handleStripeBillingWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", "free"))
		return
	}
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		writeJSON(w, http.StatusUnsupportedMediaType, errorEnvelope("INVALID_ARGUMENT", "content type must be application/json", "", "free"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	var req service.BillingWebhookRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid billing webhook payload", "", "free"))
		return
	}

	result, err := s.billing.ReconcileWebhook(r.Context(), req)
	if err != nil {
		writeBillingError(w, err, tierFromHeader(r))
		return
	}

	writeJSON(w, http.StatusAccepted, Envelope[service.BillingWebhookResponse]{
		Success: true,
		Data:    result,
		Meta:    newMeta("", tierFromHeader(r), freshness("live", 0)),
	})
}

func writeBillingError(w http.ResponseWriter, err error, tier string) {
	switch {
	case err == nil:
		return
	case errors.Is(err, service.ErrBillingInvalidRequest), errors.Is(err, service.ErrBillingPlanRequired):
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid billing request", "", tier))
	default:
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "billing request failed", "", tier))
	}
}
