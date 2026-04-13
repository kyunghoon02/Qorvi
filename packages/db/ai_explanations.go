package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type postgresAIExplanationExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type AIExplanationRecord struct {
	ID                string
	ScopeType         string
	ScopeKey          string
	InputHash         string
	RequestedByUserID string
	Model             string
	PromptVersion     string
	Status            string
	ResponseJSON      map[string]any
	RequestCount      int
	LastRequestedAt   time.Time
	RetryAfter        *time.Time
	LastError         string
	GenerationStarted *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type AIExplanationUpsert struct {
	ScopeType         string
	ScopeKey          string
	InputHash         string
	RequestedByUserID string
	Model             string
	PromptVersion     string
	Status            string
	ResponseJSON      map[string]any
	RetryAfter        *time.Time
	LastError         string
	GenerationStarted *time.Time
}

type AIExplanationStore interface {
	ReadAIExplanationByCacheKey(context.Context, string, string, string, string, string) (AIExplanationRecord, error)
	ReadLatestAIExplanationForScope(context.Context, string, string) (AIExplanationRecord, error)
	UpsertAIExplanation(context.Context, AIExplanationUpsert) (AIExplanationRecord, error)
	CountAIExplanationRequestsByUserSince(context.Context, string, time.Time) (int, error)
}

type PostgresAIExplanationStore struct {
	Querier postgresQuerier
	Execer  postgresAIExplanationExecer
	Now     func() time.Time
}

const readAIExplanationByCacheKeySQL = `
SELECT
  id,
  scope_type,
  scope_key,
  input_hash,
  requested_by_user_id,
  model,
  prompt_version,
  status,
  response_json,
  request_count,
  last_requested_at,
  created_at,
  updated_at
FROM ai_explanations
WHERE scope_type = $1
  AND scope_key = $2
  AND input_hash = $3
  AND model = $4
  AND prompt_version = $5
  AND status = 'completed'
LIMIT 1
`

const readLatestAIExplanationForScopeSQL = `
SELECT
  id,
  scope_type,
  scope_key,
  input_hash,
  requested_by_user_id,
  model,
  prompt_version,
  status,
  response_json,
  request_count,
  last_requested_at,
  retry_after,
  last_error,
  generation_started_at,
  created_at,
  updated_at
FROM ai_explanations
WHERE scope_type = $1
  AND scope_key = $2
ORDER BY last_requested_at DESC, updated_at DESC
LIMIT 1
`

const upsertAIExplanationSQL = `
INSERT INTO ai_explanations (
  scope_type,
  scope_key,
  input_hash,
  requested_by_user_id,
  model,
  prompt_version,
  status,
  response_json,
  request_count,
  last_requested_at,
  retry_after,
  last_error,
  generation_started_at,
  created_at,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, 1, $9, $10, $11, $12, $9, $9
)
ON CONFLICT (scope_type, scope_key, input_hash, model, prompt_version) DO UPDATE SET
  requested_by_user_id = EXCLUDED.requested_by_user_id,
  status = EXCLUDED.status,
  response_json = EXCLUDED.response_json,
  request_count = ai_explanations.request_count + 1,
  last_requested_at = EXCLUDED.last_requested_at,
  retry_after = EXCLUDED.retry_after,
  last_error = EXCLUDED.last_error,
  generation_started_at = EXCLUDED.generation_started_at,
  updated_at = EXCLUDED.updated_at
RETURNING
  id,
  scope_type,
  scope_key,
  input_hash,
  requested_by_user_id,
  model,
  prompt_version,
  status,
  response_json,
  request_count,
  last_requested_at,
  retry_after,
  last_error,
  generation_started_at,
  created_at,
  updated_at
`

const countAIExplanationRequestsByUserSinceSQL = `
SELECT COUNT(*)
FROM ai_explanations
WHERE requested_by_user_id = $1
  AND last_requested_at >= $2
`

func NewPostgresAIExplanationStore(querier postgresQuerier, execer postgresAIExplanationExecer) *PostgresAIExplanationStore {
	return &PostgresAIExplanationStore{
		Querier: querier,
		Execer:  execer,
		Now:     time.Now,
	}
}

func NewPostgresAIExplanationStoreFromPool(pool interface {
	postgresQuerier
	postgresAIExplanationExecer
}) *PostgresAIExplanationStore {
	return NewPostgresAIExplanationStore(pool, pool)
}

func (s *PostgresAIExplanationStore) ReadAIExplanationByCacheKey(
	ctx context.Context,
	scopeType string,
	scopeKey string,
	inputHash string,
	model string,
	promptVersion string,
) (AIExplanationRecord, error) {
	if s == nil || s.Querier == nil {
		return AIExplanationRecord{}, fmt.Errorf("ai explanation store is nil")
	}

	record, err := scanAIExplanationRecord(s.Querier.QueryRow(
		ctx,
		readAIExplanationByCacheKeySQL,
		scopeType,
		scopeKey,
		inputHash,
		model,
		promptVersion,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AIExplanationRecord{}, pgx.ErrNoRows
		}
		return AIExplanationRecord{}, fmt.Errorf("read ai explanation by cache key: %w", err)
	}
	return record, nil
}

func (s *PostgresAIExplanationStore) ReadLatestAIExplanationForScope(
	ctx context.Context,
	scopeType string,
	scopeKey string,
) (AIExplanationRecord, error) {
	if s == nil || s.Querier == nil {
		return AIExplanationRecord{}, fmt.Errorf("ai explanation store is nil")
	}

	record, err := scanAIExplanationRecord(s.Querier.QueryRow(
		ctx,
		readLatestAIExplanationForScopeSQL,
		scopeType,
		scopeKey,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AIExplanationRecord{}, pgx.ErrNoRows
		}
		return AIExplanationRecord{}, fmt.Errorf("read latest ai explanation for scope: %w", err)
	}
	return record, nil
}

func (s *PostgresAIExplanationStore) UpsertAIExplanation(
	ctx context.Context,
	input AIExplanationUpsert,
) (AIExplanationRecord, error) {
	if s == nil || s.Querier == nil || s.Execer == nil {
		return AIExplanationRecord{}, fmt.Errorf("ai explanation store is nil")
	}

	payload, err := json.Marshal(normalizeAIExplanationJSON(input.ResponseJSON))
	if err != nil {
		return AIExplanationRecord{}, fmt.Errorf("marshal ai explanation response: %w", err)
	}

	now := s.now().UTC()
	record, err := scanAIExplanationRecord(s.Querier.QueryRow(
		ctx,
		upsertAIExplanationSQL,
		input.ScopeType,
		input.ScopeKey,
		input.InputHash,
		input.RequestedByUserID,
		input.Model,
		input.PromptVersion,
		input.Status,
		payload,
		now,
		input.RetryAfter,
		input.LastError,
		input.GenerationStarted,
	))
	if err != nil {
		return AIExplanationRecord{}, fmt.Errorf("upsert ai explanation: %w", err)
	}
	return record, nil
}

func (s *PostgresAIExplanationStore) CountAIExplanationRequestsByUserSince(
	ctx context.Context,
	userID string,
	since time.Time,
) (int, error) {
	if s == nil || s.Querier == nil {
		return 0, fmt.Errorf("ai explanation store is nil")
	}
	var count int
	if err := s.Querier.QueryRow(ctx, countAIExplanationRequestsByUserSinceSQL, userID, since.UTC()).Scan(&count); err != nil {
		return 0, fmt.Errorf("count ai explanations by user since: %w", err)
	}
	return count, nil
}

func (s *PostgresAIExplanationStore) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func scanAIExplanationRecord(row pgx.Row) (AIExplanationRecord, error) {
	var (
		record      AIExplanationRecord
		responseRaw []byte
	)
	if err := row.Scan(
		&record.ID,
		&record.ScopeType,
		&record.ScopeKey,
		&record.InputHash,
		&record.RequestedByUserID,
		&record.Model,
		&record.PromptVersion,
		&record.Status,
		&responseRaw,
		&record.RequestCount,
		&record.LastRequestedAt,
		&record.RetryAfter,
		&record.LastError,
		&record.GenerationStarted,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return AIExplanationRecord{}, err
	}
	record.ResponseJSON = map[string]any{}
	if len(responseRaw) > 0 {
		if err := json.Unmarshal(responseRaw, &record.ResponseJSON); err != nil {
			return AIExplanationRecord{}, fmt.Errorf("unmarshal ai explanation response: %w", err)
		}
	}
	return record, nil
}

func normalizeAIExplanationJSON(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	return payload
}
