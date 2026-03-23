package repository

import (
	"context"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestInMemoryAlertDeliveryRepositoryInboxAndChannelCrud(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryAlertDeliveryRepository()
	now := time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC)
	repo.SeedAlertEvent(domain.AlertEvent{
		ID:          "evt_1",
		AlertRuleID: "rule_1",
		OwnerUserID: "user_1",
		EventKey:    "shadow_exit:evm:0x123",
		DedupKey:    "dedup_1",
		SignalType:  "shadow_exit",
		Severity:    domain.AlertSeverityHigh,
		Payload:     map[string]any{"score_value": 88},
		ObservedAt:  now,
		CreatedAt:   now,
	})

	events, err := repo.ListAlertInboxEvents(context.Background(), "user_1", AlertInboxQuery{Limit: 10, SignalType: "shadow_exit"})
	if err != nil {
		t.Fatalf("ListAlertInboxEvents returned error: %v", err)
	}
	if len(events.Items) != 1 {
		t.Fatalf("expected 1 inbox event, got %d", len(events.Items))
	}

	created, err := repo.CreateAlertDeliveryChannel(context.Background(), domain.AlertDeliveryChannel{
		ID:          "channel_1",
		OwnerUserID: "user_1",
		Label:       "Ops Email",
		ChannelType: domain.AlertChannelTypeEmail,
		Target:      "ops@example.com",
		Metadata:    map[string]any{"format": "compact"},
		IsEnabled:   true,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("CreateAlertDeliveryChannel returned error: %v", err)
	}
	if created.ChannelType != domain.AlertChannelTypeEmail {
		t.Fatalf("unexpected created channel %#v", created)
	}

	channels, err := repo.ListAlertDeliveryChannels(context.Background(), "user_1")
	if err != nil {
		t.Fatalf("ListAlertDeliveryChannels returned error: %v", err)
	}
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}

	created.Label = "Ops Email Updated"
	created.Target = "ops+alerts@example.com"
	created.IsEnabled = false
	created.UpdatedAt = now.Add(time.Minute)
	updated, err := repo.UpdateAlertDeliveryChannel(context.Background(), created)
	if err != nil {
		t.Fatalf("UpdateAlertDeliveryChannel returned error: %v", err)
	}
	if updated.Label != "Ops Email Updated" || updated.IsEnabled {
		t.Fatalf("unexpected updated channel %#v", updated)
	}

	if err := repo.DeleteAlertDeliveryChannel(context.Background(), "user_1", "channel_1"); err != nil {
		t.Fatalf("DeleteAlertDeliveryChannel returned error: %v", err)
	}
}
