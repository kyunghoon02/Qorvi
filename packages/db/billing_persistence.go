package db

import (
	"context"
	"time"

	"github.com/qorvi/qorvi/packages/billing"
)

type BillingCheckoutSessionStore interface {
	UpsertBillingCheckoutSession(context.Context, billing.StripeCheckoutSessionRecord) (billing.StripeCheckoutSessionRecord, error)
	FindBillingCheckoutSession(context.Context, string) (billing.StripeCheckoutSessionRecord, error)
	MarkBillingCheckoutSessionCompleted(context.Context, string, time.Time) (billing.StripeCheckoutSessionRecord, error)
}

type BillingSubscriptionStore interface {
	UpsertBillingSubscription(context.Context, billing.StripeSubscriptionRecord) (billing.StripeSubscriptionRecord, error)
	FindBillingSubscriptionByCustomerID(context.Context, string) (billing.StripeSubscriptionRecord, error)
	FindBillingSubscriptionBySubscriptionID(context.Context, string) (billing.StripeSubscriptionRecord, error)
}

type BillingWebhookEventStore interface {
	RecordBillingWebhookEvent(context.Context, billing.StripeWebhookEventRecord) error
	MarkBillingWebhookEventProcessed(context.Context, string, time.Time) error
}

type BillingSubscriptionReconciliationStore interface {
	RecordBillingSubscriptionReconciliation(context.Context, billing.StripeSubscriptionReconciliationRecord) error
	ListBillingSubscriptionReconciliations(context.Context, string) ([]billing.StripeSubscriptionReconciliationRecord, error)
}

type BillingPersistenceTableSpec = billing.BillingPersistenceTableSpec

func ExpectedBillingPersistenceTables() []BillingPersistenceTableSpec {
	return billing.CloneBillingPersistenceTableSpecs(billing.ExpectedBillingPersistenceTables())
}
