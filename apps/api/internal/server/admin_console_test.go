package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qorvi/qorvi/apps/api/internal/auth"
	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/apps/api/internal/service"
)

func TestAdminConsoleRoutesRequireAdminRole(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryAdminConsoleRepository()
	repo.SeedQuotaSnapshots([]repository.AdminQuotaSnapshot{{
		Provider:      "alchemy",
		Status:        "healthy",
		Limit:         5000,
		Used:          100,
		Reserved:      0,
		WindowStart:   time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC),
		LastCheckedAt: time.Date(2026, time.March, 21, 3, 0, 0, 0, time.UTC),
	}})
	repo.SeedObservabilitySnapshot(repository.AdminObservabilitySnapshot{
		ProviderUsage: []repository.AdminProviderUsageSnapshot{{
			Provider: "alchemy",
			Status:   "healthy",
			Used24h:  100,
		}},
		Ingest: repository.AdminIngestSnapshot{
			FreshnessSeconds: 60,
			LagStatus:        "healthy",
		},
	})
	repo.SeedDomesticPrelistingCandidates([]repository.AdminDomesticPrelistingCandidate{{
		Chain:                 "EVM",
		TokenAddress:          "0xabc",
		TokenSymbol:           "TEST",
		NormalizedAssetKey:    "test",
		TransferCount7d:       8,
		TransferCount24h:      3,
		ActiveWalletCount:     4,
		TrackedWalletCount:    2,
		TotalAmount:           "1000000",
		LargestTransferAmount: "250000",
		LatestObservedAt:      time.Date(2026, time.March, 21, 4, 0, 0, 0, time.UTC),
	}})
	srv := NewWithDependencies(Dependencies{
		AdminConsole:  service.NewAdminConsoleService(repo),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	quota := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/provider-quotas", nil)
	req.Header.Set("X-Clerk-User-Id", "operator_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "operator")
	srv.Handler().ServeHTTP(quota, req)
	if quota.Code != http.StatusOK {
		t.Fatalf("expected operator quota status 200, got %d", quota.Code)
	}

	observability := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/admin/observability", nil)
	req.Header.Set("X-Clerk-User-Id", "operator_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "operator")
	srv.Handler().ServeHTTP(observability, req)
	if observability.Code != http.StatusOK {
		t.Fatalf("expected operator observability status 200, got %d", observability.Code)
	}

	domestic := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/admin/domestic-prelisting-candidates", nil)
	req.Header.Set("X-Clerk-User-Id", "operator_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "operator")
	srv.Handler().ServeHTTP(domestic, req)
	if domestic.Code != http.StatusOK {
		t.Fatalf("expected operator domestic prelisting status 200, got %d", domestic.Code)
	}

	create := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/admin/labels", bytes.NewBufferString(`{
	  "name":"Bridge Router",
	  "description":"Known bridge route",
	  "color":"#0F766E"
	}`))
	req.Header.Set("X-Clerk-User-Id", "operator_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "operator")
	srv.Handler().ServeHTTP(create, req)
	if create.Code != http.StatusForbidden {
		t.Fatalf("expected operator create status 403, got %d", create.Code)
	}
}

func TestAdminConsoleRoutesSupportAdminCrud(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryAdminConsoleRepository()
	repo.SeedQuotaSnapshots([]repository.AdminQuotaSnapshot{{
		Provider:      "alchemy",
		Status:        "warning",
		Limit:         5000,
		Used:          3200,
		Reserved:      0,
		WindowStart:   time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC),
		LastCheckedAt: time.Date(2026, time.March, 21, 3, 0, 0, 0, time.UTC),
	}})
	repo.SeedObservabilitySnapshot(repository.AdminObservabilitySnapshot{
		ProviderUsage: []repository.AdminProviderUsageSnapshot{{
			Provider: "alchemy",
			Status:   "warning",
			Used24h:  3200,
		}},
		Ingest: repository.AdminIngestSnapshot{
			FreshnessSeconds: 90,
			LagStatus:        "healthy",
		},
		AlertDelivery: repository.AdminAlertDeliverySnapshot{
			Attempts24h:    5,
			Delivered24h:   4,
			Failed24h:      1,
			RetryableCount: 1,
		},
	})
	repo.SeedDomesticPrelistingCandidates([]repository.AdminDomesticPrelistingCandidate{{
		Chain:                     "EVM",
		TokenAddress:              "0xtoken",
		TokenSymbol:               "ALPHA",
		NormalizedAssetKey:        "alpha",
		TransferCount7d:           12,
		TransferCount24h:          5,
		ActiveWalletCount:         6,
		TrackedWalletCount:        3,
		DistinctCounterpartyCount: 9,
		TotalAmount:               "4200000",
		LargestTransferAmount:     "900000",
		LatestObservedAt:          time.Date(2026, time.March, 21, 5, 0, 0, 0, time.UTC),
	}})
	srv := NewWithDependencies(Dependencies{
		AdminConsole:  service.NewAdminConsoleService(repo),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	createLabel := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/labels", bytes.NewBufferString(`{
	  "name":"CEX Hot Wallet",
	  "description":"Known exchange wallet",
	  "color":"#F97316"
	}`))
	req.Header.Set("X-Clerk-User-Id", "admin_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")
	srv.Handler().ServeHTTP(createLabel, req)
	if createLabel.Code != http.StatusCreated {
		t.Fatalf("expected label status 201, got %d", createLabel.Code)
	}

	listLabels := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/admin/labels", nil)
	req.Header.Set("X-Clerk-User-Id", "admin_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")
	srv.Handler().ServeHTTP(listLabels, req)
	if listLabels.Code != http.StatusOK {
		t.Fatalf("expected labels status 200, got %d", listLabels.Code)
	}

	createSuppression := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/admin/suppressions", bytes.NewBufferString(`{
	  "scope":"wallet",
	  "target":"evm:0x123",
	  "reason":"Known treasury"
	}`))
	req.Header.Set("X-Clerk-User-Id", "admin_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")
	srv.Handler().ServeHTTP(createSuppression, req)
	if createSuppression.Code != http.StatusCreated {
		t.Fatalf("expected suppression status 201, got %d", createSuppression.Code)
	}

	createCurated := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/admin/curated-lists", bytes.NewBufferString(`{
	  "name":"Exchange Hot Wallets",
	  "notes":"Operator curated exchange cohort",
	  "tags":["exchange","wallet"]
	}`))
	req.Header.Set("X-Clerk-User-Id", "admin_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")
	srv.Handler().ServeHTTP(createCurated, req)
	if createCurated.Code != http.StatusCreated {
		t.Fatalf("expected curated list status 201, got %d", createCurated.Code)
	}

	var curatedBody Envelope[service.AdminCuratedListDetail]
	decode(t, createCurated.Body.Bytes(), &curatedBody)
	if !curatedBody.Success || curatedBody.Data.ID == "" {
		t.Fatalf("unexpected curated list body %#v", curatedBody)
	}

	addCuratedItem := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/admin/curated-lists/"+curatedBody.Data.ID+"/items", bytes.NewBufferString(`{
	  "itemType":"wallet",
	  "itemKey":"evm:0x123",
	  "tags":["priority"],
	  "notes":"Seed operator focus"
	}`))
	req.Header.Set("X-Clerk-User-Id", "admin_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")
	srv.Handler().ServeHTTP(addCuratedItem, req)
	if addCuratedItem.Code != http.StatusCreated {
		t.Fatalf("expected curated list item status 201, got %d", addCuratedItem.Code)
	}

	listCurated := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/admin/curated-lists", nil)
	req.Header.Set("X-Clerk-User-Id", "admin_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")
	srv.Handler().ServeHTTP(listCurated, req)
	if listCurated.Code != http.StatusOK {
		t.Fatalf("expected curated list index status 200, got %d", listCurated.Code)
	}

	listAudit := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/admin/audit-logs", nil)
	req.Header.Set("X-Clerk-User-Id", "admin_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")
	srv.Handler().ServeHTTP(listAudit, req)
	if listAudit.Code != http.StatusOK {
		t.Fatalf("expected audit log status 200, got %d", listAudit.Code)
	}

	listObservability := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/admin/observability", nil)
	req.Header.Set("X-Clerk-User-Id", "admin_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")
	srv.Handler().ServeHTTP(listObservability, req)
	if listObservability.Code != http.StatusOK {
		t.Fatalf("expected observability status 200, got %d", listObservability.Code)
	}

	listDomestic := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/admin/domestic-prelisting-candidates", nil)
	req.Header.Set("X-Clerk-User-Id", "admin_1")
	req.Header.Set("X-Clerk-Session-Id", "session_1")
	req.Header.Set("X-Clerk-Role", "admin")
	srv.Handler().ServeHTTP(listDomestic, req)
	if listDomestic.Code != http.StatusOK {
		t.Fatalf("expected domestic prelisting status 200, got %d", listDomestic.Code)
	}
}
