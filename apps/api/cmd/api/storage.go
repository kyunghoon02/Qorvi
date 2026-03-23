package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/config"
	"github.com/whalegraph/whalegraph/apps/api/internal/repository"
	"github.com/whalegraph/whalegraph/apps/api/internal/server"
	"github.com/whalegraph/whalegraph/apps/api/internal/service"
	"github.com/whalegraph/whalegraph/packages/billing"
	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/providers"
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
			db.NewPostgresWatchlistStoreFromPool(clients.Postgres),
			db.NewPostgresAuditLogStoreFromPool(clients.Postgres),
			db.NewCuratedEntityIndexStoreFromClients(clients),
		),
	)
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

	return service.NewSearchServiceWithBackfillQueue(
		wallets,
		db.NewWalletBackfillQueueStoreFromClients(clients),
	)
}

func buildWebhookIngestService(clients *db.StorageClients) server.WebhookIngestService {
	if clients == nil {
		return nil
	}

	return server.NewWebhookIngestService(
		db.NewWalletStoreFromClients(clients),
		db.NewHeuristicEntityAssignmentStoreFromClients(clients),
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
	)
}

func apiRawPayloadRoot() string {
	root := strings.TrimSpace(os.Getenv("WHALEGRAPH_RAW_PAYLOAD_ROOT"))
	if root != "" {
		return root
	}

	return ".whalegraph/raw-payloads"
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
