package service

import (
	"context"
	"fmt"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/auth"
	"github.com/whalegraph/whalegraph/packages/billing"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type AccountPrincipalSummary struct {
	UserID    string `json:"userId"`
	SessionID string `json:"sessionId"`
	Role      string `json:"role"`
	Email     string `json:"email,omitempty"`
}

type AccountAccessContextSummary struct {
	Role string `json:"role"`
	Plan string `json:"plan"`
}

type AccountPlanSummary struct {
	Tier                 string `json:"tier"`
	Name                 string `json:"name"`
	Currency             string `json:"currency"`
	MonthlyPriceCents    int    `json:"monthlyPriceCents"`
	StripePriceID        string `json:"stripePriceId"`
	EnabledFeatureCount  int    `json:"enabledFeatureCount"`
	DisabledFeatureCount int    `json:"disabledFeatureCount"`
}

type AccountEntitlementSummary struct {
	Feature              string `json:"feature"`
	Enabled              bool   `json:"enabled"`
	AccessGranted        bool   `json:"accessGranted"`
	AccessReason         string `json:"accessReason"`
	MaxGraphDepth        int    `json:"maxGraphDepth"`
	MaxFreshnessSeconds  int    `json:"maxFreshnessSeconds"`
	MaxRequestsPerMinute int    `json:"maxRequestsPerMinute"`
}

type AccountResponse struct {
	Principal    AccountPrincipalSummary     `json:"principal"`
	Access       AccountAccessContextSummary `json:"access"`
	Plan         AccountPlanSummary          `json:"plan"`
	Entitlements []AccountEntitlementSummary `json:"entitlements"`
	IssuedAt     string                      `json:"issuedAt"`
}

type AccountService struct {
	billing *BillingService
	Now     func() time.Time
}

func NewAccountService(billingService ...*BillingService) *AccountService {
	var resolver *BillingService
	if len(billingService) > 0 {
		resolver = billingService[0]
	}
	return &AccountService{
		billing: resolver,
		Now:     time.Now,
	}
}

func (s *AccountService) GetAccount(ctx context.Context, principal auth.ClerkPrincipal, tier domain.PlanTier) (AccountResponse, error) {
	_ = ctx

	role := billing.NormalizeRole(principal.Role)
	resolvedTier := tier
	if s != nil && s.billing != nil {
		resolvedTier = s.billing.ResolvePlanTier(ctx, principal.UserID, tier)
	}
	snapshot, err := billing.SnapshotForTier(resolvedTier, role)
	if err != nil {
		return AccountResponse{}, fmt.Errorf("build entitlement snapshot: %w", err)
	}

	response := AccountResponse{
		Principal: AccountPrincipalSummary{
			UserID:    principal.UserID,
			SessionID: principal.SessionID,
			Role:      principal.Role,
			Email:     principal.Email,
		},
		Access: AccountAccessContextSummary{
			Role: string(role),
			Plan: string(snapshot.Tier),
		},
		Plan: AccountPlanSummary{
			Tier:                 string(snapshot.Tier),
			Name:                 snapshot.Name,
			Currency:             snapshot.Currency,
			MonthlyPriceCents:    snapshot.MonthlyPriceCents,
			StripePriceID:        snapshot.StripePriceID,
			EnabledFeatureCount:  snapshot.EnabledFeatureCount,
			DisabledFeatureCount: snapshot.DisabledFeatureCount,
		},
		Entitlements: make([]AccountEntitlementSummary, 0, len(snapshot.Features)),
		IssuedAt:     s.now().Format(time.RFC3339Nano),
	}

	for _, entitlement := range snapshot.Features {
		response.Entitlements = append(response.Entitlements, AccountEntitlementSummary{
			Feature:              string(entitlement.Feature),
			Enabled:              entitlement.Enabled,
			AccessGranted:        entitlement.AccessGranted,
			AccessReason:         string(entitlement.AccessReason),
			MaxGraphDepth:        entitlement.MaxGraphDepth,
			MaxFreshnessSeconds:  entitlement.MaxFreshnessSeconds,
			MaxRequestsPerMinute: entitlement.MaxRequestsPerMinute,
		})
	}

	return response, nil
}

func (s *AccountService) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}

	return time.Now().UTC()
}
