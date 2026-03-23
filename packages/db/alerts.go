package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type postgresAlertQuerier interface {
	postgresQuerier
	postgresTransactionExecer
}

var ErrAlertRuleNotFound = errors.New("alert rule not found")
var ErrAlertEventNotFound = errors.New("alert event not found")
var ErrAlertEventDeduped = errors.New("alert event deduped")

type AlertRuleCreate struct {
	OwnerUserID     string
	Name            string
	RuleType        string
	Definition      map[string]any
	Notes           string
	Tags            []string
	IsEnabled       bool
	CooldownSeconds int
}

type AlertRuleUpdate struct {
	OwnerUserID     string
	RuleID          string
	Name            string
	RuleType        string
	Definition      map[string]any
	Notes           string
	Tags            []string
	IsEnabled       bool
	CooldownSeconds int
}

type AlertEventRecord struct {
	OwnerUserID string
	AlertRuleID string
	EventKey    string
	DedupKey    string
	SignalType  string
	Severity    domain.AlertSeverity
	Payload     map[string]any
	ObservedAt  time.Time
}

type AlertRuleStore interface {
	ListAlertRules(context.Context, string) ([]domain.AlertRule, error)
	ListWalletSignalAlertRules(context.Context, WalletRef, string) ([]domain.AlertRule, error)
	CreateAlertRule(context.Context, AlertRuleCreate) (domain.AlertRule, error)
	UpdateAlertRule(context.Context, AlertRuleUpdate) (domain.AlertRule, error)
	DeleteAlertRule(context.Context, string, string) error
}

type AlertEventStore interface {
	ListAlertEvents(context.Context, string, string) ([]domain.AlertEvent, error)
	CountAlertEvents(context.Context, string, string) (int, error)
	FindLatestAlertEvent(context.Context, string, string, string) (*domain.AlertEvent, error)
	RecordAlertEvent(context.Context, AlertEventRecord) (domain.AlertEvent, error)
}

type PostgresAlertStore struct {
	Querier postgresQuerier
	Execer  postgresTransactionExecer
}

const listAlertRulesSQL = `
WITH event_counts AS (
  SELECT alert_rule_id, COUNT(*)::int AS event_count
  FROM alert_events
  GROUP BY alert_rule_id
)
SELECT ar.id, ar.owner_user_id, ar.name, ar.rule_type, ar.definition, ar.notes, ar.tags, ar.is_enabled, ar.cooldown_seconds, ar.last_triggered_at, COALESCE(ec.event_count, 0), ar.created_at, ar.updated_at
FROM alert_rules ar
LEFT JOIN event_counts ec ON ec.alert_rule_id = ar.id
WHERE ar.owner_user_id = $1
ORDER BY ar.updated_at DESC, ar.created_at DESC, ar.id DESC
`

const listWalletSignalAlertRulesSQL = `
WITH event_counts AS (
  SELECT alert_rule_id, COUNT(*)::int AS event_count
  FROM alert_events
  GROUP BY alert_rule_id
)
SELECT DISTINCT ar.id, ar.owner_user_id, ar.name, ar.rule_type, ar.definition, ar.notes, ar.tags, ar.is_enabled, ar.cooldown_seconds, ar.last_triggered_at, COALESCE(ec.event_count, 0), ar.created_at, ar.updated_at
FROM alert_rules ar
JOIN watchlists w
  ON w.owner_user_id = ar.owner_user_id
 AND w.id::text = ar.definition->>'watchlistId'
JOIN watchlist_items wi
  ON wi.watchlist_id = w.id
LEFT JOIN event_counts ec
  ON ec.alert_rule_id = ar.id
WHERE ar.is_enabled = true
  AND wi.item_type = 'wallet'
  AND wi.item_key = $1
ORDER BY ar.updated_at DESC, ar.created_at DESC, ar.id DESC
`

const createAlertRuleSQL = `
WITH inserted AS (
  INSERT INTO alert_rules (
    owner_user_id,
    name,
    rule_type,
    definition,
    notes,
    tags,
    is_enabled,
    cooldown_seconds,
    updated_at
  ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
  RETURNING id, owner_user_id, name, rule_type, definition, notes, tags, is_enabled, cooldown_seconds, last_triggered_at, created_at, updated_at
)
SELECT inserted.id, inserted.owner_user_id, inserted.name, inserted.rule_type, inserted.definition, inserted.notes, inserted.tags, inserted.is_enabled, inserted.cooldown_seconds, inserted.last_triggered_at, COALESCE(ec.event_count, 0), inserted.created_at, inserted.updated_at
FROM inserted
LEFT JOIN (
  SELECT alert_rule_id, COUNT(*)::int AS event_count
  FROM alert_events
  WHERE alert_rule_id = (SELECT id FROM inserted)
  GROUP BY alert_rule_id
) ec ON TRUE
`

const updateAlertRuleSQL = `
WITH updated AS (
  UPDATE alert_rules
  SET name = $3,
      rule_type = $4,
      definition = $5,
      notes = $6,
      tags = $7,
      is_enabled = $8,
      cooldown_seconds = $9,
      updated_at = now()
  WHERE id = $1
    AND owner_user_id = $2
  RETURNING id, owner_user_id, name, rule_type, definition, notes, tags, is_enabled, cooldown_seconds, last_triggered_at, created_at, updated_at
)
SELECT updated.id, updated.owner_user_id, updated.name, updated.rule_type, updated.definition, updated.notes, updated.tags, updated.is_enabled, updated.cooldown_seconds, updated.last_triggered_at, COALESCE(ec.event_count, 0), updated.created_at, updated.updated_at
FROM updated
LEFT JOIN (
  SELECT alert_rule_id, COUNT(*)::int AS event_count
  FROM alert_events
  WHERE alert_rule_id = (SELECT id FROM updated)
  GROUP BY alert_rule_id
) ec ON TRUE
`

const deleteAlertRuleSQL = `
DELETE FROM alert_rules
WHERE id = $1
  AND owner_user_id = $2
`

const listAlertEventsSQL = `
SELECT id, alert_rule_id, owner_user_id, event_key, dedup_key, signal_type, severity, payload, observed_at, read_at, created_at
FROM alert_events
WHERE owner_user_id = $1
  AND alert_rule_id = $2
ORDER BY observed_at DESC, created_at DESC, id DESC
`

const countAlertEventsSQL = `
SELECT COUNT(*)
FROM alert_events
WHERE owner_user_id = $1
  AND alert_rule_id = $2
`

const latestAlertEventSQL = `
SELECT id, alert_rule_id, owner_user_id, event_key, dedup_key, signal_type, severity, payload, observed_at, read_at, created_at
FROM alert_events
WHERE owner_user_id = $1
  AND alert_rule_id = $2
  AND event_key = $3
ORDER BY observed_at DESC, created_at DESC, id DESC
LIMIT 1
`

const alertRuleExistsSQL = `
SELECT 1
FROM alert_rules
WHERE id = $1
  AND owner_user_id = $2
`

const insertAlertEventSQL = `
INSERT INTO alert_events (
  alert_rule_id,
  owner_user_id,
  event_key,
  dedup_key,
  signal_type,
  severity,
  payload,
  observed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (dedup_key) DO NOTHING
RETURNING id, alert_rule_id, owner_user_id, event_key, dedup_key, signal_type, severity, payload, observed_at, read_at, created_at
`

func NewPostgresAlertStore(querier postgresQuerier, execer ...postgresTransactionExecer) *PostgresAlertStore {
	store := &PostgresAlertStore{Querier: querier}
	if len(execer) > 0 {
		store.Execer = execer[0]
		return store
	}
	if adaptiveExecer, ok := querier.(postgresTransactionExecer); ok {
		store.Execer = adaptiveExecer
	}
	return store
}

func NewPostgresAlertStoreFromPool(pool postgresAlertQuerier) *PostgresAlertStore {
	if pool == nil {
		return nil
	}
	return NewPostgresAlertStore(pool, pool)
}

func (s *PostgresAlertStore) ListAlertRules(ctx context.Context, ownerUserID string) ([]domain.AlertRule, error) {
	if s == nil || s.Querier == nil {
		return nil, fmt.Errorf("alert store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil, fmt.Errorf("owner user id is required")
	}

	rows, err := s.Querier.Query(ctx, listAlertRulesSQL, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("list alert rules: %w", err)
	}
	defer rows.Close()

	rules := make([]domain.AlertRule, 0)
	for rows.Next() {
		rule, err := scanAlertRuleRow(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alert rules: %w", err)
	}

	return rules, nil
}

func (s *PostgresAlertStore) ListWalletSignalAlertRules(
	ctx context.Context,
	ref WalletRef,
	signalType string,
) ([]domain.AlertRule, error) {
	if s == nil || s.Querier == nil {
		return nil, fmt.Errorf("alert store is nil")
	}

	normalizedRef, err := NormalizeWalletRef(ref)
	if err != nil {
		return nil, err
	}
	normalizedSignalType, err := domain.NormalizeAlertRuleType(signalType)
	if err != nil {
		return nil, err
	}
	itemKey, err := BuildWatchlistWalletItemKey(normalizedRef)
	if err != nil {
		return nil, err
	}

	rows, err := s.Querier.Query(ctx, listWalletSignalAlertRulesSQL, itemKey)
	if err != nil {
		return nil, fmt.Errorf("list wallet signal alert rules: %w", err)
	}
	defer rows.Close()

	rules := make([]domain.AlertRule, 0)
	for rows.Next() {
		rule, err := scanAlertRuleRow(rows)
		if err != nil {
			return nil, err
		}
		definition, err := domain.ParseAlertRuleDefinition(rule.Definition)
		if err != nil {
			continue
		}
		if len(definition.SignalTypes) > 0 && !containsAlertSignalType(definition.SignalTypes, normalizedSignalType) {
			continue
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wallet signal alert rules: %w", err)
	}

	return rules, nil
}

func (s *PostgresAlertStore) CreateAlertRule(ctx context.Context, create AlertRuleCreate) (domain.AlertRule, error) {
	if s == nil || s.Querier == nil {
		return domain.AlertRule{}, fmt.Errorf("alert store is nil")
	}

	ownerUserID := strings.TrimSpace(create.OwnerUserID)
	if ownerUserID == "" {
		return domain.AlertRule{}, fmt.Errorf("owner user id is required")
	}

	name, err := domain.NormalizeAlertRuleName(create.Name)
	if err != nil {
		return domain.AlertRule{}, err
	}
	ruleType, err := domain.NormalizeAlertRuleType(create.RuleType)
	if err != nil {
		return domain.AlertRule{}, err
	}

	definitionJSON, err := domain.MarshalAlertDefinition(create.Definition)
	if err != nil {
		return domain.AlertRule{}, err
	}
	tagsJSON, err := json.Marshal(domain.NormalizeAlertTags(create.Tags))
	if err != nil {
		return domain.AlertRule{}, fmt.Errorf("marshal alert rule tags: %w", err)
	}
	notes := domain.NormalizeAlertNotes(create.Notes)
	cooldownSeconds := domain.NormalizeAlertCooldownSeconds(create.CooldownSeconds)
	var lastTriggeredAt sql.NullTime

	var rule domain.AlertRule
	if err := s.Querier.QueryRow(
		ctx,
		createAlertRuleSQL,
		ownerUserID,
		name,
		ruleType,
		definitionJSON,
		notes,
		tagsJSON,
		create.IsEnabled,
		cooldownSeconds,
	).Scan(
		&rule.ID,
		&rule.OwnerUserID,
		&rule.Name,
		&rule.RuleType,
		&definitionJSON,
		&rule.Notes,
		&tagsJSON,
		&rule.IsEnabled,
		&rule.CooldownSeconds,
		&lastTriggeredAt,
		&rule.EventCount,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AlertRule{}, ErrAlertRuleNotFound
		}
		return domain.AlertRule{}, fmt.Errorf("create alert rule: %w", err)
	}

	rule.Definition = map[string]any{}
	if len(definitionJSON) > 0 {
		if err := json.Unmarshal(definitionJSON, &rule.Definition); err != nil {
			return domain.AlertRule{}, fmt.Errorf("decode alert rule definition: %w", err)
		}
	}
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &rule.Tags); err != nil {
			return domain.AlertRule{}, fmt.Errorf("decode alert rule tags: %w", err)
		}
	}
	if lastTriggeredAt.Valid {
		seen := lastTriggeredAt.Time.UTC()
		rule.LastTriggeredAt = &seen
	}

	return rule, nil
}

func (s *PostgresAlertStore) UpdateAlertRule(ctx context.Context, update AlertRuleUpdate) (domain.AlertRule, error) {
	if s == nil || s.Querier == nil {
		return domain.AlertRule{}, fmt.Errorf("alert store is nil")
	}

	ownerUserID := strings.TrimSpace(update.OwnerUserID)
	ruleID := strings.TrimSpace(update.RuleID)
	if ownerUserID == "" || ruleID == "" {
		return domain.AlertRule{}, fmt.Errorf("owner user id and alert rule id are required")
	}

	name, err := domain.NormalizeAlertRuleName(update.Name)
	if err != nil {
		return domain.AlertRule{}, err
	}
	ruleType, err := domain.NormalizeAlertRuleType(update.RuleType)
	if err != nil {
		return domain.AlertRule{}, err
	}
	definitionJSON, err := domain.MarshalAlertDefinition(update.Definition)
	if err != nil {
		return domain.AlertRule{}, err
	}
	tagsJSON, err := json.Marshal(domain.NormalizeAlertTags(update.Tags))
	if err != nil {
		return domain.AlertRule{}, fmt.Errorf("marshal alert rule tags: %w", err)
	}
	var lastTriggeredAt sql.NullTime

	var rule domain.AlertRule
	if err := s.Querier.QueryRow(
		ctx,
		updateAlertRuleSQL,
		ruleID,
		ownerUserID,
		name,
		ruleType,
		definitionJSON,
		domain.NormalizeAlertNotes(update.Notes),
		tagsJSON,
		update.IsEnabled,
		domain.NormalizeAlertCooldownSeconds(update.CooldownSeconds),
	).Scan(
		&rule.ID,
		&rule.OwnerUserID,
		&rule.Name,
		&rule.RuleType,
		&definitionJSON,
		&rule.Notes,
		&tagsJSON,
		&rule.IsEnabled,
		&rule.CooldownSeconds,
		&lastTriggeredAt,
		&rule.EventCount,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AlertRule{}, ErrAlertRuleNotFound
		}
		return domain.AlertRule{}, fmt.Errorf("update alert rule: %w", err)
	}

	rule.Definition = map[string]any{}
	if len(definitionJSON) > 0 {
		if err := json.Unmarshal(definitionJSON, &rule.Definition); err != nil {
			return domain.AlertRule{}, fmt.Errorf("decode alert rule definition: %w", err)
		}
	}
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &rule.Tags); err != nil {
			return domain.AlertRule{}, fmt.Errorf("decode alert rule tags: %w", err)
		}
	}
	if lastTriggeredAt.Valid {
		seen := lastTriggeredAt.Time.UTC()
		rule.LastTriggeredAt = &seen
	}

	return rule, nil
}

func (s *PostgresAlertStore) DeleteAlertRule(ctx context.Context, ownerUserID string, ruleID string) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("alert store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	ruleID = strings.TrimSpace(ruleID)
	if ownerUserID == "" || ruleID == "" {
		return fmt.Errorf("owner user id and alert rule id are required")
	}

	commandTag, err := s.Execer.Exec(ctx, deleteAlertRuleSQL, ruleID, ownerUserID)
	if err != nil {
		return fmt.Errorf("delete alert rule: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrAlertRuleNotFound
	}

	return nil
}

func (s *PostgresAlertStore) ListAlertEvents(ctx context.Context, ownerUserID string, ruleID string) ([]domain.AlertEvent, error) {
	if s == nil || s.Querier == nil {
		return nil, fmt.Errorf("alert store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	ruleID = strings.TrimSpace(ruleID)
	if ownerUserID == "" || ruleID == "" {
		return nil, fmt.Errorf("owner user id and alert rule id are required")
	}

	rows, err := s.Querier.Query(ctx, listAlertEventsSQL, ownerUserID, ruleID)
	if err != nil {
		return nil, fmt.Errorf("list alert events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.AlertEvent, 0)
	for rows.Next() {
		event, err := scanAlertEventRow(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alert events: %w", err)
	}

	return events, nil
}

func (s *PostgresAlertStore) CountAlertEvents(ctx context.Context, ownerUserID string, ruleID string) (int, error) {
	if s == nil || s.Querier == nil {
		return 0, fmt.Errorf("alert store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	ruleID = strings.TrimSpace(ruleID)
	if ownerUserID == "" || ruleID == "" {
		return 0, fmt.Errorf("owner user id and alert rule id are required")
	}

	var count int
	if err := s.Querier.QueryRow(ctx, countAlertEventsSQL, ownerUserID, ruleID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count alert events: %w", err)
	}

	return count, nil
}

func (s *PostgresAlertStore) RecordAlertEvent(ctx context.Context, record AlertEventRecord) (domain.AlertEvent, error) {
	if s == nil || s.Querier == nil || s.Execer == nil {
		return domain.AlertEvent{}, fmt.Errorf("alert store is nil")
	}

	ownerUserID := strings.TrimSpace(record.OwnerUserID)
	ruleID := strings.TrimSpace(record.AlertRuleID)
	if ownerUserID == "" || ruleID == "" {
		return domain.AlertEvent{}, fmt.Errorf("owner user id and alert rule id are required")
	}

	eventKey, err := domain.NormalizeAlertEventKey(record.EventKey)
	if err != nil {
		return domain.AlertEvent{}, err
	}
	dedupKey, err := domain.NormalizeAlertDedupKey(record.DedupKey)
	if err != nil {
		return domain.AlertEvent{}, err
	}
	signalType, err := domain.NormalizeAlertRuleType(record.SignalType)
	if err != nil {
		return domain.AlertEvent{}, err
	}
	severity, err := domain.NormalizeAlertSeverity(string(record.Severity))
	if err != nil {
		return domain.AlertEvent{}, err
	}
	payload := domain.NormalizeAlertDefinition(record.Payload)
	observedAt := record.ObservedAt
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	observedAt = observedAt.UTC()

	var ruleExists int
	if err := s.Querier.QueryRow(ctx, alertRuleExistsSQL, ruleID, ownerUserID).Scan(&ruleExists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AlertEvent{}, ErrAlertRuleNotFound
		}
		return domain.AlertEvent{}, fmt.Errorf("load alert rule state: %w", err)
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return domain.AlertEvent{}, fmt.Errorf("marshal alert event payload: %w", err)
	}

	var (
		event  domain.AlertEvent
		readAt sql.NullTime
	)
	if err := s.Querier.QueryRow(
		ctx,
		insertAlertEventSQL,
		ruleID,
		ownerUserID,
		eventKey,
		dedupKey,
		signalType,
		severity,
		payloadJSON,
		observedAt,
	).Scan(
		&event.ID,
		&event.AlertRuleID,
		&event.OwnerUserID,
		&event.EventKey,
		&event.DedupKey,
		&event.SignalType,
		&event.Severity,
		&payloadJSON,
		&event.ObservedAt,
		&readAt,
		&event.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AlertEvent{}, ErrAlertEventDeduped
		}
		return domain.AlertEvent{}, fmt.Errorf("record alert event: %w", err)
	}

	if len(payloadJSON) > 0 {
		if err := json.Unmarshal(payloadJSON, &event.Payload); err != nil {
			return domain.AlertEvent{}, fmt.Errorf("decode alert event payload: %w", err)
		}
	}
	if readAt.Valid {
		value := readAt.Time.UTC()
		event.ReadAt = &value
	}

	return event, nil
}

func (s *PostgresAlertStore) FindLatestAlertEvent(
	ctx context.Context,
	ownerUserID string,
	ruleID string,
	eventKey string,
) (*domain.AlertEvent, error) {
	if s == nil || s.Querier == nil {
		return nil, fmt.Errorf("alert store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	ruleID = strings.TrimSpace(ruleID)
	normalizedEventKey, err := domain.NormalizeAlertEventKey(eventKey)
	if err != nil {
		return nil, err
	}
	if ownerUserID == "" || ruleID == "" {
		return nil, fmt.Errorf("owner user id and alert rule id are required")
	}

	event, err := scanOptionalAlertEventRow(s.Querier.QueryRow(ctx, latestAlertEventSQL, ownerUserID, ruleID, normalizedEventKey))
	if err != nil {
		if errors.Is(err, ErrAlertEventNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &event, nil
}

func scanAlertRuleRow(scanner interface{ Scan(...any) error }) (domain.AlertRule, error) {
	var (
		rule     domain.AlertRule
		defRaw   []byte
		tagsRaw  []byte
		lastSeen sql.NullTime
	)

	if err := scanner.Scan(
		&rule.ID,
		&rule.OwnerUserID,
		&rule.Name,
		&rule.RuleType,
		&defRaw,
		&rule.Notes,
		&tagsRaw,
		&rule.IsEnabled,
		&rule.CooldownSeconds,
		&lastSeen,
		&rule.EventCount,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	); err != nil {
		return domain.AlertRule{}, fmt.Errorf("scan alert rule: %w", err)
	}

	rule.Definition = map[string]any{}
	if len(defRaw) > 0 {
		if err := json.Unmarshal(defRaw, &rule.Definition); err != nil {
			return domain.AlertRule{}, fmt.Errorf("decode alert rule definition: %w", err)
		}
	}
	if len(tagsRaw) > 0 {
		if err := json.Unmarshal(tagsRaw, &rule.Tags); err != nil {
			return domain.AlertRule{}, fmt.Errorf("decode alert rule tags: %w", err)
		}
	} else {
		rule.Tags = []string{}
	}
	if lastSeen.Valid {
		seen := lastSeen.Time.UTC()
		rule.LastTriggeredAt = &seen
	}

	return rule, nil
}

func scanAlertEventRow(scanner interface{ Scan(...any) error }) (domain.AlertEvent, error) {
	var (
		event   domain.AlertEvent
		payload []byte
		readAt  sql.NullTime
	)

	if err := scanner.Scan(
		&event.ID,
		&event.AlertRuleID,
		&event.OwnerUserID,
		&event.EventKey,
		&event.DedupKey,
		&event.SignalType,
		&event.Severity,
		&payload,
		&event.ObservedAt,
		&readAt,
		&event.CreatedAt,
	); err != nil {
		return domain.AlertEvent{}, fmt.Errorf("scan alert event: %w", err)
	}

	event.Payload = map[string]any{}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &event.Payload); err != nil {
			return domain.AlertEvent{}, fmt.Errorf("decode alert event payload: %w", err)
		}
	}
	if readAt.Valid {
		value := readAt.Time.UTC()
		event.ReadAt = &value
	}

	return event, nil
}

func scanOptionalAlertEventRow(scanner interface{ Scan(...any) error }) (domain.AlertEvent, error) {
	var (
		event   domain.AlertEvent
		payload []byte
		readAt  sql.NullTime
	)

	if err := scanner.Scan(
		&event.ID,
		&event.AlertRuleID,
		&event.OwnerUserID,
		&event.EventKey,
		&event.DedupKey,
		&event.SignalType,
		&event.Severity,
		&payload,
		&event.ObservedAt,
		&readAt,
		&event.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AlertEvent{}, ErrAlertEventNotFound
		}
		return domain.AlertEvent{}, fmt.Errorf("scan alert event: %w", err)
	}

	event.Payload = map[string]any{}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &event.Payload); err != nil {
			return domain.AlertEvent{}, fmt.Errorf("decode alert event payload: %w", err)
		}
	}
	if readAt.Valid {
		value := readAt.Time.UTC()
		event.ReadAt = &value
	}

	return event, nil
}

func containsAlertSignalType(values []string, target string) bool {
	for _, item := range values {
		if strings.TrimSpace(item) == strings.TrimSpace(target) {
			return true
		}
	}
	return false
}
