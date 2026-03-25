package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/flowintel/flowintel/packages/ops"
)

var (
	ErrAdminLabelNotFound       = errors.New("admin label not found")
	ErrAdminLabelAlreadyExists  = errors.New("admin label already exists")
	ErrSuppressionRuleNotFound  = errors.New("suppression rule not found")
	ErrSuppressionRuleDuplicate = errors.New("suppression rule already exists")
)

type AdminLabelRecord struct {
	ID          string
	Name        string
	Description string
	Color       string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AdminSuppressionRecord struct {
	ID        string
	Rule      ops.SuppressionRule
	UpdatedAt time.Time
}

type AdminProviderUsageStatRecord struct {
	Provider     string
	Used24h      int
	Error24h     int
	AvgLatencyMs int
	LastSeenAt   *time.Time
}

type AdminIngestFreshnessRecord struct {
	LastBackfillAt *time.Time
	LastWebhookAt  *time.Time
}

type AdminAlertDeliveryHealthRecord struct {
	Attempts24h    int
	Delivered24h   int
	Failed24h      int
	RetryableCount int
	LastFailureAt  *time.Time
}

type AdminWalletTrackingOverviewRecord struct {
	CandidateCount  int
	TrackedCount    int
	LabeledCount    int
	ScoredCount     int
	StaleCount      int
	SuppressedCount int
}

type AdminWalletTrackingSubscriptionOverviewRecord struct {
	PendingCount int
	ActiveCount  int
	ErroredCount int
	PausedCount  int
	LastEventAt  *time.Time
}

type AdminJobHealthRecord struct {
	JobName        string
	LastStatus     string
	LastStartedAt  time.Time
	LastFinishedAt *time.Time
	LastSuccessAt  *time.Time
	LastError      string
}

type AdminFailureRecord struct {
	Source     string
	Kind       string
	OccurredAt time.Time
	Summary    string
	Details    map[string]any
}

type PostgresAdminConsoleStore struct {
	Querier postgresQuerier
	Execer  postgresTransactionExecer
	Now     func() time.Time
}

const listAdminLabelsSQL = `
SELECT id, name, description, color, created_by, created_at, updated_at
FROM admin_labels
ORDER BY updated_at DESC, created_at DESC, name ASC
`

const upsertAdminLabelSQL = `
INSERT INTO admin_labels (
  name,
  description,
  color,
  created_by,
  updated_at
) VALUES ($1, $2, $3, $4, now())
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    color = EXCLUDED.color,
    created_by = EXCLUDED.created_by,
    updated_at = now()
RETURNING id, name, description, color, created_by, created_at, updated_at
`

const deleteAdminLabelSQL = `
DELETE FROM admin_labels
WHERE name = $1
`

const listAdminSuppressionsSQL = `
SELECT id, suppression_type, target, reason, created_by, created_at, expires_at, active, updated_at
FROM suppressions
ORDER BY active DESC, updated_at DESC, created_at DESC, id DESC
`

const createSuppressionSQL = `
INSERT INTO suppressions (
  suppression_key,
  suppression_type,
  target,
  reason,
  created_by,
  active,
  created_at,
  updated_at,
  expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, suppression_type, target, reason, created_by, created_at, expires_at, active, updated_at
`

const deactivateSuppressionSQL = `
UPDATE suppressions
SET active = false,
    updated_at = now()
WHERE id = $1
RETURNING id, suppression_type, target, reason, created_by, created_at, expires_at, active, updated_at
`

const listProviderQuotaUsageSQL = `
SELECT provider, COUNT(*)::int AS used, COALESCE(MAX(created_at), $2) AS last_checked_at
FROM provider_usage_logs
WHERE created_at >= $1
GROUP BY provider
ORDER BY provider ASC
`

const listAdminProviderUsageStatsSQL = `
SELECT
  provider,
  COUNT(*)::int AS used_24h,
  COUNT(*) FILTER (WHERE status_code >= 400)::int AS error_24h,
  COALESCE(AVG(latency_ms)::int, 0) AS avg_latency_ms,
  MAX(created_at) AS last_seen_at
FROM provider_usage_logs
WHERE created_at >= $1
GROUP BY provider
ORDER BY provider ASC
`

const readAdminIngestFreshnessSQL = `
SELECT
  MAX(
    CASE
      WHEN job_name IN ('historical-backfill-ingest', 'wallet-backfill-drain', 'wallet-backfill-drain-batch')
       AND status = 'succeeded'
      THEN COALESCE(finished_at, started_at)
    END
  ) AS last_backfill_at,
  MAX(
    CASE
      WHEN job_name IN ('alchemy-address-activity-webhook', 'helius-address-activity-webhook')
       AND status = 'succeeded'
      THEN COALESCE(finished_at, started_at)
    END
  ) AS last_webhook_at
FROM job_runs
`

const readAdminAlertDeliveryHealthSQL = `
SELECT
  COUNT(*)::int AS attempts_24h,
  COUNT(*) FILTER (WHERE status = 'delivered')::int AS delivered_24h,
  COUNT(*) FILTER (WHERE status = 'failed')::int AS failed_24h,
  (
    SELECT COUNT(*)::int
    FROM alert_delivery_attempts a
    JOIN alert_delivery_channels c
      ON c.id = a.channel_id
    WHERE a.status = 'failed'
      AND c.is_enabled = true
      AND COALESCE((a.details->>'retry_exhausted')::boolean, false) = false
  ) AS retryable_count,
  MAX(
    CASE
      WHEN status = 'failed'
      THEN COALESCE(failed_at, attempted_at, created_at)
    END
  ) AS last_failure_at
FROM alert_delivery_attempts
WHERE created_at >= $1
`

const readAdminWalletTrackingOverviewSQL = `
SELECT
  COUNT(*) FILTER (WHERE status = 'candidate')::int AS candidate_count,
  COUNT(*) FILTER (WHERE status = 'tracked')::int AS tracked_count,
  COUNT(*) FILTER (WHERE status = 'labeled')::int AS labeled_count,
  COUNT(*) FILTER (WHERE status = 'scored')::int AS scored_count,
  COUNT(*) FILTER (WHERE status = 'stale')::int AS stale_count,
  COUNT(*) FILTER (WHERE status = 'suppressed')::int AS suppressed_count
FROM wallet_tracking_state
`

const readAdminWalletTrackingSubscriptionOverviewSQL = `
SELECT
  COUNT(*) FILTER (WHERE status = 'pending')::int AS pending_count,
  COUNT(*) FILTER (WHERE status = 'active')::int AS active_count,
  COUNT(*) FILTER (WHERE status = 'errored')::int AS errored_count,
  COUNT(*) FILTER (WHERE status = 'paused')::int AS paused_count,
  MAX(last_event_at) AS last_event_at
FROM wallet_tracking_subscriptions
`

const listAdminRecentJobHealthSQL = `
WITH latest AS (
  SELECT DISTINCT ON (job_name)
    job_name,
    status,
    started_at,
    finished_at,
    details
  FROM job_runs
  ORDER BY job_name, started_at DESC
),
last_success AS (
  SELECT
    job_name,
    MAX(COALESCE(finished_at, started_at)) AS last_success_at
  FROM job_runs
  WHERE status = 'succeeded'
  GROUP BY job_name
)
SELECT
  latest.job_name,
  latest.status,
  latest.started_at,
  latest.finished_at,
  last_success.last_success_at,
  COALESCE(NULLIF(latest.details->>'error', ''), '') AS last_error
FROM latest
LEFT JOIN last_success
  ON last_success.job_name = latest.job_name
ORDER BY latest.started_at DESC
LIMIT $1
`

const listAdminRecentFailuresSQL = `
SELECT source, kind, occurred_at, summary, details
FROM (
  SELECT
    'worker'::text AS source,
    job_name AS kind,
    COALESCE(finished_at, started_at) AS occurred_at,
    COALESCE(NULLIF(details->>'error', ''), 'job failed') AS summary,
    details
  FROM job_runs
  WHERE status = 'failed'

  UNION ALL

  SELECT
    'alert_delivery'::text AS source,
    channel_type::text AS kind,
    COALESCE(failed_at, attempted_at, created_at) AS occurred_at,
    COALESCE(NULLIF(details->>'error', ''), 'alert delivery failed') AS summary,
    details
  FROM alert_delivery_attempts
  WHERE status = 'failed'

  UNION ALL

  SELECT
    'provider'::text AS source,
    provider AS kind,
    created_at AS occurred_at,
    operation || ' returned ' || status_code::text AS summary,
    jsonb_build_object(
      'operation', operation,
      'status_code', status_code,
      'latency_ms', latency_ms
    ) AS details
  FROM provider_usage_logs
  WHERE status_code >= 500
     OR status_code = 429
) failures
ORDER BY occurred_at DESC
LIMIT $1
`

func NewPostgresAdminConsoleStore(
	querier postgresQuerier,
	execer ...postgresTransactionExecer,
) *PostgresAdminConsoleStore {
	store := &PostgresAdminConsoleStore{
		Querier: querier,
		Now:     time.Now,
	}
	if len(execer) > 0 {
		store.Execer = execer[0]
		return store
	}
	if txExecer, ok := querier.(postgresTransactionExecer); ok {
		store.Execer = txExecer
	}
	return store
}

func NewPostgresAdminConsoleStoreFromPool(
	pool postgresQuerier,
) *PostgresAdminConsoleStore {
	return NewPostgresAdminConsoleStore(pool)
}

func (s *PostgresAdminConsoleStore) ListAdminLabels(
	ctx context.Context,
) ([]AdminLabelRecord, error) {
	if s == nil || s.Querier == nil {
		return []AdminLabelRecord{}, nil
	}
	rows, err := s.Querier.Query(ctx, listAdminLabelsSQL)
	if err != nil {
		return nil, fmt.Errorf("list admin labels: %w", err)
	}
	defer rows.Close()

	records := make([]AdminLabelRecord, 0)
	for rows.Next() {
		record, err := scanAdminLabelRow(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin labels: %w", err)
	}
	return records, nil
}

func (s *PostgresAdminConsoleStore) UpsertAdminLabel(
	ctx context.Context,
	label ops.Label,
	actor string,
) (AdminLabelRecord, error) {
	if s == nil || s.Querier == nil {
		return AdminLabelRecord{}, fmt.Errorf("admin console store is nil")
	}
	if err := ops.ValidateLabel(label); err != nil {
		return AdminLabelRecord{}, err
	}
	row := s.Querier.QueryRow(
		ctx,
		upsertAdminLabelSQL,
		ops.NormalizeLabelName(label.Name),
		strings.TrimSpace(label.Description),
		strings.TrimSpace(label.Color),
		strings.TrimSpace(actor),
	)
	record, err := scanAdminLabelRow(row)
	if err != nil {
		return AdminLabelRecord{}, err
	}
	return record, nil
}

func (s *PostgresAdminConsoleStore) DeleteAdminLabel(
	ctx context.Context,
	name string,
) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("admin console store is nil")
	}
	tag, err := s.Execer.Exec(ctx, deleteAdminLabelSQL, ops.NormalizeLabelName(name))
	if err != nil {
		return fmt.Errorf("delete admin label: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAdminLabelNotFound
	}
	return nil
}

func (s *PostgresAdminConsoleStore) ListSuppressions(
	ctx context.Context,
) ([]AdminSuppressionRecord, error) {
	if s == nil || s.Querier == nil {
		return []AdminSuppressionRecord{}, nil
	}
	rows, err := s.Querier.Query(ctx, listAdminSuppressionsSQL)
	if err != nil {
		return nil, fmt.Errorf("list suppressions: %w", err)
	}
	defer rows.Close()

	records := make([]AdminSuppressionRecord, 0)
	for rows.Next() {
		record, err := scanAdminSuppressionRow(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate suppressions: %w", err)
	}
	return records, nil
}

func (s *PostgresAdminConsoleStore) CreateSuppression(
	ctx context.Context,
	rule ops.SuppressionRule,
) (AdminSuppressionRecord, error) {
	if s == nil || s.Querier == nil {
		return AdminSuppressionRecord{}, fmt.Errorf("admin console store is nil")
	}
	if err := ops.ValidateSuppressionRule(rule); err != nil {
		return AdminSuppressionRecord{}, err
	}
	key := buildSuppressionKey(rule.Scope, rule.Target)
	row := s.Querier.QueryRow(
		ctx,
		createSuppressionSQL,
		key,
		string(rule.Scope),
		strings.TrimSpace(rule.Target),
		strings.TrimSpace(rule.Reason),
		strings.TrimSpace(rule.CreatedBy),
		rule.Active,
		rule.CreatedAt.UTC(),
		rule.CreatedAt.UTC(),
		rule.ExpiresAt,
	)
	record, err := scanAdminSuppressionRow(row)
	if err != nil {
		if isUniqueViolation(err) {
			return AdminSuppressionRecord{}, ErrSuppressionRuleDuplicate
		}
		return AdminSuppressionRecord{}, err
	}
	return record, nil
}

func (s *PostgresAdminConsoleStore) DeactivateSuppression(
	ctx context.Context,
	id string,
) (AdminSuppressionRecord, error) {
	if s == nil || s.Querier == nil {
		return AdminSuppressionRecord{}, fmt.Errorf("admin console store is nil")
	}
	record, err := scanAdminSuppressionRow(s.Querier.QueryRow(ctx, deactivateSuppressionSQL, strings.TrimSpace(id)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrSuppressionRuleNotFound) {
			return AdminSuppressionRecord{}, ErrSuppressionRuleNotFound
		}
		return AdminSuppressionRecord{}, err
	}
	return record, nil
}

func (s *PostgresAdminConsoleStore) ListProviderQuotaSnapshots(
	ctx context.Context,
	window time.Duration,
	limits map[ops.ProviderName]int,
) ([]ops.ProviderQuotaSnapshot, error) {
	if s == nil || s.Querier == nil {
		return []ops.ProviderQuotaSnapshot{}, nil
	}
	if window <= 0 {
		return nil, fmt.Errorf("quota window must be positive")
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	windowStart := now.Add(-window).UTC()
	rows, err := s.Querier.Query(ctx, listProviderQuotaUsageSQL, windowStart, now.UTC())
	if err != nil {
		return nil, fmt.Errorf("list provider quota usage: %w", err)
	}
	defer rows.Close()

	type usageRow struct {
		provider    string
		used        int
		lastChecked time.Time
	}
	usageByProvider := map[string]usageRow{}
	for rows.Next() {
		var row usageRow
		if err := rows.Scan(&row.provider, &row.used, &row.lastChecked); err != nil {
			return nil, fmt.Errorf("scan provider quota usage: %w", err)
		}
		usageByProvider[strings.TrimSpace(strings.ToLower(row.provider))] = row
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider quota usage: %w", err)
	}

	snapshots := make([]ops.ProviderQuotaSnapshot, 0, len(limits))
	for provider, limit := range limits {
		usage := usageByProvider[strings.TrimSpace(strings.ToLower(string(provider)))]
		lastCheckedAt := now.UTC()
		used := 0
		if usage.provider != "" {
			lastCheckedAt = usage.lastChecked.UTC()
			used = usage.used
		}
		snapshot := ops.ProviderQuotaSnapshot{
			Provider:      provider,
			WindowStart:   windowStart,
			WindowEnd:     now.UTC(),
			Limit:         limit,
			Used:          used,
			Reserved:      0,
			LastCheckedAt: lastCheckedAt,
		}
		if _, err := ops.ClassifyQuotaStatus(snapshot); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func (s *PostgresAdminConsoleStore) ListProviderUsageStats(
	ctx context.Context,
	window time.Duration,
) ([]AdminProviderUsageStatRecord, error) {
	if s == nil || s.Querier == nil {
		return []AdminProviderUsageStatRecord{}, nil
	}
	if window <= 0 {
		return nil, fmt.Errorf("provider usage window must be positive")
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	windowStart := now.Add(-window)
	rows, err := s.Querier.Query(ctx, listAdminProviderUsageStatsSQL, windowStart)
	if err != nil {
		return nil, fmt.Errorf("list provider usage stats: %w", err)
	}
	defer rows.Close()

	records := make([]AdminProviderUsageStatRecord, 0)
	for rows.Next() {
		var (
			record     AdminProviderUsageStatRecord
			lastSeenAt sql.NullTime
		)
		if err := rows.Scan(
			&record.Provider,
			&record.Used24h,
			&record.Error24h,
			&record.AvgLatencyMs,
			&lastSeenAt,
		); err != nil {
			return nil, fmt.Errorf("scan provider usage stats: %w", err)
		}
		if lastSeenAt.Valid {
			lastSeen := lastSeenAt.Time.UTC()
			record.LastSeenAt = &lastSeen
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider usage stats: %w", err)
	}
	return records, nil
}

func (s *PostgresAdminConsoleStore) ReadIngestFreshness(
	ctx context.Context,
) (AdminIngestFreshnessRecord, error) {
	if s == nil || s.Querier == nil {
		return AdminIngestFreshnessRecord{}, nil
	}
	var (
		record         AdminIngestFreshnessRecord
		lastBackfillAt sql.NullTime
		lastWebhookAt  sql.NullTime
	)
	if err := s.Querier.QueryRow(ctx, readAdminIngestFreshnessSQL).Scan(&lastBackfillAt, &lastWebhookAt); err != nil {
		return AdminIngestFreshnessRecord{}, fmt.Errorf("read ingest freshness: %w", err)
	}
	if lastBackfillAt.Valid {
		next := lastBackfillAt.Time.UTC()
		record.LastBackfillAt = &next
	}
	if lastWebhookAt.Valid {
		next := lastWebhookAt.Time.UTC()
		record.LastWebhookAt = &next
	}
	return record, nil
}

func (s *PostgresAdminConsoleStore) ReadAlertDeliveryHealth(
	ctx context.Context,
	window time.Duration,
) (AdminAlertDeliveryHealthRecord, error) {
	if s == nil || s.Querier == nil {
		return AdminAlertDeliveryHealthRecord{}, nil
	}
	if window <= 0 {
		return AdminAlertDeliveryHealthRecord{}, fmt.Errorf("alert delivery window must be positive")
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	windowStart := now.Add(-window)

	var (
		record        AdminAlertDeliveryHealthRecord
		lastFailureAt sql.NullTime
	)
	if err := s.Querier.QueryRow(ctx, readAdminAlertDeliveryHealthSQL, windowStart).Scan(
		&record.Attempts24h,
		&record.Delivered24h,
		&record.Failed24h,
		&record.RetryableCount,
		&lastFailureAt,
	); err != nil {
		return AdminAlertDeliveryHealthRecord{}, fmt.Errorf("read alert delivery health: %w", err)
	}
	if lastFailureAt.Valid {
		next := lastFailureAt.Time.UTC()
		record.LastFailureAt = &next
	}
	return record, nil
}

func (s *PostgresAdminConsoleStore) ReadWalletTrackingOverview(
	ctx context.Context,
) (AdminWalletTrackingOverviewRecord, error) {
	if s == nil || s.Querier == nil {
		return AdminWalletTrackingOverviewRecord{}, nil
	}

	var record AdminWalletTrackingOverviewRecord
	if err := s.Querier.QueryRow(ctx, readAdminWalletTrackingOverviewSQL).Scan(
		&record.CandidateCount,
		&record.TrackedCount,
		&record.LabeledCount,
		&record.ScoredCount,
		&record.StaleCount,
		&record.SuppressedCount,
	); err != nil {
		return AdminWalletTrackingOverviewRecord{}, fmt.Errorf("read wallet tracking overview: %w", err)
	}

	return record, nil
}

func (s *PostgresAdminConsoleStore) ReadWalletTrackingSubscriptionOverview(
	ctx context.Context,
) (AdminWalletTrackingSubscriptionOverviewRecord, error) {
	if s == nil || s.Querier == nil {
		return AdminWalletTrackingSubscriptionOverviewRecord{}, nil
	}

	var (
		record      AdminWalletTrackingSubscriptionOverviewRecord
		lastEventAt sql.NullTime
	)
	if err := s.Querier.QueryRow(ctx, readAdminWalletTrackingSubscriptionOverviewSQL).Scan(
		&record.PendingCount,
		&record.ActiveCount,
		&record.ErroredCount,
		&record.PausedCount,
		&lastEventAt,
	); err != nil {
		return AdminWalletTrackingSubscriptionOverviewRecord{}, fmt.Errorf("read wallet tracking subscription overview: %w", err)
	}
	if lastEventAt.Valid {
		next := lastEventAt.Time.UTC()
		record.LastEventAt = &next
	}

	return record, nil
}

func (s *PostgresAdminConsoleStore) ListRecentJobHealth(
	ctx context.Context,
	limit int,
) ([]AdminJobHealthRecord, error) {
	if s == nil || s.Querier == nil {
		return []AdminJobHealthRecord{}, nil
	}
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.Querier.Query(ctx, listAdminRecentJobHealthSQL, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent job health: %w", err)
	}
	defer rows.Close()

	records := make([]AdminJobHealthRecord, 0)
	for rows.Next() {
		var (
			record         AdminJobHealthRecord
			lastFinishedAt sql.NullTime
			lastSuccessAt  sql.NullTime
		)
		if err := rows.Scan(
			&record.JobName,
			&record.LastStatus,
			&record.LastStartedAt,
			&lastFinishedAt,
			&lastSuccessAt,
			&record.LastError,
		); err != nil {
			return nil, fmt.Errorf("scan recent job health: %w", err)
		}
		if lastFinishedAt.Valid {
			next := lastFinishedAt.Time.UTC()
			record.LastFinishedAt = &next
		}
		if lastSuccessAt.Valid {
			next := lastSuccessAt.Time.UTC()
			record.LastSuccessAt = &next
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent job health: %w", err)
	}
	return records, nil
}

func (s *PostgresAdminConsoleStore) ListRecentFailures(
	ctx context.Context,
	limit int,
) ([]AdminFailureRecord, error) {
	if s == nil || s.Querier == nil {
		return []AdminFailureRecord{}, nil
	}
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.Querier.Query(ctx, listAdminRecentFailuresSQL, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent failures: %w", err)
	}
	defer rows.Close()

	records := make([]AdminFailureRecord, 0)
	for rows.Next() {
		var (
			record     AdminFailureRecord
			detailsRaw []byte
		)
		if err := rows.Scan(
			&record.Source,
			&record.Kind,
			&record.OccurredAt,
			&record.Summary,
			&detailsRaw,
		); err != nil {
			return nil, fmt.Errorf("scan recent failure: %w", err)
		}
		if len(detailsRaw) > 0 {
			if err := json.Unmarshal(detailsRaw, &record.Details); err != nil {
				return nil, fmt.Errorf("decode recent failure details: %w", err)
			}
		} else {
			record.Details = map[string]any{}
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent failures: %w", err)
	}
	return records, nil
}

func scanAdminLabelRow(scanner interface{ Scan(...any) error }) (AdminLabelRecord, error) {
	var record AdminLabelRecord
	if err := scanner.Scan(
		&record.ID,
		&record.Name,
		&record.Description,
		&record.Color,
		&record.CreatedBy,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminLabelRecord{}, ErrAdminLabelNotFound
		}
		return AdminLabelRecord{}, fmt.Errorf("scan admin label: %w", err)
	}
	return record, nil
}

func scanAdminSuppressionRow(scanner interface{ Scan(...any) error }) (AdminSuppressionRecord, error) {
	var (
		record    AdminSuppressionRecord
		expiresAt sql.NullTime
		scope     string
	)
	if err := scanner.Scan(
		&record.ID,
		&scope,
		&record.Rule.Target,
		&record.Rule.Reason,
		&record.Rule.CreatedBy,
		&record.Rule.CreatedAt,
		&expiresAt,
		&record.Rule.Active,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminSuppressionRecord{}, ErrSuppressionRuleNotFound
		}
		return AdminSuppressionRecord{}, fmt.Errorf("scan suppression rule: %w", err)
	}
	record.Rule.Scope = ops.SuppressionScope(strings.TrimSpace(scope))
	record.Rule.ID = record.ID
	if expiresAt.Valid {
		value := expiresAt.Time.UTC()
		record.Rule.ExpiresAt = &value
	}
	return record, nil
}

func buildSuppressionKey(scope ops.SuppressionScope, target string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(string(scope)), strings.TrimSpace(strings.ToLower(target)))
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
