package db

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/flowintel/flowintel/packages/domain"
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
       firstObservedAt: toString(coalesce(interaction.firstObservedAt, interaction.lastObservedAt)),
       lastObservedAt: toString(coalesce(interaction.lastObservedAt, interaction.firstObservedAt)),
       interactionCount: toInteger(coalesce(interaction.interactionCount, interaction.counterpartyCount, 0)),
       inboundCount: toInteger(coalesce(interaction.inboundCount, 0)),
       outboundCount: toInteger(coalesce(interaction.outboundCount, 0)),
       lastTxHash: coalesce(interaction.lastTxHash, ''),
       lastDirection: coalesce(interaction.lastDirection, ''),
       lastProvider: coalesce(interaction.lastProvider, '')
     }) AS interactions
OPTIONAL MATCH (funder:Wallet)-[funding:FUNDED_BY]->(root)
WITH root,
     clusters,
     interactions,
     collect(DISTINCT {
       id: funder.id,
       chain: funder.chain,
       address: funder.address,
       label: coalesce(funder.displayName, funder.address),
       firstObservedAt: toString(coalesce(funding.firstObservedAt, funding.lastObservedAt)),
       lastObservedAt: toString(coalesce(funding.lastObservedAt, funding.firstObservedAt)),
       interactionCount: toInteger(coalesce(funding.interactionCount, funding.counterpartyCount, 0)),
       inboundCount: toInteger(coalesce(funding.inboundCount, 0)),
       outboundCount: toInteger(coalesce(funding.outboundCount, 0)),
       lastTxHash: coalesce(funding.lastTxHash, ''),
       lastDirection: coalesce(funding.lastDirection, ''),
       lastProvider: coalesce(funding.lastProvider, '')
     }) AS funders
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
[item IN funders WHERE item.id IS NOT NULL | item][..$maxCounterparties] AS funders,
size([item IN interactions WHERE item.id IS NOT NULL | item]) > $maxCounterparties OR size([item IN funders WHERE item.id IS NOT NULL | item]) > $maxCounterparties AS densityCapped
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
		buildWalletGraphNode(root, domain.WalletGraphNodeWallet),
	}
	nodeIDs := map[string]struct{}{
		nodes[0].ID: {},
	}
	edges := make([]domain.WalletGraphEdge, 0, 8)

	for _, cluster := range sliceMapValue(values, "clusters") {
		clusterID := stringValue(cluster, "id")
		if clusterID == "" {
			continue
		}

		nodes = appendWalletGraphNode(nodes, nodeIDs, domain.WalletGraphNode{
			ID:    clusterID,
			Kind:  domain.WalletGraphNodeCluster,
			Label: firstNonEmpty(stringValue(cluster, "label"), stringValue(cluster, "clusterKey"), clusterID),
		})
		edges = append(edges, domain.WalletGraphEdge{
			SourceID:       nodes[0].ID,
			TargetID:       clusterID,
			Kind:           domain.WalletGraphEdgeMemberOf,
			Family:         domain.WalletGraphEdgeFamilyForKind(domain.WalletGraphEdgeMemberOf),
			Directionality: domain.WalletGraphEdgeDirectionalityForKind(domain.WalletGraphEdgeMemberOf, 0, 0, ""),
		})
	}

	for _, interaction := range sliceMapValue(values, "interactions") {
		record, ok := buildWalletGraphInteractionRecord(interaction)
		if !ok {
			continue
		}

		counterpartyID := record.ID
		if counterpartyID == "" {
			continue
		}

		nodes = appendWalletGraphNode(nodes, nodeIDs, domain.WalletGraphNode{
			ID:      counterpartyID,
			Kind:    domain.WalletGraphNodeWallet,
			Chain:   domain.Chain(record.Chain),
			Address: record.Address,
			Label:   firstNonEmpty(record.Label, record.Address, counterpartyID),
		})
		edges = append(edges, domain.WalletGraphEdge{
			SourceID:          nodes[0].ID,
			TargetID:          counterpartyID,
			Kind:              domain.WalletGraphEdgeInteractedWith,
			Family:            domain.WalletGraphEdgeFamilyForKind(domain.WalletGraphEdgeInteractedWith),
			Directionality:    domain.WalletGraphEdgeDirectionalityForKind(domain.WalletGraphEdgeInteractedWith, record.InboundCount, record.OutboundCount, record.LastDirection),
			FirstObservedAt:   record.FirstObservedAt.Format(time.RFC3339),
			ObservedAt:        record.LastObservedAt.Format(time.RFC3339),
			Weight:            record.InteractionCount,
			CounterpartyCount: record.InteractionCount,
			Evidence:          buildWalletGraphEdgeEvidence(domain.WalletGraphEdgeInteractedWith, record),
		})
	}

	for _, funding := range sliceMapValue(values, "funders") {
		record, ok := buildWalletGraphInteractionRecord(funding)
		if !ok {
			continue
		}

		funderID := record.ID
		if funderID == "" {
			continue
		}

		nodes = appendWalletGraphNode(nodes, nodeIDs, domain.WalletGraphNode{
			ID:      funderID,
			Kind:    domain.WalletGraphNodeWallet,
			Chain:   domain.Chain(record.Chain),
			Address: record.Address,
			Label:   firstNonEmpty(record.Label, record.Address, funderID),
		})
		edges = append(edges, domain.WalletGraphEdge{
			SourceID:          funderID,
			TargetID:          nodes[0].ID,
			Kind:              domain.WalletGraphEdgeFundedBy,
			Family:            domain.WalletGraphEdgeFamilyForKind(domain.WalletGraphEdgeFundedBy),
			Directionality:    domain.WalletGraphEdgeDirectionalityForKind(domain.WalletGraphEdgeFundedBy, record.InboundCount, record.OutboundCount, record.LastDirection),
			FirstObservedAt:   record.FirstObservedAt.Format(time.RFC3339),
			ObservedAt:        record.LastObservedAt.Format(time.RFC3339),
			Weight:            record.InteractionCount,
			CounterpartyCount: record.InteractionCount,
			Evidence:          buildWalletGraphEdgeEvidence(domain.WalletGraphEdgeFundedBy, record),
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

func buildWalletGraphNode(values map[string]any, kind domain.WalletGraphNodeKind) domain.WalletGraphNode {
	return domain.WalletGraphNode{
		ID:      stringValue(values, "id"),
		Kind:    kind,
		Chain:   domain.Chain(stringValue(values, "chain")),
		Address: stringValue(values, "address"),
		Label:   stringValue(values, "label"),
	}
}

func appendWalletGraphNode(
	nodes []domain.WalletGraphNode,
	nodeIDs map[string]struct{},
	node domain.WalletGraphNode,
) []domain.WalletGraphNode {
	if node.ID == "" {
		return nodes
	}
	if _, exists := nodeIDs[node.ID]; exists {
		return nodes
	}
	nodeIDs[node.ID] = struct{}{}
	return append(nodes, node)
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

type walletGraphInteractionRecord struct {
	ID               string
	Chain            string
	Address          string
	Label            string
	FirstObservedAt  time.Time
	LastObservedAt   time.Time
	InteractionCount int
	InboundCount     int
	OutboundCount    int
	LastTxHash       string
	LastDirection    string
	LastProvider     string
}

func buildWalletGraphInteractionRecord(interaction map[string]any) (walletGraphInteractionRecord, bool) {
	record := walletGraphInteractionRecord{
		ID:               stringValue(interaction, "id"),
		Chain:            stringValue(interaction, "chain"),
		Address:          stringValue(interaction, "address"),
		Label:            stringValue(interaction, "label"),
		InteractionCount: int(int64Value(interaction, "interactionCount")),
		InboundCount:     int(int64Value(interaction, "inboundCount")),
		OutboundCount:    int(int64Value(interaction, "outboundCount")),
		LastTxHash:       stringValue(interaction, "lastTxHash"),
		LastDirection:    stringValue(interaction, "lastDirection"),
		LastProvider:     stringValue(interaction, "lastProvider"),
	}
	if record.ID == "" {
		return walletGraphInteractionRecord{}, false
	}
	if record.InteractionCount <= 0 {
		record.InteractionCount = int(int64Value(interaction, "counterpartyCount"))
	}

	record.FirstObservedAt = parseWalletGraphTimeValue(stringValue(interaction, "firstObservedAt"))
	if record.FirstObservedAt.IsZero() {
		record.FirstObservedAt = parseWalletGraphTimeValue(stringValue(interaction, "lastObservedAt"))
	}
	record.LastObservedAt = parseWalletGraphTimeValue(stringValue(interaction, "lastObservedAt"))
	if record.LastObservedAt.IsZero() {
		record.LastObservedAt = record.FirstObservedAt
	}

	return record, true
}

func buildWalletGraphEdgeEvidence(
	kind domain.WalletGraphEdgeKind,
	record walletGraphInteractionRecord,
) *domain.WalletGraphEdgeEvidence {
	return &domain.WalletGraphEdgeEvidence{
		Source:        "neo4j-materialized",
		Confidence:    deriveWalletGraphEvidenceConfidence(kind, record.InteractionCount),
		Summary:       deriveWalletGraphEvidenceSummary(kind, record),
		LastTxHash:    record.LastTxHash,
		LastDirection: record.LastDirection,
		LastProvider:  record.LastProvider,
	}
}

func deriveWalletGraphEvidenceConfidence(
	kind domain.WalletGraphEdgeKind,
	interactionCount int,
) string {
	if interactionCount <= 0 {
		return "low"
	}

	if kind == domain.WalletGraphEdgeFundedBy {
		if interactionCount >= 3 {
			return "high"
		}
		return "medium"
	}

	if interactionCount >= 5 {
		return "high"
	}
	if interactionCount >= 2 {
		return "medium"
	}
	return "low"
}

func deriveWalletGraphEvidenceSummary(
	kind domain.WalletGraphEdgeKind,
	record walletGraphInteractionRecord,
) string {
	interactionCount := record.InteractionCount
	switch kind {
	case domain.WalletGraphEdgeFundedBy:
		if interactionCount == 1 {
			return "Observed inbound funding via 1 transfer."
		}
		return fmt.Sprintf("Observed inbound funding via %d transfers.", interactionCount)
	case domain.WalletGraphEdgeInteractedWith:
		switch {
		case record.InboundCount > 0 && record.OutboundCount > 0:
			return fmt.Sprintf(
				"Observed transfer activity in both directions (IN %d · OUT %d).",
				record.InboundCount,
				record.OutboundCount,
			)
		case record.OutboundCount > 0:
			if record.OutboundCount == 1 {
				return "Observed 1 outbound transfer to this counterparty."
			}
			return fmt.Sprintf("Observed %d outbound transfers to this counterparty.", record.OutboundCount)
		case record.InboundCount > 0:
			if record.InboundCount == 1 {
				return "Observed 1 inbound transfer from this counterparty."
			}
			return fmt.Sprintf("Observed %d inbound transfers from this counterparty.", record.InboundCount)
		case interactionCount == 1:
			return "Observed 1 direct transfer between these wallets."
		}
		return fmt.Sprintf("Observed %d direct transfers between these wallets.", interactionCount)
	default:
		return "Observed relationship metadata is available."
	}
}

func parseWalletGraphTimeValue(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}
	}

	return parsed.UTC()
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
	case domain.WalletGraphEdgeEntityLinked:
		return 1
	case domain.WalletGraphEdgeFundedBy:
		return 2
	case domain.WalletGraphEdgeInteractedWith:
		return 3
	default:
		return 4
	}
}
