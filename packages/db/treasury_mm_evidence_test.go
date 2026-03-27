package db

import "testing"

func TestTreasuryMMPathHints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		pathKind       string
		wantSource     string
		wantDownstream string
		wantStrength   string
		wantConfidence string
	}{
		{
			name:           "mm routed candidate desk",
			pathKind:       "project_to_mm_routed_candidate_desk",
			wantSource:     "desk",
			wantStrength:   "routed_candidate",
			wantConfidence: "medium_high",
		},
		{
			name:           "mm adjacency router",
			pathKind:       "project_to_mm_adjacency_router",
			wantSource:     "router",
			wantStrength:   "adjacency",
			wantConfidence: "medium",
		},
		{
			name:           "treasury routed exchange",
			pathKind:       "treasury_external_market_adjacent_routed_exchange",
			wantDownstream: "exchange",
			wantStrength:   "routed_market_adjacent",
			wantConfidence: "medium_high",
		},
		{
			name:           "treasury direct dex",
			pathKind:       "treasury_external_market_adjacent_direct_dex",
			wantSource:     "dex",
			wantStrength:   "direct_market_adjacent",
			wantConfidence: "medium",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotSource, gotDownstream, gotStrength, gotConfidence := treasuryMMPathHints(tt.pathKind)
			if gotSource != tt.wantSource || gotDownstream != tt.wantDownstream || gotStrength != tt.wantStrength || gotConfidence != tt.wantConfidence {
				t.Fatalf("unexpected hints for %q: got (%q, %q, %q, %q)", tt.pathKind, gotSource, gotDownstream, gotStrength, gotConfidence)
			}
		})
	}
}

func TestConfidenceLadders(t *testing.T) {
	t.Parallel()

	if !(treasuryExternalMarketAdjacentConfidence("exchange", true) > treasuryExternalMarketAdjacentConfidence("router", false)) {
		t.Fatalf("expected routed exchange treasury evidence to be stronger than direct router adjacency")
	}
	if !(mmRoutedCandidateConfidence("desk") > mmAdjacencyConfidence("desk")) {
		t.Fatalf("expected routed MM desk candidate to be stronger than adjacency-only desk signal")
	}
}
