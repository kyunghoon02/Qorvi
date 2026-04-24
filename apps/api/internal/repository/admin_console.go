package repository

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
	"github.com/qorvi/qorvi/packages/ops"
)

var (
	ErrAdminLabelNotFound       = errors.New("admin label not found")
	ErrAdminSuppressionNotFound = errors.New("admin suppression not found")
	ErrAdminCuratedListNotFound = errors.New("admin curated list not found")
)

type AdminLabel struct {
	ID          string
	Name        string
	Description string
	Color       string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AdminSuppression struct {
	ID        string
	Scope     string
	Target    string
	Reason    string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt *time.Time
	Active    bool
}

type AdminQuotaSnapshot struct {
	Provider      string
	WindowStart   time.Time
	WindowEnd     time.Time
	Limit         int
	Used          int
	Reserved      int
	LastCheckedAt time.Time
	Status        string
}

type AdminProviderUsageSnapshot struct {
	Provider     string
	Status       string
	Used24h      int
	Error24h     int
	AvgLatencyMs int
	LastSeenAt   *time.Time
}

type AdminIngestSnapshot struct {
	LastBackfillAt   *time.Time
	LastWebhookAt    *time.Time
	FreshnessSeconds int
	LagStatus        string
}

type AdminAlertDeliverySnapshot struct {
	Attempts24h    int
	Delivered24h   int
	Failed24h      int
	RetryableCount int
	LastFailureAt  *time.Time
}

type AdminWalletTrackingSnapshot struct {
	CandidateCount  int
	TrackedCount    int
	LabeledCount    int
	ScoredCount     int
	StaleCount      int
	SuppressedCount int
}

type AdminWalletTrackingSubscriptionSnapshot struct {
	PendingCount int
	ActiveCount  int
	ErroredCount int
	PausedCount  int
	LastEventAt  *time.Time
}

type AdminQueueDepthSnapshot struct {
	DefaultDepth  int
	PriorityDepth int
}

type AdminBackfillHealthSnapshot struct {
	Jobs24h         int
	Activities24h   int
	Transactions24h int
	Expansions24h   int
	LastSuccessAt   *time.Time
}

type AdminStaleRefreshSnapshot struct {
	Attempts24h   int
	Succeeded24h  int
	Productive24h int
	LastHitAt     *time.Time
}

type AdminJobHealthSnapshot struct {
	JobName             string
	LastStatus          string
	LastStartedAt       time.Time
	LastFinishedAt      *time.Time
	LastSuccessAt       *time.Time
	MinutesSinceSuccess int
	LastError           string
}

type AdminFailureSnapshot struct {
	Source     string
	Kind       string
	OccurredAt time.Time
	Summary    string
	Details    map[string]any
}

type AdminDomesticPrelistingCandidate struct {
	Chain                     string
	TokenAddress              string
	TokenSymbol               string
	NormalizedAssetKey        string
	TransferCount7d           int
	TransferCount24h          int
	ActiveWalletCount         int
	TrackedWalletCount        int
	DistinctCounterpartyCount int
	TotalAmount               string
	LargestTransferAmount     string
	LatestObservedAt          time.Time
	ListedOnUpbit             bool
	ListedOnBithumb           bool
}

type AdminObservabilitySnapshot struct {
	ProviderUsage         []AdminProviderUsageSnapshot
	Ingest                AdminIngestSnapshot
	AlertDelivery         AdminAlertDeliverySnapshot
	WalletTracking        AdminWalletTrackingSnapshot
	TrackingSubscriptions AdminWalletTrackingSubscriptionSnapshot
	QueueDepth            AdminQueueDepthSnapshot
	BackfillHealth        AdminBackfillHealthSnapshot
	StaleRefresh          AdminStaleRefreshSnapshot
	RecentRuns            []AdminJobHealthSnapshot
	RecentFailures        []AdminFailureSnapshot
}

type AdminCuratedList struct {
	ID        string
	Name      string
	Notes     string
	Tags      []string
	ItemCount int
	Items     []AdminCuratedListItem
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AdminCuratedListItem struct {
	ID        string
	ItemType  string
	ItemKey   string
	Tags      []string
	Notes     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AdminAuditEntry struct {
	Actor      string
	Action     string
	TargetType string
	TargetKey  string
	Note       string
	CreatedAt  time.Time
}

type AdminConsoleRepository interface {
	ListLabels(context.Context) ([]AdminLabel, error)
	UpsertLabel(context.Context, AdminLabel) (AdminLabel, error)
	DeleteLabel(context.Context, string) error
	ListSuppressions(context.Context) ([]AdminSuppression, error)
	CreateSuppression(context.Context, AdminSuppression) (AdminSuppression, error)
	DeleteSuppression(context.Context, string) error
	ListProviderQuotaSnapshots(context.Context) ([]AdminQuotaSnapshot, error)
	ListCuratedLists(context.Context) ([]AdminCuratedList, error)
	CreateCuratedList(context.Context, AdminCuratedList) (AdminCuratedList, error)
	DeleteCuratedList(context.Context, string) error
	AddCuratedListItem(context.Context, string, AdminCuratedListItem) (AdminCuratedList, error)
	DeleteCuratedListItem(context.Context, string, string) (AdminCuratedList, error)
	ListAuditEntries(context.Context, int) ([]AdminAuditEntry, error)
	RecordAuditEntry(context.Context, AdminAuditEntry) error
	GetObservabilitySnapshot(context.Context) (AdminObservabilitySnapshot, error)
	ListDomesticPrelistingCandidates(context.Context, int) ([]AdminDomesticPrelistingCandidate, error)
}

type InMemoryAdminConsoleRepository struct {
	mu            sync.RWMutex
	labels        map[string]AdminLabel
	suppressions  map[string]AdminSuppression
	quotas        []AdminQuotaSnapshot
	observability AdminObservabilitySnapshot
	domestic      []AdminDomesticPrelistingCandidate
	curated       map[string]AdminCuratedList
	audits        []AdminAuditEntry
}

func NewInMemoryAdminConsoleRepository() *InMemoryAdminConsoleRepository {
	return &InMemoryAdminConsoleRepository{
		labels:       make(map[string]AdminLabel),
		suppressions: make(map[string]AdminSuppression),
		quotas:       []AdminQuotaSnapshot{},
		curated:      make(map[string]AdminCuratedList),
		audits:       []AdminAuditEntry{},
	}
}

func (r *InMemoryAdminConsoleRepository) SeedQuotaSnapshots(items []AdminQuotaSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.quotas = append([]AdminQuotaSnapshot(nil), items...)
}

func (r *InMemoryAdminConsoleRepository) SeedObservabilitySnapshot(item AdminObservabilitySnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observability = item
}

func (r *InMemoryAdminConsoleRepository) SeedDomesticPrelistingCandidates(items []AdminDomesticPrelistingCandidate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.domestic = append([]AdminDomesticPrelistingCandidate(nil), items...)
}

func (r *InMemoryAdminConsoleRepository) ListLabels(_ context.Context) ([]AdminLabel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]AdminLabel, 0, len(r.labels))
	for _, item := range r.labels {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (r *InMemoryAdminConsoleRepository) UpsertLabel(_ context.Context, item AdminLabel) (AdminLabel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := item
	key := strings.TrimSpace(strings.ToLower(item.Name))
	r.labels[key] = cloned
	return cloned, nil
}

func (r *InMemoryAdminConsoleRepository) DeleteLabel(_ context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.TrimSpace(strings.ToLower(name))
	if _, ok := r.labels[key]; !ok {
		return ErrAdminLabelNotFound
	}
	delete(r.labels, key)
	return nil
}

func (r *InMemoryAdminConsoleRepository) ListSuppressions(_ context.Context) ([]AdminSuppression, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]AdminSuppression, 0, len(r.suppressions))
	for _, item := range r.suppressions {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Active != items[j].Active {
			return items[i].Active
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items, nil
}

func (r *InMemoryAdminConsoleRepository) CreateSuppression(_ context.Context, item AdminSuppression) (AdminSuppression, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.suppressions[item.ID] = item
	return item, nil
}

func (r *InMemoryAdminConsoleRepository) DeleteSuppression(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.suppressions[strings.TrimSpace(id)]
	if !ok {
		return ErrAdminSuppressionNotFound
	}
	item.Active = false
	item.UpdatedAt = time.Now().UTC()
	r.suppressions[item.ID] = item
	return nil
}

func (r *InMemoryAdminConsoleRepository) ListProviderQuotaSnapshots(_ context.Context) ([]AdminQuotaSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]AdminQuotaSnapshot(nil), r.quotas...), nil
}

func (r *InMemoryAdminConsoleRepository) ListCuratedLists(_ context.Context) ([]AdminCuratedList, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]AdminCuratedList, 0, len(r.curated))
	for _, item := range r.curated {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}

func (r *InMemoryAdminConsoleRepository) CreateCuratedList(_ context.Context, item AdminCuratedList) (AdminCuratedList, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.curated[item.ID] = item
	return item, nil
}

func (r *InMemoryAdminConsoleRepository) DeleteCuratedList(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.curated[strings.TrimSpace(id)]; !ok {
		return ErrAdminCuratedListNotFound
	}
	delete(r.curated, strings.TrimSpace(id))
	return nil
}

func (r *InMemoryAdminConsoleRepository) AddCuratedListItem(_ context.Context, listID string, item AdminCuratedListItem) (AdminCuratedList, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	list, ok := r.curated[strings.TrimSpace(listID)]
	if !ok {
		return AdminCuratedList{}, ErrAdminCuratedListNotFound
	}
	list.Items = append([]AdminCuratedListItem{item}, list.Items...)
	list.ItemCount = len(list.Items)
	list.UpdatedAt = item.UpdatedAt
	r.curated[list.ID] = list
	return list, nil
}

func (r *InMemoryAdminConsoleRepository) DeleteCuratedListItem(_ context.Context, listID string, itemID string) (AdminCuratedList, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	list, ok := r.curated[strings.TrimSpace(listID)]
	if !ok {
		return AdminCuratedList{}, ErrAdminCuratedListNotFound
	}
	next := make([]AdminCuratedListItem, 0, len(list.Items))
	found := false
	for _, item := range list.Items {
		if item.ID == strings.TrimSpace(itemID) {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		return AdminCuratedList{}, ErrAdminCuratedListNotFound
	}
	list.Items = next
	list.ItemCount = len(next)
	list.UpdatedAt = time.Now().UTC()
	r.curated[list.ID] = list
	return list, nil
}

func (r *InMemoryAdminConsoleRepository) ListAuditEntries(_ context.Context, limit int) ([]AdminAuditEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > len(r.audits) {
		limit = len(r.audits)
	}
	return append([]AdminAuditEntry(nil), r.audits[:limit]...), nil
}

func (r *InMemoryAdminConsoleRepository) RecordAuditEntry(_ context.Context, item AdminAuditEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.audits = append([]AdminAuditEntry{item}, r.audits...)
	return nil
}

func (r *InMemoryAdminConsoleRepository) GetObservabilitySnapshot(_ context.Context) (AdminObservabilitySnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.observability, nil
}

func (r *InMemoryAdminConsoleRepository) ListDomesticPrelistingCandidates(_ context.Context, limit int) ([]AdminDomesticPrelistingCandidate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > len(r.domestic) {
		limit = len(r.domestic)
	}
	return append([]AdminDomesticPrelistingCandidate(nil), r.domestic[:limit]...), nil
}

type PostgresAdminConsoleRepository struct {
	store          *db.PostgresAdminConsoleStore
	domestic       *db.PostgresDomesticPrelistingStore
	queueStats     *db.RedisAdminQueueStatsStore
	watchlistStore *db.PostgresWatchlistStore
	auditStore     *db.PostgresAuditLogStore
	entityIndex    *db.PostgresCuratedEntityIndexStore
}

func NewPostgresAdminConsoleRepository(
	store *db.PostgresAdminConsoleStore,
	queueStats *db.RedisAdminQueueStatsStore,
	watchlistStore *db.PostgresWatchlistStore,
	auditStore *db.PostgresAuditLogStore,
	extras ...interface{},
) *PostgresAdminConsoleRepository {
	repo := &PostgresAdminConsoleRepository{
		store:          store,
		queueStats:     queueStats,
		watchlistStore: watchlistStore,
		auditStore:     auditStore,
	}
	for _, extra := range extras {
		switch typed := extra.(type) {
		case *db.PostgresCuratedEntityIndexStore:
			repo.entityIndex = typed
		case *db.PostgresDomesticPrelistingStore:
			repo.domestic = typed
		}
	}
	return repo
}

func (r *PostgresAdminConsoleRepository) ListLabels(ctx context.Context) ([]AdminLabel, error) {
	if r == nil || r.store == nil {
		return []AdminLabel{}, nil
	}
	items, err := r.store.ListAdminLabels(ctx)
	if err != nil {
		return nil, translateAdminConsoleError(err)
	}
	result := make([]AdminLabel, 0, len(items))
	for _, item := range items {
		result = append(result, AdminLabel(item))
	}
	return result, nil
}

func (r *PostgresAdminConsoleRepository) UpsertLabel(ctx context.Context, item AdminLabel) (AdminLabel, error) {
	if r == nil || r.store == nil {
		return AdminLabel{}, nil
	}
	record, err := r.store.UpsertAdminLabel(ctx, ops.Label{
		Name:        item.Name,
		Description: item.Description,
		Color:       item.Color,
	}, item.CreatedBy)
	if err != nil {
		return AdminLabel{}, translateAdminConsoleError(err)
	}
	return AdminLabel(record), nil
}

func (r *PostgresAdminConsoleRepository) DeleteLabel(ctx context.Context, name string) error {
	if r == nil || r.store == nil {
		return nil
	}
	return translateAdminConsoleError(r.store.DeleteAdminLabel(ctx, name))
}

func (r *PostgresAdminConsoleRepository) ListSuppressions(ctx context.Context) ([]AdminSuppression, error) {
	if r == nil || r.store == nil {
		return []AdminSuppression{}, nil
	}
	items, err := r.store.ListSuppressions(ctx)
	if err != nil {
		return nil, translateAdminConsoleError(err)
	}
	result := make([]AdminSuppression, 0, len(items))
	for _, item := range items {
		result = append(result, AdminSuppression{
			ID:        item.ID,
			Scope:     string(item.Rule.Scope),
			Target:    item.Rule.Target,
			Reason:    item.Rule.Reason,
			CreatedBy: item.Rule.CreatedBy,
			CreatedAt: item.Rule.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			ExpiresAt: item.Rule.ExpiresAt,
			Active:    item.Rule.Active,
		})
	}
	return result, nil
}

func (r *PostgresAdminConsoleRepository) CreateSuppression(ctx context.Context, item AdminSuppression) (AdminSuppression, error) {
	if r == nil || r.store == nil {
		return AdminSuppression{}, nil
	}
	record, err := r.store.CreateSuppression(ctx, ops.SuppressionRule{
		ID:        item.ID,
		Scope:     ops.SuppressionScope(item.Scope),
		Target:    item.Target,
		Reason:    item.Reason,
		CreatedBy: item.CreatedBy,
		CreatedAt: item.CreatedAt,
		ExpiresAt: item.ExpiresAt,
		Active:    item.Active,
	})
	if err != nil {
		return AdminSuppression{}, translateAdminConsoleError(err)
	}
	return AdminSuppression{
		ID:        record.ID,
		Scope:     string(record.Rule.Scope),
		Target:    record.Rule.Target,
		Reason:    record.Rule.Reason,
		CreatedBy: record.Rule.CreatedBy,
		CreatedAt: record.Rule.CreatedAt,
		UpdatedAt: record.UpdatedAt,
		ExpiresAt: record.Rule.ExpiresAt,
		Active:    record.Rule.Active,
	}, nil
}

func (r *PostgresAdminConsoleRepository) DeleteSuppression(ctx context.Context, id string) error {
	if r == nil || r.store == nil {
		return nil
	}
	_, err := r.store.DeactivateSuppression(ctx, id)
	return translateAdminConsoleError(err)
}

func (r *PostgresAdminConsoleRepository) ListProviderQuotaSnapshots(ctx context.Context) ([]AdminQuotaSnapshot, error) {
	if r == nil || r.store == nil {
		return []AdminQuotaSnapshot{}, nil
	}
	items, err := r.store.ListProviderQuotaSnapshots(ctx, 24*time.Hour, map[ops.ProviderName]int{
		ops.ProviderDune:    1000,
		ops.ProviderAlchemy: 5000,
		ops.ProviderHelius:  5000,
		ops.ProviderMoralis: 2000,
	})
	if err != nil {
		return nil, translateAdminConsoleError(err)
	}
	result := make([]AdminQuotaSnapshot, 0, len(items))
	for _, item := range items {
		status, _ := ops.ClassifyQuotaStatus(item)
		result = append(result, AdminQuotaSnapshot{
			Provider:      string(item.Provider),
			WindowStart:   item.WindowStart,
			WindowEnd:     item.WindowEnd,
			Limit:         item.Limit,
			Used:          item.Used,
			Reserved:      item.Reserved,
			LastCheckedAt: item.LastCheckedAt,
			Status:        string(status),
		})
	}
	return result, nil
}

func (r *PostgresAdminConsoleRepository) ListCuratedLists(ctx context.Context) ([]AdminCuratedList, error) {
	if r == nil || r.watchlistStore == nil {
		return []AdminCuratedList{}, nil
	}
	items, err := r.watchlistStore.ListWatchlists(ctx, db.AdminCuratedOwnerUserID)
	if err != nil {
		return nil, translateAdminConsoleError(err)
	}
	result := make([]AdminCuratedList, 0, len(items))
	for _, item := range items {
		result = append(result, toAdminCuratedList(item))
	}
	return result, nil
}

func (r *PostgresAdminConsoleRepository) CreateCuratedList(ctx context.Context, item AdminCuratedList) (AdminCuratedList, error) {
	if r == nil || r.watchlistStore == nil {
		return AdminCuratedList{}, nil
	}
	created, err := r.watchlistStore.CreateWatchlist(ctx, db.AdminCuratedOwnerUserID, item.Name, item.Notes, item.Tags)
	if err != nil {
		return AdminCuratedList{}, translateAdminConsoleError(err)
	}
	if err := r.syncEntityIndex(ctx); err != nil {
		return AdminCuratedList{}, err
	}
	return toAdminCuratedList(created), nil
}

func (r *PostgresAdminConsoleRepository) DeleteCuratedList(ctx context.Context, id string) error {
	if r == nil || r.watchlistStore == nil {
		return nil
	}
	if err := r.watchlistStore.DeleteWatchlist(ctx, db.AdminCuratedOwnerUserID, id); err != nil {
		return translateAdminConsoleError(err)
	}
	return r.syncEntityIndex(ctx)
}

func (r *PostgresAdminConsoleRepository) AddCuratedListItem(ctx context.Context, listID string, item AdminCuratedListItem) (AdminCuratedList, error) {
	if r == nil || r.watchlistStore == nil {
		return AdminCuratedList{}, nil
	}
	if _, err := r.watchlistStore.AddWatchlistItem(
		ctx,
		db.AdminCuratedOwnerUserID,
		listID,
		domain.WatchlistItemType(item.ItemType),
		item.ItemKey,
		item.Tags,
		item.Notes,
	); err != nil {
		return AdminCuratedList{}, translateAdminConsoleError(err)
	}
	if err := r.syncEntityIndex(ctx); err != nil {
		return AdminCuratedList{}, err
	}
	return r.findCuratedList(ctx, listID)
}

func (r *PostgresAdminConsoleRepository) DeleteCuratedListItem(ctx context.Context, listID string, itemID string) (AdminCuratedList, error) {
	if r == nil || r.watchlistStore == nil {
		return AdminCuratedList{}, nil
	}
	if err := r.watchlistStore.DeleteWatchlistItem(ctx, db.AdminCuratedOwnerUserID, listID, itemID); err != nil {
		return AdminCuratedList{}, translateAdminConsoleError(err)
	}
	if err := r.syncEntityIndex(ctx); err != nil {
		return AdminCuratedList{}, err
	}
	return r.findCuratedList(ctx, listID)
}

func (r *PostgresAdminConsoleRepository) syncEntityIndex(ctx context.Context) error {
	if r == nil || r.entityIndex == nil {
		return nil
	}
	return r.entityIndex.SyncAdminCuratedEntityIndex(ctx, db.AdminCuratedOwnerUserID)
}

func (r *PostgresAdminConsoleRepository) ListAuditEntries(ctx context.Context, limit int) ([]AdminAuditEntry, error) {
	if r == nil || r.auditStore == nil {
		return []AdminAuditEntry{}, nil
	}
	items, err := r.auditStore.ListAuditLogs(ctx, limit)
	if err != nil {
		return nil, translateAdminConsoleError(err)
	}
	result := make([]AdminAuditEntry, 0, len(items))
	for _, item := range items {
		result = append(result, AdminAuditEntry{
			Actor:      item.ActorUserID,
			Action:     item.Action,
			TargetType: item.TargetType,
			TargetKey:  item.TargetKey,
			Note:       stringValueFromMap(item.Payload, "note"),
			CreatedAt:  item.CreatedAt,
		})
	}
	return result, nil
}

func (r *PostgresAdminConsoleRepository) RecordAuditEntry(ctx context.Context, item AdminAuditEntry) error {
	if r == nil || r.auditStore == nil {
		return nil
	}
	return translateAdminConsoleError(r.auditStore.RecordAuditLog(ctx, db.AuditLogEntry{
		ActorUserID: item.Actor,
		Action:      item.Action,
		TargetType:  item.TargetType,
		TargetKey:   item.TargetKey,
		Payload: map[string]any{
			"note": item.Note,
		},
		CreatedAt: item.CreatedAt,
	}))
}

func (r *PostgresAdminConsoleRepository) GetObservabilitySnapshot(ctx context.Context) (AdminObservabilitySnapshot, error) {
	if r == nil || r.store == nil {
		return AdminObservabilitySnapshot{}, nil
	}

	now := time.Now().UTC()
	providerRows, err := r.store.ListProviderUsageStats(ctx, 24*time.Hour)
	if err != nil {
		return AdminObservabilitySnapshot{}, translateAdminConsoleError(err)
	}
	usageByProvider := make(map[string]db.AdminProviderUsageStatRecord, len(providerRows))
	for _, item := range providerRows {
		usageByProvider[strings.TrimSpace(strings.ToLower(item.Provider))] = item
	}

	providerUsage := make([]AdminProviderUsageSnapshot, 0, 4)
	for _, provider := range []ops.ProviderName{
		ops.ProviderDune,
		ops.ProviderAlchemy,
		ops.ProviderHelius,
		ops.ProviderMoralis,
	} {
		record := usageByProvider[strings.TrimSpace(strings.ToLower(string(provider)))]
		providerUsage = append(providerUsage, AdminProviderUsageSnapshot{
			Provider:     string(provider),
			Status:       classifyProviderUsageStatus(record, now),
			Used24h:      record.Used24h,
			Error24h:     record.Error24h,
			AvgLatencyMs: record.AvgLatencyMs,
			LastSeenAt:   copyTimePtr(record.LastSeenAt),
		})
	}

	ingestRecord, err := r.store.ReadIngestFreshness(ctx)
	if err != nil {
		return AdminObservabilitySnapshot{}, translateAdminConsoleError(err)
	}
	latestSeen := maxTimePtr(ingestRecord.LastBackfillAt, ingestRecord.LastWebhookAt)
	ingest := AdminIngestSnapshot{
		LastBackfillAt:   copyTimePtr(ingestRecord.LastBackfillAt),
		LastWebhookAt:    copyTimePtr(ingestRecord.LastWebhookAt),
		FreshnessSeconds: ageSeconds(latestSeen, now),
		LagStatus:        classifyIngestLagStatus(latestSeen, now),
	}

	alertRecord, err := r.store.ReadAlertDeliveryHealth(ctx, 24*time.Hour)
	if err != nil {
		return AdminObservabilitySnapshot{}, translateAdminConsoleError(err)
	}
	alertDelivery := AdminAlertDeliverySnapshot{
		Attempts24h:    alertRecord.Attempts24h,
		Delivered24h:   alertRecord.Delivered24h,
		Failed24h:      alertRecord.Failed24h,
		RetryableCount: alertRecord.RetryableCount,
		LastFailureAt:  copyTimePtr(alertRecord.LastFailureAt),
	}

	trackingRecord, err := r.store.ReadWalletTrackingOverview(ctx)
	if err != nil {
		return AdminObservabilitySnapshot{}, translateAdminConsoleError(err)
	}
	tracking := AdminWalletTrackingSnapshot{
		CandidateCount:  trackingRecord.CandidateCount,
		TrackedCount:    trackingRecord.TrackedCount,
		LabeledCount:    trackingRecord.LabeledCount,
		ScoredCount:     trackingRecord.ScoredCount,
		StaleCount:      trackingRecord.StaleCount,
		SuppressedCount: trackingRecord.SuppressedCount,
	}

	subscriptionRecord, err := r.store.ReadWalletTrackingSubscriptionOverview(ctx)
	if err != nil {
		return AdminObservabilitySnapshot{}, translateAdminConsoleError(err)
	}
	subscriptions := AdminWalletTrackingSubscriptionSnapshot{
		PendingCount: subscriptionRecord.PendingCount,
		ActiveCount:  subscriptionRecord.ActiveCount,
		ErroredCount: subscriptionRecord.ErroredCount,
		PausedCount:  subscriptionRecord.PausedCount,
		LastEventAt:  copyTimePtr(subscriptionRecord.LastEventAt),
	}

	queueDepth := AdminQueueDepthSnapshot{
		DefaultDepth:  0,
		PriorityDepth: 0,
	}
	if r.queueStats != nil {
		queueRecord, err := r.queueStats.ReadWalletBackfillQueueDepth(ctx)
		if err != nil {
			return AdminObservabilitySnapshot{}, translateAdminConsoleError(err)
		}
		queueDepth.DefaultDepth = queueRecord.DefaultDepth
		queueDepth.PriorityDepth = queueRecord.PriorityDepth
	}

	backfillRecord, err := r.store.ReadBackfillHealth(ctx, 24*time.Hour)
	if err != nil {
		return AdminObservabilitySnapshot{}, translateAdminConsoleError(err)
	}
	backfillHealth := AdminBackfillHealthSnapshot{
		Jobs24h:         backfillRecord.Jobs24h,
		Activities24h:   backfillRecord.Activities24h,
		Transactions24h: backfillRecord.Transactions24h,
		Expansions24h:   backfillRecord.Expansions24h,
		LastSuccessAt:   copyTimePtr(backfillRecord.LastSuccessAt),
	}

	staleRefreshRecord, err := r.store.ReadStaleRefreshHealth(ctx, 24*time.Hour)
	if err != nil {
		return AdminObservabilitySnapshot{}, translateAdminConsoleError(err)
	}
	staleRefresh := AdminStaleRefreshSnapshot{
		Attempts24h:   staleRefreshRecord.Attempts24h,
		Succeeded24h:  staleRefreshRecord.Succeeded24h,
		Productive24h: staleRefreshRecord.Productive24h,
		LastHitAt:     copyTimePtr(staleRefreshRecord.LastHitAt),
	}

	runRows, err := r.store.ListRecentJobHealth(ctx, 5)
	if err != nil {
		return AdminObservabilitySnapshot{}, translateAdminConsoleError(err)
	}
	recentRuns := make([]AdminJobHealthSnapshot, 0, len(runRows))
	for _, item := range runRows {
		recentRuns = append(recentRuns, AdminJobHealthSnapshot{
			JobName:             item.JobName,
			LastStatus:          item.LastStatus,
			LastStartedAt:       item.LastStartedAt,
			LastFinishedAt:      copyTimePtr(item.LastFinishedAt),
			LastSuccessAt:       copyTimePtr(item.LastSuccessAt),
			MinutesSinceSuccess: ageMinutes(item.LastSuccessAt, now),
			LastError:           strings.TrimSpace(item.LastError),
		})
	}

	failureRows, err := r.store.ListRecentFailures(ctx, 6)
	if err != nil {
		return AdminObservabilitySnapshot{}, translateAdminConsoleError(err)
	}
	recentFailures := make([]AdminFailureSnapshot, 0, len(failureRows))
	for _, item := range failureRows {
		recentFailures = append(recentFailures, AdminFailureSnapshot{
			Source:     item.Source,
			Kind:       item.Kind,
			OccurredAt: item.OccurredAt,
			Summary:    item.Summary,
			Details:    cloneFailureDetails(item.Details),
		})
	}

	return AdminObservabilitySnapshot{
		ProviderUsage:         providerUsage,
		Ingest:                ingest,
		AlertDelivery:         alertDelivery,
		WalletTracking:        tracking,
		TrackingSubscriptions: subscriptions,
		QueueDepth:            queueDepth,
		BackfillHealth:        backfillHealth,
		StaleRefresh:          staleRefresh,
		RecentRuns:            recentRuns,
		RecentFailures:        recentFailures,
	}, nil
}

func (r *PostgresAdminConsoleRepository) ListDomesticPrelistingCandidates(
	ctx context.Context,
	limit int,
) ([]AdminDomesticPrelistingCandidate, error) {
	if r == nil || r.domestic == nil {
		return []AdminDomesticPrelistingCandidate{}, nil
	}
	items, err := r.domestic.ListDomesticPrelistingCandidates(
		ctx,
		time.Now().UTC().Add(-7*24*time.Hour),
		time.Now().UTC().Add(-24*time.Hour),
		limit,
	)
	if err != nil {
		return nil, translateAdminConsoleError(err)
	}
	result := make([]AdminDomesticPrelistingCandidate, 0, len(items))
	for _, item := range items {
		result = append(result, AdminDomesticPrelistingCandidate{
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
			LatestObservedAt:          item.LatestObservedAt,
			ListedOnUpbit:             item.ListedOnUpbit,
			ListedOnBithumb:           item.ListedOnBithumb,
		})
	}
	return result, nil
}

func translateAdminConsoleError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, db.ErrAdminLabelNotFound):
		return ErrAdminLabelNotFound
	case errors.Is(err, db.ErrSuppressionRuleNotFound):
		return ErrAdminSuppressionNotFound
	case errors.Is(err, db.ErrWatchlistNotFound), errors.Is(err, db.ErrWatchlistItemNotFound):
		return ErrAdminCuratedListNotFound
	default:
		return err
	}
}

func classifyProviderUsageStatus(item db.AdminProviderUsageStatRecord, now time.Time) string {
	if item.LastSeenAt == nil {
		return "unavailable"
	}
	if item.Used24h > 0 && item.Error24h >= item.Used24h {
		return "critical"
	}
	if now.Sub(item.LastSeenAt.UTC()) > 4*time.Hour {
		return "warning"
	}
	if item.Error24h > 0 {
		return "warning"
	}
	return "healthy"
}

func classifyIngestLagStatus(latest *time.Time, now time.Time) string {
	if latest == nil {
		return "unavailable"
	}
	age := now.Sub(latest.UTC())
	switch {
	case age <= 30*time.Minute:
		return "healthy"
	case age <= 2*time.Hour:
		return "warning"
	default:
		return "critical"
	}
}

func copyTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	next := value.UTC()
	return &next
}

func maxTimePtr(values ...*time.Time) *time.Time {
	var best *time.Time
	for _, value := range values {
		if value == nil {
			continue
		}
		if best == nil || value.UTC().After(best.UTC()) {
			best = copyTimePtr(value)
		}
	}
	return best
}

func ageSeconds(value *time.Time, now time.Time) int {
	if value == nil {
		return 0
	}
	return int(now.Sub(value.UTC()).Seconds())
}

func ageMinutes(value *time.Time, now time.Time) int {
	if value == nil {
		return 0
	}
	return int(now.Sub(value.UTC()).Minutes())
}

func cloneFailureDetails(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func (r *PostgresAdminConsoleRepository) findCuratedList(ctx context.Context, id string) (AdminCuratedList, error) {
	items, err := r.ListCuratedLists(ctx)
	if err != nil {
		return AdminCuratedList{}, err
	}
	for _, item := range items {
		if item.ID == strings.TrimSpace(id) {
			return item, nil
		}
	}
	return AdminCuratedList{}, ErrAdminCuratedListNotFound
}

func toAdminCuratedList(item domain.Watchlist) AdminCuratedList {
	result := AdminCuratedList{
		ID:        item.ID,
		Name:      item.Name,
		Notes:     item.Notes,
		Tags:      append([]string(nil), item.Tags...),
		ItemCount: item.ItemCount,
		Items:     make([]AdminCuratedListItem, 0, len(item.Items)),
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
	for _, watchlistItem := range item.Items {
		result.Items = append(result.Items, AdminCuratedListItem{
			ID:        watchlistItem.ID,
			ItemType:  string(watchlistItem.ItemType),
			ItemKey:   watchlistItem.ItemKey,
			Tags:      append([]string(nil), watchlistItem.Tags...),
			Notes:     watchlistItem.Notes,
			CreatedAt: watchlistItem.CreatedAt,
			UpdatedAt: watchlistItem.UpdatedAt,
		})
	}
	return result
}

func stringValueFromMap(input map[string]any, key string) string {
	value, ok := input[key]
	if !ok {
		return ""
	}
	stringValue, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringValue)
}
