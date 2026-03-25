package service

import (
	"context"

	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/packages/domain"
)

type FindingEvidence struct {
	Type       string         `json:"type"`
	Value      string         `json:"value,omitempty"`
	Confidence float64        `json:"confidence,omitempty"`
	ObservedAt string         `json:"observedAt,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type NextWatch struct {
	SubjectType string `json:"subjectType"`
	Chain       string `json:"chain,omitempty"`
	Address     string `json:"address,omitempty"`
	Token       string `json:"token,omitempty"`
	Label       string `json:"label,omitempty"`
}

type FindingItem struct {
	ID                     string            `json:"id"`
	Type                   string            `json:"type"`
	SubjectType            string            `json:"subjectType"`
	Chain                  string            `json:"chain,omitempty"`
	Address                string            `json:"address,omitempty"`
	Key                    string            `json:"key,omitempty"`
	Label                  string            `json:"label,omitempty"`
	Summary                string            `json:"summary"`
	ImportanceReason       []string          `json:"importanceReason"`
	ObservedFacts          []string          `json:"observedFacts"`
	InferredInterpretation []string          `json:"inferredInterpretations"`
	Confidence             float64           `json:"confidence"`
	ImportanceScore        float64           `json:"importanceScore"`
	ObservedAt             string            `json:"observedAt"`
	CoverageStartAt        string            `json:"coverageStartAt,omitempty"`
	CoverageEndAt          string            `json:"coverageEndAt,omitempty"`
	CoverageWindowDays     int               `json:"coverageWindowDays"`
	Evidence               []FindingEvidence `json:"evidence"`
	NextWatch              []NextWatch       `json:"nextWatch"`
}

type FindingsFeedResponse struct {
	GeneratedAt string        `json:"generatedAt"`
	Items       []FindingItem `json:"items"`
	NextCursor  string        `json:"nextCursor,omitempty"`
	HasMore     bool          `json:"hasMore"`
}

type FindingsFeedService struct {
	repo repository.FindingsRepository
}

func NewFindingsFeedService(repo repository.FindingsRepository) *FindingsFeedService {
	return &FindingsFeedService{repo: repo}
}

func (s *FindingsFeedService) ListFindings(ctx context.Context, cursor string, limit int, types []string) (FindingsFeedResponse, error) {
	if s == nil || s.repo == nil {
		return FindingsFeedResponse{}, nil
	}
	page, err := s.repo.FindFindings(ctx, cursor, limit, types)
	if err != nil {
		return FindingsFeedResponse{}, err
	}
	return toFindingsFeedResponse(page), nil
}

func toFindingsFeedResponse(page domain.FindingsFeedPage) FindingsFeedResponse {
	items := make([]FindingItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, toFindingItem(item))
	}
	response := FindingsFeedResponse{
		Items:   items,
		HasMore: page.HasMore,
	}
	if len(items) > 0 {
		response.GeneratedAt = items[0].ObservedAt
	}
	if page.NextCursor != nil {
		response.NextCursor = *page.NextCursor
	}
	return response
}

func toFindingItem(item domain.Finding) FindingItem {
	evidence := make([]FindingEvidence, 0, len(item.Evidence))
	for _, part := range item.Evidence {
		evidence = append(evidence, FindingEvidence{
			Type:       part.Type,
			Value:      part.Value,
			Confidence: part.Confidence,
			ObservedAt: part.ObservedAt,
			Metadata:   part.Metadata,
		})
	}
	nextWatch := make([]NextWatch, 0, len(item.NextWatch))
	for _, part := range item.NextWatch {
		nextWatch = append(nextWatch, NextWatch{
			SubjectType: string(part.SubjectType),
			Chain:       string(part.Chain),
			Address:     part.Address,
			Token:       part.Token,
			Label:       part.Label,
		})
	}
	return FindingItem{
		ID:                     item.ID,
		Type:                   string(item.Type),
		SubjectType:            string(item.Subject.SubjectType),
		Chain:                  string(item.Subject.Chain),
		Address:                item.Subject.Address,
		Key:                    item.Subject.Key,
		Label:                  item.Subject.Label,
		Summary:                item.Summary,
		ImportanceReason:       append([]string(nil), item.ImportanceReason...),
		ObservedFacts:          append([]string(nil), item.ObservedFacts...),
		InferredInterpretation: append([]string(nil), item.InferredInterpretation...),
		Confidence:             item.Confidence,
		ImportanceScore:        item.ImportanceScore,
		ObservedAt:             item.ObservedAt,
		CoverageStartAt:        item.Coverage.CoverageStartAt,
		CoverageEndAt:          item.Coverage.CoverageEndAt,
		CoverageWindowDays:     item.Coverage.CoverageWindowDays,
		Evidence:               evidence,
		NextWatch:              nextWatch,
	}
}
