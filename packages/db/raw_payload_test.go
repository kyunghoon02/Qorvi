package db

import (
	"testing"
	"time"
)

func TestBuildRawPayloadObjectKey(t *testing.T) {
	t.Parallel()

	key := BuildRawPayloadObjectKey(
		"Alchemy",
		"Transfers Backfill",
		time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC),
		"0x1234.json",
	)

	if key != "alchemy/transfers-backfill/2026/03/20/0x1234.json" {
		t.Fatalf("unexpected object key %q", key)
	}
}

func TestNormalizeRawPayloadDescriptor(t *testing.T) {
	t.Parallel()

	descriptor, err := NormalizeRawPayloadDescriptor(RawPayloadDescriptor{
		Provider:    " alchemy ",
		Operation:   " transfers.backfill ",
		ContentType: " application/json ",
		ObjectKey:   " raw/alchemy/2026/03/20/tx.json ",
		ObservedAt:  time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NormalizeRawPayloadDescriptor returned error: %v", err)
	}

	if descriptor.Provider != "alchemy" {
		t.Fatalf("unexpected provider %q", descriptor.Provider)
	}
	if descriptor.ObjectKey != "raw/alchemy/2026/03/20/tx.json" {
		t.Fatalf("unexpected object key %q", descriptor.ObjectKey)
	}
}

func TestRawPayloadSHA256(t *testing.T) {
	t.Parallel()

	hash := RawPayloadSHA256([]byte("hello"))
	if len(hash) != 64 {
		t.Fatalf("expected hex hash, got %q", hash)
	}
}
