package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

var (
	ErrAlertRuleNotFound      = errors.New("alert rule not found")
	ErrAlertRuleAlreadyExists = errors.New("alert rule already exists")
)

type AlertRuleRepository interface {
	ListAlertRules(context.Context, string) ([]domain.AlertRule, error)
	CreateAlertRule(context.Context, domain.AlertRule) (domain.AlertRule, error)
	FindAlertRule(context.Context, string, string) (domain.AlertRule, error)
	UpdateAlertRule(context.Context, domain.AlertRule) (domain.AlertRule, error)
	DeleteAlertRule(context.Context, string, string) error
	ListAlertEvents(context.Context, string, string) ([]domain.AlertEvent, error)
	FindLatestAlertEvent(context.Context, string, string, string) (*domain.AlertEvent, error)
	CreateAlertEvent(context.Context, domain.AlertEvent) (domain.AlertEvent, error)
}

type InMemoryAlertRuleRepository struct {
	mu     sync.RWMutex
	rules  map[string]map[string]domain.AlertRule
	events map[string]map[string][]domain.AlertEvent
}

func NewInMemoryAlertRuleRepository() *InMemoryAlertRuleRepository {
	return &InMemoryAlertRuleRepository{
		rules:  make(map[string]map[string]domain.AlertRule),
		events: make(map[string]map[string][]domain.AlertEvent),
	}
}

func (r *InMemoryAlertRuleRepository) ListAlertRules(_ context.Context, ownerUserID string) ([]domain.AlertRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	owned := r.rules[strings.TrimSpace(ownerUserID)]
	if len(owned) == 0 {
		return []domain.AlertRule{}, nil
	}

	items := make([]domain.AlertRule, 0, len(owned))
	for _, item := range owned {
		items = append(items, domain.CopyAlertRule(item))
	}

	sort.Slice(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].ID < items[j].ID
	})

	return items, nil
}

func (r *InMemoryAlertRuleRepository) CreateAlertRule(_ context.Context, rule domain.AlertRule) (domain.AlertRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerUserID := strings.TrimSpace(rule.OwnerUserID)
	ruleID := strings.TrimSpace(rule.ID)
	if ownerUserID == "" || ruleID == "" {
		return domain.AlertRule{}, fmt.Errorf("owner user id and alert rule id are required")
	}
	if _, ok := r.rules[ownerUserID]; !ok {
		r.rules[ownerUserID] = make(map[string]domain.AlertRule)
	}
	if _, exists := r.rules[ownerUserID][ruleID]; exists {
		return domain.AlertRule{}, ErrAlertRuleAlreadyExists
	}

	stored := domain.CopyAlertRule(rule)
	r.rules[ownerUserID][ruleID] = stored
	return domain.CopyAlertRule(stored), nil
}

func (r *InMemoryAlertRuleRepository) FindAlertRule(_ context.Context, ownerUserID string, ruleID string) (domain.AlertRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	owned := r.rules[strings.TrimSpace(ownerUserID)]
	if len(owned) == 0 {
		return domain.AlertRule{}, ErrAlertRuleNotFound
	}
	rule, ok := owned[strings.TrimSpace(ruleID)]
	if !ok {
		return domain.AlertRule{}, ErrAlertRuleNotFound
	}
	return domain.CopyAlertRule(rule), nil
}

func (r *InMemoryAlertRuleRepository) UpdateAlertRule(_ context.Context, rule domain.AlertRule) (domain.AlertRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerUserID := strings.TrimSpace(rule.OwnerUserID)
	if ownerUserID == "" {
		return domain.AlertRule{}, fmt.Errorf("owner user id is required")
	}
	if _, ok := r.rules[ownerUserID][rule.ID]; !ok {
		return domain.AlertRule{}, ErrAlertRuleNotFound
	}

	stored := domain.CopyAlertRule(rule)
	r.rules[ownerUserID][rule.ID] = stored
	return domain.CopyAlertRule(stored), nil
}

func (r *InMemoryAlertRuleRepository) DeleteAlertRule(_ context.Context, ownerUserID string, ruleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerUserID = strings.TrimSpace(ownerUserID)
	ruleID = strings.TrimSpace(ruleID)
	if _, ok := r.rules[ownerUserID][ruleID]; !ok {
		return ErrAlertRuleNotFound
	}
	delete(r.rules[ownerUserID], ruleID)
	if len(r.rules[ownerUserID]) == 0 {
		delete(r.rules, ownerUserID)
	}
	if _, ok := r.events[ownerUserID]; ok {
		delete(r.events[ownerUserID], ruleID)
		if len(r.events[ownerUserID]) == 0 {
			delete(r.events, ownerUserID)
		}
	}
	return nil
}

func (r *InMemoryAlertRuleRepository) ListAlertEvents(_ context.Context, ownerUserID string, ruleID string) ([]domain.AlertEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	owned := r.events[strings.TrimSpace(ownerUserID)]
	items := owned[strings.TrimSpace(ruleID)]
	if len(items) == 0 {
		return []domain.AlertEvent{}, nil
	}

	cloned := make([]domain.AlertEvent, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, domain.CopyAlertEvent(item))
	}
	return cloned, nil
}

func (r *InMemoryAlertRuleRepository) FindLatestAlertEvent(_ context.Context, ownerUserID string, ruleID string, eventKey string) (*domain.AlertEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.events[strings.TrimSpace(ownerUserID)][strings.TrimSpace(ruleID)]
	normalizedEventKey, err := domain.NormalizeAlertEventKey(eventKey)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.EventKey == normalizedEventKey {
			cloned := domain.CopyAlertEvent(item)
			return &cloned, nil
		}
	}
	return nil, nil
}

func (r *InMemoryAlertRuleRepository) CreateAlertEvent(_ context.Context, event domain.AlertEvent) (domain.AlertEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerUserID := strings.TrimSpace(event.OwnerUserID)
	ruleID := strings.TrimSpace(event.AlertRuleID)
	if ownerUserID == "" || ruleID == "" {
		return domain.AlertEvent{}, fmt.Errorf("owner user id and alert rule id are required")
	}
	if _, ok := r.rules[ownerUserID][ruleID]; !ok {
		return domain.AlertEvent{}, ErrAlertRuleNotFound
	}
	if _, ok := r.events[ownerUserID]; !ok {
		r.events[ownerUserID] = make(map[string][]domain.AlertEvent)
	}
	for _, existing := range r.events[ownerUserID][ruleID] {
		if existing.DedupKey == event.DedupKey {
			return domain.AlertEvent{}, db.ErrAlertEventDeduped
		}
	}

	stored := domain.CopyAlertEvent(event)
	r.events[ownerUserID][ruleID] = append([]domain.AlertEvent{stored}, r.events[ownerUserID][ruleID]...)

	rule := r.rules[ownerUserID][ruleID]
	rule.LastTriggeredAt = pointerToTime(stored.ObservedAt.UTC())
	rule.EventCount = rule.EventCount + 1
	rule.UpdatedAt = stored.CreatedAt.UTC()
	r.rules[ownerUserID][ruleID] = rule

	return domain.CopyAlertEvent(stored), nil
}

type PostgresAlertRuleRepository struct {
	store *db.PostgresAlertStore
}

func NewPostgresAlertRuleRepository(store *db.PostgresAlertStore) *PostgresAlertRuleRepository {
	return &PostgresAlertRuleRepository{store: store}
}

func (r *PostgresAlertRuleRepository) ListAlertRules(ctx context.Context, ownerUserID string) ([]domain.AlertRule, error) {
	if r == nil || r.store == nil {
		return []domain.AlertRule{}, nil
	}
	items, err := r.store.ListAlertRules(ctx, ownerUserID)
	if err != nil {
		return nil, translateAlertError(err)
	}
	return items, nil
}

func (r *PostgresAlertRuleRepository) CreateAlertRule(ctx context.Context, rule domain.AlertRule) (domain.AlertRule, error) {
	if r == nil || r.store == nil {
		return domain.AlertRule{}, nil
	}
	created, err := r.store.CreateAlertRule(ctx, db.AlertRuleCreate{
		OwnerUserID:     rule.OwnerUserID,
		Name:            rule.Name,
		RuleType:        rule.RuleType,
		Definition:      rule.Definition,
		Notes:           rule.Notes,
		Tags:            rule.Tags,
		IsEnabled:       rule.IsEnabled,
		CooldownSeconds: rule.CooldownSeconds,
	})
	if err != nil {
		return domain.AlertRule{}, translateAlertError(err)
	}
	return created, nil
}

func (r *PostgresAlertRuleRepository) FindAlertRule(ctx context.Context, ownerUserID string, ruleID string) (domain.AlertRule, error) {
	if r == nil || r.store == nil {
		return domain.AlertRule{}, ErrAlertRuleNotFound
	}
	rules, err := r.store.ListAlertRules(ctx, ownerUserID)
	if err != nil {
		return domain.AlertRule{}, translateAlertError(err)
	}
	for _, rule := range rules {
		if rule.ID == strings.TrimSpace(ruleID) {
			return rule, nil
		}
	}
	return domain.AlertRule{}, ErrAlertRuleNotFound
}

func (r *PostgresAlertRuleRepository) UpdateAlertRule(ctx context.Context, rule domain.AlertRule) (domain.AlertRule, error) {
	if r == nil || r.store == nil {
		return domain.AlertRule{}, nil
	}
	updated, err := r.store.UpdateAlertRule(ctx, db.AlertRuleUpdate{
		OwnerUserID:     rule.OwnerUserID,
		RuleID:          rule.ID,
		Name:            rule.Name,
		RuleType:        rule.RuleType,
		Definition:      rule.Definition,
		Notes:           rule.Notes,
		Tags:            rule.Tags,
		IsEnabled:       rule.IsEnabled,
		CooldownSeconds: rule.CooldownSeconds,
	})
	if err != nil {
		return domain.AlertRule{}, translateAlertError(err)
	}
	return updated, nil
}

func (r *PostgresAlertRuleRepository) DeleteAlertRule(ctx context.Context, ownerUserID string, ruleID string) error {
	if r == nil || r.store == nil {
		return nil
	}
	return translateAlertError(r.store.DeleteAlertRule(ctx, ownerUserID, ruleID))
}

func (r *PostgresAlertRuleRepository) ListAlertEvents(ctx context.Context, ownerUserID string, ruleID string) ([]domain.AlertEvent, error) {
	if r == nil || r.store == nil {
		return []domain.AlertEvent{}, nil
	}
	events, err := r.store.ListAlertEvents(ctx, ownerUserID, ruleID)
	if err != nil {
		return nil, translateAlertError(err)
	}
	return events, nil
}

func (r *PostgresAlertRuleRepository) FindLatestAlertEvent(ctx context.Context, ownerUserID string, ruleID string, eventKey string) (*domain.AlertEvent, error) {
	if r == nil || r.store == nil {
		return nil, nil
	}
	event, err := r.store.FindLatestAlertEvent(ctx, ownerUserID, ruleID, eventKey)
	if err != nil {
		return nil, translateAlertError(err)
	}
	return event, nil
}

func (r *PostgresAlertRuleRepository) CreateAlertEvent(ctx context.Context, event domain.AlertEvent) (domain.AlertEvent, error) {
	if r == nil || r.store == nil {
		return domain.AlertEvent{}, nil
	}
	created, err := r.store.RecordAlertEvent(ctx, db.AlertEventRecord{
		OwnerUserID: event.OwnerUserID,
		AlertRuleID: event.AlertRuleID,
		EventKey:    event.EventKey,
		DedupKey:    event.DedupKey,
		SignalType:  event.SignalType,
		Severity:    event.Severity,
		Payload:     event.Payload,
		ObservedAt:  event.ObservedAt,
	})
	if err != nil {
		return domain.AlertEvent{}, translateAlertError(err)
	}
	return created, nil
}

func translateAlertError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, db.ErrAlertRuleNotFound):
		return ErrAlertRuleNotFound
	default:
		return err
	}
}

func pointerToTime(value time.Time) *time.Time {
	normalized := value.UTC()
	return &normalized
}
