package db

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/qorvi/qorvi/packages/domain"
)

var ErrBillingAccountNotFound = errors.New("billing account not found")

type postgresBillingQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type postgresBillingExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type BillingAccountRecord struct {
	OwnerUserID          string
	Email                string
	CurrentTier          domain.PlanTier
	StripeCustomerID     string
	ActiveSubscriptionID string
	CurrentPriceID       string
	Status               string
	CurrentPeriodEnd     *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type BillingWebhookEventRecord struct {
	ProviderEventID string
	EventType       string
	OwnerUserID     string
	CustomerID      string
	SubscriptionID  string
	PlanTier        domain.PlanTier
	Status          string
	Payload         map[string]any
	ReceivedAt      time.Time
	ProcessedAt     *time.Time
}

type BillingAccountStore interface {
	FindBillingAccount(context.Context, string) (BillingAccountRecord, error)
	UpsertBillingAccount(context.Context, BillingAccountRecord) (BillingAccountRecord, error)
	RecordWebhookEvent(context.Context, BillingWebhookEventRecord) (BillingWebhookEventRecord, error)
}

type PostgresBillingStore struct {
	Querier postgresBillingQuerier
	Execer  postgresBillingExecer
}

func NewPostgresBillingStore(
	querier postgresBillingQuerier,
	execer ...postgresBillingExecer,
) *PostgresBillingStore {
	var resolvedExecer postgresBillingExecer
	if len(execer) > 0 {
		resolvedExecer = execer[0]
	}
	return &PostgresBillingStore{
		Querier: querier,
		Execer:  resolvedExecer,
	}
}

func NewPostgresBillingStoreFromPool(pool postgresBillingQuerier) *PostgresBillingStore {
	execer, _ := pool.(postgresBillingExecer)
	return NewPostgresBillingStore(pool, execer)
}

const findBillingAccountSQL = `
select
  owner_user_id,
  email,
  current_tier,
  stripe_customer_id,
  active_subscription_id,
  current_price_id,
  status,
  current_period_end,
  created_at,
  updated_at
from billing_accounts
where owner_user_id = $1
`

const upsertBillingAccountSQL = `
insert into billing_accounts (
  owner_user_id,
  email,
  current_tier,
  stripe_customer_id,
  active_subscription_id,
  current_price_id,
  status,
  current_period_end,
  updated_at
) values ($1, $2, $3, $4, $5, $6, $7, $8, now())
on conflict (owner_user_id) do update set
  email = excluded.email,
  current_tier = excluded.current_tier,
  stripe_customer_id = excluded.stripe_customer_id,
  active_subscription_id = excluded.active_subscription_id,
  current_price_id = excluded.current_price_id,
  status = excluded.status,
  current_period_end = excluded.current_period_end,
  updated_at = now()
returning
  owner_user_id,
  email,
  current_tier,
  stripe_customer_id,
  active_subscription_id,
  current_price_id,
  status,
  current_period_end,
  created_at,
  updated_at
`

const recordBillingWebhookEventSQL = `
insert into billing_webhook_events (
  provider_event_id,
  event_type,
  owner_user_id,
  customer_id,
  subscription_id,
  plan_tier,
  status,
  payload,
  received_at,
  processed_at
) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
on conflict (provider_event_id) do update set
  event_type = excluded.event_type,
  owner_user_id = excluded.owner_user_id,
  customer_id = excluded.customer_id,
  subscription_id = excluded.subscription_id,
  plan_tier = excluded.plan_tier,
  status = excluded.status,
  payload = excluded.payload,
  received_at = excluded.received_at,
  processed_at = excluded.processed_at
returning
  provider_event_id,
  event_type,
  owner_user_id,
  customer_id,
  subscription_id,
  plan_tier,
  status,
  payload,
  received_at,
  processed_at
`

func (s *PostgresBillingStore) FindBillingAccount(ctx context.Context, ownerUserID string) (BillingAccountRecord, error) {
	if s == nil || s.Querier == nil {
		return BillingAccountRecord{}, ErrBillingAccountNotFound
	}

	record, err := scanBillingAccountRow(s.Querier.QueryRow(ctx, findBillingAccountSQL, strings.TrimSpace(ownerUserID)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BillingAccountRecord{}, ErrBillingAccountNotFound
		}
		return BillingAccountRecord{}, err
	}
	return record, nil
}

func (s *PostgresBillingStore) UpsertBillingAccount(ctx context.Context, record BillingAccountRecord) (BillingAccountRecord, error) {
	if s == nil || s.Querier == nil {
		return BillingAccountRecord{}, ErrBillingAccountNotFound
	}

	return scanBillingAccountRow(s.Querier.QueryRow(
		ctx,
		upsertBillingAccountSQL,
		strings.TrimSpace(record.OwnerUserID),
		strings.TrimSpace(record.Email),
		strings.TrimSpace(string(record.CurrentTier)),
		strings.TrimSpace(record.StripeCustomerID),
		strings.TrimSpace(record.ActiveSubscriptionID),
		strings.TrimSpace(record.CurrentPriceID),
		strings.TrimSpace(record.Status),
		record.CurrentPeriodEnd,
	))
}

func (s *PostgresBillingStore) RecordWebhookEvent(ctx context.Context, event BillingWebhookEventRecord) (BillingWebhookEventRecord, error) {
	if s == nil || s.Querier == nil {
		return BillingWebhookEventRecord{}, ErrBillingAccountNotFound
	}

	payload := event.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return BillingWebhookEventRecord{}, err
	}

	return scanBillingWebhookEventRow(s.Querier.QueryRow(
		ctx,
		recordBillingWebhookEventSQL,
		strings.TrimSpace(event.ProviderEventID),
		strings.TrimSpace(event.EventType),
		strings.TrimSpace(event.OwnerUserID),
		strings.TrimSpace(event.CustomerID),
		strings.TrimSpace(event.SubscriptionID),
		strings.TrimSpace(string(event.PlanTier)),
		strings.TrimSpace(event.Status),
		payloadJSON,
		event.ReceivedAt,
		event.ProcessedAt,
	))
}

func scanBillingAccountRow(row pgx.Row) (BillingAccountRecord, error) {
	var record BillingAccountRecord
	var currentTier string
	var currentPeriodEnd *time.Time
	if err := row.Scan(
		&record.OwnerUserID,
		&record.Email,
		&currentTier,
		&record.StripeCustomerID,
		&record.ActiveSubscriptionID,
		&record.CurrentPriceID,
		&record.Status,
		&currentPeriodEnd,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return BillingAccountRecord{}, err
	}
	record.CurrentTier = domain.PlanTier(currentTier)
	record.CurrentPeriodEnd = currentPeriodEnd
	return record, nil
}

func scanBillingWebhookEventRow(row pgx.Row) (BillingWebhookEventRecord, error) {
	var record BillingWebhookEventRecord
	var planTier string
	var payloadJSON []byte
	if err := row.Scan(
		&record.ProviderEventID,
		&record.EventType,
		&record.OwnerUserID,
		&record.CustomerID,
		&record.SubscriptionID,
		&planTier,
		&record.Status,
		&payloadJSON,
		&record.ReceivedAt,
		&record.ProcessedAt,
	); err != nil {
		return BillingWebhookEventRecord{}, err
	}
	record.PlanTier = domain.PlanTier(planTier)
	record.Payload = map[string]any{}
	if len(payloadJSON) > 0 {
		if err := json.Unmarshal(payloadJSON, &record.Payload); err != nil {
			return BillingWebhookEventRecord{}, err
		}
	}
	return record, nil
}
