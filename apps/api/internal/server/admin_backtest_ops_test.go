package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/qorvi/qorvi/apps/api/internal/auth"
	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/apps/api/internal/service"
)

func TestAdminBacktestRoutesRequireAllowlistedAdmin(t *testing.T) {
	t.Setenv("QORVI_ADMIN_ALLOWLIST_USER_IDS", "admin_1")
	tempDir := t.TempDir()
	presetPath := filepath.Join(tempDir, "query-presets.json")
	if err := os.WriteFile(presetPath, []byte(`{"version":"1","presets":[{"name":"bridge","queryName":"qorvi_backtest_evm_known_negative_bridge_return_v1","sqlPath":"queries/dune/backtest/01_bridge_return_negative.sql","cohort":"known_negative","caseType":"bridge_return","chain":"evm","candidateOutput":"packages/intelligence/test/out.json","parameters":{"window_start":"2026-03-01T00:00:00Z","window_end":"2026-03-02T00:00:00Z","min_bridge_usd":25000,"max_return_hours":48,"post_return_hours":24,"max_post_return_recipients":3,"max_post_return_outbound_usd":50000,"limit":100,"source_url":"https://dune.com/query/1"}}]}`), 0o644); err != nil {
		t.Fatalf("write preset: %v", err)
	}

	srv := NewWithDependencies(Dependencies{
		AdminConsole:   service.NewAdminConsoleService(repository.NewInMemoryAdminConsoleRepository()),
		AdminBacktests: service.NewAdminBacktestOpsService("", presetPath, ""),
		ClerkVerifier:  auth.NewHeaderClerkVerifier(),
	})

	forbidden := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/backtests", nil)
	req.Header.Set("X-Clerk-User-Id", "admin_2")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")
	srv.Handler().ServeHTTP(forbidden, req)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden status 403, got %d", forbidden.Code)
	}

	allowed := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/admin/backtests/dune_query_presets_validate/run", nil)
	req.Header.Set("X-Clerk-User-Id", "admin_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")
	srv.Handler().ServeHTTP(allowed, req)
	if allowed.Code != http.StatusOK {
		t.Fatalf("expected allowed status 200, got %d", allowed.Code)
	}
}
