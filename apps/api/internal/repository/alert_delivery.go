package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

var (
	ErrAlertDeliveryChannelNotFound      = errors.New("alert delivery channel not found")
	ErrAlertDeliveryChannelAlreadyExists = errors.New("alert delivery channel already exists")
)

type AlertInboxQuery struct {
	Limit      int
	Severity   string
	SignalType string
	Cursor     string
	UnreadOnly bool
}

type AlertInboxPage struct {
	Items       []domain.AlertEvent
	NextCursor  *string
	HasMore     bool
	UnreadCount int
}

type AlertDeliveryRepository interface {
	ListAlertInboxEvents(context.Context, string, AlertInboxQuery) (AlertInboxPage, error)
	MarkAlertInboxEventRead(context.Context, string, string, bool) (domain.AlertEvent, error)
	ListAlertDeliveryChannels(context.Context, string) ([]domain.AlertDeliveryChannel, error)
	CreateAlertDeliveryChannel(context.Context, domain.AlertDeliveryChannel) (domain.AlertDeliveryChannel, error)
	FindAlertDeliveryChannel(context.Context, string, string) (domain.AlertDeliveryChannel, error)
	UpdateAlertDeliveryChannel(context.Context, domain.AlertDeliveryChannel) (domain.AlertDeliveryChannel, error)
	DeleteAlertDeliveryChannel(context.Context, string, string) error
}

type InMemoryAlertDeliveryRepository struct {
	mu       sync.RWMutex
	channels map[string]map[string]domain.AlertDeliveryChannel
	events   map[string][]domain.AlertEvent
}

func NewInMemoryAlertDeliveryRepository() *InMemoryAlertDeliveryRepository {
	return &InMemoryAlertDeliveryRepository{
		channels: make(map[string]map[string]domain.AlertDeliveryChannel),
		events:   make(map[string][]domain.AlertEvent),
	}
}

func (r *InMemoryAlertDeliveryRepository) SeedAlertEvent(event domain.AlertEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerUserID := strings.TrimSpace(event.OwnerUserID)
	if ownerUserID == "" {
		return
	}
	cloned := domain.CopyAlertEvent(event)
	r.events[ownerUserID] = append([]domain.AlertEvent{cloned}, r.events[ownerUserID]...)
}

func (r *InMemoryAlertDeliveryRepository) ListAlertInboxEvents(_ context.Context, ownerUserID string, query AlertInboxQuery) (AlertInboxPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ownerUserID = strings.TrimSpace(ownerUserID)
	items := r.events[ownerUserID]
	if len(items) == 0 {
		return AlertInboxPage{Items: []domain.AlertEvent{}}, nil
	}

	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	severity := strings.TrimSpace(strings.ToLower(query.Severity))
	signalType := strings.TrimSpace(strings.ToLower(query.SignalType))
	cursorID := strings.TrimSpace(query.Cursor)

	filtered := make([]domain.AlertEvent, 0, len(items))
	started := cursorID == ""
	unreadCount := 0
	for _, item := range items {
		if item.ReadAt == nil {
			unreadCount++
		}
		if severity != "" && string(item.Severity) != severity {
			continue
		}
		if signalType != "" && item.SignalType != signalType {
			continue
		}
		if query.UnreadOnly && item.ReadAt != nil {
			continue
		}
		if !started {
			if item.ID == cursorID {
				started = true
			}
			continue
		}
		filtered = append(filtered, domain.CopyAlertEvent(item))
		if len(filtered) > limit {
			break
		}
	}
	hasMore := len(filtered) > limit
	if hasMore {
		filtered = filtered[:limit]
	}
	var nextCursor *string
	if hasMore && len(filtered) > 0 {
		cursor := filtered[len(filtered)-1].ID
		nextCursor = &cursor
	}
	return AlertInboxPage{
		Items:       filtered,
		NextCursor:  nextCursor,
		HasMore:     hasMore,
		UnreadCount: unreadCount,
	}, nil
}

func (r *InMemoryAlertDeliveryRepository) MarkAlertInboxEventRead(_ context.Context, ownerUserID string, eventID string, isRead bool) (domain.AlertEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := r.events[strings.TrimSpace(ownerUserID)]
	for index, item := range items {
		if item.ID != strings.TrimSpace(eventID) {
			continue
		}
		updated := domain.CopyAlertEvent(item)
		if isRead {
			now := time.Now().UTC()
			updated.ReadAt = &now
		} else {
			updated.ReadAt = nil
		}
		r.events[strings.TrimSpace(ownerUserID)][index] = updated
		return domain.CopyAlertEvent(updated), nil
	}
	return domain.AlertEvent{}, db.ErrAlertEventNotFound
}

func (r *InMemoryAlertDeliveryRepository) ListAlertDeliveryChannels(_ context.Context, ownerUserID string) ([]domain.AlertDeliveryChannel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	owned := r.channels[strings.TrimSpace(ownerUserID)]
	if len(owned) == 0 {
		return []domain.AlertDeliveryChannel{}, nil
	}

	items := make([]domain.AlertDeliveryChannel, 0, len(owned))
	for _, item := range owned {
		items = append(items, domain.CopyAlertDeliveryChannel(item))
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func (r *InMemoryAlertDeliveryRepository) CreateAlertDeliveryChannel(_ context.Context, channel domain.AlertDeliveryChannel) (domain.AlertDeliveryChannel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerUserID := strings.TrimSpace(channel.OwnerUserID)
	channelID := strings.TrimSpace(channel.ID)
	if ownerUserID == "" || channelID == "" {
		return domain.AlertDeliveryChannel{}, fmt.Errorf("owner user id and channel id are required")
	}
	if _, ok := r.channels[ownerUserID]; !ok {
		r.channels[ownerUserID] = make(map[string]domain.AlertDeliveryChannel)
	}
	for _, existing := range r.channels[ownerUserID] {
		if existing.ChannelType == channel.ChannelType && existing.Target == channel.Target {
			return domain.AlertDeliveryChannel{}, ErrAlertDeliveryChannelAlreadyExists
		}
	}
	stored := domain.CopyAlertDeliveryChannel(channel)
	r.channels[ownerUserID][channelID] = stored
	return domain.CopyAlertDeliveryChannel(stored), nil
}

func (r *InMemoryAlertDeliveryRepository) FindAlertDeliveryChannel(_ context.Context, ownerUserID string, channelID string) (domain.AlertDeliveryChannel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	owned := r.channels[strings.TrimSpace(ownerUserID)]
	channel, ok := owned[strings.TrimSpace(channelID)]
	if !ok {
		return domain.AlertDeliveryChannel{}, ErrAlertDeliveryChannelNotFound
	}
	return domain.CopyAlertDeliveryChannel(channel), nil
}

func (r *InMemoryAlertDeliveryRepository) UpdateAlertDeliveryChannel(_ context.Context, channel domain.AlertDeliveryChannel) (domain.AlertDeliveryChannel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerUserID := strings.TrimSpace(channel.OwnerUserID)
	channelID := strings.TrimSpace(channel.ID)
	owned := r.channels[ownerUserID]
	if len(owned) == 0 {
		return domain.AlertDeliveryChannel{}, ErrAlertDeliveryChannelNotFound
	}
	if _, ok := owned[channelID]; !ok {
		return domain.AlertDeliveryChannel{}, ErrAlertDeliveryChannelNotFound
	}
	stored := domain.CopyAlertDeliveryChannel(channel)
	owned[channelID] = stored
	return domain.CopyAlertDeliveryChannel(stored), nil
}

func (r *InMemoryAlertDeliveryRepository) DeleteAlertDeliveryChannel(_ context.Context, ownerUserID string, channelID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	owned := r.channels[strings.TrimSpace(ownerUserID)]
	channelID = strings.TrimSpace(channelID)
	if _, ok := owned[channelID]; !ok {
		return ErrAlertDeliveryChannelNotFound
	}
	delete(owned, channelID)
	if len(owned) == 0 {
		delete(r.channels, strings.TrimSpace(ownerUserID))
	}
	return nil
}

type PostgresAlertDeliveryRepository struct {
	store *db.PostgresAlertDeliveryStore
}

func NewPostgresAlertDeliveryRepository(store *db.PostgresAlertDeliveryStore) *PostgresAlertDeliveryRepository {
	return &PostgresAlertDeliveryRepository{store: store}
}

func (r *PostgresAlertDeliveryRepository) ListAlertInboxEvents(ctx context.Context, ownerUserID string, query AlertInboxQuery) (AlertInboxPage, error) {
	if r == nil || r.store == nil {
		return AlertInboxPage{Items: []domain.AlertEvent{}}, nil
	}
	page, err := r.store.ListAlertInboxEvents(ctx, ownerUserID, db.AlertInboxQuery{
		Limit:      query.Limit,
		Severity:   query.Severity,
		SignalType: query.SignalType,
		Cursor:     query.Cursor,
		UnreadOnly: query.UnreadOnly,
	})
	if err != nil {
		return AlertInboxPage{}, translateAlertDeliveryError(err)
	}
	return AlertInboxPage{
		Items:       page.Items,
		NextCursor:  page.NextCursor,
		HasMore:     page.HasMore,
		UnreadCount: page.UnreadCount,
	}, nil
}

func (r *PostgresAlertDeliveryRepository) MarkAlertInboxEventRead(ctx context.Context, ownerUserID string, eventID string, isRead bool) (domain.AlertEvent, error) {
	if r == nil || r.store == nil {
		return domain.AlertEvent{}, db.ErrAlertEventNotFound
	}
	var readAt *time.Time
	if isRead {
		now := time.Now().UTC()
		readAt = &now
	}
	item, err := r.store.MarkAlertInboxEventRead(ctx, ownerUserID, eventID, readAt)
	if err != nil {
		return domain.AlertEvent{}, translateAlertDeliveryError(err)
	}
	return item, nil
}

func (r *PostgresAlertDeliveryRepository) ListAlertDeliveryChannels(ctx context.Context, ownerUserID string) ([]domain.AlertDeliveryChannel, error) {
	if r == nil || r.store == nil {
		return []domain.AlertDeliveryChannel{}, nil
	}
	items, err := r.store.ListAlertDeliveryChannels(ctx, ownerUserID)
	if err != nil {
		return nil, translateAlertDeliveryError(err)
	}
	return items, nil
}

func (r *PostgresAlertDeliveryRepository) CreateAlertDeliveryChannel(ctx context.Context, channel domain.AlertDeliveryChannel) (domain.AlertDeliveryChannel, error) {
	if r == nil || r.store == nil {
		return domain.AlertDeliveryChannel{}, nil
	}
	item, err := r.store.CreateAlertDeliveryChannel(ctx, db.AlertDeliveryChannelCreate{
		OwnerUserID: channel.OwnerUserID,
		Label:       channel.Label,
		ChannelType: string(channel.ChannelType),
		Target:      channel.Target,
		Metadata:    channel.Metadata,
		IsEnabled:   channel.IsEnabled,
	})
	if err != nil {
		return domain.AlertDeliveryChannel{}, translateAlertDeliveryError(err)
	}
	return item, nil
}

func (r *PostgresAlertDeliveryRepository) FindAlertDeliveryChannel(ctx context.Context, ownerUserID string, channelID string) (domain.AlertDeliveryChannel, error) {
	if r == nil || r.store == nil {
		return domain.AlertDeliveryChannel{}, ErrAlertDeliveryChannelNotFound
	}
	items, err := r.store.ListAlertDeliveryChannels(ctx, ownerUserID)
	if err != nil {
		return domain.AlertDeliveryChannel{}, translateAlertDeliveryError(err)
	}
	for _, item := range items {
		if item.ID == strings.TrimSpace(channelID) {
			return item, nil
		}
	}
	return domain.AlertDeliveryChannel{}, ErrAlertDeliveryChannelNotFound
}

func (r *PostgresAlertDeliveryRepository) UpdateAlertDeliveryChannel(ctx context.Context, channel domain.AlertDeliveryChannel) (domain.AlertDeliveryChannel, error) {
	if r == nil || r.store == nil {
		return domain.AlertDeliveryChannel{}, nil
	}
	item, err := r.store.UpdateAlertDeliveryChannel(ctx, db.AlertDeliveryChannelUpdate{
		OwnerUserID: channel.OwnerUserID,
		ChannelID:   channel.ID,
		Label:       channel.Label,
		Target:      channel.Target,
		Metadata:    channel.Metadata,
		IsEnabled:   channel.IsEnabled,
	})
	if err != nil {
		return domain.AlertDeliveryChannel{}, translateAlertDeliveryError(err)
	}
	return item, nil
}

func (r *PostgresAlertDeliveryRepository) DeleteAlertDeliveryChannel(ctx context.Context, ownerUserID string, channelID string) error {
	if r == nil || r.store == nil {
		return nil
	}
	return translateAlertDeliveryError(r.store.DeleteAlertDeliveryChannel(ctx, ownerUserID, channelID))
}

func translateAlertDeliveryError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, db.ErrAlertDeliveryChannelNotFound):
		return ErrAlertDeliveryChannelNotFound
	case errors.Is(err, db.ErrAlertEventNotFound):
		return ErrAlertDeliveryChannelNotFound
	case errors.Is(err, db.ErrAlertDeliveryAttemptDeduped):
		return db.ErrAlertDeliveryAttemptDeduped
	default:
		return err
	}
}
