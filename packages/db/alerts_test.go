package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/flowintel/flowintel/packages/domain"
)

type fakeAlertRows struct {
	rows  [][]any
	index int
	err   error
}

func (r *fakeAlertRows) Close() {}

func (r *fakeAlertRows) Err() error { return r.err }

func (r *fakeAlertRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *fakeAlertRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *fakeAlertRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *fakeAlertRows) Scan(dest ...any) error {
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
		case *int:
			*target = value.(int)
		case *bool:
			*target = value.(bool)
		case *[]byte:
			*target = append([]byte(nil), value.([]byte)...)
		case *time.Time:
			*target = value.(time.Time)
		case *sql.NullTime:
			switch typed := value.(type) {
			case sql.NullTime:
				*target = typed
			case nil:
				*target = sql.NullTime{}
			default:
				return errors.New("unexpected null time source type")
			}
		case *domain.AlertSeverity:
			*target = value.(domain.AlertSeverity)
		default:
			return errors.New("unexpected scan destination type")
		}
	}
	return nil
}

func (r *fakeAlertRows) Values() ([]any, error) { return nil, nil }

func (r *fakeAlertRows) RawValues() [][]byte { return nil }

func (r *fakeAlertRows) Conn() *pgx.Conn { return nil }

type fakeAlertRow struct {
	scan func(dest ...any) error
}

func (r fakeAlertRow) Scan(dest ...any) error {
	if r.scan != nil {
		return r.scan(dest...)
	}
	return nil
}

type fakeAlertStoreQuerier struct {
	queryRows map[string]pgx.Rows
	rowScans  map[string]func(dest ...any) error
	execTags  map[string]pgconn.CommandTag
}

func (q *fakeAlertStoreQuerier) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	if rows, ok := q.queryRows[sql]; ok {
		return rows, nil
	}
	return &fakeAlertRows{}, nil
}

func (q *fakeAlertStoreQuerier) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	return fakeAlertRow{scan: q.rowScans[sql]}
}

func (q *fakeAlertStoreQuerier) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	if tag, ok := q.execTags[sql]; ok {
		return tag, nil
	}
	return pgconn.NewCommandTag("DELETE 0"), nil
}

func TestPostgresAlertStoreRuleCrud(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 3, 4, 5, 0, time.UTC)
	definitionJSON, err := json.Marshal(map[string]any{"threshold": 7})
	if err != nil {
		t.Fatalf("marshal definition: %v", err)
	}
	tagsJSON, err := json.Marshal([]string{"vip", "seed"})
	if err != nil {
		t.Fatalf("marshal tags: %v", err)
	}

	querier := &fakeAlertStoreQuerier{
		queryRows: map[string]pgx.Rows{
			listAlertRulesSQL: &fakeAlertRows{
				rows: [][]any{
					{"rule_1", "owner_1", "Shadow Exit", "shadow_exit", definitionJSON, "notes", tagsJSON, true, 3600, sql.NullTime{}, 2, now, now},
				},
			},
		},
		rowScans: map[string]func(dest ...any) error{
			createAlertRuleSQL: func(dest ...any) error {
				*(dest[0].(*string)) = "rule_1"
				*(dest[1].(*string)) = "owner_1"
				*(dest[2].(*string)) = "Shadow Exit"
				*(dest[3].(*string)) = "shadow_exit"
				*(dest[4].(*[]byte)) = definitionJSON
				*(dest[5].(*string)) = "notes"
				*(dest[6].(*[]byte)) = tagsJSON
				*(dest[7].(*bool)) = true
				*(dest[8].(*int)) = 3600
				*(dest[9].(*sql.NullTime)) = sql.NullTime{}
				*(dest[10].(*int)) = 0
				*(dest[11].(*time.Time)) = now
				*(dest[12].(*time.Time)) = now
				return nil
			},
			updateAlertRuleSQL: func(dest ...any) error {
				*(dest[0].(*string)) = "rule_1"
				*(dest[1].(*string)) = "owner_1"
				*(dest[2].(*string)) = "Shadow Exit Updated"
				*(dest[3].(*string)) = "shadow_exit"
				*(dest[4].(*[]byte)) = definitionJSON
				*(dest[5].(*string)) = "notes updated"
				*(dest[6].(*[]byte)) = tagsJSON
				*(dest[7].(*bool)) = false
				*(dest[8].(*int)) = 1800
				*(dest[9].(*sql.NullTime)) = sql.NullTime{}
				*(dest[10].(*int)) = 3
				*(dest[11].(*time.Time)) = now
				*(dest[12].(*time.Time)) = now.Add(time.Minute)
				return nil
			},
		},
		execTags: map[string]pgconn.CommandTag{
			deleteAlertRuleSQL: pgconn.NewCommandTag("DELETE 1"),
		},
	}

	store := NewPostgresAlertStore(querier, querier)

	rules, err := store.ListAlertRules(context.Background(), "owner_1")
	if err != nil {
		t.Fatalf("ListAlertRules returned error: %v", err)
	}
	if len(rules) != 1 || rules[0].EventCount != 2 {
		t.Fatalf("unexpected rules %#v", rules)
	}

	created, err := store.CreateAlertRule(context.Background(), AlertRuleCreate{
		OwnerUserID:     "owner_1",
		Name:            " Shadow Exit ",
		RuleType:        "shadow_exit",
		Definition:      map[string]any{"threshold": 7},
		Notes:           " notes ",
		Tags:            []string{"seed", "vip"},
		IsEnabled:       true,
		CooldownSeconds: 3600,
	})
	if err != nil {
		t.Fatalf("CreateAlertRule returned error: %v", err)
	}
	if created.Name != "Shadow Exit" || created.CooldownSeconds != 3600 {
		t.Fatalf("unexpected created rule %#v", created)
	}

	updated, err := store.UpdateAlertRule(context.Background(), AlertRuleUpdate{
		OwnerUserID:     "owner_1",
		RuleID:          "rule_1",
		Name:            "Shadow Exit Updated",
		RuleType:        "shadow_exit",
		Definition:      map[string]any{"threshold": 7},
		Notes:           "notes updated",
		Tags:            []string{"vip", "seed"},
		IsEnabled:       false,
		CooldownSeconds: 1800,
	})
	if err != nil {
		t.Fatalf("UpdateAlertRule returned error: %v", err)
	}
	if updated.Name != "Shadow Exit Updated" || updated.IsEnabled {
		t.Fatalf("unexpected updated rule %#v", updated)
	}

	if err := store.DeleteAlertRule(context.Background(), "owner_1", "rule_1"); err != nil {
		t.Fatalf("DeleteAlertRule returned error: %v", err)
	}
}

func TestPostgresAlertStoreListWalletSignalAlertRules(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 3, 4, 5, 0, time.UTC)
	definitionJSON, err := json.Marshal(map[string]any{
		"watchlistId":                "watch_1",
		"signalTypes":                []string{"shadow_exit"},
		"minimumSeverity":            "high",
		"renotifyOnSeverityIncrease": true,
	})
	if err != nil {
		t.Fatalf("marshal definition: %v", err)
	}
	tagsJSON, err := json.Marshal([]string{"vip", "seed"})
	if err != nil {
		t.Fatalf("marshal tags: %v", err)
	}

	store := NewPostgresAlertStore(&fakeAlertStoreQuerier{
		queryRows: map[string]pgx.Rows{
			listWalletSignalAlertRulesSQL: &fakeAlertRows{
				rows: [][]any{
					{"rule_1", "owner_1", "Shadow Exit", "watchlist_signal", definitionJSON, "notes", tagsJSON, true, 3600, sql.NullTime{}, 2, now, now},
					{"rule_2", "owner_2", "First Connection", "watchlist_signal", []byte(`{"watchlistId":"watch_1","signalTypes":["first_connection"],"minimumSeverity":"medium"}`), "notes", tagsJSON, true, 0, sql.NullTime{}, 0, now, now},
				},
			},
		},
	})

	rules, err := store.ListWalletSignalAlertRules(context.Background(), WalletRef{
		Chain:   "evm",
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "shadow_exit")
	if err != nil {
		t.Fatalf("ListWalletSignalAlertRules returned error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 matching rule, got %d", len(rules))
	}
	if rules[0].ID != "rule_1" {
		t.Fatalf("unexpected rule %#v", rules[0])
	}
}

func TestPostgresAlertStoreRecordAlertEventDedupAndCooldown(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 4, 5, 6, 0, time.UTC)
	payloadJSON, err := json.Marshal(map[string]any{"score_value": 92})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	querier := &fakeAlertStoreQuerier{
		rowScans: map[string]func(dest ...any) error{
			alertRuleExistsSQL: func(dest ...any) error {
				*(dest[0].(*int)) = 1
				return nil
			},
			insertAlertEventSQL: func(dest ...any) error {
				*(dest[0].(*string)) = "event_1"
				*(dest[1].(*string)) = "rule_1"
				*(dest[2].(*string)) = "owner_1"
				*(dest[3].(*string)) = "event:1"
				*(dest[4].(*string)) = "rule_1:event:1:high:2026-03-21T04:05:06Z"
				*(dest[5].(*string)) = "shadow_exit"
				*(dest[6].(*domain.AlertSeverity)) = domain.AlertSeverityHigh
				*(dest[7].(*[]byte)) = payloadJSON
				*(dest[8].(*time.Time)) = now
				*(dest[9].(*sql.NullTime)) = sql.NullTime{}
				*(dest[10].(*time.Time)) = now
				return nil
			},
		},
	}

	store := NewPostgresAlertStore(querier, querier)

	event, err := store.RecordAlertEvent(context.Background(), AlertEventRecord{
		OwnerUserID: "owner_1",
		AlertRuleID: "rule_1",
		EventKey:    "event:1",
		DedupKey:    "rule_1:event:1:high:2026-03-21T04:05:06Z",
		SignalType:  "shadow_exit",
		Severity:    domain.AlertSeverityHigh,
		Payload:     map[string]any{"score_value": 92},
		ObservedAt:  now,
	})
	if err != nil {
		t.Fatalf("RecordAlertEvent returned error: %v", err)
	}
	if event.EventKey != "event:1" || event.SignalType != "shadow_exit" {
		t.Fatalf("unexpected recorded event %#v", event)
	}
	if event.Severity != domain.AlertSeverityHigh {
		t.Fatalf("unexpected severity %#v", event)
	}

	dedupStore := NewPostgresAlertStore(&fakeAlertStoreQuerier{
		rowScans: map[string]func(dest ...any) error{
			alertRuleExistsSQL: func(dest ...any) error {
				*(dest[0].(*int)) = 1
				return nil
			},
			insertAlertEventSQL: func(dest ...any) error {
				return pgx.ErrNoRows
			},
		},
	})
	_, err = dedupStore.RecordAlertEvent(context.Background(), AlertEventRecord{
		OwnerUserID: "owner_1",
		AlertRuleID: "rule_1",
		EventKey:    "event:1",
		DedupKey:    "rule_1:event:1:high:2026-03-21T04:05:06Z",
		SignalType:  "shadow_exit",
		Severity:    domain.AlertSeverityHigh,
		Payload:     map[string]any{"score_value": 92},
		ObservedAt:  now,
	})
	if !errors.Is(err, ErrAlertEventDeduped) {
		t.Fatalf("expected ErrAlertEventDeduped, got %v", err)
	}
}

func TestPostgresAlertStoreFindLatestAlertEvent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 4, 5, 6, 0, time.UTC)
	payloadJSON, err := json.Marshal(map[string]any{"score_value": 92})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	store := NewPostgresAlertStore(&fakeAlertStoreQuerier{
		rowScans: map[string]func(dest ...any) error{
			latestAlertEventSQL: func(dest ...any) error {
				*(dest[0].(*string)) = "event_1"
				*(dest[1].(*string)) = "rule_1"
				*(dest[2].(*string)) = "owner_1"
				*(dest[3].(*string)) = "event:1"
				*(dest[4].(*string)) = "rule_1:event:1:high:2026-03-21T04:05:06Z"
				*(dest[5].(*string)) = "shadow_exit"
				*(dest[6].(*domain.AlertSeverity)) = domain.AlertSeverityHigh
				*(dest[7].(*[]byte)) = payloadJSON
				*(dest[8].(*time.Time)) = now
				*(dest[9].(*sql.NullTime)) = sql.NullTime{}
				*(dest[10].(*time.Time)) = now
				return nil
			},
		},
	})

	event, err := store.FindLatestAlertEvent(context.Background(), "owner_1", "rule_1", "event:1")
	if err != nil {
		t.Fatalf("FindLatestAlertEvent returned error: %v", err)
	}
	if event == nil || event.DedupKey == "" || event.Severity != domain.AlertSeverityHigh {
		t.Fatalf("unexpected event %#v", event)
	}
}
