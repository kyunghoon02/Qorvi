package domain

type FindingSubjectType string

const (
	FindingSubjectWallet  FindingSubjectType = "wallet"
	FindingSubjectEntity  FindingSubjectType = "entity"
	FindingSubjectCluster FindingSubjectType = "cluster"
	FindingSubjectToken   FindingSubjectType = "token"
)

type FindingType string

const (
	FindingTypeSuspectedMMHandoff      FindingType = "suspected_mm_handoff"
	FindingTypeTreasuryRedistribution  FindingType = "treasury_redistribution"
	FindingTypeCrossChainRotation      FindingType = "cross_chain_rotation"
	FindingTypeCoordinatedAccumulation FindingType = "coordinated_accumulation"
	FindingTypeExitPreparation         FindingType = "exit_preparation"
	FindingTypeCEXDepositPressure      FindingType = "cex_deposit_pressure"
	FindingTypeSmartMoneyConvergence   FindingType = "smart_money_convergence"
	FindingTypeFundAdjacentActivity    FindingType = "fund_adjacent_activity"
	FindingTypeHighConvictionEntry     FindingType = "high_conviction_entry"
)

type FindingSubject struct {
	SubjectType FindingSubjectType `json:"subject_type"`
	Chain       Chain              `json:"chain,omitempty"`
	Address     string             `json:"address,omitempty"`
	Key         string             `json:"key,omitempty"`
	Label       string             `json:"label,omitempty"`
}

type FindingCoverage struct {
	CoverageStartAt    string `json:"coverage_start_at,omitempty"`
	CoverageEndAt      string `json:"coverage_end_at,omitempty"`
	CoverageWindowDays int    `json:"coverage_window_days,omitempty"`
}

type FindingEvidenceItem struct {
	Type       string         `json:"type"`
	Value      string         `json:"value,omitempty"`
	Confidence float64        `json:"confidence,omitempty"`
	ObservedAt string         `json:"observed_at,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type NextWatchTarget struct {
	SubjectType FindingSubjectType `json:"subject_type"`
	Chain       Chain              `json:"chain,omitempty"`
	Address     string             `json:"address,omitempty"`
	Token       string             `json:"token,omitempty"`
	Label       string             `json:"label,omitempty"`
	Metadata    map[string]any     `json:"metadata,omitempty"`
}

type Finding struct {
	ID                     string                `json:"id"`
	Type                   FindingType           `json:"finding_type"`
	Subject                FindingSubject        `json:"subject"`
	Confidence             float64               `json:"confidence"`
	ImportanceScore        float64               `json:"importance_score"`
	Summary                string                `json:"summary"`
	ImportanceReason       []string              `json:"importance_reason,omitempty"`
	ObservedFacts          []string              `json:"observed_facts,omitempty"`
	InferredInterpretation []string              `json:"inferred_interpretations,omitempty"`
	Evidence               []FindingEvidenceItem `json:"evidence"`
	NextWatch              []NextWatchTarget     `json:"next_watch,omitempty"`
	DedupKey               string                `json:"dedup_key,omitempty"`
	ObservedAt             string                `json:"observed_at"`
	Coverage               FindingCoverage       `json:"coverage"`
}

type FindingsFeedPage struct {
	Items      []Finding `json:"items"`
	NextCursor *string   `json:"next_cursor,omitempty"`
	HasMore    bool      `json:"has_more"`
}
