package db

import (
	"context"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/flowintel/flowintel/packages/domain"
)

func TestNeo4jClusterDetailReader(t *testing.T) {
	t.Parallel()

	rec := &neo4j.Record{
		Keys: []string{"clusterID", "label", "clusterType", "clusterScore", "memberCount", "members", "commonActions"},
		Values: []any{
			"cluster_seed_whales",
			"cluster_seed_whales",
			"whale",
			int64(82),
			int64(2),
			[]any{
				map[string]any{"chain": "evm", "address": "0xaaa", "label": "Seed A"},
				map[string]any{"chain": "evm", "address": "0xbbb", "label": "Seed B"},
			},
			[]any{
				map[string]any{
					"kind":              "shared_counterparty",
					"label":             "Bridge wallet",
					"chain":             "evm",
					"address":           "0xccc",
					"sharedMemberCount": int64(2),
					"interactionCount":  int64(11),
					"observedAt":        "2026-03-20T00:00:00Z",
				},
			},
		},
	}

	driver := fakeNeo4jDriver{session: fakeNeo4jSession{result: fakeNeo4jResult{record: rec}}}
	reader := NewNeo4jClusterDetailReader(driver, "neo4j")
	detail, err := reader.ReadClusterDetail(context.Background(), ClusterDetailQuery{
		ClusterID:   "cluster_seed_whales",
		MemberLimit: 8,
		ActionLimit: 5,
	})
	if err != nil {
		t.Fatalf("ReadClusterDetail returned error: %v", err)
	}
	if detail.ID != "cluster_seed_whales" {
		t.Fatalf("unexpected cluster id %q", detail.ID)
	}
	if detail.Classification != domain.ClusterClassificationStrong {
		t.Fatalf("unexpected classification %q", detail.Classification)
	}
	if len(detail.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(detail.Members))
	}
	if len(detail.CommonActions) != 1 || detail.CommonActions[0].Label != "Bridge wallet" {
		t.Fatalf("unexpected common actions %#v", detail.CommonActions)
	}
	if len(detail.Evidence) != 2 {
		t.Fatalf("expected 2 evidence items, got %#v", detail.Evidence)
	}
}

func TestBuildClusterDetailQuery(t *testing.T) {
	t.Parallel()

	query, err := BuildClusterDetailQuery(" cluster_seed_whales ", 0, 0)
	if err != nil {
		t.Fatalf("BuildClusterDetailQuery returned error: %v", err)
	}
	if query.ClusterID != "cluster_seed_whales" {
		t.Fatalf("unexpected cluster id %q", query.ClusterID)
	}
	if query.MemberLimit != 8 || query.ActionLimit != 5 {
		t.Fatalf("unexpected limits %#v", query)
	}
}
