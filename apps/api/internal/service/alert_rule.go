package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/packages/billing"
	"github.com/flowintel/flowintel/packages/domain"
)

var (
	ErrAlertRuleForbidden      = errors.New("alert feature is not available")
	ErrAlertRuleNotFound       = errors.New("alert rule not found")
	ErrAlertRuleLimitExceeded  = errors.New("alert rule limit exceeded")
	ErrAlertRuleConflict       = errors.New("alert rule conflict")
	ErrAlertRuleInvalidRequest = errors.New("invalid alert rule request")
)

type AlertRuleDefinition struct {
	WatchlistID                string   `json:"watchlistId"`
	SignalTypes                []string `json:"signalTypes"`
	MinimumSeverity            string   `json:"minimumSeverity"`
	RenotifyOnSeverityIncrease bool     `json:"renotifyOnSeverityIncrease"`
	SnoozeUntil                string   `json:"snoozeUntil,omitempty"`
}

type AlertRuleSummary struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	RuleType        string              `json:"ruleType"`
	IsEnabled       bool                `json:"isEnabled"`
	CooldownSeconds int                 `json:"cooldownSeconds"`
	EventCount      int                 `json:"eventCount"`
	LastTriggeredAt string              `json:"lastTriggeredAt,omitempty"`
	Definition      AlertRuleDefinition `json:"definition"`
	Tags            []string            `json:"tags"`
	CreatedAt       string              `json:"createdAt"`
	UpdatedAt       string              `json:"updatedAt"`
}

type AlertRuleDetail struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	RuleType        string              `json:"ruleType"`
	IsEnabled       bool                `json:"isEnabled"`
	CooldownSeconds int                 `json:"cooldownSeconds"`
	EventCount      int                 `json:"eventCount"`
	LastTriggeredAt string              `json:"lastTriggeredAt,omitempty"`
	Definition      AlertRuleDefinition `json:"definition"`
	Notes           string              `json:"notes,omitempty"`
	Tags            []string            `json:"tags"`
	CreatedAt       string              `json:"createdAt"`
	UpdatedAt       string              `json:"updatedAt"`
}

type AlertRuleCollection struct {
	Items []AlertRuleSummary `json:"items"`
}

type AlertEvent struct {
	ID          string         `json:"id"`
	AlertRuleID string         `json:"alertRuleId"`
	EventKey    string         `json:"eventKey"`
	DedupKey    string         `json:"dedupKey"`
	SignalType  string         `json:"signalType"`
	Severity    string         `json:"severity"`
	Payload     map[string]any `json:"payload"`
	ObservedAt  string         `json:"observedAt"`
	CreatedAt   string         `json:"createdAt"`
}

type AlertEventCollection struct {
	Items []AlertEvent `json:"items"`
}

type CreateAlertRuleRequest struct {
	Name            string
	RuleType        string
	IsEnabled       *bool
	CooldownSeconds int
	Definition      AlertRuleDefinition
	Notes           string
	Tags            []string
}

type UpdateAlertRuleRequest = CreateAlertRuleRequest

type AlertRuleMutationResult struct {
	Deleted bool `json:"deleted"`
}

type TriggerAlertEventRequest struct {
	EventKey   string
	SignalType string
	Severity   string
	Payload    map[string]any
	ObservedAt string
}

type TriggerAlertEventResult struct {
	Created    bool        `json:"created"`
	Suppressed bool        `json:"suppressed"`
	Reason     string      `json:"reason"`
	Event      *AlertEvent `json:"event,omitempty"`
}

type AlertRuleService struct {
	repo repository.AlertRuleRepository
	Now  func() time.Time
}

func NewAlertRuleService(repo repository.AlertRuleRepository) *AlertRuleService {
	return &AlertRuleService{repo: repo, Now: time.Now}
}

func (s *AlertRuleService) ListAlertRules(ctx context.Context, ownerUserID string, tier domain.PlanTier) (AlertRuleCollection, error) {
	if err := ensureAlertsEnabled(tier); err != nil {
		return AlertRuleCollection{}, err
	}

	items, err := s.repo.ListAlertRules(ctx, ownerUserID)
	if err != nil {
		return AlertRuleCollection{}, err
	}

	response := AlertRuleCollection{Items: make([]AlertRuleSummary, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, toAlertRuleSummary(item))
	}
	return response, nil
}

func (s *AlertRuleService) CreateAlertRule(ctx context.Context, ownerUserID string, tier domain.PlanTier, req CreateAlertRuleRequest) (AlertRuleDetail, error) {
	if err := ensureAlertsEnabled(tier); err != nil {
		return AlertRuleDetail{}, err
	}

	limits, err := alertRuleLimitForTier(tier)
	if err != nil {
		return AlertRuleDetail{}, err
	}
	items, err := s.repo.ListAlertRules(ctx, ownerUserID)
	if err != nil {
		return AlertRuleDetail{}, err
	}
	if len(items) >= limits {
		return AlertRuleDetail{}, ErrAlertRuleLimitExceeded
	}

	rule, err := s.buildAlertRule(ownerUserID, "", req, time.Time{})
	if err != nil {
		return AlertRuleDetail{}, err
	}
	created, err := s.repo.CreateAlertRule(ctx, rule)
	if err != nil {
		if errors.Is(err, repository.ErrAlertRuleAlreadyExists) {
			return AlertRuleDetail{}, ErrAlertRuleConflict
		}
		return AlertRuleDetail{}, err
	}
	return toAlertRuleDetail(created), nil
}

func (s *AlertRuleService) GetAlertRule(ctx context.Context, ownerUserID string, tier domain.PlanTier, ruleID string) (AlertRuleDetail, error) {
	if err := ensureAlertsEnabled(tier); err != nil {
		return AlertRuleDetail{}, err
	}
	rule, err := s.repo.FindAlertRule(ctx, ownerUserID, ruleID)
	if err != nil {
		if errors.Is(err, repository.ErrAlertRuleNotFound) {
			return AlertRuleDetail{}, ErrAlertRuleNotFound
		}
		return AlertRuleDetail{}, err
	}
	return toAlertRuleDetail(rule), nil
}

func (s *AlertRuleService) UpdateAlertRule(ctx context.Context, ownerUserID string, tier domain.PlanTier, ruleID string, req UpdateAlertRuleRequest) (AlertRuleDetail, error) {
	if err := ensureAlertsEnabled(tier); err != nil {
		return AlertRuleDetail{}, err
	}

	current, err := s.repo.FindAlertRule(ctx, ownerUserID, ruleID)
	if err != nil {
		if errors.Is(err, repository.ErrAlertRuleNotFound) {
			return AlertRuleDetail{}, ErrAlertRuleNotFound
		}
		return AlertRuleDetail{}, err
	}

	updated, err := s.buildAlertRule(ownerUserID, ruleID, req, current.CreatedAt)
	if err != nil {
		return AlertRuleDetail{}, err
	}
	updated.EventCount = current.EventCount
	updated.LastTriggeredAt = current.LastTriggeredAt

	result, err := s.repo.UpdateAlertRule(ctx, updated)
	if err != nil {
		if errors.Is(err, repository.ErrAlertRuleNotFound) {
			return AlertRuleDetail{}, ErrAlertRuleNotFound
		}
		return AlertRuleDetail{}, err
	}
	return toAlertRuleDetail(result), nil
}

func (s *AlertRuleService) DeleteAlertRule(ctx context.Context, ownerUserID string, tier domain.PlanTier, ruleID string) error {
	if err := ensureAlertsEnabled(tier); err != nil {
		return err
	}
	if err := s.repo.DeleteAlertRule(ctx, ownerUserID, ruleID); err != nil {
		if errors.Is(err, repository.ErrAlertRuleNotFound) {
			return ErrAlertRuleNotFound
		}
		return err
	}
	return nil
}

func (s *AlertRuleService) ListAlertEvents(ctx context.Context, ownerUserID string, tier domain.PlanTier, ruleID string) (AlertEventCollection, error) {
	if err := ensureAlertsEnabled(tier); err != nil {
		return AlertEventCollection{}, err
	}
	if _, err := s.repo.FindAlertRule(ctx, ownerUserID, ruleID); err != nil {
		if errors.Is(err, repository.ErrAlertRuleNotFound) {
			return AlertEventCollection{}, ErrAlertRuleNotFound
		}
		return AlertEventCollection{}, err
	}

	items, err := s.repo.ListAlertEvents(ctx, ownerUserID, ruleID)
	if err != nil {
		return AlertEventCollection{}, err
	}
	response := AlertEventCollection{Items: make([]AlertEvent, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, toAlertEvent(item))
	}
	return response, nil
}

func (s *AlertRuleService) EvaluateAlertEvent(
	ctx context.Context,
	ownerUserID string,
	tier domain.PlanTier,
	ruleID string,
	req TriggerAlertEventRequest,
) (TriggerAlertEventResult, error) {
	if err := ensureAlertsEnabled(tier); err != nil {
		return TriggerAlertEventResult{}, err
	}

	rule, err := s.repo.FindAlertRule(ctx, ownerUserID, ruleID)
	if err != nil {
		if errors.Is(err, repository.ErrAlertRuleNotFound) {
			return TriggerAlertEventResult{}, ErrAlertRuleNotFound
		}
		return TriggerAlertEventResult{}, err
	}
	if !rule.IsEnabled {
		return TriggerAlertEventResult{Suppressed: true, Reason: "disabled"}, nil
	}

	definition, err := domain.ParseAlertRuleDefinition(rule.Definition)
	if err != nil {
		return TriggerAlertEventResult{}, ErrAlertRuleInvalidRequest
	}

	normalizedSignalType, err := domain.NormalizeAlertRuleType(req.SignalType)
	if err != nil {
		return TriggerAlertEventResult{}, ErrAlertRuleInvalidRequest
	}
	normalizedSeverity, err := domain.NormalizeAlertSeverity(req.Severity)
	if err != nil {
		return TriggerAlertEventResult{}, ErrAlertRuleInvalidRequest
	}
	normalizedEventKey, err := domain.NormalizeAlertEventKey(req.EventKey)
	if err != nil {
		return TriggerAlertEventResult{}, ErrAlertRuleInvalidRequest
	}
	observedAt, err := parseAlertObservedAt(req.ObservedAt, s.now())
	if err != nil {
		return TriggerAlertEventResult{}, ErrAlertRuleInvalidRequest
	}

	if len(definition.SignalTypes) > 0 && !containsString(definition.SignalTypes, normalizedSignalType) {
		return TriggerAlertEventResult{Suppressed: true, Reason: "signal_type_not_matched"}, nil
	}
	if definition.SnoozeUntil != nil && observedAt.Before(definition.SnoozeUntil.UTC()) {
		return TriggerAlertEventResult{Suppressed: true, Reason: "snoozed"}, nil
	}
	if domain.CompareAlertSeverity(normalizedSeverity, definition.MinimumSeverity) < 0 {
		return TriggerAlertEventResult{Suppressed: true, Reason: "severity_below_threshold"}, nil
	}

	latest, err := s.repo.FindLatestAlertEvent(ctx, ownerUserID, ruleID, normalizedEventKey)
	if err != nil {
		return TriggerAlertEventResult{}, err
	}
	if latest != nil && withinAlertCooldown(rule, latest.ObservedAt, observedAt) {
		if !definition.RenotifyOnSeverityIncrease || domain.CompareAlertSeverity(normalizedSeverity, latest.Severity) <= 0 {
			return TriggerAlertEventResult{Suppressed: true, Reason: "cooldown_active"}, nil
		}
	}

	dedupKey, err := buildAlertDedupKey(rule.ID, normalizedEventKey, normalizedSeverity, observedAt, rule.CooldownSeconds)
	if err != nil {
		return TriggerAlertEventResult{}, err
	}

	event, err := s.repo.CreateAlertEvent(ctx, domain.AlertEvent{
		ID:          newAlertEventID(),
		AlertRuleID: rule.ID,
		OwnerUserID: strings.TrimSpace(ownerUserID),
		EventKey:    normalizedEventKey,
		DedupKey:    dedupKey,
		SignalType:  normalizedSignalType,
		Severity:    normalizedSeverity,
		Payload:     clonePayload(req.Payload),
		ObservedAt:  observedAt,
		CreatedAt:   s.now().UTC(),
	})
	if err != nil {
		if errors.Is(err, repository.ErrAlertRuleNotFound) {
			return TriggerAlertEventResult{}, ErrAlertRuleNotFound
		}
		return TriggerAlertEventResult{Suppressed: true, Reason: "duplicate"}, nil
	}

	result := toAlertEvent(event)
	return TriggerAlertEventResult{
		Created: true,
		Reason:  "created",
		Event:   &result,
	}, nil
}

func (s *AlertRuleService) buildAlertRule(ownerUserID string, ruleID string, req CreateAlertRuleRequest, createdAt time.Time) (domain.AlertRule, error) {
	name, err := domain.NormalizeAlertRuleName(req.Name)
	if err != nil {
		return domain.AlertRule{}, ErrAlertRuleInvalidRequest
	}
	ruleType, err := domain.NormalizeAlertRuleType(req.RuleType)
	if err != nil {
		return domain.AlertRule{}, ErrAlertRuleInvalidRequest
	}
	definition, err := normalizeAlertRuleDefinition(req.Definition)
	if err != nil {
		return domain.AlertRule{}, ErrAlertRuleInvalidRequest
	}

	now := s.now().UTC()
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}
	if createdAt.IsZero() {
		createdAt = now
	}

	return domain.AlertRule{
		ID:              chooseAlertRuleID(ruleID),
		OwnerUserID:     strings.TrimSpace(ownerUserID),
		Name:            name,
		RuleType:        ruleType,
		Definition:      domain.BuildAlertRuleDefinitionMap(definition),
		Notes:           domain.NormalizeAlertNotes(req.Notes),
		Tags:            domain.NormalizeAlertTags(req.Tags),
		IsEnabled:       isEnabled,
		CooldownSeconds: domain.NormalizeAlertCooldownSeconds(req.CooldownSeconds),
		CreatedAt:       createdAt.UTC(),
		UpdatedAt:       now,
	}, nil
}

func ensureAlertsEnabled(tier domain.PlanTier) error {
	plan, err := billing.FindPlan(tier)
	if err != nil {
		return ErrAlertRuleForbidden
	}
	if !billing.IsFeatureEnabled(plan, billing.FeatureAlerts) {
		return ErrAlertRuleForbidden
	}
	return nil
}

func alertRuleLimitForTier(tier domain.PlanTier) (int, error) {
	switch tier {
	case domain.PlanPro:
		return 10, nil
	case domain.PlanTeam:
		return 50, nil
	default:
		return 0, ErrAlertRuleForbidden
	}
}

func normalizeAlertRuleDefinition(input AlertRuleDefinition) (domain.AlertRuleDefinition, error) {
	watchlistID := strings.TrimSpace(input.WatchlistID)
	if watchlistID == "" {
		return domain.AlertRuleDefinition{}, errors.New("watchlist id is required")
	}
	signalTypes := make([]string, 0, len(input.SignalTypes))
	for _, item := range input.SignalTypes {
		normalized, err := domain.NormalizeAlertRuleType(item)
		if err != nil {
			return domain.AlertRuleDefinition{}, err
		}
		signalTypes = append(signalTypes, normalized)
	}
	if len(signalTypes) == 0 {
		return domain.AlertRuleDefinition{}, errors.New("at least one signal type is required")
	}
	minimumSeverity, err := domain.NormalizeAlertSeverity(input.MinimumSeverity)
	if err != nil {
		return domain.AlertRuleDefinition{}, err
	}
	var snoozeUntil *time.Time
	if strings.TrimSpace(input.SnoozeUntil) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(input.SnoozeUntil))
		if err != nil {
			return domain.AlertRuleDefinition{}, err
		}
		value := parsed.UTC()
		snoozeUntil = &value
	}
	return domain.AlertRuleDefinition{
		WatchlistID:                watchlistID,
		SignalTypes:                signalTypes,
		MinimumSeverity:            minimumSeverity,
		RenotifyOnSeverityIncrease: input.RenotifyOnSeverityIncrease,
		SnoozeUntil:                snoozeUntil,
	}, nil
}

func parseAlertObservedAt(raw string, fallback time.Time) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback.UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func buildAlertDedupKey(
	ruleID string,
	eventKey string,
	severity domain.AlertSeverity,
	observedAt time.Time,
	cooldownSeconds int,
) (string, error) {
	window := time.Second
	if cooldownSeconds > 0 {
		window = time.Duration(cooldownSeconds) * time.Second
	}
	bucket := observedAt.UTC().Truncate(window).Format(time.RFC3339)
	return domain.NormalizeAlertDedupKey(fmt.Sprintf("%s:%s:%s:%s", strings.TrimSpace(ruleID), eventKey, severity, bucket))
}

func withinAlertCooldown(rule domain.AlertRule, lastObservedAt time.Time, observedAt time.Time) bool {
	if rule.CooldownSeconds <= 0 {
		return false
	}
	return observedAt.Before(lastObservedAt.UTC().Add(time.Duration(rule.CooldownSeconds) * time.Second))
}

func toAlertRuleSummary(rule domain.AlertRule) AlertRuleSummary {
	definition, _ := domain.ParseAlertRuleDefinition(rule.Definition)
	return AlertRuleSummary{
		ID:              rule.ID,
		Name:            rule.Name,
		RuleType:        rule.RuleType,
		IsEnabled:       rule.IsEnabled,
		CooldownSeconds: rule.CooldownSeconds,
		EventCount:      rule.EventCount,
		LastTriggeredAt: formatOptionalTime(rule.LastTriggeredAt),
		Definition:      toAlertRuleDefinition(definition),
		Tags:            append([]string(nil), rule.Tags...),
		CreatedAt:       rule.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       rule.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toAlertRuleDetail(rule domain.AlertRule) AlertRuleDetail {
	definition, _ := domain.ParseAlertRuleDefinition(rule.Definition)
	return AlertRuleDetail{
		ID:              rule.ID,
		Name:            rule.Name,
		RuleType:        rule.RuleType,
		IsEnabled:       rule.IsEnabled,
		CooldownSeconds: rule.CooldownSeconds,
		EventCount:      rule.EventCount,
		LastTriggeredAt: formatOptionalTime(rule.LastTriggeredAt),
		Definition:      toAlertRuleDefinition(definition),
		Notes:           rule.Notes,
		Tags:            append([]string(nil), rule.Tags...),
		CreatedAt:       rule.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       rule.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toAlertRuleDefinition(definition domain.AlertRuleDefinition) AlertRuleDefinition {
	return AlertRuleDefinition{
		WatchlistID:                definition.WatchlistID,
		SignalTypes:                append([]string(nil), definition.SignalTypes...),
		MinimumSeverity:            string(definition.MinimumSeverity),
		RenotifyOnSeverityIncrease: definition.RenotifyOnSeverityIncrease,
		SnoozeUntil:                formatOptionalTime(definition.SnoozeUntil),
	}
}

func toAlertEvent(event domain.AlertEvent) AlertEvent {
	return AlertEvent{
		ID:          event.ID,
		AlertRuleID: event.AlertRuleID,
		EventKey:    event.EventKey,
		DedupKey:    event.DedupKey,
		SignalType:  event.SignalType,
		Severity:    string(event.Severity),
		Payload:     clonePayload(event.Payload),
		ObservedAt:  event.ObservedAt.UTC().Format(time.RFC3339),
		CreatedAt:   event.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func containsString(values []string, target string) bool {
	for _, item := range values {
		if item == target {
			return true
		}
	}
	return false
}

func clonePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func chooseAlertRuleID(ruleID string) string {
	trimmed := strings.TrimSpace(ruleID)
	if trimmed != "" {
		return trimmed
	}
	return newAlertRuleID()
}

func newAlertRuleID() string {
	return "alr_" + newAlertHex(8)
}

func newAlertEventID() string {
	return "ale_" + newAlertHex(8)
}

func newAlertHex(bytesLen int) string {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func (s *AlertRuleService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
