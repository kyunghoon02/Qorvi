package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

type AlertDeliveryChannelReader interface {
	ListEnabledAlertDeliveryChannels(context.Context, string) ([]domain.AlertDeliveryChannel, error)
}

type AlertDeliveryAttemptRecorder interface {
	CreateAlertDeliveryAttempt(context.Context, db.AlertDeliveryAttemptCreate) (domain.AlertDeliveryAttempt, error)
	UpdateAlertDeliveryAttempt(context.Context, db.AlertDeliveryAttemptUpdate) (domain.AlertDeliveryAttempt, error)
}

type AlertEmailRequest struct {
	To      string
	Subject string
	Body    string
}

type AlertDiscordWebhookRequest struct {
	WebhookURL string
	Content    string
}

type AlertTelegramRequest struct {
	BotToken string
	ChatID   string
	Text     string
}

type AlertEmailSender interface {
	SendAlertEmail(context.Context, AlertEmailRequest) (int, string, error)
}

type AlertDiscordWebhookSender interface {
	SendDiscordWebhook(context.Context, AlertDiscordWebhookRequest) (int, string, error)
}

type AlertTelegramSender interface {
	SendTelegramMessage(context.Context, AlertTelegramRequest) (int, string, error)
}

type AlertDeliveryReport struct {
	MatchedChannels int
	AttemptsCreated int
	Delivered       int
	Failed          int
	Deduped         int
}

type AlertDeliveryDispatcher interface {
	DeliverAlertEvent(context.Context, domain.AlertEvent) (AlertDeliveryReport, error)
}

type WalletSignalDeliveryDispatcher struct {
	Channels       AlertDeliveryChannelReader
	Attempts       AlertDeliveryAttemptRecorder
	Email          AlertEmailSender
	Discord        AlertDiscordWebhookSender
	Telegram       AlertTelegramSender
	RetryLimit     int
	RetryBaseDelay time.Duration
	Now            func() time.Time
}

type SMTPAlertEmailSender struct {
	Addr     string
	Host     string
	Username string
	Password string
	From     string
}

type HTTPAlertDiscordWebhookSender struct {
	Client *http.Client
}

type HTTPAlertTelegramSender struct {
	Client *http.Client
}

type AlertDeliveryRetryResult struct {
	Status    domain.AlertDeliveryStatus
	Delivered bool
	Requeued  bool
	Exhausted bool
}

func (d WalletSignalDeliveryDispatcher) DeliverAlertEvent(ctx context.Context, event domain.AlertEvent) (AlertDeliveryReport, error) {
	if d.Channels == nil || d.Attempts == nil {
		return AlertDeliveryReport{}, nil
	}

	channels, err := d.Channels.ListEnabledAlertDeliveryChannels(ctx, strings.TrimSpace(event.OwnerUserID))
	if err != nil {
		return AlertDeliveryReport{}, err
	}

	report := AlertDeliveryReport{MatchedChannels: len(channels)}
	for _, channel := range channels {
		attempt, err := d.Attempts.CreateAlertDeliveryAttempt(ctx, db.AlertDeliveryAttemptCreate{
			AlertEventID: event.ID,
			ChannelID:    channel.ID,
			OwnerUserID:  channel.OwnerUserID,
			DeliveryKey:  buildAlertDeliveryKey(event.ID, channel.ID),
			ChannelType:  channel.ChannelType,
			Target:       channel.Target,
			Status:       domain.AlertDeliveryStatusQueued,
			Details: map[string]any{
				"signal_type":     event.SignalType,
				"severity":        string(event.Severity),
				"retry_count":     0,
				"retry_exhausted": false,
			},
			AttemptedAt: ptrTimeUTC(d.now()),
		})
		if err != nil {
			if err == db.ErrAlertDeliveryAttemptDeduped {
				report.Deduped++
				continue
			}
			return report, err
		}
		report.AttemptsCreated++

		result := d.deliverThroughChannel(ctx, channel, event)
		update := d.buildAlertDeliveryAttemptUpdate(attempt, result)
		_, err = d.Attempts.UpdateAlertDeliveryAttempt(ctx, db.AlertDeliveryAttemptUpdate{
			AttemptID:    attempt.ID,
			Status:       update.Status,
			ResponseCode: update.ResponseCode,
			Details:      update.Details,
			AttemptedAt:  update.AttemptedAt,
			DeliveredAt:  update.DeliveredAt,
			FailedAt:     update.FailedAt,
		})
		if err != nil {
			return report, err
		}

		if result.Status == domain.AlertDeliveryStatusDelivered {
			report.Delivered++
		} else if result.Status == domain.AlertDeliveryStatusFailed {
			report.Failed++
		}
	}

	return report, nil
}

type alertDeliveryResult struct {
	Status       domain.AlertDeliveryStatus
	ResponseCode int
	Details      map[string]any
	DeliveredAt  time.Time
	FailedAt     time.Time
}

func (d WalletSignalDeliveryDispatcher) deliverThroughChannel(
	ctx context.Context,
	channel domain.AlertDeliveryChannel,
	event domain.AlertEvent,
) alertDeliveryResult {
	subject, body := buildAlertEmailContent(event)
	discordContent := buildAlertDiscordContent(event)

	switch channel.ChannelType {
	case domain.AlertChannelTypeEmail:
		if d.Email == nil {
			return failedAlertDeliveryResult("email sender unavailable", 0, d.now())
		}
		code, response, err := d.Email.SendAlertEmail(ctx, AlertEmailRequest{
			To:      channel.Target,
			Subject: subject,
			Body:    body,
		})
		if err != nil {
			return failedAlertDeliveryResult(err.Error(), code, d.now())
		}
		return deliveredAlertDeliveryResult(code, response, d.now())
	case domain.AlertChannelTypeDiscordWebhook:
		if d.Discord == nil {
			return failedAlertDeliveryResult("discord sender unavailable", 0, d.now())
		}
		code, response, err := d.Discord.SendDiscordWebhook(ctx, AlertDiscordWebhookRequest{
			WebhookURL: channel.Target,
			Content:    discordContent,
		})
		if err != nil {
			return failedAlertDeliveryResult(err.Error(), code, d.now())
		}
		return deliveredAlertDeliveryResult(code, response, d.now())
	case domain.AlertChannelTypeTelegram:
		if d.Telegram == nil {
			return failedAlertDeliveryResult("telegram sender unavailable", 0, d.now())
		}
		botToken, chatID, err := parseAlertTelegramTarget(channel.Target)
		if err != nil {
			return failedAlertDeliveryResult(err.Error(), 0, d.now())
		}
		code, response, err := d.Telegram.SendTelegramMessage(ctx, AlertTelegramRequest{
			BotToken: botToken,
			ChatID:   chatID,
			Text:     buildAlertTelegramContent(event),
		})
		if err != nil {
			return failedAlertDeliveryResult(err.Error(), code, d.now())
		}
		return deliveredAlertDeliveryResult(code, response, d.now())
	default:
		return failedAlertDeliveryResult("unsupported alert delivery channel", 0, d.now())
	}
}

func (d WalletSignalDeliveryDispatcher) RetryAlertDeliveryAttempt(
	ctx context.Context,
	candidate db.AlertDeliveryRetryCandidate,
) (AlertDeliveryRetryResult, error) {
	if d.Attempts == nil {
		return AlertDeliveryRetryResult{}, fmt.Errorf("alert delivery attempt store is required")
	}

	channel := domain.AlertDeliveryChannel{
		ID:          candidate.Attempt.ChannelID,
		OwnerUserID: candidate.Attempt.OwnerUserID,
		ChannelType: candidate.Attempt.ChannelType,
		Target:      candidate.Attempt.Target,
		Metadata:    map[string]any{},
		IsEnabled:   true,
	}
	result := d.deliverThroughChannel(ctx, channel, candidate.Event)
	update := d.buildAlertDeliveryAttemptUpdate(candidate.Attempt, result)
	updated, err := d.Attempts.UpdateAlertDeliveryAttempt(ctx, db.AlertDeliveryAttemptUpdate{
		AttemptID:    candidate.Attempt.ID,
		Status:       update.Status,
		ResponseCode: update.ResponseCode,
		Details:      update.Details,
		AttemptedAt:  update.AttemptedAt,
		DeliveredAt:  update.DeliveredAt,
		FailedAt:     update.FailedAt,
	})
	if err != nil {
		return AlertDeliveryRetryResult{}, err
	}

	return AlertDeliveryRetryResult{
		Status:    updated.Status,
		Delivered: updated.Status == domain.AlertDeliveryStatusDelivered,
		Requeued:  updated.Status == domain.AlertDeliveryStatusFailed && !deliveryRetryExhausted(updated.Details),
		Exhausted: updated.Status == domain.AlertDeliveryStatusFailed && deliveryRetryExhausted(updated.Details),
	}, nil
}

func (s SMTPAlertEmailSender) SendAlertEmail(_ context.Context, request AlertEmailRequest) (int, string, error) {
	message := strings.Join([]string{
		fmt.Sprintf("From: %s", s.From),
		fmt.Sprintf("To: %s", request.To),
		fmt.Sprintf("Subject: %s", request.Subject),
		"",
		request.Body,
	}, "\r\n")

	var auth smtp.Auth
	if strings.TrimSpace(s.Username) != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}

	if err := smtp.SendMail(s.Addr, auth, s.From, []string{request.To}, []byte(message)); err != nil {
		return 0, "", err
	}
	return 250, "smtp accepted", nil
}

func (s HTTPAlertDiscordWebhookSender) SendDiscordWebhook(ctx context.Context, request AlertDiscordWebhookRequest) (int, string, error) {
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}

	payload, err := json.Marshal(map[string]any{"content": request.Content})
	if err != nil {
		return 0, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, request.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body := ""
	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(resp.Body); err == nil {
		body = buffer.String()
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, body, fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}
	return resp.StatusCode, body, nil
}

func (s HTTPAlertTelegramSender) SendTelegramMessage(ctx context.Context, request AlertTelegramRequest) (int, string, error) {
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}

	payload, err := json.Marshal(map[string]any{
		"chat_id": request.ChatID,
		"text":    request.Text,
	})
	if err != nil {
		return 0, "", err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", request.BotToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body := ""
	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(resp.Body); err == nil {
		body = buffer.String()
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, body, fmt.Errorf("telegram api returned status %d", resp.StatusCode)
	}
	return resp.StatusCode, body, nil
}

func deliveredAlertDeliveryResult(responseCode int, response string, at time.Time) alertDeliveryResult {
	return alertDeliveryResult{
		Status:       domain.AlertDeliveryStatusDelivered,
		ResponseCode: responseCode,
		Details: map[string]any{
			"result":        "delivered",
			"response_body": response,
		},
		DeliveredAt: at.UTC(),
	}
}

func failedAlertDeliveryResult(reason string, responseCode int, at time.Time) alertDeliveryResult {
	return alertDeliveryResult{
		Status:       domain.AlertDeliveryStatusFailed,
		ResponseCode: responseCode,
		Details: map[string]any{
			"result": "failed",
			"error":  strings.TrimSpace(reason),
		},
		FailedAt: at.UTC(),
	}
}

func buildAlertDeliveryKey(eventID string, channelID string) string {
	return strings.TrimSpace(eventID) + ":" + strings.TrimSpace(channelID)
}

func buildAlertEmailContent(event domain.AlertEvent) (string, string) {
	subject := fmt.Sprintf("[Qorvi] %s %s", strings.ToUpper(string(event.Severity)), event.SignalType)
	body := fmt.Sprintf(
		"Signal: %s\nSeverity: %s\nObservedAt: %s\nAlertRuleID: %s\nPayload: %s\n",
		event.SignalType,
		event.Severity,
		event.ObservedAt.UTC().Format(time.RFC3339),
		event.AlertRuleID,
		mustMarshalAlertPayload(event.Payload),
	)
	return subject, body
}

func buildAlertDiscordContent(event domain.AlertEvent) string {
	return fmt.Sprintf(
		"Qorvi alert [%s] %s at %s\n%s",
		strings.ToUpper(string(event.Severity)),
		event.SignalType,
		event.ObservedAt.UTC().Format(time.RFC3339),
		mustMarshalAlertPayload(event.Payload),
	)
}

func buildAlertTelegramContent(event domain.AlertEvent) string {
	return buildAlertDiscordContent(event)
}

func mustMarshalAlertPayload(payload map[string]any) string {
	encoded, err := json.Marshal(domain.NormalizeAlertDefinition(payload))
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func ptrTimeUTC(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func parseAlertTelegramTarget(target string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(target), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("telegram target must be botToken:chatID")
	}
	botToken := strings.TrimSpace(parts[0])
	chatID := strings.TrimSpace(parts[1])
	if botToken == "" || chatID == "" {
		return "", "", fmt.Errorf("telegram target must be botToken:chatID")
	}
	return botToken, chatID, nil
}

func (d WalletSignalDeliveryDispatcher) buildAlertDeliveryAttemptUpdate(
	attempt domain.AlertDeliveryAttempt,
	result alertDeliveryResult,
) db.AlertDeliveryAttemptUpdate {
	now := d.now()
	details := cloneAlertPayload(attempt.Details)
	for key, value := range result.Details {
		details[key] = value
	}
	details["last_attempted_at"] = now.UTC().Format(time.RFC3339)
	details["response_code"] = result.ResponseCode

	retryCount := deliveryRetryCount(details)
	if result.Status == domain.AlertDeliveryStatusFailed {
		retryCount++
		details["retry_count"] = retryCount
		if retryCount >= d.retryLimit() {
			details["retry_exhausted"] = true
			delete(details, "next_retry_at")
		} else {
			details["retry_exhausted"] = false
			details["next_retry_at"] = now.UTC().Add(deliveryRetryBackoff(d.retryBaseDelay(), retryCount)).Format(time.RFC3339)
		}
	} else {
		details["retry_count"] = retryCount
		details["retry_exhausted"] = false
		delete(details, "next_retry_at")
	}

	return db.AlertDeliveryAttemptUpdate{
		AttemptID:    attempt.ID,
		Status:       result.Status,
		ResponseCode: result.ResponseCode,
		Details:      details,
		AttemptedAt:  ptrTimeUTC(now),
		DeliveredAt:  ptrTimeUTC(result.DeliveredAt),
		FailedAt:     ptrTimeUTC(result.FailedAt),
	}
}

func deliveryRetryCount(details map[string]any) int {
	switch value := details["retry_count"].(type) {
	case int:
		if value < 0 {
			return 0
		}
		return value
	case float64:
		if value < 0 {
			return 0
		}
		return int(value)
	default:
		return 0
	}
}

func deliveryRetryExhausted(details map[string]any) bool {
	value, ok := details["retry_exhausted"].(bool)
	return ok && value
}

func deliveryRetryBackoff(base time.Duration, retryCount int) time.Duration {
	if retryCount <= 1 {
		return base
	}
	backoff := base
	for step := 1; step < retryCount; step++ {
		backoff *= 2
		if backoff >= time.Hour {
			return time.Hour
		}
	}
	return backoff
}

func (d WalletSignalDeliveryDispatcher) retryLimit() int {
	if d.RetryLimit > 0 {
		return d.RetryLimit
	}
	return 3
}

func (d WalletSignalDeliveryDispatcher) retryBaseDelay() time.Duration {
	if d.RetryBaseDelay > 0 {
		return d.RetryBaseDelay
	}
	return time.Minute
}

func (d WalletSignalDeliveryDispatcher) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}
