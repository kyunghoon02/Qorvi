package domain

import "fmt"

type ClusterClassification string

const (
	ClusterClassificationWeak     ClusterClassification = "weak"
	ClusterClassificationEmerging ClusterClassification = "emerging"
	ClusterClassificationStrong   ClusterClassification = "strong"
)

type ClusterMember struct {
	Chain   Chain  `json:"chain"`
	Address string `json:"address"`
	Label   string `json:"label"`
}

type ClusterCommonAction struct {
	Kind              string `json:"kind"`
	Label             string `json:"label"`
	Chain             Chain  `json:"chain,omitempty"`
	Address           string `json:"address,omitempty"`
	SharedMemberCount int    `json:"shared_member_count"`
	InteractionCount  int    `json:"interaction_count"`
	ObservedAt        string `json:"observed_at,omitempty"`
}

type ClusterDetail struct {
	ID             string                `json:"id"`
	Label          string                `json:"label"`
	ClusterType    string                `json:"cluster_type"`
	Score          int                   `json:"score"`
	Classification ClusterClassification `json:"classification"`
	MemberCount    int                   `json:"member_count"`
	Members        []ClusterMember       `json:"members"`
	CommonActions  []ClusterCommonAction `json:"common_actions"`
	Evidence       []Evidence            `json:"evidence"`
}

func ValidateClusterDetail(detail ClusterDetail) error {
	if detail.ID == "" {
		return fmt.Errorf("cluster id is required")
	}
	if detail.Label == "" {
		return fmt.Errorf("cluster label is required")
	}
	if detail.MemberCount < 0 {
		return fmt.Errorf("member_count must be non-negative")
	}
	if len(detail.Members) == 0 {
		return fmt.Errorf("at least one cluster member is required")
	}
	if len(detail.Evidence) == 0 {
		return fmt.Errorf("cluster evidence is required")
	}
	return nil
}

func ClassifyClusterScore(score int) ClusterClassification {
	switch {
	case score >= 80:
		return ClusterClassificationStrong
	case score >= 50:
		return ClusterClassificationEmerging
	default:
		return ClusterClassificationWeak
	}
}
