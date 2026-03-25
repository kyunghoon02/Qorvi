package repository

import (
	"context"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

type ShadowExitFeedLoader interface {
	LoadShadowExitFeed(context.Context, db.ShadowExitFeedQuery) (domain.ShadowExitFeedPage, error)
}

type ShadowExitFeedRepository interface {
	FindShadowExitFeed(context.Context, string, int) (domain.ShadowExitFeedPage, error)
}

type QueryBackedShadowExitFeedRepository struct {
	loader ShadowExitFeedLoader
}

func NewQueryBackedShadowExitFeedRepository(loader ShadowExitFeedLoader) *QueryBackedShadowExitFeedRepository {
	return &QueryBackedShadowExitFeedRepository{loader: loader}
}

func (r *QueryBackedShadowExitFeedRepository) FindShadowExitFeed(
	ctx context.Context,
	cursor string,
	limit int,
) (domain.ShadowExitFeedPage, error) {
	if r == nil || r.loader == nil {
		return domain.ShadowExitFeedPage{}, nil
	}

	query, err := db.BuildShadowExitFeedQuery(limit, cursor)
	if err != nil {
		return domain.ShadowExitFeedPage{}, err
	}

	page, err := r.loader.LoadShadowExitFeed(ctx, query)
	if err != nil {
		return domain.ShadowExitFeedPage{}, err
	}

	return page, nil
}
