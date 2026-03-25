package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/flowintel/flowintel/packages/domain"
)

func TestPostgresAlertDeliveryStoreListInboxAndChannels(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 9, 10, 11, 0, time.UTC)
	channelMetadata, err := json.Marshal(map[string]any{"format": "compact"})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	eventPayload, err := json.Marshal(map[string]any{"score_value": 92})
	if err != nil {
		t.Fatalf("marshal event payload: %v", err)
	}

	store := NewPostgresAlertDeliveryStore(&fakeAlertStoreQuerier{
		queryRows: map[string]pgx.Rows{
			listAlertInboxEventsSQL: &fakeAlertRows{
				rows: [][]any{
					{"evt_1", "rule_1", "owner_1", "cluster_score:evm:0x123", "dedup_1", "cluster_score", domain.AlertSeverityCritical, eventPayload, now, nil, now},
				},
			},
			listAlertDeliveryChannelsSQL: &fakeAlertRows{
				rows: [][]any{
					{"channel_1", "owner_1", "Ops Email", "email", "ops@example.com", channelMetadata, true, now, now},
				},
			},
			listEnabledAlertDeliveryChannelsSQL: &fakeAlertRows{
				rows: [][]any{
					{"channel_1", "owner_1", "Ops Email", "email", "ops@example.com", channelMetadata, true, now, now},
				},
			},
		},
	})

	events, err := store.ListAlertInboxEvents(context.Background(), "owner_1", AlertInboxQuery{Limit: 20})
	if err != nil {
		t.Fatalf("ListAlertInboxEvents returned error: %v", err)
	}
	if len(events.Items) != 1 || events.Items[0].SignalType != "cluster_score" {
		t.Fatalf("unexpected inbox events %#v", events)
	}

	channels, err := store.ListAlertDeliveryChannels(context.Background(), "owner_1")
	if err != nil {
		t.Fatalf("ListAlertDeliveryChannels returned error: %v", err)
	}
	if len(channels) != 1 || channels[0].ChannelType != domain.AlertChannelTypeEmail {
		t.Fatalf("unexpected channels %#v", channels)
	}

	enabled, err := store.ListEnabledAlertDeliveryChannels(context.Background(), "owner_1")
	if err != nil {
		t.Fatalf("ListEnabledAlertDeliveryChannels returned error: %v", err)
	}
	if len(enabled) != 1 || !enabled[0].IsEnabled {
		t.Fatalf("unexpected enabled channels %#v", enabled)
	}
}

func TestPostgresAlertDeliveryStoreChannelCrudAndAttemptLifecycle(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 9, 10, 11, 0, time.UTC)
	metadataJSON, err := json.Marshal(map[string]any{"format": "compact"})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	detailsJSON, err := json.Marshal(map[string]any{"message": "queued"})
	if err != nil {
		t.Fatalf("marshal details: %v", err)
	}

	querier := &fakeAlertStoreQuerier{
		queryRows: map[string]pgx.Rows{
			listAlertDeliveryChannelsSQL: &fakeAlertRows{
				rows: [][]any{
					{"channel_1", "owner_1", "Ops Email", "email", "ops@example.com", metadataJSON, true, now, now},
				},
			},
		},
		rowScans: map[string]func(dest ...any) error{
			createAlertDeliveryChannelSQL: func(dest ...any) error {
				*(dest[0].(*string)) = "channel_1"
				*(dest[1].(*string)) = "owner_1"
				*(dest[2].(*string)) = "Ops Email"
				*(dest[3].(*string)) = "email"
				*(dest[4].(*string)) = "ops@example.com"
				*(dest[5].(*[]byte)) = metadataJSON
				*(dest[6].(*bool)) = true
				*(dest[7].(*time.Time)) = now
				*(dest[8].(*time.Time)) = now
				return nil
			},
			updateAlertDeliveryChannelSQL: func(dest ...any) error {
				*(dest[0].(*string)) = "channel_1"
				*(dest[1].(*string)) = "owner_1"
				*(dest[2].(*string)) = "Ops Email Updated"
				*(dest[3].(*string)) = "email"
				*(dest[4].(*string)) = "ops+alerts@example.com"
				*(dest[5].(*[]byte)) = metadataJSON
				*(dest[6].(*bool)) = false
				*(dest[7].(*time.Time)) = now
				*(dest[8].(*time.Time)) = now.Add(time.Minute)
				return nil
			},
			createAlertDeliveryAttemptSQL: func(dest ...any) error {
				*(dest[0].(*string)) = "attempt_1"
				*(dest[1].(*string)) = "evt_1"
				*(dest[2].(*string)) = "channel_1"
				*(dest[3].(*string)) = "owner_1"
				*(dest[4].(*string)) = "evt_1:channel_1"
				*(dest[5].(*string)) = "email"
				*(dest[6].(*string)) = "ops@example.com"
				*(dest[7].(*string)) = "queued"
				*(dest[8].(*int)) = 0
				*(dest[9].(*[]byte)) = detailsJSON
				*(dest[10].(*sql.NullTime)) = sql.NullTime{Time: now, Valid: true}
				*(dest[11].(*sql.NullTime)) = sql.NullTime{}
				*(dest[12].(*sql.NullTime)) = sql.NullTime{}
				*(dest[13].(*time.Time)) = now
				return nil
			},
			updateAlertDeliveryAttemptSQL: func(dest ...any) error {
				*(dest[0].(*string)) = "attempt_1"
				*(dest[1].(*string)) = "evt_1"
				*(dest[2].(*string)) = "channel_1"
				*(dest[3].(*string)) = "owner_1"
				*(dest[4].(*string)) = "evt_1:channel_1"
				*(dest[5].(*string)) = "email"
				*(dest[6].(*string)) = "ops@example.com"
				*(dest[7].(*string)) = "delivered"
				*(dest[8].(*int)) = 202
				*(dest[9].(*[]byte)) = []byte(`{"message":"delivered"}`)
				*(dest[10].(*sql.NullTime)) = sql.NullTime{Time: now, Valid: true}
				*(dest[11].(*sql.NullTime)) = sql.NullTime{Time: now.Add(2 * time.Minute), Valid: true}
				*(dest[12].(*sql.NullTime)) = sql.NullTime{}
				*(dest[13].(*time.Time)) = now
				return nil
			},
		},
		execTags: map[string]pgconn.CommandTag{
			deleteAlertDeliveryChannelSQL: pgconn.NewCommandTag("DELETE 1"),
		},
	}

	store := NewPostgresAlertDeliveryStore(querier, querier)

	created, err := store.CreateAlertDeliveryChannel(context.Background(), AlertDeliveryChannelCreate{
		OwnerUserID: "owner_1",
		Label:       "Ops Email",
		ChannelType: "email",
		Target:      "ops@example.com",
		Metadata:    map[string]any{"format": "compact"},
		IsEnabled:   true,
	})
	if err != nil {
		t.Fatalf("CreateAlertDeliveryChannel returned error: %v", err)
	}
	if created.Label != "Ops Email" || created.ChannelType != domain.AlertChannelTypeEmail {
		t.Fatalf("unexpected created channel %#v", created)
	}

	updated, err := store.UpdateAlertDeliveryChannel(context.Background(), AlertDeliveryChannelUpdate{
		OwnerUserID: "owner_1",
		ChannelID:   "channel_1",
		Label:       "Ops Email Updated",
		Target:      "ops+alerts@example.com",
		Metadata:    map[string]any{"format": "compact"},
		IsEnabled:   false,
	})
	if err != nil {
		t.Fatalf("UpdateAlertDeliveryChannel returned error: %v", err)
	}
	if updated.Label != "Ops Email Updated" || updated.IsEnabled {
		t.Fatalf("unexpected updated channel %#v", updated)
	}

	attempt, err := store.CreateAlertDeliveryAttempt(context.Background(), AlertDeliveryAttemptCreate{
		AlertEventID: "evt_1",
		ChannelID:    "channel_1",
		OwnerUserID:  "owner_1",
		DeliveryKey:  "evt_1:channel_1",
		ChannelType:  domain.AlertChannelTypeEmail,
		Target:       "ops@example.com",
		Status:       domain.AlertDeliveryStatusQueued,
		Details:      map[string]any{"message": "queued"},
		AttemptedAt:  ptrTime(now),
	})
	if err != nil {
		t.Fatalf("CreateAlertDeliveryAttempt returned error: %v", err)
	}
	if attempt.Status != domain.AlertDeliveryStatusQueued {
		t.Fatalf("unexpected attempt %#v", attempt)
	}

	completed, err := store.UpdateAlertDeliveryAttempt(context.Background(), AlertDeliveryAttemptUpdate{
		AttemptID:    "attempt_1",
		Status:       domain.AlertDeliveryStatusDelivered,
		ResponseCode: 202,
		Details:      map[string]any{"message": "delivered"},
		AttemptedAt:  ptrTime(now),
		DeliveredAt:  ptrTime(now.Add(2 * time.Minute)),
	})
	if err != nil {
		t.Fatalf("UpdateAlertDeliveryAttempt returned error: %v", err)
	}
	if completed.Status != domain.AlertDeliveryStatusDelivered || completed.ResponseCode != 202 {
		t.Fatalf("unexpected completed attempt %#v", completed)
	}

	if err := store.DeleteAlertDeliveryChannel(context.Background(), "owner_1", "channel_1"); err != nil {
		t.Fatalf("DeleteAlertDeliveryChannel returned error: %v", err)
	}
}

func TestPostgresAlertDeliveryStoreListRetryableAttempts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 9, 10, 11, 0, time.UTC)
	detailsJSON, err := json.Marshal(map[string]any{
		"retry_count":   1,
		"next_retry_at": now.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("marshal details: %v", err)
	}
	payloadJSON, err := json.Marshal(map[string]any{"score_value": 91})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	store := NewPostgresAlertDeliveryStore(&fakeAlertStoreQuerier{
		queryRows: map[string]pgx.Rows{
			listRetryableAlertDeliveryAttemptsSQL: &fakeAlertRows{
				rows: [][]any{
					{
						"attempt_1",
						"evt_1",
						"channel_1",
						"owner_1",
						"evt_1:channel_1",
						"telegram",
						"bot-token:12345",
						"failed",
						500,
						detailsJSON,
						sql.NullTime{Time: now.Add(-2 * time.Minute), Valid: true},
						sql.NullTime{},
						sql.NullTime{Time: now.Add(-time.Minute), Valid: true},
						now.Add(-2 * time.Minute),
						"evt_1",
						"rule_1",
						"owner_1",
						"cluster_score:evm:0x123",
						"dedup_1",
						"cluster_score",
						domain.AlertSeverityCritical,
						payloadJSON,
						now.Add(-time.Minute),
						now.Add(-time.Minute),
					},
				},
			},
		},
	})

	candidates, err := store.ListRetryableAlertDeliveryAttempts(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("ListRetryableAlertDeliveryAttempts returned error: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 retry candidate, got %d", len(candidates))
	}
	if candidates[0].Attempt.ChannelType != domain.AlertChannelTypeTelegram {
		t.Fatalf("unexpected candidate attempt %#v", candidates[0].Attempt)
	}
	if candidates[0].Event.SignalType != "cluster_score" {
		t.Fatalf("unexpected candidate event %#v", candidates[0].Event)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
