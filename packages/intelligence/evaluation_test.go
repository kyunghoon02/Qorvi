package intelligence

import (
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestRunBenchmarkScenarios(t *testing.T) {
	t.Parallel()

	summary := RunBenchmarkScenarios(DefaultBenchmarkScenarios())

	if summary.ScenarioCount != 11 {
		t.Fatalf("expected 11 scenarios, got %d", summary.ScenarioCount)
	}
	if summary.ExpectationCount != 11 {
		t.Fatalf("expected 11 expectations, got %d", summary.ExpectationCount)
	}
	if summary.FailedCount != 0 {
		t.Fatalf("expected all benchmark scenarios to pass, got %#v", summary.ScenarioResults)
	}
	if summary.TruePositiveHigh != 3 {
		t.Fatalf("expected 3 high-confidence true positives, got %#v", summary)
	}
	if summary.FalsePositiveHigh != 0 {
		t.Fatalf("expected 0 false-positive high scores, got %#v", summary)
	}
	if summary.PrecisionAtHigh != 1 {
		t.Fatalf("expected precision@high 1.0, got %#v", summary)
	}
	if summary.FalsePositiveRate != 0 {
		t.Fatalf("expected false-positive rate 0.0, got %#v", summary)
	}
}

func TestRunBenchmarkScenariosCapturesRegression(t *testing.T) {
	t.Parallel()

	scenarios := DefaultBenchmarkScenarios()
	// Remove the suppression requirement to prove the runner still flags an unexpected high score.
	scenarios[3].Expectations[0].MaxRating = domain.RatingLow

	summary := RunBenchmarkScenarios(scenarios)
	if summary.FailedCount == 0 {
		t.Fatalf("expected benchmark regression to fail, got %#v", summary)
	}
}
