package service

import (
	"context"
	"testing"
	"time"

	"github.com/flowintel/flowintel/apps/api/internal/auth"
	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/packages/billing"
	"github.com/flowintel/flowintel/packages/domain"
)

func TestAccountServiceBuildsPlanAndEntitlementsFromTier(t *testing.T) {
	t.Parallel()

	svc := NewAccountService()
	svc.Now = func() time.Time {
		return time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC)
	}

	account, err := svc.GetAccount(context.Background(), auth.ClerkPrincipal{
		UserID:    "user_123",
		SessionID: "sess_123",
		Role:      "admin",
		Email:     "ops@flowintel.test",
	}, "pro")
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}

	if account.Plan.Tier != "pro" || account.Access.Plan != "pro" {
		t.Fatalf("unexpected plan context %#v", account)
	}

	if account.Principal.UserID != "user_123" || account.Principal.Role != "admin" {
		t.Fatalf("unexpected principal %#v", account.Principal)
	}

	if len(account.Entitlements) == 0 {
		t.Fatal("expected entitlements")
	}
	if account.Plan.EnabledFeatureCount == 0 {
		t.Fatal("expected enabled feature count to be populated")
	}
}

func TestAccountServiceMarksRoleRestrictedFeatureAccess(t *testing.T) {
	t.Parallel()

	svc := NewAccountService()
	account, err := svc.GetAccount(context.Background(), auth.ClerkPrincipal{
		UserID:    "user_123",
		SessionID: "sess_123",
		Role:      "user",
	}, domain.PlanTeam)
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}

	var adminConsoleFound bool
	for _, entitlement := range account.Entitlements {
		if entitlement.Feature != "admin_console" {
			continue
		}

		adminConsoleFound = true
		if entitlement.AccessGranted {
			t.Fatal("expected admin console to stay role-restricted for user role")
		}
		if entitlement.AccessReason != "role_required" {
			t.Fatalf("unexpected access reason %q", entitlement.AccessReason)
		}
	}

	if !adminConsoleFound {
		t.Fatal("expected admin console entitlement to be present")
	}
}

func TestAccountServiceUsesPersistedBillingTierWhenAvailable(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryBillingRepository()
	_, err := repo.UpsertBillingAccount(context.Background(), repository.BillingAccount{
		OwnerUserID: "user_123",
		CurrentTier: domain.PlanTeam,
		Status:      "active",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc := NewAccountService(NewBillingService(repo, billing.StripeConfig{}))
	account, err := svc.GetAccount(context.Background(), auth.ClerkPrincipal{
		UserID:    "user_123",
		SessionID: "sess_123",
		Role:      "user",
	}, domain.PlanFree)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if account.Plan.Tier != "team" {
		t.Fatalf("expected persisted team tier, got %q", account.Plan.Tier)
	}
}
