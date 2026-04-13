package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/qorvi/qorvi/packages/intelligence"
)

const workerModeAnalysisDuneBacktestNormalize = "analysis-dune-backtest-normalize"

func duneBacktestQueryResultPath() string {
	return strings.TrimSpace(os.Getenv("QORVI_DUNE_QUERY_RESULT_PATH"))
}

func duneBacktestQueryName() string {
	return strings.TrimSpace(os.Getenv("QORVI_DUNE_QUERY_NAME"))
}

func duneBacktestCandidateExportPath() string {
	value := strings.TrimSpace(os.Getenv("QORVI_DUNE_CANDIDATE_EXPORT_PATH"))
	if value == "" {
		return "packages/intelligence/test/dune-backtest-candidates.json"
	}
	return value
}

func normalizeAndWriteDuneBacktestCandidates() (intelligence.DuneBacktestCandidateSummary, error) {
	queryResultPath := duneBacktestQueryResultPath()
	if queryResultPath == "" {
		return intelligence.DuneBacktestCandidateSummary{}, fmt.Errorf("QORVI_DUNE_QUERY_RESULT_PATH is required")
	}
	queryName := duneBacktestQueryName()
	if queryName == "" {
		return intelligence.DuneBacktestCandidateSummary{}, fmt.Errorf("QORVI_DUNE_QUERY_NAME is required")
	}
	result, err := intelligence.LoadDuneQueryResultEnvelope(queryResultPath)
	if err != nil {
		return intelligence.DuneBacktestCandidateSummary{}, err
	}
	export, err := intelligence.NormalizeDuneBacktestCandidateExport(result, queryName)
	if err != nil {
		return intelligence.DuneBacktestCandidateSummary{}, err
	}
	if err := intelligence.ValidateDuneBacktestCandidateExport(export); err != nil {
		return intelligence.DuneBacktestCandidateSummary{}, err
	}
	outputPath := duneBacktestCandidateExportPath()
	if err := intelligence.WriteDuneBacktestCandidateExport(outputPath, export); err != nil {
		return intelligence.DuneBacktestCandidateSummary{}, err
	}
	return intelligence.SummarizeDuneBacktestCandidateExport(outputPath, export), nil
}

func buildDuneBacktestCandidateSummary(summary intelligence.DuneBacktestCandidateSummary) string {
	return fmt.Sprintf(
		"Dune backtest candidates normalized (path=%s, query_id=%d, query_name=%s, rows=%d, cohorts=%s, case_types=%s)",
		summary.Path,
		summary.QueryID,
		summary.QueryName,
		summary.RowCount,
		formatDuneBacktestCountMap(summary.CohortCounts),
		formatDuneBacktestCountMap(summary.CaseTypeCounts),
	)
}

func formatDuneBacktestCountMap(items map[string]int) string {
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
