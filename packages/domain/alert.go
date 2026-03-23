package domain

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

type AlertRuleDefinition struct {
	WatchlistID                string        `json:"watchlistId"`
	SignalTypes                []string      `json:"signalTypes"`
	MinimumSeverity            AlertSeverity `json:"minimumSeverity"`
	RenotifyOnSeverityIncrease bool          `json:"renotifyOnSeverityIncrease"`
	SnoozeUntil                *time.Time    `json:"snoozeUntil,omitempty"`
}

type AlertRule struct {
	ID              string         `json:"id"`
	OwnerUserID     string         `json:"ownerUserId"`
	Name            string         `json:"name"`
	RuleType        string         `json:"ruleType"`
	Definition      map[string]any `json:"definition"`
	Notes           string         `json:"notes"`
	Tags            []string       `json:"tags"`
	IsEnabled       bool           `json:"isEnabled"`
	CooldownSeconds int            `json:"cooldownSeconds"`
	LastTriggeredAt *time.Time     `json:"lastTriggeredAt,omitempty"`
	EventCount      int            `json:"eventCount"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type AlertEvent struct {
	ID          string         `json:"id"`
	AlertRuleID string         `json:"alertRuleId"`
	OwnerUserID string         `json:"ownerUserId"`
	EventKey    string         `json:"eventKey"`
	DedupKey    string         `json:"dedupKey"`
	SignalType  string         `json:"signalType"`
	Severity    AlertSeverity  `json:"severity"`
	Payload     map[string]any `json:"payload"`
	ObservedAt  time.Time      `json:"observedAt"`
	ReadAt      *time.Time     `json:"readAt,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type AlertChannelType string

const (
	AlertChannelTypeEmail          AlertChannelType = "email"
	AlertChannelTypeDiscordWebhook AlertChannelType = "discord_webhook"
	AlertChannelTypeTelegram       AlertChannelType = "telegram"
)

type AlertDeliveryStatus string

const (
	AlertDeliveryStatusQueued    AlertDeliveryStatus = "queued"
	AlertDeliveryStatusDelivered AlertDeliveryStatus = "delivered"
	AlertDeliveryStatusFailed    AlertDeliveryStatus = "failed"
)

type AlertDeliveryChannel struct {
	ID          string           `json:"id"`
	OwnerUserID string           `json:"ownerUserId"`
	Label       string           `json:"label"`
	ChannelType AlertChannelType `json:"channelType"`
	Target      string           `json:"target"`
	Metadata    map[string]any   `json:"metadata"`
	IsEnabled   bool             `json:"isEnabled"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

type AlertDeliveryAttempt struct {
	ID           string              `json:"id"`
	AlertEventID string              `json:"alertEventId"`
	ChannelID    string              `json:"channelId"`
	OwnerUserID  string              `json:"ownerUserId"`
	DeliveryKey  string              `json:"deliveryKey"`
	ChannelType  AlertChannelType    `json:"channelType"`
	Target       string              `json:"target"`
	Status       AlertDeliveryStatus `json:"status"`
	ResponseCode int                 `json:"responseCode"`
	Details      map[string]any      `json:"details"`
	AttemptedAt  *time.Time          `json:"attemptedAt,omitempty"`
	DeliveredAt  *time.Time          `json:"deliveredAt,omitempty"`
	FailedAt     *time.Time          `json:"failedAt,omitempty"`
	CreatedAt    time.Time           `json:"createdAt"`
}

func NormalizeAlertRuleName(name string) (string, error) {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return "", fmt.Errorf("alert rule name is required")
	}
	if len([]rune(normalized)) > 120 {
		return "", fmt.Errorf("alert rule name must be 120 characters or fewer")
	}

	return normalized, nil
}

func NormalizeAlertRuleType(ruleType string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(ruleType))
	if normalized == "" {
		return "", fmt.Errorf("alert rule type is required")
	}
	if len([]rune(normalized)) > 80 {
		return "", fmt.Errorf("alert rule type must be 80 characters or fewer")
	}

	return normalized, nil
}

func NormalizeAlertSeverity(severity string) (AlertSeverity, error) {
	switch AlertSeverity(strings.ToLower(strings.TrimSpace(severity))) {
	case AlertSeverityLow:
		return AlertSeverityLow, nil
	case AlertSeverityMedium:
		return AlertSeverityMedium, nil
	case AlertSeverityHigh:
		return AlertSeverityHigh, nil
	case AlertSeverityCritical:
		return AlertSeverityCritical, nil
	default:
		return "", fmt.Errorf("alert severity must be one of low, medium, high, critical")
	}
}

func NormalizeAlertNotes(notes string) string {
	normalized := strings.TrimSpace(notes)
	if len([]rune(normalized)) > 500 {
		return string([]rune(normalized)[:500])
	}
	return normalized
}

func NormalizeAlertTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := strings.ToLower(strings.TrimSpace(tag))
		if value == "" {
			continue
		}
		if len([]rune(value)) > 32 {
			value = string([]rune(value)[:32])
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	slices.Sort(normalized)
	return normalized
}

func NormalizeAlertEventKey(eventKey string) (string, error) {
	normalized := strings.TrimSpace(eventKey)
	if normalized == "" {
		return "", fmt.Errorf("alert event key is required")
	}
	if len([]rune(normalized)) > 120 {
		return "", fmt.Errorf("alert event key must be 120 characters or fewer")
	}

	return normalized, nil
}

func NormalizeAlertDedupKey(dedupKey string) (string, error) {
	normalized := strings.TrimSpace(dedupKey)
	if normalized == "" {
		return "", fmt.Errorf("alert dedup key is required")
	}
	if len([]rune(normalized)) > 255 {
		return "", fmt.Errorf("alert dedup key must be 255 characters or fewer")
	}
	return normalized, nil
}

func NormalizeAlertCooldownSeconds(seconds int) int {
	if seconds < 0 {
		return 0
	}
	return seconds
}

func NormalizeAlertChannelLabel(label string) (string, error) {
	normalized := strings.TrimSpace(label)
	if normalized == "" {
		return "", fmt.Errorf("alert delivery channel label is required")
	}
	if len([]rune(normalized)) > 80 {
		return "", fmt.Errorf("alert delivery channel label must be 80 characters or fewer")
	}
	return normalized, nil
}

func NormalizeAlertChannelType(channelType string) (AlertChannelType, error) {
	switch AlertChannelType(strings.ToLower(strings.TrimSpace(channelType))) {
	case AlertChannelTypeEmail:
		return AlertChannelTypeEmail, nil
	case AlertChannelTypeDiscordWebhook:
		return AlertChannelTypeDiscordWebhook, nil
	case AlertChannelTypeTelegram:
		return AlertChannelTypeTelegram, nil
	default:
		return "", fmt.Errorf("alert delivery channel type must be one of email, discord_webhook, telegram")
	}
}

func NormalizeAlertDeliveryStatus(status string) (AlertDeliveryStatus, error) {
	switch AlertDeliveryStatus(strings.ToLower(strings.TrimSpace(status))) {
	case AlertDeliveryStatusQueued:
		return AlertDeliveryStatusQueued, nil
	case AlertDeliveryStatusDelivered:
		return AlertDeliveryStatusDelivered, nil
	case AlertDeliveryStatusFailed:
		return AlertDeliveryStatusFailed, nil
	default:
		return "", fmt.Errorf("alert delivery status must be one of queued, delivered, failed")
	}
}

func NormalizeAlertChannelTarget(channelType AlertChannelType, target string) (string, error) {
	normalized := strings.TrimSpace(target)
	if normalized == "" {
		return "", fmt.Errorf("alert delivery channel target is required")
	}

	switch channelType {
	case AlertChannelTypeEmail:
		if !strings.Contains(normalized, "@") {
			return "", fmt.Errorf("alert delivery email target must contain @")
		}
	case AlertChannelTypeDiscordWebhook:
		if !strings.HasPrefix(normalized, "https://") {
			return "", fmt.Errorf("discord webhook target must be an https url")
		}
	case AlertChannelTypeTelegram:
		if !strings.Contains(normalized, ":") {
			return "", fmt.Errorf("telegram target must be botToken:chatID")
		}
	default:
		return "", fmt.Errorf("unsupported alert delivery channel type")
	}

	if len([]rune(normalized)) > 255 {
		return "", fmt.Errorf("alert delivery channel target must be 255 characters or fewer")
	}
	return normalized, nil
}

func ValidateAlertRule(rule AlertRule) error {
	if strings.TrimSpace(rule.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if _, err := NormalizeAlertRuleName(rule.Name); err != nil {
		return err
	}
	if _, err := NormalizeAlertRuleType(rule.RuleType); err != nil {
		return err
	}
	if rule.Definition == nil {
		return fmt.Errorf("alert rule definition is required")
	}
	if rule.CooldownSeconds < 0 {
		return fmt.Errorf("alert rule cooldown seconds must be non-negative")
	}
	if _, err := ParseAlertRuleDefinition(rule.Definition); err != nil {
		return err
	}

	return nil
}

func ValidateAlertEvent(event AlertEvent) error {
	if strings.TrimSpace(event.AlertRuleID) == "" {
		return fmt.Errorf("alert rule id is required")
	}
	if strings.TrimSpace(event.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if _, err := NormalizeAlertEventKey(event.EventKey); err != nil {
		return err
	}
	if _, err := NormalizeAlertDedupKey(event.DedupKey); err != nil {
		return err
	}
	if _, err := NormalizeAlertRuleType(event.SignalType); err != nil {
		return err
	}
	if _, err := NormalizeAlertSeverity(string(event.Severity)); err != nil {
		return err
	}
	if event.Payload == nil {
		return fmt.Errorf("alert event payload is required")
	}

	return nil
}

func ValidateAlertDeliveryChannel(channel AlertDeliveryChannel) error {
	if strings.TrimSpace(channel.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if _, err := NormalizeAlertChannelLabel(channel.Label); err != nil {
		return err
	}
	channelType, err := NormalizeAlertChannelType(string(channel.ChannelType))
	if err != nil {
		return err
	}
	if _, err := NormalizeAlertChannelTarget(channelType, channel.Target); err != nil {
		return err
	}
	if channel.Metadata == nil {
		return fmt.Errorf("alert delivery channel metadata is required")
	}
	return nil
}

func ValidateAlertDeliveryAttempt(attempt AlertDeliveryAttempt) error {
	if strings.TrimSpace(attempt.AlertEventID) == "" {
		return fmt.Errorf("alert event id is required")
	}
	if strings.TrimSpace(attempt.ChannelID) == "" {
		return fmt.Errorf("channel id is required")
	}
	if strings.TrimSpace(attempt.OwnerUserID) == "" {
		return fmt.Errorf("owner user id is required")
	}
	if _, err := NormalizeAlertDedupKey(attempt.DeliveryKey); err != nil {
		return err
	}
	channelType, err := NormalizeAlertChannelType(string(attempt.ChannelType))
	if err != nil {
		return err
	}
	if _, err := NormalizeAlertChannelTarget(channelType, attempt.Target); err != nil {
		return err
	}
	if _, err := NormalizeAlertDeliveryStatus(string(attempt.Status)); err != nil {
		return err
	}
	if attempt.Details == nil {
		return fmt.Errorf("alert delivery attempt details is required")
	}
	return nil
}

func BuildAlertEventDedupKey(ruleID string, eventKey string) (string, error) {
	ruleID = strings.TrimSpace(ruleID)
	key, err := NormalizeAlertEventKey(eventKey)
	if err != nil {
		return "", err
	}
	if ruleID == "" {
		return "", fmt.Errorf("alert rule id is required")
	}

	return fmt.Sprintf("%s:%s", ruleID, key), nil
}

func CanTriggerAlertRule(rule AlertRule, observedAt time.Time) bool {
	if rule.CooldownSeconds <= 0 || rule.LastTriggeredAt == nil {
		return true
	}

	return !observedAt.Before(rule.LastTriggeredAt.Add(time.Duration(rule.CooldownSeconds) * time.Second))
}

func ParseAlertRuleDefinition(definition map[string]any) (AlertRuleDefinition, error) {
	normalized := NormalizeAlertDefinition(definition)
	result := AlertRuleDefinition{
		WatchlistID: strings.TrimSpace(stringValue(normalized["watchlistId"])),
		SignalTypes: normalizeAlertSignalTypes(normalized["signalTypes"]),
	}
	if severityValue := strings.TrimSpace(stringValue(normalized["minimumSeverity"])); severityValue != "" {
		severity, err := NormalizeAlertSeverity(severityValue)
		if err != nil {
			return AlertRuleDefinition{}, err
		}
		result.MinimumSeverity = severity
	} else {
		result.MinimumSeverity = AlertSeverityMedium
	}
	result.RenotifyOnSeverityIncrease = boolValue(normalized["renotifyOnSeverityIncrease"])
	if rawSnoozeUntil := strings.TrimSpace(stringValue(normalized["snoozeUntil"])); rawSnoozeUntil != "" {
		parsed, err := time.Parse(time.RFC3339, rawSnoozeUntil)
		if err != nil {
			return AlertRuleDefinition{}, fmt.Errorf("invalid snoozeUntil")
		}
		value := parsed.UTC()
		result.SnoozeUntil = &value
	}
	return result, nil
}

func BuildAlertRuleDefinitionMap(definition AlertRuleDefinition) map[string]any {
	result := map[string]any{
		"watchlistId":                strings.TrimSpace(definition.WatchlistID),
		"signalTypes":                append([]string(nil), normalizeAlertSignalTypes(definition.SignalTypes)...),
		"minimumSeverity":            string(normalizeDefinitionSeverity(definition.MinimumSeverity)),
		"renotifyOnSeverityIncrease": definition.RenotifyOnSeverityIncrease,
	}
	if definition.SnoozeUntil != nil {
		result["snoozeUntil"] = definition.SnoozeUntil.UTC().Format(time.RFC3339)
	}
	return result
}

func CompareAlertSeverity(left AlertSeverity, right AlertSeverity) int {
	return alertSeverityRank(left) - alertSeverityRank(right)
}

func NormalizeAlertDefinition(definition map[string]any) map[string]any {
	if definition == nil {
		return map[string]any{}
	}

	clone := make(map[string]any, len(definition))
	for key, value := range definition {
		clone[key] = value
	}
	return clone
}

func MarshalAlertDefinition(definition map[string]any) ([]byte, error) {
	return json.Marshal(NormalizeAlertDefinition(definition))
}

func CopyAlertRule(rule AlertRule) AlertRule {
	cloned := rule
	cloned.Definition = NormalizeAlertDefinition(rule.Definition)
	cloned.Tags = append([]string(nil), rule.Tags...)
	if rule.LastTriggeredAt != nil {
		lastTriggeredAt := rule.LastTriggeredAt.UTC()
		cloned.LastTriggeredAt = &lastTriggeredAt
	}

	return cloned
}

func CopyAlertEvent(event AlertEvent) AlertEvent {
	cloned := event
	cloned.Payload = NormalizeAlertDefinition(event.Payload)
	if event.ReadAt != nil {
		readAt := event.ReadAt.UTC()
		cloned.ReadAt = &readAt
	}
	return cloned
}

func CopyAlertDeliveryChannel(channel AlertDeliveryChannel) AlertDeliveryChannel {
	cloned := channel
	cloned.Metadata = NormalizeAlertDefinition(channel.Metadata)
	return cloned
}

func CopyAlertDeliveryAttempt(attempt AlertDeliveryAttempt) AlertDeliveryAttempt {
	cloned := attempt
	cloned.Details = NormalizeAlertDefinition(attempt.Details)
	if attempt.AttemptedAt != nil {
		at := attempt.AttemptedAt.UTC()
		cloned.AttemptedAt = &at
	}
	if attempt.DeliveredAt != nil {
		at := attempt.DeliveredAt.UTC()
		cloned.DeliveredAt = &at
	}
	if attempt.FailedAt != nil {
		at := attempt.FailedAt.UTC()
		cloned.FailedAt = &at
	}
	return cloned
}

func normalizeAlertSignalTypes(raw any) []string {
	switch typed := raw.(type) {
	case []string:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if normalized, err := NormalizeAlertRuleType(item); err == nil {
				result = append(result, normalized)
			}
		}
		slices.Sort(result)
		return slices.Compact(result)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if normalized, err := NormalizeAlertRuleType(stringValue(item)); err == nil {
				result = append(result, normalized)
			}
		}
		slices.Sort(result)
		return slices.Compact(result)
	default:
		return []string{}
	}
}

func normalizeDefinitionSeverity(severity AlertSeverity) AlertSeverity {
	if normalized, err := NormalizeAlertSeverity(string(severity)); err == nil {
		return normalized
	}
	return AlertSeverityMedium
}

func alertSeverityRank(severity AlertSeverity) int {
	switch normalizeDefinitionSeverity(severity) {
	case AlertSeverityLow:
		return 1
	case AlertSeverityMedium:
		return 2
	case AlertSeverityHigh:
		return 3
	case AlertSeverityCritical:
		return 4
	default:
		return 0
	}
}

func stringValue(raw any) string {
	if raw == nil {
		return ""
	}
	switch typed := raw.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func boolValue(raw any) bool {
	if raw == nil {
		return false
	}
	switch typed := raw.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "on":
			return true
		}
	}
	return false
}
