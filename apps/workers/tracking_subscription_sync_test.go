package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

type fakeWalletTrackingRegistryReader struct {
	refsByProvider map[string][]db.WalletRef
	errByProvider  map[string]error
	calls          []string
}

type fakeProviderAddressReconciler struct {
	webhookIDs []string
	addresses  [][]string
	err        error
}

func (f *fakeWalletTrackingRegistryReader) ListWalletRefsForRealtimeTracking(
	_ context.Context,
	provider string,
	_ int,
) ([]db.WalletRef, error) {
	f.calls = append(f.calls, provider)
	if err := f.errByProvider[provider]; err != nil {
		return nil, err
	}
	return append([]db.WalletRef(nil), f.refsByProvider[provider]...), nil
}

func (f *fakeProviderAddressReconciler) ReplaceWebhookAddresses(_ context.Context, webhookID string, addresses []string) error {
	if f.err != nil {
		return f.err
	}
	f.webhookIDs = append(f.webhookIDs, webhookID)
	f.addresses = append(f.addresses, append([]string(nil), addresses...))
	return nil
}

func TestTrackingSubscriptionSyncServiceRunBatchUpsertsSubscriptions(t *testing.T) {
	t.Setenv("ALCHEMY_ADDRESS_ACTIVITY_WEBHOOK_ID", "wh_alchemy_live")
	t.Setenv("HELIUS_ADDRESS_ACTIVITY_WEBHOOK_ID", "")

	registry := &fakeWalletTrackingRegistryReader{
		refsByProvider: map[string][]db.WalletRef{
			"alchemy": {
				{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
			},
			"helius": {
				{Chain: domain.ChainSolana, Address: "So11111111111111111111111111111111111111112"},
			},
		},
	}
	tracking := &fakeWalletTrackingStateStore{}
	jobRuns := &fakeJobRunStore{}
	alchemy := &fakeProviderAddressReconciler{}
	now := time.Date(2026, time.March, 25, 12, 0, 0, 0, time.UTC)

	report, err := (TrackingSubscriptionSyncService{
		Registry:          registry,
		Tracking:          tracking,
		JobRuns:           jobRuns,
		AlchemyReconciler: alchemy,
		Now:               func() time.Time { return now },
	}).RunBatch(t.Context(), 100)
	if err != nil {
		t.Fatalf("RunBatch returned error: %v", err)
	}

	if report.AlchemyWallets != 1 || report.HeliusWallets != 1 {
		t.Fatalf("unexpected wallet counts %#v", report)
	}
	if report.Subscriptions != 2 || report.ActiveCount != 1 || report.PendingCount != 1 || report.ErroredCount != 0 {
		t.Fatalf("unexpected sync report %#v", report)
	}
	if len(tracking.subscriptions) != 3 {
		t.Fatalf("expected 3 subscription upserts, got %d", len(tracking.subscriptions))
	}
	var (
		alchemySeen bool
		heliusSeen  bool
	)
	for _, subscription := range tracking.subscriptions {
		switch subscription.Provider {
		case "alchemy":
			alchemySeen = true
			if subscription.SubscriptionKey != "flowintel:alchemy:address-activity" {
				t.Fatalf("unexpected alchemy subscription key %#v", subscription)
			}
		case "helius":
			heliusSeen = true
			if subscription.Status != "pending" {
				t.Fatalf("unexpected helius subscription %#v", subscription)
			}
		}
	}
	if !alchemySeen || !heliusSeen {
		t.Fatalf("expected both alchemy and helius subscriptions, got %#v", tracking.subscriptions)
	}
	if len(alchemy.webhookIDs) != 1 || alchemy.webhookIDs[0] != "wh_alchemy_live" {
		t.Fatalf("unexpected alchemy reconcile calls %#v", alchemy.webhookIDs)
	}
	if len(jobRuns.entries) != 1 || jobRuns.entries[0].Status != db.JobRunStatusSucceeded {
		t.Fatalf("expected succeeded job run, got %#v", jobRuns.entries)
	}
}

func TestTrackingSubscriptionSyncServiceRunBatchReturnsRegistryFailure(t *testing.T) {
	registry := &fakeWalletTrackingRegistryReader{
		errByProvider: map[string]error{
			"alchemy": errors.New("registry unavailable"),
		},
	}
	jobRuns := &fakeJobRunStore{}

	_, err := (TrackingSubscriptionSyncService{
		Registry: registry,
		Tracking: &fakeWalletTrackingStateStore{},
		JobRuns:  jobRuns,
	}).RunBatch(t.Context(), 100)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(jobRuns.entries) != 1 || jobRuns.entries[0].Status != db.JobRunStatusFailed {
		t.Fatalf("expected failed job run, got %#v", jobRuns.entries)
	}
}

func TestTrackingSubscriptionHelpers(t *testing.T) {
	t.Setenv("FLOWINTEL_TRACKING_SUBSCRIPTION_SYNC_LIMIT", "")
	if got := trackingSubscriptionSyncLimitFromEnv(); got != 1000 {
		t.Fatalf("unexpected default sync limit %d", got)
	}

	t.Setenv("FLOWINTEL_TRACKING_SUBSCRIPTION_SYNC_LIMIT", "2500")
	if got := trackingSubscriptionSyncLimitFromEnv(); got != 2500 {
		t.Fatalf("unexpected parsed sync limit %d", got)
	}

	t.Setenv("ALCHEMY_ADDRESS_ACTIVITY_WEBHOOK_ID", "")
	if got := trackingSubscriptionStatus("alchemy"); got != "pending" {
		t.Fatalf("expected pending without configured webhook id, got %q", got)
	}

	t.Setenv("ALCHEMY_ADDRESS_ACTIVITY_WEBHOOK_ID", "wh_live")
	if got := trackingSubscriptionStatus("alchemy"); got != "active" {
		t.Fatalf("expected active with configured webhook id, got %q", got)
	}
}

func TestTrackingSubscriptionSyncServiceMarksErroredOnReconcileFailure(t *testing.T) {
	t.Setenv("ALCHEMY_ADDRESS_ACTIVITY_WEBHOOK_ID", "wh_alchemy_live")

	tracking := &fakeWalletTrackingStateStore{}
	report, err := (TrackingSubscriptionSyncService{
		Registry: &fakeWalletTrackingRegistryReader{
			refsByProvider: map[string][]db.WalletRef{
				"alchemy": {
					{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
				},
			},
		},
		Tracking:          tracking,
		AlchemyReconciler: &fakeProviderAddressReconciler{err: errors.New("alchemy reconcile failed")},
	}).RunBatch(t.Context(), 100)
	if err != nil {
		t.Fatalf("RunBatch returned error: %v", err)
	}
	if report.ErroredCount != 1 {
		t.Fatalf("expected errored count, got %#v", report)
	}
	if len(tracking.subscriptions) != 2 {
		t.Fatalf("expected initial + errored upserts, got %d", len(tracking.subscriptions))
	}
	last := tracking.subscriptions[len(tracking.subscriptions)-1]
	if last.Status != "errored" {
		t.Fatalf("expected errored subscription status, got %#v", last)
	}
}
