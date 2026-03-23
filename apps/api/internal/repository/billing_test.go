package repository

import (
	"context"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestInMemoryBillingRepositoryUpsertAndFindAccount(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryBillingRepository()
	now := time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)

	_, err := repo.UpsertBillingAccount(context.Background(), BillingAccount{
		OwnerUserID:      "user_123",
		Email:            "ops@whalegraph.test",
		CurrentTier:      domain.PlanPro,
		Status:           "active",
		CurrentPeriodEnd: &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, err := repo.FindBillingAccount(context.Background(), "user_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account.CurrentTier != domain.PlanPro {
		t.Fatalf("expected pro tier, got %q", account.CurrentTier)
	}
}

func TestInMemoryBillingRepositoryRecordWebhookEvent(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryBillingRepository()
	recorded, err := repo.RecordWebhookEvent(context.Background(), BillingWebhookEvent{
		ProviderEventID: "evt_123",
		EventType:       "checkout.session.completed",
		OwnerUserID:     "user_123",
		PlanTier:        domain.PlanPro,
		Status:          "processed",
		Payload:         map[string]any{"ownerUserId": "user_123"},
		ReceivedAt:      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recorded.ProviderEventID != "evt_123" {
		t.Fatalf("unexpected provider event id %q", recorded.ProviderEventID)
	}
}
