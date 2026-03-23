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

type WalletCounterparty struct {
	Chain            Chain                            `json:"chain"`
	Address          string                           `json:"address"`
	EntityKey        string                           `json:"entity_key"`
	EntityType       string                           `json:"entity_type"`
	EntityLabel      string                           `json:"entity_label"`
	InteractionCount int                              `json:"interaction_count"`
	InboundCount     int                              `json:"inbound_count"`
	OutboundCount    int                              `json:"outbound_count"`
	InboundAmount    string                           `json:"inbound_amount"`
	OutboundAmount   string                           `json:"outbound_amount"`
	PrimaryToken     string                           `json:"primary_token"`
	TokenBreakdowns  []WalletCounterpartyTokenSummary `json:"token_breakdowns"`
	DirectionLabel   string                           `json:"direction_label"`
	FirstSeenAt      string                           `json:"first_seen_at"`
	LatestActivityAt string                           `json:"latest_activity_at"`
}

type WalletCounterpartyTokenSummary struct {
	Symbol         string `json:"symbol"`
	InboundAmount  string `json:"inbound_amount"`
	OutboundAmount string `json:"outbound_amount"`
}

type WalletRecentFlow struct {
	IncomingTxCount7d  int    `json:"incoming_tx_count_7d"`
	OutgoingTxCount7d  int    `json:"outgoing_tx_count_7d"`
	IncomingTxCount30d int    `json:"incoming_tx_count_30d"`
	OutgoingTxCount30d int    `json:"outgoing_tx_count_30d"`
	NetDirection7d     string `json:"net_direction_7d"`
	NetDirection30d    string `json:"net_direction_30d"`
}

type WalletEnrichment struct {
	Provider               string          `json:"provider"`
	NetWorthUSD            string          `json:"net_worth_usd"`
	NativeBalance          string          `json:"native_balance"`
	NativeBalanceFormatted string          `json:"native_balance_formatted"`
	ActiveChains           []string        `json:"active_chains"`
	ActiveChainCount       int             `json:"active_chain_count"`
	Holdings               []WalletHolding `json:"holdings"`
	HoldingCount           int             `json:"holding_count"`
	Source                 string          `json:"source"`
	UpdatedAt              string          `json:"updated_at"`
}

type WalletHolding struct {
	Symbol              string  `json:"symbol"`
	TokenAddress        string  `json:"token_address"`
	Balance             string  `json:"balance"`
	BalanceFormatted    string  `json:"balance_formatted"`
	ValueUSD            string  `json:"value_usd"`
	PortfolioPercentage float64 `json:"portfolio_percentage"`
	IsNative            bool    `json:"is_native"`
}

type WalletIndexingState struct {
	Status             string `json:"status"`
	LastIndexedAt      string `json:"last_indexed_at"`
	CoverageStartAt    string `json:"coverage_start_at"`
	CoverageEndAt      string `json:"coverage_end_at"`
	CoverageWindowDays int    `json:"coverage_window_days"`
}

type WalletLatestSignal struct {
	Name       ScoreName   `json:"name"`
	Value      int         `json:"value"`
	Rating     ScoreRating `json:"rating"`
	Label      string      `json:"label"`
	Source     string      `json:"source"`
	ObservedAt string      `json:"observed_at"`
}

type WalletSummary struct {
	Chain             Chain                `json:"chain"`
	Address           string               `json:"address"`
	DisplayName       string               `json:"display_name"`
	ClusterID         *string              `json:"cluster_id"`
	Counterparties    int                  `json:"counterparties"`
	LatestActivityAt  string               `json:"latest_activity_at"`
	TopCounterparties []WalletCounterparty `json:"top_counterparties"`
	RecentFlow        WalletRecentFlow     `json:"recent_flow"`
	Enrichment        *WalletEnrichment    `json:"enrichment,omitempty"`
	Indexing          WalletIndexingState  `json:"indexing"`
	LatestSignals     []WalletLatestSignal `json:"latest_signals"`
	Tags              []string             `json:"tags"`
	Scores            []Score              `json:"scores"`
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
		TopCounterparties: []WalletCounterparty{
			{
				Chain:            chain,
				Address:          "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				InteractionCount: 9,
				InboundCount:     2,
				OutboundCount:    7,
				InboundAmount:    "24.100000",
				OutboundAmount:   "214.550000",
				PrimaryToken:     "WETH",
				TokenBreakdowns: []WalletCounterpartyTokenSummary{
					{
						Symbol:         "WETH",
						InboundAmount:  "24.100000",
						OutboundAmount: "214.550000",
					},
				},
				DirectionLabel:   "outbound",
				FirstSeenAt:      "2026-03-10T00:00:00Z",
				LatestActivityAt: "2026-03-19T00:00:00Z",
			},
		},
		RecentFlow: WalletRecentFlow{
			IncomingTxCount7d:  4,
			OutgoingTxCount7d:  9,
			IncomingTxCount30d: 13,
			OutgoingTxCount30d: 29,
			NetDirection7d:     "outbound",
			NetDirection30d:    "outbound",
		},
		Enrichment: &WalletEnrichment{
			Provider:               "moralis",
			NetWorthUSD:            "157.00",
			NativeBalance:          "0.00402",
			NativeBalanceFormatted: "0.00402 ETH",
			ActiveChains: []string{
				"Ethereum",
				"Base",
				"Arbitrum",
				"Optimism",
				"Polygon",
				"Blast",
			},
			ActiveChainCount: 6,
			Holdings: []WalletHolding{
				{
					Symbol:              "USDC",
					TokenAddress:        "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Balance:             "149.20",
					BalanceFormatted:    "149.20",
					ValueUSD:            "149.20",
					PortfolioPercentage: 94.8,
					IsNative:            false,
				},
				{
					Symbol:              "WETH",
					TokenAddress:        "0xC02aaA39b223FE8D0A0E5C4F27eAD9083C756Cc2",
					Balance:             "0.00402",
					BalanceFormatted:    "0.00402",
					ValueUSD:            "8.14",
					PortfolioPercentage: 5.2,
					IsNative:            false,
				},
			},
			HoldingCount: 2,
			Source:       "fallback",
			UpdatedAt:    "2026-03-21T00:00:00Z",
		},
		Indexing: WalletIndexingState{
			Status:             "ready",
			LastIndexedAt:      "2026-03-21T00:00:00Z",
			CoverageStartAt:    "2026-03-10T00:00:00Z",
			CoverageEndAt:      "2026-03-19T00:00:00Z",
			CoverageWindowDays: 10,
		},
		LatestSignals: []WalletLatestSignal{
			{
				Name:       ScoreCluster,
				Value:      82,
				Rating:     RatingHigh,
				Label:      "공통 counterparties 6건",
				Source:     "cluster-engine",
				ObservedAt: "2026-03-19T00:00:00Z",
			},
			{
				Name:       ScoreShadowExit,
				Value:      34,
				Rating:     RatingMedium,
				Label:      "브리지 이동 1건",
				Source:     "shadow-exit-engine",
				ObservedAt: "2026-03-19T00:00:00Z",
			},
		},
		Tags: []string{"seed", "watchlist"},
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
