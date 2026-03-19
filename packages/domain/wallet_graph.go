package domain

import "fmt"

type WalletGraphNodeKind string

const (
	WalletGraphNodeWallet  WalletGraphNodeKind = "wallet"
	WalletGraphNodeCluster WalletGraphNodeKind = "cluster"
	WalletGraphNodeEntity  WalletGraphNodeKind = "entity"
)

type WalletGraphEdgeKind string

const (
	WalletGraphEdgeMemberOf       WalletGraphEdgeKind = "member_of"
	WalletGraphEdgeInteractedWith WalletGraphEdgeKind = "interacted_with"
	WalletGraphEdgeFundedBy       WalletGraphEdgeKind = "funded_by"
)

type WalletGraphNode struct {
	ID      string              `json:"id"`
	Kind    WalletGraphNodeKind `json:"kind"`
	Chain   Chain               `json:"chain,omitempty"`
	Address string              `json:"address,omitempty"`
	Label   string              `json:"label"`
}

type WalletGraphEdge struct {
	SourceID          string              `json:"sourceId"`
	TargetID          string              `json:"targetId"`
	Kind              WalletGraphEdgeKind `json:"kind"`
	ObservedAt        string              `json:"observedAt,omitempty"`
	Weight            int                 `json:"weight,omitempty"`
	CounterpartyCount int                 `json:"counterpartyCount,omitempty"`
}

type WalletGraph struct {
	Chain          Chain             `json:"chain"`
	Address        string            `json:"address"`
	DepthRequested int               `json:"depthRequested"`
	DepthResolved  int               `json:"depthResolved"`
	DensityCapped  bool              `json:"densityCapped"`
	Nodes          []WalletGraphNode `json:"nodes"`
	Edges          []WalletGraphEdge `json:"edges"`
}

func ValidateWalletGraph(graph WalletGraph) error {
	if graph.Chain == "" {
		return fmt.Errorf("chain is required")
	}
	if graph.Address == "" {
		return fmt.Errorf("address is required")
	}
	if graph.DepthResolved <= 0 {
		return fmt.Errorf("depth_resolved must be positive")
	}
	if len(graph.Nodes) == 0 {
		return fmt.Errorf("at least one node is required")
	}
	for _, node := range graph.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node id is required")
		}
		if node.Kind == "" {
			return fmt.Errorf("node kind is required")
		}
		if node.Label == "" {
			return fmt.Errorf("node label is required")
		}
	}
	for _, edge := range graph.Edges {
		if edge.SourceID == "" || edge.TargetID == "" {
			return fmt.Errorf("edge endpoints are required")
		}
		if edge.Kind == "" {
			return fmt.Errorf("edge kind is required")
		}
	}

	return nil
}
