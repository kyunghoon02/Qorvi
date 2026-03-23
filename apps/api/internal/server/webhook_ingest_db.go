package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
	"github.com/whalegraph/whalegraph/packages/providers"
)

type webhookWalletEnsurer interface {
	EnsureWallet(context.Context, db.WalletRef) (db.WalletSummaryIdentity, error)
}

type webhookWalletEntityAssignmentWriter interface {
	UpsertHeuristicEntityAssignments(context.Context, []db.WalletEntityAssignment) error
}

type providerWebhookPersistingService struct {
	Wallets        webhookWalletEnsurer
	EntityAssign   webhookWalletEntityAssignmentWriter
	Transactions   db.NormalizedTransactionStore
	DailyStats     db.WalletDailyStatsRefresher
	Graph          db.TransactionGraphMaterializer
	GraphCache     db.WalletGraphCache
	GraphSnapshots db.WalletGraphSnapshotStore
	SummaryCache   db.WalletSummaryCache
	Dedup          db.IngestDedupStore
	RawPayloads    db.RawPayloadStore
	ProviderUsage  db.ProviderUsageLogStore
	JobRuns        db.JobRunStore
	Now            func() time.Time
}

func newProviderWebhookPersistingService(
	wallets webhookWalletEnsurer,
	entityAssign webhookWalletEntityAssignmentWriter,
	transactions db.NormalizedTransactionStore,
	dailyStats db.WalletDailyStatsRefresher,
	graph db.TransactionGraphMaterializer,
	graphCache db.WalletGraphCache,
	graphSnapshots db.WalletGraphSnapshotStore,
	summaryCache db.WalletSummaryCache,
	dedup db.IngestDedupStore,
	rawPayloads db.RawPayloadStore,
	providerUsage db.ProviderUsageLogStore,
	jobRuns db.JobRunStore,
) WebhookIngestService {
	return providerWebhookPersistingService{
		Wallets:        wallets,
		EntityAssign:   entityAssign,
		Transactions:   transactions,
		DailyStats:     dailyStats,
		Graph:          graph,
		GraphCache:     graphCache,
		GraphSnapshots: graphSnapshots,
		SummaryCache:   summaryCache,
		Dedup:          dedup,
		RawPayloads:    rawPayloads,
		ProviderUsage:  providerUsage,
		JobRuns:        jobRuns,
		Now:            time.Now,
	}
}

func (s providerWebhookPersistingService) IngestAlchemyAddressActivity(
	ctx context.Context,
	payload AlchemyAddressActivityWebhook,
) (WebhookIngestResult, error) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return WebhookIngestResult{}, fmt.Errorf("marshal alchemy webhook payload: %w", err)
	}

	return s.ingestAlchemyAddressActivity(ctx, payload, rawPayload)
}

func (s providerWebhookPersistingService) IngestProviderWebhook(
	ctx context.Context,
	provider string,
	raw json.RawMessage,
) (WebhookIngestResult, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "alchemy":
		var payload AlchemyAddressActivityWebhook
		if err := json.Unmarshal(raw, &payload); err != nil {
			return WebhookIngestResult{}, err
		}
		if err := validateAlchemyAddressActivityWebhook(payload); err != nil {
			return WebhookIngestResult{}, err
		}
		return s.ingestAlchemyAddressActivity(ctx, payload, raw)
	case "helius":
		payload, err := parseHeliusAddressActivityWebhook(raw)
		if err != nil {
			return WebhookIngestResult{}, err
		}
		return s.ingestHeliusAddressActivity(ctx, payload, raw)
	default:
		return newCountingWebhookIngestService().IngestProviderWebhook(ctx, provider, raw)
	}
}

func (s providerWebhookPersistingService) ingestAlchemyAddressActivity(
	ctx context.Context,
	payload AlchemyAddressActivityWebhook,
	rawPayload []byte,
) (WebhookIngestResult, error) {
	startedAt := s.now()
	descriptor := s.buildRawPayloadDescriptor(payload, rawPayload)
	if s.RawPayloads != nil {
		if err := s.RawPayloads.StoreRawPayload(ctx, descriptor, rawPayload); err != nil {
			return WebhookIngestResult{}, fmt.Errorf("store raw payload: %w", err)
		}
	}

	activities := buildAlchemyWebhookActivities(payload, descriptor.ObjectKey)
	if err := s.assignHeuristicEntities(ctx, activities); err != nil {
		return WebhookIngestResult{}, err
	}
	writes := make([]db.NormalizedTransactionWrite, 0, len(activities))
	claimedKeys := make([]string, 0, len(activities))
	duplicates := 0
	for _, activity := range activities {
		tx, err := providers.NormalizeProviderActivity(activity)
		if err != nil {
			return WebhookIngestResult{}, fmt.Errorf("normalize activity: %w", err)
		}
		key, claimed, err := s.claimNormalizedTransaction(ctx, tx)
		if err != nil {
			return WebhookIngestResult{}, err
		}
		if !claimed {
			duplicates++
			continue
		}
		claimedKeys = append(claimedKeys, key)

		identity, err := s.Wallets.EnsureWallet(ctx, db.WalletRef{
			Chain:   activity.Chain,
			Address: activity.WalletAddress,
		})
		if err != nil {
			_ = s.releaseDedupKeys(ctx, claimedKeys)
			return WebhookIngestResult{}, fmt.Errorf("ensure wallet: %w", err)
		}
		writes = append(writes, db.NormalizedTransactionWrite{
			WalletID:    identity.WalletID,
			Transaction: tx,
		})
	}

	if len(writes) > 0 && s.Transactions != nil {
		if err := s.Transactions.UpsertNormalizedTransactions(ctx, writes); err != nil {
			_ = s.releaseDedupKeys(ctx, claimedKeys)
			return WebhookIngestResult{}, fmt.Errorf("upsert normalized transactions: %w", err)
		}
	}
	if err := s.refreshDailyStats(ctx, writes); err != nil {
		_ = s.releaseDedupKeys(ctx, claimedKeys)
		return WebhookIngestResult{}, fmt.Errorf("refresh wallet daily stats: %w", err)
	}
	if len(writes) > 0 && s.Graph != nil {
		if err := s.Graph.MaterializeNormalizedTransactions(ctx, writes); err != nil {
			_ = s.releaseDedupKeys(ctx, claimedKeys)
			return WebhookIngestResult{}, fmt.Errorf("materialize transaction graph: %w", err)
		}
	}
	if err := s.invalidateWalletSummaries(ctx, writes); err != nil {
		_ = s.releaseDedupKeys(ctx, claimedKeys)
		return WebhookIngestResult{}, fmt.Errorf("invalidate wallet summary cache: %w", err)
	}
	if err := s.invalidateWalletGraphs(ctx, writes); err != nil {
		_ = s.releaseDedupKeys(ctx, claimedKeys)
		return WebhookIngestResult{}, fmt.Errorf("invalidate wallet graph snapshot: %w", err)
	}

	latency := s.now().Sub(startedAt)
	_ = s.recordProviderUsage(ctx, "alchemy", "address_activity_webhook", 202, latency)
	_ = s.recordJobRun(ctx, db.JobRunEntry{
		JobName:   "alchemy-address-activity-webhook",
		Status:    db.JobRunStatusSucceeded,
		StartedAt: startedAt,
		FinishedAt: func() *time.Time {
			finishedAt := s.now().UTC()
			return &finishedAt
		}(),
		Details: map[string]any{
			"webhook_id":   payload.WebhookID,
			"event_id":     payload.ID,
			"activities":   len(payload.Event.Activity),
			"transactions": len(writes),
			"duplicates":   duplicates,
			"raw_payload":  descriptor.ObjectKey,
			"network":      payload.Event.Network,
		},
	})

	return WebhookIngestResult{
		AcceptedCount: len(payload.Event.Activity),
		EventKind:     "address_activity",
	}, nil
}

func (s providerWebhookPersistingService) ingestHeliusAddressActivity(
	ctx context.Context,
	payload []HeliusAddressActivityWebhookEvent,
	rawPayload []byte,
) (WebhookIngestResult, error) {
	startedAt := s.now()
	descriptor := s.buildHeliusRawPayloadDescriptor(payload, rawPayload)
	if s.RawPayloads != nil {
		if err := s.RawPayloads.StoreRawPayload(ctx, descriptor, rawPayload); err != nil {
			return WebhookIngestResult{}, fmt.Errorf("store helius raw payload: %w", err)
		}
	}

	activities := buildHeliusWebhookActivities(payload, descriptor.ObjectKey, startedAt.UTC())
	if err := s.assignHeuristicEntities(ctx, activities); err != nil {
		return WebhookIngestResult{}, err
	}
	writes := make([]db.NormalizedTransactionWrite, 0, len(activities))
	claimedKeys := make([]string, 0, len(activities))
	duplicates := 0
	for _, activity := range activities {
		tx, err := providers.NormalizeProviderActivity(activity)
		if err != nil {
			return WebhookIngestResult{}, fmt.Errorf("normalize helius activity: %w", err)
		}
		key, claimed, err := s.claimNormalizedTransaction(ctx, tx)
		if err != nil {
			return WebhookIngestResult{}, err
		}
		if !claimed {
			duplicates++
			continue
		}
		claimedKeys = append(claimedKeys, key)

		identity, err := s.Wallets.EnsureWallet(ctx, db.WalletRef{
			Chain:   activity.Chain,
			Address: activity.WalletAddress,
		})
		if err != nil {
			_ = s.releaseDedupKeys(ctx, claimedKeys)
			return WebhookIngestResult{}, fmt.Errorf("ensure helius wallet: %w", err)
		}
		writes = append(writes, db.NormalizedTransactionWrite{
			WalletID:    identity.WalletID,
			Transaction: tx,
		})
	}

	if len(writes) > 0 && s.Transactions != nil {
		if err := s.Transactions.UpsertNormalizedTransactions(ctx, writes); err != nil {
			_ = s.releaseDedupKeys(ctx, claimedKeys)
			return WebhookIngestResult{}, fmt.Errorf("upsert helius normalized transactions: %w", err)
		}
	}
	if err := s.refreshDailyStats(ctx, writes); err != nil {
		_ = s.releaseDedupKeys(ctx, claimedKeys)
		return WebhookIngestResult{}, fmt.Errorf("refresh wallet daily stats: %w", err)
	}
	if len(writes) > 0 && s.Graph != nil {
		if err := s.Graph.MaterializeNormalizedTransactions(ctx, writes); err != nil {
			_ = s.releaseDedupKeys(ctx, claimedKeys)
			return WebhookIngestResult{}, fmt.Errorf("materialize helius transaction graph: %w", err)
		}
	}
	if err := s.invalidateWalletSummaries(ctx, writes); err != nil {
		_ = s.releaseDedupKeys(ctx, claimedKeys)
		return WebhookIngestResult{}, fmt.Errorf("invalidate wallet summary cache: %w", err)
	}
	if err := s.invalidateWalletGraphs(ctx, writes); err != nil {
		_ = s.releaseDedupKeys(ctx, claimedKeys)
		return WebhookIngestResult{}, fmt.Errorf("invalidate wallet graph snapshot: %w", err)
	}

	latency := s.now().Sub(startedAt)
	_ = s.recordProviderUsage(ctx, "helius", "address_activity_webhook", 202, latency)
	_ = s.recordJobRun(ctx, db.JobRunEntry{
		JobName:   "helius-address-activity-webhook",
		Status:    db.JobRunStatusSucceeded,
		StartedAt: startedAt,
		FinishedAt: func() *time.Time {
			finishedAt := s.now().UTC()
			return &finishedAt
		}(),
		Details: map[string]any{
			"events":       len(payload),
			"transactions": len(writes),
			"duplicates":   duplicates,
			"raw_payload":  descriptor.ObjectKey,
		},
	})

	return WebhookIngestResult{
		AcceptedCount: len(payload),
		EventKind:     "webhook_batch",
	}, nil
}

func (s providerWebhookPersistingService) buildRawPayloadDescriptor(
	payload AlchemyAddressActivityWebhook,
	raw []byte,
) db.RawPayloadDescriptor {
	observedAt := s.now().UTC()
	identifier := strings.TrimSpace(payload.ID)
	if identifier == "" {
		identifier = "event"
	}

	return db.RawPayloadDescriptor{
		Provider:    "alchemy",
		Operation:   "address_activity_webhook",
		ContentType: "application/json",
		ObjectKey: db.BuildRawPayloadObjectKey(
			"alchemy",
			"address_activity_webhook",
			observedAt,
			identifier+".json",
		),
		SHA256:     db.RawPayloadSHA256(raw),
		ObservedAt: observedAt,
	}
}

func (s providerWebhookPersistingService) buildHeliusRawPayloadDescriptor(
	payload []HeliusAddressActivityWebhookEvent,
	raw []byte,
) db.RawPayloadDescriptor {
	observedAt := s.now().UTC()
	identifier := "batch.json"
	if len(payload) > 0 && strings.TrimSpace(payload[0].Signature) != "" {
		identifier = strings.TrimSpace(payload[0].Signature) + ".json"
	}

	return db.RawPayloadDescriptor{
		Provider:    "helius",
		Operation:   "address_activity_webhook",
		ContentType: "application/json",
		ObjectKey: db.BuildRawPayloadObjectKey(
			"helius",
			"address_activity_webhook",
			observedAt,
			identifier,
		),
		SHA256:     db.RawPayloadSHA256(raw),
		ObservedAt: observedAt,
	}
}

func buildAlchemyWebhookActivities(
	payload AlchemyAddressActivityWebhook,
	rawPayloadPath string,
) []providers.ProviderWalletActivity {
	chain := chainFromAlchemyNetwork(payload.Event.Network)
	observedAt := parseWebhookCreatedAt(payload.CreatedAt)
	activities := make([]providers.ProviderWalletActivity, 0, len(payload.Event.Activity)*2)

	for index, activity := range payload.Event.Activity {
		from := strings.TrimSpace(activity.FromAddress)
		to := strings.TrimSpace(activity.ToAddress)
		hash := strings.TrimSpace(activity.Hash)
		category := strings.TrimSpace(activity.Category)
		asset := strings.TrimSpace(activity.Asset)

		if from != "" {
			activities = append(activities, providers.CreateProviderActivityFixture(providers.ProviderActivityFixtureInput{
				Provider:      providers.ProviderAlchemy,
				Chain:         chain,
				WalletAddress: from,
				SourceID:      "alchemy_address_activity_webhook",
				Kind:          categoryOrDefault(category),
				Confidence:    0.95,
				ObservedAt:    observedAt.Add(time.Duration(index) * time.Second),
				Metadata: map[string]any{
					"tx_hash":              hash,
					"direction":            outboundDirection(from, to),
					"counterparty_address": to,
					"raw_payload_path":     rawPayloadPath,
					"token_symbol":         asset,
				},
			}))
		}
		if to != "" && !strings.EqualFold(to, from) {
			activities = append(activities, providers.CreateProviderActivityFixture(providers.ProviderActivityFixtureInput{
				Provider:      providers.ProviderAlchemy,
				Chain:         chain,
				WalletAddress: to,
				SourceID:      "alchemy_address_activity_webhook",
				Kind:          categoryOrDefault(category),
				Confidence:    0.95,
				ObservedAt:    observedAt.Add(time.Duration(index) * time.Second),
				Metadata: map[string]any{
					"tx_hash":              hash,
					"direction":            string(domain.TransactionDirectionInbound),
					"counterparty_address": from,
					"raw_payload_path":     rawPayloadPath,
					"token_symbol":         asset,
				},
			}))
		}
	}

	return activities
}

func outboundDirection(from string, to string) string {
	if strings.EqualFold(from, to) && from != "" {
		return string(domain.TransactionDirectionSelf)
	}

	return string(domain.TransactionDirectionOutbound)
}

func categoryOrDefault(category string) string {
	if strings.TrimSpace(category) == "" {
		return "transfer"
	}

	return strings.TrimSpace(category)
}

func chainFromAlchemyNetwork(network string) domain.Chain {
	normalized := strings.ToUpper(strings.TrimSpace(network))
	if strings.Contains(normalized, "SOL") {
		return domain.ChainSolana
	}

	return domain.ChainEVM
}

func parseWebhookCreatedAt(raw string) time.Time {
	if value, err := time.Parse(time.RFC3339, strings.TrimSpace(raw)); err == nil {
		return value.UTC()
	}

	return time.Now().UTC()
}

func (s providerWebhookPersistingService) recordProviderUsage(
	ctx context.Context,
	provider string,
	operation string,
	statusCode int,
	latency time.Duration,
) error {
	if s.ProviderUsage == nil {
		return nil
	}
	return s.ProviderUsage.RecordProviderUsageLog(ctx, db.ProviderUsageLogEntry{
		Provider:   provider,
		Operation:  operation,
		StatusCode: statusCode,
		Latency:    latency,
	})
}

func (s providerWebhookPersistingService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}
	return s.JobRuns.RecordJobRun(ctx, entry)
}

func (s providerWebhookPersistingService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s providerWebhookPersistingService) claimNormalizedTransaction(
	ctx context.Context,
	tx domain.NormalizedTransaction,
) (string, bool, error) {
	if s.Dedup == nil {
		return "", true, nil
	}

	key := db.BuildIngestDedupKey("normalized-transaction", domain.BuildTransactionCanonicalKey(tx))
	claimed, err := s.Dedup.Claim(ctx, key, 90*24*time.Hour)
	if err != nil {
		return "", false, fmt.Errorf("claim webhook transaction dedup: %w", err)
	}

	return key, claimed, nil
}

func (s providerWebhookPersistingService) releaseDedupKeys(ctx context.Context, keys []string) error {
	if s.Dedup == nil {
		return nil
	}
	for _, key := range keys {
		if err := s.Dedup.Release(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (s providerWebhookPersistingService) refreshDailyStats(
	ctx context.Context,
	writes []db.NormalizedTransactionWrite,
) error {
	if s.DailyStats == nil || len(writes) == 0 {
		return nil
	}

	walletIDs := make(map[string]struct{}, len(writes))
	for _, write := range writes {
		walletID := strings.TrimSpace(write.WalletID)
		if walletID == "" {
			continue
		}
		walletIDs[walletID] = struct{}{}
	}

	for walletID := range walletIDs {
		if err := s.DailyStats.RefreshWalletDailyStats(ctx, walletID); err != nil {
			return err
		}
	}

	return nil
}

func (s providerWebhookPersistingService) invalidateWalletSummaries(
	ctx context.Context,
	writes []db.NormalizedTransactionWrite,
) error {
	if s.SummaryCache == nil || len(writes) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(writes))
	for _, write := range writes {
		ref := db.WalletRef{
			Chain:   write.Transaction.Wallet.Chain,
			Address: write.Transaction.Wallet.Address,
		}
		key := domain.BuildWalletCanonicalKey(ref.Chain, ref.Address)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if err := db.InvalidateWalletSummaryCache(ctx, s.SummaryCache, ref); err != nil {
			return err
		}
	}

	return nil
}

func (s providerWebhookPersistingService) invalidateWalletGraphs(
	ctx context.Context,
	writes []db.NormalizedTransactionWrite,
) error {
	if len(writes) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(writes))
	for _, write := range writes {
		ref := db.WalletRef{
			Chain:   write.Transaction.Wallet.Chain,
			Address: write.Transaction.Wallet.Address,
		}
		key := domain.BuildWalletCanonicalKey(ref.Chain, ref.Address)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if err := db.InvalidateWalletGraphSnapshot(ctx, s.GraphCache, s.GraphSnapshots, ref); err != nil {
			return err
		}
	}

	return nil
}

func (s providerWebhookPersistingService) assignHeuristicEntities(
	ctx context.Context,
	activities []providers.ProviderWalletActivity,
) error {
	if s.EntityAssign == nil || len(activities) == 0 {
		return nil
	}

	hints := providers.DeriveHeuristicEntityAssignments(activities)
	if len(hints) == 0 {
		return nil
	}

	assignments := make([]db.WalletEntityAssignment, 0, len(hints))
	for _, hint := range hints {
		assignments = append(assignments, db.WalletEntityAssignment{
			Chain:       hint.Chain,
			Address:     hint.Address,
			EntityKey:   hint.EntityKey,
			EntityType:  hint.EntityType,
			EntityLabel: hint.EntityLabel,
			Source:      hint.Source,
		})
	}

	return s.EntityAssign.UpsertHeuristicEntityAssignments(ctx, assignments)
}
