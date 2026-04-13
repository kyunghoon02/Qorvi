package intelligence

import (
	"fmt"
	"strings"

	"github.com/qorvi/qorvi/packages/domain"
)

type BenchmarkCohort string

const (
	BenchmarkCohortKnownPositive BenchmarkCohort = "known_positive"
	BenchmarkCohortFalsePositive BenchmarkCohort = "false_positive"
	BenchmarkCohortNegative      BenchmarkCohort = "negative"
)

type BenchmarkScenario struct {
	Name            string
	Cohort          BenchmarkCohort
	Description     string
	Cluster         *ClusterSignal
	ShadowExit      *ShadowExitSignal
	FirstConnection *FirstConnectionSignal
	Expectations    []BenchmarkExpectation
}

type BenchmarkExpectation struct {
	ScoreName            domain.ScoreName
	MinRating            domain.ScoreRating
	MaxRating            domain.ScoreRating
	RequireSuppression   bool
	RequireContradiction bool
	RequireRatingBlock   bool
}

type BenchmarkScoreResult struct {
	ScoreName             domain.ScoreName
	Value                 int
	Rating                domain.ScoreRating
	Passed                bool
	SuppressionObserved   bool
	ContradictionObserved bool
	RatingBlockObserved   bool
	FailureReason         string
}

type BenchmarkScenarioResult struct {
	Name        string
	Cohort      BenchmarkCohort
	Description string
	Passed      bool
	Results     []BenchmarkScoreResult
}

type BenchmarkSummary struct {
	ScenarioCount     int
	ExpectationCount  int
	PassedCount       int
	FailedCount       int
	HighPredictions   int
	TruePositiveHigh  int
	FalsePositiveHigh int
	PrecisionAtHigh   float64
	FalsePositiveRate float64
	ScenarioResults   []BenchmarkScenarioResult
}

func RunBenchmarkScenarios(scenarios []BenchmarkScenario) BenchmarkSummary {
	summary := BenchmarkSummary{
		ScenarioCount:   len(scenarios),
		ScenarioResults: make([]BenchmarkScenarioResult, 0, len(scenarios)),
	}

	falsePositiveEligible := 0
	for _, scenario := range scenarios {
		result := runBenchmarkScenario(scenario)
		summary.ScenarioResults = append(summary.ScenarioResults, result)
		if result.Passed {
			summary.PassedCount++
		} else {
			summary.FailedCount++
		}
		for _, scoreResult := range result.Results {
			summary.ExpectationCount++
			if scoreResult.Rating == domain.RatingHigh {
				summary.HighPredictions++
				switch scenario.Cohort {
				case BenchmarkCohortKnownPositive:
					summary.TruePositiveHigh++
				case BenchmarkCohortFalsePositive, BenchmarkCohortNegative:
					summary.FalsePositiveHigh++
				}
			}
			if scenario.Cohort == BenchmarkCohortFalsePositive || scenario.Cohort == BenchmarkCohortNegative {
				falsePositiveEligible++
			}
		}
	}

	if summary.HighPredictions > 0 {
		summary.PrecisionAtHigh = float64(summary.TruePositiveHigh) / float64(summary.HighPredictions)
	}
	if falsePositiveEligible > 0 {
		summary.FalsePositiveRate = float64(summary.FalsePositiveHigh) / float64(falsePositiveEligible)
	}

	return summary
}

func runBenchmarkScenario(scenario BenchmarkScenario) BenchmarkScenarioResult {
	result := BenchmarkScenarioResult{
		Name:        strings.TrimSpace(scenario.Name),
		Cohort:      scenario.Cohort,
		Description: strings.TrimSpace(scenario.Description),
		Passed:      true,
		Results:     make([]BenchmarkScoreResult, 0, len(scenario.Expectations)),
	}
	scores := benchmarkScenarioScores(scenario)
	for _, expectation := range scenario.Expectations {
		score, ok := scores[expectation.ScoreName]
		if !ok {
			result.Passed = false
			result.Results = append(result.Results, BenchmarkScoreResult{
				ScoreName:     expectation.ScoreName,
				Passed:        false,
				FailureReason: "score_not_generated",
			})
			continue
		}
		scoreResult := evaluateBenchmarkExpectation(score, expectation)
		if !scoreResult.Passed {
			result.Passed = false
		}
		result.Results = append(result.Results, scoreResult)
	}
	return result
}

func benchmarkScenarioScores(scenario BenchmarkScenario) map[domain.ScoreName]domain.Score {
	scores := make(map[domain.ScoreName]domain.Score, 3)
	if scenario.Cluster != nil {
		score := BuildClusterScore(*scenario.Cluster)
		scores[score.Name] = score
	}
	if scenario.ShadowExit != nil {
		score := BuildShadowExitRiskScore(*scenario.ShadowExit)
		scores[score.Name] = score
	}
	if scenario.FirstConnection != nil {
		score := BuildFirstConnectionScore(*scenario.FirstConnection)
		scores[score.Name] = score
	}
	return scores
}

func evaluateBenchmarkExpectation(score domain.Score, expectation BenchmarkExpectation) BenchmarkScoreResult {
	suppressionObserved := scoreHasMetadataList(score, "suppression_reasons")
	contradictionObserved := scoreHasMetadataList(score, "contradiction_reasons")
	ratingBlockObserved := scoreHasMetadataString(score, "rating_block_reason")
	passed := true
	failures := make([]string, 0, 4)

	if ratingOrder(score.Rating) < ratingOrder(expectation.MinRating) {
		passed = false
		failures = append(failures, fmt.Sprintf("rating_below_min:%s<%s", score.Rating, expectation.MinRating))
	}
	if expectation.MaxRating != "" && ratingOrder(score.Rating) > ratingOrder(expectation.MaxRating) {
		passed = false
		failures = append(failures, fmt.Sprintf("rating_above_max:%s>%s", score.Rating, expectation.MaxRating))
	}
	if expectation.RequireSuppression && !suppressionObserved {
		passed = false
		failures = append(failures, "missing_suppression")
	}
	if expectation.RequireContradiction && !contradictionObserved {
		passed = false
		failures = append(failures, "missing_contradiction")
	}
	if expectation.RequireRatingBlock && !ratingBlockObserved {
		passed = false
		failures = append(failures, "missing_rating_block")
	}

	return BenchmarkScoreResult{
		ScoreName:             score.Name,
		Value:                 score.Value,
		Rating:                score.Rating,
		Passed:                passed,
		SuppressionObserved:   suppressionObserved,
		ContradictionObserved: contradictionObserved,
		RatingBlockObserved:   ratingBlockObserved,
		FailureReason:         strings.Join(failures, ","),
	}
}

func ratingOrder(rating domain.ScoreRating) int {
	switch rating {
	case domain.RatingHigh:
		return 3
	case domain.RatingMedium:
		return 2
	case domain.RatingLow:
		return 1
	default:
		return 0
	}
}

func scoreHasMetadataList(score domain.Score, key string) bool {
	for _, evidence := range score.Evidence {
		if values, ok := evidence.Metadata[key].([]string); ok && len(values) > 0 {
			return true
		}
		if values, ok := evidence.Metadata[key].([]any); ok && len(values) > 0 {
			return true
		}
	}
	return false
}

func scoreHasMetadataString(score domain.Score, key string) bool {
	for _, evidence := range score.Evidence {
		if value, ok := evidence.Metadata[key].(string); ok && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}
