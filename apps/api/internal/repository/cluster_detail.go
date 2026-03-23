package repository

import (
	"context"
	"errors"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
)

var ErrClusterDetailNotFound = errors.New("cluster detail not found")

type ClusterDetailLoader interface {
	LoadClusterDetail(context.Context, string) (domain.ClusterDetail, error)
}

type ClusterDetailRepository interface {
	FindClusterDetail(context.Context, string) (domain.ClusterDetail, error)
}

type QueryBackedClusterDetailRepository struct {
	loader ClusterDetailLoader
}

func NewQueryBackedClusterDetailRepository(loader ClusterDetailLoader) *QueryBackedClusterDetailRepository {
	return &QueryBackedClusterDetailRepository{loader: loader}
}

func (r *QueryBackedClusterDetailRepository) FindClusterDetail(
	ctx context.Context,
	clusterID string,
) (domain.ClusterDetail, error) {
	if r == nil || r.loader == nil {
		return domain.ClusterDetail{}, ErrClusterDetailNotFound
	}

	detail, err := r.loader.LoadClusterDetail(ctx, clusterID)
	if err != nil {
		if errors.Is(err, db.ErrClusterDetailNotFound) || errors.Is(err, ErrClusterDetailNotFound) {
			return domain.ClusterDetail{}, ErrClusterDetailNotFound
		}
		return domain.ClusterDetail{}, err
	}

	return detail, nil
}
