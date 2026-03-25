package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/db"
)

const workerModeWalletTrackingSubscriptionSync = "wallet-tracking-subscription-sync"

type walletTrackingRegistryReader interface {
	ListWalletRefsForRealtimeTracking(context.Context, string, int) ([]db.WalletRef, error)
}

type providerAddressReconciler interface {
	ReplaceWebhookAddresses(context.Context, string, []string) error
}

type TrackingSubscriptionSyncService struct {
	Registry          walletTrackingRegistryReader
	Tracking          db.WalletTrackingStateStore
	JobRuns           db.JobRunStore
	AlchemyReconciler providerAddressReconciler
	HeliusReconciler  providerAddressReconciler
	Now               func() time.Time
}

type TrackingSubscriptionSyncReport struct {
	AlchemyWallets int
	HeliusWallets  int
	Subscriptions  int
	PendingCount   int
	ActiveCount    int
	ErroredCount   int
}

func (s TrackingSubscriptionSyncService) RunBatch(ctx context.Context, limit int) (TrackingSubscriptionSyncReport, error) {
	if s.Registry == nil || s.Tracking == nil {
		return TrackingSubscriptionSyncReport{}, fmt.Errorf("tracking subscription sync dependencies are required")
	}
	if limit <= 0 {
		limit = 1000
	}

	startedAt := s.now().UTC()
	report, err := s.syncProviders(ctx, limit)
	if err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeWalletTrackingSubscriptionSync,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details:    map[string]any{"error": err.Error()},
		})
		return TrackingSubscriptionSyncReport{}, err
	}

	if err := s.recordJobRun(ctx, db.JobRunEntry{
		JobName:    workerModeWalletTrackingSubscriptionSync,
		Status:     db.JobRunStatusSucceeded,
		StartedAt:  startedAt,
		FinishedAt: pointerToTime(s.now().UTC()),
		Details: map[string]any{
			"alchemy_wallets": report.AlchemyWallets,
			"helius_wallets":  report.HeliusWallets,
			"subscriptions":   report.Subscriptions,
			"pending":         report.PendingCount,
			"active":          report.ActiveCount,
		},
	}); err != nil {
		return TrackingSubscriptionSyncReport{}, err
	}

	return report, nil
}

func (s TrackingSubscriptionSyncService) syncProviders(ctx context.Context, limit int) (TrackingSubscriptionSyncReport, error) {
	report := TrackingSubscriptionSyncReport{}
	for _, provider := range []string{"alchemy", "helius"} {
		refs, err := s.Registry.ListWalletRefsForRealtimeTracking(ctx, provider, limit)
		if err != nil {
			return report, fmt.Errorf("list tracked wallets for %s: %w", provider, err)
		}
		if provider == "alchemy" {
			report.AlchemyWallets = len(refs)
		} else {
			report.HeliusWallets = len(refs)
		}

		subscriptionKey := trackingSubscriptionRegistryKey(provider)
		status := trackingSubscriptionStatus(provider)
		configuredKey := trackingProviderWebhookKey(provider)
		now := s.now().UTC()

		for _, ref := range refs {
			metadata := map[string]any{
				"source":                    "tracking_registry_sync",
				"provider_subscription_key": configuredKey,
				"configured":                configuredKey != "",
			}
			if err := s.Tracking.UpsertWalletTrackingSubscription(ctx, db.WalletTrackingSubscription{
				Chain:           ref.Chain,
				Address:         ref.Address,
				Provider:        provider,
				SubscriptionKey: subscriptionKey,
				Status:          status,
				LastSyncedAt:    &now,
				Metadata:        metadata,
			}); err != nil {
				return report, fmt.Errorf("upsert tracking subscription for %s %s: %w", provider, ref.Address, err)
			}
			report.Subscriptions++
			if status == "active" {
				report.ActiveCount++
			} else {
				report.PendingCount++
			}
		}

		if err := s.reconcileProvider(ctx, provider, subscriptionKey, configuredKey, refs); err != nil {
			report.ActiveCount -= countStatus(provider, refs, status, "active")
			report.PendingCount -= countStatus(provider, refs, status, "pending")
			report.ErroredCount += len(refs)
			if markErr := s.markProviderErrored(ctx, provider, subscriptionKey, refs, err); markErr != nil {
				return report, fmt.Errorf("mark provider errored after reconcile failure: %w", markErr)
			}
		}
	}

	return report, nil
}

func buildTrackingSubscriptionSyncSummary(report TrackingSubscriptionSyncReport) string {
	return fmt.Sprintf(
		"Wallet tracking subscription sync complete (alchemy=%d, helius=%d, subscriptions=%d, active=%d, pending=%d, errored=%d)",
		report.AlchemyWallets,
		report.HeliusWallets,
		report.Subscriptions,
		report.ActiveCount,
		report.PendingCount,
		report.ErroredCount,
	)
}

func trackingSubscriptionSyncLimitFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("FLOWINTEL_TRACKING_SUBSCRIPTION_SYNC_LIMIT"))
	if raw == "" {
		return 1000
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return 1000
	}
	if parsed > 100000 {
		return 100000
	}
	return parsed
}

func trackingSubscriptionRegistryKey(provider string) string {
	return "flowintel:" + strings.ToLower(strings.TrimSpace(provider)) + ":address-activity"
}

func trackingSubscriptionStatus(provider string) string {
	if trackingProviderWebhookKey(provider) != "" {
		return "active"
	}
	return "pending"
}

func trackingProviderWebhookKey(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "alchemy":
		return strings.TrimSpace(os.Getenv("ALCHEMY_ADDRESS_ACTIVITY_WEBHOOK_ID"))
	case "helius":
		return strings.TrimSpace(os.Getenv("HELIUS_ADDRESS_ACTIVITY_WEBHOOK_ID"))
	default:
		return ""
	}
}

func (s TrackingSubscriptionSyncService) reconcileProvider(
	ctx context.Context,
	provider string,
	subscriptionKey string,
	configuredKey string,
	refs []db.WalletRef,
) error {
	reconciler := s.reconcilerForProvider(provider)
	if strings.TrimSpace(configuredKey) == "" || reconciler == nil || len(refs) == 0 {
		return nil
	}
	addresses := make([]string, 0, len(refs))
	for _, ref := range refs {
		addresses = append(addresses, ref.Address)
	}
	if err := reconciler.ReplaceWebhookAddresses(ctx, configuredKey, addresses); err != nil {
		return err
	}
	now := s.now().UTC()
	for _, ref := range refs {
		if err := s.Tracking.UpsertWalletTrackingSubscription(ctx, db.WalletTrackingSubscription{
			Chain:           ref.Chain,
			Address:         ref.Address,
			Provider:        provider,
			SubscriptionKey: subscriptionKey,
			Status:          "active",
			LastSyncedAt:    &now,
			Metadata: map[string]any{
				"source":                    "tracking_registry_sync",
				"provider_subscription_key": configuredKey,
				"configured":                true,
				"remote_reconciled":         true,
				"remote_address_count":      len(addresses),
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s TrackingSubscriptionSyncService) markProviderErrored(
	ctx context.Context,
	provider string,
	subscriptionKey string,
	refs []db.WalletRef,
	reconcileErr error,
) error {
	now := s.now().UTC()
	for _, ref := range refs {
		if err := s.Tracking.UpsertWalletTrackingSubscription(ctx, db.WalletTrackingSubscription{
			Chain:           ref.Chain,
			Address:         ref.Address,
			Provider:        provider,
			SubscriptionKey: subscriptionKey,
			Status:          "errored",
			LastSyncedAt:    &now,
			Metadata: map[string]any{
				"source":            "tracking_registry_sync",
				"remote_reconciled": false,
				"reconcile_error":   reconcileErr.Error(),
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s TrackingSubscriptionSyncService) reconcilerForProvider(provider string) providerAddressReconciler {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "alchemy":
		return s.AlchemyReconciler
	case "helius":
		return s.HeliusReconciler
	default:
		return nil
	}
}

func countStatus(_ string, refs []db.WalletRef, current string, target string) int {
	if current == target {
		return len(refs)
	}
	return 0
}

func (s TrackingSubscriptionSyncService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s TrackingSubscriptionSyncService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}
	return s.JobRuns.RecordJobRun(ctx, entry)
}
