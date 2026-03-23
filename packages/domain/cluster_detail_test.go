package domain

import "testing"

func TestValidateClusterDetail(t *testing.T) {
	t.Parallel()

	err := ValidateClusterDetail(ClusterDetail{
		ID:             "cluster_seed_whales",
		Label:          "Seed whales",
		ClusterType:    "whale",
		Score:          82,
		Classification: ClusterClassificationStrong,
		MemberCount:    2,
		Members: []ClusterMember{
			{Chain: ChainEVM, Address: "0x1234", Label: "Seed Whale"},
		},
		CommonActions: []ClusterCommonAction{
			{Kind: "shared_counterparty", Label: "Bridge wallet", SharedMemberCount: 2, InteractionCount: 11},
		},
		Evidence: []Evidence{
			{Kind: EvidenceClusterOverlap, Label: "cluster overlap", Source: "cluster-detail", Confidence: 0.9, ObservedAt: "2026-03-20T00:00:00Z"},
		},
	})
	if err != nil {
		t.Fatalf("expected valid cluster detail, got %v", err)
	}
}

func TestClassifyClusterScore(t *testing.T) {
	t.Parallel()

	if got := ClassifyClusterScore(84); got != ClusterClassificationStrong {
		t.Fatalf("unexpected strong classification %q", got)
	}
	if got := ClassifyClusterScore(60); got != ClusterClassificationEmerging {
		t.Fatalf("unexpected emerging classification %q", got)
	}
	if got := ClassifyClusterScore(22); got != ClusterClassificationWeak {
		t.Fatalf("unexpected weak classification %q", got)
	}
}
