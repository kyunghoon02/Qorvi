package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/qorvi/qorvi/apps/api/internal/config"
	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/apps/api/internal/server"
	"github.com/qorvi/qorvi/apps/api/internal/service"
	"github.com/qorvi/qorvi/packages/billing"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/providers"
)

func openStorageClients(ctx context.Context, cfg config.Config) (*db.StorageClients, error) {
	return db.OpenStorageClients(ctx, cfg.StorageHandles)
}

func buildWalletSummaryService(clients *db.StorageClients, cacheTTL time.Duration) *service.WalletSummaryService {
	if clients == nil {
		return service.NewWalletSummaryService(repository.NewQueryBackedWalletSummaryRepository(nil), nil)
	}

	return service.NewWalletSummaryService(
		repository.NewQueryBackedWalletSummaryRepository(
			db.NewWalletSummaryRepositoryFromClients(clients, cacheTTL),
		),
		buildMoralisWalletSummaryEnricher(clients),
	)
}

func buildWalletGraphService(clients *db.StorageClients, cacheTTL time.Duration) *service.WalletGraphService {
	if clients == nil {
		return service.NewWalletGraphService(repository.NewQueryBackedWalletGraphRepository(nil))
	}

	return service.NewWalletGraphService(
		repository.NewQueryBackedWalletGraphRepository(
			db.NewWalletGraphRepositoryFromClients(clients, cacheTTL),
		),
	)
}

func buildFindingsFeedService(clients *db.StorageClients) *service.FindingsFeedService {
	if clients == nil {
		return service.NewFindingsFeedService(repository.NewQueryBackedFindingsRepository(nil))
	}

	return service.NewFindingsFeedService(
		repository.NewQueryBackedFindingsRepository(
			db.NewFindingStoreFromClients(clients),
		),
	)
}

func buildDiscoverService(clients *db.StorageClients) *service.DiscoverService {
	if clients == nil {
		return service.NewDiscoverService(nil)
	}

	return service.NewDiscoverService(
		db.NewPostgresWatchlistWalletSeedReaderFromPool(clients.Postgres),
		db.NewDomesticPrelistingStoreFromClients(clients),
	)
}

func buildWalletBriefService(
	clients *db.StorageClients,
	wallets *service.WalletSummaryService,
) *service.WalletBriefService {
	if clients == nil {
		return service.NewWalletBriefService(
			repository.NewQueryBackedWalletSummaryRepository(nil),
			nil,
			repository.NewQueryBackedFindingsRepository(nil),
			repository.NewQueryBackedWalletEntryFeaturesRepository(nil),
		)
	}

	var enricher service.WalletSummaryEnricher
	if wallets != nil {
		enricher = buildMoralisWalletSummaryEnricher(clients)
	}

	return service.NewWalletBriefService(
		repository.NewQueryBackedWalletSummaryRepository(
			db.NewWalletSummaryRepositoryFromClients(clients, 5*time.Minute),
		),
		enricher,
		repository.NewQueryBackedFindingsRepository(
			db.NewFindingStoreFromClients(clients),
		),
		repository.NewQueryBackedWalletEntryFeaturesRepository(
			db.NewWalletEntryFeaturesStoreFromClients(clients),
		),
	)
}

func buildEntityInterpretationService(clients *db.StorageClients) *service.EntityInterpretationService {
	if clients == nil {
		return service.NewEntityInterpretationService(repository.NewQueryBackedEntityInterpretationRepository(nil))
	}

	return service.NewEntityInterpretationService(
		repository.NewQueryBackedEntityInterpretationRepository(
			db.NewEntityInterpretationReaderFromClients(clients),
		),
	)
}

func buildAnalystToolsService(
	wallets *service.WalletSummaryService,
	walletBriefs *service.WalletBriefService,
	graphs *service.WalletGraphService,
) *service.AnalystToolsService {
	return service.NewAnalystToolsService(wallets, walletBriefs, graphs)
}

func buildAnalystFindingDrilldownService(
	clients *db.StorageClients,
	wallets *service.WalletSummaryService,
) *service.AnalystFindingDrilldownService {
	if clients == nil {
		return service.NewAnalystFindingDrilldownService(
			repository.NewQueryBackedFindingsRepository(nil),
			wallets,
			repository.NewQueryBackedWalletEntryFeaturesRepository(nil),
		)
	}
	return service.NewAnalystFindingDrilldownService(
		repository.NewQueryBackedFindingsRepository(
			db.NewFindingStoreFromClients(clients),
		),
		wallets,
		repository.NewQueryBackedWalletEntryFeaturesRepository(
			db.NewWalletEntryFeaturesStoreFromClients(clients),
		),
	)
}

func buildAnalystFindingExplanationService(
	clients *db.StorageClients,
	walletBriefs *service.WalletBriefService,
) *service.AnalystFindingExplanationService {
	findingsRepo := repository.NewQueryBackedFindingsRepository(nil)
	var explanationStore *db.PostgresAIExplanationStore
	var auditStore *db.PostgresAuditLogStore
	if clients != nil {
		findingsRepo = repository.NewQueryBackedFindingsRepository(
			db.NewFindingStoreFromClients(clients),
		)
		explanationStore = db.NewAIExplanationStoreFromClients(clients)
		auditStore = db.NewPostgresAuditLogStoreFromPool(clients.Postgres)
	}

	return service.NewAnalystFindingExplanationService(
		findingsRepo,
		walletBriefs,
		explanationStore,
		auditStore,
		service.NewOpenAIChatFindingExplanationClient(
			strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
			strings.TrimSpace(os.Getenv("OPENAI_MODEL")),
		),
	)
}

func buildInteractiveAnalystService(
	walletBriefs *service.WalletBriefService,
	analystTools *service.AnalystToolsService,
	analystFindings *service.AnalystFindingDrilldownService,
	entities *service.EntityInterpretationService,
) *service.InteractiveAnalystService {
	return service.NewInteractiveAnalystService(walletBriefs, analystTools, analystFindings, entities)
}

func buildClusterDetailService(clients *db.StorageClients) *service.ClusterDetailService {
	if clients == nil {
		return service.NewClusterDetailService(repository.NewQueryBackedClusterDetailRepository(nil))
	}

	return service.NewClusterDetailService(
		repository.NewQueryBackedClusterDetailRepository(
			db.NewClusterDetailRepositoryFromClients(clients),
		),
	)
}

func buildShadowExitFeedService(clients *db.StorageClients) *service.ShadowExitFeedService {
	if clients == nil {
		return service.NewShadowExitFeedService(repository.NewQueryBackedShadowExitFeedRepository(nil))
	}

	return service.NewShadowExitFeedService(
		repository.NewQueryBackedShadowExitFeedRepository(
			db.NewShadowExitFeedRepositoryFromClients(clients),
		),
	)
}

func buildFirstConnectionFeedService(clients *db.StorageClients) *service.FirstConnectionFeedService {
	if clients == nil {
		return service.NewFirstConnectionFeedService(repository.NewQueryBackedFirstConnectionFeedRepository(nil))
	}

	return service.NewFirstConnectionFeedService(
		repository.NewQueryBackedFirstConnectionFeedRepository(
			db.NewCachedFirstConnectionFeedReaderFromClients(clients, 2*time.Minute),
		),
	)
}

func buildAlertRuleService(clients *db.StorageClients) *service.AlertRuleService {
	if clients == nil {
		return service.NewAlertRuleService(repository.NewInMemoryAlertRuleRepository())
	}

	return service.NewAlertRuleService(
		repository.NewPostgresAlertRuleRepository(
			db.NewPostgresAlertStoreFromPool(clients.Postgres),
		),
	)
}

func buildAlertDeliveryService(clients *db.StorageClients) *service.AlertDeliveryService {
	if clients == nil {
		return service.NewAlertDeliveryService(repository.NewInMemoryAlertDeliveryRepository())
	}

	return service.NewAlertDeliveryService(
		repository.NewPostgresAlertDeliveryRepository(
			db.NewAlertDeliveryStoreFromClients(clients),
		),
	)
}

func buildWatchlistService(clients *db.StorageClients) *service.WatchlistService {
	if clients == nil {
		return service.NewWatchlistService(repository.NewInMemoryWatchlistRepository())
	}

	return service.NewWatchlistService(
		repository.NewPostgresWatchlistRepository(
			db.NewPostgresWatchlistStoreFromPool(clients.Postgres),
		),
	)
}

func buildAdminConsoleService(clients *db.StorageClients) *service.AdminConsoleService {
	if clients == nil {
		return service.NewAdminConsoleService(repository.NewInMemoryAdminConsoleRepository())
	}

	return service.NewAdminConsoleService(
		repository.NewPostgresAdminConsoleRepository(
			db.NewPostgresAdminConsoleStoreFromPool(clients.Postgres),
			db.NewRedisAdminQueueStatsStore(clients.Redis),
			db.NewPostgresWatchlistStoreFromPool(clients.Postgres),
			db.NewPostgresAuditLogStoreFromPool(clients.Postgres),
			db.NewCuratedEntityIndexStoreFromClients(clients),
			db.NewDomesticPrelistingStoreFromClients(clients),
		),
	)
}

func buildAdminBacktestOpsService() *service.AdminBacktestOpsService {
	manifestPath := strings.TrimSpace(os.Getenv("QORVI_BACKTEST_MANIFEST_PATH"))
	if manifestPath == "" {
		manifestPath = "packages/intelligence/test/backtest-manifest.json"
	}
	presetPath := strings.TrimSpace(os.Getenv("QORVI_DUNE_QUERY_PRESET_PATH"))
	if presetPath == "" {
		presetPath = "queries/dune/backtest/query-presets.json"
	}
	candidateExportPath := strings.TrimSpace(os.Getenv("QORVI_DUNE_CANDIDATE_EXPORT_PATH"))
	if candidateExportPath == "" {
		candidateExportPath = "packages/intelligence/test/dune-backtest-candidates.json"
	}
	svc := service.NewAdminBacktestOpsService(
		manifestPath,
		presetPath,
		candidateExportPath,
	)
	svcBaseURL := strings.TrimSpace(os.Getenv("DUNE_API_BASE_URL"))
	if svcBaseURL != "" {
		svcBaseURL = strings.TrimRight(svcBaseURL, "/")
	}
	if svcBaseURL == "" {
		svcBaseURL = "https://api.dune.com/api/v1"
	}
	svc.ConfigureDuneFetch(
		strings.TrimSpace(os.Getenv("DUNE_API_KEY")),
		svcBaseURL,
		&http.Client{Timeout: 20 * time.Second},
	)
	return svc
}

func buildBillingService(clients *db.StorageClients) *service.BillingService {
	repo := repository.BillingRepository(repository.NewInMemoryBillingRepository())
	options := make([]service.BillingServiceOption, 0, 4)
	if clients != nil {
		repo = repository.NewPostgresBillingRepository(
			db.NewBillingStoreFromClients(clients),
		)
		options = append(options,
			service.WithBillingCheckoutSessionStore(db.NewBillingCheckoutSessionStoreFromClients(clients)),
			service.WithBillingSubscriptionStore(db.NewBillingSubscriptionStoreFromClients(clients)),
			service.WithBillingSubscriptionReconciliationStore(db.NewBillingSubscriptionReconciliationStoreFromClients(clients)),
		)
	}

	config := billing.StripeConfig{
		BaseURL:        strings.TrimSpace(os.Getenv("STRIPE_BASE_URL")),
		SecretKey:      strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY")),
		WebhookSecret:  strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET")),
		PublishableKey: strings.TrimSpace(os.Getenv("STRIPE_PUBLISHABLE_KEY")),
		SuccessURL:     strings.TrimSpace(os.Getenv("STRIPE_SUCCESS_URL")),
		CancelURL:      strings.TrimSpace(os.Getenv("STRIPE_CANCEL_URL")),
	}

	if strings.TrimSpace(config.SecretKey) != "" {
		options = append(options, service.WithStripeClient(billing.NewHTTPStripeClient(&http.Client{
			Timeout: 15 * time.Second,
		})))
	}

	return service.NewBillingService(repo, config, options...)
}

func buildAccountService(billingService *service.BillingService) *service.AccountService {
	return service.NewAccountService(billingService)
}

func buildSearchService(
	clients *db.StorageClients,
	wallets *service.WalletSummaryService,
) *service.SearchService {
	if wallets == nil {
		wallets = buildWalletSummaryService(clients, 5*time.Minute)
	}

	if clients == nil {
		return service.NewSearchService(wallets)
	}

	return service.NewSearchServiceWithBackfillQueueAndTracking(
		wallets,
		db.NewWalletBackfillQueueStoreFromClients(clients),
		db.NewWalletTrackingStateStoreFromClients(clients),
	)
}

func buildWebhookIngestService(clients *db.StorageClients) server.WebhookIngestService {
	if clients == nil {
		return nil
	}

	return server.NewWebhookIngestService(
		db.NewWalletStoreFromClients(clients),
		db.NewHeuristicEntityAssignmentStoreFromClients(clients),
		db.NewWalletLabelingStoreFromClients(clients),
		db.NewNormalizedTransactionStoreFromClients(clients),
		db.NewWalletDailyStatsStoreFromClients(clients),
		db.NewTransactionGraphMaterializerFromClients(clients),
		db.NewRedisWalletGraphCache(clients.Redis),
		db.NewPostgresWalletGraphSnapshotStoreFromPool(clients.Postgres),
		db.NewRedisWalletSummaryCache(clients.Redis),
		db.NewIngestDedupStoreFromClients(clients),
		db.NewFilesystemRawPayloadStore(apiRawPayloadRoot()),
		db.NewProviderUsageLogStoreFromClients(clients),
		db.NewJobRunStoreFromClients(clients),
		db.NewWalletTrackingStateStoreFromClients(clients),
	)
}

func apiRawPayloadRoot() string {
	root := strings.TrimSpace(os.Getenv("QORVI_RAW_PAYLOAD_ROOT"))
	if root != "" {
		return root
	}

	return ".qorvi/raw-payloads"
}

func buildMoralisWalletSummaryEnricher(clients *db.StorageClients) service.WalletSummaryEnricher {
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
