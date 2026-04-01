package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCuratedWalletSeedsFromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "curated-wallet-seeds.json")
	payload := `[
		{
			"chain": "evm",
			"address": "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
			"displayName": "vitalik.eth",
			"description": "Founder wallet",
			"category": "founder",
			"trackingPriority": 260,
			"candidateScore": 0.97,
			"confidence": 0.99,
			"tags": ["Founder", "Ethereum", "featured"]
		}
	]`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	seeds, err := LoadCuratedWalletSeedsFromFile(path)
	if err != nil {
		t.Fatalf("LoadCuratedWalletSeedsFromFile returned error: %v", err)
	}
	if len(seeds) != 1 {
		t.Fatalf("expected 1 seed, got %d", len(seeds))
	}
	if seeds[0].Chain != "evm" {
		t.Fatalf("unexpected chain %q", seeds[0].Chain)
	}
	if seeds[0].TrackingPriority != 260 {
		t.Fatalf("unexpected tracking priority %d", seeds[0].TrackingPriority)
	}
	if len(seeds[0].Tags) != 3 || seeds[0].Tags[0] != "founder" {
		t.Fatalf("unexpected tags %#v", seeds[0].Tags)
	}
}

func TestCuratedWalletSeedsPathFromEnvDefaults(t *testing.T) {
	t.Setenv("QORVI_CURATED_WALLET_SEEDS_PATH", "")
	if path := CuratedWalletSeedsPathFromEnv(); path != DefaultCuratedWalletSeedsPath {
		t.Fatalf("unexpected default path %q", path)
	}
}
