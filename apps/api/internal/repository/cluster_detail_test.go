package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

type fakeClusterDetailLoader struct {
	detail domain.ClusterDetail
	err    error
	called bool
}

func (f *fakeClusterDetailLoader) LoadClusterDetail(context.Context, string) (domain.ClusterDetail, error) {
	f.called = true
	return f.detail, f.err
}

func TestQueryBackedClusterDetailRepositoryReturnsDetail(t *testing.T) {
	t.Parallel()

	loader := &fakeClusterDetailLoader{
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
			Evidence: []domain.Evidence{
				{Kind: domain.EvidenceClusterOverlap, Label: "cluster overlap", Source: "cluster-detail", Confidence: 0.9},
			},
		},
	}

	repo := NewQueryBackedClusterDetailRepository(loader)
	detail, err := repo.FindClusterDetail(context.Background(), "cluster_seed_whales")
	if err != nil {
		t.Fatalf("FindClusterDetail returned error: %v", err)
	}
	if !loader.called {
		t.Fatal("expected loader to be called")
	}
	if detail.ID != "cluster_seed_whales" {
		t.Fatalf("unexpected detail %#v", detail)
	}
}

func TestQueryBackedClusterDetailRepositoryReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := NewQueryBackedClusterDetailRepository(&fakeClusterDetailLoader{err: ErrClusterDetailNotFound})
	_, err := repo.FindClusterDetail(context.Background(), "cluster_missing")
	if !errors.Is(err, ErrClusterDetailNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}
