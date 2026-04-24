package service

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/db"
)

type DiscoverFeaturedWallet struct {
	Chain       string   `json:"chain"`
	Address     string   `json:"address"`
	DisplayName string   `json:"displayName"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Provider    string   `json:"provider,omitempty"`
	Confidence  float64  `json:"confidence,omitempty"`
	ObservedAt  string   `json:"observedAt,omitempty"`
}

type DiscoverFeaturedWalletResponse struct {
	Items []DiscoverFeaturedWallet `json:"items"`
}

type DiscoverDomesticPrelistingCandidate struct {
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
	RepresentativeWalletChain string `json:"representativeWalletChain,omitempty"`
	RepresentativeWallet      string `json:"representativeWallet,omitempty"`
	RepresentativeLabel       string `json:"representativeLabel,omitempty"`
}

type DiscoverDomesticPrelistingResponse struct {
	Items []DiscoverDomesticPrelistingCandidate `json:"items"`
}

type DiscoverWalletSeedReader interface {
	ListAdminCuratedWalletSeeds(context.Context) ([]db.CuratedWalletSeed, error)
}

type DiscoverAutomaticWalletReader interface {
	ListAutoDiscoverWallets(context.Context, int) ([]db.AutoDiscoverWallet, error)
}

type DiscoverDomesticPrelistingReader interface {
	ListDomesticPrelistingCandidates(context.Context, time.Time, time.Time, int) ([]db.DomesticPrelistingCandidateRecord, error)
}

type DiscoverService struct {
	Seeds              DiscoverWalletSeedReader
	AutoCandidates     DiscoverAutomaticWalletReader
	DomesticPrelisting DiscoverDomesticPrelistingReader
	Now                func() time.Time
}

const discoverFeaturedWalletTargetCount = 12

func NewDiscoverService(seeds DiscoverWalletSeedReader, readers ...any) *DiscoverService {
	service := &DiscoverService{
		Seeds: seeds,
		Now:   time.Now,
	}

	for _, reader := range readers {
		switch typed := reader.(type) {
		case DiscoverAutomaticWalletReader:
			service.AutoCandidates = typed
		case DiscoverDomesticPrelistingReader:
			service.DomesticPrelisting = typed
		}
	}

	return service
}

func (s *DiscoverService) ListFeaturedWallets(ctx context.Context) (DiscoverFeaturedWalletResponse, error) {
	if s == nil {
		return DiscoverFeaturedWalletResponse{Items: []DiscoverFeaturedWallet{}}, nil
	}

	featured := make([]DiscoverFeaturedWallet, 0, discoverFeaturedWalletTargetCount)
	seen := make(map[string]struct{}, discoverFeaturedWalletTargetCount)

	if s.Seeds != nil {
		items, err := s.Seeds.ListAdminCuratedWalletSeeds(ctx)
		if err != nil {
			return DiscoverFeaturedWalletResponse{}, err
		}
		featured = appendUniqueDiscoverWallets(featured, seen, mapAdminCuratedWalletSeeds(items))
	}

	if len(featured) < discoverFeaturedWalletTargetCount && s.AutoCandidates != nil {
		items, err := s.AutoCandidates.ListAutoDiscoverWallets(ctx, discoverFeaturedWalletTargetCount*2)
		if err != nil {
			return DiscoverFeaturedWalletResponse{}, err
		}
		featured = appendUniqueDiscoverWallets(featured, seen, mapAutoDiscoverWallets(items))
	}

	if len(featured) > discoverFeaturedWalletTargetCount {
		featured = featured[:discoverFeaturedWalletTargetCount]
	}

	return DiscoverFeaturedWalletResponse{
		Items: featured,
	}, nil
}

func (s *DiscoverService) ListDomesticPrelistingCandidates(
	ctx context.Context,
	limit int,
) (DiscoverDomesticPrelistingResponse, error) {
	if s == nil || s.DomesticPrelisting == nil {
		return DiscoverDomesticPrelistingResponse{
			Items: []DiscoverDomesticPrelistingCandidate{},
		}, nil
	}

	now := time.Now
	if s.Now != nil {
		now = s.Now
	}

	items, err := s.DomesticPrelisting.ListDomesticPrelistingCandidates(
		ctx,
		now().UTC().Add(-7*24*time.Hour),
		now().UTC().Add(-24*time.Hour),
		limit,
	)
	if err != nil {
		return DiscoverDomesticPrelistingResponse{}, err
	}

	result := DiscoverDomesticPrelistingResponse{
		Items: make([]DiscoverDomesticPrelistingCandidate, 0, len(items)),
	}
	for _, item := range items {
		candidate := DiscoverDomesticPrelistingCandidate{
			Chain:                     strings.ToLower(strings.TrimSpace(item.Chain)),
			TokenAddress:              strings.TrimSpace(item.TokenAddress),
			TokenSymbol:               strings.TrimSpace(item.TokenSymbol),
			NormalizedAssetKey:        strings.TrimSpace(item.NormalizedAssetKey),
			TransferCount7d:           item.TransferCount7d,
			TransferCount24h:          item.TransferCount24h,
			ActiveWalletCount:         item.ActiveWalletCount,
			TrackedWalletCount:        item.TrackedWalletCount,
			DistinctCounterpartyCount: item.DistinctCounterpartyCount,
			TotalAmount:               strings.TrimSpace(item.TotalAmount),
			LargestTransferAmount:     strings.TrimSpace(item.LargestTransferAmount),
			RepresentativeWalletChain: strings.ToLower(strings.TrimSpace(item.RepresentativeWalletChain)),
			RepresentativeWallet:      strings.TrimSpace(item.RepresentativeWallet),
			RepresentativeLabel:       strings.TrimSpace(item.RepresentativeLabel),
		}
		if !item.LatestObservedAt.IsZero() {
			candidate.LatestObservedAt = item.LatestObservedAt.UTC().Format(time.RFC3339)
		}
		result.Items = append(result.Items, candidate)
	}
	return result, nil
}

func mapAdminCuratedWalletSeeds(items []db.CuratedWalletSeed) []DiscoverFeaturedWallet {
	if len(items) == 0 {
		return []DiscoverFeaturedWallet{}
	}

	featured := make([]DiscoverFeaturedWallet, 0, len(items))
	seen := make(map[string]struct{}, len(items))

	for _, item := range items {
		chain := strings.TrimSpace(string(item.Chain))
		address := strings.TrimSpace(item.Address)
		if chain == "" || address == "" {
			continue
		}

		key := chain + ":" + strings.ToLower(address)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		tags := mergeDiscoverTags(item.ListTags, item.ItemTags)
		featured = append(featured, DiscoverFeaturedWallet{
			Chain:       chain,
			Address:     address,
			DisplayName: compactDiscoverAddress(address),
			Description: discoverCuratedWalletDescription(item),
			Category:    discoverWalletCategory(item.ItemTags, item.ListTags),
			Tags:        tags,
			ObservedAt:  discoverCuratedObservedAt(item.UpdatedAt),
		})
	}

	slices.SortFunc(featured, func(left, right DiscoverFeaturedWallet) int {
		leftTier := discoverTierRank(left.Tags)
		rightTier := discoverTierRank(right.Tags)
		if leftTier != rightTier {
			return rightTier - leftTier
		}
		if left.ObservedAt != right.ObservedAt {
			return strings.Compare(right.ObservedAt, left.ObservedAt)
		}
		return strings.Compare(left.Address, right.Address)
	})

	return featured
}

func discoverCuratedWalletDescription(item db.CuratedWalletSeed) string {
	if strings.TrimSpace(item.ItemNotes) != "" {
		return strings.TrimSpace(item.ItemNotes)
	}
	if strings.TrimSpace(item.ListNotes) != "" {
		return strings.TrimSpace(item.ListNotes)
	}
	return "Curated wallet kept warm for discover coverage and proactive indexing."
}

func discoverCuratedObservedAt(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func discoverWalletCategory(tagSets ...[]string) string {
	for _, tags := range tagSets {
		for _, tag := range tags {
			cleaned := strings.TrimSpace(strings.ToLower(tag))
			switch cleaned {
			case "", "admin-curated", "wallet-seeds", "seed-import", "featured", "verified-public", "probable", "public-label", "official-docs", "official-treasury", "public-ens", "multisig", "evm", "solana":
				continue
			default:
				return cleaned
			}
		}
	}
	return "curated"
}

func mergeDiscoverTags(tagSets ...[]string) []string {
	seen := map[string]struct{}{}
	merged := make([]string, 0)
	for _, tags := range tagSets {
		for _, tag := range tags {
			cleaned := strings.TrimSpace(strings.ToLower(tag))
			if cleaned == "" {
				continue
			}
			if _, ok := seen[cleaned]; ok {
				continue
			}
			seen[cleaned] = struct{}{}
			merged = append(merged, cleaned)
		}
	}
	return merged
}

func appendUniqueDiscoverWallets(
	base []DiscoverFeaturedWallet,
	seen map[string]struct{},
	items []DiscoverFeaturedWallet,
) []DiscoverFeaturedWallet {
	for _, item := range items {
		chain := strings.TrimSpace(strings.ToLower(item.Chain))
		address := strings.TrimSpace(strings.ToLower(item.Address))
		if chain == "" || address == "" {
			continue
		}

		key := chain + ":" + address
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		base = append(base, item)
	}

	return base
}

func mapAutoDiscoverWallets(items []db.AutoDiscoverWallet) []DiscoverFeaturedWallet {
	if len(items) == 0 {
		return []DiscoverFeaturedWallet{}
	}

	featured := make([]DiscoverFeaturedWallet, 0, len(items))
	for _, item := range items {
		chain := strings.TrimSpace(string(item.Chain))
		address := strings.TrimSpace(item.Address)
		if chain == "" || address == "" {
			continue
		}

		featured = append(featured, DiscoverFeaturedWallet{
			Chain:       chain,
			Address:     address,
			DisplayName: discoverAutoDisplayName(item),
			Description: discoverAutoWalletDescription(item),
			Category:    discoverAutoWalletCategory(item),
			Tags:        discoverAutoWalletTags(item),
			ObservedAt:  discoverAutoObservedAt(item),
		})
	}

	return featured
}

func discoverAutoDisplayName(item db.AutoDiscoverWallet) string {
	displayName := strings.TrimSpace(item.DisplayName)
	if displayName != "" && !strings.EqualFold(displayName, item.Address) {
		return displayName
	}
	return compactDiscoverAddress(strings.TrimSpace(item.Address))
}

func discoverAutoWalletDescription(item db.AutoDiscoverWallet) string {
	status := strings.TrimSpace(strings.ToLower(item.Status))
	source := strings.TrimSpace(strings.ToLower(item.SourceType))

	switch source {
	case "user_search":
		return "Auto-discovered from repeated wallet searches and kept warm for deeper analysis."
	case "watchlist", "seed_list":
		return "Tracked automatically from an active seed/watchlist pipeline and ready for investigation."
	case "hop_expansion":
		return "Expanded automatically from nearby graph activity and promoted into discover coverage."
	case "dune_candidate", "mobula_candidate":
		return "Candidate wallet surfaced by automated ranking and promoted into discover coverage."
	}

	switch status {
	case db.WalletTrackingStatusScored:
		return "Auto-discovered wallet with enough activity to score and inspect immediately."
	case db.WalletTrackingStatusLabeled:
		return "Auto-discovered wallet with labeling signals already attached."
	case db.WalletTrackingStatusTracked:
		return "Auto-discovered wallet currently under active tracking."
	default:
		return "Auto-discovered wallet queued from recent indexing and ready for follow-up."
	}
}

func discoverAutoWalletCategory(item db.AutoDiscoverWallet) string {
	switch strings.TrimSpace(strings.ToLower(item.SourceType)) {
	case "user_search":
		return "searched"
	case "watchlist", "seed_list":
		return "tracked"
	case "hop_expansion":
		return "graph"
	case "dune_candidate", "mobula_candidate":
		return "candidate"
	default:
		return "auto"
	}
}

func discoverAutoWalletTags(item db.AutoDiscoverWallet) []string {
	tags := []string{"auto-discovered"}

	status := strings.TrimSpace(strings.ToLower(item.Status))
	if status != "" {
		tags = append(tags, status)
	}

	source := strings.TrimSpace(strings.ToLower(item.SourceType))
	if source != "" {
		tags = append(tags, source)
	}

	return mergeDiscoverTags(tags)
}

func discoverAutoObservedAt(item db.AutoDiscoverWallet) string {
	for _, value := range []*time.Time{item.LastActivityAt, item.LastRealtimeAt, item.FirstDiscoveredAt} {
		if value != nil && !value.IsZero() {
			return value.UTC().Format(time.RFC3339)
		}
	}
	if !item.UpdatedAt.IsZero() {
		return item.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return ""
}

func discoverTierRank(tags []string) int {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), "verified-public") {
			return 2
		}
	}
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), "probable") {
			return 1
		}
	}
	return 0
}

func compactDiscoverAddress(value string) string {
	if len(value) <= 18 {
		return value
	}
	return value[:8] + "…" + value[len(value)-6:]
}
