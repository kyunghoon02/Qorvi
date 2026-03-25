package db

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/flowintel/flowintel/packages/domain"
)

const clusterDetailCypher = `
MATCH (c:Cluster {clusterKey: $clusterKey})
CALL {
  WITH c
  MATCH (member:Wallet)-[:MEMBER_OF]->(c)
  WITH member
  ORDER BY coalesce(member.displayName, member.address, member.id) ASC
  RETURN collect(DISTINCT {
    chain: coalesce(member.chain, ''),
    address: coalesce(member.address, ''),
    label: coalesce(member.displayName, member.address, member.id)
  })[..$memberLimit] AS members,
  count(DISTINCT member) AS memberCount
}
CALL {
  WITH c
  MATCH (member:Wallet)-[:MEMBER_OF]->(c)
  MATCH (member)-[interaction:INTERACTED_WITH]->(counterparty:Wallet)
  WITH counterparty,
       count(DISTINCT member) AS sharedMemberCount,
       sum(toInteger(coalesce(interaction.interactionCount, interaction.counterpartyCount, 1))) AS interactionCount,
       max(toString(coalesce(interaction.lastObservedAt, interaction.firstObservedAt))) AS observedAt
  ORDER BY sharedMemberCount DESC, interactionCount DESC, coalesce(counterparty.displayName, counterparty.address, counterparty.id) ASC
  RETURN collect({
    kind: 'shared_counterparty',
    label: coalesce(counterparty.displayName, counterparty.address, counterparty.id),
    chain: coalesce(counterparty.chain, ''),
    address: coalesce(counterparty.address, ''),
    sharedMemberCount: toInteger(sharedMemberCount),
    interactionCount: toInteger(interactionCount),
    observedAt: coalesce(observedAt, '')
  })[..$actionLimit] AS commonActions
}
RETURN
  coalesce(c.clusterKey, c.id) AS clusterID,
  coalesce(c.clusterKey, c.id) AS label,
  coalesce(c.clusterType, '') AS clusterType,
  toInteger(coalesce(c.clusterScore, 0)) AS clusterScore,
  toInteger(memberCount) AS memberCount,
  members,
  commonActions
LIMIT 1
`

type ClusterDetailQuery struct {
	ClusterID   string
	MemberLimit int
	ActionLimit int
}

type ClusterDetailReader interface {
	ReadClusterDetail(context.Context, ClusterDetailQuery) (domain.ClusterDetail, error)
}

type ClusterDetailRepository struct {
	Reader ClusterDetailReader
}

type Neo4jClusterDetailReader struct {
	Driver   Neo4jDriver
	Database string
}

var ErrClusterDetailNotFound = fmt.Errorf("cluster detail not found")

func NewClusterDetailRepository(reader ClusterDetailReader) *ClusterDetailRepository {
	return &ClusterDetailRepository{Reader: reader}
}

func NewNeo4jClusterDetailReader(driver Neo4jDriver, database string) *Neo4jClusterDetailReader {
	return &Neo4jClusterDetailReader{Driver: driver, Database: database}
}

func NewClusterDetailRepositoryFromClients(clients *StorageClients) *ClusterDetailRepository {
	if clients == nil {
		return nil
	}
	return NewClusterDetailRepository(NewNeo4jClusterDetailReader(clients.Neo4j, "neo4j"))
}

func BuildClusterDetailQuery(clusterID string, memberLimit int, actionLimit int) (ClusterDetailQuery, error) {
	normalizedID := strings.TrimSpace(clusterID)
	if normalizedID == "" {
		return ClusterDetailQuery{}, fmt.Errorf("cluster id is required")
	}
	if memberLimit <= 0 {
		memberLimit = 8
	}
	if actionLimit <= 0 {
		actionLimit = 5
	}
	return ClusterDetailQuery{
		ClusterID:   normalizedID,
		MemberLimit: memberLimit,
		ActionLimit: actionLimit,
	}, nil
}

func (r *ClusterDetailRepository) LoadClusterDetail(
	ctx context.Context,
	clusterID string,
) (domain.ClusterDetail, error) {
	if r == nil || r.Reader == nil {
		return domain.ClusterDetail{}, ErrClusterDetailNotFound
	}
	query, err := BuildClusterDetailQuery(clusterID, 8, 5)
	if err != nil {
		return domain.ClusterDetail{}, err
	}
	detail, err := r.Reader.ReadClusterDetail(ctx, query)
	if err != nil {
		return domain.ClusterDetail{}, err
	}
	return detail, nil
}

func (r *Neo4jClusterDetailReader) ReadClusterDetail(
	ctx context.Context,
	query ClusterDetailQuery,
) (domain.ClusterDetail, error) {
	if r == nil || r.Driver == nil {
		return domain.ClusterDetail{}, fmt.Errorf("neo4j cluster detail reader is nil")
	}

	session := r.Driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: r.Database})
	defer func() {
		_ = session.Close(ctx)
	}()

	result, err := session.Run(ctx, clusterDetailCypher, map[string]any{
		"clusterKey":  query.ClusterID,
		"memberLimit": query.MemberLimit,
		"actionLimit": query.ActionLimit,
	})
	if err != nil {
		return domain.ClusterDetail{}, fmt.Errorf("run neo4j cluster detail query: %w", err)
	}
	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return domain.ClusterDetail{}, fmt.Errorf("neo4j cluster detail result error: %w", err)
		}
		return domain.ClusterDetail{}, ErrClusterDetailNotFound
	}

	record := result.Record()
	if record == nil {
		return domain.ClusterDetail{}, ErrClusterDetailNotFound
	}
	values := record.AsMap()

	detail := domain.ClusterDetail{
		ID:             stringValue(values, "clusterID"),
		Label:          stringValue(values, "label"),
		ClusterType:    stringValue(values, "clusterType"),
		Score:          int(int64Value(values, "clusterScore")),
		Classification: domain.ClassifyClusterScore(int(int64Value(values, "clusterScore"))),
		MemberCount:    int(int64Value(values, "memberCount")),
		Members:        buildClusterMembers(sliceMapValue(values, "members")),
		CommonActions:  buildClusterCommonActions(sliceMapValue(values, "commonActions")),
	}
	detail.Evidence = buildClusterDetailEvidence(detail)

	normalizeClusterMembers(detail.Members)
	normalizeClusterCommonActions(detail.CommonActions)

	if err := domain.ValidateClusterDetail(detail); err != nil {
		return domain.ClusterDetail{}, fmt.Errorf("validate cluster detail: %w", err)
	}

	return detail, nil
}

func buildClusterMembers(items []map[string]any) []domain.ClusterMember {
	members := make([]domain.ClusterMember, 0, len(items))
	for _, item := range items {
		address := strings.TrimSpace(stringValue(item, "address"))
		if address == "" {
			continue
		}
		members = append(members, domain.ClusterMember{
			Chain:   domain.Chain(strings.TrimSpace(stringValue(item, "chain"))),
			Address: address,
			Label:   firstNonEmpty(stringValue(item, "label"), address),
		})
	}
	return members
}

func buildClusterCommonActions(items []map[string]any) []domain.ClusterCommonAction {
	actions := make([]domain.ClusterCommonAction, 0, len(items))
	for _, item := range items {
		actions = append(actions, domain.ClusterCommonAction{
			Kind:              strings.TrimSpace(stringValue(item, "kind")),
			Label:             firstNonEmpty(stringValue(item, "label"), stringValue(item, "address")),
			Chain:             domain.Chain(strings.TrimSpace(stringValue(item, "chain"))),
			Address:           strings.TrimSpace(stringValue(item, "address")),
			SharedMemberCount: int(int64Value(item, "sharedMemberCount")),
			InteractionCount:  int(int64Value(item, "interactionCount")),
			ObservedAt:        strings.TrimSpace(stringValue(item, "observedAt")),
		})
	}
	return actions
}

func buildClusterDetailEvidence(detail domain.ClusterDetail) []domain.Evidence {
	evidence := []domain.Evidence{
		{
			Kind:       domain.EvidenceClusterOverlap,
			Label:      "cluster member overlap",
			Source:     "cluster-detail",
			Confidence: 0.88,
			ObservedAt: firstClusterObservedAt(detail),
			Metadata: map[string]any{
				"cluster_id":     detail.ID,
				"cluster_type":   detail.ClusterType,
				"score":          detail.Score,
				"member_count":   detail.MemberCount,
				"classification": detail.Classification,
			},
		},
	}

	if len(detail.CommonActions) > 0 {
		top := detail.CommonActions[0]
		evidence = append(evidence, domain.Evidence{
			Kind:       domain.EvidenceTransfer,
			Label:      "shared counterparty activity",
			Source:     "cluster-detail",
			Confidence: 0.74,
			ObservedAt: top.ObservedAt,
			Metadata: map[string]any{
				"action_kind":         top.Kind,
				"action_label":        top.Label,
				"shared_member_count": top.SharedMemberCount,
				"interaction_count":   top.InteractionCount,
				"address":             top.Address,
			},
		})
	}

	return evidence
}

func firstClusterObservedAt(detail domain.ClusterDetail) string {
	for _, action := range detail.CommonActions {
		if strings.TrimSpace(action.ObservedAt) != "" {
			return action.ObservedAt
		}
	}
	return ""
}

func normalizeClusterMembers(items []domain.ClusterMember) {
	sort.SliceStable(items, func(left int, right int) bool {
		if items[left].Label == items[right].Label {
			return items[left].Address < items[right].Address
		}
		return items[left].Label < items[right].Label
	})
}

func normalizeClusterCommonActions(items []domain.ClusterCommonAction) {
	sort.SliceStable(items, func(left int, right int) bool {
		if items[left].SharedMemberCount == items[right].SharedMemberCount {
			if items[left].InteractionCount == items[right].InteractionCount {
				return items[left].Label < items[right].Label
			}
			return items[left].InteractionCount > items[right].InteractionCount
		}
		return items[left].SharedMemberCount > items[right].SharedMemberCount
	})
}
