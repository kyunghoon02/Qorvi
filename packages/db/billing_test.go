package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeBillingRow struct {
	values []any
	err    error
}

func (r fakeBillingRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for idx := range dest {
		switch target := dest[idx].(type) {
		case *string:
			*target = r.values[idx].(string)
		case **time.Time:
			if r.values[idx] == nil {
				*target = nil
			} else {
				value := r.values[idx].(time.Time)
				*target = &value
			}
		case *time.Time:
			*target = r.values[idx].(time.Time)
		case *[]byte:
			*target = append([]byte(nil), r.values[idx].([]byte)...)
		default:
			panic("unsupported scan target")
		}
	}
	return nil
}

type fakeBillingQuerier struct {
	row   pgx.Row
	query string
	args  []any
}

func (q *fakeBillingQuerier) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	q.query = query
	q.args = args
	return q.row
}

func TestPostgresBillingStoreFindBillingAccountTranslatesNotFound(t *testing.T) {
	t.Parallel()

	store := NewPostgresBillingStore(&fakeBillingQuerier{row: fakeBillingRow{err: pgx.ErrNoRows}})
	_, err := store.FindBillingAccount(context.Background(), "user_123")
	if err != ErrBillingAccountNotFound {
		t.Fatalf("expected ErrBillingAccountNotFound, got %v", err)
	}
}

func TestPostgresBillingStoreUpsertBillingAccount(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)
	querier := &fakeBillingQuerier{
		row: fakeBillingRow{
			values: []any{
				"user_123",
				"ops@qorvi.test",
				"pro",
				"cus_123",
				"sub_123",
				"price_pro_placeholder",
				"active",
				now,
				now,
				now,
			},
		},
	}

	store := NewPostgresBillingStore(querier)
	record, err := store.UpsertBillingAccount(context.Background(), BillingAccountRecord{
		OwnerUserID:          "user_123",
		Email:                "ops@qorvi.test",
		CurrentTier:          domain.PlanPro,
		StripeCustomerID:     "cus_123",
		ActiveSubscriptionID: "sub_123",
		CurrentPriceID:       "price_pro_placeholder",
		Status:               "active",
		CurrentPeriodEnd:     &now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.CurrentTier != domain.PlanPro {
		t.Fatalf("expected pro tier, got %q", record.CurrentTier)
	}
	if querier.query == "" || len(querier.args) == 0 {
		t.Fatal("expected query to be executed")
	}
}

func TestPostgresBillingStoreRecordWebhookEvent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)
	querier := &fakeBillingQuerier{
		row: fakeBillingRow{
			values: []any{
				"evt_123",
				"checkout.session.completed",
				"user_123",
				"cus_123",
				"sub_123",
				"pro",
				"processed",
				[]byte(`{"source":"stripe"}`),
				now,
				now,
			},
		},
	}

	store := NewPostgresBillingStore(querier)
	record, err := store.RecordWebhookEvent(context.Background(), BillingWebhookEventRecord{
		ProviderEventID: "evt_123",
		EventType:       "checkout.session.completed",
		OwnerUserID:     "user_123",
		CustomerID:      "cus_123",
		SubscriptionID:  "sub_123",
		PlanTier:        domain.PlanPro,
		Status:          "processed",
		Payload:         map[string]any{"source": "stripe"},
		ReceivedAt:      now,
		ProcessedAt:     &now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.PlanTier != domain.PlanPro {
		t.Fatalf("expected pro tier, got %q", record.PlanTier)
	}
	if record.Payload["source"] != "stripe" {
		t.Fatalf("unexpected payload: %#v", record.Payload)
	}
}
