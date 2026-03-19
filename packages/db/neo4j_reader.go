package db

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jWalletGraphSignalReader struct {
	Driver   Neo4jDriver
	Database string
}

func NewNeo4jWalletGraphSignalReader(driver Neo4jDriver, database string) *Neo4jWalletGraphSignalReader {
	return &Neo4jWalletGraphSignalReader{
		Driver:   driver,
		Database: database,
	}
}

func (r *Neo4jWalletGraphSignalReader) ReadWalletGraphSignals(
	ctx context.Context,
	plan WalletSummaryQueryPlan,
) (WalletGraphSignals, error) {
	if r == nil || r.Driver == nil {
		return WalletGraphSignals{}, fmt.Errorf("neo4j graph signal reader is nil")
	}

	session := r.Driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: r.Database,
	})
	defer func() {
		_ = session.Close(ctx)
	}()

	result, err := session.Run(ctx, plan.SignalsCypher, plan.SignalsParams)
	if err != nil {
		return WalletGraphSignals{}, fmt.Errorf("run neo4j query: %w", err)
	}

	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return WalletGraphSignals{}, fmt.Errorf("neo4j result error: %w", err)
		}
		return WalletGraphSignals{}, ErrWalletSummaryNotFound
	}

	record := result.Record()
	if record == nil {
		return WalletGraphSignals{}, ErrWalletSummaryNotFound
	}

	values := record.AsMap()

	return WalletGraphSignals{
		ClusterKey:            stringValue(values, "clusterKey"),
		ClusterType:           stringValue(values, "clusterType"),
		ClusterScore:          int(int64Value(values, "clusterScore")),
		ClusterMemberCount:    int64Value(values, "clusterMemberCount"),
		InteractedWalletCount: int64Value(values, "interactedWalletCount"),
		BridgeTransferCount:   int64Value(values, "bridgeTransferCount"),
		CEXProximityCount:     int64Value(values, "cexProximityCount"),
	}, nil
}

func stringValue(values map[string]any, key string) string {
	value := values[key]
	if value == nil {
		return ""
	}

	if str, ok := value.(string); ok {
		return str
	}

	return fmt.Sprint(value)
}

func int64Value(values map[string]any, key string) int64 {
	value := values[key]
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case nil:
		return 0
	default:
		return 0
	}
}
