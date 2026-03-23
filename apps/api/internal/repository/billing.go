package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
)

var ErrBillingAccountNotFound = errors.New("billing account not found")

type BillingAccount struct {
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

type BillingWebhookEvent struct {
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

type BillingRepository interface {
	FindBillingAccount(context.Context, string) (BillingAccount, error)
	UpsertBillingAccount(context.Context, BillingAccount) (BillingAccount, error)
	RecordWebhookEvent(context.Context, BillingWebhookEvent) (BillingWebhookEvent, error)
}

type InMemoryBillingRepository struct {
	mu       sync.RWMutex
	accounts map[string]BillingAccount
	events   map[string]BillingWebhookEvent
}

func NewInMemoryBillingRepository() *InMemoryBillingRepository {
	return &InMemoryBillingRepository{
		accounts: make(map[string]BillingAccount),
		events:   make(map[string]BillingWebhookEvent),
	}
}

func (r *InMemoryBillingRepository) FindBillingAccount(_ context.Context, ownerUserID string) (BillingAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	account, ok := r.accounts[strings.TrimSpace(ownerUserID)]
	if !ok {
		return BillingAccount{}, ErrBillingAccountNotFound
	}
	return copyBillingAccount(account), nil
}

func (r *InMemoryBillingRepository) UpsertBillingAccount(_ context.Context, account BillingAccount) (BillingAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerUserID := strings.TrimSpace(account.OwnerUserID)
	if ownerUserID == "" {
		return BillingAccount{}, fmt.Errorf("owner user id is required")
	}

	stored := copyBillingAccount(account)
	if existing, ok := r.accounts[ownerUserID]; ok {
		stored.CreatedAt = existing.CreatedAt
	}
	r.accounts[ownerUserID] = stored
	return copyBillingAccount(stored), nil
}

func (r *InMemoryBillingRepository) RecordWebhookEvent(_ context.Context, event BillingWebhookEvent) (BillingWebhookEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	providerEventID := strings.TrimSpace(event.ProviderEventID)
	if providerEventID == "" {
		return BillingWebhookEvent{}, fmt.Errorf("provider event id is required")
	}

	stored := copyBillingWebhookEvent(event)
	r.events[providerEventID] = stored
	return copyBillingWebhookEvent(stored), nil
}

type PostgresBillingRepository struct {
	store db.BillingAccountStore
}

func NewPostgresBillingRepository(store db.BillingAccountStore) *PostgresBillingRepository {
	return &PostgresBillingRepository{store: store}
}

func (r *PostgresBillingRepository) FindBillingAccount(ctx context.Context, ownerUserID string) (BillingAccount, error) {
	if r == nil || r.store == nil {
		return BillingAccount{}, ErrBillingAccountNotFound
	}

	record, err := r.store.FindBillingAccount(ctx, ownerUserID)
	if err != nil {
		return BillingAccount{}, translateBillingError(err)
	}
	return fromDBBillingAccount(record), nil
}

func (r *PostgresBillingRepository) UpsertBillingAccount(ctx context.Context, account BillingAccount) (BillingAccount, error) {
	if r == nil || r.store == nil {
		return BillingAccount{}, ErrBillingAccountNotFound
	}

	record, err := r.store.UpsertBillingAccount(ctx, toDBBillingAccount(account))
	if err != nil {
		return BillingAccount{}, translateBillingError(err)
	}
	return fromDBBillingAccount(record), nil
}

func (r *PostgresBillingRepository) RecordWebhookEvent(ctx context.Context, event BillingWebhookEvent) (BillingWebhookEvent, error) {
	if r == nil || r.store == nil {
		return BillingWebhookEvent{}, ErrBillingAccountNotFound
	}

	record, err := r.store.RecordWebhookEvent(ctx, toDBBillingWebhookEvent(event))
	if err != nil {
		return BillingWebhookEvent{}, translateBillingError(err)
	}
	return fromDBBillingWebhookEvent(record), nil
}

func translateBillingError(err error) error {
	if errors.Is(err, db.ErrBillingAccountNotFound) {
		return ErrBillingAccountNotFound
	}
	return err
}

func fromDBBillingAccount(record db.BillingAccountRecord) BillingAccount {
	return BillingAccount{
		OwnerUserID:          record.OwnerUserID,
		Email:                record.Email,
		CurrentTier:          record.CurrentTier,
		StripeCustomerID:     record.StripeCustomerID,
		ActiveSubscriptionID: record.ActiveSubscriptionID,
		CurrentPriceID:       record.CurrentPriceID,
		Status:               record.Status,
		CurrentPeriodEnd:     cloneTimePointer(record.CurrentPeriodEnd),
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
	}
}

func toDBBillingAccount(account BillingAccount) db.BillingAccountRecord {
	return db.BillingAccountRecord{
		OwnerUserID:          strings.TrimSpace(account.OwnerUserID),
		Email:                strings.TrimSpace(account.Email),
		CurrentTier:          account.CurrentTier,
		StripeCustomerID:     strings.TrimSpace(account.StripeCustomerID),
		ActiveSubscriptionID: strings.TrimSpace(account.ActiveSubscriptionID),
		CurrentPriceID:       strings.TrimSpace(account.CurrentPriceID),
		Status:               strings.TrimSpace(account.Status),
		CurrentPeriodEnd:     cloneTimePointer(account.CurrentPeriodEnd),
		CreatedAt:            account.CreatedAt,
		UpdatedAt:            account.UpdatedAt,
	}
}

func fromDBBillingWebhookEvent(record db.BillingWebhookEventRecord) BillingWebhookEvent {
	return BillingWebhookEvent{
		ProviderEventID: record.ProviderEventID,
		EventType:       record.EventType,
		OwnerUserID:     record.OwnerUserID,
		CustomerID:      record.CustomerID,
		SubscriptionID:  record.SubscriptionID,
		PlanTier:        record.PlanTier,
		Status:          record.Status,
		Payload:         copyMap(record.Payload),
		ReceivedAt:      record.ReceivedAt,
		ProcessedAt:     cloneTimePointer(record.ProcessedAt),
	}
}

func toDBBillingWebhookEvent(event BillingWebhookEvent) db.BillingWebhookEventRecord {
	return db.BillingWebhookEventRecord{
		ProviderEventID: strings.TrimSpace(event.ProviderEventID),
		EventType:       strings.TrimSpace(event.EventType),
		OwnerUserID:     strings.TrimSpace(event.OwnerUserID),
		CustomerID:      strings.TrimSpace(event.CustomerID),
		SubscriptionID:  strings.TrimSpace(event.SubscriptionID),
		PlanTier:        event.PlanTier,
		Status:          strings.TrimSpace(event.Status),
		Payload:         copyMap(event.Payload),
		ReceivedAt:      event.ReceivedAt,
		ProcessedAt:     cloneTimePointer(event.ProcessedAt),
	}
}

func copyBillingAccount(account BillingAccount) BillingAccount {
	return BillingAccount{
		OwnerUserID:          account.OwnerUserID,
		Email:                account.Email,
		CurrentTier:          account.CurrentTier,
		StripeCustomerID:     account.StripeCustomerID,
		ActiveSubscriptionID: account.ActiveSubscriptionID,
		CurrentPriceID:       account.CurrentPriceID,
		Status:               account.Status,
		CurrentPeriodEnd:     cloneTimePointer(account.CurrentPeriodEnd),
		CreatedAt:            account.CreatedAt,
		UpdatedAt:            account.UpdatedAt,
	}
}

func copyBillingWebhookEvent(event BillingWebhookEvent) BillingWebhookEvent {
	return BillingWebhookEvent{
		ProviderEventID: event.ProviderEventID,
		EventType:       event.EventType,
		OwnerUserID:     event.OwnerUserID,
		CustomerID:      event.CustomerID,
		SubscriptionID:  event.SubscriptionID,
		PlanTier:        event.PlanTier,
		Status:          event.Status,
		Payload:         copyMap(event.Payload),
		ReceivedAt:      event.ReceivedAt,
		ProcessedAt:     cloneTimePointer(event.ProcessedAt),
	}
}

func copyMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}
