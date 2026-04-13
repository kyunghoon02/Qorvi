package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/intelligence"
)

var ErrAdminBacktestOpNotFound = errors.New("admin backtest op not found")

type AdminBacktestCheckSummary struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Configured  bool   `json:"configured"`
	Path        string `json:"path,omitempty"`
}

type AdminBacktestOpsPreview struct {
	StatusMessage string                      `json:"statusMessage"`
	Checks        []AdminBacktestCheckSummary `json:"checks"`
}

type AdminBacktestRunResult struct {
	Key        string         `json:"key"`
	Label      string         `json:"label"`
	Status     string         `json:"status"`
	Summary    string         `json:"summary"`
	ExecutedAt string         `json:"executedAt"`
	Details    map[string]any `json:"details"`
}

type AdminBacktestOpsService struct {
	backtestManifestPath    string
	dunePresetPath          string
	duneCandidateExportPath string
	duneAPIKey              string
	duneAPIBaseURL          string
	duneHTTPClient          *http.Client
	now                     func() time.Time
}

func NewAdminBacktestOpsService(
	backtestManifestPath string,
	dunePresetPath string,
	duneCandidateExportPath string,
) *AdminBacktestOpsService {
	return &AdminBacktestOpsService{
		backtestManifestPath:    strings.TrimSpace(backtestManifestPath),
		dunePresetPath:          strings.TrimSpace(dunePresetPath),
		duneCandidateExportPath: strings.TrimSpace(duneCandidateExportPath),
		duneAPIBaseURL:          intelligence.DefaultDuneAPIBaseURL,
		duneHTTPClient:          &http.Client{Timeout: 20 * time.Second},
		now:                     time.Now,
	}
}

func (s *AdminBacktestOpsService) ConfigureDuneFetch(
	apiKey string,
	baseURL string,
	client *http.Client,
) {
	s.duneAPIKey = strings.TrimSpace(apiKey)
	s.duneAPIBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if s.duneAPIBaseURL == "" {
		s.duneAPIBaseURL = intelligence.DefaultDuneAPIBaseURL
	}
	if client != nil {
		s.duneHTTPClient = client
	}
}

func (s *AdminBacktestOpsService) Preview(
	ctx context.Context,
	role string,
) (AdminBacktestOpsPreview, error) {
	_ = ctx
	if err := ensureAdminConsoleAccess(role, false); err != nil {
		return AdminBacktestOpsPreview{}, err
	}

	checks := []AdminBacktestCheckSummary{
		{
			Key:         "analysis_benchmark_fixture",
			Label:       "Analysis benchmark fixture",
			Description: "Runs the release-gate benchmark scenarios against the current scoring engine.",
			Status:      "ready",
			Configured:  true,
		},
		buildConfiguredBacktestCheck(
			"backtest_manifest_validate",
			"Backtest manifest validate",
			"Validates the reviewed real-world backtest manifest before release.",
			s.backtestManifestPath,
		),
		buildConfiguredBacktestCheck(
			"dune_query_presets_validate",
			"Dune query presets validate",
			"Validates Dune backtest query preset definitions used for candidate collection.",
			s.dunePresetPath,
		),
		buildConfiguredBacktestCheck(
			"dune_candidate_export_validate",
			"Dune candidate export validate",
			"Validates the latest reviewed Dune candidate export before promotion.",
			s.duneCandidateExportPath,
		),
	}
	if duneChecks, err := s.buildDuneFetchChecks(); err == nil {
		checks = append(checks, duneChecks...)
	}

	return AdminBacktestOpsPreview{
		StatusMessage: "Run release-gate validation checks on demand. Candidate collection still requires analyst review before promotion.",
		Checks:        checks,
	}, nil
}

func (s *AdminBacktestOpsService) Run(
	ctx context.Context,
	role string,
	key string,
) (AdminBacktestRunResult, error) {
	_ = ctx
	if err := ensureAdminConsoleAccess(role, true); err != nil {
		return AdminBacktestRunResult{}, err
	}

	executedAt := s.now().UTC().Format(time.RFC3339)
	switch strings.TrimSpace(key) {
	case "analysis_benchmark_fixture":
		summary := intelligence.RunBenchmarkScenarios(
			intelligence.DefaultBenchmarkScenarios(),
		)
		return AdminBacktestRunResult{
			Key:        "analysis_benchmark_fixture",
			Label:      "Analysis benchmark fixture",
			Status:     "succeeded",
			Summary:    fmt.Sprintf("Passed %d/%d expectations across %d scenarios.", summary.PassedCount, summary.ExpectationCount, summary.ScenarioCount),
			ExecutedAt: executedAt,
			Details: map[string]any{
				"scenarioCount":     summary.ScenarioCount,
				"expectationCount":  summary.ExpectationCount,
				"passedCount":       summary.PassedCount,
				"failedCount":       summary.FailedCount,
				"precisionAtHigh":   summary.PrecisionAtHigh,
				"falsePositiveRate": summary.FalsePositiveRate,
				"truePositiveHigh":  summary.TruePositiveHigh,
				"falsePositiveHigh": summary.FalsePositiveHigh,
				"highPredictions":   summary.HighPredictions,
			},
		}, nil
	case "backtest_manifest_validate":
		return s.runBacktestManifestValidate(executedAt), nil
	case "dune_query_presets_validate":
		return s.runDuneQueryPresetsValidate(executedAt), nil
	case "dune_candidate_export_validate":
		return s.runDuneCandidateExportValidate(executedAt), nil
	default:
		if result, ok := s.runDuneFetchNormalizeOperation(ctx, executedAt, strings.TrimSpace(key)); ok {
			return result, nil
		}
		return AdminBacktestRunResult{}, ErrAdminBacktestOpNotFound
	}
}

func (s *AdminBacktestOpsService) runBacktestManifestValidate(
	executedAt string,
) AdminBacktestRunResult {
	path := strings.TrimSpace(s.backtestManifestPath)
	if path == "" {
		return failedAdminBacktestRunResult(
			"backtest_manifest_validate",
			"Backtest manifest validate",
			executedAt,
			ErrAdminConsoleInvalidRequest,
			path,
		)
	}

	manifest, err := intelligence.LoadBacktestManifest(path)
	if err != nil {
		return failedAdminBacktestRunResult(
			"backtest_manifest_validate",
			"Backtest manifest validate",
			executedAt,
			err,
			path,
		)
	}
	if err := intelligence.ValidateBacktestManifest(manifest); err != nil {
		return failedAdminBacktestRunResult(
			"backtest_manifest_validate",
			"Backtest manifest validate",
			executedAt,
			err,
			path,
		)
	}

	summary := intelligence.SummarizeBacktestManifest(path, manifest)
	return AdminBacktestRunResult{
		Key:        "backtest_manifest_validate",
		Label:      "Backtest manifest validate",
		Status:     "succeeded",
		Summary:    fmt.Sprintf("Validated %d reviewed datasets.", summary.DatasetCount),
		ExecutedAt: executedAt,
		Details: map[string]any{
			"path":           summary.Path,
			"version":        summary.Version,
			"datasetCount":   summary.DatasetCount,
			"cohortCounts":   summary.CohortCounts,
			"caseTypeCounts": summary.CaseTypeCounts,
		},
	}
}

func (s *AdminBacktestOpsService) runDuneQueryPresetsValidate(
	executedAt string,
) AdminBacktestRunResult {
	path := strings.TrimSpace(s.dunePresetPath)
	if path == "" {
		return failedAdminBacktestRunResult(
			"dune_query_presets_validate",
			"Dune query presets validate",
			executedAt,
			ErrAdminConsoleInvalidRequest,
			path,
		)
	}

	collection, err := intelligence.LoadDuneBacktestQueryPresets(path)
	if err != nil {
		return failedAdminBacktestRunResult(
			"dune_query_presets_validate",
			"Dune query presets validate",
			executedAt,
			err,
			path,
		)
	}
	if err := intelligence.ValidateDuneBacktestQueryPresets(collection); err != nil {
		return failedAdminBacktestRunResult(
			"dune_query_presets_validate",
			"Dune query presets validate",
			executedAt,
			err,
			path,
		)
	}

	summary := intelligence.SummarizeDuneBacktestQueryPresets(path, collection)
	return AdminBacktestRunResult{
		Key:        "dune_query_presets_validate",
		Label:      "Dune query presets validate",
		Status:     "succeeded",
		Summary:    fmt.Sprintf("Validated %d query presets.", summary.PresetCount),
		ExecutedAt: executedAt,
		Details: map[string]any{
			"path":         summary.Path,
			"version":      summary.Version,
			"presetCount":  summary.PresetCount,
			"queryNames":   summary.QueryNames,
			"caseTypes":    summary.CaseTypes,
			"cohortCounts": summary.CohortCounts,
		},
	}
}

func (s *AdminBacktestOpsService) runDuneCandidateExportValidate(
	executedAt string,
) AdminBacktestRunResult {
	path := strings.TrimSpace(s.duneCandidateExportPath)
	if path == "" {
		return failedAdminBacktestRunResult(
			"dune_candidate_export_validate",
			"Dune candidate export validate",
			executedAt,
			ErrAdminConsoleInvalidRequest,
			path,
		)
	}

	export, err := intelligence.LoadDuneBacktestCandidateExport(path)
	if err != nil {
		return failedAdminBacktestRunResult(
			"dune_candidate_export_validate",
			"Dune candidate export validate",
			executedAt,
			err,
			path,
		)
	}
	if err := intelligence.ValidateDuneBacktestCandidateExport(export); err != nil {
		return failedAdminBacktestRunResult(
			"dune_candidate_export_validate",
			"Dune candidate export validate",
			executedAt,
			err,
			path,
		)
	}

	return AdminBacktestRunResult{
		Key:        "dune_candidate_export_validate",
		Label:      "Dune candidate export validate",
		Status:     "succeeded",
		Summary:    fmt.Sprintf("Validated %d candidate rows from %s.", len(export.Rows), export.Source.QueryName),
		ExecutedAt: executedAt,
		Details: map[string]any{
			"path":        path,
			"provider":    export.Source.Provider,
			"queryName":   export.Source.QueryName,
			"queryID":     export.Source.QueryID,
			"executionId": export.Source.ExecutionID,
			"rowCount":    len(export.Rows),
		},
	}
}

func buildConfiguredBacktestCheck(
	key string,
	label string,
	description string,
	path string,
) AdminBacktestCheckSummary {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return AdminBacktestCheckSummary{
			Key:         key,
			Label:       label,
			Description: description,
			Status:      "not_configured",
			Configured:  false,
		}
	}
	if _, err := os.Stat(trimmed); err != nil {
		return AdminBacktestCheckSummary{
			Key:         key,
			Label:       label,
			Description: description,
			Status:      "missing",
			Configured:  false,
			Path:        trimmed,
		}
	}
	return AdminBacktestCheckSummary{
		Key:         key,
		Label:       label,
		Description: description,
		Status:      "ready",
		Configured:  true,
		Path:        trimmed,
	}
}

const duneFetchNormalizeOpPrefix = "dune_fetch_normalize__"

func (s *AdminBacktestOpsService) buildDuneFetchChecks() ([]AdminBacktestCheckSummary, error) {
	path := strings.TrimSpace(s.dunePresetPath)
	if path == "" {
		return nil, nil
	}
	collection, err := intelligence.LoadDuneBacktestQueryPresets(path)
	if err != nil {
		return nil, err
	}
	checks := make([]AdminBacktestCheckSummary, 0, len(collection.Presets))
	for _, preset := range collection.Presets {
		checks = append(checks, buildDuneFetchCheckSummary(preset, strings.TrimSpace(s.duneAPIKey) != ""))
	}
	return checks, nil
}

func buildDuneFetchCheckSummary(
	preset intelligence.DuneBacktestQueryPreset,
	hasAPIKey bool,
) AdminBacktestCheckSummary {
	status := "ready"
	configured := true
	if preset.QueryID <= 0 || !hasAPIKey {
		status = "not_configured"
		configured = false
	}
	return AdminBacktestCheckSummary{
		Key:         duneFetchNormalizeOperationKey(preset.Name),
		Label:       fmt.Sprintf("Fetch Dune %s", strings.TrimSpace(preset.Name)),
		Description: fmt.Sprintf("Fetches the latest saved Dune result for %s, normalizes it, and writes a validated candidate export.", strings.TrimSpace(preset.QueryName)),
		Status:      status,
		Configured:  configured,
		Path:        strings.TrimSpace(preset.CandidateOutput),
	}
}

func duneFetchNormalizeOperationKey(presetName string) string {
	return duneFetchNormalizeOpPrefix + sanitizeAdminBacktestKeyPart(presetName)
}

func sanitizeAdminBacktestKeyPart(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "unnamed"
	}
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastUnderscore = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	sanitized := strings.Trim(builder.String(), "_")
	if sanitized == "" {
		return "unnamed"
	}
	return sanitized
}

func (s *AdminBacktestOpsService) runDuneFetchNormalizeOperation(
	ctx context.Context,
	executedAt string,
	key string,
) (AdminBacktestRunResult, bool) {
	if !strings.HasPrefix(key, duneFetchNormalizeOpPrefix) {
		return AdminBacktestRunResult{}, false
	}

	collection, err := intelligence.LoadDuneBacktestQueryPresets(strings.TrimSpace(s.dunePresetPath))
	if err != nil {
		return failedAdminBacktestRunResult(
			key,
			"Dune fetch and normalize",
			executedAt,
			err,
			s.dunePresetPath,
		), true
	}
	for _, preset := range collection.Presets {
		if duneFetchNormalizeOperationKey(preset.Name) == key {
			return s.runDuneFetchNormalize(ctx, executedAt, preset), true
		}
	}
	return AdminBacktestRunResult{}, false
}

func (s *AdminBacktestOpsService) runDuneFetchNormalize(
	ctx context.Context,
	executedAt string,
	preset intelligence.DuneBacktestQueryPreset,
) AdminBacktestRunResult {
	label := fmt.Sprintf("Fetch Dune %s", strings.TrimSpace(preset.Name))
	outputPath := strings.TrimSpace(preset.CandidateOutput)
	if strings.TrimSpace(s.duneAPIKey) == "" {
		return failedAdminBacktestRunResult(
			duneFetchNormalizeOperationKey(preset.Name),
			label,
			executedAt,
			fmt.Errorf("DUNE_API_KEY is required to fetch Dune saved query results"),
			outputPath,
		)
	}
	if preset.QueryID <= 0 {
		return failedAdminBacktestRunResult(
			duneFetchNormalizeOperationKey(preset.Name),
			label,
			executedAt,
			fmt.Errorf("preset queryId must be configured before fetch can run"),
			outputPath,
		)
	}

	result, err := intelligence.FetchLatestDuneQueryResult(
		ctx,
		s.duneAPIKey,
		s.duneAPIBaseURL,
		preset.QueryID,
		s.duneHTTPClient,
	)
	if err != nil {
		return failedAdminBacktestRunResult(
			duneFetchNormalizeOperationKey(preset.Name),
			label,
			executedAt,
			err,
			outputPath,
		)
	}

	export, err := intelligence.NormalizeDuneBacktestCandidateExport(result, preset.QueryName)
	if err != nil {
		return failedAdminBacktestRunResult(
			duneFetchNormalizeOperationKey(preset.Name),
			label,
			executedAt,
			err,
			outputPath,
		)
	}
	if err := intelligence.ValidateDuneBacktestCandidateExport(export); err != nil {
		return failedAdminBacktestRunResult(
			duneFetchNormalizeOperationKey(preset.Name),
			label,
			executedAt,
			err,
			outputPath,
		)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return failedAdminBacktestRunResult(
			duneFetchNormalizeOperationKey(preset.Name),
			label,
			executedAt,
			fmt.Errorf("prepare candidate export directory: %w", err),
			outputPath,
		)
	}
	if err := intelligence.WriteDuneBacktestCandidateExport(outputPath, export); err != nil {
		return failedAdminBacktestRunResult(
			duneFetchNormalizeOperationKey(preset.Name),
			label,
			executedAt,
			err,
			outputPath,
		)
	}

	summary := intelligence.SummarizeDuneBacktestCandidateExport(outputPath, export)
	return AdminBacktestRunResult{
		Key:        duneFetchNormalizeOperationKey(preset.Name),
		Label:      label,
		Status:     "succeeded",
		Summary:    fmt.Sprintf("Fetched query %d and wrote %d candidate rows.", preset.QueryID, summary.RowCount),
		ExecutedAt: executedAt,
		Details: map[string]any{
			"path":           outputPath,
			"presetName":     preset.Name,
			"queryName":      preset.QueryName,
			"queryID":        preset.QueryID,
			"executionId":    export.Source.ExecutionID,
			"rowCount":       summary.RowCount,
			"cohortCounts":   summary.CohortCounts,
			"caseTypeCounts": summary.CaseTypeCounts,
		},
	}
}

func failedAdminBacktestRunResult(
	key string,
	label string,
	executedAt string,
	err error,
	path string,
) AdminBacktestRunResult {
	result := AdminBacktestRunResult{
		Key:        key,
		Label:      label,
		Status:     "failed",
		Summary:    err.Error(),
		ExecutedAt: executedAt,
		Details: map[string]any{
			"error": err.Error(),
		},
	}
	if strings.TrimSpace(path) != "" {
		result.Details["path"] = strings.TrimSpace(path)
	}
	return result
}
