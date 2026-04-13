package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qorvi/qorvi/apps/api/internal/service"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeDiscoverSeedReader struct {
	items []db.CuratedWalletSeed
	err   error
}

func (r *fakeDiscoverSeedReader) ListAdminCuratedWalletSeeds(_ context.Context) ([]db.CuratedWalletSeed, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]db.CuratedWalletSeed(nil), r.items...), nil
}

func TestDiscoverFeaturedWalletsRouteReturnsAdminCuratedItems(t *testing.T) {
	t.Parallel()

	srv := NewWithDependencies(Dependencies{
		Discover: service.NewDiscoverService(&fakeDiscoverSeedReader{
			items: []db.CuratedWalletSeed{
				{
					Chain:     domain.ChainEVM,
					Address:   "0x1111111111111111111111111111111111111111",
					ListTags:  []string{"admin-curated", "wallet-seeds", "exchange"},
					ItemTags:  []string{"featured", "verified-public", "exchange"},
					ItemNotes: "Public explorer-labeled exchange wallet.",
					UpdatedAt: time.Date(2026, time.March, 29, 12, 0, 0, 0, time.UTC),
				},
			},
		}),
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/discover/featured-wallets", nil)
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var payload Envelope[service.DiscoverFeaturedWalletResponse]
	decode(t, rr.Body.Bytes(), &payload)
	if len(payload.Data.Items) != 1 {
		t.Fatalf("expected 1 featured wallet, got %d", len(payload.Data.Items))
	}
	if payload.Data.Items[0].Category != "exchange" {
		t.Fatalf("unexpected category %q", payload.Data.Items[0].Category)
	}
	if payload.Data.Items[0].Description != "Public explorer-labeled exchange wallet." {
		t.Fatalf("unexpected display name %q", payload.Data.Items[0].DisplayName)
	}
}
