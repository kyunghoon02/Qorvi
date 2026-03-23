package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/repository"
	"github.com/whalegraph/whalegraph/packages/domain"
)

var (
	ErrAlertDeliveryForbidden      = errors.New("alert delivery feature is not available")
	ErrAlertDeliveryNotFound       = errors.New("alert delivery channel not found")
	ErrAlertDeliveryConflict       = errors.New("alert delivery channel already exists")
	ErrAlertDeliveryInvalidRequest = errors.New("invalid alert delivery request")
	ErrAlertDeliveryLimitExceeded  = errors.New("alert delivery channel limit exceeded")
)

type AlertInboxItem struct {
	ID          string         `json:"id"`
	AlertRuleID string         `json:"alertRuleId"`
	SignalType  string         `json:"signalType"`
	Severity    string         `json:"severity"`
	Payload     map[string]any `json:"payload"`
	ObservedAt  string         `json:"observedAt"`
	IsRead      bool           `json:"isRead"`
	ReadAt      string         `json:"readAt,omitempty"`
	CreatedAt   string         `json:"createdAt"`
}

type AlertInboxCollection struct {
	Items       []AlertInboxItem `json:"items"`
	NextCursor  *string          `json:"nextCursor,omitempty"`
	HasMore     bool             `json:"hasMore"`
	UnreadCount int              `json:"unreadCount"`
}

type AlertInboxQuery struct {
	Limit      int
	Severity   string
	SignalType string
	Cursor     string
	Status     string
}

type UpdateAlertInboxEventRequest struct {
	IsRead bool `json:"isRead"`
}

type AlertDeliveryChannelSummary struct {
	ID          string         `json:"id"`
	Label       string         `json:"label"`
	ChannelType string         `json:"channelType"`
	Target      string         `json:"target"`
	Metadata    map[string]any `json:"metadata"`
	IsEnabled   bool           `json:"isEnabled"`
	CreatedAt   string         `json:"createdAt"`
	UpdatedAt   string         `json:"updatedAt"`
}

type AlertDeliveryChannelCollection struct {
	Items []AlertDeliveryChannelSummary `json:"items"`
}

type CreateAlertDeliveryChannelRequest struct {
	Label       string         `json:"label"`
	ChannelType string         `json:"channelType"`
	Target      string         `json:"target"`
	Metadata    map[string]any `json:"metadata"`
	IsEnabled   *bool          `json:"isEnabled"`
}

type UpdateAlertDeliveryChannelRequest struct {
	Label     string         `json:"label"`
	Target    string         `json:"target"`
	Metadata  map[string]any `json:"metadata"`
	IsEnabled *bool          `json:"isEnabled"`
}

type AlertDeliveryMutationResult struct {
	Deleted bool `json:"deleted"`
}

type AlertInboxMutationResult struct {
	Event AlertInboxItem `json:"event"`
}

type AlertDeliveryService struct {
	repo repository.AlertDeliveryRepository
	Now  func() time.Time
}

func NewAlertDeliveryService(repo repository.AlertDeliveryRepository) *AlertDeliveryService {
	return &AlertDeliveryService{repo: repo, Now: time.Now}
}

func (s *AlertDeliveryService) ListInboxEvents(ctx context.Context, ownerUserID string, tier domain.PlanTier, query AlertInboxQuery) (AlertInboxCollection, error) {
	if err := ensureAlertDeliveryEnabled(tier); err != nil {
		return AlertInboxCollection{}, err
	}

	items, err := s.repo.ListAlertInboxEvents(ctx, ownerUserID, repository.AlertInboxQuery{
		Limit:      normalizeAlertInboxLimit(query.Limit),
		Severity:   strings.TrimSpace(strings.ToLower(query.Severity)),
		SignalType: strings.TrimSpace(strings.ToLower(query.SignalType)),
		Cursor:     strings.TrimSpace(query.Cursor),
		UnreadOnly: strings.EqualFold(strings.TrimSpace(query.Status), "unread"),
	})
	if err != nil {
		return AlertInboxCollection{}, err
	}

	response := AlertInboxCollection{
		Items:       make([]AlertInboxItem, 0, len(items.Items)),
		NextCursor:  items.NextCursor,
		HasMore:     items.HasMore,
		UnreadCount: items.UnreadCount,
	}
	for _, item := range items.Items {
		response.Items = append(response.Items, toAlertInboxItem(item))
	}
	return response, nil
}

func (s *AlertDeliveryService) UpdateInboxEvent(
	ctx context.Context,
	ownerUserID string,
	tier domain.PlanTier,
	eventID string,
	req UpdateAlertInboxEventRequest,
) (AlertInboxMutationResult, error) {
	if err := ensureAlertDeliveryEnabled(tier); err != nil {
		return AlertInboxMutationResult{}, err
	}

	event, err := s.repo.MarkAlertInboxEventRead(ctx, ownerUserID, eventID, req.IsRead)
	if err != nil {
		if errors.Is(err, repository.ErrAlertDeliveryChannelNotFound) || errors.Is(err, repository.ErrAlertRuleNotFound) {
			return AlertInboxMutationResult{}, ErrAlertDeliveryNotFound
		}
		return AlertInboxMutationResult{}, err
	}
	return AlertInboxMutationResult{Event: toAlertInboxItem(event)}, nil
}

func (s *AlertDeliveryService) ListAlertDeliveryChannels(ctx context.Context, ownerUserID string, tier domain.PlanTier) (AlertDeliveryChannelCollection, error) {
	if err := ensureAlertDeliveryEnabled(tier); err != nil {
		return AlertDeliveryChannelCollection{}, err
	}

	items, err := s.repo.ListAlertDeliveryChannels(ctx, ownerUserID)
	if err != nil {
		return AlertDeliveryChannelCollection{}, err
	}

	response := AlertDeliveryChannelCollection{Items: make([]AlertDeliveryChannelSummary, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, toAlertDeliveryChannelSummary(item))
	}
	return response, nil
}

func (s *AlertDeliveryService) CreateAlertDeliveryChannel(ctx context.Context, ownerUserID string, tier domain.PlanTier, req CreateAlertDeliveryChannelRequest) (AlertDeliveryChannelSummary, error) {
	if err := ensureAlertDeliveryEnabled(tier); err != nil {
		return AlertDeliveryChannelSummary{}, err
	}

	existing, err := s.repo.ListAlertDeliveryChannels(ctx, ownerUserID)
	if err != nil {
		return AlertDeliveryChannelSummary{}, err
	}
	limit, err := alertDeliveryChannelLimitForTier(tier)
	if err != nil {
		return AlertDeliveryChannelSummary{}, err
	}
	if len(existing) >= limit {
		return AlertDeliveryChannelSummary{}, ErrAlertDeliveryLimitExceeded
	}

	channel, err := s.buildAlertDeliveryChannel(ownerUserID, "", req.Label, req.ChannelType, req.Target, req.Metadata, req.IsEnabled, time.Time{})
	if err != nil {
		return AlertDeliveryChannelSummary{}, err
	}
	created, err := s.repo.CreateAlertDeliveryChannel(ctx, channel)
	if err != nil {
		if errors.Is(err, repository.ErrAlertDeliveryChannelAlreadyExists) {
			return AlertDeliveryChannelSummary{}, ErrAlertDeliveryConflict
		}
		return AlertDeliveryChannelSummary{}, err
	}
	return toAlertDeliveryChannelSummary(created), nil
}

func (s *AlertDeliveryService) GetAlertDeliveryChannel(ctx context.Context, ownerUserID string, tier domain.PlanTier, channelID string) (AlertDeliveryChannelSummary, error) {
	if err := ensureAlertDeliveryEnabled(tier); err != nil {
		return AlertDeliveryChannelSummary{}, err
	}
	channel, err := s.repo.FindAlertDeliveryChannel(ctx, ownerUserID, channelID)
	if err != nil {
		if errors.Is(err, repository.ErrAlertDeliveryChannelNotFound) {
			return AlertDeliveryChannelSummary{}, ErrAlertDeliveryNotFound
		}
		return AlertDeliveryChannelSummary{}, err
	}
	return toAlertDeliveryChannelSummary(channel), nil
}

func (s *AlertDeliveryService) UpdateAlertDeliveryChannel(ctx context.Context, ownerUserID string, tier domain.PlanTier, channelID string, req UpdateAlertDeliveryChannelRequest) (AlertDeliveryChannelSummary, error) {
	if err := ensureAlertDeliveryEnabled(tier); err != nil {
		return AlertDeliveryChannelSummary{}, err
	}

	current, err := s.repo.FindAlertDeliveryChannel(ctx, ownerUserID, channelID)
	if err != nil {
		if errors.Is(err, repository.ErrAlertDeliveryChannelNotFound) {
			return AlertDeliveryChannelSummary{}, ErrAlertDeliveryNotFound
		}
		return AlertDeliveryChannelSummary{}, err
	}

	next := domain.CopyAlertDeliveryChannel(current)
	label, err := domain.NormalizeAlertChannelLabel(req.Label)
	if err != nil {
		return AlertDeliveryChannelSummary{}, ErrAlertDeliveryInvalidRequest
	}
	target, err := domain.NormalizeAlertChannelTarget(current.ChannelType, req.Target)
	if err != nil {
		return AlertDeliveryChannelSummary{}, ErrAlertDeliveryInvalidRequest
	}
	next.Label = label
	next.Target = target
	next.Metadata = domain.NormalizeAlertDefinition(req.Metadata)
	next.IsEnabled = current.IsEnabled
	if req.IsEnabled != nil {
		next.IsEnabled = *req.IsEnabled
	}
	next.UpdatedAt = s.now()

	updated, err := s.repo.UpdateAlertDeliveryChannel(ctx, next)
	if err != nil {
		if errors.Is(err, repository.ErrAlertDeliveryChannelNotFound) {
			return AlertDeliveryChannelSummary{}, ErrAlertDeliveryNotFound
		}
		return AlertDeliveryChannelSummary{}, err
	}
	return toAlertDeliveryChannelSummary(updated), nil
}

func (s *AlertDeliveryService) DeleteAlertDeliveryChannel(ctx context.Context, ownerUserID string, tier domain.PlanTier, channelID string) error {
	if err := ensureAlertDeliveryEnabled(tier); err != nil {
		return err
	}
	if err := s.repo.DeleteAlertDeliveryChannel(ctx, ownerUserID, channelID); err != nil {
		if errors.Is(err, repository.ErrAlertDeliveryChannelNotFound) {
			return ErrAlertDeliveryNotFound
		}
		return err
	}
	return nil
}

func ensureAlertDeliveryEnabled(tier domain.PlanTier) error {
	if err := ensureAlertsEnabled(tier); err != nil {
		return ErrAlertDeliveryForbidden
	}
	return nil
}

func alertDeliveryChannelLimitForTier(tier domain.PlanTier) (int, error) {
	switch tier {
	case domain.PlanPro:
		return 3, nil
	case domain.PlanTeam:
		return 10, nil
	default:
		return 0, ErrAlertDeliveryForbidden
	}
}

func normalizeAlertInboxLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 50
	}
	return limit
}

func normalizeOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (s *AlertDeliveryService) buildAlertDeliveryChannel(
	ownerUserID string,
	channelID string,
	label string,
	channelType string,
	target string,
	metadata map[string]any,
	isEnabled *bool,
	createdAt time.Time,
) (domain.AlertDeliveryChannel, error) {
	normalizedLabel, err := domain.NormalizeAlertChannelLabel(label)
	if err != nil {
		return domain.AlertDeliveryChannel{}, ErrAlertDeliveryInvalidRequest
	}
	normalizedChannelType, err := domain.NormalizeAlertChannelType(channelType)
	if err != nil {
		return domain.AlertDeliveryChannel{}, ErrAlertDeliveryInvalidRequest
	}
	normalizedTarget, err := domain.NormalizeAlertChannelTarget(normalizedChannelType, target)
	if err != nil {
		return domain.AlertDeliveryChannel{}, ErrAlertDeliveryInvalidRequest
	}
	now := s.now()
	if createdAt.IsZero() {
		createdAt = now
	}
	channel := domain.AlertDeliveryChannel{
		ID:          channelID,
		OwnerUserID: strings.TrimSpace(ownerUserID),
		Label:       normalizedLabel,
		ChannelType: normalizedChannelType,
		Target:      normalizedTarget,
		Metadata:    domain.NormalizeAlertDefinition(metadata),
		IsEnabled:   true,
		CreatedAt:   createdAt.UTC(),
		UpdatedAt:   now,
	}
	if channel.ID == "" {
		channel.ID = newAlertDeliveryChannelID()
	}
	if isEnabled != nil {
		channel.IsEnabled = *isEnabled
	}
	if err := domain.ValidateAlertDeliveryChannel(channel); err != nil {
		return domain.AlertDeliveryChannel{}, ErrAlertDeliveryInvalidRequest
	}
	return channel, nil
}

func toAlertInboxItem(event domain.AlertEvent) AlertInboxItem {
	return AlertInboxItem{
		ID:          event.ID,
		AlertRuleID: event.AlertRuleID,
		SignalType:  event.SignalType,
		Severity:    string(event.Severity),
		Payload:     domain.NormalizeAlertDefinition(event.Payload),
		ObservedAt:  event.ObservedAt.UTC().Format(time.RFC3339),
		IsRead:      event.ReadAt != nil,
		ReadAt:      normalizeOptionalTime(event.ReadAt),
		CreatedAt:   event.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toAlertDeliveryChannelSummary(channel domain.AlertDeliveryChannel) AlertDeliveryChannelSummary {
	return AlertDeliveryChannelSummary{
		ID:          channel.ID,
		Label:       channel.Label,
		ChannelType: string(channel.ChannelType),
		Target:      channel.Target,
		Metadata:    domain.NormalizeAlertDefinition(channel.Metadata),
		IsEnabled:   channel.IsEnabled,
		CreatedAt:   channel.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   channel.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func newAlertDeliveryChannelID() string {
	return "channel_" + randomAlertDeliveryHex(8)
}

func randomAlertDeliveryHex(size int) string {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return strings.Repeat("0", size*2)
	}
	return hex.EncodeToString(buffer)
}

func (s *AlertDeliveryService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
