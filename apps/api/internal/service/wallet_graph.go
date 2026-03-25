package service

import (
	"context"
	"errors"
	"strings"

	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/packages/billing"
	"github.com/flowintel/flowintel/packages/domain"
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
	tier string,
) (domain.WalletGraph, error) {
	if depth <= 0 {
		depth = 1
	}

	maxDepth := maxGraphDepthForTier(tier)
	if depth > maxDepth {
		return domain.WalletGraph{}, ErrWalletGraphDepthNotAllowed
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

func maxGraphDepthForTier(tier string) int {
	normalizedTier := strings.ToLower(strings.TrimSpace(tier))
	if normalizedTier == "" {
		normalizedTier = string(domain.PlanFree)
	}

	plan, err := billing.FindPlan(domain.PlanTier(normalizedTier))
	if err != nil {
		return 1
	}

	entitlement, ok := billing.EntitlementFor(plan, billing.FeatureGraph)
	if !ok || entitlement.MaxGraphDepth <= 0 {
		return 1
	}

	return entitlement.MaxGraphDepth
}
