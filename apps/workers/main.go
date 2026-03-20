package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/whalegraph/whalegraph/packages/config"
	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/providers"
)

func main() {
	providerEnv, err := providers.ParseProviderEnvFromOS()
	if err != nil {
		log.Fatalf("provider env validation failed: %v", err)
	}
	env := providerEnv.Worker

	mode := os.Getenv("WHALEGRAPH_WORKER_MODE")
	appCtx := context.Background()
	registry := providers.DefaultRegistry()
	if mode == workerModeHistoricalBackfillIngest {
		registry = providers.NewConfiguredRegistry(providerEnv)
	}
	runner := NewHistoricalBackfillJobRunner(registry)
	ingest := HistoricalBackfillIngestService{}

	var clients *db.StorageClients
	if mode == workerModeHistoricalBackfillIngest {
		clients, err = openWorkerStorageClients(appCtx, env)
		if err != nil {
			log.Fatalf("worker storage init failed: %v", err)
		}
		defer func() {
			if err := clients.Close(context.Background()); err != nil {
				log.Printf("worker storage close error: %v", err)
			}
		}()

		ingest = NewHistoricalBackfillIngestService(
			registry,
			db.NewWalletStoreFromClients(clients),
			db.NewNormalizedTransactionStoreFromClients(clients),
		)
		ingest.RawPayloads = db.NewFilesystemRawPayloadStore(rawPayloadRoot())
		ingest.ProviderUsage = db.NewProviderUsageLogStoreFromClients(clients)
		ingest.JobRuns = db.NewJobRunStoreFromClients(clients)
	}

	output, err := buildWorkerOutput(
		appCtx,
		mode,
		env,
		runner,
		ingest,
	)
	if err != nil {
		log.Fatalf("worker execution failed: %v", err)
	}

	fmt.Println(output)
}

func buildStartupMessage(env config.WorkerEnv) string {
	return fmt.Sprintf(
		"WhaleGraph workers ready (env=%s, postgres=%s, redis=%s)",
		env.NodeEnv,
		env.PostgresURL,
		env.RedisURL,
	)
}

func rawPayloadRoot() string {
	root := strings.TrimSpace(os.Getenv("WHALEGRAPH_RAW_PAYLOAD_ROOT"))
	if root != "" {
		return root
	}

	return ".whalegraph/raw-payloads"
}
