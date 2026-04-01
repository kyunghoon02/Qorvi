package db

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type fakeAIExplanationExecer struct{}

func (fakeAIExplanationExecer) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func TestPostgresAIExplanationStoreReadsCacheKey(t *testing.T) {
	t.Parallel()

	response, err := json.Marshal(map[string]any{
		"summary": "cached summary",
	})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	store := NewPostgresAIExplanationStore(
		fakePostgresQuerier{
			row: fakeRow{
				scan: func(dest ...any) error {
					*(dest[0].(*string)) = "exp_123"
					*(dest[1].(*string)) = "finding"
					*(dest[2].(*string)) = "finding_123"
					*(dest[3].(*string)) = "hash_123"
					*(dest[4].(*string)) = "user_123"
					*(dest[5].(*string)) = "gpt-4o-mini"
					*(dest[6].(*string)) = "finding-explainer-v1"
					*(dest[7].(*string)) = "completed"
					*(dest[8].(*[]byte)) = response
					*(dest[9].(*int)) = 2
					*(dest[10].(*time.Time)) = time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC)
					*(dest[11].(**time.Time)) = nil
					*(dest[12].(*string)) = ""
					*(dest[13].(**time.Time)) = nil
					*(dest[14].(*time.Time)) = time.Date(2026, 3, 29, 8, 59, 0, 0, time.UTC)
					*(dest[15].(*time.Time)) = time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC)
					return nil
				},
			},
		},
		fakeAIExplanationExecer{},
	)

	record, err := store.ReadAIExplanationByCacheKey(context.Background(), "finding", "finding_123", "hash_123", "gpt-4o-mini", "finding-explainer-v1")
	if err != nil {
		t.Fatalf("read explanation by cache key: %v", err)
	}
	if record.ID != "exp_123" || record.RequestCount != 2 {
		t.Fatalf("unexpected record %+v", record)
	}
	if record.ResponseJSON["summary"] != "cached summary" {
		t.Fatalf("unexpected response payload %+v", record.ResponseJSON)
	}
}

func TestPostgresAIExplanationStoreUpsertsExplanation(t *testing.T) {
	t.Parallel()

	store := NewPostgresAIExplanationStore(
		fakePostgresQuerier{
			row: fakeRow{
				scan: func(dest ...any) error {
					*(dest[0].(*string)) = "exp_123"
					*(dest[1].(*string)) = "finding"
					*(dest[2].(*string)) = "finding_123"
					*(dest[3].(*string)) = "hash_123"
					*(dest[4].(*string)) = "user_123"
					*(dest[5].(*string)) = "gpt-4o-mini"
					*(dest[6].(*string)) = "finding-explainer-v1"
					*(dest[7].(*string)) = "completed"
					*(dest[8].(*[]byte)) = []byte(`{"summary":"fresh summary"}`)
					*(dest[9].(*int)) = 1
					*(dest[10].(*time.Time)) = time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC)
					*(dest[11].(**time.Time)) = nil
					*(dest[12].(*string)) = ""
					*(dest[13].(**time.Time)) = nil
					*(dest[14].(*time.Time)) = time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC)
					*(dest[15].(*time.Time)) = time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC)
					return nil
				},
			},
		},
		fakeAIExplanationExecer{},
	)
	store.Now = func() time.Time { return time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC) }

	record, err := store.UpsertAIExplanation(context.Background(), AIExplanationUpsert{
		ScopeType:         "finding",
		ScopeKey:          "finding_123",
		InputHash:         "hash_123",
		RequestedByUserID: "user_123",
		Model:             "gpt-4o-mini",
		PromptVersion:     "finding-explainer-v1",
		Status:            "completed",
		ResponseJSON: map[string]any{
			"summary": "fresh summary",
		},
	})
	if err != nil {
		t.Fatalf("upsert explanation: %v", err)
	}
	if record.ResponseJSON["summary"] != "fresh summary" {
		t.Fatalf("unexpected response payload %+v", record.ResponseJSON)
	}
}

func TestPostgresAIExplanationStoreCountsRequestsByUserSince(t *testing.T) {
	t.Parallel()

	store := NewPostgresAIExplanationStore(
		fakePostgresQuerier{
			row: fakeRow{
				scan: func(dest ...any) error {
					*(dest[0].(*int)) = 7
					return nil
				},
			},
		},
		fakeAIExplanationExecer{},
	)

	count, err := store.CountAIExplanationRequestsByUserSince(context.Background(), "user_123", time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("count requests: %v", err)
	}
	if count != 7 {
		t.Fatalf("expected count 7, got %d", count)
	}
}
