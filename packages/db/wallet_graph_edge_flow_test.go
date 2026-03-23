package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type fakeWalletGraphEdgeFlowRow struct {
	chain          string
	address        string
	inboundCount   int64
	outboundCount  int64
	inboundAmount  string
	outboundAmount string
	primaryToken   string
	tokenRaw       []byte
}

type fakeWalletGraphEdgeFlowRows struct {
	rows  []fakeWalletGraphEdgeFlowRow
	index int
	err   error
}

func (r *fakeWalletGraphEdgeFlowRows) Close()                                       {}
func (r *fakeWalletGraphEdgeFlowRows) Err() error                                   { return r.err }
func (r *fakeWalletGraphEdgeFlowRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeWalletGraphEdgeFlowRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeWalletGraphEdgeFlowRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeWalletGraphEdgeFlowRows) RawValues() [][]byte                          { return nil }
func (r *fakeWalletGraphEdgeFlowRows) Conn() *pgx.Conn                              { return nil }

func (r *fakeWalletGraphEdgeFlowRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *fakeWalletGraphEdgeFlowRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("scan called out of range")
	}
	if len(dest) != 9 {
		return errors.New("unexpected scan destination count")
	}

	row := r.rows[r.index-1]
	*(dest[0].(*string)) = row.chain
	*(dest[1].(*string)) = row.address
	*(dest[2].(*int64)) = row.inboundCount + row.outboundCount
	*(dest[3].(*int64)) = row.inboundCount
	*(dest[4].(*int64)) = row.outboundCount
	*(dest[5].(*string)) = row.inboundAmount
	*(dest[6].(*string)) = row.outboundAmount
	*(dest[7].(*string)) = row.primaryToken
	*(dest[8].(*[]byte)) = append([]byte(nil), row.tokenRaw...)
	return nil
}

type fakeWalletGraphEdgeFlowQuerier struct {
	row      fakeRow
	rows     *fakeWalletGraphEdgeFlowRows
	queryErr error
}

func (q *fakeWalletGraphEdgeFlowQuerier) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return q.row
}

func (q *fakeWalletGraphEdgeFlowQuerier) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	if q.queryErr != nil {
		return nil, q.queryErr
	}
	return q.rows, nil
}

func TestEnrichedWalletGraphReaderAttachesTokenFlow(t *testing.T) {
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
				SourceID: "evm:0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				TargetID: "evm:0x1234567890abcdef1234567890abcdef12345678",
				Kind:     domain.WalletGraphEdgeFundedBy,
				Family:   domain.WalletGraphEdgeFamilyDerived,
			},
			{
				SourceID: "evm:0x1234567890abcdef1234567890abcdef12345678",
				TargetID: "cluster:alpha",
				Kind:     domain.WalletGraphEdgeMemberOf,
				Family:   domain.WalletGraphEdgeFamilyDerived,
			},
		},
	}

	reader := NewEnrichedWalletGraphReader(
		&stubWalletGraphReader{graph: graph},
		NewPostgresWalletGraphEdgeFlowReader(&fakeWalletGraphEdgeFlowQuerier{
			rows: &fakeWalletGraphEdgeFlowRows{
				rows: []fakeWalletGraphEdgeFlowRow{
					{
						chain:          "evm",
						address:        "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
						inboundCount:   2,
						outboundCount:  5,
						inboundAmount:  "42",
						outboundAmount: "616.06",
						primaryToken:   "USDC",
						tokenRaw:       []byte(`[{"symbol":"USDC","inbound_amount":"42","outbound_amount":"616.06"}]`),
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
		DepthRequested:    2,
		DepthResolved:     1,
		MaxCounterparties: 25,
	})
	if err != nil {
		t.Fatalf("ReadWalletGraph returned error: %v", err)
	}

	if enriched.Edges[0].TokenFlow == nil || enriched.Edges[0].TokenFlow.PrimaryToken != "USDC" {
		t.Fatalf("expected interacted_with edge token flow, got %#v", enriched.Edges[0].TokenFlow)
	}
	if enriched.Edges[1].TokenFlow == nil || enriched.Edges[1].TokenFlow.OutboundAmount != "616.06" {
		t.Fatalf("expected funded_by edge token flow, got %#v", enriched.Edges[1].TokenFlow)
	}
	if enriched.Edges[2].TokenFlow != nil {
		t.Fatalf("did not expect member_of edge token flow, got %#v", enriched.Edges[2].TokenFlow)
	}
}
