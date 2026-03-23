package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/whalegraph/whalegraph/apps/api/internal/auth"
	"github.com/whalegraph/whalegraph/apps/api/internal/service"
	"github.com/whalegraph/whalegraph/packages/domain"
)

func (s *Server) handleWatchlistCollection(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}

	tier := s.resolveAuthorizedTier(r, principal.UserID)

	switch r.Method {
	case http.MethodGet:
		page, err := s.watchlists.ListWatchlists(r.Context(), principal.UserID, tier)
		if err != nil {
			writeWatchlistError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusOK, Envelope[service.WatchlistCollection]{
			Success: true,
			Data:    page,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	case http.MethodPost:
		var req service.CreateWatchlistRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid watchlist payload", "", string(tier)))
			return
		}

		detail, err := s.watchlists.CreateWatchlist(r.Context(), principal.UserID, tier, req)
		if err != nil {
			writeWatchlistError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusCreated, Envelope[service.WatchlistDetail]{
			Success: true,
			Data:    detail,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", string(tier)))
	}
}

func (s *Server) handleWatchlistRoute(w http.ResponseWriter, r *http.Request) {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.UserID) == "" {
		writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "clerk session is required", "", "free"))
		return
	}

	tier := s.resolveAuthorizedTier(r, principal.UserID)
	watchlistID, resource, itemID, ok := parseWatchlistRoutePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "watchlist route not found", "", string(tier)))
		return
	}

	switch resource {
	case "":
		s.handleWatchlistDetailRoute(w, r, principal.UserID, tier, watchlistID)
	case "items":
		s.handleWatchlistItemsRoute(w, r, principal.UserID, tier, watchlistID, itemID)
	default:
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "watchlist route not found", "", string(tier)))
	}
}

func (s *Server) handleWatchlistDetailRoute(w http.ResponseWriter, r *http.Request, ownerUserID string, tier domain.PlanTier, watchlistID string) {
	switch r.Method {
	case http.MethodGet:
		detail, err := s.watchlists.GetWatchlist(r.Context(), ownerUserID, tier, watchlistID)
		if err != nil {
			writeWatchlistError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusOK, Envelope[service.WatchlistDetail]{
			Success: true,
			Data:    detail,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	case http.MethodPatch:
		var req service.UpdateWatchlistRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid watchlist payload", "", string(tier)))
			return
		}

		detail, err := s.watchlists.UpdateWatchlist(r.Context(), ownerUserID, tier, watchlistID, req)
		if err != nil {
			writeWatchlistError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusOK, Envelope[service.WatchlistDetail]{
			Success: true,
			Data:    detail,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	case http.MethodDelete:
		if err := s.watchlists.DeleteWatchlist(r.Context(), ownerUserID, tier, watchlistID); err != nil {
			writeWatchlistError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusOK, Envelope[service.WatchlistMutationResult]{
			Success: true,
			Data:    service.WatchlistMutationResult{Deleted: true},
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", string(tier)))
	}
}

func (s *Server) handleWatchlistItemsRoute(w http.ResponseWriter, r *http.Request, ownerUserID string, tier domain.PlanTier, watchlistID string, itemID string) {
	switch r.Method {
	case http.MethodGet:
		if itemID != "" {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "watchlist route not found", "", string(tier)))
			return
		}

		detail, err := s.watchlists.GetWatchlist(r.Context(), ownerUserID, tier, watchlistID)
		if err != nil {
			writeWatchlistError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusOK, Envelope[service.WatchlistDetail]{
			Success: true,
			Data:    detail,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	case http.MethodPost:
		if itemID != "" {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "watchlist route not found", "", string(tier)))
			return
		}

		var req service.CreateWatchlistItemRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid watchlist item payload", "", string(tier)))
			return
		}

		detail, err := s.watchlists.AddWatchlistItem(r.Context(), ownerUserID, tier, watchlistID, req)
		if err != nil {
			writeWatchlistError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusCreated, Envelope[service.WatchlistDetail]{
			Success: true,
			Data:    detail,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	case http.MethodPatch:
		if itemID == "" {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "watchlist route not found", "", string(tier)))
			return
		}

		var req service.UpdateWatchlistItemRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid watchlist item payload", "", string(tier)))
			return
		}

		detail, err := s.watchlists.UpdateWatchlistItem(r.Context(), ownerUserID, tier, watchlistID, itemID, req)
		if err != nil {
			writeWatchlistError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusOK, Envelope[service.WatchlistDetail]{
			Success: true,
			Data:    detail,
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	case http.MethodDelete:
		if itemID == "" {
			writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", "watchlist route not found", "", string(tier)))
			return
		}

		if err := s.watchlists.DeleteWatchlistItem(r.Context(), ownerUserID, tier, watchlistID, itemID); err != nil {
			writeWatchlistError(w, err, tier)
			return
		}

		writeJSON(w, http.StatusOK, Envelope[service.WatchlistMutationResult]{
			Success: true,
			Data:    service.WatchlistMutationResult{Deleted: true},
			Meta:    newMeta("", string(tier), freshness("snapshot", 300)),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorEnvelope("METHOD_NOT_ALLOWED", "unsupported method", "", string(tier)))
	}
}

func parseWatchlistRoutePath(path string) (string, string, string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSuffix(path, "/"), "/v1/watchlists/")
	if !ok {
		return "", "", "", false
	}

	segments := strings.Split(rest, "/")
	if len(segments) < 1 || segments[0] == "" {
		return "", "", "", false
	}

	watchlistID := strings.TrimSpace(segments[0])
	if watchlistID == "" {
		return "", "", "", false
	}

	if len(segments) == 1 {
		return watchlistID, "", "", true
	}

	if len(segments) == 2 && segments[1] == "items" {
		return watchlistID, "items", "", true
	}

	if len(segments) == 3 && segments[1] == "items" && segments[2] != "" {
		return watchlistID, "items", segments[2], true
	}

	return "", "", "", false
}

func writeWatchlistError(w http.ResponseWriter, err error, tier domain.PlanTier) {
	switch {
	case errors.Is(err, service.ErrWatchlistNotFound):
		writeJSON(w, http.StatusNotFound, errorEnvelope("NOT_FOUND", err.Error(), "", string(tier)))
	case errors.Is(err, service.ErrWatchlistForbidden):
		writeJSON(w, http.StatusForbidden, errorEnvelope("FORBIDDEN", err.Error(), "", string(tier)))
	case errors.Is(err, service.ErrWatchlistLimitExceeded):
		writeJSON(w, http.StatusForbidden, errorEnvelope("FORBIDDEN", err.Error(), "", string(tier)))
	case errors.Is(err, service.ErrWatchlistConflict):
		writeJSON(w, http.StatusConflict, errorEnvelope("CONFLICT", err.Error(), "", string(tier)))
	case errors.Is(err, service.ErrWatchlistInvalidRequest):
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", err.Error(), "", string(tier)))
	default:
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "watchlist operation failed", "", string(tier)))
	}
}

func decodeJSONBody(r *http.Request, dst any) error {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return errors.New("empty body")
	}
	return json.Unmarshal(raw, dst)
}
