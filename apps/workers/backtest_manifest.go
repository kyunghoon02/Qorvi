package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/qorvi/qorvi/packages/intelligence"
)

const workerModeAnalysisBacktestManifestValidate = "analysis-backtest-manifest-validate"

func backtestManifestPath() string {
	if value := strings.TrimSpace(os.Getenv("QORVI_BACKTEST_MANIFEST_PATH")); value != "" {
		return value
	}
	return "packages/intelligence/test/backtest-manifest.json"
}

func loadAndValidateBacktestManifest() (intelligence.BacktestManifestSummary, error) {
	path := backtestManifestPath()
	manifest, err := intelligence.LoadBacktestManifest(path)
	if err != nil {
		return intelligence.BacktestManifestSummary{}, err
	}
	if err := intelligence.ValidateBacktestManifest(manifest); err != nil {
		return intelligence.BacktestManifestSummary{}, err
	}
	return intelligence.SummarizeBacktestManifest(path, manifest), nil
}

func buildBacktestManifestSummary(summary intelligence.BacktestManifestSummary) string {
	return fmt.Sprintf(
		"Backtest manifest valid (path=%s, version=%s, datasets=%d, cohorts=%s, case_types=%s)",
		summary.Path,
		summary.Version,
		summary.DatasetCount,
		formatBacktestCountMap(summary.CohortCounts),
		formatBacktestCountMap(summary.CaseTypeCounts),
	)
}

func formatBacktestCountMap(items map[string]int) string {
	if len(items) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", key, items[key]))
	}
	return strings.Join(parts, ",")
}
