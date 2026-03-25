package service

import (
	"context"

	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/packages/domain"
)

type FirstConnectionFeedItem struct {
	WalletID    string     `json:"walletId"`
	Chain       string     `json:"chain"`
	Address     string     `json:"address"`
	Label       string     `json:"label"`
	WalletRoute string     `json:"walletRoute"`
	Explanation string     `json:"explanation"`
	ObservedAt  string     `json:"observedAt"`
	Score       int        `json:"score"`
	Rating      string     `json:"rating"`
	Evidence    []Evidence `json:"evidence"`
}

type FirstConnectionFeedResponse struct {
	WindowLabel string                    `json:"windowLabel"`
	GeneratedAt string                    `json:"generatedAt"`
	Items       []FirstConnectionFeedItem `json:"items"`
	NextCursor  string                    `json:"nextCursor,omitempty"`
	HasMore     bool                      `json:"hasMore"`
}

type FirstConnectionFeedService struct {
	repo repository.FirstConnectionFeedRepository
}

func NewFirstConnectionFeedService(repo repository.FirstConnectionFeedRepository) *FirstConnectionFeedService {
	return &FirstConnectionFeedService{repo: repo}
}

func (s *FirstConnectionFeedService) ListFirstConnectionFeed(ctx context.Context, cursor string, limit int) (FirstConnectionFeedResponse, error) {
	return s.ListFirstConnectionFeedSorted(ctx, cursor, limit, "")
}

func (s *FirstConnectionFeedService) ListFirstConnectionFeedSorted(ctx context.Context, cursor string, limit int, sort string) (FirstConnectionFeedResponse, error) {
	if s == nil || s.repo == nil {
		return FirstConnectionFeedResponse{}, nil
	}

	page, err := s.repo.FindFirstConnectionFeed(ctx, cursor, limit, sort)
	if err != nil {
		return FirstConnectionFeedResponse{}, err
	}

	return toFirstConnectionFeedResponse(page), nil
}

func toFirstConnectionFeedResponse(page domain.FirstConnectionFeedPage) FirstConnectionFeedResponse {
	items := make([]FirstConnectionFeedItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, FirstConnectionFeedItem{
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

	response := FirstConnectionFeedResponse{
		WindowLabel: "Hot feed baseline",
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
