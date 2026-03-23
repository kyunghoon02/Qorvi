package billing

import (
	"slices"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
)

type EntitlementSnapshot struct {
	Tier              domain.PlanTier `json:"tier"`
	Name              string          `json:"name"`
	Currency          string          `json:"currency"`
	MonthlyPriceCents int             `json:"monthly_price_cents"`
	StripePriceID     string          `json:"stripe_price_id"`
	CapturedAt        time.Time       `json:"captured_at"`
	Entitlements      []Entitlement   `json:"entitlements"`
}

func SnapshotPlanEntitlements(plan Plan) EntitlementSnapshot {
	return NormalizeEntitlementSnapshot(EntitlementSnapshot{
		Tier:              plan.Tier,
		Name:              plan.Name,
		Currency:          plan.Currency,
		MonthlyPriceCents: plan.MonthlyPriceCents,
		StripePriceID:     plan.StripePriceID,
		Entitlements:      append([]Entitlement(nil), plan.Entitlements...),
		CapturedAt:        time.Now().UTC(),
	})
}

func NormalizeEntitlementSnapshot(snapshot EntitlementSnapshot) EntitlementSnapshot {
	normalized := snapshot
	normalized.Tier = domain.PlanTier(strings.TrimSpace(string(snapshot.Tier)))
	normalized.Name = strings.TrimSpace(snapshot.Name)
	normalized.Currency = strings.ToLower(strings.TrimSpace(snapshot.Currency))
	normalized.StripePriceID = strings.TrimSpace(snapshot.StripePriceID)
	if normalized.CapturedAt.IsZero() {
		normalized.CapturedAt = time.Now().UTC()
	}

	normalized.Entitlements = normalizeEntitlementRows(snapshot.Entitlements)
	return normalized
}

func EntitlementForSnapshot(snapshot EntitlementSnapshot, feature Feature) (Entitlement, bool) {
	normalizedFeature := normalizeFeature(feature)
	for _, entitlement := range snapshot.Entitlements {
		if entitlement.Feature == normalizedFeature {
			return entitlement, true
		}
	}

	return Entitlement{}, false
}

func IsFeatureEnabledSnapshot(snapshot EntitlementSnapshot, feature Feature) bool {
	entitlement, ok := EntitlementForSnapshot(snapshot, feature)
	return ok && entitlement.Enabled
}

func MaxGraphDepthForSnapshot(snapshot EntitlementSnapshot, feature Feature) int {
	entitlement, ok := EntitlementForSnapshot(snapshot, feature)
	if !ok || entitlement.MaxGraphDepth <= 0 {
		return 0
	}

	return entitlement.MaxGraphDepth
}

func MaxFreshnessSecondsForSnapshot(snapshot EntitlementSnapshot, feature Feature) int {
	entitlement, ok := EntitlementForSnapshot(snapshot, feature)
	if !ok || entitlement.MaxFreshnessSeconds <= 0 {
		return 0
	}

	return entitlement.MaxFreshnessSeconds
}

func MaxRequestsPerMinuteForSnapshot(snapshot EntitlementSnapshot, feature Feature) int {
	entitlement, ok := EntitlementForSnapshot(snapshot, feature)
	if !ok || entitlement.MaxRequestsPerMinute <= 0 {
		return 0
	}

	return entitlement.MaxRequestsPerMinute
}

func normalizeEntitlementRows(items []Entitlement) []Entitlement {
	if len(items) == 0 {
		return []Entitlement{}
	}

	seen := make(map[Feature]struct{}, len(items))
	normalized := make([]Entitlement, 0, len(items))
	for _, item := range items {
		item.Feature = normalizeFeature(item.Feature)
		if item.Feature == "" {
			continue
		}
		if _, ok := seen[item.Feature]; ok {
			continue
		}
		seen[item.Feature] = struct{}{}
		normalized = append(normalized, item)
	}

	slices.SortFunc(normalized, func(left, right Entitlement) int {
		return strings.Compare(string(left.Feature), string(right.Feature))
	})
	return normalized
}

func normalizeFeature(feature Feature) Feature {
	return Feature(strings.ToLower(strings.TrimSpace(string(feature))))
}
