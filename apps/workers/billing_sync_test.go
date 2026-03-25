package main

import (
	"context"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/billing"
	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

type fakeBillingAccountSyncReader struct {
	accounts []db.BillingAccountRecord
}

func (r fakeBillingAccountSyncReader) ListBillingAccountsForSubscriptionSync(context.Context, int) ([]db.BillingAccountRecord, error) {
	return append([]db.BillingAccountRecord(nil), r.accounts...), nil
}

type fakeBillingAccountStore struct {
	record db.BillingAccountRecord
}

func (s *fakeBillingAccountStore) FindBillingAccount(context.Context, string) (db.BillingAccountRecord, error) {
	return db.BillingAccountRecord{}, db.ErrBillingAccountNotFound
}

func (s *fakeBillingAccountStore) UpsertBillingAccount(_ context.Context, record db.BillingAccountRecord) (db.BillingAccountRecord, error) {
	s.record = record
	return record, nil
}

func (s *fakeBillingAccountStore) RecordWebhookEvent(context.Context, db.BillingWebhookEventRecord) (db.BillingWebhookEventRecord, error) {
	return db.BillingWebhookEventRecord{}, nil
}

type fakeBillingSubscriptionStoreWriter struct {
	record billing.StripeSubscriptionRecord
}

func (s *fakeBillingSubscriptionStoreWriter) UpsertBillingSubscription(_ context.Context, record billing.StripeSubscriptionRecord) (billing.StripeSubscriptionRecord, error) {
	s.record = record
	return record, nil
}

func (s *fakeBillingSubscriptionStoreWriter) FindBillingSubscriptionByCustomerID(context.Context, string) (billing.StripeSubscriptionRecord, error) {
	return billing.StripeSubscriptionRecord{}, nil
}

func (s *fakeBillingSubscriptionStoreWriter) FindBillingSubscriptionBySubscriptionID(context.Context, string) (billing.StripeSubscriptionRecord, error) {
	return billing.StripeSubscriptionRecord{}, nil
}

type fakeBillingReconciliationStore struct {
	record billing.StripeSubscriptionReconciliationRecord
}

func (s *fakeBillingReconciliationStore) RecordBillingSubscriptionReconciliation(_ context.Context, record billing.StripeSubscriptionReconciliationRecord) error {
	s.record = record
	return nil
}

func (s *fakeBillingReconciliationStore) ListBillingSubscriptionReconciliations(context.Context, string) ([]billing.StripeSubscriptionReconciliationRecord, error) {
	return nil, nil
}

type fakeWorkerStripeClient struct {
	subscription billing.StripeSubscriptionRecord
}

func (c fakeWorkerStripeClient) CreateCheckoutSession(context.Context, billing.StripeConfig, billing.StripeCheckoutSessionCreateRequest) (billing.StripeCheckoutSessionRecord, error) {
	return billing.StripeCheckoutSessionRecord{}, nil
}

func (c fakeWorkerStripeClient) GetSubscription(context.Context, billing.StripeConfig, string) (billing.StripeSubscriptionRecord, error) {
	return c.subscription, nil
}

func TestBillingSubscriptionSyncServiceRunBatch(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 23, 9, 0, 0, 0, time.UTC)
	accountStore := &fakeBillingAccountStore{}
	subscriptionStore := &fakeBillingSubscriptionStoreWriter{}
	reconciliationStore := &fakeBillingReconciliationStore{}

	service := BillingSubscriptionSyncService{
		Accounts: fakeBillingAccountSyncReader{
			accounts: []db.BillingAccountRecord{{
				OwnerUserID:          "user_123",
				Email:                "ops@flowintel.test",
				CurrentTier:          domain.PlanFree,
				StripeCustomerID:     "cus_123",
				ActiveSubscriptionID: "sub_123",
				CurrentPriceID:       "price_pro_placeholder",
				Status:               "active",
				CreatedAt:            now.Add(-24 * time.Hour),
				UpdatedAt:            now.Add(-1 * time.Hour),
			}},
		},
		AccountStore:    accountStore,
		Subscriptions:   subscriptionStore,
		Reconciliations: reconciliationStore,
		StripeClient: fakeWorkerStripeClient{
			subscription: billing.NormalizeStripeSubscriptionRecord(billing.StripeSubscriptionRecord{
				SubscriptionID:     "sub_123",
				CustomerID:         "cus_123",
				CustomerEmail:      "ops@flowintel.test",
				StripePriceID:      "price_pro_placeholder",
				Tier:               domain.PlanPro,
				Status:             billing.StripeSubscriptionStatusActive,
				CurrentPeriodStart: now.Add(-12 * time.Hour),
				CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
				SyncedAt:           now,
				UpdatedAt:          now,
			}),
		},
		StripeConfig: billing.StripeConfig{SecretKey: "sk_live_test"},
		Now: func() time.Time {
			return now
		},
	}

	report, err := service.RunBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("RunBatch returned error: %v", err)
	}

	if report.SubscriptionsSynced != 1 || report.AccountsUpdated != 1 || report.ReconciliationsSaved != 1 {
		t.Fatalf("unexpected report %#v", report)
	}
	if accountStore.record.CurrentTier != domain.PlanPro {
		t.Fatalf("expected upgraded account tier, got %#v", accountStore.record)
	}
	if subscriptionStore.record.SubscriptionID != "sub_123" {
		t.Fatalf("expected subscription persistence, got %#v", subscriptionStore.record)
	}
	if reconciliationStore.record.SubscriptionID != "sub_123" {
		t.Fatalf("expected reconciliation persistence, got %#v", reconciliationStore.record)
	}
}
