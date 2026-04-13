package repository

import (
	"context"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

type EntityInterpretationLoader interface {
	ReadEntityInterpretation(context.Context, db.EntityInterpretationQuery) (domain.EntityInterpretation, error)
}

type EntityInterpretationRepository interface {
	FindEntityInterpretation(context.Context, string) (domain.EntityInterpretation, error)
}

type QueryBackedEntityInterpretationRepository struct {
	loader EntityInterpretationLoader
}

func NewQueryBackedEntityInterpretationRepository(loader EntityInterpretationLoader) *QueryBackedEntityInterpretationRepository {
	return &QueryBackedEntityInterpretationRepository{loader: loader}
}

func (r *QueryBackedEntityInterpretationRepository) FindEntityInterpretation(
	ctx context.Context,
	entityKey string,
) (domain.EntityInterpretation, error) {
	if r == nil || r.loader == nil {
		return domain.EntityInterpretation{}, db.ErrEntityInterpretationNotFound
	}
	query, err := db.BuildEntityInterpretationQuery(entityKey, 12, 8)
	if err != nil {
		return domain.EntityInterpretation{}, err
	}
	return r.loader.ReadEntityInterpretation(ctx, query)
}
