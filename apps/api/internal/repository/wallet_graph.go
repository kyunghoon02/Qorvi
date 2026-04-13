package repository

import (
	"context"
	"errors"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

var ErrWalletGraphNotFound = errors.New("wallet graph not found")

type WalletGraphLoader interface {
	LoadWalletGraph(context.Context, db.WalletGraphQuery) (domain.WalletGraph, error)
}

type WalletGraphRepository interface {
	FindWalletGraph(context.Context, string, string, int) (domain.WalletGraph, error)
}

type QueryBackedWalletGraphRepository struct {
	loader WalletGraphLoader
}

func NewQueryBackedWalletGraphRepository(loader WalletGraphLoader) *QueryBackedWalletGraphRepository {
	return &QueryBackedWalletGraphRepository{loader: loader}
}

func (r *QueryBackedWalletGraphRepository) FindWalletGraph(
	ctx context.Context,
	chain string,
	address string,
	depth int,
) (domain.WalletGraph, error) {
	if r == nil || r.loader == nil {
		return domain.WalletGraph{}, ErrWalletGraphNotFound
	}

	query, err := db.BuildWalletGraphQuery(db.WalletRef{
		Chain:   domain.Chain(chain),
		Address: address,
	}, depth, minGraphDepth(depth, 1), 25)
	if err != nil {
		return domain.WalletGraph{}, err
	}

	graph, err := r.loader.LoadWalletGraph(ctx, query)
	if err != nil {
		if errors.Is(err, db.ErrWalletGraphNotFound) || errors.Is(err, ErrWalletGraphNotFound) {
			return domain.WalletGraph{}, ErrWalletGraphNotFound
		}

		return domain.WalletGraph{}, err
	}

	return graph, nil
}

func minGraphDepth(left int, right int) int {
	if left < right {
		return left
	}

	return right
}
