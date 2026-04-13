package service

import (
	"context"
	"testing"
	"time"

	"github.com/qorvi/qorvi/apps/api/internal/repository"
)

func TestAdminConsoleServiceListAndMutateResources(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryAdminConsoleRepository()
	repo.SeedQuotaSnapshots([]repository.AdminQuotaSnapshot{{
		Provider:      "alchemy",
		Status:        "warning",
		Limit:         5000,
		Used:          3200,
		Reserved:      0,
		WindowStart:   time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC),
		LastCheckedAt: time.Date(2026, time.March, 21, 3, 0, 0, 0, time.UTC),
	}})
	repo.SeedObservabilitySnapshot(repository.AdminObservabilitySnapshot{
		ProviderUsage: []repository.AdminProviderUsageSnapshot{{
			Provider:     "alchemy",
			Status:       "healthy",
			Used24h:      120,
			Error24h:     3,
			AvgLatencyMs: 210,
			LastSeenAt:   ptrTime(time.Date(2026, time.March, 21, 3, 55, 0, 0, time.UTC)),
		}},
		Ingest: repository.AdminIngestSnapshot{
			LastBackfillAt:   ptrTime(time.Date(2026, time.March, 21, 3, 50, 0, 0, time.UTC)),
			LastWebhookAt:    ptrTime(time.Date(2026, time.March, 21, 3, 58, 0, 0, time.UTC)),
			FreshnessSeconds: 120,
			LagStatus:        "healthy",
		},
		AlertDelivery: repository.AdminAlertDeliverySnapshot{
			Attempts24h:    12,
			Delivered24h:   11,
			Failed24h:      1,
			RetryableCount: 1,
			LastFailureAt:  ptrTime(time.Date(2026, time.March, 21, 3, 40, 0, 0, time.UTC)),
		},
		WalletTracking: repository.AdminWalletTrackingSnapshot{
			CandidateCount:  4,
			TrackedCount:    8,
			LabeledCount:    3,
			ScoredCount:     2,
			StaleCount:      1,
			SuppressedCount: 0,
		},
		TrackingSubscriptions: repository.AdminWalletTrackingSubscriptionSnapshot{
			PendingCount: 1,
			ActiveCount:  5,
			ErroredCount: 1,
			PausedCount:  0,
			LastEventAt:  ptrTime(time.Date(2026, time.March, 21, 3, 59, 0, 0, time.UTC)),
		},
		RecentRuns: []repository.AdminJobHealthSnapshot{{
			JobName:             "wallet-backfill-drain-batch",
			LastStatus:          "succeeded",
			LastStartedAt:       time.Date(2026, time.March, 21, 3, 57, 0, 0, time.UTC),
			LastSuccessAt:       ptrTime(time.Date(2026, time.March, 21, 3, 58, 0, 0, time.UTC)),
			MinutesSinceSuccess: 2,
		}},
		RecentFailures: []repository.AdminFailureSnapshot{{
			Source:     "provider",
			Kind:       "alchemy",
			OccurredAt: time.Date(2026, time.March, 21, 3, 40, 0, 0, time.UTC),
			Summary:    "transfers.backfill returned 500",
			Details:    map[string]any{"status_code": 500},
		}},
	})

	svc := NewAdminConsoleService(repo)
	svc.Now = func() time.Time {
		return time.Date(2026, time.March, 21, 4, 0, 0, 0, time.UTC)
	}

	label, err := svc.UpsertLabel(context.Background(), "admin", "admin_1", UpsertAdminLabelRequest{
		Name:        "CEX Hot Wallet",
		Description: "Known exchange wallet",
		Color:       "#F97316",
	})
	if err != nil {
		t.Fatalf("UpsertLabel returned error: %v", err)
	}
	if label.Name != "cex-hot-wallet" {
		t.Fatalf("unexpected label %#v", label)
	}

	suppression, err := svc.CreateSuppression(context.Background(), "admin", "admin_1", CreateAdminSuppressionRequest{
		Scope:  "wallet",
		Target: "evm:0x123",
		Reason: "Known treasury",
	})
	if err != nil {
		t.Fatalf("CreateSuppression returned error: %v", err)
	}
	if suppression.Scope != "wallet" || !suppression.Active {
		t.Fatalf("unexpected suppression %#v", suppression)
	}

	labels, err := svc.ListLabels(context.Background(), "operator")
	if err != nil {
		t.Fatalf("ListLabels returned error: %v", err)
	}
	if len(labels.Items) != 1 {
		t.Fatalf("expected 1 label, got %d", len(labels.Items))
	}

	quotas, err := svc.ListProviderQuotas(context.Background(), "operator")
	if err != nil {
		t.Fatalf("ListProviderQuotas returned error: %v", err)
	}
	if len(quotas.Items) != 1 || quotas.Items[0].Provider != "alchemy" {
		t.Fatalf("unexpected quotas %#v", quotas)
	}

	observability, err := svc.ListObservability(context.Background(), "operator")
	if err != nil {
		t.Fatalf("ListObservability returned error: %v", err)
	}
	if len(observability.ProviderUsage) != 1 || observability.ProviderUsage[0].Provider != "alchemy" {
		t.Fatalf("unexpected observability provider usage %#v", observability)
	}
	if observability.Ingest.LagStatus != "healthy" {
		t.Fatalf("unexpected ingest snapshot %#v", observability.Ingest)
	}
	if observability.WalletTracking.TrackedCount != 8 {
		t.Fatalf("unexpected wallet tracking snapshot %#v", observability.WalletTracking)
	}
	if observability.TrackingSubscriptions.ActiveCount != 5 {
		t.Fatalf("unexpected tracking subscription snapshot %#v", observability.TrackingSubscriptions)
	}
	if len(observability.RecentFailures) != 1 {
		t.Fatalf("unexpected recent failures %#v", observability.RecentFailures)
	}

	curated, err := svc.CreateCuratedList(context.Background(), "admin", "admin_1", CreateAdminCuratedListRequest{
		Name:  "Exchange Hot Wallets",
		Notes: "Operator curated exchange cohort",
		Tags:  []string{"exchange", "wallet"},
	})
	if err != nil {
		t.Fatalf("CreateCuratedList returned error: %v", err)
	}
	if curated.ItemCount != 0 || curated.Name == "" {
		t.Fatalf("unexpected curated list %#v", curated)
	}

	curated, err = svc.AddCuratedListItem(context.Background(), "admin", "admin_1", curated.ID, CreateAdminCuratedListItemRequest{
		ItemType: "wallet",
		ItemKey:  "evm:0x123",
		Tags:     []string{"priority"},
		Notes:    "Seed entity",
	})
	if err != nil {
		t.Fatalf("AddCuratedListItem returned error: %v", err)
	}
	if curated.ItemCount != 1 {
		t.Fatalf("expected 1 curated item, got %#v", curated)
	}

	curatedLists, err := svc.ListCuratedLists(context.Background(), "operator")
	if err != nil {
		t.Fatalf("ListCuratedLists returned error: %v", err)
	}
	if len(curatedLists.Items) != 1 || len(curatedLists.Items[0].Items) != 1 {
		t.Fatalf("unexpected curated lists %#v", curatedLists)
	}

	auditEntries, err := svc.ListAuditEntries(context.Background(), "operator", 10)
	if err != nil {
		t.Fatalf("ListAuditEntries returned error: %v", err)
	}
	if len(auditEntries.Items) < 3 {
		t.Fatalf("expected audit trail entries, got %#v", auditEntries)
	}
}

func TestAdminConsoleServiceBlocksOperatorMutations(t *testing.T) {
	t.Parallel()

	svc := NewAdminConsoleService(repository.NewInMemoryAdminConsoleRepository())
	if _, err := svc.UpsertLabel(context.Background(), "operator", "operator_1", UpsertAdminLabelRequest{
		Name:        "Ops",
		Description: "Ops",
		Color:       "#fff",
	}); err == nil {
		t.Fatal("expected operator mutation to be blocked")
	}
}

func ptrTime(value time.Time) *time.Time {
	value = value.UTC()
	return &value
}
