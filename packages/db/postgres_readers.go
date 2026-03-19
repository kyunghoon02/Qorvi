package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type postgresQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type PostgresWalletIdentityReader struct {
	Querier postgresQuerier
}

type PostgresWalletStatsReader struct {
	Querier postgresQuerier
}

func NewPostgresWalletIdentityReader(querier postgresQuerier) *PostgresWalletIdentityReader {
	return &PostgresWalletIdentityReader{Querier: querier}
}

func NewPostgresWalletStatsReader(querier postgresQuerier) *PostgresWalletStatsReader {
	return &PostgresWalletStatsReader{Querier: querier}
}

func (r *PostgresWalletIdentityReader) ReadWalletIdentity(
	ctx context.Context,
	plan WalletSummaryQueryPlan,
) (WalletSummaryIdentity, error) {
	if r == nil || r.Querier == nil {
		return WalletSummaryIdentity{}, fmt.Errorf("postgres identity reader is nil")
	}

	var identity WalletSummaryIdentity
	if err := r.Querier.QueryRow(ctx, plan.IdentitySQL, plan.IdentityArgs...).Scan(
		&identity.WalletID,
		&identity.Chain,
		&identity.Address,
		&identity.DisplayName,
		&identity.EntityKey,
		&identity.CreatedAt,
		&identity.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WalletSummaryIdentity{}, ErrWalletSummaryNotFound
		}

		return WalletSummaryIdentity{}, fmt.Errorf("scan wallet identity: %w", err)
	}

	return identity, nil
}

func (r *PostgresWalletStatsReader) ReadWalletStats(
	ctx context.Context,
	plan WalletSummaryQueryPlan,
) (WalletSummaryStats, error) {
	if r == nil || r.Querier == nil {
		return WalletSummaryStats{}, fmt.Errorf("postgres stats reader is nil")
	}

	var (
		walletID         string
		stats            WalletSummaryStats
		latestActivityAt sql.NullTime
	)

	if err := r.Querier.QueryRow(ctx, plan.StatsSQL, plan.StatsArgs...).Scan(
		&walletID,
		&stats.AsOfDate,
		&stats.TransactionCount,
		&stats.CounterpartyCount,
		&latestActivityAt,
		&stats.IncomingTxCount,
		&stats.OutgoingTxCount,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WalletSummaryStats{}, ErrWalletSummaryNotFound
		}

		return WalletSummaryStats{}, fmt.Errorf("scan wallet stats: %w", err)
	}

	if latestActivityAt.Valid {
		value := latestActivityAt.Time
		stats.LatestActivityAt = &value
	}

	return stats, nil
}

func NewPostgresWalletIdentityReaderFromPool(pool postgresQuerier) *PostgresWalletIdentityReader {
	return NewPostgresWalletIdentityReader(pool)
}

func NewPostgresWalletStatsReaderFromPool(pool postgresQuerier) *PostgresWalletStatsReader {
	return NewPostgresWalletStatsReader(pool)
}

func BuildWalletSummaryFromPostgres(
	identity WalletSummaryIdentity,
	stats WalletSummaryStats,
) domain.WalletSummary {
	clusterID := ""
	return domain.WalletSummary{
		Chain:            identity.Chain,
		Address:          identity.Address,
		DisplayName:      identity.DisplayName,
		ClusterID:        &clusterID,
		Counterparties:   int(stats.CounterpartyCount),
		LatestActivityAt: statsTime(stats),
		Tags:             []string{"wallet", "postgres"},
		Scores:           nil,
	}
}

func statsTime(stats WalletSummaryStats) string {
	if stats.LatestActivityAt == nil {
		return ""
	}
	return stats.LatestActivityAt.UTC().Format("2006-01-02T15:04:05Z07:00")
}
