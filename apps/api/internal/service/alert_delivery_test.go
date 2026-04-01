package service

import (
	"context"
	"testing"
	"time"

	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/packages/domain"
)

func TestAlertDeliveryServiceInboxAndChannelCrudForOpenAccessOwner(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryAlertDeliveryRepository()
	now := time.Date(2026, time.March, 21, 12, 30, 0, 0, time.UTC)
	repo.SeedAlertEvent(domain.AlertEvent{
		ID:          "evt_1",
		AlertRuleID: "rule_1",
		OwnerUserID: "user_1",
		EventKey:    "cluster_score:evm:0x123",
		DedupKey:    "dedup_1",
		SignalType:  "cluster_score",
		Severity:    domain.AlertSeverityCritical,
		Payload:     map[string]any{"score_value": 91},
		ObservedAt:  now,
		CreatedAt:   now,
	})

	svc := NewAlertDeliveryService(repo)
	svc.Now = func() time.Time { return now }

	inbox, err := svc.ListInboxEvents(context.Background(), "user_1", domain.PlanFree, AlertInboxQuery{Limit: 10})
	if err != nil {
		t.Fatalf("ListInboxEvents returned error: %v", err)
	}
	if len(inbox.Items) != 1 || inbox.Items[0].SignalType != "cluster_score" {
		t.Fatalf("unexpected inbox %#v", inbox)
	}

	created, err := svc.CreateAlertDeliveryChannel(context.Background(), "user_1", domain.PlanFree, CreateAlertDeliveryChannelRequest{
		Label:       "Ops Email",
		ChannelType: "email",
		Target:      "ops@example.com",
		Metadata:    map[string]any{"format": "compact"},
		IsEnabled:   boolPtr(true),
	})
	if err != nil {
		t.Fatalf("CreateAlertDeliveryChannel returned error: %v", err)
	}
	if created.ChannelType != "email" {
		t.Fatalf("unexpected created channel %#v", created)
	}

	list, err := svc.ListAlertDeliveryChannels(context.Background(), "user_1", domain.PlanFree)
	if err != nil {
		t.Fatalf("ListAlertDeliveryChannels returned error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(list.Items))
	}

	updated, err := svc.UpdateAlertDeliveryChannel(context.Background(), "user_1", domain.PlanFree, created.ID, UpdateAlertDeliveryChannelRequest{
		Label:     "Ops Email Updated",
		Target:    "ops+alerts@example.com",
		Metadata:  map[string]any{"format": "long"},
		IsEnabled: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("UpdateAlertDeliveryChannel returned error: %v", err)
	}
	if updated.Label != "Ops Email Updated" || updated.IsEnabled {
		t.Fatalf("unexpected updated channel %#v", updated)
	}

	if err := svc.DeleteAlertDeliveryChannel(context.Background(), "user_1", domain.PlanFree, created.ID); err != nil {
		t.Fatalf("DeleteAlertDeliveryChannel returned error: %v", err)
	}
}

func TestAlertDeliveryServiceAllowsFreeTierInboxAccess(t *testing.T) {
	t.Parallel()

	svc := NewAlertDeliveryService(repository.NewInMemoryAlertDeliveryRepository())
	if _, err := svc.ListInboxEvents(context.Background(), "user_1", domain.PlanFree, AlertInboxQuery{}); err != nil {
		t.Fatalf("expected free tier inbox access, got %v", err)
	}
}
