package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/db"
)

const workerModeAlertDeliveryRetryBatch = "alert-delivery-retry-batch"

type AlertDeliveryRetryCandidateLoader interface {
	ListRetryableAlertDeliveryAttempts(context.Context, time.Time, int) ([]db.AlertDeliveryRetryCandidate, error)
}

type AlertDeliveryAttemptRetrier interface {
	RetryAlertDeliveryAttempt(context.Context, db.AlertDeliveryRetryCandidate) (AlertDeliveryRetryResult, error)
}

type AlertDeliveryRetryService struct {
	Attempts   AlertDeliveryRetryCandidateLoader
	Deliveries AlertDeliveryAttemptRetrier
	JobRuns    db.JobRunStore
	Now        func() time.Time
}

type AlertDeliveryRetryBatchReport struct {
	AttemptsFound     int
	AttemptsProcessed int
	Delivered         int
	Requeued          int
	Exhausted         int
}

func (s AlertDeliveryRetryService) RunBatch(ctx context.Context, limit int) (AlertDeliveryRetryBatchReport, error) {
	if s.Attempts == nil {
		return AlertDeliveryRetryBatchReport{}, fmt.Errorf("alert delivery attempt loader is required")
	}
	if s.Deliveries == nil {
		return AlertDeliveryRetryBatchReport{}, fmt.Errorf("alert delivery retrier is required")
	}

	startedAt := s.now().UTC()
	candidates, err := s.Attempts.ListRetryableAlertDeliveryAttempts(ctx, startedAt, limit)
	if err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeAlertDeliveryRetryBatch,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details: map[string]any{
				"error": err.Error(),
			},
		})
		return AlertDeliveryRetryBatchReport{}, err
	}

	report := AlertDeliveryRetryBatchReport{
		AttemptsFound: len(candidates),
	}
	for _, candidate := range candidates {
		result, err := s.Deliveries.RetryAlertDeliveryAttempt(ctx, candidate)
		if err != nil {
			_ = s.recordJobRun(ctx, db.JobRunEntry{
				JobName:    workerModeAlertDeliveryRetryBatch,
				Status:     db.JobRunStatusFailed,
				StartedAt:  startedAt,
				FinishedAt: pointerToTime(s.now().UTC()),
				Details: map[string]any{
					"attempt_id": candidate.Attempt.ID,
					"event_id":   candidate.Event.ID,
					"error":      err.Error(),
				},
			})
			return report, err
		}
		report.AttemptsProcessed++
		if result.Delivered {
			report.Delivered++
		}
		if result.Requeued {
			report.Requeued++
		}
		if result.Exhausted {
			report.Exhausted++
		}
	}

	if err := s.recordJobRun(ctx, db.JobRunEntry{
		JobName:   workerModeAlertDeliveryRetryBatch,
		Status:    db.JobRunStatusSucceeded,
		StartedAt: startedAt,
		FinishedAt: func() *time.Time {
			finishedAt := s.now().UTC()
			return &finishedAt
		}(),
		Details: map[string]any{
			"attempts_found":     report.AttemptsFound,
			"attempts_processed": report.AttemptsProcessed,
			"delivered":          report.Delivered,
			"requeued":           report.Requeued,
			"exhausted":          report.Exhausted,
		},
	}); err != nil {
		return AlertDeliveryRetryBatchReport{}, err
	}

	return report, nil
}

func buildAlertDeliveryRetryBatchSummary(report AlertDeliveryRetryBatchReport) string {
	return fmt.Sprintf(
		"Alert delivery retry batch complete (found=%d, processed=%d, delivered=%d, requeued=%d, exhausted=%d)",
		report.AttemptsFound,
		report.AttemptsProcessed,
		report.Delivered,
		report.Requeued,
		report.Exhausted,
	)
}

func alertDeliveryRetryBatchLimitFromEnv() int {
	value := strings.TrimSpace(os.Getenv("FLOWINTEL_ALERT_DELIVERY_RETRY_BATCH_LIMIT"))
	if value == "" {
		return 25
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 25
	}
	if parsed > 100 {
		return 100
	}
	return parsed
}

func alertDeliveryRetryLimitFromEnv() int {
	value := strings.TrimSpace(os.Getenv("FLOWINTEL_ALERT_DELIVERY_RETRY_LIMIT"))
	if value == "" {
		return 3
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 3
	}
	return parsed
}

func alertDeliveryRetryBaseDelayFromEnv() time.Duration {
	value := strings.TrimSpace(os.Getenv("FLOWINTEL_ALERT_DELIVERY_RETRY_BASE_DELAY_SECONDS"))
	if value == "" {
		return time.Minute
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return time.Minute
	}
	return time.Duration(parsed) * time.Second
}

func (s AlertDeliveryRetryService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s AlertDeliveryRetryService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}
	return s.JobRuns.RecordJobRun(ctx, entry)
}
