package domain

import "fmt"

type WalletGraphNodeKind string

const (
	WalletGraphNodeWallet  WalletGraphNodeKind = "wallet"
	WalletGraphNodeCluster WalletGraphNodeKind = "cluster"
	WalletGraphNodeEntity  WalletGraphNodeKind = "entity"
)

type WalletGraphEdgeKind string
type WalletGraphEdgeFamily string
type WalletGraphEdgeDirectionality string

const (
	WalletGraphEdgeMemberOf       WalletGraphEdgeKind = "member_of"
	WalletGraphEdgeInteractedWith WalletGraphEdgeKind = "interacted_with"
	WalletGraphEdgeFundedBy       WalletGraphEdgeKind = "funded_by"
	WalletGraphEdgeEntityLinked   WalletGraphEdgeKind = "entity_linked"
)

const (
	WalletGraphEdgeFamilyBase    WalletGraphEdgeFamily = "base"
	WalletGraphEdgeFamilyDerived WalletGraphEdgeFamily = "derived"
)

const (
	WalletGraphEdgeDirectionalityLinked   WalletGraphEdgeDirectionality = "linked"
	WalletGraphEdgeDirectionalitySent     WalletGraphEdgeDirectionality = "sent"
	WalletGraphEdgeDirectionalityReceived WalletGraphEdgeDirectionality = "received"
	WalletGraphEdgeDirectionalityMixed    WalletGraphEdgeDirectionality = "mixed"
)

type WalletGraphNode struct {
	ID      string              `json:"id"`
	Kind    WalletGraphNodeKind `json:"kind"`
	Chain   Chain               `json:"chain,omitempty"`
	Address string              `json:"address,omitempty"`
	Label   string              `json:"label"`
	Labels  WalletLabelSet      `json:"labels,omitempty"`
}

type WalletGraphEdgeEvidence struct {
	Source        string `json:"source"`
	Confidence    string `json:"confidence"`
	Summary       string `json:"summary"`
	LastTxHash    string `json:"lastTxHash,omitempty"`
	LastDirection string `json:"lastDirection,omitempty"`
	LastProvider  string `json:"lastProvider,omitempty"`
}

type WalletGraphEdgeTokenBreakdown struct {
	Symbol         string `json:"symbol"`
	InboundAmount  string `json:"inboundAmount,omitempty"`
	OutboundAmount string `json:"outboundAmount,omitempty"`
}

type WalletGraphEdgeTokenFlow struct {
	PrimaryToken   string                          `json:"primaryToken,omitempty"`
	InboundCount   int                             `json:"inboundCount,omitempty"`
	OutboundCount  int                             `json:"outboundCount,omitempty"`
	InboundAmount  string                          `json:"inboundAmount,omitempty"`
	OutboundAmount string                          `json:"outboundAmount,omitempty"`
	Breakdowns     []WalletGraphEdgeTokenBreakdown `json:"breakdowns,omitempty"`
}

type WalletGraphEdge struct {
	SourceID          string                        `json:"sourceId"`
	TargetID          string                        `json:"targetId"`
	Kind              WalletGraphEdgeKind           `json:"kind"`
	Family            WalletGraphEdgeFamily         `json:"family,omitempty"`
	Directionality    WalletGraphEdgeDirectionality `json:"directionality,omitempty"`
	FirstObservedAt   string                        `json:"firstObservedAt,omitempty"`
	ObservedAt        string                        `json:"observedAt,omitempty"`
	Weight            int                           `json:"weight,omitempty"`
	CounterpartyCount int                           `json:"counterpartyCount,omitempty"`
	Evidence          *WalletGraphEdgeEvidence      `json:"evidence,omitempty"`
	TokenFlow         *WalletGraphEdgeTokenFlow     `json:"tokenFlow,omitempty"`
}

func WalletGraphEdgeFamilyForKind(kind WalletGraphEdgeKind) WalletGraphEdgeFamily {
	switch kind {
	case WalletGraphEdgeInteractedWith:
		return WalletGraphEdgeFamilyBase
	case WalletGraphEdgeMemberOf, WalletGraphEdgeFundedBy, WalletGraphEdgeEntityLinked:
		return WalletGraphEdgeFamilyDerived
	default:
		return WalletGraphEdgeFamilyDerived
	}
}

func WalletGraphEdgeDirectionalityForKind(
	kind WalletGraphEdgeKind,
	inboundCount int,
	outboundCount int,
	lastDirection string,
) WalletGraphEdgeDirectionality {
	switch kind {
	case WalletGraphEdgeFundedBy:
		return WalletGraphEdgeDirectionalityReceived
	case WalletGraphEdgeInteractedWith:
		switch {
		case inboundCount > 0 && outboundCount > 0:
			return WalletGraphEdgeDirectionalityMixed
		case outboundCount > 0:
			return WalletGraphEdgeDirectionalitySent
		case inboundCount > 0:
			return WalletGraphEdgeDirectionalityReceived
		case lastDirection == "outbound":
			return WalletGraphEdgeDirectionalitySent
		case lastDirection == "inbound":
			return WalletGraphEdgeDirectionalityReceived
		default:
			return WalletGraphEdgeDirectionalityMixed
		}
	default:
		return WalletGraphEdgeDirectionalityLinked
	}
}

type WalletGraphNeighborhoodSummary struct {
	NeighborNodeCount      int    `json:"neighborNodeCount"`
	WalletNodeCount        int    `json:"walletNodeCount"`
	ClusterNodeCount       int    `json:"clusterNodeCount"`
	EntityNodeCount        int    `json:"entityNodeCount"`
	InteractionEdgeCount   int    `json:"interactionEdgeCount"`
	TotalInteractionWeight int    `json:"totalInteractionWeight"`
	LatestObservedAt       string `json:"latestObservedAt,omitempty"`
}

type WalletGraphSnapshot struct {
	Key           string `json:"key"`
	Source        string `json:"source"`
	GeneratedAt   string `json:"generatedAt"`
	MaxAgeSeconds int    `json:"maxAgeSeconds"`
}

type WalletGraph struct {
	Chain               Chain                           `json:"chain"`
	Address             string                          `json:"address"`
	DepthRequested      int                             `json:"depthRequested"`
	DepthResolved       int                             `json:"depthResolved"`
	DensityCapped       bool                            `json:"densityCapped"`
	Snapshot            *WalletGraphSnapshot            `json:"snapshot,omitempty"`
	NeighborhoodSummary *WalletGraphNeighborhoodSummary `json:"neighborhoodSummary,omitempty"`
	Nodes               []WalletGraphNode               `json:"nodes"`
	Edges               []WalletGraphEdge               `json:"edges"`
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

func BuildWalletGraphNeighborhoodSummary(
	graph WalletGraph,
) WalletGraphNeighborhoodSummary {
	summary := WalletGraphNeighborhoodSummary{}

	for index, node := range graph.Nodes {
		if index > 0 {
			summary.NeighborNodeCount++
		}

		switch node.Kind {
		case WalletGraphNodeWallet:
			summary.WalletNodeCount++
		case WalletGraphNodeCluster:
			summary.ClusterNodeCount++
		case WalletGraphNodeEntity:
			summary.EntityNodeCount++
		}
	}

	for _, edge := range graph.Edges {
		if edge.Kind == WalletGraphEdgeInteractedWith {
			summary.InteractionEdgeCount++
			if edge.Weight > 0 {
				summary.TotalInteractionWeight += edge.Weight
			} else if edge.CounterpartyCount > 0 {
				summary.TotalInteractionWeight += edge.CounterpartyCount
			}
		}

		observedAt := edge.ObservedAt
		if observedAt == "" {
			observedAt = edge.FirstObservedAt
		}
		if observedAt > summary.LatestObservedAt {
			summary.LatestObservedAt = observedAt
		}
	}

	return summary
}
