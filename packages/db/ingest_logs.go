package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type postgresExecExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type ProviderUsageLogEntry struct {
	Provider   string
	Operation  string
	StatusCode int
	Latency    time.Duration
}

type ProviderUsageLogStore interface {
	RecordProviderUsageLog(context.Context, ProviderUsageLogEntry) error
	RecordProviderUsageLogs(context.Context, []ProviderUsageLogEntry) error
}

type PostgresProviderUsageLogStore struct {
	Execer postgresExecExecer
}

const insertProviderUsageLogSQL = `
INSERT INTO provider_usage_logs (
  provider,
  operation,
  status_code,
  latency_ms
) VALUES ($1, $2, $3, $4)
`

func NewPostgresProviderUsageLogStore(execer postgresExecExecer) *PostgresProviderUsageLogStore {
	return &PostgresProviderUsageLogStore{Execer: execer}
}

func NewPostgresProviderUsageLogStoreFromPool(pool postgresExecExecer) *PostgresProviderUsageLogStore {
	return NewPostgresProviderUsageLogStore(pool)
}

func (s *PostgresProviderUsageLogStore) RecordProviderUsageLog(
	ctx context.Context,
	entry ProviderUsageLogEntry,
) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("provider usage log store is nil")
	}

	normalized, err := normalizeProviderUsageLogEntry(entry)
	if err != nil {
		return err
	}

	if _, err := s.Execer.Exec(
		ctx,
		insertProviderUsageLogSQL,
		normalized.Provider,
		normalized.Operation,
		normalized.StatusCode,
		normalized.Latency.Milliseconds(),
	); err != nil {
		return fmt.Errorf("record provider usage log: %w", err)
	}

	return nil
}

func (s *PostgresProviderUsageLogStore) RecordProviderUsageLogs(
	ctx context.Context,
	entries []ProviderUsageLogEntry,
) error {
	for _, entry := range entries {
		if err := s.RecordProviderUsageLog(ctx, entry); err != nil {
			return err
		}
	}

	return nil
}

func normalizeProviderUsageLogEntry(entry ProviderUsageLogEntry) (ProviderUsageLogEntry, error) {
	entry.Provider = strings.TrimSpace(entry.Provider)
	entry.Operation = strings.TrimSpace(entry.Operation)
	if entry.Provider == "" {
		return ProviderUsageLogEntry{}, fmt.Errorf("provider is required")
	}
	if entry.Operation == "" {
		return ProviderUsageLogEntry{}, fmt.Errorf("operation is required")
	}
	if entry.StatusCode < 100 || entry.StatusCode > 599 {
		return ProviderUsageLogEntry{}, fmt.Errorf("status code must be between 100 and 599")
	}
	if entry.Latency < 0 {
		return ProviderUsageLogEntry{}, fmt.Errorf("latency must be non-negative")
	}

	return entry, nil
}

type JobRunStatus string

const (
	JobRunStatusRunning   JobRunStatus = "running"
	JobRunStatusSucceeded JobRunStatus = "succeeded"
	JobRunStatusFailed    JobRunStatus = "failed"
)

type JobRunEntry struct {
	JobName    string
	Status     JobRunStatus
	StartedAt  time.Time
	FinishedAt *time.Time
	Details    map[string]any
}

type JobRunStore interface {
	RecordJobRun(context.Context, JobRunEntry) error
	RecordJobRuns(context.Context, []JobRunEntry) error
}

type PostgresJobRunStore struct {
	Execer postgresExecExecer
	Now    func() time.Time
}

const insertJobRunSQL = `
INSERT INTO job_runs (
  job_name,
  status,
  started_at,
  finished_at,
  details
) VALUES ($1, $2, $3, $4, $5)
`

func NewPostgresJobRunStore(execer postgresExecExecer) *PostgresJobRunStore {
	return &PostgresJobRunStore{
		Execer: execer,
		Now:    time.Now,
	}
}

func NewPostgresJobRunStoreFromPool(pool postgresExecExecer) *PostgresJobRunStore {
	return NewPostgresJobRunStore(pool)
}

func (s *PostgresJobRunStore) RecordJobRun(ctx context.Context, entry JobRunEntry) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("job run store is nil")
	}

	normalized, err := normalizeJobRunEntry(entry)
	if err != nil {
		return err
	}

	details, err := json.Marshal(normalized.Details)
	if err != nil {
		return fmt.Errorf("marshal job run details: %w", err)
	}

	if _, err := s.Execer.Exec(
		ctx,
		insertJobRunSQL,
		normalized.JobName,
		string(normalized.Status),
		normalized.StartedAt.UTC(),
		normalized.FinishedAt,
		details,
	); err != nil {
		return fmt.Errorf("record job run: %w", err)
	}

	return nil
}

func (s *PostgresJobRunStore) RecordJobRuns(ctx context.Context, entries []JobRunEntry) error {
	for _, entry := range entries {
		if err := s.RecordJobRun(ctx, entry); err != nil {
			return err
		}
	}

	return nil
}

func normalizeJobRunEntry(entry JobRunEntry) (JobRunEntry, error) {
	entry.JobName = strings.TrimSpace(entry.JobName)
	if entry.JobName == "" {
		return JobRunEntry{}, fmt.Errorf("job name is required")
	}
	switch entry.Status {
	case JobRunStatusRunning, JobRunStatusSucceeded, JobRunStatusFailed:
	default:
		return JobRunEntry{}, fmt.Errorf("unsupported job status %q", entry.Status)
	}

	if entry.StartedAt.IsZero() {
		entry.StartedAt = time.Now()
	}
	if entry.Details == nil {
		entry.Details = map[string]any{}
	}
	if entry.FinishedAt != nil {
		finishedAt := entry.FinishedAt.UTC()
		entry.FinishedAt = &finishedAt
	}

	return entry, nil
}
