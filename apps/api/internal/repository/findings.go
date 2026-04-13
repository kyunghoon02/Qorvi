package repository

import (
	"context"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

type FindingsLoader interface {
	ListFindings(context.Context, db.FindingsQuery) (domain.FindingsFeedPage, error)
	ListWalletFindings(context.Context, db.WalletRef, int) ([]domain.Finding, error)
	GetFindingByID(context.Context, string) (domain.Finding, error)
}

type FindingsRepository interface {
	FindFindings(context.Context, string, int, []string) (domain.FindingsFeedPage, error)
	FindWalletFindings(context.Context, string, string, int) ([]domain.Finding, error)
	FindFindingByID(context.Context, string) (domain.Finding, error)
}

type QueryBackedFindingsRepository struct {
	loader FindingsLoader
}

func NewQueryBackedFindingsRepository(loader FindingsLoader) *QueryBackedFindingsRepository {
	return &QueryBackedFindingsRepository{loader: loader}
}

func (r *QueryBackedFindingsRepository) FindFindings(
	ctx context.Context,
	cursor string,
	limit int,
	types []string,
) (domain.FindingsFeedPage, error) {
	if r == nil || r.loader == nil {
		return domain.FindingsFeedPage{}, nil
	}
	query, err := db.BuildFindingsQuery(limit, cursor, types)
	if err != nil {
		return domain.FindingsFeedPage{}, err
	}
	return r.loader.ListFindings(ctx, query)
}

func (r *QueryBackedFindingsRepository) FindWalletFindings(
	ctx context.Context,
	chain string,
	address string,
	limit int,
) ([]domain.Finding, error) {
	if r == nil || r.loader == nil {
		return nil, nil
	}
	return r.loader.ListWalletFindings(ctx, db.WalletRef{
		Chain:   domain.Chain(chain),
		Address: address,
	}, limit)
}

func (r *QueryBackedFindingsRepository) FindFindingByID(
	ctx context.Context,
	id string,
) (domain.Finding, error) {
	if r == nil || r.loader == nil {
		return domain.Finding{}, nil
	}
	return r.loader.GetFindingByID(ctx, id)
}
