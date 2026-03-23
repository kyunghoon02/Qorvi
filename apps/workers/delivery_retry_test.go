package main

import (
	"context"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type fakeAlertDeliveryRetryCandidateLoader struct {
	candidates []db.AlertDeliveryRetryCandidate
}

func (l fakeAlertDeliveryRetryCandidateLoader) ListRetryableAlertDeliveryAttempts(_ context.Context, _ time.Time, _ int) ([]db.AlertDeliveryRetryCandidate, error) {
	return append([]db.AlertDeliveryRetryCandidate(nil), l.candidates...), nil
}

type fakeAlertDeliveryRetrier struct {
	results []AlertDeliveryRetryResult
	index   int
}

func (r *fakeAlertDeliveryRetrier) RetryAlertDeliveryAttempt(_ context.Context, _ db.AlertDeliveryRetryCandidate) (AlertDeliveryRetryResult, error) {
	result := r.results[r.index]
	r.index++
	return result, nil
}

func TestAlertDeliveryRetryServiceRunBatch(t *testing.T) {
	t.Parallel()

	service := AlertDeliveryRetryService{
		Attempts: fakeAlertDeliveryRetryCandidateLoader{
			candidates: []db.AlertDeliveryRetryCandidate{
				{
					Attempt: domain.AlertDeliveryAttempt{ID: "attempt_1"},
					Event:   domain.AlertEvent{ID: "evt_1"},
				},
				{
					Attempt: domain.AlertDeliveryAttempt{ID: "attempt_2"},
					Event:   domain.AlertEvent{ID: "evt_2"},
				},
			},
		},
		Deliveries: &fakeAlertDeliveryRetrier{
			results: []AlertDeliveryRetryResult{
				{Status: domain.AlertDeliveryStatusDelivered, Delivered: true},
				{Status: domain.AlertDeliveryStatusFailed, Requeued: true},
			},
		},
		JobRuns: &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 21, 14, 0, 0, 0, time.UTC)
		},
	}

	report, err := service.RunBatch(t.Context(), 10)
	if err != nil {
		t.Fatalf("RunBatch returned error: %v", err)
	}
	if report.AttemptsFound != 2 || report.AttemptsProcessed != 2 {
		t.Fatalf("unexpected retry report %#v", report)
	}
	if report.Delivered != 1 || report.Requeued != 1 || report.Exhausted != 0 {
		t.Fatalf("unexpected retry counters %#v", report)
	}
}
