package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/qorvi/qorvi/packages/intelligence"
)

const workerModeAnalysisDuneBacktestPromote = "analysis-dune-backtest-promote"
const workerModeAnalysisDuneBacktestCandidateValidate = "analysis-dune-backtest-candidate-validate"

func duneCandidateExportPath() string {
	value := strings.TrimSpace(os.Getenv("QORVI_DUNE_CANDIDATE_EXPORT_PATH"))
	if value == "" {
		return "packages/intelligence/test/dune-backtest-candidates.json"
	}
	return value
}

func draftBacktestManifestPath() string {
	value := strings.TrimSpace(os.Getenv("QORVI_BACKTEST_MANIFEST_PATH"))
	if value == "" {
		return "packages/intelligence/test/backtest-manifest.json"
	}
	return value
}

func loadAndValidateDuneCandidateExport() (intelligence.DuneBacktestCandidateSummary, error) {
	path := duneCandidateExportPath()
	export, err := intelligence.LoadDuneBacktestCandidateExport(path)
	if err != nil {
		return intelligence.DuneBacktestCandidateSummary{}, err
	}
	if err := intelligence.ValidateDuneBacktestCandidateExport(export); err != nil {
		return intelligence.DuneBacktestCandidateSummary{}, err
	}
	return intelligence.SummarizeDuneBacktestCandidateExport(path, export), nil
}

func promoteReviewedDuneCandidatesToManifest() (intelligence.BacktestManifestSummary, int, error) {
	candidatePath := duneCandidateExportPath()
	export, err := intelligence.LoadDuneBacktestCandidateExport(candidatePath)
	if err != nil {
		return intelligence.BacktestManifestSummary{}, 0, err
	}
	manifestPath := draftBacktestManifestPath()
	manifest := intelligence.BacktestManifest{}
	if _, err := os.Stat(manifestPath); err == nil {
		info, statErr := os.Stat(manifestPath)
		if statErr != nil {
			return intelligence.BacktestManifestSummary{}, 0, statErr
		}
		if info.Size() > 0 {
			loaded, loadErr := intelligence.LoadBacktestManifest(manifestPath)
			if loadErr != nil {
				return intelligence.BacktestManifestSummary{}, 0, loadErr
			}
			manifest = loaded
		}
	}
	promotedManifest, promoted, err := intelligence.PromoteReviewedDuneBacktestCandidates(manifest, export)
	if err != nil {
		return intelligence.BacktestManifestSummary{}, 0, err
	}
	if err := intelligence.WriteBacktestManifest(manifestPath, promotedManifest); err != nil {
		return intelligence.BacktestManifestSummary{}, 0, err
	}
	return intelligence.SummarizeBacktestManifest(manifestPath, promotedManifest), promoted, nil
}

func buildDuneCandidateValidationSummary(summary intelligence.DuneBacktestCandidateSummary) string {
	return fmt.Sprintf(
		"Dune backtest candidate export valid (path=%s, query_id=%d, query_name=%s, rows=%d, cohorts=%s, case_types=%s)",
		summary.Path,
		summary.QueryID,
		summary.QueryName,
		summary.RowCount,
		formatDuneBacktestCountMap(summary.CohortCounts),
		formatDuneBacktestCountMap(summary.CaseTypeCounts),
	)
}

func buildDunePromotionSummary(summary intelligence.BacktestManifestSummary, promoted int) string {
	return fmt.Sprintf(
		"Dune backtest candidates promoted (path=%s, version=%s, promoted=%d, datasets=%d, cohorts=%s, case_types=%s)",
		summary.Path,
		summary.Version,
		promoted,
		summary.DatasetCount,
		formatBacktestCountMap(summary.CohortCounts),
		formatBacktestCountMap(summary.CaseTypeCounts),
	)
}
