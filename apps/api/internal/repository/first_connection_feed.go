package repository

import (
	"context"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

type FirstConnectionFeedLoader interface {
	LoadFirstConnectionFeed(context.Context, db.FirstConnectionFeedQuery) (domain.FirstConnectionFeedPage, error)
}

type FirstConnectionFeedRepository interface {
	FindFirstConnectionFeed(context.Context, string, int, string) (domain.FirstConnectionFeedPage, error)
}

type QueryBackedFirstConnectionFeedRepository struct {
	loader FirstConnectionFeedLoader
}

func NewQueryBackedFirstConnectionFeedRepository(loader FirstConnectionFeedLoader) *QueryBackedFirstConnectionFeedRepository {
	return &QueryBackedFirstConnectionFeedRepository{loader: loader}
}

func (r *QueryBackedFirstConnectionFeedRepository) FindFirstConnectionFeed(
	ctx context.Context,
	cursor string,
	limit int,
	sort string,
) (domain.FirstConnectionFeedPage, error) {
	if r == nil || r.loader == nil {
		return domain.FirstConnectionFeedPage{}, nil
	}

	query, err := db.BuildFirstConnectionFeedQuery(limit, cursor, sort)
	if err != nil {
		return domain.FirstConnectionFeedPage{}, err
	}

	page, err := r.loader.LoadFirstConnectionFeed(ctx, query)
	if err != nil {
		return domain.FirstConnectionFeedPage{}, err
	}

	return page, nil
}
