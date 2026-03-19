package ops

import "time"

type Label struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type SuppressionScope string

const (
	SuppressionScopeWallet   SuppressionScope = "wallet"
	SuppressionScopeCluster  SuppressionScope = "cluster"
	SuppressionScopeToken    SuppressionScope = "token"
	SuppressionScopeEntity   SuppressionScope = "entity"
	SuppressionScopeProvider SuppressionScope = "provider"
)

type SuppressionRule struct {
	ID        string           `json:"id"`
	Scope     SuppressionScope `json:"scope"`
	Target    string           `json:"target"`
	Reason    string           `json:"reason"`
	CreatedBy string           `json:"created_by"`
	CreatedAt time.Time        `json:"created_at"`
	ExpiresAt *time.Time       `json:"expires_at,omitempty"`
	Active    bool             `json:"active"`
}

type ProviderName string

const (
	ProviderDune    ProviderName = "dune"
	ProviderAlchemy ProviderName = "alchemy"
	ProviderHelius  ProviderName = "helius"
	ProviderMoralis ProviderName = "moralis"
)

type ProviderQuotaSnapshot struct {
	Provider      ProviderName `json:"provider"`
	WindowStart   time.Time    `json:"window_start"`
	WindowEnd     time.Time    `json:"window_end"`
	Limit         int          `json:"limit"`
	Used          int          `json:"used"`
	Reserved      int          `json:"reserved"`
	LastCheckedAt time.Time    `json:"last_checked_at"`
}

type QuotaStatus string

const (
	QuotaStatusHealthy   QuotaStatus = "healthy"
	QuotaStatusWarning   QuotaStatus = "warning"
	QuotaStatusCritical  QuotaStatus = "critical"
	QuotaStatusExhausted QuotaStatus = "exhausted"
)

type AuditAction string

const (
	AuditActionLabelUpsert       AuditAction = "label_upsert"
	AuditActionSuppressionAdd    AuditAction = "suppression_add"
	AuditActionSuppressionRemove AuditAction = "suppression_remove"
	AuditActionQuotaReview       AuditAction = "quota_review"
)

type AuditEvent struct {
	ID        string      `json:"id"`
	Action    AuditAction `json:"action"`
	Actor     string      `json:"actor"`
	Target    string      `json:"target"`
	Note      string      `json:"note"`
	CreatedAt time.Time   `json:"created_at"`
}
