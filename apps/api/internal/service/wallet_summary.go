package service

import (
	"context"
	"errors"
	"strings"

	"github.com/whalegraph/whalegraph/apps/api/internal/repository"
	"github.com/whalegraph/whalegraph/packages/domain"
)

var ErrWalletSummaryNotFound = errors.New("wallet summary not found")

type Evidence struct {
	Kind       string         `json:"kind"`
	Label      string         `json:"label"`
	Source     string         `json:"source"`
	Confidence float64        `json:"confidence"`
	ObservedAt string         `json:"observedAt"`
	Metadata   map[string]any `json:"metadata"`
}

type Score struct {
	Name     string     `json:"name"`
	Value    int        `json:"value"`
	Rating   string     `json:"rating"`
	Evidence []Evidence `json:"evidence"`
}

type WalletSummary struct {
	Chain            string   `json:"chain"`
	Address          string   `json:"address"`
	DisplayName      string   `json:"displayName"`
	ClusterID        string   `json:"clusterId"`
	Counterparties   int      `json:"counterparties"`
	LatestActivityAt string   `json:"latestActivityAt"`
	Tags             []string `json:"tags"`
	Scores           []Score  `json:"scores"`
}

type WalletSummaryService struct {
	repo repository.WalletSummaryRepository
}

func NewWalletSummaryService(repo repository.WalletSummaryRepository) *WalletSummaryService {
	return &WalletSummaryService{repo: repo}
}

func (s *WalletSummaryService) GetWalletSummary(ctx context.Context, chain, address string) (WalletSummary, error) {
	record, err := s.repo.FindWalletSummary(ctx, chain, address)
	if err != nil {
		if errors.Is(err, repository.ErrWalletSummaryNotFound) {
			return WalletSummary{}, ErrWalletSummaryNotFound
		}

		return WalletSummary{}, err
	}

	return toResponse(record), nil
}

func toResponse(summary domain.WalletSummary) WalletSummary {
	clusterID := ""
	if summary.ClusterID != nil {
		clusterID = *summary.ClusterID
	}

	scores := make([]Score, 0, len(summary.Scores))
	for _, score := range summary.Scores {
		scores = append(scores, Score{
			Name:     string(score.Name),
			Value:    score.Value,
			Rating:   string(score.Rating),
			Evidence: convertEvidence(score.Evidence),
		})
	}

	return WalletSummary{
		Chain:            string(summary.Chain),
		Address:          strings.TrimSpace(summary.Address),
		DisplayName:      summary.DisplayName,
		ClusterID:        clusterID,
		Counterparties:   summary.Counterparties,
		LatestActivityAt: summary.LatestActivityAt,
		Tags:             append([]string(nil), summary.Tags...),
		Scores:           scores,
	}
}

func convertEvidence(items []domain.Evidence) []Evidence {
	out := make([]Evidence, 0, len(items))
	for _, item := range items {
		out = append(out, Evidence{
			Kind:       string(item.Kind),
			Label:      item.Label,
			Source:     item.Source,
			Confidence: item.Confidence,
			ObservedAt: item.ObservedAt,
			Metadata:   copyMetadata(item.Metadata),
		})
	}

	return out
}

func copyMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}

	return cloned
}
