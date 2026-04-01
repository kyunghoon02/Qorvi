package main

import (
	"fmt"

	"github.com/qorvi/qorvi/packages/intelligence"
)

const workerModeAnalysisBenchmarkFixture = "analysis-benchmark-fixture"

func buildAnalysisBenchmarkSummary(summary intelligence.BenchmarkSummary) string {
	return fmt.Sprintf(
		"Analysis benchmark complete (scenarios=%d, expectations=%d, passed=%d, failed=%d, precision_at_high=%.2f, false_positive_rate=%.2f, true_positive_high=%d, false_positive_high=%d)",
		summary.ScenarioCount,
		summary.ExpectationCount,
		summary.PassedCount,
		summary.FailedCount,
		summary.PrecisionAtHigh,
		summary.FalsePositiveRate,
		summary.TruePositiveHigh,
		summary.FalsePositiveHigh,
	)
}
