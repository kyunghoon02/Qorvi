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

type DiscoverDomesticPrelistingReader interface {
	ListDomesticPrelistingCandidates(context.Context, time.Time, time.Time, int) ([]db.DomesticPrelistingCandidateRecord, error)
}

type DiscoverService struct {
	Seeds              DiscoverWalletSeedReader
	DomesticPrelisting DiscoverDomesticPrelistingReader
	Now                func() time.Time
}

func NewDiscoverService(seeds DiscoverWalletSeedReader, domestic ...DiscoverDomesticPrelistingReader) *DiscoverService {
	var domesticReader DiscoverDomesticPrelistingReader
	if len(domestic) > 0 {
		domesticReader = domestic[0]
	}
	return &DiscoverService{
		Seeds:              seeds,
		DomesticPrelisting: domesticReader,
		Now:                time.Now,
	}
}

func (s *DiscoverService) ListFeaturedWallets(ctx context.Context) (DiscoverFeaturedWalletResponse, error) {
	if s == nil || s.Seeds == nil {
		return DiscoverFeaturedWalletResponse{Items: []DiscoverFeaturedWallet{}}, nil
	}

	items, err := s.Seeds.ListAdminCuratedWalletSeeds(ctx)
	if err != nil {
		return DiscoverFeaturedWalletResponse{}, err
	}

	return DiscoverFeaturedWalletResponse{
		Items: mapAdminCuratedWalletSeeds(items),
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
