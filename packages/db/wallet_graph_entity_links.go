package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowintel/flowintel/packages/domain"
)

const walletGraphEntityLinksSQL = `
WITH targets AS (
  SELECT DISTINCT chain, address
  FROM unnest($1::text[], $2::text[]) AS target(chain, address)
)
SELECT
  w.chain,
  w.address,
  COALESCE(w.display_name, w.address) AS wallet_label,
  COALESCE(w.entity_key, '') AS entity_key,
  COALESCE(e.entity_type, '') AS entity_type,
  COALESCE(NULLIF(e.display_name, ''), '') AS entity_label
FROM targets target
JOIN wallets w
  ON w.chain = target.chain
 AND w.address = target.address
LEFT JOIN entities e
  ON e.entity_key = w.entity_key
WHERE COALESCE(w.entity_key, '') <> ''
`

type PostgresWalletGraphEntityLinkReader struct {
	Querier postgresQuerier
}

func NewPostgresWalletGraphEntityLinkReader(
	querier postgresQuerier,
) *PostgresWalletGraphEntityLinkReader {
	return &PostgresWalletGraphEntityLinkReader{Querier: querier}
}

type EntityLinkedWalletGraphReader struct {
	Loader     WalletGraphReader
	EntityLink *PostgresWalletGraphEntityLinkReader
}

func NewEntityLinkedWalletGraphReader(
	loader WalletGraphReader,
	entityLink *PostgresWalletGraphEntityLinkReader,
) *EntityLinkedWalletGraphReader {
	return &EntityLinkedWalletGraphReader{
		Loader:     loader,
		EntityLink: entityLink,
	}
}

func (r *EntityLinkedWalletGraphReader) ReadWalletGraph(
	ctx context.Context,
	query WalletGraphQuery,
) (domain.WalletGraph, error) {
	if r == nil || r.Loader == nil {
		return domain.WalletGraph{}, fmt.Errorf("wallet graph reader is nil")
	}

	graph, err := r.Loader.ReadWalletGraph(ctx, query)
	if err != nil {
		return domain.WalletGraph{}, err
	}

	if r.EntityLink == nil {
		return graph, nil
	}

	links, err := r.EntityLink.ReadWalletGraphEntityLinks(ctx, graph)
	if err != nil {
		return domain.WalletGraph{}, fmt.Errorf("read wallet graph entity links: %w", err)
	}
	if len(links) == 0 {
		return graph, nil
	}

	next := graph
	nodeIDs := make(map[string]struct{}, len(next.Nodes))
	for _, node := range next.Nodes {
		nodeIDs[node.ID] = struct{}{}
	}

	for _, node := range graph.Nodes {
		if node.Kind != domain.WalletGraphNodeWallet || node.Address == "" {
			continue
		}

		link, exists := links[buildWalletGraphCounterpartyKey(string(node.Chain), node.Address)]
		if !exists {
			continue
		}

		if _, seen := nodeIDs[link.EntityNode.ID]; !seen {
			next.Nodes = append(next.Nodes, link.EntityNode)
			nodeIDs[link.EntityNode.ID] = struct{}{}
		}

		next.Edges = append(next.Edges, domain.WalletGraphEdge{
			SourceID: node.ID,
			TargetID: link.EntityNode.ID,
			Kind:     domain.WalletGraphEdgeEntityLinked,
			Family:   domain.WalletGraphEdgeFamilyForKind(domain.WalletGraphEdgeEntityLinked),
			Evidence: &domain.WalletGraphEdgeEvidence{
				Source:     walletEntityLinkEvidenceSource(link.EntityNode.ID),
				Confidence: "medium",
				Summary:    fmt.Sprintf("Wallet linked to %s entity via entity_key.", link.EntityType),
			},
		})
	}

	normalizeWalletGraph(&next)
	return next, nil
}

type walletGraphEntityLinkRecord struct {
	WalletNodeID string
	EntityNode   domain.WalletGraphNode
	EntityType   string
}

func (r *PostgresWalletGraphEntityLinkReader) ReadWalletGraphEntityLinks(
	ctx context.Context,
	graph domain.WalletGraph,
) (map[string]walletGraphEntityLinkRecord, error) {
	if r == nil || r.Querier == nil {
		return nil, fmt.Errorf("postgres wallet graph entity link reader is nil")
	}

	chains := make([]string, 0, len(graph.Nodes))
	addresses := make([]string, 0, len(graph.Nodes))
	seen := make(map[string]struct{})
	for _, node := range graph.Nodes {
		if node.Kind != domain.WalletGraphNodeWallet || node.Address == "" {
			continue
		}

		key := buildWalletGraphCounterpartyKey(string(node.Chain), node.Address)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		chains = append(chains, string(node.Chain))
		addresses = append(addresses, node.Address)
	}

	if len(chains) == 0 {
		return map[string]walletGraphEntityLinkRecord{}, nil
	}

	rows, err := r.Querier.Query(ctx, walletGraphEntityLinksSQL, chains, addresses)
	if err != nil {
		return nil, fmt.Errorf("query wallet graph entity links: %w", err)
	}
	defer rows.Close()

	links := make(map[string]walletGraphEntityLinkRecord, len(chains))
	for rows.Next() {
		var (
			chain       string
			address     string
			walletLabel string
			entityKey   string
			entityType  string
			entityLabel string
		)

		if err := rows.Scan(
			&chain,
			&address,
			&walletLabel,
			&entityKey,
			&entityType,
			&entityLabel,
		); err != nil {
			return nil, fmt.Errorf("scan wallet graph entity link: %w", err)
		}

		entityKey = strings.TrimSpace(entityKey)
		entityType = strings.TrimSpace(entityType)
		if entityKey == "" {
			continue
		}
		entityNodeID := domain.BuildEntityCanonicalKey(firstNonEmpty(entityType, "entity"), entityKey)
		label := firstNonEmpty(strings.TrimSpace(entityLabel), buildWalletGraphEntityLabel(entityType, entityKey))
		links[buildWalletGraphCounterpartyKey(chain, address)] = walletGraphEntityLinkRecord{
			WalletNodeID: domain.BuildWalletCanonicalKey(domain.Chain(chain), address),
			EntityType:   firstNonEmpty(entityType, "entity"),
			EntityNode: domain.WalletGraphNode{
				ID:    entityNodeID,
				Kind:  domain.WalletGraphNodeEntity,
				Label: firstNonEmpty(label, walletLabel, entityKey),
			},
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("wallet graph entity link rows: %w", err)
	}

	return links, nil
}

func buildWalletGraphEntityLabel(entityType string, entityKey string) string {
	trimmedType := strings.TrimSpace(entityType)
	trimmedKey := strings.TrimSpace(entityKey)
	switch {
	case trimmedType != "" && trimmedKey != "":
		return fmt.Sprintf("%s · %s", trimmedType, trimmedKey)
	case trimmedType != "":
		return trimmedType
	default:
		return trimmedKey
	}
}

func walletEntityLinkEvidenceSource(entityNodeID string) string {
	switch {
	case strings.Contains(entityNodeID, "heuristic:"):
		return "provider-heuristic-identity"
	case strings.Contains(entityNodeID, "curated:"):
		return "curated-identity-index"
	default:
		return "postgres-wallet-identity"
	}
}

var _ WalletGraphReader = (*EntityLinkedWalletGraphReader)(nil)
