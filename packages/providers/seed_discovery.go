package providers

import (
	"fmt"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

type SeedDiscoveryBatch struct {
	Provider    ProviderName
	Request     ProviderRequestContext
	WindowStart time.Time
	WindowEnd   time.Time
	Limit       int
	Metadata    map[string]any
}

type SeedDiscoveryCandidate struct {
	Provider      ProviderName
	Chain         domain.Chain
	WalletAddress string
	SourceID      string
	ObservedAt    time.Time
	Kind          string
	Confidence    float64
	Metadata      map[string]any
}

type SeedDiscoveryCandidateInput struct {
	Provider      ProviderName
	Chain         domain.Chain
	WalletAddress string
	SourceID      string
	Kind          string
	Confidence    float64
	ObservedAt    time.Time
	Metadata      map[string]any
}

type SeedDiscoveryAdapter interface {
	ProviderAdapter
	FetchSeedDiscoveryCandidates(batch SeedDiscoveryBatch) ([]SeedDiscoveryCandidate, error)
}

type SeedDiscoveryResult struct {
	Batch      SeedDiscoveryBatch
	Candidates []SeedDiscoveryCandidate
}

type SeedDiscoveryRunner struct {
	Registry Registry
}

func NewSeedDiscoveryRunner(registry Registry) SeedDiscoveryRunner {
	return SeedDiscoveryRunner{Registry: registry}
}

func (b SeedDiscoveryBatch) Validate() error {
	if b.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if !domain.IsSupportedChain(b.Request.Chain) {
		return fmt.Errorf("unsupported chain %q", b.Request.Chain)
	}
	if strings.TrimSpace(b.Request.WalletAddress) == "" {
		return fmt.Errorf("wallet address is required")
	}
	if b.WindowStart.IsZero() || b.WindowEnd.IsZero() {
		return fmt.Errorf("window start and end are required")
	}
	if b.WindowEnd.Before(b.WindowStart) {
		return fmt.Errorf("window end must be after window start")
	}
	if b.Limit <= 0 {
		return fmt.Errorf("limit must be positive")
	}

	return nil
}

func (r SeedDiscoveryRunner) Run(batch SeedDiscoveryBatch) (SeedDiscoveryResult, error) {
	if err := batch.Validate(); err != nil {
		return SeedDiscoveryResult{}, err
	}

	adapter, ok := r.Registry[batch.Provider]
	if !ok {
		return SeedDiscoveryResult{}, fmt.Errorf("provider %q is not registered", batch.Provider)
	}

	seedAdapter, ok := adapter.(SeedDiscoveryAdapter)
	if !ok {
		return SeedDiscoveryResult{}, fmt.Errorf("provider %q does not support seed discovery", batch.Provider)
	}

	candidates, err := seedAdapter.FetchSeedDiscoveryCandidates(batch)
	if err != nil {
		return SeedDiscoveryResult{}, err
	}

	return SeedDiscoveryResult{
		Batch:      batch,
		Candidates: append([]SeedDiscoveryCandidate(nil), candidates...),
	}, nil
}

func CreateSeedDiscoveryBatchFixture(provider ProviderName, chain domain.Chain, walletAddress string) SeedDiscoveryBatch {
	return SeedDiscoveryBatch{
		Provider: provider,
		Request: ProviderRequestContext{
			Chain:         chain,
			WalletAddress: walletAddress,
			Access: domain.AccessContext{
				Role: domain.RoleOperator,
				Plan: domain.PlanPro,
			},
		},
		WindowStart: fixedObservedAt.Add(-24 * time.Hour),
		WindowEnd:   fixedObservedAt,
		Limit:       100,
	}
}

func CreateSeedDiscoveryCandidateFixture(batch SeedDiscoveryBatch, input SeedDiscoveryCandidateInput) SeedDiscoveryCandidate {
	metadata := map[string]any{}
	for key, value := range batch.Metadata {
		metadata[key] = value
	}
	for key, value := range input.Metadata {
		metadata[key] = value
	}

	metadata["seed_discovery_provider"] = string(batch.Provider)
	metadata["seed_discovery_window_start"] = batch.WindowStart.Format(time.RFC3339)
	metadata["seed_discovery_window_end"] = batch.WindowEnd.Format(time.RFC3339)
	metadata["seed_discovery_limit"] = batch.Limit

	observedAt := input.ObservedAt
	if observedAt.IsZero() {
		observedAt = fixedObservedAt
	}
	kind := strings.TrimSpace(input.Kind)
	if kind == "" {
		kind = "seed_candidate"
	}
	sourceID := strings.TrimSpace(input.SourceID)
	if sourceID == "" {
		sourceID = fmt.Sprintf("%s_seed_discovery_v0", batch.Provider)
	}

	return SeedDiscoveryCandidate{
		Provider:      input.Provider,
		Chain:         input.Chain,
		WalletAddress: strings.TrimSpace(input.WalletAddress),
		SourceID:      sourceID,
		ObservedAt:    observedAt,
		Kind:          kind,
		Confidence:    input.Confidence,
		Metadata:      metadata,
	}
}
