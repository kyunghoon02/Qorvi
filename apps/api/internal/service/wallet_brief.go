package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/packages/domain"
)

type WalletLabel struct {
	Key             string  `json:"key"`
	Name            string  `json:"name"`
	Class           string  `json:"class"`
	EntityType      string  `json:"entityType"`
	Source          string  `json:"source"`
	Confidence      float64 `json:"confidence"`
	EvidenceSummary string  `json:"evidenceSummary"`
	ObservedAt      string  `json:"observedAt"`
}

type WalletBrief struct {
	Chain             string               `json:"chain"`
	Address           string               `json:"address"`
	DisplayName       string               `json:"displayName"`
	AISummary         string               `json:"aiSummary"`
	KeyFindings       []FindingItem        `json:"keyFindings"`
	VerifiedLabels    []WalletLabel        `json:"verifiedLabels"`
	ProbableLabels    []WalletLabel        `json:"probableLabels"`
	BehavioralLabels  []WalletLabel        `json:"behavioralLabels"`
	TopCounterparties []WalletCounterparty `json:"topCounterparties"`
	RecentFlow        WalletRecentFlow     `json:"recentFlow"`
	Enrichment        *WalletEnrichment    `json:"enrichment,omitempty"`
	Indexing          WalletIndexingState  `json:"indexing"`
	LatestSignals     []WalletLatestSignal `json:"latestSignals"`
	Scores            []Score              `json:"scores"`
}

type WalletBriefService struct {
	repo        repository.WalletSummaryRepository
	enricher    WalletSummaryEnricher
	findings    repository.FindingsRepository
	findingLimit int
}

func NewWalletBriefService(
	repo repository.WalletSummaryRepository,
	enricher WalletSummaryEnricher,
	findings repository.FindingsRepository,
) *WalletBriefService {
	return &WalletBriefService{
		repo:         repo,
		enricher:     enricher,
		findings:     findings,
		findingLimit: 5,
	}
}

func (s *WalletBriefService) GetWalletBrief(ctx context.Context, chain, address string) (WalletBrief, error) {
	if s == nil || s.repo == nil {
		return WalletBrief{}, ErrWalletSummaryNotFound
	}
	record, err := s.repo.FindWalletSummary(ctx, chain, address)
	if err != nil {
		if errors.Is(err, repository.ErrWalletSummaryNotFound) {
			return WalletBrief{}, ErrWalletSummaryNotFound
		}
		return WalletBrief{}, err
	}
	if s.enricher != nil {
		if enriched, enrichErr := s.enricher.EnrichWalletSummary(ctx, record); enrichErr == nil {
			record = enriched
		}
	}
	var findings []domain.Finding
	if s.findings != nil {
		found, err := s.findings.FindWalletFindings(ctx, chain, address, s.findingLimit)
		if err != nil {
			return WalletBrief{}, err
		}
		findings = found
	}
	summary := toResponse(record)
	return WalletBrief{
		Chain:             summary.Chain,
		Address:           summary.Address,
		DisplayName:       summary.DisplayName,
		AISummary:         buildDeterministicWalletBrief(record, findings),
		KeyFindings:       convertFindingItems(findings),
		VerifiedLabels:    convertWalletLabels(record.Labels.Verified),
		ProbableLabels:    convertWalletLabels(record.Labels.Inferred),
		BehavioralLabels:  convertWalletLabels(record.Labels.Behavioral),
		TopCounterparties: summary.TopCounterparties,
		RecentFlow:        summary.RecentFlow,
		Enrichment:        summary.Enrichment,
		Indexing:          summary.Indexing,
		LatestSignals:     summary.LatestSignals,
		Scores:            summary.Scores,
	}, nil
}

func buildDeterministicWalletBrief(summary domain.WalletSummary, findings []domain.Finding) string {
	if len(findings) > 0 && strings.TrimSpace(findings[0].Summary) != "" {
		return strings.TrimSpace(findings[0].Summary)
	}
	if len(summary.Labels.Behavioral) > 0 {
		label := summary.Labels.Behavioral[0]
		if strings.TrimSpace(label.Name) != "" {
			return fmt.Sprintf("%s recently exhibits %s behavior within the indexed coverage window.", summary.DisplayName, strings.ToLower(label.Name))
		}
	}
	if len(summary.LatestSignals) > 0 {
		signal := summary.LatestSignals[0]
		if strings.TrimSpace(signal.Label) != "" {
			return fmt.Sprintf("%s most recently triggered %s.", summary.DisplayName, signal.Label)
		}
		return fmt.Sprintf("%s has recent %s activity worth reviewing.", summary.DisplayName, strings.ReplaceAll(string(signal.Name), "_", " "))
	}
	return fmt.Sprintf("%s has indexed activity, but no major findings have been materialized yet.", summary.DisplayName)
}

func convertFindingItems(items []domain.Finding) []FindingItem {
	out := make([]FindingItem, 0, len(items))
	for _, item := range items {
		out = append(out, toFindingItem(item))
	}
	return out
}

func convertWalletLabels(items []domain.WalletLabel) []WalletLabel {
	out := make([]WalletLabel, 0, len(items))
	for _, item := range items {
		out = append(out, WalletLabel{
			Key:             item.Key,
			Name:            item.Name,
			Class:           string(item.Class),
			EntityType:      item.EntityType,
			Source:          item.Source,
			Confidence:      item.Confidence,
			EvidenceSummary: item.EvidenceSummary,
			ObservedAt:      item.ObservedAt,
		})
	}
	return out
}
