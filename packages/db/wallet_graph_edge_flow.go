package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qorvi/qorvi/packages/domain"
)

const walletGraphEdgeFlowSQL = `
WITH root AS (
  SELECT id, chain
  FROM wallets
  WHERE chain = $1 AND address = $2
  LIMIT 1
),
targets AS (
  SELECT DISTINCT chain, address
  FROM unnest($3::text[], $4::text[]) AS target(chain, address)
),
tx_base AS (
  SELECT
    COALESCE(NULLIF(t.counterparty_chain, ''), r.chain) AS counterparty_chain,
    NULLIF(t.counterparty_address, '') AS counterparty_address,
    t.direction,
    COALESCE(t.amount_numeric, 0::numeric) AS amount_numeric,
    COALESCE(NULLIF(t.token_symbol, ''), 'token') AS token_symbol,
    t.observed_at
  FROM root r
  JOIN transactions t ON t.wallet_id = r.id
  WHERE NULLIF(t.counterparty_address, '') IS NOT NULL
),
targeted AS (
  SELECT
    target.chain,
    target.address,
    tx.direction,
    tx.amount_numeric,
    tx.token_symbol,
    tx.observed_at
  FROM targets target
  JOIN tx_base tx
    ON tx.counterparty_chain = target.chain
   AND tx.counterparty_address = target.address
),
token_rollup AS (
  SELECT
    chain,
    address,
    token_symbol,
    COUNT(*) FILTER (WHERE direction = 'inbound') AS inbound_count,
    COUNT(*) FILTER (WHERE direction = 'outbound') AS outbound_count,
    COALESCE(SUM(CASE WHEN direction = 'inbound' THEN amount_numeric ELSE 0 END), 0::numeric) AS inbound_amount,
    COALESCE(SUM(CASE WHEN direction = 'outbound' THEN amount_numeric ELSE 0 END), 0::numeric) AS outbound_amount,
    COALESCE(SUM(amount_numeric), 0::numeric) AS total_amount
  FROM targeted
  GROUP BY chain, address, token_symbol
)
SELECT
  target.chain,
  target.address,
  COUNT(*) AS interaction_count,
  COUNT(*) FILTER (WHERE target.direction = 'inbound') AS inbound_count,
  COUNT(*) FILTER (WHERE target.direction = 'outbound') AS outbound_count,
  COALESCE(SUM(CASE WHEN target.direction = 'inbound' THEN target.amount_numeric ELSE 0 END), 0::numeric)::text AS inbound_amount,
  COALESCE(SUM(CASE WHEN target.direction = 'outbound' THEN target.amount_numeric ELSE 0 END), 0::numeric)::text AS outbound_amount,
  COALESCE((
    SELECT rollup.token_symbol
    FROM token_rollup rollup
    WHERE rollup.chain = target.chain AND rollup.address = target.address
    ORDER BY rollup.total_amount DESC, rollup.token_symbol ASC
    LIMIT 1
  ), '') AS primary_token,
  COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
      'symbol', rollup.token_symbol,
      'inbound_amount', rollup.inbound_amount::text,
      'outbound_amount', rollup.outbound_amount::text
    ) ORDER BY rollup.total_amount DESC, rollup.token_symbol ASC)
    FROM token_rollup rollup
    WHERE rollup.chain = target.chain AND rollup.address = target.address
  ), '[]'::jsonb) AS token_breakdowns
FROM targeted target
GROUP BY target.chain, target.address
`

type PostgresWalletGraphEdgeFlowReader struct {
	Querier postgresQuerier
}

func NewPostgresWalletGraphEdgeFlowReader(
	querier postgresQuerier,
) *PostgresWalletGraphEdgeFlowReader {
	return &PostgresWalletGraphEdgeFlowReader{Querier: querier}
}

type EnrichedWalletGraphReader struct {
	Loader     WalletGraphReader
	FlowReader *PostgresWalletGraphEdgeFlowReader
}

func NewEnrichedWalletGraphReader(
	loader WalletGraphReader,
	flowReader *PostgresWalletGraphEdgeFlowReader,
) *EnrichedWalletGraphReader {
	return &EnrichedWalletGraphReader{
		Loader:     loader,
		FlowReader: flowReader,
	}
}

func (r *EnrichedWalletGraphReader) ReadWalletGraph(
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

	if r.FlowReader == nil {
		return graph, nil
	}

	tokenFlows, err := r.FlowReader.ReadWalletGraphEdgeTokenFlows(ctx, query.Ref, graph)
	if err != nil {
		return domain.WalletGraph{}, fmt.Errorf("read wallet graph edge token flows: %w", err)
	}

	if len(tokenFlows) == 0 {
		return graph, nil
	}

	next := graph
	for index := range next.Edges {
		counterpartyKey, ok := walletGraphEdgeCounterpartyKey(next, next.Edges[index], query.Ref)
		if !ok {
			continue
		}

		tokenFlow, exists := tokenFlows[counterpartyKey]
		if !exists {
			continue
		}
		next.Edges[index].TokenFlow = &tokenFlow
	}

	return next, nil
}

func (r *PostgresWalletGraphEdgeFlowReader) ReadWalletGraphEdgeTokenFlows(
	ctx context.Context,
	ref WalletRef,
	graph domain.WalletGraph,
) (map[string]domain.WalletGraphEdgeTokenFlow, error) {
	if r == nil || r.Querier == nil {
		return nil, fmt.Errorf("postgres wallet graph edge flow reader is nil")
	}

	chains := make([]string, 0, len(graph.Edges))
	addresses := make([]string, 0, len(graph.Edges))
	seen := make(map[string]struct{})
	for _, edge := range graph.Edges {
		counterpartyKey, ok := walletGraphEdgeCounterpartyKey(graph, edge, ref)
		if !ok {
			continue
		}
		if _, exists := seen[counterpartyKey]; exists {
			continue
		}
		seen[counterpartyKey] = struct{}{}

		chain, address := splitWalletGraphCounterpartyKey(counterpartyKey)
		chains = append(chains, chain)
		addresses = append(addresses, address)
	}

	if len(chains) == 0 {
		return map[string]domain.WalletGraphEdgeTokenFlow{}, nil
	}

	rows, err := r.Querier.Query(
		ctx,
		walletGraphEdgeFlowSQL,
		string(ref.Chain),
		ref.Address,
		chains,
		addresses,
	)
	if err != nil {
		return nil, fmt.Errorf("query wallet graph edge token flows: %w", err)
	}
	defer rows.Close()

	flows := make(map[string]domain.WalletGraphEdgeTokenFlow, len(chains))
	for rows.Next() {
		var (
			chain          string
			address        string
			inboundCount   int64
			outboundCount  int64
			inboundAmount  string
			outboundAmount string
			primaryToken   string
			tokenRaw       []byte
		)

		if err := rows.Scan(
			&chain,
			&address,
			new(int64),
			&inboundCount,
			&outboundCount,
			&inboundAmount,
			&outboundAmount,
			&primaryToken,
			&tokenRaw,
		); err != nil {
			return nil, fmt.Errorf("scan wallet graph edge token flow: %w", err)
		}

		breakdowns, err := decodeWalletGraphEdgeTokenBreakdowns(tokenRaw)
		if err != nil {
			return nil, fmt.Errorf("decode wallet graph edge token breakdowns: %w", err)
		}

		flows[buildWalletGraphCounterpartyKey(chain, address)] = domain.WalletGraphEdgeTokenFlow{
			PrimaryToken:   strings.TrimSpace(primaryToken),
			InboundCount:   int(inboundCount),
			OutboundCount:  int(outboundCount),
			InboundAmount:  strings.TrimSpace(inboundAmount),
			OutboundAmount: strings.TrimSpace(outboundAmount),
			Breakdowns:     breakdowns,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("wallet graph edge token flow rows: %w", err)
	}

	return flows, nil
}

func decodeWalletGraphEdgeTokenBreakdowns(
	raw []byte,
) ([]domain.WalletGraphEdgeTokenBreakdown, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var records []walletSummaryTokenBreakdownRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		return nil, err
	}

	breakdowns := make([]domain.WalletGraphEdgeTokenBreakdown, 0, len(records))
	for _, record := range records {
		breakdowns = append(breakdowns, domain.WalletGraphEdgeTokenBreakdown{
			Symbol:         strings.TrimSpace(record.Symbol),
			InboundAmount:  strings.TrimSpace(record.InboundAmount),
			OutboundAmount: strings.TrimSpace(record.OutboundAmount),
		})
	}

	return breakdowns, nil
}

func walletGraphEdgeCounterpartyKey(
	graph domain.WalletGraph,
	edge domain.WalletGraphEdge,
	ref WalletRef,
) (string, bool) {
	rootKey := domain.BuildWalletCanonicalKey(ref.Chain, ref.Address)
	if edge.Kind != domain.WalletGraphEdgeInteractedWith && edge.Kind != domain.WalletGraphEdgeFundedBy {
		return "", false
	}

	nodeByID := make(map[string]domain.WalletGraphNode, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodeByID[node.ID] = node
	}

	var counterparty domain.WalletGraphNode
	switch {
	case strings.EqualFold(edge.SourceID, rootKey):
		counterparty = nodeByID[edge.TargetID]
	case strings.EqualFold(edge.TargetID, rootKey):
		counterparty = nodeByID[edge.SourceID]
	default:
		return "", false
	}

	if counterparty.Kind != domain.WalletGraphNodeWallet || counterparty.Address == "" {
		return "", false
	}

	return buildWalletGraphCounterpartyKey(string(counterparty.Chain), counterparty.Address), true
}

func buildWalletGraphCounterpartyKey(chain string, address string) string {
	return strings.ToLower(strings.TrimSpace(chain)) + ":" + strings.ToLower(strings.TrimSpace(address))
}

func splitWalletGraphCounterpartyKey(key string) (string, string) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

var _ WalletGraphReader = (*EnrichedWalletGraphReader)(nil)
