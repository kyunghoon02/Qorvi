package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

var evmAddressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

const (
	searchKindEVMAddress      = "evm_address"
	searchKindSolanaAddress   = "solana_address"
	searchKindENSName         = "ens_name"
	searchKindUnknown         = "unknown"
	searchTypeWallet          = "wallet"
	searchTypeIdentity        = "identity"
	searchTypeUnknown         = "unknown"
	searchLookupMissSource    = "search_lookup_miss"
	searchStaleRefreshSource  = "search_stale_refresh"
	searchManualRefreshSource = "search_manual_refresh"
)

const (
	searchStaleRefreshAfter      = 30 * time.Minute
	searchStaleRefreshWindowDays = 90
	searchStaleRefreshLimit      = 500
	searchStaleRefreshDepth      = 1
	searchManualRefreshWindowDays = 365
	searchManualRefreshLimit      = 1000
	searchManualRefreshDepth      = 1
	searchLookupMissWindowDays    = 180
	searchLookupMissLimit         = 750
	searchLookupMissDepth        = 1
)

type SearchResult struct {
	Type        string  `json:"type"`
	Kind        string  `json:"kind"`
	KindLabel   string  `json:"kindLabel,omitempty"`
	Label       string  `json:"label"`
	Chain       string  `json:"chain,omitempty"`
	ChainLabel  string  `json:"chainLabel,omitempty"`
	WalletRoute string  `json:"walletRoute,omitempty"`
	Queued      bool    `json:"queued,omitempty"`
	Explanation string  `json:"explanation"`
	Confidence  float64 `json:"confidence"`
	Navigation  bool    `json:"navigation"`
}

type SearchResponse struct {
	Query       string         `json:"query"`
	InputKind   string         `json:"inputKind"`
	Explanation string         `json:"explanation"`
	Results     []SearchResult `json:"results"`
}

type SearchOptions struct {
	ManualRefresh bool
}

type WalletSummaryLookup interface {
	GetWalletSummary(context.Context, string, string) (WalletSummary, error)
}

type SearchService struct {
	wallets WalletSummaryLookup
	queue   db.WalletBackfillQueueStore
	tracking db.WalletTrackingStateStore
	Now     func() time.Time
}

func NewSearchService(wallets WalletSummaryLookup) *SearchService {
	return &SearchService{wallets: wallets, Now: time.Now}
}

func NewSearchServiceWithBackfillQueue(
	wallets WalletSummaryLookup,
	queue db.WalletBackfillQueueStore,
) *SearchService {
	return NewSearchServiceWithBackfillQueueAndTracking(wallets, queue, nil)
}

func NewSearchServiceWithBackfillQueueAndTracking(
	wallets WalletSummaryLookup,
	queue db.WalletBackfillQueueStore,
	tracking db.WalletTrackingStateStore,
) *SearchService {
	return &SearchService{
		wallets: wallets,
		queue:   queue,
		tracking: tracking,
		Now:     time.Now,
	}
}

func (s *SearchService) Search(ctx context.Context, query string) SearchResponse {
	return s.SearchWithOptions(ctx, query, SearchOptions{})
}

func (s *SearchService) SearchWithOptions(
	ctx context.Context,
	query string,
	options SearchOptions,
) SearchResponse {
	trimmed := strings.TrimSpace(query)
	classification := classifySearchQuery(trimmed)

	result := SearchResult{
		Type:        classification.resultType,
		Kind:        classification.kind,
		KindLabel:   classification.kindLabel,
		Label:       classification.label,
		Chain:       classification.chain,
		ChainLabel:  classification.chainLabel,
		WalletRoute: classification.walletRoute,
		Explanation: classification.explanation,
		Confidence:  classification.confidence,
		Navigation:  classification.walletRoute != "",
	}

	if classification.resultType == searchTypeWallet && s != nil && s.wallets != nil {
		if summary, err := s.wallets.GetWalletSummary(ctx, classification.chain, trimmed); err == nil {
			result = enrichSearchResult(result, summary)
			result.Explanation = fmt.Sprintf("Found wallet summary for %s.", result.Label)
			if s.queue != nil {
				if options.ManualRefresh && shouldQueueManualRefresh(summary) {
					job := db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
						Chain:       domain.Chain(classification.chain),
						Address:     trimmed,
						Source:      searchManualRefreshSource,
						RequestedAt: s.now().UTC(),
						Metadata: map[string]any{
							"input_kind":                      classification.kind,
							"query":                           trimmed,
							"refresh_reason":                  "manual",
							"last_indexed_at":                 strings.TrimSpace(summary.Indexing.LastIndexedAt),
							"backfill_window_days":            searchManualRefreshWindowDays,
							"backfill_limit":                  searchManualRefreshLimit,
							"backfill_expansion_depth":        searchManualRefreshDepth,
							"backfill_stop_service_addresses": true,
						},
					})
					if enqueueErr := s.queue.EnqueueWalletBackfill(ctx, job); enqueueErr == nil {
						result.Queued = true
						result.Explanation = fmt.Sprintf(
							"Found wallet summary for %s. Queued a background coverage expansion on demand.",
							result.Label,
						)
					}
				} else if shouldRefreshStaleWalletSummary(summary, s.now()) {
					job := db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
						Chain:       domain.Chain(classification.chain),
						Address:     trimmed,
						Source:      searchStaleRefreshSource,
						RequestedAt: s.now().UTC(),
						Metadata: map[string]any{
							"input_kind":                      classification.kind,
							"query":                           trimmed,
							"refresh_reason":                  "stale_summary",
							"last_indexed_at":                 strings.TrimSpace(summary.Indexing.LastIndexedAt),
							"backfill_window_days":            searchStaleRefreshWindowDays,
							"backfill_limit":                  searchStaleRefreshLimit,
							"backfill_expansion_depth":        searchStaleRefreshDepth,
							"backfill_stop_service_addresses": true,
						},
					})
					if enqueueErr := s.queue.EnqueueWalletBackfill(ctx, job); enqueueErr == nil {
						result.Queued = true
						result.Explanation = fmt.Sprintf(
							"Found wallet summary for %s. Queued a background refresh because the indexed view is stale.",
							result.Label,
						)
					}
				}
			}
		} else if s.queue != nil {
			job := db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
				Chain:       domain.Chain(classification.chain),
				Address:     trimmed,
				Source:      searchLookupMissSource,
				RequestedAt: s.now().UTC(),
				Metadata: map[string]any{
					"reason":                          "user_search",
					"priority":                        120,
					"source_type":                     db.WalletTrackingSourceTypeUserSearch,
					"source_ref":                      trimmed,
					"candidate_score":                 1.0,
					"tracking_status_target":          db.WalletTrackingStatusCandidate,
					"input_kind":                      classification.kind,
					"query":                           trimmed,
					"backfill_window_days":            searchLookupMissWindowDays,
					"backfill_limit":                  searchLookupMissLimit,
					"backfill_expansion_depth":        searchLookupMissDepth,
					"backfill_stop_service_addresses": true,
				},
			})
			if s.tracking != nil {
				_ = s.tracking.RecordWalletCandidate(ctx, db.WalletTrackingCandidate{
					Chain:            domain.Chain(classification.chain),
					Address:          trimmed,
					SourceType:       db.WalletTrackingSourceTypeUserSearch,
					SourceRef:        trimmed,
					DiscoveryReason:  "user_search",
					Confidence:       1,
					CandidateScore:   1,
					TrackingPriority: 120,
					ObservedAt:       s.now().UTC(),
					Payload: map[string]any{
						"input_kind": classification.kind,
						"query":      trimmed,
					},
					Notes: map[string]any{
						"queued_via": searchLookupMissSource,
					},
				})
			}
			if enqueueErr := s.queue.EnqueueWalletBackfill(ctx, job); enqueueErr == nil {
				result.Queued = true
				result.Explanation = fmt.Sprintf("Wallet not indexed yet. Queued background backfill for %s.", result.ChainLabel)
			}
		}
	}

	return SearchResponse{
		Query:       trimmed,
		InputKind:   classification.kind,
		Explanation: result.Explanation,
		Results:     []SearchResult{result},
	}
}

func shouldQueueManualRefresh(summary WalletSummary) bool {
	return !strings.EqualFold(strings.TrimSpace(summary.Indexing.Status), "indexing")
}

func (s *SearchService) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}

	return time.Now()
}

func shouldRefreshStaleWalletSummary(summary WalletSummary, now time.Time) bool {
	if strings.EqualFold(strings.TrimSpace(summary.Indexing.Status), "indexing") {
		return false
	}

	lastIndexedAt, ok := parseIndexedAt(summary.Indexing.LastIndexedAt)
	if !ok {
		return false
	}

	return now.UTC().Sub(lastIndexedAt) >= searchStaleRefreshAfter
}

func parseIndexedAt(value string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, false
	}

	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.UTC(), true
		}
	}

	return time.Time{}, false
}

type searchClassification struct {
	resultType  string
	kind        string
	kindLabel   string
	label       string
	chain       string
	chainLabel  string
	walletRoute string
	explanation string
	confidence  float64
}

func classifySearchQuery(query string) searchClassification {
	if query == "" {
		return searchClassification{
			resultType:  searchTypeUnknown,
			kind:        searchKindUnknown,
			kindLabel:   "Unknown input",
			label:       "Empty query",
			explanation: "Enter an EVM address, Solana address, or ENS-like name to search.",
			confidence:  0,
		}
	}

	if isEVMAddress(query) {
		return searchClassification{
			resultType:  searchTypeWallet,
			kind:        searchKindEVMAddress,
			kindLabel:   "EVM wallet address",
			label:       fmt.Sprintf("EVM wallet %s", query),
			chain:       string(domain.ChainEVM),
			chainLabel:  "EVM",
			walletRoute: walletDetailRoute(string(domain.ChainEVM), query),
			explanation: "Recognized as an EVM wallet address.",
			confidence:  1,
		}
	}

	if isSolanaAddress(query) {
		return searchClassification{
			resultType:  searchTypeWallet,
			kind:        searchKindSolanaAddress,
			kindLabel:   "Solana wallet address",
			label:       fmt.Sprintf("Solana wallet %s", query),
			chain:       string(domain.ChainSolana),
			chainLabel:  "Solana",
			walletRoute: walletDetailRoute(string(domain.ChainSolana), query),
			explanation: "Recognized as a Solana wallet address.",
			confidence:  1,
		}
	}

	if isENSLike(query) {
		return searchClassification{
			resultType:  searchTypeIdentity,
			kind:        searchKindENSName,
			kindLabel:   "ENS-like name",
			label:       query,
			explanation: "Recognized as an ENS-like name. Resolve it before navigating to a wallet.",
			confidence:  0.82,
		}
	}

	return searchClassification{
		resultType:  searchTypeUnknown,
		kind:        searchKindUnknown,
		kindLabel:   "Unknown input",
		label:       query,
		explanation: "The query does not match an EVM address, Solana address, or ENS-like name.",
		confidence:  0.1,
	}
}

func enrichSearchResult(result SearchResult, summary WalletSummary) SearchResult {
	enriched := result
	enriched.Label = walletLabel(summary, result.Label)
	chain := strings.TrimSpace(string(summary.Chain))
	address := strings.TrimSpace(summary.Address)
	if chain != "" {
		enriched.Chain = chain
		enriched.ChainLabel = chainLabel(domain.Chain(chain))
	}
	if chain != "" && address != "" {
		enriched.WalletRoute = walletDetailRoute(chain, address)
		enriched.Navigation = true
	}

	return enriched
}

func walletLabel(summary WalletSummary, fallback string) string {
	label := strings.TrimSpace(summary.DisplayName)
	if label != "" {
		return label
	}

	if address := strings.TrimSpace(summary.Address); address != "" {
		return address
	}

	return fallback
}

func chainLabel(chain domain.Chain) string {
	switch chain {
	case domain.ChainEVM:
		return "EVM"
	case domain.ChainSolana:
		return "Solana"
	default:
		return strings.ToUpper(strings.TrimSpace(string(chain)))
	}
}

func isEVMAddress(query string) bool {
	return evmAddressPattern.MatchString(query)
}

func isSolanaAddress(query string) bool {
	if len(query) < 32 || len(query) > 44 {
		return false
	}

	for _, r := range query {
		if !strings.ContainsRune("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", r) {
			return false
		}
	}

	return true
}

func isENSLike(query string) bool {
	lowered := strings.ToLower(query)
	if !strings.HasSuffix(lowered, ".eth") {
		return false
	}

	labels := strings.Split(lowered, ".")
	if len(labels) < 2 {
		return false
	}

	for _, label := range labels {
		if label == "" {
			return false
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}

	return true
}

func walletDetailRoute(chain string, address string) string {
	return "/v1/wallets/" + chain + "/" + address + "/summary"
}
