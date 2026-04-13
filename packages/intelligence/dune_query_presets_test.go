package intelligence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndValidateDuneBacktestQueryPresets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dune-query-presets.json")
	payload, err := json.Marshal(validDuneBacktestQueryPresetCollection())
	if err != nil {
		t.Fatalf("marshal preset collection: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write preset collection: %v", err)
	}

	loaded, err := LoadDuneBacktestQueryPresets(path)
	if err != nil {
		t.Fatalf("load preset collection: %v", err)
	}
	if err := ValidateDuneBacktestQueryPresets(loaded); err != nil {
		t.Fatalf("validate preset collection: %v", err)
	}

	summary := SummarizeDuneBacktestQueryPresets(path, loaded)
	if summary.PresetCount != 3 {
		t.Fatalf("expected 3 presets, got %+v", summary)
	}
	if summary.CohortCounts["known_negative"] != 2 || summary.CohortCounts["known_positive"] != 1 {
		t.Fatalf("unexpected cohort counts %+v", summary.CohortCounts)
	}
}

func TestValidateDuneBacktestQueryPresetsRejectsMissingRequiredParameter(t *testing.T) {
	t.Parallel()

	collection := validDuneBacktestQueryPresetCollection()
	delete(collection.Presets[0].Parameters, "min_bridge_usd")

	err := ValidateDuneBacktestQueryPresets(collection)
	if err == nil || !strings.Contains(err.Error(), "min_bridge_usd") {
		t.Fatalf("expected min_bridge_usd validation error, got %v", err)
	}
}

func TestFindDuneBacktestQueryPresetByName(t *testing.T) {
	t.Parallel()

	collection := validDuneBacktestQueryPresetCollection()
	preset, ok := FindDuneBacktestQueryPresetByName(collection, "bridge-return-default")
	if !ok {
		t.Fatalf("expected preset to be found")
	}
	if preset.QueryName != "qorvi_backtest_evm_known_negative_bridge_return_v1" {
		t.Fatalf("unexpected preset %+v", preset)
	}
}

func validDuneBacktestQueryPresetCollection() DuneBacktestQueryPresetCollection {
	return DuneBacktestQueryPresetCollection{
		Version: "2026-03-31",
		Presets: []DuneBacktestQueryPreset{
			{
				Name:            "bridge-return-default",
				QueryID:         4242,
				QueryName:       "qorvi_backtest_evm_known_negative_bridge_return_v1",
				SQLPath:         "queries/dune/backtest/01_bridge_return_negative.sql",
				Cohort:          "known_negative",
				CaseType:        "bridge_return",
				Chain:           "evm",
				CandidateOutput: "packages/intelligence/test/dune-backtest-candidates-evm-bridge-return-2026-03-31.json",
				Parameters: map[string]any{
					"window_start":                 "2026-03-01T00:00:00Z",
					"window_end":                   "2026-03-31T00:00:00Z",
					"min_bridge_usd":               25000,
					"max_return_hours":             48,
					"post_return_hours":            24,
					"max_post_return_recipients":   3,
					"max_post_return_outbound_usd": 50000,
					"limit":                        100,
					"source_url":                   "https://example.com/reviews/bridge-return",
				},
			},
			{
				Name:            "aggregator-routing-default",
				QueryID:         4243,
				QueryName:       "qorvi_backtest_evm_known_negative_aggregator_routing_v1",
				SQLPath:         "queries/dune/backtest/02_aggregator_routing_negative.sql",
				Cohort:          "known_negative",
				CaseType:        "aggregator_routing",
				Chain:           "evm",
				CandidateOutput: "packages/intelligence/test/dune-backtest-candidates-evm-aggregator-routing-2026-03-31.json",
				Parameters: map[string]any{
					"window_start":            "2026-03-01T00:00:00Z",
					"window_end":              "2026-03-31T00:00:00Z",
					"min_router_touch_count":  8,
					"min_unique_router_count": 2,
					"min_router_touch_ratio":  0.55,
					"limit":                   100,
					"source_url":              "https://example.com/reviews/aggregator-routing",
				},
			},
			{
				Name:            "smart-money-early-entry-default",
				QueryID:         4244,
				QueryName:       "qorvi_backtest_evm_known_positive_smart_money_early_entry_v1",
				SQLPath:         "queries/dune/backtest/03_smart_money_early_entry_positive.sql",
				Cohort:          "known_positive",
				CaseType:        "smart_money_early_entry",
				Chain:           "evm",
				CandidateOutput: "packages/intelligence/test/dune-backtest-candidates-evm-smart-money-early-entry-2026-03-31.json",
				Parameters: map[string]any{
					"window_start":          "2026-01-01T00:00:00Z",
					"window_end":            "2026-03-31T00:00:00Z",
					"min_entry_usd":         20000,
					"min_broader_wallets":   25,
					"min_lead_hours":        12,
					"hold_window_hours":     72,
					"min_subsequent_trades": 1,
					"limit":                 100,
					"source_url":            "https://example.com/reviews/smart-money",
				},
			},
		},
	}
}
