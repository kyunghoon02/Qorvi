package db

import (
	"context"
	"testing"
	"time"
)

func TestPostgresProviderUsageLogStoreRecordProviderUsageLog(t *testing.T) {
	t.Parallel()

	exec := &capturedExec{}
	store := NewPostgresProviderUsageLogStore(exec)

	err := store.RecordProviderUsageLog(context.Background(), ProviderUsageLogEntry{
		Provider:   " alchemy ",
		Operation:  " transfers.backfill ",
		StatusCode: 200,
		Latency:    1250 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("RecordProviderUsageLog returned error: %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(exec.calls))
	}

	call := exec.calls[0]
	if call.query != insertProviderUsageLogSQL {
		t.Fatalf("unexpected sql %q", call.query)
	}
	if got := call.args[0]; got != "alchemy" {
		t.Fatalf("unexpected provider arg %#v", got)
	}
	if got := call.args[1]; got != "transfers.backfill" {
		t.Fatalf("unexpected operation arg %#v", got)
	}
	if got := call.args[2]; got != 200 {
		t.Fatalf("unexpected status code arg %#v", got)
	}
	if got := call.args[3]; got != int64(1250) {
		t.Fatalf("unexpected latency arg %#v", got)
	}
}

func TestPostgresJobRunStoreRecordJobRun(t *testing.T) {
	t.Parallel()

	exec := &capturedExec{}
	store := NewPostgresJobRunStore(exec)
	startedAt := time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC)
	finishedAt := startedAt.Add(2 * time.Minute)

	err := store.RecordJobRun(context.Background(), JobRunEntry{
		JobName:    "historical-backfill",
		Status:     JobRunStatusSucceeded,
		StartedAt:  startedAt,
		FinishedAt: &finishedAt,
		Details: map[string]any{
			"provider": "alchemy",
			"count":    2,
		},
	})
	if err != nil {
		t.Fatalf("RecordJobRun returned error: %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(exec.calls))
	}

	call := exec.calls[0]
	if call.query != insertJobRunSQL {
		t.Fatalf("unexpected sql %q", call.query)
	}
	if got := call.args[0]; got != "historical-backfill" {
		t.Fatalf("unexpected job name arg %#v", got)
	}
	if got := call.args[1]; got != "succeeded" {
		t.Fatalf("unexpected status arg %#v", got)
	}
	if got, ok := call.args[2].(time.Time); !ok || !got.Equal(startedAt.UTC()) {
		t.Fatalf("unexpected started at arg %#v", call.args[2])
	}
	if got, ok := call.args[3].(*time.Time); !ok || got == nil || !got.Equal(finishedAt.UTC()) {
		t.Fatalf("unexpected finished at arg %#v", call.args[3])
	}
	if got, ok := call.args[4].([]byte); !ok || len(got) == 0 {
		t.Fatalf("unexpected details arg %#v", call.args[4])
	}
}

func TestNormalizeProviderUsageLogEntryRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	_, err := normalizeProviderUsageLogEntry(ProviderUsageLogEntry{
		Provider:   " ",
		Operation:  "ingest",
		StatusCode: 200,
		Latency:    time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected error for empty provider")
	}
}

func TestNormalizeJobRunEntryDefaultsDetailsAndStartedAt(t *testing.T) {
	t.Parallel()

	entry, err := normalizeJobRunEntry(JobRunEntry{
		JobName: " ingestion ",
		Status:  JobRunStatusRunning,
	})
	if err != nil {
		t.Fatalf("normalizeJobRunEntry returned error: %v", err)
	}

	if entry.JobName != "ingestion" {
		t.Fatalf("unexpected job name %q", entry.JobName)
	}
	if entry.Details == nil {
		t.Fatal("expected details map")
	}
	if entry.StartedAt.IsZero() {
		t.Fatal("expected started at to be defaulted")
	}
}
