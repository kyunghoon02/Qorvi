package db

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/whalegraph/whalegraph/packages/billing"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type BillingAccountSyncReader interface {
	ListBillingAccountsForSubscriptionSync(context.Context, int) ([]BillingAccountRecord, error)
}

type postgresBillingStripeQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type PostgresBillingCheckoutSessionStore struct {
	Querier postgresBillingStripeQuerier
}

type PostgresBillingSubscriptionStore struct {
	Querier postgresBillingStripeQuerier
}

type PostgresBillingSubscriptionReconciliationStore struct {
	Querier postgresBillingStripeQuerier
	Execer  postgresBillingExecer
}

func NewPostgresBillingCheckoutSessionStoreFromPool(querier postgresBillingStripeQuerier) *PostgresBillingCheckoutSessionStore {
	return &PostgresBillingCheckoutSessionStore{Querier: querier}
}

func NewPostgresBillingSubscriptionStoreFromPool(querier postgresBillingStripeQuerier) *PostgresBillingSubscriptionStore {
	return &PostgresBillingSubscriptionStore{Querier: querier}
}

func NewPostgresBillingSubscriptionReconciliationStoreFromPool(querier postgresBillingStripeQuerier) *PostgresBillingSubscriptionReconciliationStore {
	execer, _ := querier.(postgresBillingExecer)
	return &PostgresBillingSubscriptionReconciliationStore{
		Querier: querier,
		Execer:  execer,
	}
}

const upsertBillingCheckoutSessionSQL = `
insert into billing_checkout_sessions (
  session_id,
  customer_id,
  customer_email,
  subscription_id,
  tier,
  stripe_price_id,
  status,
  success_url,
  cancel_url,
  metadata,
  created_at,
  updated_at,
  completed_at
) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
on conflict (session_id) do update set
  customer_id = excluded.customer_id,
  customer_email = excluded.customer_email,
  subscription_id = excluded.subscription_id,
  tier = excluded.tier,
  stripe_price_id = excluded.stripe_price_id,
  status = excluded.status,
  success_url = excluded.success_url,
  cancel_url = excluded.cancel_url,
  metadata = excluded.metadata,
  created_at = excluded.created_at,
  updated_at = excluded.updated_at,
  completed_at = excluded.completed_at
returning
  session_id, customer_id, customer_email, subscription_id, tier, stripe_price_id, status,
  success_url, cancel_url, metadata, created_at, updated_at, completed_at
`

const findBillingCheckoutSessionSQL = `
select
  session_id, customer_id, customer_email, subscription_id, tier, stripe_price_id, status,
  success_url, cancel_url, metadata, created_at, updated_at, completed_at
from billing_checkout_sessions
where session_id = $1
`

const markBillingCheckoutSessionCompletedSQL = `
update billing_checkout_sessions
set status = 'completed', completed_at = $2, updated_at = $2
where session_id = $1
returning
  session_id, customer_id, customer_email, subscription_id, tier, stripe_price_id, status,
  success_url, cancel_url, metadata, created_at, updated_at, completed_at
`

const upsertBillingSubscriptionSQL = `
insert into billing_subscriptions (
  subscription_id,
  customer_id,
  customer_email,
  stripe_price_id,
  tier,
  status,
  current_period_start,
  current_period_end,
  cancel_at,
  canceled_at,
  metadata,
  synced_at,
  updated_at
) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
on conflict (subscription_id) do update set
  customer_id = excluded.customer_id,
  customer_email = excluded.customer_email,
  stripe_price_id = excluded.stripe_price_id,
  tier = excluded.tier,
  status = excluded.status,
  current_period_start = excluded.current_period_start,
  current_period_end = excluded.current_period_end,
  cancel_at = excluded.cancel_at,
  canceled_at = excluded.canceled_at,
  metadata = excluded.metadata,
  synced_at = excluded.synced_at,
  updated_at = excluded.updated_at
returning
  subscription_id, customer_id, customer_email, stripe_price_id, tier, status,
  current_period_start, current_period_end, cancel_at, canceled_at, metadata, synced_at, updated_at
`

const findBillingSubscriptionByCustomerIDSQL = `
select
  subscription_id, customer_id, customer_email, stripe_price_id, tier, status,
  current_period_start, current_period_end, cancel_at, canceled_at, metadata, synced_at, updated_at
from billing_subscriptions
where customer_id = $1
order by updated_at desc
limit 1
`

const findBillingSubscriptionBySubscriptionIDSQL = `
select
  subscription_id, customer_id, customer_email, stripe_price_id, tier, status,
  current_period_start, current_period_end, cancel_at, canceled_at, metadata, synced_at, updated_at
from billing_subscriptions
where subscription_id = $1
`

const insertBillingSubscriptionReconciliationSQL = `
insert into billing_subscription_reconciliations (
  event_id,
  provider,
  customer_id,
  subscription_id,
  previous_tier,
  current_tier,
  stripe_price_id,
  status,
  observed_at,
  reconciled_at,
  notes,
  metadata
) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
`

func (s *PostgresBillingCheckoutSessionStore) UpsertBillingCheckoutSession(
	ctx context.Context,
	record billing.StripeCheckoutSessionRecord,
) (billing.StripeCheckoutSessionRecord, error) {
	if s == nil || s.Querier == nil {
		return billing.StripeCheckoutSessionRecord{}, ErrBillingAccountNotFound
	}
	payload, err := marshalStringMap(record.Metadata)
	if err != nil {
		return billing.StripeCheckoutSessionRecord{}, err
	}
	return scanBillingCheckoutSessionRow(s.Querier.QueryRow(ctx, upsertBillingCheckoutSessionSQL,
		record.SessionID,
		record.CustomerID,
		record.CustomerEmail,
		record.SubscriptionID,
		string(record.Tier),
		record.StripePriceID,
		string(record.Status),
		record.SuccessURL,
		record.CancelURL,
		payload,
		record.CreatedAt,
		record.UpdatedAt,
		record.CompletedAt,
	))
}

func (s *PostgresBillingCheckoutSessionStore) FindBillingCheckoutSession(
	ctx context.Context,
	sessionID string,
) (billing.StripeCheckoutSessionRecord, error) {
	if s == nil || s.Querier == nil {
		return billing.StripeCheckoutSessionRecord{}, ErrBillingAccountNotFound
	}
	record, err := scanBillingCheckoutSessionRow(s.Querier.QueryRow(ctx, findBillingCheckoutSessionSQL, strings.TrimSpace(sessionID)))
	if err != nil {
		if err == pgx.ErrNoRows {
			return billing.StripeCheckoutSessionRecord{}, ErrBillingAccountNotFound
		}
		return billing.StripeCheckoutSessionRecord{}, err
	}
	return record, nil
}

func (s *PostgresBillingCheckoutSessionStore) MarkBillingCheckoutSessionCompleted(
	ctx context.Context,
	sessionID string,
	completedAt time.Time,
) (billing.StripeCheckoutSessionRecord, error) {
	if s == nil || s.Querier == nil {
		return billing.StripeCheckoutSessionRecord{}, ErrBillingAccountNotFound
	}
	return scanBillingCheckoutSessionRow(s.Querier.QueryRow(ctx, markBillingCheckoutSessionCompletedSQL, strings.TrimSpace(sessionID), completedAt.UTC()))
}

func (s *PostgresBillingSubscriptionStore) UpsertBillingSubscription(
	ctx context.Context,
	record billing.StripeSubscriptionRecord,
) (billing.StripeSubscriptionRecord, error) {
	if s == nil || s.Querier == nil {
		return billing.StripeSubscriptionRecord{}, ErrBillingAccountNotFound
	}
	payload, err := marshalStringMap(record.Metadata)
	if err != nil {
		return billing.StripeSubscriptionRecord{}, err
	}
	return scanBillingSubscriptionRow(s.Querier.QueryRow(ctx, upsertBillingSubscriptionSQL,
		record.SubscriptionID,
		record.CustomerID,
		record.CustomerEmail,
		record.StripePriceID,
		string(record.Tier),
		string(record.Status),
		record.CurrentPeriodStart,
		record.CurrentPeriodEnd,
		record.CancelAt,
		record.CanceledAt,
		payload,
		record.SyncedAt,
		record.UpdatedAt,
	))
}

func (s *PostgresBillingSubscriptionStore) FindBillingSubscriptionByCustomerID(
	ctx context.Context,
	customerID string,
) (billing.StripeSubscriptionRecord, error) {
	if s == nil || s.Querier == nil {
		return billing.StripeSubscriptionRecord{}, ErrBillingAccountNotFound
	}
	record, err := scanBillingSubscriptionRow(s.Querier.QueryRow(ctx, findBillingSubscriptionByCustomerIDSQL, strings.TrimSpace(customerID)))
	if err != nil {
		if err == pgx.ErrNoRows {
			return billing.StripeSubscriptionRecord{}, ErrBillingAccountNotFound
		}
		return billing.StripeSubscriptionRecord{}, err
	}
	return record, nil
}

func (s *PostgresBillingSubscriptionStore) FindBillingSubscriptionBySubscriptionID(
	ctx context.Context,
	subscriptionID string,
) (billing.StripeSubscriptionRecord, error) {
	if s == nil || s.Querier == nil {
		return billing.StripeSubscriptionRecord{}, ErrBillingAccountNotFound
	}
	record, err := scanBillingSubscriptionRow(s.Querier.QueryRow(ctx, findBillingSubscriptionBySubscriptionIDSQL, strings.TrimSpace(subscriptionID)))
	if err != nil {
		if err == pgx.ErrNoRows {
			return billing.StripeSubscriptionRecord{}, ErrBillingAccountNotFound
		}
		return billing.StripeSubscriptionRecord{}, err
	}
	return record, nil
}

func (s *PostgresBillingSubscriptionReconciliationStore) RecordBillingSubscriptionReconciliation(
	ctx context.Context,
	record billing.StripeSubscriptionReconciliationRecord,
) error {
	if s == nil || s.Execer == nil {
		return ErrBillingAccountNotFound
	}
	payload, err := marshalStringMap(record.Metadata)
	if err != nil {
		return err
	}
	_, err = s.Execer.Exec(ctx, insertBillingSubscriptionReconciliationSQL,
		record.EventID,
		record.Provider,
		record.CustomerID,
		record.SubscriptionID,
		string(record.PreviousTier),
		string(record.CurrentTier),
		record.StripePriceID,
		record.Status,
		record.ObservedAt,
		record.ReconciledAt,
		record.Notes,
		payload,
	)
	return err
}

func (s *PostgresBillingSubscriptionReconciliationStore) ListBillingSubscriptionReconciliations(
	context.Context,
	string,
) ([]billing.StripeSubscriptionReconciliationRecord, error) {
	return nil, nil
}

const listBillingAccountsForSubscriptionSyncSQL = `
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
where active_subscription_id <> ''
order by updated_at asc
limit $1
`

func (s *PostgresBillingStore) ListBillingAccountsForSubscriptionSync(
	ctx context.Context,
	limit int,
) ([]BillingAccountRecord, error) {
	if s == nil || s.Querier == nil {
		return nil, ErrBillingAccountNotFound
	}
	if limit <= 0 {
		limit = 25
	}

	querier, ok := s.Querier.(postgresBillingStripeQuerier)
	if !ok {
		return nil, ErrBillingAccountNotFound
	}
	rows, err := querier.Query(ctx, listBillingAccountsForSubscriptionSyncSQL, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]BillingAccountRecord, 0)
	for rows.Next() {
		record, err := scanBillingAccountRow(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func scanBillingCheckoutSessionRow(row pgx.Row) (billing.StripeCheckoutSessionRecord, error) {
	var record billing.StripeCheckoutSessionRecord
	var tier string
	var status string
	var payload []byte
	if err := row.Scan(
		&record.SessionID,
		&record.CustomerID,
		&record.CustomerEmail,
		&record.SubscriptionID,
		&tier,
		&record.StripePriceID,
		&status,
		&record.SuccessURL,
		&record.CancelURL,
		&payload,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.CompletedAt,
	); err != nil {
		return billing.StripeCheckoutSessionRecord{}, err
	}
	record.Tier = domain.PlanTier(tier)
	record.Status = billing.StripeSessionStatus(status)
	record.Metadata = unmarshalStringMap(payload)
	return record, nil
}

func scanBillingSubscriptionRow(row pgx.Row) (billing.StripeSubscriptionRecord, error) {
	var record billing.StripeSubscriptionRecord
	var tier string
	var status string
	var payload []byte
	if err := row.Scan(
		&record.SubscriptionID,
		&record.CustomerID,
		&record.CustomerEmail,
		&record.StripePriceID,
		&tier,
		&status,
		&record.CurrentPeriodStart,
		&record.CurrentPeriodEnd,
		&record.CancelAt,
		&record.CanceledAt,
		&payload,
		&record.SyncedAt,
		&record.UpdatedAt,
	); err != nil {
		return billing.StripeSubscriptionRecord{}, err
	}
	record.Tier = domain.PlanTier(tier)
	record.Status = billing.StripeSubscriptionStatus(status)
	record.Metadata = unmarshalStringMap(payload)
	return record, nil
}

func marshalStringMap(payload map[string]string) ([]byte, error) {
	if len(payload) == 0 {
		return []byte(`{}`), nil
	}
	return json.Marshal(payload)
}

func unmarshalStringMap(payload []byte) map[string]string {
	if len(payload) == 0 {
		return map[string]string{}
	}
	decoded := make(map[string]string)
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return map[string]string{}
	}
	return decoded
}
