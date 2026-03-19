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
		NewRedisWalletSummaryCache(clients.Redis),
		cacheTTL,
	)
}

func NewWalletGraphRepositoryFromClients(clients *StorageClients) *WalletGraphRepository {
	if clients == nil {
		return nil
	}

	return NewWalletGraphRepository(
		NewNeo4jWalletGraphReader(clients.Neo4j, "neo4j"),
	)
}

func NewWalletGraphReaderFromClients(clients *StorageClients) *Neo4jWalletGraphReader {
	if clients == nil {
		return nil
	}

	return NewNeo4jWalletGraphReader(clients.Neo4j, "neo4j")
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
