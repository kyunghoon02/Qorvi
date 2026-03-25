package server

import (
	"context"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flowintel/flowintel/apps/api/internal/auth"
	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/apps/api/internal/service"
	"github.com/flowintel/flowintel/packages/billing"
	"github.com/flowintel/flowintel/packages/domain"
)

func TestWatchlistRoutesRequireAuthAndPlan(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		Watchlists:    service.NewWatchlistService(repository.NewInMemoryWatchlistRepository()),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/watchlists", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}

	forbidden := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/watchlists", nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	srv.Handler().ServeHTTP(forbidden, req)

	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", forbidden.Code)
	}
}

func TestWatchlistRoutesSupportCRUDForProOwner(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		Watchlists:    service.NewWatchlistService(repository.NewInMemoryWatchlistRepository()),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	create := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/watchlists", bytes.NewBufferString(`{"name":"Ops Watchlist"}`))
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(create, req)

	if create.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", create.Code)
	}

	var created Envelope[service.WatchlistDetail]
	decode(t, create.Body.Bytes(), &created)
	if created.Data.Name != "Ops Watchlist" {
		t.Fatalf("unexpected watchlist name %s", created.Data.Name)
	}

	itemCreate := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/watchlists/"+created.Data.ID+"/items", bytes.NewBufferString(`{"chain":"evm","address":"0x1234567890abcdef1234567890abcdef12345678","tags":["seed","hot"],"note":"primary counterparty"}`))
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(itemCreate, req)

	if itemCreate.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", itemCreate.Code)
	}

	var itemCreated Envelope[service.WatchlistDetail]
	decode(t, itemCreate.Body.Bytes(), &itemCreated)
	if itemCreated.Data.ItemCount != 1 {
		t.Fatalf("expected one item, got %d", itemCreated.Data.ItemCount)
	}
	if len(itemCreated.Data.Items) != 1 {
		t.Fatalf("expected one item payload, got %d", len(itemCreated.Data.Items))
	}
	if itemCreated.Data.Items[0].Chain != "evm" || itemCreated.Data.Items[0].Address != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected item %#v", itemCreated.Data.Items[0])
	}

	get := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/watchlists/"+created.Data.ID, nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(get, req)

	if get.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", get.Code)
	}

	var detail Envelope[service.WatchlistDetail]
	decode(t, get.Body.Bytes(), &detail)
	if detail.Data.ItemCount != 1 {
		t.Fatalf("expected one item in detail, got %d", detail.Data.ItemCount)
	}

	updateItem := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/v1/watchlists/"+created.Data.ID+"/items/"+itemCreated.Data.Items[0].ID, bytes.NewBufferString(`{"tags":["updated"],"note":"updated note"}`))
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(updateItem, req)

	if updateItem.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", updateItem.Code)
	}

	var updated Envelope[service.WatchlistDetail]
	decode(t, updateItem.Body.Bytes(), &updated)
	if updated.Data.Items[0].Note != "updated note" {
		t.Fatalf("unexpected updated note %q", updated.Data.Items[0].Note)
	}

	deleteItem := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/v1/watchlists/"+created.Data.ID+"/items/"+itemCreated.Data.Items[0].ID, nil)
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	req.Header.Set("X-Whalegraph-Plan", "pro")
	srv.Handler().ServeHTTP(deleteItem, req)

	if deleteItem.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", deleteItem.Code)
	}

	var deleted Envelope[service.WatchlistMutationResult]
	decode(t, deleteItem.Body.Bytes(), &deleted)
	if !deleted.Data.Deleted {
		t.Fatal("expected deleted true")
	}
}

func TestWatchlistRoutesResolvePersistedBillingPlan(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryBillingRepository()
	_, err := repo.UpsertBillingAccount(context.Background(), repository.BillingAccount{
		OwnerUserID: "user_123",
		CurrentTier: domain.PlanPro,
		Status:      "active",
	})
	if err != nil {
		t.Fatalf("upsert billing account: %v", err)
	}

	srv := NewWithDependencies(Dependencies{
		Watchlists:    service.NewWatchlistService(repository.NewInMemoryWatchlistRepository()),
		Billing:       service.NewBillingService(repo, billing.StripeConfig{}),
		ClerkVerifier: auth.NewHeaderClerkVerifier(),
	})

	create := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/watchlists", bytes.NewBufferString(`{"name":"Persisted plan watchlist"}`))
	req.Header.Set("X-Clerk-User-Id", "user_123")
	req.Header.Set("X-Clerk-Session-Id", "session_123")
	req.Header.Set("X-Clerk-Role", "user")
	srv.Handler().ServeHTTP(create, req)

	if create.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", create.Code)
	}
}
