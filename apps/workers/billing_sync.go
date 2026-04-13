package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/billing"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

const workerModeBillingSubscriptionSync = "billing-subscription-sync"

type BillingSubscriptionSyncService struct {
	Accounts        db.BillingAccountSyncReader
	AccountStore    db.BillingAccountStore
	Subscriptions   db.BillingSubscriptionStore
	Reconciliations db.BillingSubscriptionReconciliationStore
	StripeClient    billing.StripeClient
	StripeConfig    billing.StripeConfig
	Now             func() time.Time
}

type BillingSubscriptionSyncReport struct {
	AccountsScanned      int
	SubscriptionsSynced  int
	AccountsUpdated      int
	ReconciliationsSaved int
}

func (s BillingSubscriptionSyncService) RunBatch(ctx context.Context, limit int) (BillingSubscriptionSyncReport, error) {
	if s.Accounts == nil || s.AccountStore == nil || s.Subscriptions == nil || s.StripeClient == nil {
		return BillingSubscriptionSyncReport{}, fmt.Errorf("billing subscription sync dependencies are required")
	}
	if strings.TrimSpace(s.StripeConfig.SecretKey) == "" {
		return BillingSubscriptionSyncReport{}, fmt.Errorf("stripe secret key is required")
	}
	if limit <= 0 {
		limit = 25
	}

	accounts, err := s.Accounts.ListBillingAccountsForSubscriptionSync(ctx, limit)
	if err != nil {
		return BillingSubscriptionSyncReport{}, err
	}

	report := BillingSubscriptionSyncReport{AccountsScanned: len(accounts)}
	for _, account := range accounts {
		subscriptionID := strings.TrimSpace(account.ActiveSubscriptionID)
		if subscriptionID == "" {
			continue
		}

		subscription, err := s.StripeClient.GetSubscription(ctx, s.StripeConfig, subscriptionID)
		if err != nil {
			return report, fmt.Errorf("sync stripe subscription %s: %w", subscriptionID, err)
		}
		if strings.TrimSpace(subscription.CustomerEmail) == "" {
			subscription.CustomerEmail = strings.TrimSpace(account.Email)
		}

		subscription, err = s.Subscriptions.UpsertBillingSubscription(ctx, subscription)
		if err != nil {
			return report, fmt.Errorf("persist stripe subscription %s: %w", subscriptionID, err)
		}
		report.SubscriptionsSynced++

		now := s.now().UTC()
		nextTier := subscription.Tier
		if shouldDowngradeSubscription(subscription.Status) {
			nextTier = domain.PlanFree
		}

		updatedAccount, err := s.AccountStore.UpsertBillingAccount(ctx, db.BillingAccountRecord{
			OwnerUserID:          account.OwnerUserID,
			Email:                firstNonEmpty(strings.TrimSpace(account.Email), strings.TrimSpace(subscription.CustomerEmail)),
			CurrentTier:          nextTier,
			StripeCustomerID:     firstNonEmpty(strings.TrimSpace(subscription.CustomerID), strings.TrimSpace(account.StripeCustomerID)),
			ActiveSubscriptionID: strings.TrimSpace(subscription.SubscriptionID),
			CurrentPriceID:       firstNonEmpty(strings.TrimSpace(subscription.StripePriceID), strings.TrimSpace(account.CurrentPriceID)),
			Status:               string(subscription.Status),
			CurrentPeriodEnd:     cloneBillingTimePointer(&subscription.CurrentPeriodEnd),
			CreatedAt:            account.CreatedAt,
			UpdatedAt:            now,
		})
		if err != nil {
			return report, fmt.Errorf("update billing account %s: %w", account.OwnerUserID, err)
		}
		report.AccountsUpdated++

		if s.Reconciliations != nil {
			reconciledAt := now
			if err := s.Reconciliations.RecordBillingSubscriptionReconciliation(ctx, billing.NormalizeStripeSubscriptionReconciliationRecord(
				billing.StripeSubscriptionReconciliationRecord{
					EventID:        buildBillingSyncEventID(updatedAccount, now),
					Provider:       "stripe",
					CustomerID:     updatedAccount.StripeCustomerID,
					SubscriptionID: updatedAccount.ActiveSubscriptionID,
					PreviousTier:   account.CurrentTier,
					CurrentTier:    updatedAccount.CurrentTier,
					StripePriceID:  updatedAccount.CurrentPriceID,
					Status:         updatedAccount.Status,
					ObservedAt:     now,
					ReconciledAt:   &reconciledAt,
					Metadata: map[string]string{
						"source": "subscription_sync",
					},
				},
			)); err != nil {
				return report, fmt.Errorf("record subscription reconciliation %s: %w", subscription.SubscriptionID, err)
			}
			report.ReconciliationsSaved++
		}
	}

	return report, nil
}

func buildBillingSubscriptionSyncSummary(report BillingSubscriptionSyncReport) string {
	return fmt.Sprintf(
		"Billing subscription sync complete (accounts=%d, subscriptions=%d, updated=%d, reconciliations=%d)",
		report.AccountsScanned,
		report.SubscriptionsSynced,
		report.AccountsUpdated,
		report.ReconciliationsSaved,
	)
}

func billingSubscriptionSyncLimitFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("QORVI_BILLING_SYNC_LIMIT"))
	if raw == "" {
		return 25
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return 25
	}
	if parsed > 250 {
		return 250
	}
	return parsed
}

func buildBillingSyncEventID(account db.BillingAccountRecord, observedAt time.Time) string {
	return fmt.Sprintf(
		"sync_%s_%s",
		strings.TrimSpace(account.ActiveSubscriptionID),
		observedAt.UTC().Format("20060102T150405Z"),
	)
}

func shouldDowngradeSubscription(status billing.StripeSubscriptionStatus) bool {
	switch status {
	case billing.StripeSubscriptionStatusCanceled:
		return true
	default:
		return false
	}
}

func firstNonEmpty(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func cloneBillingTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func (s BillingSubscriptionSyncService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
