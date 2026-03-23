package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type AuditLogEntry struct {
	ActorUserID string
	Action      string
	TargetType  string
	TargetKey   string
	Payload     map[string]any
	CreatedAt   time.Time
}

type PostgresAuditLogStore struct {
	Querier postgresQuerier
	Execer  postgresExecExecer
}

const insertAuditLogSQL = `
INSERT INTO audit_logs (
  actor_user_id,
  action,
  target_type,
  target_key,
  payload,
  created_at
) VALUES ($1, $2, $3, $4, $5, $6)
`

const listAuditLogsSQL = `
SELECT actor_user_id, action, target_type, target_key, payload, created_at
FROM audit_logs
ORDER BY created_at DESC, id DESC
LIMIT $1
`

func NewPostgresAuditLogStore(
	querier postgresQuerier,
	execer ...postgresExecExecer,
) *PostgresAuditLogStore {
	store := &PostgresAuditLogStore{Querier: querier}
	if len(execer) > 0 {
		store.Execer = execer[0]
		return store
	}
	if querier != nil {
		if adaptive, ok := any(querier).(postgresExecExecer); ok {
			store.Execer = adaptive
		}
	}
	return store
}

func NewPostgresAuditLogStoreFromPool(pool postgresQuerier) *PostgresAuditLogStore {
	return NewPostgresAuditLogStore(pool)
}

func (s *PostgresAuditLogStore) RecordAuditLog(ctx context.Context, entry AuditLogEntry) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("audit log store is nil")
	}
	normalized, err := normalizeAuditLogEntry(entry)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(normalized.Payload)
	if err != nil {
		return fmt.Errorf("marshal audit payload: %w", err)
	}
	if _, err := s.Execer.Exec(
		ctx,
		insertAuditLogSQL,
		normalized.ActorUserID,
		normalized.Action,
		normalized.TargetType,
		normalized.TargetKey,
		payload,
		normalized.CreatedAt.UTC(),
	); err != nil {
		return fmt.Errorf("record audit log: %w", err)
	}
	return nil
}

func (s *PostgresAuditLogStore) ListAuditLogs(ctx context.Context, limit int) ([]AuditLogEntry, error) {
	if s == nil || s.Querier == nil {
		return []AuditLogEntry{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.Querier.Query(ctx, listAuditLogsSQL, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	items := make([]AuditLogEntry, 0)
	for rows.Next() {
		var (
			item    AuditLogEntry
			payload []byte
		)
		if err := rows.Scan(
			&item.ActorUserID,
			&item.Action,
			&item.TargetType,
			&item.TargetKey,
			&payload,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		item.Payload = map[string]any{}
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &item.Payload); err != nil {
				return nil, fmt.Errorf("decode audit payload: %w", err)
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit logs: %w", err)
	}
	return items, nil
}

func normalizeAuditLogEntry(entry AuditLogEntry) (AuditLogEntry, error) {
	entry.ActorUserID = strings.TrimSpace(entry.ActorUserID)
	entry.Action = strings.TrimSpace(entry.Action)
	entry.TargetType = strings.TrimSpace(entry.TargetType)
	entry.TargetKey = strings.TrimSpace(entry.TargetKey)
	if entry.ActorUserID == "" {
		return AuditLogEntry{}, fmt.Errorf("audit actor is required")
	}
	if entry.Action == "" {
		return AuditLogEntry{}, fmt.Errorf("audit action is required")
	}
	if entry.TargetType == "" {
		return AuditLogEntry{}, fmt.Errorf("audit target type is required")
	}
	if entry.TargetKey == "" {
		return AuditLogEntry{}, fmt.Errorf("audit target key is required")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	if entry.Payload == nil {
		entry.Payload = map[string]any{}
	}
	return entry, nil
}
