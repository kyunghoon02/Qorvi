package main

import (
	"context"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/config"
	"github.com/whalegraph/whalegraph/apps/api/internal/repository"
	"github.com/whalegraph/whalegraph/apps/api/internal/service"
	"github.com/whalegraph/whalegraph/packages/db"
)

func openStorageClients(ctx context.Context, cfg config.Config) (*db.StorageClients, error) {
	return db.OpenStorageClients(ctx, cfg.StorageHandles)
}

func buildWalletSummaryService(clients *db.StorageClients, cacheTTL time.Duration) *service.WalletSummaryService {
	if clients == nil {
		return service.NewWalletSummaryService(repository.NewQueryBackedWalletSummaryRepository(nil))
	}

	return service.NewWalletSummaryService(
		repository.NewQueryBackedWalletSummaryRepository(
			db.NewWalletSummaryRepositoryFromClients(clients, cacheTTL),
		),
	)
}

func buildWalletGraphService(clients *db.StorageClients, cacheTTL time.Duration) *service.WalletGraphService {
	if clients == nil {
		return service.NewWalletGraphService(repository.NewQueryBackedWalletGraphRepository(nil))
	}

	return service.NewWalletGraphService(
		repository.NewQueryBackedWalletGraphRepository(
			db.NewWalletGraphRepositoryFromClients(clients),
		),
	)
}
