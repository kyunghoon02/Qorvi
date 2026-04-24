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

type fakeStripeClient struct {
	checkoutRecord     billing.StripeCheckoutSessionRecord
	subscriptionRecord billing.StripeSubscriptionRecord
}

func (f fakeStripeClient) CreateCheckoutSession(context.Context, billing.StripeConfig, billing.StripeCheckoutSessionCreateRequest) (billing.StripeCheckoutSessionRecord, error) {
	return f.checkoutRecord, nil
}

func (f fakeStripeClient) GetSubscription(context.Context, billing.StripeConfig, string) (billing.StripeSubscriptionRecord, error) {
	return f.subscriptionRecord, nil
}

type fakeCheckoutSessionStore struct {
	record billing.StripeCheckoutSessionRecord
}

func (s *fakeCheckoutSessionStore) UpsertBillingCheckoutSession(_ context.Context, record billing.StripeCheckoutSessionRecord) (billing.StripeCheckoutSessionRecord, error) {
	s.record = record
	return record, nil
}

type fakeSubscriptionStore struct {
	record billing.StripeSubscriptionRecord
}

func (s *fakeSubscriptionStore) UpsertBillingSubscription(_ context.Context, record billing.StripeSubscriptionRecord) (billing.StripeSubscriptionRecord, error) {
	s.record = record
	return record, nil
}

type fakeReconciliationStore struct {
	record billing.StripeSubscriptionReconciliationRecord
}

func (s *fakeReconciliationStore) RecordBillingSubscriptionReconciliation(_ context.Context, record billing.StripeSubscriptionReconciliationRecord) error {
	s.record = record
	return nil
}

func TestBillingServiceCreateCheckoutSession(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryBillingRepository()
	svc := NewBillingService(repo, billing.StripeConfig{
		SecretKey:      "test-stripe-secret",
		PublishableKey: "test-stripe-publishable",
		SuccessURL:     "http://localhost:3000/account/success",
		CancelURL:      "http://localhost:3000/account/cancel",
	}, WithStripeClient(fakeStripeClient{
		checkoutRecord: billing.NormalizeStripeCheckoutSessionRecord(billing.StripeCheckoutSessionRecord{
			SessionID:      "cs_live_123",
			CustomerID:     "cus_live_123",
			CustomerEmail:  "ops@flowintel.test",
			SubscriptionID: "sub_live_123",
			Tier:           domain.PlanPro,
			StripePriceID:  "price_pro_placeholder",
			Status:         billing.StripeSessionStatusOpen,
			SuccessURL:     "http://localhost:3000/account/success",
			CancelURL:      "http://localhost:3000/account/cancel",
			CreatedAt:      time.Date(2026, time.March, 21, 10, 0, 0, 0, time.UTC),
			UpdatedAt:      time.Date(2026, time.March, 21, 10, 0, 0, 0, time.UTC),
		}),
	}), WithBillingCheckoutSessionStore(&fakeCheckoutSessionStore{}))

	session, err := svc.CreateCheckoutSession(context.Background(), auth.ClerkPrincipal{
		UserID: "user_123",
		Email:  "ops@flowintel.test",
	}, domain.PlanFree, CreateCheckoutSessionRequest{Tier: "pro"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.TargetTier != "pro" {
		t.Fatalf("expected target tier pro, got %q", session.TargetTier)
	}
	if session.PriceID == "" {
		t.Fatal("expected price id")
	}
	if session.SessionID != "cs_live_123" {
		t.Fatalf("expected live stripe session id, got %q", session.SessionID)
	}
}

func TestBillingServiceReconcileStripeWebhookUpdatesAccount(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryBillingRepository()
	subscriptions := &fakeSubscriptionStore{}
	reconciliations := &fakeReconciliationStore{}
	svc := NewBillingService(repo, billing.StripeConfig{},
		WithBillingSubscriptionStore(subscriptions),
		WithBillingSubscriptionReconciliationStore(reconciliations),
	)
	svc.Now = func() time.Time {
		return time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)
	}

	result, err := svc.ReconcileStripeWebhook(context.Background(), StripeWebhookReconciliationRequest{
		ProviderEventID: "evt_123",
		EventType:       "checkout.session.completed",
		OwnerUserID:     "user_123",
		CustomerID:      "cus_123",
		SubscriptionID:  "sub_123",
		PlanTier:        domain.PlanPro,
		CurrentPriceID:  "price_pro_placeholder",
		Payload:         map[string]any{"ownerUserId": "user_123"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Processed {
		t.Fatal("expected processed result")
	}

	account, err := repo.FindBillingAccount(context.Background(), "user_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account.CurrentTier != domain.PlanPro {
		t.Fatalf("expected persisted pro tier, got %q", account.CurrentTier)
	}
	if subscriptions.record.SubscriptionID != "sub_123" {
		t.Fatalf("expected subscription persistence, got %#v", subscriptions.record)
	}
	if reconciliations.record.EventID != "evt_123" {
		t.Fatalf("expected reconciliation persistence, got %#v", reconciliations.record)
	}
}

func TestBillingServiceResolvePlanTierPrefersPersistedTier(t *testing.T) {
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

	svc := NewBillingService(repo, billing.StripeConfig{})
	if got := svc.ResolvePlanTier(context.Background(), "user_123", domain.PlanFree); got != domain.PlanTeam {
		t.Fatalf("expected team tier, got %q", got)
	}
}

func TestBillingServiceListPlansReturnsCatalog(t *testing.T) {
	t.Parallel()

	svc := NewBillingService(repository.NewInMemoryBillingRepository(), billing.StripeConfig{})
	result, err := svc.ListPlans(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Plans) < 3 {
		t.Fatalf("expected at least 3 plans, got %d", len(result.Plans))
	}
	if result.Plans[0].CheckoutSessionPath != "/v1/billing/checkout-sessions" {
		t.Fatalf("unexpected checkout session path %q", result.Plans[0].CheckoutSessionPath)
	}
	if result.Plans[1].Tier != "pro" {
		t.Fatalf("expected pro tier in pricing catalog, got %q", result.Plans[1].Tier)
	}
	if len(result.Plans[1].Features) == 0 {
		t.Fatal("expected plan feature list")
	}
}
