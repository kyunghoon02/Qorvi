package service

import (
	"context"

	"github.com/whalegraph/whalegraph/apps/api/internal/repository"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type ShadowExitFeedItem struct {
	WalletID    string     `json:"walletId"`
	Chain       string     `json:"chain"`
	Address     string     `json:"address"`
	Label       string     `json:"label"`
	ClusterID   string     `json:"clusterId,omitempty"`
	WalletRoute string     `json:"walletRoute"`
	Explanation string     `json:"explanation"`
	ObservedAt  string     `json:"observedAt"`
	Score       int        `json:"score"`
	Rating      string     `json:"rating"`
	Evidence    []Evidence `json:"evidence"`
}

type ShadowExitFeedResponse struct {
	WindowLabel string               `json:"windowLabel"`
	GeneratedAt string               `json:"generatedAt"`
	Items       []ShadowExitFeedItem `json:"items"`
	NextCursor  string               `json:"nextCursor,omitempty"`
	HasMore     bool                 `json:"hasMore"`
}

type ShadowExitFeedService struct {
	repo repository.ShadowExitFeedRepository
}

func NewShadowExitFeedService(repo repository.ShadowExitFeedRepository) *ShadowExitFeedService {
	return &ShadowExitFeedService{repo: repo}
}

func (s *ShadowExitFeedService) ListShadowExitFeed(ctx context.Context, cursor string, limit int) (ShadowExitFeedResponse, error) {
	if s == nil || s.repo == nil {
		return ShadowExitFeedResponse{}, nil
	}

	page, err := s.repo.FindShadowExitFeed(ctx, cursor, limit)
	if err != nil {
		return ShadowExitFeedResponse{}, err
	}

	return toShadowExitFeedResponse(page), nil
}

func toShadowExitFeedResponse(page domain.ShadowExitFeedPage) ShadowExitFeedResponse {
	items := make([]ShadowExitFeedItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, ShadowExitFeedItem{
			WalletID:    item.WalletID,
			Chain:       string(item.Chain),
			Address:     item.Address,
			Label:       item.Label,
			WalletRoute: item.WalletRoute,
			Explanation: item.Recommendation,
			ObservedAt:  item.ObservedAt,
			Score:       item.Score.Value,
			Rating:      string(item.Score.Rating),
			Evidence:    convertEvidence(item.Score.Evidence),
		})
	}

	response := ShadowExitFeedResponse{
		WindowLabel: "Last 24 hours",
		Items:       items,
		HasMore:     page.HasMore,
	}
	if len(items) > 0 {
		response.GeneratedAt = items[0].ObservedAt
	}
	if page.NextCursor != nil {
		response.NextCursor = *page.NextCursor
	}

	return response
}
