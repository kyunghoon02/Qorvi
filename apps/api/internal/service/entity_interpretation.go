package service

import (
	"context"
	"errors"

	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

var ErrEntityInterpretationNotFound = errors.New("entity interpretation not found")

type EntityMember struct {
	Chain            string        `json:"chain"`
	Address          string        `json:"address"`
	DisplayName      string        `json:"displayName"`
	LatestActivityAt string        `json:"latestActivityAt,omitempty"`
	VerifiedLabels   []WalletLabel `json:"verifiedLabels"`
	ProbableLabels   []WalletLabel `json:"probableLabels"`
	BehavioralLabels []WalletLabel `json:"behavioralLabels"`
}

type EntityInterpretation struct {
	EntityKey        string       `json:"entityKey"`
	EntityType       string       `json:"entityType"`
	DisplayName      string       `json:"displayName"`
	WalletCount      int          `json:"walletCount"`
	LatestActivityAt string       `json:"latestActivityAt,omitempty"`
	Members          []EntityMember `json:"members"`
	Findings         []FindingItem `json:"findings"`
}

type EntityInterpretationService struct {
	repo repository.EntityInterpretationRepository
}

func NewEntityInterpretationService(repo repository.EntityInterpretationRepository) *EntityInterpretationService {
	return &EntityInterpretationService{repo: repo}
}

func (s *EntityInterpretationService) GetEntityInterpretation(ctx context.Context, entityKey string) (EntityInterpretation, error) {
	if s == nil || s.repo == nil {
		return EntityInterpretation{}, ErrEntityInterpretationNotFound
	}
	record, err := s.repo.FindEntityInterpretation(ctx, entityKey)
	if err != nil {
		if errors.Is(err, db.ErrEntityInterpretationNotFound) {
			return EntityInterpretation{}, ErrEntityInterpretationNotFound
		}
		return EntityInterpretation{}, err
	}
	return toEntityInterpretationResponse(record), nil
}

func toEntityInterpretationResponse(input domain.EntityInterpretation) EntityInterpretation {
	members := make([]EntityMember, 0, len(input.Members))
	for _, item := range input.Members {
		members = append(members, EntityMember{
			Chain:            string(item.Chain),
			Address:          item.Address,
			DisplayName:      item.DisplayName,
			LatestActivityAt: item.LatestActivityAt,
			VerifiedLabels:   convertWalletLabels(item.Labels.Verified),
			ProbableLabels:   convertWalletLabels(item.Labels.Inferred),
			BehavioralLabels: convertWalletLabels(item.Labels.Behavioral),
		})
	}

	return EntityInterpretation{
		EntityKey:        input.EntityKey,
		EntityType:       input.EntityType,
		DisplayName:      input.DisplayName,
		WalletCount:      input.WalletCount,
		LatestActivityAt: input.LatestActivityAt,
		Members:          members,
		Findings:         convertFindingItems(input.Findings),
	}
}
