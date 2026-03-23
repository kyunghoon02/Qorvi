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

var ErrAlertDeliveryChannelNotFound = errors.New("alert delivery channel not found")
var ErrAlertDeliveryAttemptNotFound = errors.New("alert delivery attempt not found")
var ErrAlertDeliveryAttemptDeduped = errors.New("alert delivery attempt deduped")

type postgresAlertDeliveryQuerier interface {
	postgresQuerier
	postgresTransactionExecer
}

type AlertInboxQuery struct {
	Limit      int
	Severity   string
	SignalType string
	Cursor     string
	UnreadOnly bool
}

type AlertInboxPage struct {
	Items       []domain.AlertEvent
	NextCursor  *string
	HasMore     bool
	UnreadCount int
}

type AlertDeliveryChannelCreate struct {
	OwnerUserID string
	Label       string
	ChannelType string
	Target      string
	Metadata    map[string]any
	IsEnabled   bool
}

type AlertDeliveryChannelUpdate struct {
	OwnerUserID string
	ChannelID   string
	Label       string
	Target      string
	Metadata    map[string]any
	IsEnabled   bool
}

type AlertDeliveryAttemptCreate struct {
	AlertEventID string
	ChannelID    string
	OwnerUserID  string
	DeliveryKey  string
	ChannelType  domain.AlertChannelType
	Target       string
	Status       domain.AlertDeliveryStatus
	ResponseCode int
	Details      map[string]any
	AttemptedAt  *time.Time
	DeliveredAt  *time.Time
	FailedAt     *time.Time
}

type AlertDeliveryAttemptUpdate struct {
	AttemptID    string
	Status       domain.AlertDeliveryStatus
	ResponseCode int
	Details      map[string]any
	AttemptedAt  *time.Time
	DeliveredAt  *time.Time
	FailedAt     *time.Time
}

type AlertDeliveryRetryCandidate struct {
	Attempt domain.AlertDeliveryAttempt
	Event   domain.AlertEvent
}

type AlertInboxStore interface {
	ListAlertInboxEvents(context.Context, string, AlertInboxQuery) (AlertInboxPage, error)
	MarkAlertInboxEventRead(context.Context, string, string, *time.Time) (domain.AlertEvent, error)
}

type AlertDeliveryChannelStore interface {
	ListAlertDeliveryChannels(context.Context, string) ([]domain.AlertDeliveryChannel, error)
	ListEnabledAlertDeliveryChannels(context.Context, string) ([]domain.AlertDeliveryChannel, error)
	CreateAlertDeliveryChannel(context.Context, AlertDeliveryChannelCreate) (domain.AlertDeliveryChannel, error)
	UpdateAlertDeliveryChannel(context.Context, AlertDeliveryChannelUpdate) (domain.AlertDeliveryChannel, error)
	DeleteAlertDeliveryChannel(context.Context, string, string) error
}

type AlertDeliveryAttemptStore interface {
	CreateAlertDeliveryAttempt(context.Context, AlertDeliveryAttemptCreate) (domain.AlertDeliveryAttempt, error)
	UpdateAlertDeliveryAttempt(context.Context, AlertDeliveryAttemptUpdate) (domain.AlertDeliveryAttempt, error)
	ListRetryableAlertDeliveryAttempts(context.Context, time.Time, int) ([]AlertDeliveryRetryCandidate, error)
}

type PostgresAlertDeliveryStore struct {
	Querier postgresQuerier
	Execer  postgresTransactionExecer
}

const listAlertInboxEventsSQL = `
SELECT id, alert_rule_id, owner_user_id, event_key, dedup_key, signal_type, severity, payload, observed_at, read_at, created_at
FROM alert_events
WHERE owner_user_id = $1
  AND ($2 = '' OR severity = $2)
  AND ($3 = '' OR signal_type = $3)
  AND ($4 = false OR read_at IS NULL)
  AND (
    $5 = false OR observed_at < $6 OR (observed_at = $6 AND id < $7)
  )
ORDER BY observed_at DESC, id DESC
LIMIT $8
`

const countUnreadAlertInboxEventsSQL = `
SELECT COUNT(*)
FROM alert_events
WHERE owner_user_id = $1
  AND read_at IS NULL
`

const markAlertInboxEventReadSQL = `
UPDATE alert_events
SET read_at = $3
WHERE owner_user_id = $1
  AND id = $2
RETURNING id, alert_rule_id, owner_user_id, event_key, dedup_key, signal_type, severity, payload, observed_at, read_at, created_at
`

const listAlertDeliveryChannelsSQL = `
SELECT id, owner_user_id, label, channel_type, target, metadata, is_enabled, created_at, updated_at
FROM alert_delivery_channels
WHERE owner_user_id = $1
ORDER BY updated_at DESC, created_at DESC, id DESC
`

const listEnabledAlertDeliveryChannelsSQL = `
SELECT id, owner_user_id, label, channel_type, target, metadata, is_enabled, created_at, updated_at
FROM alert_delivery_channels
WHERE owner_user_id = $1
  AND is_enabled = true
ORDER BY updated_at DESC, created_at DESC, id DESC
`

const createAlertDeliveryChannelSQL = `
INSERT INTO alert_delivery_channels (
  owner_user_id,
  label,
  channel_type,
  target,
  metadata,
  is_enabled,
  updated_at
) VALUES ($1, $2, $3, $4, $5, $6, now())
RETURNING id, owner_user_id, label, channel_type, target, metadata, is_enabled, created_at, updated_at
`

const updateAlertDeliveryChannelSQL = `
UPDATE alert_delivery_channels
SET label = $3,
    target = $4,
    metadata = $5,
    is_enabled = $6,
    updated_at = now()
WHERE id = $1
  AND owner_user_id = $2
RETURNING id, owner_user_id, label, channel_type, target, metadata, is_enabled, created_at, updated_at
`

const deleteAlertDeliveryChannelSQL = `
DELETE FROM alert_delivery_channels
WHERE id = $1
  AND owner_user_id = $2
`

const createAlertDeliveryAttemptSQL = `
INSERT INTO alert_delivery_attempts (
  alert_event_id,
  channel_id,
  owner_user_id,
  delivery_key,
  channel_type,
  target,
  status,
  response_code,
  details,
  attempted_at,
  delivered_at,
  failed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (delivery_key) DO NOTHING
RETURNING id, alert_event_id, channel_id, owner_user_id, delivery_key, channel_type, target, status, response_code, details, attempted_at, delivered_at, failed_at, created_at
`

const updateAlertDeliveryAttemptSQL = `
UPDATE alert_delivery_attempts
SET status = $2,
    response_code = $3,
    details = $4,
    attempted_at = $5,
    delivered_at = $6,
    failed_at = $7
WHERE id = $1
RETURNING id, alert_event_id, channel_id, owner_user_id, delivery_key, channel_type, target, status, response_code, details, attempted_at, delivered_at, failed_at, created_at
`

const listRetryableAlertDeliveryAttemptsSQL = `
SELECT
  a.id,
  a.alert_event_id,
  a.channel_id,
  a.owner_user_id,
  a.delivery_key,
  a.channel_type,
  a.target,
  a.status,
  a.response_code,
  a.details,
  a.attempted_at,
  a.delivered_at,
  a.failed_at,
  a.created_at,
  e.id,
  e.alert_rule_id,
  e.owner_user_id,
  e.event_key,
  e.dedup_key,
  e.signal_type,
  e.severity,
  e.payload,
  e.observed_at,
  e.created_at
FROM alert_delivery_attempts a
JOIN alert_events e
  ON e.id = a.alert_event_id
JOIN alert_delivery_channels c
  ON c.id = a.channel_id
WHERE a.status = 'failed'
  AND c.is_enabled = true
  AND COALESCE((a.details->>'retry_exhausted')::boolean, false) = false
  AND COALESCE(
    NULLIF(a.details->>'next_retry_at', '')::timestamptz,
    a.failed_at,
    a.attempted_at,
    a.created_at
  ) <= $1
ORDER BY
  COALESCE(
    NULLIF(a.details->>'next_retry_at', '')::timestamptz,
    a.failed_at,
    a.attempted_at,
    a.created_at
  ) ASC,
  a.created_at ASC,
  a.id ASC
LIMIT $2
`

func NewPostgresAlertDeliveryStore(querier postgresQuerier, execer ...postgresTransactionExecer) *PostgresAlertDeliveryStore {
	store := &PostgresAlertDeliveryStore{Querier: querier}
	if len(execer) > 0 {
		store.Execer = execer[0]
		return store
	}
	if adaptiveExecer, ok := querier.(postgresTransactionExecer); ok {
		store.Execer = adaptiveExecer
	}
	return store
}

func NewPostgresAlertDeliveryStoreFromPool(pool postgresAlertDeliveryQuerier) *PostgresAlertDeliveryStore {
	if pool == nil {
		return nil
	}
	return NewPostgresAlertDeliveryStore(pool, pool)
}

func (s *PostgresAlertDeliveryStore) ListAlertInboxEvents(
	ctx context.Context,
	ownerUserID string,
	query AlertInboxQuery,
) (AlertInboxPage, error) {
	if s == nil || s.Querier == nil {
		return AlertInboxPage{}, fmt.Errorf("alert delivery store is nil")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return AlertInboxPage{}, fmt.Errorf("owner user id is required")
	}

	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	severity := strings.TrimSpace(strings.ToLower(query.Severity))
	signalType := strings.TrimSpace(strings.ToLower(query.SignalType))
	cursorObservedAt, cursorID, hasCursor, err := decodeAlertInboxCursor(strings.TrimSpace(query.Cursor))
	if err != nil {
		return AlertInboxPage{}, err
	}

	rows, err := s.Querier.Query(
		ctx,
		listAlertInboxEventsSQL,
		ownerUserID,
		severity,
		signalType,
		query.UnreadOnly,
		hasCursor,
		cursorObservedAt,
		cursorID,
		limit+1,
	)
	if err != nil {
		return AlertInboxPage{}, fmt.Errorf("list alert inbox events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.AlertEvent, 0)
	for rows.Next() {
		event, err := scanAlertEventRow(rows)
		if err != nil {
			return AlertInboxPage{}, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return AlertInboxPage{}, fmt.Errorf("iterate alert inbox events: %w", err)
	}

	hasMore := len(events) > limit
	if hasMore {
		events = events[:limit]
	}

	unreadCount := 0
	if err := s.Querier.QueryRow(ctx, countUnreadAlertInboxEventsSQL, ownerUserID).Scan(&unreadCount); err != nil {
		return AlertInboxPage{}, fmt.Errorf("count unread alert inbox events: %w", err)
	}

	var nextCursor *string
	if hasMore && len(events) > 0 {
		cursor := encodeAlertInboxCursor(events[len(events)-1].ObservedAt, events[len(events)-1].ID)
		nextCursor = &cursor
	}

	return AlertInboxPage{
		Items:       events,
		NextCursor:  nextCursor,
		HasMore:     hasMore,
		UnreadCount: unreadCount,
	}, nil
}

func (s *PostgresAlertDeliveryStore) MarkAlertInboxEventRead(
	ctx context.Context,
	ownerUserID string,
	eventID string,
	readAt *time.Time,
) (domain.AlertEvent, error) {
	if s == nil || s.Querier == nil {
		return domain.AlertEvent{}, fmt.Errorf("alert delivery store is nil")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	eventID = strings.TrimSpace(eventID)
	if ownerUserID == "" || eventID == "" {
		return domain.AlertEvent{}, fmt.Errorf("owner user id and event id are required")
	}

	event, err := scanOptionalAlertEventRow(s.Querier.QueryRow(ctx, markAlertInboxEventReadSQL, ownerUserID, eventID, readAt))
	if err != nil {
		if errors.Is(err, ErrAlertEventNotFound) {
			return domain.AlertEvent{}, ErrAlertEventNotFound
		}
		return domain.AlertEvent{}, err
	}
	return event, nil
}

func (s *PostgresAlertDeliveryStore) ListAlertDeliveryChannels(
	ctx context.Context,
	ownerUserID string,
) ([]domain.AlertDeliveryChannel, error) {
	return s.listAlertDeliveryChannels(ctx, ownerUserID, false)
}

func (s *PostgresAlertDeliveryStore) ListEnabledAlertDeliveryChannels(
	ctx context.Context,
	ownerUserID string,
) ([]domain.AlertDeliveryChannel, error) {
	return s.listAlertDeliveryChannels(ctx, ownerUserID, true)
}

func (s *PostgresAlertDeliveryStore) listAlertDeliveryChannels(
	ctx context.Context,
	ownerUserID string,
	enabledOnly bool,
) ([]domain.AlertDeliveryChannel, error) {
	if s == nil || s.Querier == nil {
		return nil, fmt.Errorf("alert delivery store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil, fmt.Errorf("owner user id is required")
	}

	query := listAlertDeliveryChannelsSQL
	if enabledOnly {
		query = listEnabledAlertDeliveryChannelsSQL
	}

	rows, err := s.Querier.Query(ctx, query, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("list alert delivery channels: %w", err)
	}
	defer rows.Close()

	channels := make([]domain.AlertDeliveryChannel, 0)
	for rows.Next() {
		channel, err := scanAlertDeliveryChannelRow(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alert delivery channels: %w", err)
	}

	return channels, nil
}

func (s *PostgresAlertDeliveryStore) CreateAlertDeliveryChannel(
	ctx context.Context,
	create AlertDeliveryChannelCreate,
) (domain.AlertDeliveryChannel, error) {
	if s == nil || s.Querier == nil {
		return domain.AlertDeliveryChannel{}, fmt.Errorf("alert delivery store is nil")
	}

	ownerUserID := strings.TrimSpace(create.OwnerUserID)
	label, err := domain.NormalizeAlertChannelLabel(create.Label)
	if err != nil {
		return domain.AlertDeliveryChannel{}, err
	}
	channelType, err := domain.NormalizeAlertChannelType(create.ChannelType)
	if err != nil {
		return domain.AlertDeliveryChannel{}, err
	}
	target, err := domain.NormalizeAlertChannelTarget(channelType, create.Target)
	if err != nil {
		return domain.AlertDeliveryChannel{}, err
	}
	metadata, err := json.Marshal(domain.NormalizeAlertDefinition(create.Metadata))
	if err != nil {
		return domain.AlertDeliveryChannel{}, fmt.Errorf("marshal alert delivery channel metadata: %w", err)
	}

	channel, err := scanAlertDeliveryChannelRow(
		s.Querier.QueryRow(ctx, createAlertDeliveryChannelSQL, ownerUserID, label, channelType, target, metadata, create.IsEnabled),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AlertDeliveryChannel{}, ErrAlertDeliveryChannelNotFound
		}
		return domain.AlertDeliveryChannel{}, err
	}
	return channel, nil
}

func (s *PostgresAlertDeliveryStore) UpdateAlertDeliveryChannel(
	ctx context.Context,
	update AlertDeliveryChannelUpdate,
) (domain.AlertDeliveryChannel, error) {
	if s == nil || s.Querier == nil {
		return domain.AlertDeliveryChannel{}, fmt.Errorf("alert delivery store is nil")
	}

	ownerUserID := strings.TrimSpace(update.OwnerUserID)
	channelID := strings.TrimSpace(update.ChannelID)
	if ownerUserID == "" || channelID == "" {
		return domain.AlertDeliveryChannel{}, fmt.Errorf("owner user id and channel id are required")
	}

	label, err := domain.NormalizeAlertChannelLabel(update.Label)
	if err != nil {
		return domain.AlertDeliveryChannel{}, err
	}

	current, err := s.findAlertDeliveryChannel(ctx, ownerUserID, channelID)
	if err != nil {
		return domain.AlertDeliveryChannel{}, err
	}

	target, err := domain.NormalizeAlertChannelTarget(current.ChannelType, update.Target)
	if err != nil {
		return domain.AlertDeliveryChannel{}, err
	}
	metadata, err := json.Marshal(domain.NormalizeAlertDefinition(update.Metadata))
	if err != nil {
		return domain.AlertDeliveryChannel{}, fmt.Errorf("marshal alert delivery channel metadata: %w", err)
	}

	channel, err := scanAlertDeliveryChannelRow(
		s.Querier.QueryRow(ctx, updateAlertDeliveryChannelSQL, channelID, ownerUserID, label, target, metadata, update.IsEnabled),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AlertDeliveryChannel{}, ErrAlertDeliveryChannelNotFound
		}
		return domain.AlertDeliveryChannel{}, err
	}
	return channel, nil
}

func (s *PostgresAlertDeliveryStore) DeleteAlertDeliveryChannel(ctx context.Context, ownerUserID string, channelID string) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("alert delivery store is nil")
	}
	ownerUserID = strings.TrimSpace(ownerUserID)
	channelID = strings.TrimSpace(channelID)
	if ownerUserID == "" || channelID == "" {
		return fmt.Errorf("owner user id and channel id are required")
	}

	commandTag, err := s.Execer.Exec(ctx, deleteAlertDeliveryChannelSQL, channelID, ownerUserID)
	if err != nil {
		return fmt.Errorf("delete alert delivery channel: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrAlertDeliveryChannelNotFound
	}
	return nil
}

func (s *PostgresAlertDeliveryStore) CreateAlertDeliveryAttempt(
	ctx context.Context,
	create AlertDeliveryAttemptCreate,
) (domain.AlertDeliveryAttempt, error) {
	if s == nil || s.Querier == nil {
		return domain.AlertDeliveryAttempt{}, fmt.Errorf("alert delivery store is nil")
	}

	deliveryKey, err := domain.NormalizeAlertDedupKey(create.DeliveryKey)
	if err != nil {
		return domain.AlertDeliveryAttempt{}, err
	}
	channelType, err := domain.NormalizeAlertChannelType(string(create.ChannelType))
	if err != nil {
		return domain.AlertDeliveryAttempt{}, err
	}
	target, err := domain.NormalizeAlertChannelTarget(channelType, create.Target)
	if err != nil {
		return domain.AlertDeliveryAttempt{}, err
	}
	status, err := domain.NormalizeAlertDeliveryStatus(string(create.Status))
	if err != nil {
		return domain.AlertDeliveryAttempt{}, err
	}
	details, err := json.Marshal(domain.NormalizeAlertDefinition(create.Details))
	if err != nil {
		return domain.AlertDeliveryAttempt{}, fmt.Errorf("marshal alert delivery attempt details: %w", err)
	}

	attempt, err := scanOptionalAlertDeliveryAttemptRow(
		s.Querier.QueryRow(
			ctx,
			createAlertDeliveryAttemptSQL,
			strings.TrimSpace(create.AlertEventID),
			strings.TrimSpace(create.ChannelID),
			strings.TrimSpace(create.OwnerUserID),
			deliveryKey,
			channelType,
			target,
			status,
			create.ResponseCode,
			details,
			create.AttemptedAt,
			create.DeliveredAt,
			create.FailedAt,
		),
	)
	if err != nil {
		if errors.Is(err, ErrAlertDeliveryAttemptNotFound) {
			return domain.AlertDeliveryAttempt{}, ErrAlertDeliveryAttemptDeduped
		}
		return domain.AlertDeliveryAttempt{}, err
	}
	return attempt, nil
}

func (s *PostgresAlertDeliveryStore) UpdateAlertDeliveryAttempt(
	ctx context.Context,
	update AlertDeliveryAttemptUpdate,
) (domain.AlertDeliveryAttempt, error) {
	if s == nil || s.Querier == nil {
		return domain.AlertDeliveryAttempt{}, fmt.Errorf("alert delivery store is nil")
	}

	attemptID := strings.TrimSpace(update.AttemptID)
	if attemptID == "" {
		return domain.AlertDeliveryAttempt{}, fmt.Errorf("attempt id is required")
	}
	status, err := domain.NormalizeAlertDeliveryStatus(string(update.Status))
	if err != nil {
		return domain.AlertDeliveryAttempt{}, err
	}
	details, err := json.Marshal(domain.NormalizeAlertDefinition(update.Details))
	if err != nil {
		return domain.AlertDeliveryAttempt{}, fmt.Errorf("marshal alert delivery attempt details: %w", err)
	}

	attempt, err := scanOptionalAlertDeliveryAttemptRow(
		s.Querier.QueryRow(
			ctx,
			updateAlertDeliveryAttemptSQL,
			attemptID,
			status,
			update.ResponseCode,
			details,
			update.AttemptedAt,
			update.DeliveredAt,
			update.FailedAt,
		),
	)
	if err != nil {
		if errors.Is(err, ErrAlertDeliveryAttemptNotFound) {
			return domain.AlertDeliveryAttempt{}, ErrAlertDeliveryAttemptNotFound
		}
		return domain.AlertDeliveryAttempt{}, err
	}
	return attempt, nil
}

func (s *PostgresAlertDeliveryStore) ListRetryableAlertDeliveryAttempts(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]AlertDeliveryRetryCandidate, error) {
	if s == nil || s.Querier == nil {
		return nil, fmt.Errorf("alert delivery store is nil")
	}

	if limit <= 0 || limit > 100 {
		limit = 25
	}

	rows, err := s.Querier.Query(ctx, listRetryableAlertDeliveryAttemptsSQL, now.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("list retryable alert delivery attempts: %w", err)
	}
	defer rows.Close()

	candidates := make([]AlertDeliveryRetryCandidate, 0)
	for rows.Next() {
		candidate, err := scanAlertDeliveryRetryCandidateRow(rows)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate retryable alert delivery attempts: %w", err)
	}

	return candidates, nil
}

func (s *PostgresAlertDeliveryStore) findAlertDeliveryChannel(
	ctx context.Context,
	ownerUserID string,
	channelID string,
) (domain.AlertDeliveryChannel, error) {
	channels, err := s.ListAlertDeliveryChannels(ctx, ownerUserID)
	if err != nil {
		return domain.AlertDeliveryChannel{}, err
	}
	for _, channel := range channels {
		if channel.ID == channelID {
			return channel, nil
		}
	}
	return domain.AlertDeliveryChannel{}, ErrAlertDeliveryChannelNotFound
}

func scanAlertDeliveryChannelRow(scanner interface{ Scan(...any) error }) (domain.AlertDeliveryChannel, error) {
	var (
		channelTypeRaw string
		metadataRaw    []byte
		channel        domain.AlertDeliveryChannel
	)
	if err := scanner.Scan(
		&channel.ID,
		&channel.OwnerUserID,
		&channel.Label,
		&channelTypeRaw,
		&channel.Target,
		&metadataRaw,
		&channel.IsEnabled,
		&channel.CreatedAt,
		&channel.UpdatedAt,
	); err != nil {
		return domain.AlertDeliveryChannel{}, fmt.Errorf("scan alert delivery channel: %w", err)
	}

	channelType, err := domain.NormalizeAlertChannelType(channelTypeRaw)
	if err != nil {
		return domain.AlertDeliveryChannel{}, err
	}
	channel.ChannelType = channelType
	channel.Metadata = map[string]any{}
	if len(metadataRaw) > 0 {
		if err := json.Unmarshal(metadataRaw, &channel.Metadata); err != nil {
			return domain.AlertDeliveryChannel{}, fmt.Errorf("decode alert delivery channel metadata: %w", err)
		}
	}

	return channel, nil
}

func scanOptionalAlertDeliveryAttemptRow(scanner interface{ Scan(...any) error }) (domain.AlertDeliveryAttempt, error) {
	attempt, err := scanAlertDeliveryAttemptRow(scanner)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AlertDeliveryAttempt{}, ErrAlertDeliveryAttemptNotFound
		}
		return domain.AlertDeliveryAttempt{}, err
	}
	return attempt, nil
}

func scanAlertDeliveryAttemptRow(scanner interface{ Scan(...any) error }) (domain.AlertDeliveryAttempt, error) {
	var (
		channelTypeRaw string
		statusRaw      string
		detailsRaw     []byte
		attemptedAt    sql.NullTime
		deliveredAt    sql.NullTime
		failedAt       sql.NullTime
		attempt        domain.AlertDeliveryAttempt
	)
	if err := scanner.Scan(
		&attempt.ID,
		&attempt.AlertEventID,
		&attempt.ChannelID,
		&attempt.OwnerUserID,
		&attempt.DeliveryKey,
		&channelTypeRaw,
		&attempt.Target,
		&statusRaw,
		&attempt.ResponseCode,
		&detailsRaw,
		&attemptedAt,
		&deliveredAt,
		&failedAt,
		&attempt.CreatedAt,
	); err != nil {
		return domain.AlertDeliveryAttempt{}, fmt.Errorf("scan alert delivery attempt: %w", err)
	}

	channelType, err := domain.NormalizeAlertChannelType(channelTypeRaw)
	if err != nil {
		return domain.AlertDeliveryAttempt{}, err
	}
	status, err := domain.NormalizeAlertDeliveryStatus(statusRaw)
	if err != nil {
		return domain.AlertDeliveryAttempt{}, err
	}

	attempt.ChannelType = channelType
	attempt.Status = status
	attempt.Details = map[string]any{}
	if len(detailsRaw) > 0 {
		if err := json.Unmarshal(detailsRaw, &attempt.Details); err != nil {
			return domain.AlertDeliveryAttempt{}, fmt.Errorf("decode alert delivery attempt details: %w", err)
		}
	}
	if attemptedAt.Valid {
		value := attemptedAt.Time.UTC()
		attempt.AttemptedAt = &value
	}
	if deliveredAt.Valid {
		value := deliveredAt.Time.UTC()
		attempt.DeliveredAt = &value
	}
	if failedAt.Valid {
		value := failedAt.Time.UTC()
		attempt.FailedAt = &value
	}

	return attempt, nil
}

func scanAlertDeliveryRetryCandidateRow(scanner interface{ Scan(...any) error }) (AlertDeliveryRetryCandidate, error) {
	var (
		attempt        domain.AlertDeliveryAttempt
		channelTypeRaw string
		statusRaw      string
		detailsRaw     []byte
		attemptedAt    sql.NullTime
		deliveredAt    sql.NullTime
		failedAt       sql.NullTime
		event          domain.AlertEvent
		payloadRaw     []byte
	)

	if err := scanner.Scan(
		&attempt.ID,
		&attempt.AlertEventID,
		&attempt.ChannelID,
		&attempt.OwnerUserID,
		&attempt.DeliveryKey,
		&channelTypeRaw,
		&attempt.Target,
		&statusRaw,
		&attempt.ResponseCode,
		&detailsRaw,
		&attemptedAt,
		&deliveredAt,
		&failedAt,
		&attempt.CreatedAt,
		&event.ID,
		&event.AlertRuleID,
		&event.OwnerUserID,
		&event.EventKey,
		&event.DedupKey,
		&event.SignalType,
		&event.Severity,
		&payloadRaw,
		&event.ObservedAt,
		&event.CreatedAt,
	); err != nil {
		return AlertDeliveryRetryCandidate{}, fmt.Errorf("scan alert delivery retry candidate: %w", err)
	}

	channelType, err := domain.NormalizeAlertChannelType(channelTypeRaw)
	if err != nil {
		return AlertDeliveryRetryCandidate{}, err
	}
	status, err := domain.NormalizeAlertDeliveryStatus(statusRaw)
	if err != nil {
		return AlertDeliveryRetryCandidate{}, err
	}

	attempt.ChannelType = channelType
	attempt.Status = status
	attempt.Details = map[string]any{}
	if len(detailsRaw) > 0 {
		if err := json.Unmarshal(detailsRaw, &attempt.Details); err != nil {
			return AlertDeliveryRetryCandidate{}, fmt.Errorf("decode alert delivery retry details: %w", err)
		}
	}
	if attemptedAt.Valid {
		value := attemptedAt.Time.UTC()
		attempt.AttemptedAt = &value
	}
	if deliveredAt.Valid {
		value := deliveredAt.Time.UTC()
		attempt.DeliveredAt = &value
	}
	if failedAt.Valid {
		value := failedAt.Time.UTC()
		attempt.FailedAt = &value
	}

	event.Payload = map[string]any{}
	if len(payloadRaw) > 0 {
		if err := json.Unmarshal(payloadRaw, &event.Payload); err != nil {
			return AlertDeliveryRetryCandidate{}, fmt.Errorf("decode alert delivery retry payload: %w", err)
		}
	}

	return AlertDeliveryRetryCandidate{
		Attempt: attempt,
		Event:   event,
	}, nil
}

func decodeAlertInboxCursor(cursor string) (time.Time, string, bool, error) {
	trimmed := strings.TrimSpace(cursor)
	if trimmed == "" {
		return time.Time{}, "", false, nil
	}
	parts := strings.SplitN(trimmed, "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", false, fmt.Errorf("invalid alert inbox cursor")
	}
	observedAt, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return time.Time{}, "", false, fmt.Errorf("invalid alert inbox cursor")
	}
	eventID := strings.TrimSpace(parts[1])
	if eventID == "" {
		return time.Time{}, "", false, fmt.Errorf("invalid alert inbox cursor")
	}
	return observedAt.UTC(), eventID, true, nil
}

func encodeAlertInboxCursor(observedAt time.Time, eventID string) string {
	return observedAt.UTC().Format(time.RFC3339) + "|" + strings.TrimSpace(eventID)
}
