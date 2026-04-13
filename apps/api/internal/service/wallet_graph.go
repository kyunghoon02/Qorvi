package service

import (
	"context"
	"errors"

	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/packages/domain"
)

var ErrWalletGraphNotFound = errors.New("wallet graph not found")
var ErrWalletGraphDepthNotAllowed = errors.New("wallet graph depth not allowed")

type WalletGraphService struct {
	repo repository.WalletGraphRepository
}

func NewWalletGraphService(repo repository.WalletGraphRepository) *WalletGraphService {
	return &WalletGraphService{repo: repo}
}

func (s *WalletGraphService) GetWalletGraph(
	ctx context.Context,
	chain string,
	address string,
	depth int,
	_ string,
) (domain.WalletGraph, error) {
	if depth <= 0 {
		depth = 1
	}

	graph, err := s.repo.FindWalletGraph(ctx, chain, address, depth)
	if err != nil {
		if errors.Is(err, repository.ErrWalletGraphNotFound) {
			return domain.WalletGraph{}, ErrWalletGraphNotFound
		}

		return domain.WalletGraph{}, err
	}

	summary := domain.BuildWalletGraphNeighborhoodSummary(graph)
	graph.NeighborhoodSummary = &summary

	return graph, nil
}
