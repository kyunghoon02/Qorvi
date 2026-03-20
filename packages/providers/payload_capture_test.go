package providers

import (
	"testing"
	"time"
)

func TestBuildProviderRawPayloadMetadata(t *testing.T) {
	t.Parallel()

	metadata := buildProviderRawPayloadMetadata(
		ProviderAlchemy,
		"alchemy_getAssetTransfers",
		time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC),
		"0xdeadbeef",
		[]byte(`{"hello":"world"}`),
	)

	if metadata["raw_payload_object_key"] == "" {
		t.Fatal("expected raw payload object key")
	}
	if metadata["raw_payload_sha256"] == "" {
		t.Fatal("expected raw payload sha256")
	}
	if metadata["raw_payload_size_bytes"] != 17 {
		t.Fatalf("unexpected size %#v", metadata["raw_payload_size_bytes"])
	}
	if metadata["raw_payload_content_type"] != "application/json" {
		t.Fatalf("unexpected content type %#v", metadata["raw_payload_content_type"])
	}
	if metadata["raw_payload_body"] != `{"hello":"world"}` {
		t.Fatalf("unexpected payload body %#v", metadata["raw_payload_body"])
	}
}

func TestCapturePagePayloadMetadataMergesExistingMetadata(t *testing.T) {
	t.Parallel()

	metadata := capturePagePayloadMetadata(
		ProviderHelius,
		"getTransactionsForAddress",
		time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC),
		"signature",
		[]byte(`{"ok":true}`),
		map[string]any{"existing": "value"},
	)

	if metadata["existing"] != "value" {
		t.Fatalf("expected existing metadata to be preserved, got %#v", metadata["existing"])
	}
	if metadata["raw_payload_provider"] != "helius" {
		t.Fatalf("unexpected payload provider %#v", metadata["raw_payload_provider"])
	}
}
