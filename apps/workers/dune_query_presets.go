package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/qorvi/qorvi/packages/intelligence"
)

const workerModeAnalysisDuneBacktestPresetValidate = "analysis-dune-backtest-preset-validate"

func duneBacktestPresetPath() string {
	value := strings.TrimSpace(os.Getenv("QORVI_DUNE_QUERY_PRESET_PATH"))
	if value == "" {
		return "queries/dune/backtest/query-presets.json"
	}
	return value
}

func loadAndValidateDuneBacktestQueryPresets() (intelligence.DuneBacktestQueryPresetSummary, error) {
	path := duneBacktestPresetPath()
	collection, err := intelligence.LoadDuneBacktestQueryPresets(path)
	if err != nil {
		return intelligence.DuneBacktestQueryPresetSummary{}, err
	}
	if err := intelligence.ValidateDuneBacktestQueryPresets(collection); err != nil {
		return intelligence.DuneBacktestQueryPresetSummary{}, err
	}
	return intelligence.SummarizeDuneBacktestQueryPresets(path, collection), nil
}

func buildDuneBacktestPresetSummary(summary intelligence.DuneBacktestQueryPresetSummary) string {
	return fmt.Sprintf(
		"Dune backtest query presets valid (path=%s, version=%s, presets=%d, queries=%s, case_types=%s, cohorts=%s)",
		summary.Path,
		summary.Version,
		summary.PresetCount,
		strings.Join(summary.QueryNames, ","),
		strings.Join(summary.CaseTypes, ","),
		formatDunePresetCountMap(summary.CohortCounts),
	)
}

func formatDunePresetCountMap(items map[string]int) string {
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
