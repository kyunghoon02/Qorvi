package db

import (
	"context"
	"fmt"

	"github.com/qorvi/qorvi/packages/domain"
)

type WalletGraphQuery struct {
	Ref               WalletRef
	DepthRequested    int
	DepthResolved     int
	MaxCounterparties int
}

const DefaultWalletGraphMaxCounterparties = 25

type WalletGraphReader interface {
	ReadWalletGraph(context.Context, WalletGraphQuery) (domain.WalletGraph, error)
}

type WalletGraphRepository struct {
	Reader WalletGraphReader
}

var ErrWalletGraphNotFound = fmt.Errorf("wallet graph not found")

func NewWalletGraphRepository(reader WalletGraphReader) *WalletGraphRepository {
	return &WalletGraphRepository{Reader: reader}
}

func BuildWalletGraphQuery(
	ref WalletRef,
	depthRequested int,
	depthResolved int,
	maxCounterparties int,
) (WalletGraphQuery, error) {
	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return WalletGraphQuery{}, err
	}

	if depthRequested <= 0 {
		depthRequested = 1
	}
	if depthResolved <= 0 {
		depthResolved = 1
	}
	if maxCounterparties <= 0 {
		maxCounterparties = DefaultWalletGraphMaxCounterparties
	}

	return WalletGraphQuery{
		Ref:               normalized,
		DepthRequested:    depthRequested,
		DepthResolved:     depthResolved,
		MaxCounterparties: maxCounterparties,
	}, nil
}

func (r *WalletGraphRepository) LoadWalletGraph(
	ctx context.Context,
	query WalletGraphQuery,
) (domain.WalletGraph, error) {
	if r == nil || r.Reader == nil {
		return domain.WalletGraph{}, ErrWalletGraphNotFound
	}

	graph, err := r.Reader.ReadWalletGraph(ctx, query)
	if err != nil {
		return domain.WalletGraph{}, err
	}

	return graph, nil
}
