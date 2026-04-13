package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeAlertDeliveryChannelReader struct {
	channels []domain.AlertDeliveryChannel
	err      error
}

func (r *fakeAlertDeliveryChannelReader) ListEnabledAlertDeliveryChannels(_ context.Context, _ string) ([]domain.AlertDeliveryChannel, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.channels, nil
}

type fakeAlertDeliveryAttemptRecorder struct {
	creates []db.AlertDeliveryAttemptCreate
	updates []db.AlertDeliveryAttemptUpdate
}

func (r *fakeAlertDeliveryAttemptRecorder) CreateAlertDeliveryAttempt(_ context.Context, create db.AlertDeliveryAttemptCreate) (domain.AlertDeliveryAttempt, error) {
	r.creates = append(r.creates, create)
	return domain.AlertDeliveryAttempt{
		ID:           "attempt_" + create.ChannelID,
		AlertEventID: create.AlertEventID,
		ChannelID:    create.ChannelID,
		OwnerUserID:  create.OwnerUserID,
		DeliveryKey:  create.DeliveryKey,
		ChannelType:  create.ChannelType,
		Target:       create.Target,
		Status:       create.Status,
		Details:      create.Details,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

func (r *fakeAlertDeliveryAttemptRecorder) UpdateAlertDeliveryAttempt(_ context.Context, update db.AlertDeliveryAttemptUpdate) (domain.AlertDeliveryAttempt, error) {
	r.updates = append(r.updates, update)
	return domain.AlertDeliveryAttempt{
		ID:           update.AttemptID,
		Status:       update.Status,
		ResponseCode: update.ResponseCode,
		Details:      update.Details,
		AttemptedAt:  update.AttemptedAt,
		DeliveredAt:  update.DeliveredAt,
		FailedAt:     update.FailedAt,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

type fakeAlertEmailSender struct{}

func (s fakeAlertEmailSender) SendAlertEmail(_ context.Context, request AlertEmailRequest) (int, string, error) {
	if request.To == "ops@example.com" {
		return 250, "smtp accepted", nil
	}
	return 0, "", errors.New("unexpected target")
}

type fakeAlertDiscordSender struct{}

func (s fakeAlertDiscordSender) SendDiscordWebhook(_ context.Context, request AlertDiscordWebhookRequest) (int, string, error) {
	if request.WebhookURL == "https://discord.example/webhook" {
		return 204, "", nil
	}
	return 500, "", errors.New("unexpected discord webhook")
}

type fakeAlertTelegramSender struct{}

func (s fakeAlertTelegramSender) SendTelegramMessage(_ context.Context, request AlertTelegramRequest) (int, string, error) {
	if request.BotToken == "bot-token" && request.ChatID == "12345" {
		return 200, `{"ok":true}`, nil
	}
	return 500, "", errors.New("unexpected telegram target")
}

func TestWalletSignalDeliveryDispatcherCreatesAttemptsAndDelivers(t *testing.T) {
	t.Parallel()

	recorder := &fakeAlertDeliveryAttemptRecorder{}
	dispatcher := WalletSignalDeliveryDispatcher{
		Channels: &fakeAlertDeliveryChannelReader{
			channels: []domain.AlertDeliveryChannel{
				{
					ID:          "channel_email",
					OwnerUserID: "user_1",
					Label:       "Ops Email",
					ChannelType: domain.AlertChannelTypeEmail,
					Target:      "ops@example.com",
					Metadata:    map[string]any{},
					IsEnabled:   true,
				},
				{
					ID:          "channel_discord",
					OwnerUserID: "user_1",
					Label:       "Ops Discord",
					ChannelType: domain.AlertChannelTypeDiscordWebhook,
					Target:      "https://discord.example/webhook",
					Metadata:    map[string]any{},
					IsEnabled:   true,
				},
				{
					ID:          "channel_telegram",
					OwnerUserID: "user_1",
					Label:       "Ops Telegram",
					ChannelType: domain.AlertChannelTypeTelegram,
					Target:      "bot-token:12345",
					Metadata:    map[string]any{},
					IsEnabled:   true,
				},
			},
		},
		Attempts: recorder,
		Email:    fakeAlertEmailSender{},
		Discord:  fakeAlertDiscordSender{},
		Telegram: fakeAlertTelegramSender{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 21, 13, 0, 0, 0, time.UTC)
		},
	}

	report, err := dispatcher.DeliverAlertEvent(context.Background(), domain.AlertEvent{
		ID:          "evt_1",
		AlertRuleID: "rule_1",
		OwnerUserID: "user_1",
		EventKey:    "cluster_score:evm:0x123",
		DedupKey:    "dedup_1",
		SignalType:  "cluster_score",
		Severity:    domain.AlertSeverityCritical,
		Payload:     map[string]any{"score_value": 91},
		ObservedAt:  time.Date(2026, time.March, 21, 12, 59, 0, 0, time.UTC),
		CreatedAt:   time.Date(2026, time.March, 21, 12, 59, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("DeliverAlertEvent returned error: %v", err)
	}
	if report.MatchedChannels != 3 || report.AttemptsCreated != 3 || report.Delivered != 3 {
		t.Fatalf("unexpected delivery report %#v", report)
	}
	if len(recorder.creates) != 3 || len(recorder.updates) != 3 {
		t.Fatalf("unexpected attempt activity creates=%d updates=%d", len(recorder.creates), len(recorder.updates))
	}
}

func TestWalletSignalDeliveryDispatcherRetryAlertDeliveryAttemptRequeuesFailures(t *testing.T) {
	t.Parallel()

	recorder := &fakeAlertDeliveryAttemptRecorder{}
	dispatcher := WalletSignalDeliveryDispatcher{
		Attempts:       recorder,
		Discord:        fakeAlertDiscordSender{},
		RetryLimit:     3,
		RetryBaseDelay: time.Minute,
		Now: func() time.Time {
			return time.Date(2026, time.March, 21, 13, 0, 0, 0, time.UTC)
		},
	}

	result, err := dispatcher.RetryAlertDeliveryAttempt(context.Background(), db.AlertDeliveryRetryCandidate{
		Attempt: domain.AlertDeliveryAttempt{
			ID:           "attempt_discord",
			AlertEventID: "evt_1",
			ChannelID:    "channel_discord",
			OwnerUserID:  "user_1",
			ChannelType:  domain.AlertChannelTypeDiscordWebhook,
			Target:       "https://discord.example/invalid",
			Status:       domain.AlertDeliveryStatusFailed,
			Details: map[string]any{
				"retry_count": 0,
			},
		},
		Event: domain.AlertEvent{
			ID:          "evt_1",
			AlertRuleID: "rule_1",
			OwnerUserID: "user_1",
			EventKey:    "shadow_exit:evm:0x123",
			DedupKey:    "dedup_1",
			SignalType:  "shadow_exit",
			Severity:    domain.AlertSeverityHigh,
			Payload:     map[string]any{"score_value": 82},
			ObservedAt:  time.Date(2026, time.March, 21, 12, 59, 0, 0, time.UTC),
			CreatedAt:   time.Date(2026, time.March, 21, 12, 59, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("RetryAlertDeliveryAttempt returned error: %v", err)
	}
	if !result.Requeued || result.Delivered || result.Exhausted {
		t.Fatalf("unexpected retry result %#v", result)
	}
	if len(recorder.updates) != 1 {
		t.Fatalf("expected 1 attempt update, got %d", len(recorder.updates))
	}
	if got := recorder.updates[0].Details["retry_count"]; got != 1 {
		t.Fatalf("expected retry_count 1, got %v", got)
	}
	if _, ok := recorder.updates[0].Details["next_retry_at"]; !ok {
		t.Fatalf("expected next_retry_at in retry details")
	}
}

func TestParseAlertTelegramTarget(t *testing.T) {
	t.Parallel()

	botToken, chatID, err := parseAlertTelegramTarget("bot-token:12345")
	if err != nil {
		t.Fatalf("parseAlertTelegramTarget returned error: %v", err)
	}
	if botToken != "bot-token" || chatID != "12345" {
		t.Fatalf("unexpected telegram target values %q %q", botToken, chatID)
	}
}
