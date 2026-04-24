package service

import (
	"context"
	"testing"
	"time"

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

type fakeDiscoverAutoReader struct {
	items []db.AutoDiscoverWallet
	err   error
}

func (r *fakeDiscoverAutoReader) ListAutoDiscoverWallets(_ context.Context, _ int) ([]db.AutoDiscoverWallet, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]db.AutoDiscoverWallet(nil), r.items...), nil
}

type fakeDiscoverDomesticReader struct {
	items []db.DomesticPrelistingCandidateRecord
	err   error
}

func (r *fakeDiscoverDomesticReader) ListDomesticPrelistingCandidates(_ context.Context, _, _ time.Time, _ int) ([]db.DomesticPrelistingCandidateRecord, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]db.DomesticPrelistingCandidateRecord(nil), r.items...), nil
}

func TestDiscoverServiceListFeaturedWalletsUsesAdminCuratedSeeds(t *testing.T) {
	t.Parallel()

	svc := NewDiscoverService(&fakeDiscoverSeedReader{
		items: []db.CuratedWalletSeed{
			{
				Chain:     domain.ChainEVM,
				Address:   "0x1111111111111111111111111111111111111111",
				ListTags:  []string{"admin-curated", "wallet-seeds", "exchange"},
				ItemTags:  []string{"featured", "verified-public", "exchange"},
				ItemNotes: "Public explorer-labeled exchange wallet.",
				UpdatedAt: time.Date(2026, time.March, 29, 12, 0, 0, 0, time.UTC),
			},
			{
				Chain:     domain.ChainEVM,
				Address:   "0x2222222222222222222222222222222222222222",
				ListTags:  []string{"admin-curated", "wallet-seeds", "fund"},
				ItemTags:  []string{"probable", "fund"},
				ItemNotes: "Probable fund wallet.",
				UpdatedAt: time.Date(2026, time.March, 28, 8, 0, 0, 0, time.UTC),
			},
		},
	})
	response, err := svc.ListFeaturedWallets(context.Background())
	if err != nil {
		t.Fatalf("ListFeaturedWallets returned error: %v", err)
	}

	if len(response.Items) != 2 {
		t.Fatalf("expected 2 featured wallets, got %d", len(response.Items))
	}
	if response.Items[0].Category != "exchange" {
		t.Fatalf("unexpected top category %q", response.Items[0].Category)
	}
	if response.Items[0].Description != "Public explorer-labeled exchange wallet." {
		t.Fatalf("unexpected top description %q", response.Items[0].Description)
	}
	if response.Items[0].ObservedAt != "2026-03-29T12:00:00Z" {
		t.Fatalf("unexpected observed_at %q", response.Items[0].ObservedAt)
	}
	if response.Items[1].Category != "fund" {
		t.Fatalf("unexpected secondary category %q", response.Items[1].Category)
	}
}

func TestDiscoverServiceListFeaturedWalletsReturnsEmptyWithoutSeeds(t *testing.T) {
	t.Parallel()

	svc := NewDiscoverService(nil)
	response, err := svc.ListFeaturedWallets(context.Background())
	if err != nil {
		t.Fatalf("ListFeaturedWallets returned error: %v", err)
	}
	if len(response.Items) != 0 {
		t.Fatalf("expected empty items, got %d", len(response.Items))
	}
}

func TestDiscoverServiceListFeaturedWalletsFallsBackToAutoDiscoveredWallets(t *testing.T) {
	t.Parallel()

	lastActivityAt := time.Date(2026, time.April, 12, 9, 30, 0, 0, time.UTC)
	svc := NewDiscoverService(nil, &fakeDiscoverAutoReader{
		items: []db.AutoDiscoverWallet{
			{
				Chain:          domain.ChainEVM,
				Address:        "0x3333333333333333333333333333333333333333",
				DisplayName:    "Fresh search wallet",
				Status:         db.WalletTrackingStatusTracked,
				SourceType:     db.WalletTrackingSourceTypeUserSearch,
				LastActivityAt: &lastActivityAt,
				UpdatedAt:      lastActivityAt,
			},
		},
	})

	response, err := svc.ListFeaturedWallets(context.Background())
	if err != nil {
		t.Fatalf("ListFeaturedWallets returned error: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected 1 auto-discovered wallet, got %d", len(response.Items))
	}
	if response.Items[0].Category != "searched" {
		t.Fatalf("unexpected category %q", response.Items[0].Category)
	}
	if response.Items[0].DisplayName != "Fresh search wallet" {
		t.Fatalf("unexpected display name %q", response.Items[0].DisplayName)
	}
	if response.Items[0].ObservedAt != "2026-04-12T09:30:00Z" {
		t.Fatalf("unexpected observed_at %q", response.Items[0].ObservedAt)
	}
}

func TestDiscoverServiceListFeaturedWalletsAppendsAutoDiscoveredWalletsWithoutDuplicates(t *testing.T) {
	t.Parallel()

	svc := NewDiscoverService(
		&fakeDiscoverSeedReader{
			items: []db.CuratedWalletSeed{
				{
					Chain:    domain.ChainEVM,
					Address:  "0x1111111111111111111111111111111111111111",
					ListTags: []string{"featured"},
				},
			},
		},
		&fakeDiscoverAutoReader{
			items: []db.AutoDiscoverWallet{
				{
					Chain:       domain.ChainEVM,
					Address:     "0x1111111111111111111111111111111111111111",
					DisplayName: "Duplicate wallet",
					Status:      db.WalletTrackingStatusTracked,
					SourceType:  db.WalletTrackingSourceTypeUserSearch,
					UpdatedAt:   time.Date(2026, time.April, 12, 10, 0, 0, 0, time.UTC),
				},
				{
					Chain:       domain.ChainSolana,
					Address:     "GBrURzmtWujJRTA3Bkvo7ZgWuZYLMMwPCwre7BejJXnK",
					DisplayName: "Solana candidate",
					Status:      db.WalletTrackingStatusScored,
					SourceType:  db.WalletTrackingSourceTypeHopExpansion,
					UpdatedAt:   time.Date(2026, time.April, 12, 11, 0, 0, 0, time.UTC),
				},
			},
		},
	)

	response, err := svc.ListFeaturedWallets(context.Background())
	if err != nil {
		t.Fatalf("ListFeaturedWallets returned error: %v", err)
	}
	if len(response.Items) != 2 {
		t.Fatalf("expected 2 merged wallets, got %d", len(response.Items))
	}
	if response.Items[1].Category != "graph" {
		t.Fatalf("unexpected auto-discovered category %q", response.Items[1].Category)
	}
}

func TestDiscoverServiceListDomesticPrelistingCandidatesReturnsRepresentativeWallet(t *testing.T) {
	t.Parallel()

	svc := NewDiscoverService(
		nil,
		&fakeDiscoverDomesticReader{
			items: []db.DomesticPrelistingCandidateRecord{
				{
					Chain:                     "EVM",
					TokenAddress:              "0x3333333333333333333333333333333333333333",
					TokenSymbol:               "NEWT",
					NormalizedAssetKey:        "newt",
					TransferCount7d:           42,
					TransferCount24h:          11,
					ActiveWalletCount:         7,
					TrackedWalletCount:        3,
					DistinctCounterpartyCount: 9,
					TotalAmount:               "123456.78",
					LargestTransferAmount:     "50000",
					LatestObservedAt:          time.Date(2026, time.April, 18, 2, 0, 0, 0, time.UTC),
					RepresentativeWalletChain: "evm",
					RepresentativeWallet:      "0x1111111111111111111111111111111111111111",
					RepresentativeLabel:       "Tracked whale",
				},
			},
		},
	)
	svc.Now = func() time.Time {
		return time.Date(2026, time.April, 18, 12, 0, 0, 0, time.UTC)
	}

	response, err := svc.ListDomesticPrelistingCandidates(context.Background(), 12)
	if err != nil {
		t.Fatalf("ListDomesticPrelistingCandidates returned error: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(response.Items))
	}
	if response.Items[0].RepresentativeWallet != "0x1111111111111111111111111111111111111111" {
		t.Fatalf("unexpected representative wallet %q", response.Items[0].RepresentativeWallet)
	}
	if response.Items[0].LatestObservedAt != "2026-04-18T02:00:00Z" {
		t.Fatalf("unexpected observed_at %q", response.Items[0].LatestObservedAt)
	}
}
