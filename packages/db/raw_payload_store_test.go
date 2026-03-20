package db

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFilesystemRawPayloadStoreStoreRawPayload(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	store := NewFilesystemRawPayloadStore(rootDir)
	store.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC)
	}

	descriptor := RawPayloadDescriptor{
		Provider:    " alchemy ",
		Operation:   " transfers.backfill ",
		ContentType: " application/json ",
		ObjectKey:   " raw/alchemy/2026/03/20/tx.json ",
		ObservedAt:  time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC),
	}
	payload := []byte(`{"hello":"world"}`)

	err := store.StoreRawPayload(context.Background(), descriptor, payload)
	if err != nil {
		t.Fatalf("StoreRawPayload returned error: %v", err)
	}

	payloadPath := filepath.Join(rootDir, "raw/alchemy/2026/03/20/tx.json")
	storedPayload, err := os.ReadFile(payloadPath)
	if err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if string(storedPayload) != string(payload) {
		t.Fatalf("unexpected payload %q", storedPayload)
	}

	metadataPath := payloadPath + ".meta.json"
	rawMetadata, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}

	var metadata filesystemRawPayloadMetadata
	if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}

	if metadata.Descriptor.Provider != "alchemy" {
		t.Fatalf("unexpected provider %q", metadata.Descriptor.Provider)
	}
	if metadata.Descriptor.SHA256 != RawPayloadSHA256(payload) {
		t.Fatalf("unexpected sha256 %q", metadata.Descriptor.SHA256)
	}
	if metadata.ByteLength != len(payload) {
		t.Fatalf("unexpected byte length %d", metadata.ByteLength)
	}
	if metadata.StoredAt != store.now().UTC() {
		t.Fatalf("unexpected stored_at %v", metadata.StoredAt)
	}
}

func TestFilesystemRawPayloadStoreRejectsEscapingObjectKey(t *testing.T) {
	t.Parallel()

	store := NewFilesystemRawPayloadStore(t.TempDir())
	err := store.StoreRawPayload(context.Background(), RawPayloadDescriptor{
		Provider:    "alchemy",
		Operation:   "transfers.backfill",
		ContentType: "application/json",
		ObjectKey:   "../escape.json",
		ObservedAt:  time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC),
	}, []byte("payload"))
	if err == nil {
		t.Fatal("expected escaping object key to fail")
	}
}

func TestFilesystemRawPayloadStoreRejectsShaMismatch(t *testing.T) {
	t.Parallel()

	store := NewFilesystemRawPayloadStore(t.TempDir())
	err := store.StoreRawPayload(context.Background(), RawPayloadDescriptor{
		Provider:    "alchemy",
		Operation:   "transfers.backfill",
		ContentType: "application/json",
		ObjectKey:   "raw/alchemy/2026/03/20/tx.json",
		SHA256:      "deadbeef",
		ObservedAt:  time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC),
	}, []byte("payload"))
	if err == nil {
		t.Fatal("expected sha mismatch to fail")
	}
}
