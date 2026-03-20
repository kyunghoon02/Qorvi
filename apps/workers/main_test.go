package main

import (
	"os"
	"strings"
	"testing"

	"github.com/whalegraph/whalegraph/packages/config"
	"github.com/whalegraph/whalegraph/packages/providers"
)

func TestBuildStartupMessage(t *testing.T) {
	t.Parallel()

	message := buildStartupMessage(config.WorkerEnv{
		NodeEnv:     "development",
		PostgresURL: "postgres://postgres:postgres@localhost:5432/whalegraph",
		RedisURL:    "redis://localhost:6379",
	})

	if !strings.Contains(message, "WhaleGraph workers ready") {
		t.Fatalf("unexpected startup message %q", message)
	}
}

func TestRawPayloadRootDefaultsAndHonorsEnv(t *testing.T) {
	t.Setenv("WHALEGRAPH_RAW_PAYLOAD_ROOT", "")
	if got := rawPayloadRoot(); got != ".whalegraph/raw-payloads" {
		t.Fatalf("unexpected default raw payload root %q", got)
	}

	t.Setenv("WHALEGRAPH_RAW_PAYLOAD_ROOT", "/tmp/whalegraph/raw")
	if got := rawPayloadRoot(); got != "/tmp/whalegraph/raw" {
		t.Fatalf("unexpected configured raw payload root %q", got)
	}

	_ = os.Getenv("WHALEGRAPH_RAW_PAYLOAD_ROOT")
}

func TestBuildWorkerOutputRunsHistoricalBackfillFixtureFlow(t *testing.T) {
	t.Parallel()

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeHistoricalBackfillFixture,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/whalegraph",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Historical backfill fixture complete") {
		t.Fatalf("unexpected fixture output %q", output)
	}
	if !strings.Contains(output, "alchemy,helius") {
		t.Fatalf("expected provider list in fixture output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsHistoricalBackfillIngestFlow(t *testing.T) {
	t.Parallel()

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeHistoricalBackfillIngest,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/whalegraph",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		NewHistoricalBackfillIngestService(
			providers.DefaultRegistry(),
			&fakeWalletStore{},
			&fakeTransactionStore{},
		),
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Historical backfill ingest complete") {
		t.Fatalf("unexpected ingest output %q", output)
	}
	if !strings.Contains(output, "transactions=2") {
		t.Fatalf("expected transaction count in ingest output, got %q", output)
	}
}
