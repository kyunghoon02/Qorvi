package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/providers"
)

const workerModeWalletTrackingSubscriptionSync = "wallet-tracking-subscription-sync"

type walletTrackingRegistryReader interface {
	ListWalletRefsForRealtimeTracking(context.Context, string, int) ([]db.WalletRef, error)
}

type providerAddressReconciler interface {
	EnsureWebhookAddresses(context.Context, string, string, []string) (providers.WebhookEnsureResult, error)
}

type TrackingSubscriptionSyncService struct {
	Registry          walletTrackingRegistryReader
	Tracking          db.WalletTrackingStateStore
	JobRuns           db.JobRunStore
	AlchemyReconciler providerAddressReconciler
	HeliusReconciler  providerAddressReconciler
	WebhookBaseURL    string
	Now               func() time.Time
}

type providerSyncState struct {
	subscriptionKey string
	status          string
	metadata        map[string]any
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

		state, err := s.syncProviderState(ctx, provider, refs)
		if err != nil {
			report.ErroredCount += len(refs)
			if markErr := s.markProviderErrored(ctx, provider, trackingSubscriptionRegistryKey(provider), refs, err); markErr != nil {
				return report, fmt.Errorf("mark provider errored after reconcile failure: %w", markErr)
			}
			continue
		}

		now := s.now().UTC()
		for _, ref := range refs {
			if err := s.Tracking.UpsertWalletTrackingSubscription(ctx, db.WalletTrackingSubscription{
				Chain:           ref.Chain,
				Address:         ref.Address,
				Provider:        provider,
				SubscriptionKey: state.subscriptionKey,
				Status:          state.status,
				LastSyncedAt:    &now,
				Metadata:        cloneMetadata(state.metadata),
			}); err != nil {
				return report, fmt.Errorf("upsert tracking subscription for %s %s: %w", provider, ref.Address, err)
			}
			report.Subscriptions++
			if state.status == "active" {
				report.ActiveCount++
			} else {
				report.PendingCount++
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
	raw := strings.TrimSpace(os.Getenv("QORVI_TRACKING_SUBSCRIPTION_SYNC_LIMIT"))
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
	return "qorvi:" + strings.ToLower(strings.TrimSpace(provider)) + ":address-activity"
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

func (s TrackingSubscriptionSyncService) syncProviderState(
	ctx context.Context,
	provider string,
	refs []db.WalletRef,
) (providerSyncState, error) {
	subscriptionKey := trackingSubscriptionRegistryKey(provider)
	configuredKey := trackingProviderWebhookKey(provider)
	callbackURL := s.callbackURLForProvider(provider)
	reconciler := s.reconcilerForProvider(provider)
	metadata := map[string]any{
		"source":                    "tracking_registry_sync",
		"provider_subscription_key": configuredKey,
		"configured":                configuredKey != "",
	}
	if callbackURL != "" {
		metadata["provider_callback_url"] = callbackURL
	}

	if len(refs) == 0 {
		return providerSyncState{
			subscriptionKey: subscriptionKey,
			status:          trackingSubscriptionStatus(provider),
			metadata:        metadata,
		}, nil
	}

	if reconciler == nil {
		metadata["remote_reconciled"] = false
		metadata["pending_reason"] = "provider_reconciler_not_configured"
		return providerSyncState{
			subscriptionKey: subscriptionKey,
			status:          "pending",
			metadata:        metadata,
		}, nil
	}
	if strings.TrimSpace(configuredKey) == "" && callbackURL == "" {
		metadata["remote_reconciled"] = false
		metadata["pending_reason"] = "webhook_public_base_url_required"
		return providerSyncState{
			subscriptionKey: subscriptionKey,
			status:          "pending",
			metadata:        metadata,
		}, nil
	}

	addresses := make([]string, 0, len(refs))
	for _, ref := range refs {
		addresses = append(addresses, ref.Address)
	}

	result, err := reconciler.EnsureWebhookAddresses(ctx, configuredKey, callbackURL, addresses)
	if err != nil {
		return providerSyncState{}, err
	}

	metadata["remote_reconciled"] = true
	metadata["remote_address_count"] = len(addresses)
	metadata["provider_subscription_key"] = result.WebhookID
	metadata["configured"] = configuredKey != ""
	if result.Created {
		metadata["provider_subscription_created"] = true
	}
	if result.Discovered {
		metadata["provider_subscription_discovered"] = true
	}

	return providerSyncState{
		subscriptionKey: subscriptionKey,
		status:          "active",
		metadata:        metadata,
	}, nil
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

func (s TrackingSubscriptionSyncService) callbackURLForProvider(provider string) string {
	baseURL := publicWebhookBaseURL(s.WebhookBaseURL)
	if baseURL == "" {
		return ""
	}

	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "alchemy":
		return baseURL + "/v1/webhooks/providers/alchemy/address-activity"
	case "helius":
		return baseURL + "/v1/webhooks/providers/helius/address-activity"
	default:
		return ""
	}
}

func (s TrackingSubscriptionSyncService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func cloneMetadata(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func publicWebhookBaseURL(fallback string) string {
	baseURL := strings.TrimSpace(os.Getenv("QORVI_PROVIDER_WEBHOOK_BASE_URL"))
	if baseURL == "" {
		baseURL = strings.TrimSpace(fallback)
	}
	if baseURL == "" {
		return ""
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return ""
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" || strings.EqualFold(host, "localhost") {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified()) {
		return ""
	}
	return strings.TrimRight(parsed.String(), "/")
}

func (s TrackingSubscriptionSyncService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}
	return s.JobRuns.RecordJobRun(ctx, entry)
}
