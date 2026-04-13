package service

import (
	"context"
	"errors"

	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/packages/domain"
)

var ErrClusterDetailNotFound = errors.New("cluster detail not found")

type ClusterMember struct {
	Chain            string `json:"chain"`
	Address          string `json:"address"`
	Label            string `json:"label"`
	InteractionCount int    `json:"interactionCount"`
	LatestActivityAt string `json:"latestActivityAt,omitempty"`
	Role             string `json:"role,omitempty"`
}

type ClusterCommonAction struct {
	Label       string `json:"label"`
	Description string `json:"description"`
	Href        string `json:"href,omitempty"`
}

type ClusterDetail struct {
	ID             string                `json:"id"`
	Label          string                `json:"label"`
	ClusterType    string                `json:"clusterType"`
	Score          int                   `json:"score"`
	Classification string                `json:"classification"`
	MemberCount    int                   `json:"memberCount"`
	Members        []ClusterMember       `json:"members"`
	CommonActions  []ClusterCommonAction `json:"commonActions"`
	Evidence       []Evidence            `json:"evidence"`
}

type ClusterDetailService struct {
	repo repository.ClusterDetailRepository
}

func NewClusterDetailService(repo repository.ClusterDetailRepository) *ClusterDetailService {
	return &ClusterDetailService{repo: repo}
}

func (s *ClusterDetailService) GetClusterDetail(ctx context.Context, clusterID string) (ClusterDetail, error) {
	record, err := s.repo.FindClusterDetail(ctx, clusterID)
	if err != nil {
		if errors.Is(err, repository.ErrClusterDetailNotFound) {
			return ClusterDetail{}, ErrClusterDetailNotFound
		}
		return ClusterDetail{}, err
	}
	return toClusterDetailResponse(record), nil
}

func toClusterDetailResponse(detail domain.ClusterDetail) ClusterDetail {
	members := make([]ClusterMember, 0, len(detail.Members))
	for _, member := range detail.Members {
		members = append(members, ClusterMember{
			Chain:            string(member.Chain),
			Address:          member.Address,
			Label:            member.Label,
			InteractionCount: 0,
		})
	}

	actions := make([]ClusterCommonAction, 0, len(detail.CommonActions))
	for _, action := range detail.CommonActions {
		actions = append(actions, ClusterCommonAction{
			Label:       action.Label,
			Description: buildClusterActionDescription(action),
			Href:        buildClusterActionHref(action),
		})
	}

	return ClusterDetail{
		ID:             detail.ID,
		Label:          detail.Label,
		ClusterType:    detail.ClusterType,
		Score:          detail.Score,
		Classification: string(detail.Classification),
		MemberCount:    detail.MemberCount,
		Members:        members,
		CommonActions:  actions,
		Evidence:       convertEvidence(detail.Evidence),
	}
}

func buildClusterActionDescription(action domain.ClusterCommonAction) string {
	if action.SharedMemberCount > 0 || action.InteractionCount > 0 {
		return "Shared member activity converges on this counterparty and merits follow-up review."
	}

	return "Cluster members repeatedly converge on this action and it should be reviewed."
}

func buildClusterActionHref(action domain.ClusterCommonAction) string {
	if action.Chain == "" || action.Address == "" {
		return ""
	}

	return "/wallets/" + string(action.Chain) + "/" + action.Address
}
