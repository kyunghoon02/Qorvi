package domain

import "fmt"

type Chain string

const (
	ChainEVM    Chain = "evm"
	ChainSolana Chain = "solana"
)

func IsSupportedChain(chain Chain) bool {
	return chain == ChainEVM || chain == ChainSolana
}

type Role string

const (
	RoleAnonymous Role = "anonymous"
	RoleUser      Role = "user"
	RoleAdmin     Role = "admin"
	RoleOperator  Role = "operator"
)

type PlanTier string

const (
	PlanFree PlanTier = "free"
	PlanPro  PlanTier = "pro"
	PlanTeam PlanTier = "team"
)

type AccessContext struct {
	Role Role
	Plan PlanTier
}

type EvidenceKind string

const (
	EvidenceTransfer       EvidenceKind = "transfer"
	EvidenceClusterOverlap EvidenceKind = "cluster_overlap"
	EvidenceBridge         EvidenceKind = "bridge"
	EvidenceCEXProximity   EvidenceKind = "cex_proximity"
	EvidenceLabel          EvidenceKind = "label"
)

type Evidence struct {
	Kind       EvidenceKind   `json:"kind"`
	Label      string         `json:"label"`
	Source     string         `json:"source"`
	Confidence float64        `json:"confidence"`
	ObservedAt string         `json:"observed_at"`
	Metadata   map[string]any `json:"metadata"`
}

type ScoreName string

const (
	ScoreCluster    ScoreName = "cluster_score"
	ScoreShadowExit ScoreName = "shadow_exit_risk"
	ScoreAlpha      ScoreName = "alpha_score"
)

type ScoreRating string

const (
	RatingLow    ScoreRating = "low"
	RatingMedium ScoreRating = "medium"
	RatingHigh   ScoreRating = "high"
)

type Score struct {
	Name     ScoreName   `json:"name"`
	Value    int         `json:"value"`
	Rating   ScoreRating `json:"rating"`
	Evidence []Evidence  `json:"evidence"`
}

type FreshnessSource string

const (
	FreshnessCache    FreshnessSource = "cache"
	FreshnessLive     FreshnessSource = "live"
	FreshnessSnapshot FreshnessSource = "snapshot"
)

type FreshnessMeta struct {
	GeneratedAt   string          `json:"generated_at"`
	Source        FreshnessSource `json:"source"`
	MaxAgeSeconds int             `json:"max_age_seconds"`
}

type PaginationMeta struct {
	NextCursor *string `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

type APIError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type ResponseMeta struct {
	RequestID  string          `json:"request_id"`
	Timestamp  string          `json:"timestamp"`
	Chain      Chain           `json:"chain,omitempty"`
	Tier       PlanTier        `json:"tier,omitempty"`
	Freshness  FreshnessMeta   `json:"freshness"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

type ResponseEnvelope[T any] struct {
	Success bool         `json:"success"`
	Data    *T           `json:"data"`
	Error   *APIError    `json:"error"`
	Meta    ResponseMeta `json:"meta"`
}

type WalletSummary struct {
	Chain            Chain    `json:"chain"`
	Address          string   `json:"address"`
	DisplayName      string   `json:"display_name"`
	ClusterID        *string  `json:"cluster_id"`
	Counterparties   int      `json:"counterparties"`
	LatestActivityAt string   `json:"latest_activity_at"`
	Tags             []string `json:"tags"`
	Scores           []Score  `json:"scores"`
}

func HasAnyRole(context AccessContext, allowedRoles ...Role) bool {
	for _, allowedRole := range allowedRoles {
		if context.Role == allowedRole {
			return true
		}
	}

	return false
}

func CreateWalletSummaryFixture(chain Chain, address string) WalletSummary {
	clusterID := "cluster_seed_whales"

	return WalletSummary{
		Chain:            chain,
		Address:          address,
		DisplayName:      "Seed Whale",
		ClusterID:        &clusterID,
		Counterparties:   18,
		LatestActivityAt: "2026-03-19T00:00:00Z",
		Tags:             []string{"seed", "watchlist"},
		Scores: []Score{
			{
				Name:   ScoreCluster,
				Value:  82,
				Rating: RatingHigh,
				Evidence: []Evidence{
					{
						Kind:       EvidenceClusterOverlap,
						Label:      "공통 counterparties 6건",
						Source:     "cluster-engine",
						Confidence: 0.88,
						ObservedAt: "2026-03-19T00:00:00Z",
						Metadata: map[string]any{
							"overlapping_wallets": 6,
						},
					},
				},
			},
			{
				Name:   ScoreShadowExit,
				Value:  34,
				Rating: RatingMedium,
				Evidence: []Evidence{
					{
						Kind:       EvidenceBridge,
						Label:      "브리지 이동 1건",
						Source:     "shadow-exit-engine",
						Confidence: 0.58,
						ObservedAt: "2026-03-19T00:00:00Z",
						Metadata: map[string]any{
							"bridge_transfers": 1,
						},
					},
				},
			},
		},
	}
}

func ValidateWalletSummary(summary WalletSummary) error {
	if summary.Address == "" {
		return fmt.Errorf("address is required")
	}

	if len(summary.Scores) == 0 {
		return fmt.Errorf("at least one score is required")
	}

	for _, score := range summary.Scores {
		if len(score.Evidence) == 0 {
			return fmt.Errorf("score %s must include evidence", score.Name)
		}
	}

	return nil
}
