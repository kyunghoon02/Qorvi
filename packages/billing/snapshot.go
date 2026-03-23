package billing

import (
	"fmt"
	"strings"

	"github.com/whalegraph/whalegraph/packages/domain"
)

type FeatureAccessReason string

const (
	FeatureAccessGranted      FeatureAccessReason = "granted"
	FeatureAccessPlanDisabled FeatureAccessReason = "plan_disabled"
	FeatureAccessRoleRequired FeatureAccessReason = "role_required"
)

type FeatureSnapshot struct {
	Feature              Feature             `json:"feature"`
	Enabled              bool                `json:"enabled"`
	AccessGranted        bool                `json:"access_granted"`
	AccessReason         FeatureAccessReason `json:"access_reason"`
	MaxGraphDepth        int                 `json:"max_graph_depth"`
	MaxFreshnessSeconds  int                 `json:"max_freshness_seconds"`
	MaxRequestsPerMinute int                 `json:"max_requests_per_minute"`
}

type PlanSnapshot struct {
	Tier                 domain.PlanTier   `json:"tier"`
	Name                 string            `json:"name"`
	Currency             string            `json:"currency"`
	MonthlyPriceCents    int               `json:"monthly_price_cents"`
	StripePriceID        string            `json:"stripe_price_id"`
	EnabledFeatureCount  int               `json:"enabled_feature_count"`
	DisabledFeatureCount int               `json:"disabled_feature_count"`
	Features             []FeatureSnapshot `json:"features"`
}

func NormalizePlanTier(raw string) domain.PlanTier {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(domain.PlanPro):
		return domain.PlanPro
	case string(domain.PlanTeam):
		return domain.PlanTeam
	default:
		return domain.PlanFree
	}
}

func NormalizeRole(raw string) domain.Role {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(domain.RoleAdmin):
		return domain.RoleAdmin
	case string(domain.RoleOperator):
		return domain.RoleOperator
	case string(domain.RoleUser):
		return domain.RoleUser
	default:
		return domain.RoleAnonymous
	}
}

func SnapshotForTier(tier domain.PlanTier, role domain.Role) (PlanSnapshot, error) {
	plan, err := FindPlan(NormalizePlanTier(string(tier)))
	if err != nil {
		return PlanSnapshot{}, err
	}

	return SnapshotForPlan(plan, role), nil
}

func SnapshotForPlan(plan Plan, role domain.Role) PlanSnapshot {
	normalizedRole := NormalizeRole(string(role))
	features := make([]FeatureSnapshot, 0, len(plan.Entitlements))
	enabledCount := 0
	disabledCount := 0

	for _, entitlement := range plan.Entitlements {
		accessGranted, reason := evaluateFeatureAccess(entitlement, normalizedRole)
		if accessGranted {
			enabledCount++
		} else {
			disabledCount++
		}

		features = append(features, FeatureSnapshot{
			Feature:              entitlement.Feature,
			Enabled:              entitlement.Enabled,
			AccessGranted:        accessGranted,
			AccessReason:         reason,
			MaxGraphDepth:        entitlement.MaxGraphDepth,
			MaxFreshnessSeconds:  entitlement.MaxFreshnessSeconds,
			MaxRequestsPerMinute: entitlement.MaxRequestsPerMinute,
		})
	}

	return PlanSnapshot{
		Tier:                 plan.Tier,
		Name:                 plan.Name,
		Currency:             plan.Currency,
		MonthlyPriceCents:    plan.MonthlyPriceCents,
		StripePriceID:        plan.StripePriceID,
		EnabledFeatureCount:  enabledCount,
		DisabledFeatureCount: disabledCount,
		Features:             features,
	}
}

func LookupFeatureSnapshot(snapshot PlanSnapshot, feature Feature) (FeatureSnapshot, bool) {
	for _, item := range snapshot.Features {
		if item.Feature == feature {
			return item, true
		}
	}

	return FeatureSnapshot{}, false
}

func evaluateFeatureAccess(
	entitlement Entitlement,
	role domain.Role,
) (bool, FeatureAccessReason) {
	if !entitlement.Enabled {
		return false, FeatureAccessPlanDisabled
	}

	switch entitlement.Feature {
	case FeatureAdminConsole:
		if role != domain.RoleAdmin && role != domain.RoleOperator {
			return false, FeatureAccessRoleRequired
		}
	}

	return true, FeatureAccessGranted
}

func (s PlanSnapshot) Validate() error {
	if s.Tier == "" {
		return fmt.Errorf("snapshot tier is required")
	}
	if len(s.Features) == 0 {
		return fmt.Errorf("snapshot features are required")
	}
	return nil
}
