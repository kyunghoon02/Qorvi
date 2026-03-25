package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/flowintel/flowintel/packages/domain"
)

type fakeWalletGraphEntityLinkRow struct {
	chain       string
	address     string
	walletLabel string
	entityKey   string
	entityType  string
	entityLabel string
}

type fakeWalletGraphEntityLinkRows struct {
	rows  []fakeWalletGraphEntityLinkRow
	index int
	err   error
}

func (r *fakeWalletGraphEntityLinkRows) Close()                                       {}
func (r *fakeWalletGraphEntityLinkRows) Err() error                                   { return r.err }
func (r *fakeWalletGraphEntityLinkRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeWalletGraphEntityLinkRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeWalletGraphEntityLinkRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeWalletGraphEntityLinkRows) RawValues() [][]byte                          { return nil }
func (r *fakeWalletGraphEntityLinkRows) Conn() *pgx.Conn                              { return nil }

func (r *fakeWalletGraphEntityLinkRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *fakeWalletGraphEntityLinkRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("scan called out of range")
	}
	if len(dest) != 6 {
		return errors.New("unexpected scan destination count")
	}

	row := r.rows[r.index-1]
	*(dest[0].(*string)) = row.chain
	*(dest[1].(*string)) = row.address
	*(dest[2].(*string)) = row.walletLabel
	*(dest[3].(*string)) = row.entityKey
	*(dest[4].(*string)) = row.entityType
	*(dest[5].(*string)) = row.entityLabel
	return nil
}

type fakeWalletGraphEntityLinkQuerier struct {
	row      fakeRow
	rows     *fakeWalletGraphEntityLinkRows
	queryErr error
}

func (q *fakeWalletGraphEntityLinkQuerier) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return q.row
}

func (q *fakeWalletGraphEntityLinkQuerier) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	if q.queryErr != nil {
		return nil, q.queryErr
	}
	return q.rows, nil
}

func TestEntityLinkedWalletGraphReaderAttachesEntityNodesAndEdges(t *testing.T) {
	t.Parallel()

	graph := domain.WalletGraph{
		Chain:         domain.ChainEVM,
		Address:       "0x1234567890abcdef1234567890abcdef12345678",
		DepthResolved: 1,
		Nodes: []domain.WalletGraphNode{
			{
				ID:      "evm:0x1234567890abcdef1234567890abcdef12345678",
				Kind:    domain.WalletGraphNodeWallet,
				Chain:   domain.ChainEVM,
				Address: "0x1234567890abcdef1234567890abcdef12345678",
				Label:   "Seed Whale",
			},
			{
				ID:      "evm:0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				Kind:    domain.WalletGraphNodeWallet,
				Chain:   domain.ChainEVM,
				Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				Label:   "Counterparty",
			},
			{
				ID:    "cluster:alpha",
				Kind:  domain.WalletGraphNodeCluster,
				Label: "Alpha",
			},
		},
		Edges: []domain.WalletGraphEdge{
			{
				SourceID: "evm:0x1234567890abcdef1234567890abcdef12345678",
				TargetID: "evm:0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				Kind:     domain.WalletGraphEdgeInteractedWith,
				Family:   domain.WalletGraphEdgeFamilyBase,
			},
			{
				SourceID: "evm:0x1234567890abcdef1234567890abcdef12345678",
				TargetID: "cluster:alpha",
				Kind:     domain.WalletGraphEdgeMemberOf,
				Family:   domain.WalletGraphEdgeFamilyDerived,
			},
		},
	}

	reader := NewEntityLinkedWalletGraphReader(
		&stubWalletGraphReader{graph: graph},
		NewPostgresWalletGraphEntityLinkReader(&fakeWalletGraphEntityLinkQuerier{
			rows: &fakeWalletGraphEntityLinkRows{
				rows: []fakeWalletGraphEntityLinkRow{
					{
						chain:       "evm",
						address:     "0x1234567890abcdef1234567890abcdef12345678",
						walletLabel: "Seed Whale",
						entityKey:   "entity_seed_whale",
						entityType:  "exchange",
						entityLabel: "Seed Whale Exchange",
					},
					{
						chain:       "evm",
						address:     "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
						walletLabel: "Counterparty",
						entityKey:   "entity_counterparty",
						entityType:  "bridge",
						entityLabel: "Bridge Counterparty",
					},
				},
			},
		}),
	)

	enriched, err := reader.ReadWalletGraph(context.Background(), WalletGraphQuery{
		Ref: WalletRef{
			Chain:   domain.ChainEVM,
			Address: "0x1234567890abcdef1234567890abcdef12345678",
		},
		DepthRequested:    1,
		DepthResolved:     1,
		MaxCounterparties: 25,
	})
	if err != nil {
		t.Fatalf("ReadWalletGraph returned error: %v", err)
	}

	if len(enriched.Nodes) != 5 {
		t.Fatalf("expected 5 nodes after entity enrich, got %d", len(enriched.Nodes))
	}
	if len(enriched.Edges) != 4 {
		t.Fatalf("expected 4 edges after entity enrich, got %d", len(enriched.Edges))
	}
	if enriched.Nodes[1].Kind != domain.WalletGraphNodeCluster || enriched.Nodes[2].Kind != domain.WalletGraphNodeEntity || enriched.Nodes[3].Kind != domain.WalletGraphNodeEntity {
		t.Fatalf("expected cluster followed by entity nodes before wallet peers, got %#v", enriched.Nodes)
	}
	if enriched.Edges[1].Kind != domain.WalletGraphEdgeEntityLinked || enriched.Edges[2].Kind != domain.WalletGraphEdgeEntityLinked {
		t.Fatalf("expected entity_linked edges after member_of, got %#v", enriched.Edges)
	}
	if enriched.Edges[1].Evidence == nil || enriched.Edges[1].Evidence.Source != "postgres-wallet-identity" {
		t.Fatalf("expected entity_linked edge evidence, got %#v", enriched.Edges[1].Evidence)
	}
	entityLabels := []string{}
	for _, node := range enriched.Nodes {
		if node.Kind == domain.WalletGraphNodeEntity {
			entityLabels = append(entityLabels, node.Label)
		}
	}
	if len(entityLabels) != 2 || entityLabels[0] != "Bridge Counterparty" || entityLabels[1] != "Seed Whale Exchange" {
		t.Fatalf("expected entity display labels to prefer entity display_name, got %#v", entityLabels)
	}
}

func TestWalletEntityLinkEvidenceSourceReflectsEntityNamespace(t *testing.T) {
	t.Parallel()

	if got := walletEntityLinkEvidenceSource("entity:heuristic:solana:jupiter"); got != "provider-heuristic-identity" {
		t.Fatalf("unexpected heuristic source %q", got)
	}
	if got := walletEntityLinkEvidenceSource("entity:curated:binance"); got != "curated-identity-index" {
		t.Fatalf("unexpected curated source %q", got)
	}
	if got := walletEntityLinkEvidenceSource("entity:plain"); got != "postgres-wallet-identity" {
		t.Fatalf("unexpected default source %q", got)
	}
}
