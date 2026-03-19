package db

import (
	"context"
	"fmt"
	"sort"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/whalegraph/whalegraph/packages/domain"
)

const walletGraphCypher = `
MATCH (root:Wallet {chain: $chain, address: $address})
OPTIONAL MATCH (root)-[:MEMBER_OF]->(cluster:Cluster)
WITH root, collect(DISTINCT cluster) AS clusters
OPTIONAL MATCH (root)-[interaction:INTERACTED_WITH]->(counterparty:Wallet)
WITH root,
     clusters,
     collect(DISTINCT {
       id: counterparty.id,
       chain: counterparty.chain,
       address: counterparty.address,
       label: coalesce(counterparty.displayName, counterparty.address),
       observedAt: toString(interaction.lastObservedAt),
       counterpartyCount: toInteger(coalesce(interaction.counterpartyCount, 0))
     }) AS interactions
RETURN {
  id: root.id,
  chain: root.chain,
  address: root.address,
  label: coalesce(root.displayName, root.address)
} AS root,
[cluster IN clusters WHERE cluster IS NOT NULL | {
  id: cluster.id,
  clusterKey: coalesce(cluster.clusterKey, ''),
  label: coalesce(cluster.clusterKey, cluster.id)
}] AS clusters,
[item IN interactions WHERE item.id IS NOT NULL | item][..$maxCounterparties] AS interactions,
size([item IN interactions WHERE item.id IS NOT NULL | item]) > $maxCounterparties AS densityCapped
LIMIT 1
`

type Neo4jWalletGraphReader struct {
	Driver   Neo4jDriver
	Database string
}

func NewNeo4jWalletGraphReader(driver Neo4jDriver, database string) *Neo4jWalletGraphReader {
	return &Neo4jWalletGraphReader{
		Driver:   driver,
		Database: database,
	}
}

func (r *Neo4jWalletGraphReader) ReadWalletGraph(
	ctx context.Context,
	query WalletGraphQuery,
) (domain.WalletGraph, error) {
	if r == nil || r.Driver == nil {
		return domain.WalletGraph{}, fmt.Errorf("neo4j wallet graph reader is nil")
	}

	session := r.Driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: r.Database})
	defer func() {
		_ = session.Close(ctx)
	}()

	result, err := session.Run(ctx, walletGraphCypher, map[string]any{
		"chain":             string(query.Ref.Chain),
		"address":           query.Ref.Address,
		"maxCounterparties": query.MaxCounterparties,
	})
	if err != nil {
		return domain.WalletGraph{}, fmt.Errorf("run neo4j wallet graph query: %w", err)
	}

	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return domain.WalletGraph{}, fmt.Errorf("neo4j wallet graph result error: %w", err)
		}
		return domain.WalletGraph{}, ErrWalletGraphNotFound
	}

	record := result.Record()
	if record == nil {
		return domain.WalletGraph{}, ErrWalletGraphNotFound
	}

	values := record.AsMap()
	root := mapValue(values, "root")
	if len(root) == 0 {
		return domain.WalletGraph{}, ErrWalletGraphNotFound
	}

	nodes := []domain.WalletGraphNode{
		{
			ID:      stringValue(root, "id"),
			Kind:    domain.WalletGraphNodeWallet,
			Chain:   domain.Chain(stringValue(root, "chain")),
			Address: stringValue(root, "address"),
			Label:   stringValue(root, "label"),
		},
	}
	edges := make([]domain.WalletGraphEdge, 0, 8)

	for _, cluster := range sliceMapValue(values, "clusters") {
		clusterID := stringValue(cluster, "id")
		if clusterID == "" {
			continue
		}

		nodes = append(nodes, domain.WalletGraphNode{
			ID:    clusterID,
			Kind:  domain.WalletGraphNodeCluster,
			Label: firstNonEmpty(stringValue(cluster, "label"), stringValue(cluster, "clusterKey"), clusterID),
		})
		edges = append(edges, domain.WalletGraphEdge{
			SourceID: nodes[0].ID,
			TargetID: clusterID,
			Kind:     domain.WalletGraphEdgeMemberOf,
		})
	}

	for _, interaction := range sliceMapValue(values, "interactions") {
		counterpartyID := stringValue(interaction, "id")
		if counterpartyID == "" {
			continue
		}

		nodes = append(nodes, domain.WalletGraphNode{
			ID:      counterpartyID,
			Kind:    domain.WalletGraphNodeWallet,
			Chain:   domain.Chain(stringValue(interaction, "chain")),
			Address: stringValue(interaction, "address"),
			Label:   firstNonEmpty(stringValue(interaction, "label"), stringValue(interaction, "address"), counterpartyID),
		})
		edges = append(edges, domain.WalletGraphEdge{
			SourceID:          nodes[0].ID,
			TargetID:          counterpartyID,
			Kind:              domain.WalletGraphEdgeInteractedWith,
			ObservedAt:        stringValue(interaction, "observedAt"),
			Weight:            int(int64Value(interaction, "counterpartyCount")),
			CounterpartyCount: int(int64Value(interaction, "counterpartyCount")),
		})
	}

	graph := domain.WalletGraph{
		Chain:          query.Ref.Chain,
		Address:        query.Ref.Address,
		DepthRequested: query.DepthRequested,
		DepthResolved:  query.DepthResolved,
		DensityCapped:  boolValue(values, "densityCapped"),
		Nodes:          nodes,
		Edges:          edges,
	}

	normalizeWalletGraph(&graph)

	if err := domain.ValidateWalletGraph(graph); err != nil {
		return domain.WalletGraph{}, fmt.Errorf("validate wallet graph: %w", err)
	}

	return graph, nil
}

func mapValue(values map[string]any, key string) map[string]any {
	value, ok := values[key].(map[string]any)
	if ok {
		return value
	}

	return map[string]any{}
}

func sliceMapValue(values map[string]any, key string) []map[string]any {
	raw, ok := values[key].([]any)
	if !ok {
		return nil
	}

	items := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		mapped, ok := item.(map[string]any)
		if ok {
			items = append(items, mapped)
		}
	}

	return items
}

func boolValue(values map[string]any, key string) bool {
	value := values[key]
	typed, ok := value.(bool)
	return ok && typed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func normalizeWalletGraph(graph *domain.WalletGraph) {
	if graph == nil {
		return
	}

	if len(graph.Nodes) > 1 {
		root := graph.Nodes[0]
		rest := append([]domain.WalletGraphNode(nil), graph.Nodes[1:]...)
		sort.SliceStable(rest, func(i, j int) bool {
			left := rest[i]
			right := rest[j]

			leftRank := walletGraphNodeKindRank(left.Kind)
			rightRank := walletGraphNodeKindRank(right.Kind)
			if leftRank != rightRank {
				return leftRank < rightRank
			}
			if left.Label != right.Label {
				return left.Label < right.Label
			}
			return left.ID < right.ID
		})
		graph.Nodes = append([]domain.WalletGraphNode{root}, rest...)
	}

	if len(graph.Edges) > 1 {
		sort.SliceStable(graph.Edges, func(i, j int) bool {
			left := graph.Edges[i]
			right := graph.Edges[j]

			leftRank := walletGraphEdgeKindRank(left.Kind)
			rightRank := walletGraphEdgeKindRank(right.Kind)
			if leftRank != rightRank {
				return leftRank < rightRank
			}
			if left.SourceID != right.SourceID {
				return left.SourceID < right.SourceID
			}
			if left.TargetID != right.TargetID {
				return left.TargetID < right.TargetID
			}
			if left.ObservedAt != right.ObservedAt {
				return left.ObservedAt < right.ObservedAt
			}
			if left.Weight != right.Weight {
				return left.Weight < right.Weight
			}
			return left.CounterpartyCount < right.CounterpartyCount
		})
	}
}

func walletGraphNodeKindRank(kind domain.WalletGraphNodeKind) int {
	switch kind {
	case domain.WalletGraphNodeCluster:
		return 0
	case domain.WalletGraphNodeEntity:
		return 1
	case domain.WalletGraphNodeWallet:
		return 2
	default:
		return 3
	}
}

func walletGraphEdgeKindRank(kind domain.WalletGraphEdgeKind) int {
	switch kind {
	case domain.WalletGraphEdgeMemberOf:
		return 0
	case domain.WalletGraphEdgeFundedBy:
		return 1
	case domain.WalletGraphEdgeInteractedWith:
		return 2
	default:
		return 3
	}
}
