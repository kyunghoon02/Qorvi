package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/config"
	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
	"github.com/whalegraph/whalegraph/packages/providers"
)

const workerModeHistoricalBackfillFixture = "historical-backfill-fixture"
const workerModeHistoricalBackfillIngest = "historical-backfill-ingest"

type WalletEnsurer interface {
	EnsureWallet(context.Context, db.WalletRef) (db.WalletSummaryIdentity, error)
}

type NormalizedTransactionWriter interface {
	UpsertNormalizedTransactions(context.Context, []db.NormalizedTransactionWrite) error
}

type HistoricalBackfillJobRunner struct {
	Runner providers.HistoricalBackfillRunner
}

type HistoricalBackfillIngestService struct {
	Runner        HistoricalBackfillJobRunner
	Wallets       WalletEnsurer
	Transactions  NormalizedTransactionWriter
	RawPayloads   db.RawPayloadStore
	ProviderUsage db.ProviderUsageLogStore
	JobRuns       db.JobRunStore
	Now           func() time.Time
}

type HistoricalBackfillIngestReport struct {
	BatchesWritten      int
	ActivitiesFetched   int
	RawPayloadsStored   int
	TransactionsWritten int
	ProvidersSeen       []string
}

func NewHistoricalBackfillJobRunner(registry providers.Registry) HistoricalBackfillJobRunner {
	return HistoricalBackfillJobRunner{
		Runner: providers.NewHistoricalBackfillRunner(registry),
	}
}

func NewHistoricalBackfillIngestService(
	registry providers.Registry,
	wallets WalletEnsurer,
	transactions NormalizedTransactionWriter,
) HistoricalBackfillIngestService {
	return HistoricalBackfillIngestService{
		Runner:       NewHistoricalBackfillJobRunner(registry),
		Wallets:      wallets,
		Transactions: transactions,
		Now:          time.Now,
	}
}

func (r HistoricalBackfillJobRunner) RunFixtureFlow() ([]providers.HistoricalBackfillResult, error) {
	batches := buildHistoricalBackfillFixtureBatches()
	results := make([]providers.HistoricalBackfillResult, 0, len(batches))

	for _, batch := range batches {
		result, err := r.Runner.Run(batch)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

func buildHistoricalBackfillFixtureBatches() []providers.HistoricalBackfillBatch {
	return []providers.HistoricalBackfillBatch{
		providers.CreateHistoricalBackfillBatchFixture(
			providers.ProviderAlchemy,
			domain.ChainEVM,
			"0x1234567890abcdef1234567890abcdef12345678",
		),
		providers.CreateHistoricalBackfillBatchFixture(
			providers.ProviderHelius,
			domain.ChainSolana,
			"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
		),
	}
}

func buildHistoricalBackfillSummary(results []providers.HistoricalBackfillResult) string {
	providersSeen := make([]string, 0, len(results))
	totalActivities := 0

	for _, result := range results {
		providersSeen = append(providersSeen, string(result.Batch.Provider))
		totalActivities += len(result.Activities)
	}

	return fmt.Sprintf(
		"Historical backfill fixture complete (providers=%s, batches=%d, activities=%d)",
		strings.Join(providersSeen, ","),
		len(results),
		totalActivities,
	)
}

func (s HistoricalBackfillIngestService) RunFixtureIngest(ctx context.Context) (HistoricalBackfillIngestReport, error) {
	if s.Wallets == nil {
		return HistoricalBackfillIngestReport{}, fmt.Errorf("wallet store is required")
	}
	if s.Transactions == nil {
		return HistoricalBackfillIngestReport{}, fmt.Errorf("transaction store is required")
	}

	startedAt := s.now().UTC()
	batches := buildHistoricalBackfillFixtureBatches()
	report := HistoricalBackfillIngestReport{
		ProvidersSeen: make([]string, 0, len(batches)),
	}

	for _, batch := range batches {
		batchStartedAt := s.now()
		result, err := s.Runner.Runner.Run(batch)
		if err != nil {
			if logErr := s.recordProviderUsage(ctx, batch.Provider, "historical_backfill", 500, s.now().Sub(batchStartedAt)); logErr != nil {
				return HistoricalBackfillIngestReport{}, logErr
			}
			_ = s.recordJobRun(ctx, db.JobRunEntry{
				JobName:   "historical-backfill-ingest",
				Status:    db.JobRunStatusFailed,
				StartedAt: startedAt,
				FinishedAt: func() *time.Time {
					finishedAt := s.now().UTC()
					return &finishedAt
				}(),
				Details: map[string]any{
					"failed_provider": string(batch.Provider),
					"error":           err.Error(),
				},
			})
			return HistoricalBackfillIngestReport{}, err
		}

		identity, err := s.Wallets.EnsureWallet(ctx, db.WalletRef{
			Chain:   batch.Request.Chain,
			Address: batch.Request.WalletAddress,
		})
		if err != nil {
			return HistoricalBackfillIngestReport{}, err
		}

		activities, storedRawPayloads, err := s.persistRawPayloads(ctx, result.Activities)
		if err != nil {
			_ = s.recordProviderUsage(ctx, batch.Provider, "historical_backfill", 500, s.now().Sub(batchStartedAt))
			return HistoricalBackfillIngestReport{}, err
		}

		transactions, err := providers.NormalizeProviderActivities(activities)
		if err != nil {
			return HistoricalBackfillIngestReport{}, err
		}

		writes := make([]db.NormalizedTransactionWrite, 0, len(transactions))
		for _, tx := range transactions {
			writes = append(writes, db.NormalizedTransactionWrite{
				WalletID:    identity.WalletID,
				Transaction: tx,
			})
		}

		if err := s.Transactions.UpsertNormalizedTransactions(ctx, writes); err != nil {
			_ = s.recordProviderUsage(ctx, batch.Provider, "historical_backfill", 500, s.now().Sub(batchStartedAt))
			return HistoricalBackfillIngestReport{}, err
		}
		if err := s.recordProviderUsage(ctx, batch.Provider, "historical_backfill", 200, s.now().Sub(batchStartedAt)); err != nil {
			return HistoricalBackfillIngestReport{}, err
		}

		report.BatchesWritten++
		report.ActivitiesFetched += len(result.Activities)
		report.RawPayloadsStored += storedRawPayloads
		report.TransactionsWritten += len(writes)
		report.ProvidersSeen = append(report.ProvidersSeen, string(batch.Provider))
	}

	if err := s.recordJobRun(ctx, db.JobRunEntry{
		JobName:   "historical-backfill-ingest",
		Status:    db.JobRunStatusSucceeded,
		StartedAt: startedAt,
		FinishedAt: func() *time.Time {
			finishedAt := s.now().UTC()
			return &finishedAt
		}(),
		Details: map[string]any{
			"providers":    append([]string(nil), report.ProvidersSeen...),
			"batches":      report.BatchesWritten,
			"activities":   report.ActivitiesFetched,
			"raw_payloads": report.RawPayloadsStored,
			"transactions": report.TransactionsWritten,
		},
	}); err != nil {
		return HistoricalBackfillIngestReport{}, err
	}

	return report, nil
}

func buildHistoricalBackfillIngestSummary(report HistoricalBackfillIngestReport) string {
	return fmt.Sprintf(
		"Historical backfill ingest complete (providers=%s, batches=%d, activities=%d, raw_payloads=%d, transactions=%d)",
		strings.Join(report.ProvidersSeen, ","),
		report.BatchesWritten,
		report.ActivitiesFetched,
		report.RawPayloadsStored,
		report.TransactionsWritten,
	)
}

func buildWorkerOutput(
	ctx context.Context,
	mode string,
	env config.WorkerEnv,
	runner HistoricalBackfillJobRunner,
	ingest HistoricalBackfillIngestService,
) (string, error) {
	if mode == workerModeHistoricalBackfillFixture {
		results, err := runner.RunFixtureFlow()
		if err != nil {
			return "", err
		}
		return buildHistoricalBackfillSummary(results), nil
	}
	if mode == workerModeHistoricalBackfillIngest {
		report, err := ingest.RunFixtureIngest(ctx)
		if err != nil {
			return "", err
		}
		return buildHistoricalBackfillIngestSummary(report), nil
	}

	return buildStartupMessage(env), nil
}

func (s HistoricalBackfillIngestService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}

	return time.Now()
}

func (s HistoricalBackfillIngestService) recordProviderUsage(
	ctx context.Context,
	provider providers.ProviderName,
	operation string,
	statusCode int,
	latency time.Duration,
) error {
	if s.ProviderUsage == nil {
		return nil
	}

	return s.ProviderUsage.RecordProviderUsageLog(ctx, db.ProviderUsageLogEntry{
		Provider:   string(provider),
		Operation:  operation,
		StatusCode: statusCode,
		Latency:    latency,
	})
}

func (s HistoricalBackfillIngestService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}

	return s.JobRuns.RecordJobRun(ctx, entry)
}

func (s HistoricalBackfillIngestService) persistRawPayloads(
	ctx context.Context,
	activities []providers.ProviderWalletActivity,
) ([]providers.ProviderWalletActivity, int, error) {
	cloned := make([]providers.ProviderWalletActivity, 0, len(activities))
	stored := 0

	for _, activity := range activities {
		clonedActivity := activity
		clonedActivity.Metadata = cloneActivityMetadata(activity.Metadata)

		if s.RawPayloads != nil {
			descriptor, payload, ok, err := buildRawPayloadRecord(activity)
			if err != nil {
				return nil, 0, err
			}
			if ok {
				if err := s.RawPayloads.StoreRawPayload(ctx, descriptor, payload); err != nil {
					return nil, 0, err
				}
				clonedActivity.Metadata["raw_payload_path"] = descriptor.ObjectKey
				stored++
			}
		}

		cloned = append(cloned, clonedActivity)
	}

	return cloned, stored, nil
}

func buildRawPayloadRecord(
	activity providers.ProviderWalletActivity,
) (db.RawPayloadDescriptor, []byte, bool, error) {
	body, ok := activity.Metadata["raw_payload_body"]
	if !ok {
		return db.RawPayloadDescriptor{}, nil, false, nil
	}

	bodyString, ok := body.(string)
	if !ok || strings.TrimSpace(bodyString) == "" {
		return db.RawPayloadDescriptor{}, nil, false, nil
	}

	objectKey, _ := activity.Metadata["raw_payload_object_key"].(string)
	if strings.TrimSpace(objectKey) == "" {
		objectKey = db.BuildRawPayloadObjectKey(
			string(activity.Provider),
			activity.SourceID,
			activity.ObservedAt,
			activity.WalletAddress+".json",
		)
	}

	contentType, _ := activity.Metadata["raw_payload_content_type"].(string)
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/json"
	}

	sha256, _ := activity.Metadata["raw_payload_sha256"].(string)

	descriptor, err := db.NormalizeRawPayloadDescriptor(db.RawPayloadDescriptor{
		Provider:    string(activity.Provider),
		Operation:   activity.SourceID,
		ContentType: contentType,
		ObjectKey:   objectKey,
		SHA256:      sha256,
		ObservedAt:  activity.ObservedAt,
	})
	if err != nil {
		return db.RawPayloadDescriptor{}, nil, false, err
	}

	return descriptor, []byte(bodyString), true, nil
}

func cloneActivityMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}

	return cloned
}
