package server

import (
	"net/http"
	"strings"

	"github.com/qorvi/qorvi/apps/api/internal/auth"
	"github.com/qorvi/qorvi/apps/api/internal/service"
)

func (s *Server) handleAdminLabels(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}
	if !s.ensureAdminPrincipalAccess(w, principal) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		page, err := s.adminConsole.ListLabels(r.Context(), principal.Role)
		if err != nil {
			writeAdminConsoleError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, Envelope[service.AdminLabelCollection]{
			Success: true,
			Data:    page,
			Meta:    newMeta("", "admin", freshness("snapshot", 300)),
		})
	case http.MethodPost:
		var req service.UpsertAdminLabelRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid admin label payload", "", "admin"))
			return
		}
		item, err := s.adminConsole.UpsertLabel(r.Context(), principal.Role, principal.UserID, req)
		if err != nil {
			writeAdminConsoleError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, Envelope[service.AdminLabelSummary]{
			Success: true,
			Data:    item,
			Meta:    newMeta("", "admin", freshness("snapshot", 300)),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", "admin"))
	}
}

func (s *Server) handleAdminLabelRoute(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}
	if !s.ensureAdminPrincipalAccess(w, principal) {
		return
	}
	name, ok := parseAdminResourcePath(r.URL.Path, "/v1/admin/labels/")
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "admin label route not found", "", "admin"))
		return
	}
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", "admin"))
		return
	}
	if err := s.adminConsole.DeleteLabel(r.Context(), principal.Role, principal.UserID, name); err != nil {
		writeAdminConsoleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope[service.AdminMutationResult]{
		Success: true,
		Data:    service.AdminMutationResult{Deleted: true},
		Meta:    newMeta("", "admin", freshness("snapshot", 300)),
	})
}

func (s *Server) handleAdminSuppressions(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}
	if !s.ensureAdminPrincipalAccess(w, principal) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		page, err := s.adminConsole.ListSuppressions(r.Context(), principal.Role)
		if err != nil {
			writeAdminConsoleError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, Envelope[service.AdminSuppressionCollection]{
			Success: true,
			Data:    page,
			Meta:    newMeta("", "admin", freshness("snapshot", 300)),
		})
	case http.MethodPost:
		var req service.CreateAdminSuppressionRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid suppression payload", "", "admin"))
			return
		}
		item, err := s.adminConsole.CreateSuppression(r.Context(), principal.Role, principal.UserID, req)
		if err != nil {
			writeAdminConsoleError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, Envelope[service.AdminSuppressionSummary]{
			Success: true,
			Data:    item,
			Meta:    newMeta("", "admin", freshness("snapshot", 300)),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", "admin"))
	}
}

func (s *Server) handleAdminSuppressionRoute(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}
	if !s.ensureAdminPrincipalAccess(w, principal) {
		return
	}
	id, ok := parseAdminResourcePath(r.URL.Path, "/v1/admin/suppressions/")
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "suppression route not found", "", "admin"))
		return
	}
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", "admin"))
		return
	}
	if err := s.adminConsole.DeleteSuppression(r.Context(), principal.Role, principal.UserID, id); err != nil {
		writeAdminConsoleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope[service.AdminMutationResult]{
		Success: true,
		Data:    service.AdminMutationResult{Deleted: true},
		Meta:    newMeta("", "admin", freshness("snapshot", 300)),
	})
}

func (s *Server) handleAdminProviderQuotas(w http.ResponseWriter, r *http.Request) {
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
	page, err := s.adminConsole.ListProviderQuotas(r.Context(), principal.Role)
	if err != nil {
		writeAdminConsoleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope[service.AdminProviderQuotaCollection]{
		Success: true,
		Data:    page,
		Meta:    newMeta("", "admin", freshness("snapshot", 300)),
	})
}

func (s *Server) handleAdminObservability(w http.ResponseWriter, r *http.Request) {
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
	page, err := s.adminConsole.ListObservability(r.Context(), principal.Role)
	if err != nil {
		writeAdminConsoleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope[service.AdminObservabilityCollection]{
		Success: true,
		Data:    page,
		Meta:    newMeta("", "admin", freshness("snapshot", 120)),
	})
}

func (s *Server) handleAdminCuratedLists(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}
	if !s.ensureAdminPrincipalAccess(w, principal) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		page, err := s.adminConsole.ListCuratedLists(r.Context(), principal.Role)
		if err != nil {
			writeAdminConsoleError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, Envelope[service.AdminCuratedListCollection]{
			Success: true,
			Data:    page,
			Meta:    newMeta("", "admin", freshness("snapshot", 300)),
		})
	case http.MethodPost:
		var req service.CreateAdminCuratedListRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid curated list payload", "", "admin"))
			return
		}
		item, err := s.adminConsole.CreateCuratedList(r.Context(), principal.Role, principal.UserID, req)
		if err != nil {
			writeAdminConsoleError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, Envelope[service.AdminCuratedListDetail]{
			Success: true,
			Data:    item,
			Meta:    newMeta("", "admin", freshness("snapshot", 300)),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", "admin"))
	}
}

func (s *Server) handleAdminCuratedListRoute(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}
	if !s.ensureAdminPrincipalAccess(w, principal) {
		return
	}
	listID, resource, itemID, ok := parseNestedAdminResourcePath(r.URL.Path, "/v1/admin/curated-lists/")
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "curated list route not found", "", "admin"))
		return
	}
	switch resource {
	case "":
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", "admin"))
			return
		}
		if err := s.adminConsole.DeleteCuratedList(r.Context(), principal.Role, principal.UserID, listID); err != nil {
			writeAdminConsoleError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, Envelope[service.AdminMutationResult]{
			Success: true,
			Data:    service.AdminMutationResult{Deleted: true},
			Meta:    newMeta("", "admin", freshness("snapshot", 300)),
		})
	case "items":
		switch r.Method {
		case http.MethodPost:
			var req service.CreateAdminCuratedListItemRequest
			if err := decodeJSONBody(r, &req); err != nil {
				writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid curated list item payload", "", "admin"))
				return
			}
			item, err := s.adminConsole.AddCuratedListItem(r.Context(), principal.Role, principal.UserID, listID, req)
			if err != nil {
				writeAdminConsoleError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, Envelope[service.AdminCuratedListDetail]{
				Success: true,
				Data:    item,
				Meta:    newMeta("", "admin", freshness("snapshot", 300)),
			})
		case http.MethodDelete:
			if strings.TrimSpace(itemID) == "" {
				writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "curated list item route not found", "", "admin"))
				return
			}
			item, err := s.adminConsole.DeleteCuratedListItem(r.Context(), principal.Role, principal.UserID, listID, itemID)
			if err != nil {
				writeAdminConsoleError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, Envelope[service.AdminCuratedListDetail]{
				Success: true,
				Data:    item,
				Meta:    newMeta("", "admin", freshness("snapshot", 300)),
			})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", "admin"))
		}
	default:
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "curated list route not found", "", "admin"))
	}
}

func (s *Server) handleAdminAuditLogs(w http.ResponseWriter, r *http.Request) {
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
	page, err := s.adminConsole.ListAuditEntries(r.Context(), principal.Role, 20)
	if err != nil {
		writeAdminConsoleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope[service.AdminAuditCollection]{
		Success: true,
		Data:    page,
		Meta:    newMeta("", "admin", freshness("snapshot", 300)),
	})
}

func parseAdminResourcePath(path string, prefix string) (string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSuffix(path, "/"), prefix)
	if !ok || strings.TrimSpace(rest) == "" || strings.Contains(rest, "/") {
		return "", false
	}
	return rest, true
}

func parseNestedAdminResourcePath(path string, prefix string) (string, string, string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSuffix(path, "/"), prefix)
	if !ok || strings.TrimSpace(rest) == "" {
		return "", "", "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) == 1 {
		return parts[0], "", "", true
	}
	if len(parts) == 2 {
		return parts[0], parts[1], "", true
	}
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2], true
	}
	return "", "", "", false
}

func writeAdminConsoleError(w http.ResponseWriter, err error) {
	switch err {
	case nil:
		return
	case service.ErrAdminConsoleForbidden:
		writeJSON(w, http.StatusForbidden, errorEnvelope("FORBIDDEN", "admin console access is forbidden", "", "admin"))
	case service.ErrAdminLabelNotFound, service.ErrAdminSuppressionNotFound, service.ErrAdminCuratedListNotFound:
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "admin resource not found", "", "admin"))
	case service.ErrAdminConsoleInvalidRequest:
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid admin console request", "", "admin"))
	default:
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "internal server error", "", "admin"))
	}
}
