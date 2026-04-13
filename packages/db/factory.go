package db

import (
	"context"
	"time"
)

func NewWalletSummaryRepositoryFromClients(clients *StorageClients, cacheTTL time.Duration) *WalletSummaryRepository {
	if clients == nil {
		return nil
	}

	return NewWalletSummaryRepository(
		NewPostgresWalletIdentityReaderFromPool(clients.Postgres),
		NewPostgresWalletStatsReaderFromPool(clients.Postgres),
		NewNeo4jWalletGraphSignalReader(clients.Neo4j, "neo4j"),
		NewPostgresWalletLabelingStoreWithInvalidation(
			NewPostgresWalletStoreFromPool(clients.Postgres),
			clients.Postgres,
			clients.Postgres,
			NewRedisWalletSummaryCache(clients.Redis),
			NewRedisWalletGraphCache(clients.Redis),
			NewPostgresWalletGraphSnapshotStoreFromPool(clients.Postgres),
		),
		NewWalletEnrichmentSnapshotStoreFromPool(clients.Postgres),
		NewPostgresClusterScoreSnapshotReaderFromPool(clients.Postgres),
		NewPostgresShadowExitSnapshotReaderFromPool(clients.Postgres),
		NewPostgresFirstConnectionSnapshotReaderFromPool(clients.Postgres),
		NewPostgresWalletLatestSignalsReaderFromPool(clients.Postgres),
		NewRedisWalletSummaryCache(clients.Redis),
		cacheTTL,
	)
}

func NewWalletGraphRepositoryFromClients(clients *StorageClients, cacheTTL time.Duration) *WalletGraphRepository {
	if clients == nil {
		return nil
	}

	return NewWalletGraphRepository(
		NewCachedWalletGraphReader(
			NewLabeledWalletGraphReader(
				NewEntityLinkedWalletGraphReader(
					NewEnrichedWalletGraphReader(
						NewNeo4jWalletGraphReader(clients.Neo4j, "neo4j"),
						NewPostgresWalletGraphEdgeFlowReader(clients.Postgres),
					),
					NewPostgresWalletGraphEntityLinkReader(clients.Postgres),
				),
				NewPostgresWalletLabelingStoreWithInvalidation(
					NewPostgresWalletStoreFromPool(clients.Postgres),
					clients.Postgres,
					clients.Postgres,
					NewRedisWalletSummaryCache(clients.Redis),
					NewRedisWalletGraphCache(clients.Redis),
					NewPostgresWalletGraphSnapshotStoreFromPool(clients.Postgres),
				),
			),
			NewRedisWalletGraphCache(clients.Redis),
			NewPostgresWalletGraphSnapshotStoreFromPool(clients.Postgres),
			cacheTTL,
		),
	)
}

func NewShadowExitFeedRepositoryFromClients(clients *StorageClients) *PostgresShadowExitFeedReader {
	if clients == nil {
		return nil
	}

	return NewPostgresShadowExitFeedReaderFromPool(clients.Postgres)
}

func NewFirstConnectionFeedReaderFromClients(clients *StorageClients) *PostgresFirstConnectionFeedReader {
	if clients == nil {
		return nil
	}

	return NewPostgresFirstConnectionFeedReaderFromPool(clients.Postgres)
}

func NewCachedFirstConnectionFeedReaderFromClients(
	clients *StorageClients,
	cacheTTL time.Duration,
) *CachedFirstConnectionFeedReader {
	if clients == nil {
		return nil
	}

	return NewCachedFirstConnectionFeedReader(
		NewPostgresFirstConnectionFeedReaderFromPool(clients.Postgres),
		NewRedisFirstConnectionFeedCache(clients.Redis),
		cacheTTL,
	)
}

func NewWalletGraphReaderFromClients(clients *StorageClients) *Neo4jWalletGraphReader {
	if clients == nil {
		return nil
	}

	return NewNeo4jWalletGraphReader(clients.Neo4j, "neo4j")
}

func NewTransactionGraphMaterializerFromClients(clients *StorageClients) *Neo4jTransactionGraphMaterializer {
	if clients == nil {
		return nil
	}

	return NewNeo4jTransactionGraphMaterializer(clients.Neo4j, "neo4j")
}

func NewWalletStoreFromClients(clients *StorageClients) *PostgresWalletStore {
	if clients == nil {
		return nil
	}

	return NewPostgresWalletStoreFromPool(clients.Postgres)
}

func NewWalletLabelingStoreFromClients(clients *StorageClients) *PostgresWalletLabelingStore {
	if clients == nil {
		return nil
	}

	return NewPostgresWalletLabelingStoreWithInvalidation(
		NewPostgresWalletStoreFromPool(clients.Postgres),
		clients.Postgres,
		clients.Postgres,
		NewRedisWalletSummaryCache(clients.Redis),
		NewRedisWalletGraphCache(clients.Redis),
		NewPostgresWalletGraphSnapshotStoreFromPool(clients.Postgres),
	)
}

func NewWalletTrackingStateStoreFromClients(clients *StorageClients) *PostgresWalletTrackingStateStore {
	if clients == nil {
		return nil
	}

	return NewPostgresWalletTrackingStateStore(
		NewPostgresWalletStoreFromPool(clients.Postgres),
		clients.Postgres,
	)
}

func NewHeuristicEntityAssignmentStoreFromClients(
	clients *StorageClients,
) *PostgresHeuristicEntityAssignmentStore {
	if clients == nil {
		return nil
	}

	return NewPostgresHeuristicEntityAssignmentStoreWithGraphInvalidation(
		clients.Postgres,
		NewRedisWalletGraphCache(clients.Redis),
		NewPostgresWalletGraphSnapshotStoreFromPool(clients.Postgres),
	)
}

func NewCuratedEntityIndexStoreFromClients(
	clients *StorageClients,
) *PostgresCuratedEntityIndexStore {
	if clients == nil {
		return nil
	}

	return NewPostgresCuratedEntityIndexStoreWithGraphInvalidation(
		clients.Postgres,
		clients.Postgres,
		NewRedisWalletGraphCache(clients.Redis),
		NewPostgresWalletGraphSnapshotStoreFromPool(clients.Postgres),
	)
}

func NewNormalizedTransactionStoreFromClients(clients *StorageClients) *PostgresNormalizedTransactionStore {
	if clients == nil {
		return nil
	}

	return NewPostgresNormalizedTransactionStoreFromPool(clients.Postgres)
}

func NewWalletDailyStatsStoreFromClients(clients *StorageClients) *PostgresWalletDailyStatsStore {
	if clients == nil {
		return nil
	}

	return NewPostgresWalletDailyStatsStoreFromPool(clients.Postgres)
}

func NewWalletEnrichmentSnapshotStoreFromClients(clients *StorageClients) *PostgresWalletEnrichmentSnapshotStore {
	if clients == nil {
		return nil
	}

	return NewWalletEnrichmentSnapshotStoreFromPool(clients.Postgres)
}

func NewProviderUsageLogStoreFromClients(clients *StorageClients) *PostgresProviderUsageLogStore {
	if clients == nil {
		return nil
	}

	return NewPostgresProviderUsageLogStoreFromPool(clients.Postgres)
}

func NewSignalEventStoreFromClients(clients *StorageClients) *PostgresSignalEventStore {
	if clients == nil {
		return nil
	}

	return NewPostgresSignalEventStoreFromPool(clients.Postgres)
}

func NewFindingStoreFromClients(clients *StorageClients) *PostgresFindingStore {
	if clients == nil {
		return nil
	}

	return NewPostgresFindingStoreFromPool(clients.Postgres)
}

func NewWalletBridgeExchangeEvidenceStoreFromClients(clients *StorageClients) *PostgresWalletBridgeExchangeEvidenceStore {
	if clients == nil {
		return nil
	}

	return NewPostgresWalletBridgeExchangeEvidenceStoreFromPool(clients.Postgres)
}

func NewWalletTreasuryMMEvidenceStoreFromClients(clients *StorageClients) *PostgresWalletTreasuryMMEvidenceStore {
	if clients == nil {
		return nil
	}

	return NewPostgresWalletTreasuryMMEvidenceStoreFromPool(
		clients.Postgres,
		NewWalletLabelingStoreFromClients(clients),
	)
}

func NewWalletEntryFeaturesStoreFromClients(clients *StorageClients) *PostgresWalletEntryFeaturesStore {
	if clients == nil {
		return nil
	}

	return NewPostgresWalletEntryFeaturesStoreFromPool(clients.Postgres)
}

func NewAIExplanationStoreFromClients(clients *StorageClients) *PostgresAIExplanationStore {
	if clients == nil {
		return nil
	}

	return NewPostgresAIExplanationStoreFromPool(clients.Postgres)
}

func NewEntityInterpretationReaderFromClients(clients *StorageClients) *PostgresEntityInterpretationReader {
	if clients == nil {
		return nil
	}

	return NewPostgresEntityInterpretationReader(
		clients.Postgres,
		NewWalletLabelingStoreFromClients(clients),
	)
}

func NewShadowExitCandidateReaderFromClients(clients *StorageClients) *PostgresShadowExitCandidateReader {
	if clients == nil {
		return nil
	}

	return NewPostgresShadowExitCandidateReaderFromPool(clients.Postgres)
}

func NewFirstConnectionCandidateReaderFromClients(clients *StorageClients) *PostgresFirstConnectionCandidateReader {
	if clients == nil {
		return nil
	}

	return NewPostgresFirstConnectionCandidateReaderFromPool(clients.Postgres)
}

func NewJobRunStoreFromClients(clients *StorageClients) *PostgresJobRunStore {
	if clients == nil {
		return nil
	}

	return NewPostgresJobRunStoreFromPool(clients.Postgres)
}

func NewIngestDedupStoreFromClients(clients *StorageClients) *RedisIngestDedupStore {
	if clients == nil {
		return nil
	}

	return NewRedisIngestDedupStore(clients.Redis)
}

func NewWalletBackfillQueueStoreFromClients(clients *StorageClients) *RedisWalletBackfillQueueStore {
	if clients == nil {
		return nil
	}

	return NewRedisWalletBackfillQueueStore(clients.Redis)
}

func NewWatchlistWalletSeedReaderFromClients(clients *StorageClients) *PostgresWatchlistWalletSeedReader {
	if clients == nil {
		return nil
	}

	return NewPostgresWatchlistWalletSeedReaderFromPool(clients.Postgres)
}

func NewAutoDiscoverWalletReaderFromClients(clients *StorageClients) *PostgresAutoDiscoverWalletReader {
	if clients == nil {
		return nil
	}

	return NewPostgresAutoDiscoverWalletReaderFromPool(clients.Postgres)
}

func NewAlertStoreFromClients(clients *StorageClients) *PostgresAlertStore {
	if clients == nil {
		return nil
	}

	return NewPostgresAlertStoreFromPool(clients.Postgres)
}

func NewAlertDeliveryStoreFromClients(clients *StorageClients) *PostgresAlertDeliveryStore {
	if clients == nil {
		return nil
	}

	return NewPostgresAlertDeliveryStoreFromPool(clients.Postgres)
}

func NewBillingStoreFromClients(clients *StorageClients) *PostgresBillingStore {
	if clients == nil {
		return nil
	}

	return NewPostgresBillingStoreFromPool(clients.Postgres)
}

func NewBillingCheckoutSessionStoreFromClients(clients *StorageClients) *PostgresBillingCheckoutSessionStore {
	if clients == nil {
		return nil
	}

	return NewPostgresBillingCheckoutSessionStoreFromPool(clients.Postgres)
}

func NewBillingSubscriptionStoreFromClients(clients *StorageClients) *PostgresBillingSubscriptionStore {
	if clients == nil {
		return nil
	}

	return NewPostgresBillingSubscriptionStoreFromPool(clients.Postgres)
}

func NewBillingSubscriptionReconciliationStoreFromClients(clients *StorageClients) *PostgresBillingSubscriptionReconciliationStore {
	if clients == nil {
		return nil
	}

	return NewPostgresBillingSubscriptionReconciliationStoreFromPool(clients.Postgres)
}

func NewWalletSummaryRepositoryFromHandles(
	ctx context.Context,
	handles Handles,
	cacheTTL time.Duration,
) (*WalletSummaryRepository, *StorageClients, error) {
	clients, err := OpenStorageClients(ctx, handles)
	if err != nil {
		return nil, nil, err
	}

	return NewWalletSummaryRepositoryFromClients(clients, cacheTTL), clients, nil
}
