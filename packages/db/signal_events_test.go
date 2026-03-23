package db

import (
	"context"
	"testing"
	"time"
)

func TestPostgresSignalEventStoreRecordSignalEvent(t *testing.T) {
	t.Parallel()

	exec := &capturedExec{}
	store := NewPostgresSignalEventStore(exec)
	observedAt := time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC)

	err := store.RecordSignalEvent(context.Background(), SignalEventEntry{
		WalletID:   " wallet_1 ",
		SignalType: " cluster_score_snapshot ",
		Payload: map[string]any{
			"cluster_score": 91,
			"reason":        "shared counterparties",
		},
		ObservedAt: observedAt,
	})
	if err != nil {
		t.Fatalf("RecordSignalEvent returned error: %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(exec.calls))
	}

	call := exec.calls[0]
	if call.query != insertSignalEventSQL {
		t.Fatalf("unexpected sql %q", call.query)
	}
	if got := call.args[0]; got != "cluster_score_snapshot" {
		t.Fatalf("unexpected signal type arg %#v", got)
	}
	if got := call.args[1]; got != "wallet_1" {
		t.Fatalf("unexpected wallet id arg %#v", got)
	}
	if got, ok := call.args[2].([]byte); !ok || string(got) != `{"cluster_score":91,"reason":"shared counterparties"}` {
		t.Fatalf("unexpected payload arg %#v", call.args[2])
	}
	if got, ok := call.args[3].(time.Time); !ok || !got.Equal(observedAt.UTC()) {
		t.Fatalf("unexpected observed at arg %#v", call.args[3])
	}
}

func TestPostgresSignalEventStoreRecordSignalEvents(t *testing.T) {
	t.Parallel()

	exec := &capturedExec{}
	store := NewPostgresSignalEventStore(exec)

	err := store.RecordSignalEvents(context.Background(), []SignalEventEntry{
		{
			WalletID:   "wallet_1",
			SignalType: "cluster_score_snapshot",
			Payload:    map[string]any{"cluster_score": 91},
			ObservedAt: time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC),
		},
		{
			WalletID:   "wallet_2",
			SignalType: "cluster_score_snapshot",
			Payload:    map[string]any{"cluster_score": 88},
			ObservedAt: time.Date(2026, time.March, 20, 1, 3, 4, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("RecordSignalEvents returned error: %v", err)
	}

	if len(exec.calls) != 2 {
		t.Fatalf("expected 2 exec calls, got %d", len(exec.calls))
	}
}

func TestNormalizeSignalEventEntryRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	_, err := normalizeSignalEventEntry(SignalEventEntry{
		WalletID:   " ",
		SignalType: "cluster_score_snapshot",
	})
	if err == nil {
		t.Fatal("expected empty wallet id to fail")
	}

	_, err = normalizeSignalEventEntry(SignalEventEntry{
		WalletID:   "wallet_1",
		SignalType: " ",
	})
	if err == nil {
		t.Fatal("expected empty signal type to fail")
	}
}
