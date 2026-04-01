package server

import (
	"net/http"
	"os"
	"strings"

	"github.com/qorvi/qorvi/apps/api/internal/auth"
)

type adminPrincipalAllowlist struct {
	userIDs map[string]struct{}
	emails  map[string]struct{}
}

func loadAdminPrincipalAllowlistFromEnv() adminPrincipalAllowlist {
	return adminPrincipalAllowlist{
		userIDs: parseAdminAllowlistValues(os.Getenv("QORVI_ADMIN_ALLOWLIST_USER_IDS")),
		emails:  parseAdminAllowlistValues(os.Getenv("QORVI_ADMIN_ALLOWLIST_EMAILS")),
	}
}

func parseAdminAllowlistValues(raw string) map[string]struct{} {
	values := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		trimmed := strings.ToLower(strings.TrimSpace(part))
		if trimmed == "" {
			continue
		}
		values[trimmed] = struct{}{}
	}
	return values
}

func (a adminPrincipalAllowlist) allows(principal auth.ClerkPrincipal) bool {
	if len(a.userIDs) == 0 && len(a.emails) == 0 {
		return true
	}
	if _, ok := a.userIDs[strings.ToLower(strings.TrimSpace(principal.UserID))]; ok {
		return true
	}
	if _, ok := a.emails[strings.ToLower(strings.TrimSpace(principal.Email))]; ok {
		return true
	}
	return false
}

func (s *Server) ensureAdminPrincipalAccess(
	w http.ResponseWriter,
	principal auth.ClerkPrincipal,
) bool {
	if s.adminAllowlist.allows(principal) {
		return true
	}
	writeJSON(w, http.StatusForbidden, errorEnvelope("FORBIDDEN", "admin console access is forbidden", "", "admin"))
	return false
}
