package db

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/flowintel/flowintel/packages/domain"
)

func TestNeo4jTransactionGraphMaterializerMaterializesInteraction(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)
	tx := domain.NormalizeNormalizedTransaction(domain.NormalizedTransaction{
		Chain:  domain.Chain(" EVM "),
		TxHash: " 0xdeadbeef ",
		Wallet: domain.WalletRef{
			Chain:   domain.Chain(" EVM "),
			Address: " 0x1234567890abcdef1234567890abcdef12345678 ",
		},
		Counterparty: &domain.WalletRef{
			Chain:   domain.Chain(" EVM "),
			Address: " 0xabcdefabcdefabcdefabcdefabcdefabcdefabcd ",
		},
		Direction:      domain.TransactionDirectionOutbound,
		ObservedAt:     observedAt,
		RawPayloadPath: " s3://flowintel/raw/2026/03/19/tx.json ",
		Provider:       " alchemy ",
	})

	session := &capturingNeo4jSession{result: &capturingNeo4jResult{}}
	driver := &capturingNeo4jDriver{session: session}

	err := NewNeo4jTransactionGraphMaterializer(driver, "neo4j").MaterializeNormalizedTransaction(context.Background(), NormalizedTransactionWrite{
		Transaction: tx,
	})
	if err != nil {
		t.Fatalf("expected materialization to succeed, got %v", err)
	}

	if session.runs != 1 {
		t.Fatalf("expected 1 run, got %d", session.runs)
	}
	if session.query != materializeWalletInteractionCypher {
		t.Fatalf("unexpected query %q", session.query)
	}
	if !strings.Contains(session.query, "interaction.interactionCount") {
		t.Fatalf("expected interaction count metadata in query: %q", session.query)
	}
	if !strings.Contains(session.query, "interaction.inboundCount") || !strings.Contains(session.query, "interaction.outboundCount") {
		t.Fatalf("expected directional interaction counters in query: %q", session.query)
	}
	if !strings.Contains(session.query, "interaction.firstObservedAt") || !strings.Contains(session.query, "interaction.lastObservedAt") {
		t.Fatalf("expected first/last observed metadata in query: %q", session.query)
	}

	if got := session.params["walletChain"]; got != "evm" {
		t.Fatalf("unexpected wallet chain %#v", got)
	}
	if got := session.params["walletAddress"]; got != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected wallet address %#v", got)
	}
	if got := session.params["walletID"]; got != "evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected wallet id %#v", got)
	}
	if got := session.params["counterpartyID"]; got != "evm:0xabcdefabcdefabcdefabcdefabcdefabcdefabcd" {
		t.Fatalf("unexpected counterparty id %#v", got)
	}
	if got := session.params["walletDisplayName"]; got != defaultWalletDisplayName(WalletRef{Chain: tx.Wallet.Chain, Address: tx.Wallet.Address}) {
		t.Fatalf("unexpected wallet display name %#v", got)
	}
	if got := session.params["counterpartyDisplayName"]; got != defaultWalletDisplayName(WalletRef{Chain: tx.Counterparty.Chain, Address: tx.Counterparty.Address}) {
		t.Fatalf("unexpected counterparty display name %#v", got)
	}
	if got, ok := session.params["observedAt"].(time.Time); !ok || !got.Equal(observedAt) {
		t.Fatalf("unexpected observedAt %#v", session.params["observedAt"])
	}
	if got := session.params["txHash"]; got != "0xdeadbeef" {
		t.Fatalf("unexpected tx hash %#v", got)
	}
	if got := session.params["provider"]; got != "alchemy" {
		t.Fatalf("unexpected provider %#v", got)
	}
	if got := session.params["materializeFunding"]; got != false {
		t.Fatalf("unexpected materializeFunding %#v", got)
	}
}

func TestNeo4jTransactionGraphMaterializerMaterializesFundedByForInboundInteraction(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)
	tx := domain.NormalizeNormalizedTransaction(domain.NormalizedTransaction{
		Chain:  domain.ChainEVM,
		TxHash: "0xfeedbead",
		Wallet: domain.WalletRef{
			Chain:   domain.ChainEVM,
			Address: "0x1234567890abcdef1234567890abcdef12345678",
		},
		Counterparty: &domain.WalletRef{
			Chain:   domain.ChainEVM,
			Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
		},
		Direction:      domain.TransactionDirectionInbound,
		ObservedAt:     observedAt,
		RawPayloadPath: "s3://flowintel/raw/2026/03/19/inbound.json",
		Provider:       "alchemy",
	})

	session := &capturingNeo4jSession{result: &capturingNeo4jResult{}}
	driver := &capturingNeo4jDriver{session: session}

	err := NewNeo4jTransactionGraphMaterializer(driver, "neo4j").MaterializeNormalizedTransaction(context.Background(), NormalizedTransactionWrite{
		Transaction: tx,
	})
	if err != nil {
		t.Fatalf("expected inbound materialization to succeed, got %v", err)
	}

	if !strings.Contains(session.query, "MERGE (counterparty)-[funding:FUNDED_BY]->(wallet)") {
		t.Fatalf("expected funded_by materialization in query: %q", session.query)
	}
	if !strings.Contains(session.query, "funding.inboundCount") || !strings.Contains(session.query, "funding.outboundCount") {
		t.Fatalf("expected directional funding counters in query: %q", session.query)
	}
	if got := session.params["materializeFunding"]; got != true {
		t.Fatalf("unexpected materializeFunding %#v", got)
	}
}

func TestNeo4jTransactionGraphMaterializerMaterializesWalletOnly(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)
	tx := domain.NormalizeNormalizedTransaction(domain.NormalizedTransaction{
		Chain:          domain.ChainEVM,
		TxHash:         "0xfeedface",
		Wallet:         domain.WalletRef{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
		ObservedAt:     observedAt,
		RawPayloadPath: "s3://flowintel/raw/2026/03/19/tx.json",
		Provider:       "alchemy",
	})

	session := &capturingNeo4jSession{result: &capturingNeo4jResult{}}
	driver := &capturingNeo4jDriver{session: session}

	err := NewNeo4jTransactionGraphMaterializer(driver, "neo4j").MaterializeNormalizedTransaction(context.Background(), NormalizedTransactionWrite{
		Transaction: tx,
	})
	if err != nil {
		t.Fatalf("expected wallet-only materialization to succeed, got %v", err)
	}

	if session.query != materializeWalletNodeCypher {
		t.Fatalf("unexpected query %q", session.query)
	}
	if _, ok := session.params["counterpartyChain"]; ok {
		t.Fatalf("did not expect counterparty params %#v", session.params)
	}
}

func TestNeo4jTransactionGraphMaterializerMaterializeNormalizedTransactions(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)
	first := domain.NormalizeNormalizedTransaction(domain.NormalizedTransaction{
		Chain:          domain.ChainEVM,
		TxHash:         "0xfeedface",
		Wallet:         domain.WalletRef{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
		ObservedAt:     observedAt,
		RawPayloadPath: "s3://flowintel/raw/2026/03/19/tx.json",
		Provider:       "alchemy",
	})
	second := domain.NormalizeNormalizedTransaction(domain.NormalizedTransaction{
		Chain:          domain.ChainSolana,
		TxHash:         "9Q1z8F7a6B5c4D3e2F1g0h9j8k7l6m5n4o3p2q1r0s9t8u7v6w5x4y3z2a1b0c9d8",
		Wallet:         domain.WalletRef{Chain: domain.ChainSolana, Address: "7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq"},
		ObservedAt:     observedAt,
		RawPayloadPath: "s3://flowintel/raw/2026/03/19/tx-2.json",
		Provider:       "helius",
	})

	session := &capturingNeo4jSession{result: &capturingNeo4jResult{}}
	driver := &capturingNeo4jDriver{session: session}

	err := NewNeo4jTransactionGraphMaterializer(driver, "neo4j").MaterializeNormalizedTransactions(context.Background(), []NormalizedTransactionWrite{
		{Transaction: first},
		{Transaction: second},
	})
	if err != nil {
		t.Fatalf("expected batch materialization to succeed, got %v", err)
	}

	if session.runs != 2 {
		t.Fatalf("expected 2 runs, got %d", session.runs)
	}
}

func TestNeo4jTransactionGraphMaterializerRejectsInvalidTransaction(t *testing.T) {
	t.Parallel()

	session := &capturingNeo4jSession{result: &capturingNeo4jResult{}}
	driver := &capturingNeo4jDriver{session: session}

	err := NewNeo4jTransactionGraphMaterializer(driver, "neo4j").MaterializeNormalizedTransaction(context.Background(), NormalizedTransactionWrite{
		Transaction: domain.NormalizedTransaction{
			Chain:         domain.ChainEVM,
			TxHash:        "0xdeadbeef",
			Wallet:        domain.WalletRef{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
			ObservedAt:    time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC),
			Provider:      "alchemy",
			SchemaVersion: 1,
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if session.runs != 0 {
		t.Fatalf("expected no runs after validation failure, got %d", session.runs)
	}
}

type capturingNeo4jDriver struct {
	session *capturingNeo4jSession
}

func (d *capturingNeo4jDriver) NewSession(context.Context, neo4j.SessionConfig) Neo4jSession {
	return d.session
}

func (d *capturingNeo4jDriver) VerifyConnectivity(context.Context) error { return nil }

func (d *capturingNeo4jDriver) Close(context.Context) error { return nil }

type capturingNeo4jSession struct {
	query  string
	params map[string]any
	runs   int
	result *capturingNeo4jResult
}

func (s *capturingNeo4jSession) Run(_ context.Context, query string, params map[string]any, _ ...func(*neo4j.TransactionConfig)) (Neo4jResult, error) {
	s.query = query
	s.params = cloneAnyMap(params)
	s.runs++
	if s.result == nil {
		s.result = &capturingNeo4jResult{}
	}

	return s.result, nil
}

func (s *capturingNeo4jSession) Close(context.Context) error { return nil }

type capturingNeo4jResult struct {
	err error
}

func (r *capturingNeo4jResult) Next(context.Context) bool { return false }

func (r *capturingNeo4jResult) Err() error { return r.err }

func (r *capturingNeo4jResult) Record() *neo4j.Record { return nil }

func cloneAnyMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}

	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}

	return cloned
}
