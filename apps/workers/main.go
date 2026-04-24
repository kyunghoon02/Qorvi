package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/qorvi/qorvi/packages/billing"
	"github.com/qorvi/qorvi/packages/config"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/providers"
)

func main() {
	mode := os.Getenv("QORVI_WORKER_MODE")
	env := workerEnvFromOS()
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	registry := providers.DefaultRegistry()
	if requiresProviderRegistry(mode) {
		providerEnv, err := providers.ParseProviderEnvFromOS()
		if err != nil {
			log.Fatalf("provider env validation failed: %v", err)
		}
		registry = providers.NewConfiguredRegistry(providerEnv)
	}
	runner := NewHistoricalBackfillJobRunner(registry)
	seedDiscovery := NewSeedDiscoveryJobRunner(registry)
	ingest := HistoricalBackfillIngestService{}
	enrichmentRefresh := WalletEnrichmentRefreshService{}
	watchlistBootstrap := WatchlistBootstrapService{}
	clusterScore := ClusterScoreSnapshotService{}
	shadowExit := ShadowExitSnapshotService{}
	firstConnection := FirstConnectionSnapshotService{}
	alerts := &WalletSignalAlertDispatcher{}
	deliveries := &WalletSignalDeliveryDispatcher{}
	alertDeliveryRetry := AlertDeliveryRetryService{}
	trackingSubscriptionSync := TrackingSubscriptionSyncService{}
	billingSubscriptionSync := BillingSubscriptionSyncService{}
	exchangeListingRegistrySync := ExchangeListingRegistrySyncService{
		Upbit:   providers.NewUpbitExchangeListingClient("", nil),
		Bithumb: providers.NewBithumbExchangeListingClient("", nil),
	}

	var clients *db.StorageClients
	if requiresWorkerStorage(mode) {
		var err error
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
		ingest.DailyStats = db.NewWalletDailyStatsStoreFromClients(clients)
		ingest.Graph = db.NewTransactionGraphMaterializerFromClients(clients)
		ingest.GraphCache = db.NewRedisWalletGraphCache(clients.Redis)
		ingest.GraphSnapshots = db.NewPostgresWalletGraphSnapshotStoreFromPool(clients.Postgres)
		ingest.Dedup = db.NewIngestDedupStoreFromClients(clients)
		ingest.Queue = db.NewWalletBackfillQueueStoreFromClients(clients)
		ingest.RawPayloads = db.NewFilesystemRawPayloadStore(rawPayloadRoot())
		ingest.ProviderUsage = db.NewProviderUsageLogStoreFromClients(clients)
		ingest.JobRuns = db.NewJobRunStoreFromClients(clients)
		ingest.Enrichment = buildMoralisWalletSummaryEnricher(clients)
		ingest.SummaryCache = db.NewRedisWalletSummaryCache(clients.Redis)
		ingest.EntityIndex = db.NewHeuristicEntityAssignmentStoreFromClients(clients)
		ingest.Labeling = db.NewWalletLabelingStoreFromClients(clients)
		ingest.Tracking = db.NewWalletTrackingStateStoreFromClients(clients)
		enrichmentRefresh = WalletEnrichmentRefreshService{
			Enrichment: buildMoralisWalletSummaryEnricher(clients),
		}

		watchlistBootstrap = WatchlistBootstrapService{
			Watchlists: db.NewWatchlistWalletSeedReaderFromClients(clients),
			Tracking:   db.NewWalletTrackingStateStoreFromClients(clients),
			Queue:      db.NewWalletBackfillQueueStoreFromClients(clients),
			Dedup:      db.NewIngestDedupStoreFromClients(clients),
			JobRuns:    db.NewJobRunStoreFromClients(clients),
		}
		seedDiscovery.Queue = db.NewWalletBackfillQueueStoreFromClients(clients)
		seedDiscovery.Tracking = db.NewWalletTrackingStateStoreFromClients(clients)
		seedDiscovery.CuratedSeeds = db.NewPostgresWatchlistWalletSeedReaderFromPool(clients.Postgres)
		seedDiscovery.EntityIndex = db.NewPostgresCuratedEntityIndexStoreFromPool(clients.Postgres)
		seedDiscovery.Dedup = db.NewIngestDedupStoreFromClients(clients)
		seedDiscovery.JobRuns = db.NewJobRunStoreFromClients(clients)
		seedDiscovery.Watchlists = db.NewPostgresWatchlistStoreFromPool(clients.Postgres)
		deliveries = buildAlertDeliveryDispatcher(clients)
		alertDeliveryRetry = AlertDeliveryRetryService{
			Attempts:   db.NewAlertDeliveryStoreFromClients(clients),
			Deliveries: deliveries,
			JobRuns:    db.NewJobRunStoreFromClients(clients),
		}
		alerts = &WalletSignalAlertDispatcher{
			Rules:      db.NewAlertStoreFromClients(clients),
			Events:     db.NewAlertStoreFromClients(clients),
			Deliveries: deliveries,
		}
		clusterScore = ClusterScoreSnapshotService{
			Wallets:  db.NewWalletStoreFromClients(clients),
			Graphs:   db.NewWalletGraphRepositoryFromClients(clients, 2*time.Minute),
			Signals:  db.NewSignalEventStoreFromClients(clients),
			Tracking: db.NewWalletTrackingStateStoreFromClients(clients),
			Labels:   db.NewWalletLabelingStoreFromClients(clients),
			Findings: db.NewFindingStoreFromClients(clients),
			Cache:    db.NewRedisWalletSummaryCache(clients.Redis),
			Alerts:   alerts,
			JobRuns:  db.NewJobRunStoreFromClients(clients),
		}
		shadowExit = ShadowExitSnapshotService{
			Wallets:        db.NewWalletStoreFromClients(clients),
			Candidates:     db.NewShadowExitCandidateReaderFromClients(clients),
			BridgeExchange: db.NewWalletBridgeExchangeEvidenceStoreFromClients(clients),
			TreasuryMM:     db.NewWalletTreasuryMMEvidenceStoreFromClients(clients),
			Signals:        db.NewSignalEventStoreFromClients(clients),
			Tracking:       db.NewWalletTrackingStateStoreFromClients(clients),
			Labels:         db.NewWalletLabelingStoreFromClients(clients),
			Findings:       db.NewFindingStoreFromClients(clients),
			Cache:          db.NewRedisWalletSummaryCache(clients.Redis),
			Alerts:         alerts,
			JobRuns:        db.NewJobRunStoreFromClients(clients),
		}
		firstConnection = FirstConnectionSnapshotService{
			Wallets:       db.NewWalletStoreFromClients(clients),
			Candidates:    db.NewFirstConnectionCandidateReaderFromClients(clients),
			EntryFeatures: db.NewWalletEntryFeaturesStoreFromClients(clients),
			Signals:       db.NewSignalEventStoreFromClients(clients),
			Tracking:      db.NewWalletTrackingStateStoreFromClients(clients),
			Labels:        db.NewWalletLabelingStoreFromClients(clients),
			Findings:      db.NewFindingStoreFromClients(clients),
			Cache:         db.NewRedisWalletSummaryCache(clients.Redis),
			Alerts:        alerts,
			JobRuns:       db.NewJobRunStoreFromClients(clients),
		}
		trackingSubscriptionSync = TrackingSubscriptionSyncService{
			Registry:          db.NewPostgresWalletTrackingRegistryReaderFromPool(clients.Postgres),
			Tracking:          db.NewWalletTrackingStateStoreFromClients(clients),
			JobRuns:           db.NewJobRunStoreFromClients(clients),
			AlchemyReconciler: buildAlchemyWebhookReconcilerFromOS(),
			HeliusReconciler:  buildHeliusWebhookReconcilerFromOS(),
			WebhookBaseURL:    env.AppBaseURL,
		}
		billingSubscriptionSync = BillingSubscriptionSyncService{
			Accounts:        db.NewBillingStoreFromClients(clients),
			AccountStore:    db.NewBillingStoreFromClients(clients),
			Subscriptions:   db.NewBillingSubscriptionStoreFromClients(clients),
			Reconciliations: db.NewBillingSubscriptionReconciliationStoreFromClients(clients),
			StripeClient:    billing.NewHTTPStripeClient(&http.Client{Timeout: 15 * time.Second}),
			StripeConfig: billing.StripeConfig{
				BaseURL:        strings.TrimSpace(os.Getenv("STRIPE_BASE_URL")),
				SecretKey:      strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY")),
				WebhookSecret:  strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET")),
				PublishableKey: strings.TrimSpace(os.Getenv("STRIPE_PUBLISHABLE_KEY")),
				SuccessURL:     strings.TrimSpace(os.Getenv("STRIPE_SUCCESS_URL")),
				CancelURL:      strings.TrimSpace(os.Getenv("STRIPE_CANCEL_URL")),
			},
		}
		exchangeListingRegistrySync.Store = db.NewExchangeListingRegistryStoreFromClients(clients)
		exchangeListingRegistrySync.JobRuns = db.NewJobRunStoreFromClients(clients)
	}

	autoIndexLoop := NewAutoIndexLoopService(seedDiscovery, watchlistBootstrap, ingest)

	output, err := buildWorkerOutputWithAutoIndex(
		appCtx,
		mode,
		env,
		runner,
		ingest,
		enrichmentRefresh,
		seedDiscovery,
		watchlistBootstrap,
		clusterScore,
		shadowExit,
		firstConnection,
		alertDeliveryRetry,
		trackingSubscriptionSync,
		exchangeListingRegistrySync,
		billingSubscriptionSync,
		autoIndexLoop,
	)
	if err != nil {
		log.Fatalf("worker execution failed: %v", err)
	}

	fmt.Println(output)
}

func buildAlertDeliveryDispatcher(clients *db.StorageClients) *WalletSignalDeliveryDispatcher {
	if clients == nil {
		return &WalletSignalDeliveryDispatcher{}
	}

	dispatcher := &WalletSignalDeliveryDispatcher{
		Channels: db.NewAlertDeliveryStoreFromClients(clients),
		Attempts: db.NewAlertDeliveryStoreFromClients(clients),
		Discord: HTTPAlertDiscordWebhookSender{
			Client: http.DefaultClient,
		},
		Telegram: HTTPAlertTelegramSender{
			Client: http.DefaultClient,
		},
		RetryLimit:     alertDeliveryRetryLimitFromEnv(),
		RetryBaseDelay: alertDeliveryRetryBaseDelayFromEnv(),
	}

	if sender := buildSMTPAlertEmailSenderFromOS(); sender != nil {
		dispatcher.Email = sender
	}

	return dispatcher
}

func buildSMTPAlertEmailSenderFromOS() AlertEmailSender {
	host := strings.TrimSpace(os.Getenv("QORVI_ALERT_SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("QORVI_ALERT_SMTP_PORT"))
	from := strings.TrimSpace(os.Getenv("QORVI_ALERT_SMTP_FROM"))
	if host == "" || port == "" || from == "" {
		return nil
	}

	return SMTPAlertEmailSender{
		Addr:     net.JoinHostPort(host, port),
		Host:     host,
		Username: strings.TrimSpace(os.Getenv("QORVI_ALERT_SMTP_USERNAME")),
		Password: strings.TrimSpace(os.Getenv("QORVI_ALERT_SMTP_PASSWORD")),
		From:     from,
	}
}

func buildStartupMessage(env config.WorkerEnv) string {
	return fmt.Sprintf(
		"Qorvi workers ready (env=%s, postgres=%s, redis=%s)",
		env.NodeEnv,
		env.PostgresURL,
		env.RedisURL,
	)
}

func rawPayloadRoot() string {
	root := strings.TrimSpace(os.Getenv("QORVI_RAW_PAYLOAD_ROOT"))
	if root != "" {
		return root
	}

	return ".qorvi/raw-payloads"
}

func buildMoralisWalletSummaryEnricher(clients *db.StorageClients) WalletSummaryEnrichmentRefresher {
	if clients == nil || clients.Redis == nil {
		return nil
	}

	apiKey := strings.TrimSpace(os.Getenv("MORALIS_API_KEY"))
	if len(apiKey) < 8 {
		return nil
	}

	baseURL := strings.TrimSpace(os.Getenv("MORALIS_BASE_URL"))
	if baseURL == "" {
		baseURL = providers.DefaultMoralisBaseURL
	}

	return providers.NewMoralisWalletSummaryEnricher(
		providers.NewMoralisClient(providers.ProviderCredentials{
			Provider: providers.ProviderMoralis,
			APIKey:   apiKey,
			BaseURL:  baseURL,
		}, nil),
		providers.NewRedisMoralisWalletEnrichmentCache(clients.Redis),
		db.NewWalletEnrichmentSnapshotStoreFromClients(clients),
		db.NewRedisWalletSummaryCache(clients.Redis),
		15*time.Minute,
	)
}

func buildAlchemyWebhookReconcilerFromOS() providerAddressReconciler {
	authToken := strings.TrimSpace(os.Getenv("ALCHEMY_NOTIFY_AUTH_TOKEN"))
	if authToken == "" {
		return nil
	}
	baseURL := strings.TrimSpace(os.Getenv("ALCHEMY_NOTIFY_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://dashboard.alchemy.com"
	}
	network := strings.TrimSpace(os.Getenv("ALCHEMY_ADDRESS_ACTIVITY_NETWORK"))
	return providers.NewAlchemyWebhookClient(baseURL, authToken, network, &http.Client{Timeout: 15 * time.Second})
}

func buildHeliusWebhookReconcilerFromOS() providerAddressReconciler {
	apiKey := strings.TrimSpace(os.Getenv("HELIUS_API_KEY"))
	if apiKey == "" {
		return nil
	}
	baseURL := strings.TrimSpace(os.Getenv("HELIUS_DATA_API_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api-mainnet.helius-rpc.com/v0"
	}
	authHeader := strings.TrimSpace(os.Getenv("QORVI_PROVIDER_WEBHOOK_AUTH_HEADER"))
	return providers.NewHeliusWebhookClient(baseURL, apiKey, authHeader, &http.Client{Timeout: 15 * time.Second})
}

func workerEnvFromOS() config.WorkerEnv {
	return config.WorkerEnv{
		NodeEnv:       strings.TrimSpace(os.Getenv("NODE_ENV")),
		PostgresURL:   strings.TrimSpace(os.Getenv("POSTGRES_URL")),
		Neo4jURL:      strings.TrimSpace(os.Getenv("NEO4J_URL")),
		Neo4jUsername: strings.TrimSpace(os.Getenv("NEO4J_USERNAME")),
		Neo4jPassword: strings.TrimSpace(os.Getenv("NEO4J_PASSWORD")),
		RedisURL:      strings.TrimSpace(os.Getenv("REDIS_URL")),
	}
}

func requiresProviderRegistry(mode string) bool {
	return mode == workerModeHistoricalBackfillIngest ||
		mode == workerModeWalletBackfillDrain ||
		mode == workerModeWalletBackfillDrainPriority ||
		mode == workerModeWalletBackfillDrainBatch ||
		mode == workerModeWalletBackfillDrainLoop ||
		mode == workerModeAutoIndexLoop ||
		mode == workerModeMoralisEnrichmentRefresh ||
		mode == workerModeMobulaSmartMoneyEnqueue ||
		mode == workerModeSeedDiscoveryEnqueue ||
		mode == workerModeSeedDiscoverySeedWatchlist
}

func requiresWorkerStorage(mode string) bool {
	return mode == workerModeHistoricalBackfillIngest ||
		mode == workerModeWalletBackfillDrain ||
		mode == workerModeWalletBackfillDrainPriority ||
		mode == workerModeWalletBackfillDrainBatch ||
		mode == workerModeWalletBackfillDrainLoop ||
		mode == workerModeAutoIndexLoop ||
		mode == workerModeMoralisEnrichmentRefresh ||
		mode == workerModeMobulaSmartMoneyEnqueue ||
		mode == workerModeAdminCuratedWalletImport ||
		mode == workerModeCuratedWalletSeedEnqueue ||
		mode == workerModeSeedDiscoveryEnqueue ||
		mode == workerModeSeedDiscoverySeedWatchlist ||
		mode == workerModeWatchlistBootstrapEnqueue ||
		mode == workerModeClusterScoreSnapshot ||
		mode == workerModeShadowExitSnapshot ||
		mode == workerModeFirstConnectionSnapshot ||
		mode == workerModeAlertDeliveryRetryBatch ||
		mode == workerModeWalletTrackingSubscriptionSync ||
		mode == workerModeExchangeListingRegistrySync ||
		mode == workerModeBillingSubscriptionSync
}
