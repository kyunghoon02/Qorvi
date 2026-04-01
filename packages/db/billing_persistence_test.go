package db

import (
	"context"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/billing"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeBillingCheckoutSessionStore struct{}

func (fakeBillingCheckoutSessionStore) UpsertBillingCheckoutSession(context.Context, billing.StripeCheckoutSessionRecord) (billing.StripeCheckoutSessionRecord, error) {
	return billing.StripeCheckoutSessionRecord{}, nil
}

func (fakeBillingCheckoutSessionStore) FindBillingCheckoutSession(context.Context, string) (billing.StripeCheckoutSessionRecord, error) {
	return billing.StripeCheckoutSessionRecord{}, nil
}

func (fakeBillingCheckoutSessionStore) MarkBillingCheckoutSessionCompleted(context.Context, string, time.Time) (billing.StripeCheckoutSessionRecord, error) {
	return billing.StripeCheckoutSessionRecord{}, nil
}

type fakeBillingSubscriptionStore struct{}

func (fakeBillingSubscriptionStore) UpsertBillingSubscription(context.Context, billing.StripeSubscriptionRecord) (billing.StripeSubscriptionRecord, error) {
	return billing.StripeSubscriptionRecord{}, nil
}

func (fakeBillingSubscriptionStore) FindBillingSubscriptionByCustomerID(context.Context, string) (billing.StripeSubscriptionRecord, error) {
	return billing.StripeSubscriptionRecord{}, nil
}

func (fakeBillingSubscriptionStore) FindBillingSubscriptionBySubscriptionID(context.Context, string) (billing.StripeSubscriptionRecord, error) {
	return billing.StripeSubscriptionRecord{}, nil
}

type fakeBillingWebhookEventStore struct{}

func (fakeBillingWebhookEventStore) RecordBillingWebhookEvent(context.Context, billing.StripeWebhookEventRecord) error {
	return nil
}

func (fakeBillingWebhookEventStore) MarkBillingWebhookEventProcessed(context.Context, string, time.Time) error {
	return nil
}

type fakeBillingSubscriptionReconciliationStore struct{}

func (fakeBillingSubscriptionReconciliationStore) RecordBillingSubscriptionReconciliation(context.Context, billing.StripeSubscriptionReconciliationRecord) error {
	return nil
}

func (fakeBillingSubscriptionReconciliationStore) ListBillingSubscriptionReconciliations(context.Context, string) ([]billing.StripeSubscriptionReconciliationRecord, error) {
	return nil, nil
}

func TestBillingPersistenceInterfacesCompile(t *testing.T) {
	t.Parallel()

	var _ BillingCheckoutSessionStore = fakeBillingCheckoutSessionStore{}
	var _ BillingSubscriptionStore = fakeBillingSubscriptionStore{}
	var _ BillingWebhookEventStore = fakeBillingWebhookEventStore{}
	var _ BillingSubscriptionReconciliationStore = fakeBillingSubscriptionReconciliationStore{}
}

func TestExpectedBillingPersistenceTables(t *testing.T) {
	t.Parallel()

	specs := ExpectedBillingPersistenceTables()
	if len(specs) != 4 {
		t.Fatalf("expected 4 table specs, got %d", len(specs))
	}
	if specs[0].Name != "billing_checkout_sessions" {
		t.Fatalf("unexpected first table %q", specs[0].Name)
	}
	if specs[1].Name != "billing_subscriptions" {
		t.Fatalf("unexpected second table %q", specs[1].Name)
	}
	if specs[2].Name != "billing_webhook_events" {
		t.Fatalf("unexpected third table %q", specs[2].Name)
	}
	if specs[3].Name != "billing_subscription_reconciliations" {
		t.Fatalf("unexpected fourth table %q", specs[3].Name)
	}
}

func TestBillingPersistenceSpecUsesSupportedPlanTierOnly(t *testing.T) {
	t.Parallel()

	checkout := billing.NormalizeStripeCheckoutSessionRecord(billing.StripeCheckoutSessionRecord{
		SessionID:     "session_1",
		CustomerID:    "customer_1",
		Tier:          domain.PlanTier("pro"),
		StripePriceID: "price_pro_placeholder",
		Status:        billing.StripeSessionStatusOpen,
		SuccessURL:    "https://example.test/success",
		CancelURL:     "https://example.test/cancel",
	})

	if checkout.Tier != domain.PlanPro {
		t.Fatalf("unexpected normalized tier %q", checkout.Tier)
	}
}
