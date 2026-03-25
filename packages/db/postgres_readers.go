package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/flowintel/flowintel/packages/domain"
)

type postgresQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type PostgresWalletIdentityReader struct {
	Querier postgresQuerier
}

type PostgresWalletStatsReader struct {
	Querier postgresQuerier
}

func NewPostgresWalletIdentityReader(querier postgresQuerier) *PostgresWalletIdentityReader {
	return &PostgresWalletIdentityReader{Querier: querier}
}

func NewPostgresWalletStatsReader(querier postgresQuerier) *PostgresWalletStatsReader {
	return &PostgresWalletStatsReader{Querier: querier}
}

func (r *PostgresWalletIdentityReader) ReadWalletIdentity(
	ctx context.Context,
	plan WalletSummaryQueryPlan,
) (WalletSummaryIdentity, error) {
	if r == nil || r.Querier == nil {
		return WalletSummaryIdentity{}, fmt.Errorf("postgres identity reader is nil")
	}

	var identity WalletSummaryIdentity
	if err := r.Querier.QueryRow(ctx, plan.IdentitySQL, plan.IdentityArgs...).Scan(
		&identity.WalletID,
		&identity.Chain,
		&identity.Address,
		&identity.DisplayName,
		&identity.EntityKey,
		&identity.CreatedAt,
		&identity.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WalletSummaryIdentity{}, ErrWalletSummaryNotFound
		}

		return WalletSummaryIdentity{}, fmt.Errorf("scan wallet identity: %w", err)
	}

	return identity, nil
}

func (r *PostgresWalletStatsReader) ReadWalletStats(
	ctx context.Context,
	plan WalletSummaryQueryPlan,
) (WalletSummaryStats, error) {
	if r == nil || r.Querier == nil {
		return WalletSummaryStats{}, fmt.Errorf("postgres stats reader is nil")
	}

	var (
		walletID             string
		stats                WalletSummaryStats
		earliestActivityAt   sql.NullTime
		latestActivityAt     sql.NullTime
		topCounterpartiesRaw []byte
	)

	if err := r.Querier.QueryRow(ctx, plan.StatsSQL, plan.StatsArgs...).Scan(
		&walletID,
		&stats.AsOfDate,
		&stats.TransactionCount,
		&stats.CounterpartyCount,
		&earliestActivityAt,
		&latestActivityAt,
		&stats.IncomingTxCount,
		&stats.OutgoingTxCount,
		&stats.IncomingTxCount7d,
		&stats.OutgoingTxCount7d,
		&stats.IncomingTxCount30d,
		&stats.OutgoingTxCount30d,
		&topCounterpartiesRaw,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WalletSummaryStats{}, ErrWalletSummaryNotFound
		}

		return WalletSummaryStats{}, fmt.Errorf("scan wallet stats: %w", err)
	}

	if earliestActivityAt.Valid {
		value := earliestActivityAt.Time
		stats.EarliestActivityAt = &value
	}
	if latestActivityAt.Valid {
		value := latestActivityAt.Time
		stats.LatestActivityAt = &value
	}
	if len(topCounterpartiesRaw) > 0 {
		counterparties, err := decodeWalletSummaryCounterparties(topCounterpartiesRaw)
		if err != nil {
			return WalletSummaryStats{}, fmt.Errorf("decode top counterparties: %w", err)
		}
		stats.TopCounterparties = counterparties
	}

	return stats, nil
}

func NewPostgresWalletIdentityReaderFromPool(pool postgresQuerier) *PostgresWalletIdentityReader {
	return NewPostgresWalletIdentityReader(pool)
}

func NewPostgresWalletStatsReaderFromPool(pool postgresQuerier) *PostgresWalletStatsReader {
	return NewPostgresWalletStatsReader(pool)
}

func BuildWalletSummaryFromPostgres(
	identity WalletSummaryIdentity,
	stats WalletSummaryStats,
) domain.WalletSummary {
	clusterID := ""
	return domain.WalletSummary{
		Chain:             identity.Chain,
		Address:           identity.Address,
		DisplayName:       identity.DisplayName,
		Labels:            identity.Labels,
		ClusterID:         &clusterID,
		Counterparties:    int(stats.CounterpartyCount),
		LatestActivityAt:  statsTime(stats),
		TopCounterparties: buildDomainCounterparties(stats.TopCounterparties),
		RecentFlow:        buildDomainRecentFlow(stats),
		Indexing: domain.WalletIndexingState{
			Status:             "ready",
			LastIndexedAt:      statsTime(stats),
			CoverageStartAt:    formatOptionalTime(stats.EarliestActivityAt),
			CoverageEndAt:      formatOptionalTime(stats.LatestActivityAt),
			CoverageWindowDays: 0,
		},
		Tags:   []string{"wallet", "postgres"},
		Scores: nil,
	}
}

func statsTime(stats WalletSummaryStats) string {
	if stats.LatestActivityAt == nil {
		return ""
	}
	return stats.LatestActivityAt.UTC().Format("2006-01-02T15:04:05Z07:00")
}

type walletSummaryCounterpartyRecord struct {
	Chain            string                              `json:"chain"`
	Address          string                              `json:"address"`
	EntityKey        string                              `json:"entity_key"`
	EntityType       string                              `json:"entity_type"`
	EntityLabel      string                              `json:"entity_label"`
	InteractionCount int64                               `json:"interaction_count"`
	InboundCount     int64                               `json:"inbound_count"`
	OutboundCount    int64                               `json:"outbound_count"`
	InboundAmount    string                              `json:"inbound_amount"`
	OutboundAmount   string                              `json:"outbound_amount"`
	PrimaryToken     string                              `json:"primary_token"`
	TokenBreakdowns  []walletSummaryTokenBreakdownRecord `json:"token_breakdowns"`
	DirectionLabel   string                              `json:"direction_label"`
	FirstSeenAt      string                              `json:"first_seen_at"`
	LatestActivityAt string                              `json:"latest_activity_at"`
}

type walletSummaryTokenBreakdownRecord struct {
	Symbol         string `json:"symbol"`
	InboundAmount  string `json:"inbound_amount"`
	OutboundAmount string `json:"outbound_amount"`
}

func decodeWalletSummaryCounterparties(raw []byte) ([]WalletSummaryCounterparty, error) {
	var records []walletSummaryCounterpartyRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		return nil, err
	}

	counterparties := make([]WalletSummaryCounterparty, 0, len(records))
	for _, record := range records {
		counterparty := WalletSummaryCounterparty{
			Chain:            domain.Chain(strings.TrimSpace(record.Chain)),
			Address:          strings.TrimSpace(record.Address),
			EntityKey:        strings.TrimSpace(record.EntityKey),
			EntityType:       strings.TrimSpace(record.EntityType),
			EntityLabel:      strings.TrimSpace(record.EntityLabel),
			InteractionCount: record.InteractionCount,
			InboundCount:     record.InboundCount,
			OutboundCount:    record.OutboundCount,
			InboundAmount:    strings.TrimSpace(record.InboundAmount),
			OutboundAmount:   strings.TrimSpace(record.OutboundAmount),
			PrimaryToken:     strings.TrimSpace(record.PrimaryToken),
			TokenBreakdowns:  buildWalletSummaryTokenBreakdowns(record.TokenBreakdowns),
			DirectionLabel:   strings.TrimSpace(record.DirectionLabel),
		}
		if value, err := time.Parse(time.RFC3339, strings.TrimSpace(record.FirstSeenAt)); err == nil {
			timestamp := value.UTC()
			counterparty.FirstSeenAt = &timestamp
		}
		if value, err := time.Parse(time.RFC3339, strings.TrimSpace(record.LatestActivityAt)); err == nil {
			timestamp := value.UTC()
			counterparty.LatestActivityAt = &timestamp
		}
		counterparties = append(counterparties, counterparty)
	}

	return counterparties, nil
}

func buildWalletSummaryTokenBreakdowns(
	records []walletSummaryTokenBreakdownRecord,
) []WalletSummaryCounterpartyTokenSummary {
	items := make([]WalletSummaryCounterpartyTokenSummary, 0, len(records))
	for _, record := range records {
		items = append(items, WalletSummaryCounterpartyTokenSummary{
			Symbol:         strings.TrimSpace(record.Symbol),
			InboundAmount:  strings.TrimSpace(record.InboundAmount),
			OutboundAmount: strings.TrimSpace(record.OutboundAmount),
		})
	}

	return items
}

func buildDomainCounterparties(items []WalletSummaryCounterparty) []domain.WalletCounterparty {
	counterparties := make([]domain.WalletCounterparty, 0, len(items))
	for _, item := range items {
		counterparties = append(counterparties, domain.WalletCounterparty{
			Chain:            item.Chain,
			Address:          item.Address,
			EntityKey:        item.EntityKey,
			EntityType:       item.EntityType,
			EntityLabel:      item.EntityLabel,
			Labels:           item.Labels,
			InteractionCount: int(item.InteractionCount),
			InboundCount:     int(item.InboundCount),
			OutboundCount:    int(item.OutboundCount),
			InboundAmount:    item.InboundAmount,
			OutboundAmount:   item.OutboundAmount,
			PrimaryToken:     item.PrimaryToken,
			TokenBreakdowns:  buildDomainTokenBreakdowns(item.TokenBreakdowns),
			DirectionLabel:   item.DirectionLabel,
			FirstSeenAt:      formatOptionalTime(item.FirstSeenAt),
			LatestActivityAt: formatOptionalTime(item.LatestActivityAt),
		})
	}

	return counterparties
}

func buildDomainTokenBreakdowns(
	items []WalletSummaryCounterpartyTokenSummary,
) []domain.WalletCounterpartyTokenSummary {
	result := make([]domain.WalletCounterpartyTokenSummary, 0, len(items))
	for _, item := range items {
		result = append(result, domain.WalletCounterpartyTokenSummary{
			Symbol:         item.Symbol,
			InboundAmount:  item.InboundAmount,
			OutboundAmount: item.OutboundAmount,
		})
	}

	return result
}

func buildDomainRecentFlow(stats WalletSummaryStats) domain.WalletRecentFlow {
	return domain.WalletRecentFlow{
		IncomingTxCount7d:  int(stats.IncomingTxCount7d),
		OutgoingTxCount7d:  int(stats.OutgoingTxCount7d),
		IncomingTxCount30d: int(stats.IncomingTxCount30d),
		OutgoingTxCount30d: int(stats.OutgoingTxCount30d),
		NetDirection7d:     flowDirection(stats.IncomingTxCount7d, stats.OutgoingTxCount7d),
		NetDirection30d:    flowDirection(stats.IncomingTxCount30d, stats.OutgoingTxCount30d),
	}
}

func flowDirection(incoming int64, outgoing int64) string {
	switch {
	case incoming > outgoing:
		return "inbound"
	case outgoing > incoming:
		return "outbound"
	default:
		return "balanced"
	}
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}

	return value.UTC().Format(time.RFC3339)
}
