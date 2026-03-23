package db

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeAuditRows struct {
	rows  [][]any
	index int
	err   error
}

func (r *fakeAuditRows) Close() {}

func (r *fakeAuditRows) Err() error { return r.err }

func (r *fakeAuditRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *fakeAuditRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *fakeAuditRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *fakeAuditRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("scan called out of range")
	}
	row := r.rows[r.index-1]
	if len(dest) != len(row) {
		return errors.New("unexpected scan destination count")
	}
	for index, value := range row {
		switch target := dest[index].(type) {
		case *string:
			*target = value.(string)
		case *[]byte:
			*target = append([]byte(nil), value.([]byte)...)
		case *time.Time:
			*target = value.(time.Time)
		default:
			return errors.New("unexpected scan destination type")
		}
	}
	return nil
}

func (r *fakeAuditRows) Values() ([]any, error) { return nil, nil }

func (r *fakeAuditRows) RawValues() [][]byte { return nil }

func (r *fakeAuditRows) Conn() *pgx.Conn { return nil }

type fakeAuditQuerier struct {
	queryRows map[string]pgx.Rows
}

func (q *fakeAuditQuerier) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	if rows, ok := q.queryRows[sql]; ok {
		return rows, nil
	}
	return &fakeAuditRows{}, nil
}

func (q *fakeAuditQuerier) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return fakeRow{}
}

func (q *fakeAuditQuerier) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("INSERT 1"), nil
}

func TestPostgresAuditLogStoreRecordAndList(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 10, 11, 12, 0, time.UTC)
	querier := &fakeAuditQuerier{
		queryRows: map[string]pgx.Rows{
			listAuditLogsSQL: &fakeAuditRows{
				rows: [][]any{
					{
						"admin_1",
						"label_upsert",
						"label",
						"label/high-value",
						[]byte(`{"note":"updated label color"}`),
						now,
					},
				},
			},
		},
	}
	store := NewPostgresAuditLogStore(querier, querier)

	if err := store.RecordAuditLog(context.Background(), AuditLogEntry{
		ActorUserID: " admin_1 ",
		Action:      " label_upsert ",
		TargetType:  " label ",
		TargetKey:   " label/high-value ",
		Payload:     map[string]any{"note": "updated label color"},
		CreatedAt:   now,
	}); err != nil {
		t.Fatalf("RecordAuditLog returned error: %v", err)
	}

	records, err := store.ListAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListAuditLogs returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(records))
	}
	if records[0].ActorUserID != "admin_1" || records[0].TargetType != "label" {
		t.Fatalf("unexpected audit log %#v", records[0])
	}
	if got := records[0].Payload["note"]; got != "updated label color" {
		t.Fatalf("unexpected audit payload %#v", records[0].Payload)
	}
}

func TestNormalizeAuditLogEntryDefaultsPayloadAndTimestamp(t *testing.T) {
	t.Parallel()

	entry, err := normalizeAuditLogEntry(AuditLogEntry{
		ActorUserID: " admin_1 ",
		Action:      " label_upsert ",
		TargetType:  " label ",
		TargetKey:   " label/high-value ",
	})
	if err != nil {
		t.Fatalf("normalizeAuditLogEntry returned error: %v", err)
	}
	if entry.Payload == nil {
		t.Fatal("expected payload map")
	}
	if entry.CreatedAt.IsZero() {
		t.Fatal("expected created at to be defaulted")
	}

	payload, err := json.Marshal(entry.Payload)
	if err != nil || string(payload) != "{}" {
		t.Fatalf("unexpected payload json %q err=%v", payload, err)
	}
}
