package intelligence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const DuneBacktestCandidateExportVersion = "v1"

type DuneQueryResultEnvelope struct {
	QueryID             int64  `json:"query_id"`
	ExecutionID         string `json:"execution_id"`
	SubmittedAt         string `json:"submitted_at"`
	ExecutionEndedAt    string `json:"execution_ended_at"`
	IsExecutionFinished bool   `json:"is_execution_finished"`
	Result              struct {
		Rows []map[string]any `json:"rows"`
	} `json:"result"`
}

type DuneBacktestCandidateExport struct {
	Version string                      `json:"version"`
	Source  DuneBacktestCandidateSource `json:"source"`
	Rows    []DuneBacktestCandidateRow  `json:"rows"`
}

type DuneBacktestCandidateSource struct {
	Provider    string `json:"provider"`
	QueryID     int64  `json:"queryId"`
	QueryName   string `json:"queryName"`
	ExecutionID string `json:"executionId"`
	GeneratedAt string `json:"generatedAt"`
}

type DuneBacktestCandidateRow struct {
	CaseID              string               `json:"caseId"`
	Chain               string               `json:"chain"`
	Cohort              string               `json:"cohort"`
	CaseType            string               `json:"caseType"`
	SubjectAddress      string               `json:"subjectAddress,omitempty"`
	EntityKey           string               `json:"entityKey,omitempty"`
	SubjectRole         string               `json:"subjectRole"`
	WindowStartAt       string               `json:"windowStartAt"`
	WindowEndAt         string               `json:"windowEndAt"`
	ObservationCutoffAt string               `json:"observationCutoffAt,omitempty"`
	DetectionDeadlineAt string               `json:"detectionDeadlineAt,omitempty"`
	ExpectedOutcome     string               `json:"expectedOutcome"`
	ExpectedSignal      string               `json:"expectedSignal"`
	ExpectedRoute       string               `json:"expectedRoute"`
	SourceTxHash        string               `json:"sourceTxHash"`
	SourceBlockNumber   int64                `json:"sourceBlockNumber,omitempty"`
	SourceTitle         string               `json:"sourceTitle"`
	SourceURL           string               `json:"sourceUrl"`
	Narrative           string               `json:"narrative"`
	AnalystNote         string               `json:"analystNote,omitempty"`
	Review              *DuneCandidateReview `json:"review,omitempty"`
	Metadata            map[string]any       `json:"metadata,omitempty"`
}

type DuneCandidateReview struct {
	CuratedBy           string   `json:"curatedBy"`
	ReviewStatus        string   `json:"reviewStatus"`
	CaseTicket          string   `json:"caseTicket,omitempty"`
	ExpectedHighSignals []string `json:"expectedHighSignals,omitempty"`
	ExpectedSuppressed  []string `json:"expectedSuppressed,omitempty"`
}

type DuneBacktestCandidateSummary struct {
	Path           string
	QueryID        int64
	QueryName      string
	ExecutionID    string
	RowCount       int
	CohortCounts   map[string]int
	CaseTypeCounts map[string]int
}

func LoadDuneQueryResultEnvelope(path string) (DuneQueryResultEnvelope, error) {
	payload, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return DuneQueryResultEnvelope{}, err
	}
	var result DuneQueryResultEnvelope
	if err := json.Unmarshal(payload, &result); err != nil {
		return DuneQueryResultEnvelope{}, fmt.Errorf("decode dune query result: %w", err)
	}
	return result, nil
}

func NormalizeDuneBacktestCandidateExport(result DuneQueryResultEnvelope, queryName string) (DuneBacktestCandidateExport, error) {
	if len(result.Result.Rows) == 0 {
		return DuneBacktestCandidateExport{}, fmt.Errorf("dune query result contains no rows")
	}
	export := DuneBacktestCandidateExport{
		Version: DuneBacktestCandidateExportVersion,
		Source: DuneBacktestCandidateSource{
			Provider:    "dune",
			QueryID:     result.QueryID,
			QueryName:   strings.TrimSpace(queryName),
			ExecutionID: strings.TrimSpace(result.ExecutionID),
			GeneratedAt: normalizeDuneGeneratedAt(result.ExecutionEndedAt, result.SubmittedAt),
		},
		Rows: make([]DuneBacktestCandidateRow, 0, len(result.Result.Rows)),
	}
	for index, row := range result.Result.Rows {
		normalized, err := normalizeDuneBacktestCandidateRow(row)
		if err != nil {
			return DuneBacktestCandidateExport{}, fmt.Errorf("normalize dune row %d: %w", index, err)
		}
		export.Rows = append(export.Rows, normalized)
	}
	return export, nil
}

func ValidateDuneBacktestCandidateExport(export DuneBacktestCandidateExport) error {
	if strings.TrimSpace(export.Version) == "" {
		return fmt.Errorf("candidate export version is required")
	}
	if strings.TrimSpace(export.Source.Provider) != "dune" {
		return fmt.Errorf("candidate export source.provider must be dune")
	}
	if strings.TrimSpace(export.Source.QueryName) == "" {
		return fmt.Errorf("candidate export source.queryName is required")
	}
	if strings.TrimSpace(export.Source.GeneratedAt) == "" {
		return fmt.Errorf("candidate export source.generatedAt is required")
	}
	if _, err := time.Parse(time.RFC3339, export.Source.GeneratedAt); err != nil {
		return fmt.Errorf("candidate export source.generatedAt must be RFC3339: %w", err)
	}
	if len(export.Rows) == 0 {
		return fmt.Errorf("candidate export must contain at least one row")
	}
	for index, row := range export.Rows {
		label := fmt.Sprintf("rows[%d]", index)
		if strings.TrimSpace(row.CaseID) == "" {
			return fmt.Errorf("%s.caseId is required", label)
		}
		if strings.TrimSpace(row.Chain) == "" {
			return fmt.Errorf("%s.chain is required", label)
		}
		if !isAllowedBacktestCohort(row.Cohort) {
			return fmt.Errorf("%s.cohort must be one of known_positive, known_negative, control", label)
		}
		if strings.TrimSpace(row.CaseType) == "" {
			return fmt.Errorf("%s.caseType is required", label)
		}
		if strings.TrimSpace(row.SubjectAddress) == "" && strings.TrimSpace(row.EntityKey) == "" {
			return fmt.Errorf("%s.subjectAddress or entityKey is required", label)
		}
		if strings.TrimSpace(row.SubjectRole) == "" {
			return fmt.Errorf("%s.subjectRole is required", label)
		}
		if err := validateBacktestWindow(BacktestWindow{
			StartAt:             row.WindowStartAt,
			EndAt:               row.WindowEndAt,
			ObservationCutoffAt: row.ObservationCutoffAt,
			DetectionDeadlineAt: row.DetectionDeadlineAt,
		}, label); err != nil {
			return err
		}
		if strings.TrimSpace(row.ExpectedOutcome) == "" {
			return fmt.Errorf("%s.expectedOutcome is required", label)
		}
		if strings.TrimSpace(row.ExpectedSignal) == "" {
			return fmt.Errorf("%s.expectedSignal is required", label)
		}
		if strings.TrimSpace(row.ExpectedRoute) == "" {
			return fmt.Errorf("%s.expectedRoute is required", label)
		}
		if strings.TrimSpace(row.SourceTxHash) == "" {
			return fmt.Errorf("%s.sourceTxHash is required", label)
		}
		if strings.TrimSpace(row.SourceTitle) == "" || strings.TrimSpace(row.SourceURL) == "" {
			return fmt.Errorf("%s.sourceTitle and sourceUrl are required", label)
		}
		if strings.TrimSpace(row.Narrative) == "" {
			return fmt.Errorf("%s.narrative is required", label)
		}
	}
	return nil
}

func WriteDuneBacktestCandidateExport(path string, export DuneBacktestCandidateExport) error {
	payload, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return fmt.Errorf("encode dune candidate export: %w", err)
	}
	if err := os.WriteFile(filepath.Clean(path), append(payload, '\n'), 0o644); err != nil {
		return fmt.Errorf("write dune candidate export: %w", err)
	}
	return nil
}

func SummarizeDuneBacktestCandidateExport(path string, export DuneBacktestCandidateExport) DuneBacktestCandidateSummary {
	summary := DuneBacktestCandidateSummary{
		Path:           path,
		QueryID:        export.Source.QueryID,
		QueryName:      export.Source.QueryName,
		ExecutionID:    export.Source.ExecutionID,
		RowCount:       len(export.Rows),
		CohortCounts:   make(map[string]int),
		CaseTypeCounts: make(map[string]int),
	}
	for _, row := range export.Rows {
		summary.CohortCounts[row.Cohort]++
		summary.CaseTypeCounts[row.CaseType]++
	}
	return summary
}

func normalizeDuneBacktestCandidateRow(row map[string]any) (DuneBacktestCandidateRow, error) {
	metadata := normalizeDuneMetadata(row)
	value := DuneBacktestCandidateRow{
		CaseID:              firstNonEmptyDuneString(row, "case_id", "caseId", "dataset_id", "datasetId", "dedup_key", "dedupKey"),
		Chain:               readDuneString(row, "chain"),
		Cohort:              readDuneString(row, "cohort"),
		CaseType:            firstNonEmptyDuneString(row, "case_type", "caseType"),
		SubjectAddress:      firstNonEmptyDuneString(row, "subject_address", "subjectAddress"),
		EntityKey:           firstNonEmptyDuneString(row, "entity_key", "entityKey"),
		SubjectRole:         firstNonEmptyDuneString(row, "subject_role", "subjectRole"),
		WindowStartAt:       firstNonEmptyDuneString(row, "window_start_at", "windowStartAt"),
		WindowEndAt:         firstNonEmptyDuneString(row, "window_end_at", "windowEndAt"),
		ObservationCutoffAt: firstNonEmptyDuneString(row, "observation_cutoff_at", "observationCutoffAt"),
		DetectionDeadlineAt: firstNonEmptyDuneString(row, "detection_deadline_at", "detectionDeadlineAt"),
		ExpectedOutcome:     firstNonEmptyDuneString(row, "expected_outcome", "expectedOutcome"),
		ExpectedSignal:      firstNonEmptyDuneString(row, "expected_signal", "expectedSignal"),
		ExpectedRoute:       firstNonEmptyDuneString(row, "expected_route", "expectedRoute"),
		SourceTxHash:        firstNonEmptyDuneString(row, "source_tx_hash", "sourceTxHash"),
		SourceBlockNumber:   firstNonEmptyDuneInt(row, "source_block_number", "sourceBlockNumber"),
		SourceTitle:         firstNonEmptyDuneString(row, "source_title", "sourceTitle"),
		SourceURL:           firstNonEmptyDuneString(row, "source_url", "sourceUrl"),
		Narrative:           firstNonEmptyDuneString(row, "narrative"),
		AnalystNote:         firstNonEmptyDuneString(row, "analyst_note", "analystNote"),
		Review:              normalizeDuneCandidateReview(row),
		Metadata:            metadata,
	}
	if strings.TrimSpace(value.CaseID) == "" {
		value.CaseID = deriveDuneBacktestCaseID(value)
	}
	if len(value.Metadata) == 0 {
		value.Metadata = nil
	}
	if strings.TrimSpace(value.Chain) == "" {
		return DuneBacktestCandidateRow{}, fmt.Errorf("chain is required")
	}
	return value, nil
}

func normalizeDuneGeneratedAt(executionEndedAt string, submittedAt string) string {
	for _, raw := range []string{strings.TrimSpace(executionEndedAt), strings.TrimSpace(submittedAt)} {
		if raw == "" {
			continue
		}
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func readDuneString(row map[string]any, key string) string {
	return firstNonEmptyDuneString(row, key)
}

func firstNonEmptyDuneString(row map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := row[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if trimmed := strings.TrimSpace(typed); trimmed != "" {
				return trimmed
			}
		case json.Number:
			if trimmed := strings.TrimSpace(typed.String()); trimmed != "" {
				return trimmed
			}
		case float64:
			return strconv.FormatFloat(typed, 'f', -1, 64)
		case int64:
			return strconv.FormatInt(typed, 10)
		case int:
			return strconv.Itoa(typed)
		}
	}
	return ""
}

func firstNonEmptyDuneInt(row map[string]any, keys ...string) int64 {
	for _, key := range keys {
		value, ok := row[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case int64:
			return typed
		case int:
			return int64(typed)
		case float64:
			return int64(typed)
		case json.Number:
			if parsed, err := typed.Int64(); err == nil {
				return parsed
			}
		case string:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func normalizeDuneMetadata(row map[string]any) map[string]any {
	known := map[string]struct{}{
		"case_id": {}, "caseId": {}, "dataset_id": {}, "datasetId": {}, "dedup_key": {}, "dedupKey": {},
		"chain": {}, "cohort": {}, "case_type": {}, "caseType": {}, "subject_address": {}, "subjectAddress": {},
		"entity_key": {}, "entityKey": {}, "subject_role": {}, "subjectRole": {}, "window_start_at": {}, "windowStartAt": {},
		"window_end_at": {}, "windowEndAt": {}, "observation_cutoff_at": {}, "observationCutoffAt": {}, "detection_deadline_at": {}, "detectionDeadlineAt": {},
		"expected_outcome": {}, "expectedOutcome": {}, "expected_signal": {}, "expectedSignal": {}, "expected_route": {}, "expectedRoute": {},
		"source_tx_hash": {}, "sourceTxHash": {}, "source_block_number": {}, "sourceBlockNumber": {}, "source_title": {}, "sourceTitle": {}, "source_url": {}, "sourceUrl": {},
		"narrative": {}, "analyst_note": {}, "analystNote": {},
		"curated_by": {}, "curatedBy": {}, "review_status": {}, "reviewStatus": {}, "case_ticket": {}, "caseTicket": {},
		"expected_high_signals": {}, "expectedHighSignals": {}, "expected_suppressed": {}, "expectedSuppressed": {},
	}
	metadata := make(map[string]any)
	for key, value := range row {
		if _, ok := known[key]; ok {
			continue
		}
		metadata[key] = value
	}
	return metadata
}

func normalizeDuneCandidateReview(row map[string]any) *DuneCandidateReview {
	review := DuneCandidateReview{
		CuratedBy:           firstNonEmptyDuneString(row, "curated_by", "curatedBy"),
		ReviewStatus:        firstNonEmptyDuneString(row, "review_status", "reviewStatus"),
		CaseTicket:          firstNonEmptyDuneString(row, "case_ticket", "caseTicket"),
		ExpectedHighSignals: normalizeDuneStringList(row, "expected_high_signals", "expectedHighSignals"),
		ExpectedSuppressed:  normalizeDuneStringList(row, "expected_suppressed", "expectedSuppressed"),
	}
	if strings.TrimSpace(review.CuratedBy) == "" &&
		strings.TrimSpace(review.ReviewStatus) == "" &&
		strings.TrimSpace(review.CaseTicket) == "" &&
		len(review.ExpectedHighSignals) == 0 &&
		len(review.ExpectedSuppressed) == 0 {
		return nil
	}
	return &review
}

func normalizeDuneStringList(row map[string]any, keys ...string) []string {
	for _, key := range keys {
		value, ok := row[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case []string:
			return compactStrings(typed)
		case []any:
			items := make([]string, 0, len(typed))
			for _, item := range typed {
				if str, ok := item.(string); ok {
					items = append(items, str)
				}
			}
			return compactStrings(items)
		case string:
			if strings.TrimSpace(typed) == "" {
				return nil
			}
			parts := strings.Split(typed, ",")
			return compactStrings(parts)
		}
	}
	return nil
}

func compactStrings(items []string) []string {
	next := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		next = append(next, trimmed)
	}
	return next
}

func deriveDuneBacktestCaseID(row DuneBacktestCandidateRow) string {
	anchor := strings.TrimSpace(row.SubjectAddress)
	if anchor == "" {
		anchor = strings.TrimSpace(row.EntityKey)
	}
	anchor = strings.ToLower(strings.ReplaceAll(anchor, ":", "-"))
	startAt := strings.TrimSpace(row.WindowStartAt)
	if parsed, err := time.Parse(time.RFC3339, startAt); err == nil {
		startAt = parsed.UTC().Format("2006-01-02")
	}
	parts := []string{
		strings.TrimSpace(row.Chain),
		strings.TrimSpace(row.Cohort),
		strings.TrimSpace(row.CaseType),
		anchor,
		startAt,
	}
	return strings.Join(compactStrings(parts), "-")
}
