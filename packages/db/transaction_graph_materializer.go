package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/qorvi/qorvi/packages/domain"
)

const materializeWalletNodeCypher = `
MERGE (wallet:Wallet {chain: $walletChain, address: $walletAddress})
ON CREATE SET
  wallet.id = $walletID,
  wallet.displayName = $walletDisplayName
SET
  wallet.lastObservedAt = $observedAt,
  wallet.updatedAt = $observedAt
`

const materializeWalletInteractionCypher = `
MERGE (wallet:Wallet {chain: $walletChain, address: $walletAddress})
ON CREATE SET
  wallet.id = $walletID,
  wallet.displayName = $walletDisplayName
SET
  wallet.lastObservedAt = $observedAt,
  wallet.updatedAt = $observedAt
MERGE (counterparty:Wallet {chain: $counterpartyChain, address: $counterpartyAddress})
ON CREATE SET
  counterparty.id = $counterpartyID,
  counterparty.displayName = $counterpartyDisplayName
SET
  counterparty.lastObservedAt = $observedAt,
  counterparty.updatedAt = $observedAt
MERGE (wallet)-[interaction:INTERACTED_WITH]->(counterparty)
ON CREATE SET
  interaction.interactionCount = 0,
  interaction.inboundCount = 0,
  interaction.outboundCount = 0,
  interaction.counterpartyCount = 0,
  interaction.firstObservedAt = $observedAt
SET
  interaction.interactionCount = coalesce(interaction.interactionCount, interaction.counterpartyCount, 0) + 1,
  interaction.inboundCount = coalesce(interaction.inboundCount, 0) + CASE WHEN $direction = 'inbound' THEN 1 ELSE 0 END,
  interaction.outboundCount = coalesce(interaction.outboundCount, 0) + CASE WHEN $direction = 'outbound' THEN 1 ELSE 0 END,
  interaction.counterpartyCount = interaction.interactionCount,
  interaction.firstObservedAt = coalesce(interaction.firstObservedAt, $observedAt),
  interaction.lastObservedAt = $observedAt,
  interaction.lastTxHash = $txHash,
  interaction.lastDirection = $direction,
  interaction.lastProvider = $provider,
  interaction.lastRawPayloadPath = $rawPayloadPath
FOREACH (_ IN CASE WHEN $materializeFunding THEN [1] ELSE [] END |
  MERGE (counterparty)-[funding:FUNDED_BY]->(wallet)
  ON CREATE SET
    funding.interactionCount = 0,
    funding.inboundCount = 0,
    funding.outboundCount = 0,
    funding.counterpartyCount = 0,
    funding.firstObservedAt = $observedAt
  SET
    funding.interactionCount = coalesce(funding.interactionCount, funding.counterpartyCount, 0) + 1,
    funding.inboundCount = coalesce(funding.inboundCount, 0) + 1,
    funding.outboundCount = coalesce(funding.outboundCount, 0),
    funding.counterpartyCount = funding.interactionCount,
    funding.firstObservedAt = coalesce(funding.firstObservedAt, $observedAt),
    funding.lastObservedAt = $observedAt,
    funding.lastTxHash = $txHash,
    funding.lastDirection = $direction,
    funding.lastProvider = $provider,
    funding.lastRawPayloadPath = $rawPayloadPath
)
`

type TransactionGraphMaterializer interface {
	MaterializeNormalizedTransaction(context.Context, NormalizedTransactionWrite) error
	MaterializeNormalizedTransactions(context.Context, []NormalizedTransactionWrite) error
}

type Neo4jTransactionGraphMaterializer struct {
	Driver   Neo4jDriver
	Database string
	Now      func() time.Time
}

func NewNeo4jTransactionGraphMaterializer(driver Neo4jDriver, database string) *Neo4jTransactionGraphMaterializer {
	return &Neo4jTransactionGraphMaterializer{
		Driver:   driver,
		Database: database,
		Now:      time.Now,
	}
}

func (m *Neo4jTransactionGraphMaterializer) MaterializeNormalizedTransaction(
	ctx context.Context,
	write NormalizedTransactionWrite,
) error {
	if m == nil || m.Driver == nil {
		return fmt.Errorf("neo4j transaction graph materializer is nil")
	}

	record := domain.NormalizeNormalizedTransaction(write.Transaction)
	if err := domain.ValidateNormalizedTransaction(record); err != nil {
		return fmt.Errorf("validate normalized transaction: %w", err)
	}

	session := m.Driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.Database})
	defer func() {
		_ = session.Close(ctx)
	}()

	query, params := m.buildMaterializationQuery(record)
	result, err := session.Run(ctx, query, params)
	if err != nil {
		return fmt.Errorf("run neo4j transaction graph materialization: %w", err)
	}
	if result != nil {
		if err := result.Err(); err != nil {
			return fmt.Errorf("neo4j transaction graph materialization result error: %w", err)
		}
	}

	return nil
}

func (m *Neo4jTransactionGraphMaterializer) MaterializeNormalizedTransactions(
	ctx context.Context,
	writes []NormalizedTransactionWrite,
) error {
	for _, write := range writes {
		if err := m.MaterializeNormalizedTransaction(ctx, write); err != nil {
			return err
		}
	}

	return nil
}

func (m *Neo4jTransactionGraphMaterializer) buildMaterializationQuery(
	record domain.NormalizedTransaction,
) (string, map[string]any) {
	now := record.ObservedAt.UTC()
	if now.IsZero() {
		now = m.now().UTC()
	}

	walletID := domain.BuildWalletCanonicalKey(record.Wallet.Chain, record.Wallet.Address)
	params := map[string]any{
		"walletChain":        string(record.Wallet.Chain),
		"walletAddress":      record.Wallet.Address,
		"walletID":           walletID,
		"walletDisplayName":  defaultWalletDisplayName(WalletRef{Chain: record.Wallet.Chain, Address: record.Wallet.Address}),
		"observedAt":         now,
		"txHash":             record.TxHash,
		"direction":          string(record.Direction),
		"provider":           record.Provider,
		"rawPayloadPath":     record.RawPayloadPath,
		"materializeFunding": record.Direction == domain.TransactionDirectionInbound,
	}

	if shouldMaterializeInteraction(record) {
		params["counterpartyChain"] = string(record.Counterparty.Chain)
		params["counterpartyAddress"] = record.Counterparty.Address
		params["counterpartyID"] = domain.BuildWalletCanonicalKey(record.Counterparty.Chain, record.Counterparty.Address)
		params["counterpartyDisplayName"] = defaultWalletDisplayName(WalletRef{Chain: record.Counterparty.Chain, Address: record.Counterparty.Address})
		return materializeWalletInteractionCypher, params
	}

	return materializeWalletNodeCypher, params
}

func shouldMaterializeInteraction(record domain.NormalizedTransaction) bool {
	if record.Counterparty == nil {
		return false
	}

	return !strings.EqualFold(
		domain.BuildWalletCanonicalKey(record.Wallet.Chain, record.Wallet.Address),
		domain.BuildWalletCanonicalKey(record.Counterparty.Chain, record.Counterparty.Address),
	)
}

func (m *Neo4jTransactionGraphMaterializer) now() time.Time {
	if m != nil && m.Now != nil {
		return m.Now()
	}

	return time.Now()
}
