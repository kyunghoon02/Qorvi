package billing

import (
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestNormalizePlanTier(t *testing.T) {
	t.Parallel()

	if got := NormalizePlanTier("team"); got != domain.PlanTeam {
		t.Fatalf("expected team tier, got %q", got)
	}
	if got := NormalizePlanTier("unknown"); got != domain.PlanFree {
		t.Fatalf("expected free tier fallback, got %q", got)
	}
}

func TestSnapshotForTierMarksRoleRestrictedFeatures(t *testing.T) {
	t.Parallel()

	snapshot, err := SnapshotForTier(domain.PlanTeam, domain.RoleUser)
	if err != nil {
		t.Fatalf("expected snapshot, got %v", err)
	}

	adminConsole, ok := LookupFeatureSnapshot(snapshot, FeatureAdminConsole)
	if !ok {
		t.Fatal("expected admin console feature in snapshot")
	}

	if adminConsole.AccessGranted {
		t.Fatal("expected admin console to require elevated role")
	}
	if adminConsole.AccessReason != FeatureAccessRoleRequired {
		t.Fatalf("expected role required reason, got %q", adminConsole.AccessReason)
	}
}

func TestSnapshotForTierCountsGrantedFeatures(t *testing.T) {
	t.Parallel()

	snapshot, err := SnapshotForTier(domain.PlanPro, domain.RoleUser)
	if err != nil {
		t.Fatalf("expected snapshot, got %v", err)
	}

	if err := snapshot.Validate(); err != nil {
		t.Fatalf("expected valid snapshot, got %v", err)
	}
	if snapshot.EnabledFeatureCount == 0 {
		t.Fatal("expected enabled features to be counted")
	}

	graph, ok := LookupFeatureSnapshot(snapshot, FeatureGraph)
	if !ok {
		t.Fatal("expected graph feature in snapshot")
	}
	if !graph.AccessGranted {
		t.Fatal("expected graph access to be granted for pro")
	}
	if graph.MaxGraphDepth != 2 {
		t.Fatalf("expected pro graph depth 2, got %d", graph.MaxGraphDepth)
	}
}
