package service

import (
	"context"
	"errors"
	"testing"

	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeClusterDetailRepository struct {
	detail domain.ClusterDetail
	err    error
	called bool
}

func (f *fakeClusterDetailRepository) FindClusterDetail(context.Context, string) (domain.ClusterDetail, error) {
	f.called = true
	return f.detail, f.err
}

func TestClusterDetailServiceConvertsRepositoryRecord(t *testing.T) {
	t.Parallel()

	repo := &fakeClusterDetailRepository{
		detail: domain.ClusterDetail{
			ID:             "cluster_seed_whales",
			Label:          "cluster_seed_whales",
			ClusterType:    "whale",
			Score:          82,
			Classification: domain.ClusterClassificationStrong,
			MemberCount:    2,
			Members: []domain.ClusterMember{
				{Chain: domain.ChainEVM, Address: "0xaaa", Label: "Seed A"},
			},
			CommonActions: []domain.ClusterCommonAction{
				{Kind: "shared_counterparty", Label: "Bridge wallet", Chain: domain.ChainEVM, Address: "0xbbb", SharedMemberCount: 2, InteractionCount: 11},
			},
			Evidence: []domain.Evidence{
				{Kind: domain.EvidenceClusterOverlap, Label: "cluster overlap", Source: "cluster-detail", Confidence: 0.9},
			},
		},
	}

	svc := NewClusterDetailService(repo)
	detail, err := svc.GetClusterDetail(context.Background(), "cluster_seed_whales")
	if err != nil {
		t.Fatalf("GetClusterDetail returned error: %v", err)
	}
	if !repo.called {
		t.Fatal("expected repository to be called")
	}
	if detail.Classification != "strong" || len(detail.Members) != 1 || len(detail.CommonActions) != 1 {
		t.Fatalf("unexpected detail %#v", detail)
	}
}

func TestClusterDetailServiceReturnsNotFound(t *testing.T) {
	t.Parallel()

	svc := NewClusterDetailService(&fakeClusterDetailRepository{err: repository.ErrClusterDetailNotFound})
	_, err := svc.GetClusterDetail(context.Background(), "cluster_missing")
	if !errors.Is(err, ErrClusterDetailNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}
