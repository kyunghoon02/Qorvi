package billing

import (
	"testing"
)

func TestSnapshotPlanEntitlementsNormalizesAndClones(t *testing.T) {
	t.Parallel()

	plan := Plan{
		Tier:              "pro",
		Name:              " Pro ",
		Currency:          " USD ",
		MonthlyPriceCents: 4900,
		StripePriceID:     " price_pro_placeholder ",
		Entitlements: []Entitlement{
			{Feature: FeatureWatchlist, Enabled: true, MaxRequestsPerMinute: 48},
			{Feature: FeatureSearch, Enabled: true, MaxGraphDepth: 2, MaxFreshnessSeconds: 300, MaxRequestsPerMinute: 60},
			{Feature: FeatureSearch, Enabled: false, MaxGraphDepth: 1, MaxFreshnessSeconds: 120, MaxRequestsPerMinute: 1},
		},
	}

	snapshot := SnapshotPlanEntitlements(plan)
	if snapshot.Tier != "pro" {
		t.Fatalf("expected normalized tier, got %q", snapshot.Tier)
	}
	if snapshot.Name != "Pro" {
		t.Fatalf("expected trimmed name, got %q", snapshot.Name)
	}
	if snapshot.Currency != "usd" {
		t.Fatalf("expected normalized currency, got %q", snapshot.Currency)
	}
	if snapshot.StripePriceID != "price_pro_placeholder" {
		t.Fatalf("expected trimmed stripe price id, got %q", snapshot.StripePriceID)
	}
	if len(snapshot.Entitlements) != 2 {
		t.Fatalf("expected duplicate features to be collapsed, got %d", len(snapshot.Entitlements))
	}
	if snapshot.Entitlements[0].Feature != FeatureSearch || snapshot.Entitlements[1].Feature != FeatureWatchlist {
		t.Fatalf("expected stable feature ordering, got %#v", snapshot.Entitlements)
	}

	snapshot.Entitlements[0].Enabled = false
	if !plan.Entitlements[1].Enabled {
		t.Fatal("expected snapshot mutation to leave original plan untouched")
	}
}

func TestEntitlementSnapshotFeatureLookup(t *testing.T) {
	t.Parallel()

	snapshot := NormalizeEntitlementSnapshot(EntitlementSnapshot{
		Tier: "team",
		Entitlements: []Entitlement{
			{Feature: FeatureAlerts, Enabled: true, MaxRequestsPerMinute: 120},
			{Feature: FeatureGraph, Enabled: true, MaxGraphDepth: 3, MaxFreshnessSeconds: 120},
		},
	})

	entitlement, ok := EntitlementForSnapshot(snapshot, FeatureGraph)
	if !ok {
		t.Fatal("expected graph entitlement to be found")
	}
	if entitlement.MaxGraphDepth != 3 {
		t.Fatalf("expected max graph depth 3, got %d", entitlement.MaxGraphDepth)
	}
	if !IsFeatureEnabledSnapshot(snapshot, FeatureAlerts) {
		t.Fatal("expected alerts feature to be enabled")
	}
	if MaxGraphDepthForSnapshot(snapshot, FeatureWatchlist) != 0 {
		t.Fatal("expected missing feature to return zero graph depth")
	}
	if MaxFreshnessSecondsForSnapshot(snapshot, FeatureWatchlist) != 0 {
		t.Fatal("expected missing feature to return zero freshness")
	}
	if MaxRequestsPerMinuteForSnapshot(snapshot, FeatureWatchlist) != 0 {
		t.Fatal("expected missing feature to return zero rpm")
	}
}
