package service

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/packages/domain"
)

var ErrFindingNotFound = errors.New("finding not found")

type AnalystFindingDetail struct {
	Finding FindingItem `json:"finding"`
}

type AnalystFindingTimelineItem struct {
	ObservedAt  string         `json:"observedAt"`
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	Summary     string         `json:"summary"`
	Confidence  float64        `json:"confidence,omitempty"`
	Source      string         `json:"source,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type AnalystFindingEvidenceTimeline struct {
	FindingID   string                     `json:"findingId"`
	SubjectType string                     `json:"subjectType"`
	Chain       string                     `json:"chain,omitempty"`
	Address     string                     `json:"address,omitempty"`
	Key         string                     `json:"key,omitempty"`
	Label       string                     `json:"label,omitempty"`
	Items       []AnalystFindingTimelineItem `json:"items"`
}

type AnalystHistoricalAnalogs struct {
	FindingID string        `json:"findingId"`
	Items     []FindingItem `json:"items"`
}

type AnalystFindingDrilldownService struct {
	findings repository.FindingsRepository
	wallets  *WalletSummaryService
}

func NewAnalystFindingDrilldownService(
	findings repository.FindingsRepository,
	wallets *WalletSummaryService,
) *AnalystFindingDrilldownService {
	return &AnalystFindingDrilldownService{
		findings: findings,
		wallets:  wallets,
	}
}

func (s *AnalystFindingDrilldownService) GetFindingDetail(ctx context.Context, findingID string) (AnalystFindingDetail, error) {
	finding, err := s.lookupFinding(ctx, findingID)
	if err != nil {
		return AnalystFindingDetail{}, err
	}
	return AnalystFindingDetail{Finding: toFindingItem(finding)}, nil
}

func (s *AnalystFindingDrilldownService) GetEvidenceTimeline(ctx context.Context, findingID string) (AnalystFindingEvidenceTimeline, error) {
	finding, err := s.lookupFinding(ctx, findingID)
	if err != nil {
		return AnalystFindingEvidenceTimeline{}, err
	}

	items := make([]AnalystFindingTimelineItem, 0, len(finding.Evidence)+len(finding.ObservedFacts)+4)
	for _, evidence := range finding.Evidence {
		items = append(items, AnalystFindingTimelineItem{
			ObservedAt: evidence.ObservedAt,
			Type:       evidence.Type,
			Title:      strings.ReplaceAll(evidence.Type, "_", " "),
			Summary:    evidence.Value,
			Confidence: evidence.Confidence,
			Source:     stringValueAny(evidence.Metadata["source"]),
			Metadata:   cloneMap(evidence.Metadata),
		})
	}
	for _, fact := range finding.ObservedFacts {
		items = append(items, AnalystFindingTimelineItem{
			ObservedAt: finding.ObservedAt,
			Type:       "observed_fact",
			Title:      "observed fact",
			Summary:    fact,
		})
	}

	if s.wallets != nil && finding.Subject.SubjectType == domain.FindingSubjectWallet &&
		finding.Subject.Chain != "" && strings.TrimSpace(finding.Subject.Address) != "" {
		if summary, err := s.wallets.GetWalletSummary(ctx, string(finding.Subject.Chain), finding.Subject.Address); err == nil {
			for _, signal := range summary.LatestSignals {
				items = append(items, AnalystFindingTimelineItem{
					ObservedAt: signal.ObservedAt,
					Type:       "latest_signal",
					Title:      signal.Label,
					Summary:    signal.Source,
					Confidence: float64(signal.Value) / 100,
					Source:     signal.Source,
					Metadata: map[string]any{
						"name":   signal.Name,
						"rating": signal.Rating,
						"value":  signal.Value,
					},
				})
			}
			for _, score := range summary.Scores {
				for _, evidence := range score.Evidence {
					items = append(items, AnalystFindingTimelineItem{
						ObservedAt: evidence.ObservedAt,
						Type:       "score_evidence",
						Title:      evidence.Label,
						Summary:    string(score.Name),
						Confidence: evidence.Confidence,
						Source:     evidence.Source,
						Metadata: map[string]any{
							"scoreName":  score.Name,
							"scoreValue": score.Value,
							"rating":     score.Rating,
						},
					})
				}
			}
		}
	}

	slices.SortStableFunc(items, func(left, right AnalystFindingTimelineItem) int {
		switch {
		case left.ObservedAt == right.ObservedAt:
			return strings.Compare(left.Type, right.Type)
		case left.ObservedAt > right.ObservedAt:
			return -1
		default:
			return 1
		}
	})

	return AnalystFindingEvidenceTimeline{
		FindingID:   finding.ID,
		SubjectType: string(finding.Subject.SubjectType),
		Chain:       string(finding.Subject.Chain),
		Address:     finding.Subject.Address,
		Key:         finding.Subject.Key,
		Label:       finding.Subject.Label,
		Items:       items,
	}, nil
}

func (s *AnalystFindingDrilldownService) GetHistoricalAnalogs(ctx context.Context, findingID string, limit int) (AnalystHistoricalAnalogs, error) {
	finding, err := s.lookupFinding(ctx, findingID)
	if err != nil {
		return AnalystHistoricalAnalogs{}, err
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	page, err := s.findings.FindFindings(ctx, "", max(limit*4, 20), []string{string(finding.Type)})
	if err != nil {
		return AnalystHistoricalAnalogs{}, err
	}

	items := make([]FindingItem, 0, limit)
	for _, item := range page.Items {
		if item.ID == finding.ID {
			continue
		}
		items = append(items, toFindingItem(item))
		if len(items) >= limit {
			break
		}
	}

	return AnalystHistoricalAnalogs{
		FindingID: finding.ID,
		Items:     items,
	}, nil
}

func (s *AnalystFindingDrilldownService) lookupFinding(ctx context.Context, findingID string) (domain.Finding, error) {
	if s == nil || s.findings == nil {
		return domain.Finding{}, ErrFindingNotFound
	}
	finding, err := s.findings.FindFindingByID(ctx, findingID)
	if err != nil {
		return domain.Finding{}, err
	}
	if strings.TrimSpace(finding.ID) == "" {
		return domain.Finding{}, ErrFindingNotFound
	}
	return finding, nil
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func stringValueAny(input any) string {
	value, _ := input.(string)
	return strings.TrimSpace(value)
}
