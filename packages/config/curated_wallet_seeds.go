package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qorvi/qorvi/packages/domain"
)

const DefaultCuratedWalletSeedsPath = "configs/curated-wallet-seeds.json"

type CuratedWalletSeed struct {
	Chain            domain.Chain `json:"chain"`
	Address          string       `json:"address"`
	DisplayName      string       `json:"displayName"`
	Description      string       `json:"description"`
	Category         string       `json:"category"`
	TrackingPriority int          `json:"trackingPriority"`
	CandidateScore   float64      `json:"candidateScore"`
	Confidence       float64      `json:"confidence"`
	Tags             []string     `json:"tags"`
}

func CuratedWalletSeedsPathFromEnv() string {
	path := strings.TrimSpace(os.Getenv("QORVI_CURATED_WALLET_SEEDS_PATH"))
	if path == "" {
		return DefaultCuratedWalletSeedsPath
	}
	return path
}

func LoadCuratedWalletSeedsFromFile(path string) ([]CuratedWalletSeed, error) {
	resolvedPath := strings.TrimSpace(path)
	if resolvedPath == "" {
		resolvedPath = DefaultCuratedWalletSeedsPath
	}
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Clean(filepath.Join(".", resolvedPath))
	}

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("read curated wallet seeds: %w", err)
	}

	var seeds []CuratedWalletSeed
	if err := json.Unmarshal(content, &seeds); err != nil {
		return nil, fmt.Errorf("parse curated wallet seeds: %w", err)
	}

	normalized := make([]CuratedWalletSeed, 0, len(seeds))
	for index, seed := range seeds {
		cleaned, err := normalizeCuratedWalletSeed(seed)
		if err != nil {
			return nil, fmt.Errorf("normalize curated wallet seed %d: %w", index, err)
		}
		normalized = append(normalized, cleaned)
	}

	return normalized, nil
}

func normalizeCuratedWalletSeed(seed CuratedWalletSeed) (CuratedWalletSeed, error) {
	chain := domain.Chain(strings.ToLower(strings.TrimSpace(string(seed.Chain))))
	if !domain.IsSupportedChain(chain) {
		return CuratedWalletSeed{}, fmt.Errorf("unsupported chain %q", seed.Chain)
	}
	address := strings.TrimSpace(seed.Address)
	if address == "" {
		return CuratedWalletSeed{}, fmt.Errorf("wallet address is required")
	}

	cleaned := CuratedWalletSeed{
		Chain:            chain,
		Address:          address,
		DisplayName:      strings.TrimSpace(seed.DisplayName),
		Description:      strings.TrimSpace(seed.Description),
		Category:         strings.TrimSpace(strings.ToLower(seed.Category)),
		TrackingPriority: seed.TrackingPriority,
		CandidateScore:   seed.CandidateScore,
		Confidence:       seed.Confidence,
		Tags:             normalizeCuratedWalletSeedTags(seed.Tags),
	}
	if cleaned.TrackingPriority <= 0 {
		cleaned.TrackingPriority = 240
	}
	if cleaned.CandidateScore <= 0 {
		cleaned.CandidateScore = 0.9
	}
	if cleaned.Confidence <= 0 {
		cleaned.Confidence = cleaned.CandidateScore
	}
	if cleaned.Confidence > 1 {
		cleaned.Confidence = 1
	}
	if cleaned.CandidateScore > 1 {
		cleaned.CandidateScore = 1
	}
	if cleaned.DisplayName == "" {
		cleaned.DisplayName = cleaned.Address
	}

	return cleaned, nil
}

func normalizeCuratedWalletSeedTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		cleaned := strings.TrimSpace(strings.ToLower(tag))
		if cleaned == "" {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		normalized = append(normalized, cleaned)
	}
	return normalized
}
