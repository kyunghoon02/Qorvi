package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type postgresSignalEventExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type SignalEventEntry struct {
	WalletID   string
	SignalType string
	Payload    map[string]any
	ObservedAt time.Time
}

type SignalEventStore interface {
	RecordSignalEvent(context.Context, SignalEventEntry) error
	RecordSignalEvents(context.Context, []SignalEventEntry) error
}

type PostgresSignalEventStore struct {
	Execer postgresSignalEventExecer
	Now    func() time.Time
}

const insertSignalEventSQL = `
INSERT INTO signal_events (
  signal_type,
  wallet_id,
  payload,
  observed_at
) VALUES ($1, $2, $3, $4)
`

func NewPostgresSignalEventStore(execer postgresSignalEventExecer) *PostgresSignalEventStore {
	return &PostgresSignalEventStore{
		Execer: execer,
		Now:    time.Now,
	}
}

func NewPostgresSignalEventStoreFromPool(pool postgresSignalEventExecer) *PostgresSignalEventStore {
	return NewPostgresSignalEventStore(pool)
}

func (s *PostgresSignalEventStore) RecordSignalEvent(ctx context.Context, entry SignalEventEntry) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("signal event store is nil")
	}

	normalized, err := normalizeSignalEventEntry(entry)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(normalized.Payload)
	if err != nil {
		return fmt.Errorf("marshal signal event payload: %w", err)
	}

	if _, err := s.Execer.Exec(
		ctx,
		insertSignalEventSQL,
		normalized.SignalType,
		normalized.WalletID,
		payload,
		normalized.ObservedAt.UTC(),
	); err != nil {
		return fmt.Errorf("record signal event: %w", err)
	}

	return nil
}

func (s *PostgresSignalEventStore) RecordSignalEvents(ctx context.Context, entries []SignalEventEntry) error {
	for _, entry := range entries {
		if err := s.RecordSignalEvent(ctx, entry); err != nil {
			return err
		}
	}

	return nil
}

func normalizeSignalEventEntry(entry SignalEventEntry) (SignalEventEntry, error) {
	entry.WalletID = strings.TrimSpace(entry.WalletID)
	entry.SignalType = strings.TrimSpace(entry.SignalType)
	if entry.WalletID == "" {
		return SignalEventEntry{}, fmt.Errorf("wallet id is required")
	}
	if entry.SignalType == "" {
		return SignalEventEntry{}, fmt.Errorf("signal type is required")
	}
	if entry.Payload == nil {
		entry.Payload = map[string]any{}
	}
	if entry.ObservedAt.IsZero() {
		entry.ObservedAt = time.Now()
	}

	return entry, nil
}
