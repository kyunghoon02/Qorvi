package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/packages/domain"
	"github.com/qorvi/qorvi/packages/ops"
)

var (
	ErrAdminConsoleForbidden      = errors.New("admin console access is forbidden")
	ErrAdminLabelNotFound         = errors.New("admin label not found")
	ErrAdminSuppressionNotFound   = errors.New("admin suppression not found")
	ErrAdminCuratedListNotFound   = errors.New("admin curated list not found")
	ErrAdminConsoleInvalidRequest = errors.New("invalid admin console request")
)

type AdminLabelSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	CreatedBy   string `json:"createdBy"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type AdminSuppressionSummary struct {
	ID        string `json:"id"`
	Scope     string `json:"scope"`
	Target    string `json:"target"`
	Reason    string `json:"reason"`
	CreatedBy string `json:"createdBy"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	Active    bool   `json:"active"`
}

type AdminProviderQuotaSummary struct {
	Provider      string `json:"provider"`
	Status        string `json:"status"`
	Limit         int    `json:"limit"`
	Used          int    `json:"used"`
	Reserved      int    `json:"reserved"`
	WindowStart   string `json:"windowStart"`
	WindowEnd     string `json:"windowEnd"`
	LastCheckedAt string `json:"lastCheckedAt"`
}

type AdminProviderUsageSummary struct {
	Provider     string `json:"provider"`
	Status       string `json:"status"`
	Used24h      int    `json:"used24h"`
	Error24h     int    `json:"error24h"`
	AvgLatencyMs int    `json:"avgLatencyMs"`
	LastSeenAt   string `json:"lastSeenAt,omitempty"`
}

type AdminIngestSummary struct {
	LastBackfillAt   string `json:"lastBackfillAt,omitempty"`
	LastWebhookAt    string `json:"lastWebhookAt,omitempty"`
	FreshnessSeconds int    `json:"freshnessSeconds"`
	LagStatus        string `json:"lagStatus"`
}

type AdminAlertDeliverySummary struct {
	Attempts24h    int    `json:"attempts24h"`
	Delivered24h   int    `json:"delivered24h"`
	Failed24h      int    `json:"failed24h"`
	RetryableCount int    `json:"retryableCount"`
	LastFailureAt  string `json:"lastFailureAt,omitempty"`
}

type AdminWalletTrackingSummary struct {
	CandidateCount  int `json:"candidateCount"`
	TrackedCount    int `json:"trackedCount"`
	LabeledCount    int `json:"labeledCount"`
	ScoredCount     int `json:"scoredCount"`
	StaleCount      int `json:"staleCount"`
	SuppressedCount int `json:"suppressedCount"`
}

type AdminWalletTrackingSubscriptionSummary struct {
	PendingCount int    `json:"pendingCount"`
	ActiveCount  int    `json:"activeCount"`
	ErroredCount int    `json:"erroredCount"`
	PausedCount  int    `json:"pausedCount"`
	LastEventAt  string `json:"lastEventAt,omitempty"`
}

type AdminQueueDepthSummary struct {
	DefaultDepth  int `json:"defaultDepth"`
	PriorityDepth int `json:"priorityDepth"`
}

type AdminBackfillHealthSummary struct {
	Jobs24h         int    `json:"jobs24h"`
	Activities24h   int    `json:"activities24h"`
	Transactions24h int    `json:"transactions24h"`
	Expansions24h   int    `json:"expansions24h"`
	LastSuccessAt   string `json:"lastSuccessAt,omitempty"`
}

type AdminStaleRefreshSummary struct {
	Attempts24h   int    `json:"attempts24h"`
	Succeeded24h  int    `json:"succeeded24h"`
	Productive24h int    `json:"productive24h"`
	LastHitAt     string `json:"lastHitAt,omitempty"`
}

type AdminJobHealthSummary struct {
	JobName             string `json:"jobName"`
	LastStatus          string `json:"lastStatus"`
	LastStartedAt       string `json:"lastStartedAt"`
	LastFinishedAt      string `json:"lastFinishedAt,omitempty"`
	LastSuccessAt       string `json:"lastSuccessAt,omitempty"`
	MinutesSinceSuccess int    `json:"minutesSinceSuccess"`
	LastError           string `json:"lastError,omitempty"`
}

type AdminFailureSummary struct {
	Source     string         `json:"source"`
	Kind       string         `json:"kind"`
	OccurredAt string         `json:"occurredAt"`
	Summary    string         `json:"summary"`
	Details    map[string]any `json:"details"`
}

type AdminDomesticPrelistingCandidateSummary struct {
	Chain                     string `json:"chain"`
	TokenAddress              string `json:"tokenAddress"`
	TokenSymbol               string `json:"tokenSymbol"`
	NormalizedAssetKey        string `json:"normalizedAssetKey"`
	TransferCount7d           int    `json:"transferCount7d"`
	TransferCount24h          int    `json:"transferCount24h"`
	ActiveWalletCount         int    `json:"activeWalletCount"`
	TrackedWalletCount        int    `json:"trackedWalletCount"`
	DistinctCounterpartyCount int    `json:"distinctCounterpartyCount"`
	TotalAmount               string `json:"totalAmount"`
	LargestTransferAmount     string `json:"largestTransferAmount"`
	LatestObservedAt          string `json:"latestObservedAt"`
	ListedOnUpbit             bool   `json:"listedOnUpbit"`
	ListedOnBithumb           bool   `json:"listedOnBithumb"`
}

type AdminDomesticPrelistingCollection struct {
	Items []AdminDomesticPrelistingCandidateSummary `json:"items"`
}

type AdminObservabilityCollection struct {
	ProviderUsage         []AdminProviderUsageSummary            `json:"providerUsage"`
	Ingest                AdminIngestSummary                     `json:"ingest"`
	AlertDelivery         AdminAlertDeliverySummary              `json:"alertDelivery"`
	WalletTracking        AdminWalletTrackingSummary             `json:"walletTracking"`
	TrackingSubscriptions AdminWalletTrackingSubscriptionSummary `json:"trackingSubscriptions"`
	QueueDepth            AdminQueueDepthSummary                 `json:"queueDepth"`
	BackfillHealth        AdminBackfillHealthSummary             `json:"backfillHealth"`
	StaleRefresh          AdminStaleRefreshSummary               `json:"staleRefresh"`
	RecentRuns            []AdminJobHealthSummary                `json:"recentRuns"`
	RecentFailures        []AdminFailureSummary                  `json:"recentFailures"`
}

type AdminCuratedListItem struct {
	ID        string   `json:"id"`
	ItemType  string   `json:"itemType"`
	ItemKey   string   `json:"itemKey"`
	Tags      []string `json:"tags"`
	Notes     string   `json:"notes,omitempty"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

type AdminCuratedListDetail struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Notes     string                 `json:"notes,omitempty"`
	Tags      []string               `json:"tags"`
	ItemCount int                    `json:"itemCount"`
	Items     []AdminCuratedListItem `json:"items"`
	CreatedAt string                 `json:"createdAt"`
	UpdatedAt string                 `json:"updatedAt"`
}

type AdminCuratedListCollection struct {
	Items []AdminCuratedListDetail `json:"items"`
}

type AdminAuditEntry struct {
	Actor      string `json:"actor"`
	Action     string `json:"action"`
	TargetType string `json:"targetType"`
	TargetKey  string `json:"targetKey"`
	Note       string `json:"note,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

type AdminAuditCollection struct {
	Items []AdminAuditEntry `json:"items"`
}

type AdminLabelCollection struct {
	Items []AdminLabelSummary `json:"items"`
}

type AdminSuppressionCollection struct {
	Items []AdminSuppressionSummary `json:"items"`
}

type AdminProviderQuotaCollection struct {
	Items []AdminProviderQuotaSummary `json:"items"`
}

type AdminMutationResult struct {
	Deleted bool `json:"deleted"`
}

type UpsertAdminLabelRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type CreateAdminSuppressionRequest struct {
	Scope     string `json:"scope"`
	Target    string `json:"target"`
	Reason    string `json:"reason"`
	ExpiresAt string `json:"expiresAt"`
}

type CreateAdminCuratedListRequest struct {
	Name  string   `json:"name"`
	Notes string   `json:"notes"`
	Tags  []string `json:"tags"`
}

type CreateAdminCuratedListItemRequest struct {
	ItemType string   `json:"itemType"`
	ItemKey  string   `json:"itemKey"`
	Tags     []string `json:"tags"`
	Notes    string   `json:"notes"`
}

type AdminConsoleService struct {
	repo repository.AdminConsoleRepository
	Now  func() time.Time
}

func NewAdminConsoleService(repo repository.AdminConsoleRepository) *AdminConsoleService {
	return &AdminConsoleService{repo: repo, Now: time.Now}
}

func (s *AdminConsoleService) ListLabels(
	ctx context.Context,
	role string,
) (AdminLabelCollection, error) {
	if err := ensureAdminConsoleAccess(role, false); err != nil {
		return AdminLabelCollection{}, err
	}
	items, err := s.repo.ListLabels(ctx)
	if err != nil {
		return AdminLabelCollection{}, err
	}
	result := AdminLabelCollection{Items: make([]AdminLabelSummary, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, toAdminLabelSummary(item))
	}
	return result, nil
}

func (s *AdminConsoleService) UpsertLabel(
	ctx context.Context,
	role string,
	actor string,
	req UpsertAdminLabelRequest,
) (AdminLabelSummary, error) {
	if err := ensureAdminConsoleAccess(role, true); err != nil {
		return AdminLabelSummary{}, err
	}
	label, err := ops.BuildLabel(req.Name, req.Description, req.Color)
	if err != nil {
		return AdminLabelSummary{}, ErrAdminConsoleInvalidRequest
	}
	now := s.now()
	item, err := s.repo.UpsertLabel(ctx, repository.AdminLabel{
		ID:          newAdminResourceID("label"),
		Name:        label.Name,
		Description: label.Description,
		Color:       label.Color,
		CreatedBy:   strings.TrimSpace(actor),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return AdminLabelSummary{}, err
	}
	_ = s.repo.RecordAuditEntry(ctx, repository.AdminAuditEntry{
		Actor:      strings.TrimSpace(actor),
		Action:     "label_upsert",
		TargetType: "label",
		TargetKey:  item.Name,
		Note:       item.Description,
		CreatedAt:  s.now(),
	})
	return toAdminLabelSummary(item), nil
}

func (s *AdminConsoleService) DeleteLabel(
	ctx context.Context,
	role string,
	actor string,
	name string,
) error {
	if err := ensureAdminConsoleAccess(role, true); err != nil {
		return err
	}
	if err := s.repo.DeleteLabel(ctx, name); err != nil {
		if errors.Is(err, repository.ErrAdminLabelNotFound) {
			return ErrAdminLabelNotFound
		}
		return err
	}
	_ = s.repo.RecordAuditEntry(ctx, repository.AdminAuditEntry{
		Actor:      strings.TrimSpace(actor),
		Action:     "label_delete",
		TargetType: "label",
		TargetKey:  strings.TrimSpace(name),
		CreatedAt:  s.now(),
	})
	return nil
}

func (s *AdminConsoleService) ListSuppressions(
	ctx context.Context,
	role string,
) (AdminSuppressionCollection, error) {
	if err := ensureAdminConsoleAccess(role, false); err != nil {
		return AdminSuppressionCollection{}, err
	}
	items, err := s.repo.ListSuppressions(ctx)
	if err != nil {
		return AdminSuppressionCollection{}, err
	}
	result := AdminSuppressionCollection{Items: make([]AdminSuppressionSummary, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, toAdminSuppressionSummary(item))
	}
	return result, nil
}

func (s *AdminConsoleService) CreateSuppression(
	ctx context.Context,
	role string,
	actor string,
	req CreateAdminSuppressionRequest,
) (AdminSuppressionSummary, error) {
	if err := ensureAdminConsoleAccess(role, true); err != nil {
		return AdminSuppressionSummary{}, err
	}
	scope := ops.SuppressionScope(strings.TrimSpace(strings.ToLower(req.Scope)))
	var ttl time.Duration
	if strings.TrimSpace(req.ExpiresAt) != "" {
		expiresAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ExpiresAt))
		if err != nil {
			return AdminSuppressionSummary{}, ErrAdminConsoleInvalidRequest
		}
		ttl = time.Until(expiresAt)
		if ttl <= 0 {
			return AdminSuppressionSummary{}, ErrAdminConsoleInvalidRequest
		}
	}
	rule, err := ops.BuildSuppressionRule(
		scope,
		req.Target,
		req.Reason,
		actor,
		true,
		ttl,
	)
	if err != nil {
		return AdminSuppressionSummary{}, ErrAdminConsoleInvalidRequest
	}
	item, err := s.repo.CreateSuppression(ctx, repository.AdminSuppression{
		ID:        rule.ID,
		Scope:     string(rule.Scope),
		Target:    rule.Target,
		Reason:    rule.Reason,
		CreatedBy: rule.CreatedBy,
		CreatedAt: rule.CreatedAt,
		UpdatedAt: rule.CreatedAt,
		ExpiresAt: rule.ExpiresAt,
		Active:    rule.Active,
	})
	if err != nil {
		return AdminSuppressionSummary{}, err
	}
	_ = s.repo.RecordAuditEntry(ctx, repository.AdminAuditEntry{
		Actor:      strings.TrimSpace(actor),
		Action:     "suppression_add",
		TargetType: "suppression",
		TargetKey:  item.ID,
		Note:       item.Reason,
		CreatedAt:  s.now(),
	})
	return toAdminSuppressionSummary(item), nil
}

func (s *AdminConsoleService) DeleteSuppression(
	ctx context.Context,
	role string,
	actor string,
	id string,
) error {
	if err := ensureAdminConsoleAccess(role, true); err != nil {
		return err
	}
	if err := s.repo.DeleteSuppression(ctx, id); err != nil {
		if errors.Is(err, repository.ErrAdminSuppressionNotFound) {
			return ErrAdminSuppressionNotFound
		}
		return err
	}
	_ = s.repo.RecordAuditEntry(ctx, repository.AdminAuditEntry{
		Actor:      strings.TrimSpace(actor),
		Action:     "suppression_remove",
		TargetType: "suppression",
		TargetKey:  strings.TrimSpace(id),
		CreatedAt:  s.now(),
	})
	return nil
}

func (s *AdminConsoleService) ListProviderQuotas(
	ctx context.Context,
	role string,
) (AdminProviderQuotaCollection, error) {
	if err := ensureAdminConsoleAccess(role, false); err != nil {
		return AdminProviderQuotaCollection{}, err
	}
	items, err := s.repo.ListProviderQuotaSnapshots(ctx)
	if err != nil {
		return AdminProviderQuotaCollection{}, err
	}
	result := AdminProviderQuotaCollection{Items: make([]AdminProviderQuotaSummary, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, AdminProviderQuotaSummary{
			Provider:      item.Provider,
			Status:        item.Status,
			Limit:         item.Limit,
			Used:          item.Used,
			Reserved:      item.Reserved,
			WindowStart:   item.WindowStart.UTC().Format(time.RFC3339),
			WindowEnd:     item.WindowEnd.UTC().Format(time.RFC3339),
			LastCheckedAt: item.LastCheckedAt.UTC().Format(time.RFC3339),
		})
	}
	return result, nil
}

func (s *AdminConsoleService) ListObservability(
	ctx context.Context,
	role string,
) (AdminObservabilityCollection, error) {
	if err := ensureAdminConsoleAccess(role, false); err != nil {
		return AdminObservabilityCollection{}, err
	}
	item, err := s.repo.GetObservabilitySnapshot(ctx)
	if err != nil {
		return AdminObservabilityCollection{}, err
	}

	result := AdminObservabilityCollection{
		ProviderUsage: make([]AdminProviderUsageSummary, 0, len(item.ProviderUsage)),
		Ingest: AdminIngestSummary{
			FreshnessSeconds: item.Ingest.FreshnessSeconds,
			LagStatus:        item.Ingest.LagStatus,
		},
		AlertDelivery: AdminAlertDeliverySummary{
			Attempts24h:    item.AlertDelivery.Attempts24h,
			Delivered24h:   item.AlertDelivery.Delivered24h,
			Failed24h:      item.AlertDelivery.Failed24h,
			RetryableCount: item.AlertDelivery.RetryableCount,
		},
		WalletTracking: AdminWalletTrackingSummary{
			CandidateCount:  item.WalletTracking.CandidateCount,
			TrackedCount:    item.WalletTracking.TrackedCount,
			LabeledCount:    item.WalletTracking.LabeledCount,
			ScoredCount:     item.WalletTracking.ScoredCount,
			StaleCount:      item.WalletTracking.StaleCount,
			SuppressedCount: item.WalletTracking.SuppressedCount,
		},
		TrackingSubscriptions: AdminWalletTrackingSubscriptionSummary{
			PendingCount: item.TrackingSubscriptions.PendingCount,
			ActiveCount:  item.TrackingSubscriptions.ActiveCount,
			ErroredCount: item.TrackingSubscriptions.ErroredCount,
			PausedCount:  item.TrackingSubscriptions.PausedCount,
		},
		QueueDepth: AdminQueueDepthSummary{
			DefaultDepth:  item.QueueDepth.DefaultDepth,
			PriorityDepth: item.QueueDepth.PriorityDepth,
		},
		BackfillHealth: AdminBackfillHealthSummary{
			Jobs24h:         item.BackfillHealth.Jobs24h,
			Activities24h:   item.BackfillHealth.Activities24h,
			Transactions24h: item.BackfillHealth.Transactions24h,
			Expansions24h:   item.BackfillHealth.Expansions24h,
		},
		StaleRefresh: AdminStaleRefreshSummary{
			Attempts24h:   item.StaleRefresh.Attempts24h,
			Succeeded24h:  item.StaleRefresh.Succeeded24h,
			Productive24h: item.StaleRefresh.Productive24h,
		},
		RecentRuns:     make([]AdminJobHealthSummary, 0, len(item.RecentRuns)),
		RecentFailures: make([]AdminFailureSummary, 0, len(item.RecentFailures)),
	}
	for _, usage := range item.ProviderUsage {
		summary := AdminProviderUsageSummary{
			Provider:     usage.Provider,
			Status:       usage.Status,
			Used24h:      usage.Used24h,
			Error24h:     usage.Error24h,
			AvgLatencyMs: usage.AvgLatencyMs,
		}
		if usage.LastSeenAt != nil {
			summary.LastSeenAt = usage.LastSeenAt.UTC().Format(time.RFC3339)
		}
		result.ProviderUsage = append(result.ProviderUsage, summary)
	}
	if item.Ingest.LastBackfillAt != nil {
		result.Ingest.LastBackfillAt = item.Ingest.LastBackfillAt.UTC().Format(time.RFC3339)
	}
	if item.Ingest.LastWebhookAt != nil {
		result.Ingest.LastWebhookAt = item.Ingest.LastWebhookAt.UTC().Format(time.RFC3339)
	}
	if item.AlertDelivery.LastFailureAt != nil {
		result.AlertDelivery.LastFailureAt = item.AlertDelivery.LastFailureAt.UTC().Format(time.RFC3339)
	}
	if item.TrackingSubscriptions.LastEventAt != nil {
		result.TrackingSubscriptions.LastEventAt = item.TrackingSubscriptions.LastEventAt.UTC().Format(time.RFC3339)
	}
	if item.BackfillHealth.LastSuccessAt != nil {
		result.BackfillHealth.LastSuccessAt = item.BackfillHealth.LastSuccessAt.UTC().Format(time.RFC3339)
	}
	if item.StaleRefresh.LastHitAt != nil {
		result.StaleRefresh.LastHitAt = item.StaleRefresh.LastHitAt.UTC().Format(time.RFC3339)
	}
	for _, run := range item.RecentRuns {
		summary := AdminJobHealthSummary{
			JobName:             run.JobName,
			LastStatus:          run.LastStatus,
			LastStartedAt:       run.LastStartedAt.UTC().Format(time.RFC3339),
			MinutesSinceSuccess: run.MinutesSinceSuccess,
			LastError:           run.LastError,
		}
		if run.LastFinishedAt != nil {
			summary.LastFinishedAt = run.LastFinishedAt.UTC().Format(time.RFC3339)
		}
		if run.LastSuccessAt != nil {
			summary.LastSuccessAt = run.LastSuccessAt.UTC().Format(time.RFC3339)
		}
		result.RecentRuns = append(result.RecentRuns, summary)
	}
	for _, failure := range item.RecentFailures {
		result.RecentFailures = append(result.RecentFailures, AdminFailureSummary{
			Source:     failure.Source,
			Kind:       failure.Kind,
			OccurredAt: failure.OccurredAt.UTC().Format(time.RFC3339),
			Summary:    failure.Summary,
			Details:    cloneAdminConsoleDetails(failure.Details),
		})
	}
	return result, nil
}

func (s *AdminConsoleService) ListDomesticPrelistingCandidates(
	ctx context.Context,
	role string,
	limit int,
) (AdminDomesticPrelistingCollection, error) {
	if err := ensureAdminConsoleAccess(role, false); err != nil {
		return AdminDomesticPrelistingCollection{}, err
	}
	items, err := s.repo.ListDomesticPrelistingCandidates(ctx, limit)
	if err != nil {
		return AdminDomesticPrelistingCollection{}, err
	}
	result := AdminDomesticPrelistingCollection{
		Items: make([]AdminDomesticPrelistingCandidateSummary, 0, len(items)),
	}
	for _, item := range items {
		result.Items = append(result.Items, AdminDomesticPrelistingCandidateSummary{
			Chain:                     item.Chain,
			TokenAddress:              item.TokenAddress,
			TokenSymbol:               item.TokenSymbol,
			NormalizedAssetKey:        item.NormalizedAssetKey,
			TransferCount7d:           item.TransferCount7d,
			TransferCount24h:          item.TransferCount24h,
			ActiveWalletCount:         item.ActiveWalletCount,
			TrackedWalletCount:        item.TrackedWalletCount,
			DistinctCounterpartyCount: item.DistinctCounterpartyCount,
			TotalAmount:               item.TotalAmount,
			LargestTransferAmount:     item.LargestTransferAmount,
			LatestObservedAt:          item.LatestObservedAt.UTC().Format(time.RFC3339),
			ListedOnUpbit:             item.ListedOnUpbit,
			ListedOnBithumb:           item.ListedOnBithumb,
		})
	}
	return result, nil
}

func (s *AdminConsoleService) ListCuratedLists(ctx context.Context, role string) (AdminCuratedListCollection, error) {
	if err := ensureAdminConsoleAccess(role, false); err != nil {
		return AdminCuratedListCollection{}, err
	}
	items, err := s.repo.ListCuratedLists(ctx)
	if err != nil {
		return AdminCuratedListCollection{}, err
	}
	result := AdminCuratedListCollection{Items: make([]AdminCuratedListDetail, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, toAdminCuratedListDetail(item))
	}
	return result, nil
}

func (s *AdminConsoleService) CreateCuratedList(ctx context.Context, role string, actor string, req CreateAdminCuratedListRequest) (AdminCuratedListDetail, error) {
	if err := ensureAdminConsoleAccess(role, true); err != nil {
		return AdminCuratedListDetail{}, err
	}
	name, err := domain.NormalizeWatchlistName(req.Name)
	if err != nil {
		return AdminCuratedListDetail{}, ErrAdminConsoleInvalidRequest
	}
	item, err := s.repo.CreateCuratedList(ctx, repository.AdminCuratedList{
		ID:        newAdminResourceID("curated"),
		Name:      name,
		Notes:     domain.NormalizeWatchlistNotes(req.Notes),
		Tags:      domain.NormalizeWatchlistTags(req.Tags),
		CreatedAt: s.now(),
		UpdatedAt: s.now(),
	})
	if err != nil {
		return AdminCuratedListDetail{}, err
	}
	_ = s.repo.RecordAuditEntry(ctx, repository.AdminAuditEntry{
		Actor:      strings.TrimSpace(actor),
		Action:     "curated_list_create",
		TargetType: "curated_list",
		TargetKey:  item.ID,
		Note:       item.Name,
		CreatedAt:  s.now(),
	})
	return toAdminCuratedListDetail(item), nil
}

func (s *AdminConsoleService) DeleteCuratedList(ctx context.Context, role string, actor string, id string) error {
	if err := ensureAdminConsoleAccess(role, true); err != nil {
		return err
	}
	if err := s.repo.DeleteCuratedList(ctx, id); err != nil {
		if errors.Is(err, repository.ErrAdminCuratedListNotFound) {
			return ErrAdminCuratedListNotFound
		}
		return err
	}
	_ = s.repo.RecordAuditEntry(ctx, repository.AdminAuditEntry{
		Actor:      strings.TrimSpace(actor),
		Action:     "curated_list_delete",
		TargetType: "curated_list",
		TargetKey:  strings.TrimSpace(id),
		CreatedAt:  s.now(),
	})
	return nil
}

func (s *AdminConsoleService) AddCuratedListItem(ctx context.Context, role string, actor string, listID string, req CreateAdminCuratedListItemRequest) (AdminCuratedListDetail, error) {
	if err := ensureAdminConsoleAccess(role, true); err != nil {
		return AdminCuratedListDetail{}, err
	}
	itemType, err := domain.NormalizeWatchlistItemType(req.ItemType)
	if err != nil || strings.TrimSpace(req.ItemKey) == "" {
		return AdminCuratedListDetail{}, ErrAdminConsoleInvalidRequest
	}
	item, err := s.repo.AddCuratedListItem(ctx, listID, repository.AdminCuratedListItem{
		ID:        newAdminResourceID("curated_item"),
		ItemType:  string(itemType),
		ItemKey:   strings.TrimSpace(req.ItemKey),
		Tags:      domain.NormalizeWatchlistTags(req.Tags),
		Notes:     domain.NormalizeWatchlistNotes(req.Notes),
		CreatedAt: s.now(),
		UpdatedAt: s.now(),
	})
	if err != nil {
		if errors.Is(err, repository.ErrAdminCuratedListNotFound) {
			return AdminCuratedListDetail{}, ErrAdminCuratedListNotFound
		}
		return AdminCuratedListDetail{}, err
	}
	_ = s.repo.RecordAuditEntry(ctx, repository.AdminAuditEntry{
		Actor:      strings.TrimSpace(actor),
		Action:     "curated_item_add",
		TargetType: "curated_list",
		TargetKey:  strings.TrimSpace(listID),
		Note:       strings.TrimSpace(req.ItemKey),
		CreatedAt:  s.now(),
	})
	return toAdminCuratedListDetail(item), nil
}

func (s *AdminConsoleService) DeleteCuratedListItem(ctx context.Context, role string, actor string, listID string, itemID string) (AdminCuratedListDetail, error) {
	if err := ensureAdminConsoleAccess(role, true); err != nil {
		return AdminCuratedListDetail{}, err
	}
	item, err := s.repo.DeleteCuratedListItem(ctx, listID, itemID)
	if err != nil {
		if errors.Is(err, repository.ErrAdminCuratedListNotFound) {
			return AdminCuratedListDetail{}, ErrAdminCuratedListNotFound
		}
		return AdminCuratedListDetail{}, err
	}
	_ = s.repo.RecordAuditEntry(ctx, repository.AdminAuditEntry{
		Actor:      strings.TrimSpace(actor),
		Action:     "curated_item_delete",
		TargetType: "curated_list",
		TargetKey:  strings.TrimSpace(listID),
		Note:       strings.TrimSpace(itemID),
		CreatedAt:  s.now(),
	})
	return toAdminCuratedListDetail(item), nil
}

func (s *AdminConsoleService) ListAuditEntries(ctx context.Context, role string, limit int) (AdminAuditCollection, error) {
	if err := ensureAdminConsoleAccess(role, false); err != nil {
		return AdminAuditCollection{}, err
	}
	items, err := s.repo.ListAuditEntries(ctx, limit)
	if err != nil {
		return AdminAuditCollection{}, err
	}
	result := AdminAuditCollection{Items: make([]AdminAuditEntry, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, AdminAuditEntry{
			Actor:      item.Actor,
			Action:     item.Action,
			TargetType: item.TargetType,
			TargetKey:  item.TargetKey,
			Note:       item.Note,
			CreatedAt:  item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return result, nil
}

func toAdminLabelSummary(item repository.AdminLabel) AdminLabelSummary {
	return AdminLabelSummary{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		Color:       item.Color,
		CreatedBy:   item.CreatedBy,
		CreatedAt:   item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toAdminSuppressionSummary(item repository.AdminSuppression) AdminSuppressionSummary {
	summary := AdminSuppressionSummary{
		ID:        item.ID,
		Scope:     item.Scope,
		Target:    item.Target,
		Reason:    item.Reason,
		CreatedBy: item.CreatedBy,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
		Active:    item.Active,
	}
	if item.ExpiresAt != nil {
		summary.ExpiresAt = item.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return summary
}

func toAdminCuratedListDetail(item repository.AdminCuratedList) AdminCuratedListDetail {
	result := AdminCuratedListDetail{
		ID:        item.ID,
		Name:      item.Name,
		Notes:     item.Notes,
		Tags:      append([]string(nil), item.Tags...),
		ItemCount: item.ItemCount,
		Items:     make([]AdminCuratedListItem, 0, len(item.Items)),
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
	}
	for _, listItem := range item.Items {
		result.Items = append(result.Items, AdminCuratedListItem{
			ID:        listItem.ID,
			ItemType:  listItem.ItemType,
			ItemKey:   listItem.ItemKey,
			Tags:      append([]string(nil), listItem.Tags...),
			Notes:     listItem.Notes,
			CreatedAt: listItem.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: listItem.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return result
}

func ensureAdminConsoleAccess(role string, mutating bool) error {
	normalized := strings.TrimSpace(strings.ToLower(role))
	switch normalized {
	case "admin":
		return nil
	case "operator":
		if mutating {
			return ErrAdminConsoleForbidden
		}
		return nil
	default:
		return ErrAdminConsoleForbidden
	}
}

func newAdminResourceID(prefix string) string {
	entropy := make([]byte, 8)
	if _, err := rand.Read(entropy); err != nil {
		return prefix + "_fallback"
	}
	return prefix + "_" + hex.EncodeToString(entropy)
}

func (s *AdminConsoleService) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func cloneAdminConsoleDetails(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
