package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
	"github.com/flowintel/flowintel/packages/intelligence"
)

type fakeFindingStore struct {
	entries []db.FindingEntry
}

func (s *fakeFindingStore) UpsertFinding(_ context.Context, entry db.FindingEntry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func (s *fakeFindingStore) ListFindings(context.Context, db.FindingsQuery) (domain.FindingsFeedPage, error) {
	return domain.FindingsFeedPage{}, nil
}

func (s *fakeFindingStore) ListWalletFindings(context.Context, db.WalletRef, int) ([]domain.Finding, error) {
	return nil, nil
}

type fakeWalletLabelReader struct {
	labels map[string]domain.WalletLabelSet
}

func (s *fakeWalletLabelReader) ReadWalletLabels(
	_ context.Context,
	refs []db.WalletRef,
) (map[string]domain.WalletLabelSet, error) {
	out := make(map[string]domain.WalletLabelSet, len(refs))
	for _, ref := range refs {
		key := strings.ToLower(string(ref.Chain)) + "|" + strings.ToLower(ref.Address)
		if labels, ok := s.labels[key]; ok {
			out[key] = labels
		}
	}
	return out, nil
}

func TestShadowExitSnapshotServiceAddsSuspectedMMHandoffFinding(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := ShadowExitSnapshotService{
		Signals:  &fakeSignalEventStore{},
		Labels:   &fakeWalletLabelReader{labels: map[string]domain.WalletLabelSet{"solana|so11111111111111111111111111111111111111112": {Inferred: []domain.WalletLabel{{Key: "inferred:market_maker:wintermute", Name: "Wintermute", Class: domain.WalletLabelClassInferred, EntityType: "market_maker"}}}}},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshot(context.Background(), intelligence.ShadowExitSignal{
		WalletID:          "wallet_fixture",
		Chain:             domain.ChainSolana,
		Address:           "So11111111111111111111111111111111111111112",
		ObservedAt:        "2026-03-20T09:10:11Z",
		BridgeTransfers:   2,
		CEXProximityCount: 1,
		FanOutCount:       1,
	})
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}

	if !hasFindingType(findings.entries, domain.FindingTypeSuspectedMMHandoff) {
		t.Fatalf("expected suspected_mm_handoff finding, got %#v", findings.entries)
	}

	entry := firstFindingByType(findings.entries, domain.FindingTypeSuspectedMMHandoff)
	if entry == nil {
		t.Fatalf("expected suspected_mm_handoff entry")
	}
	if got := stringSliceFromBundle(entry.Bundle, "observed_facts"); len(got) == 0 || !containsSubstring(got, "Bridge escape count") {
		t.Fatalf("expected bridge/path observed facts in bundle, got %#v", entry.Bundle)
	}
	if got := evidenceTypesFromBundle(entry.Bundle); !containsExact(got, "bridge_escape_count") || !containsExact(got, "cex_proximity_count") {
		t.Fatalf("expected flow evidence items in bundle, got %#v", got)
	}
	if got := nextWatchLabelsFromBundle(entry.Bundle); !containsSubstring(got, "Exchange-adjacent") {
		t.Fatalf("expected next_watch labels in bundle, got %#v", got)
	}
}

func TestShadowExitSnapshotServiceAddsTreasuryRedistributionFinding(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := ShadowExitSnapshotService{
		Signals:  &fakeSignalEventStore{},
		Labels:   &fakeWalletLabelReader{labels: map[string]domain.WalletLabelSet{"evm|0x1234567890abcdef1234567890abcdef12345678": {Inferred: []domain.WalletLabel{{Key: "inferred:treasury:treasury", Name: "Treasury", Class: domain.WalletLabelClassInferred, EntityType: "treasury"}}}}},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshot(context.Background(), intelligence.ShadowExitSignal{
		WalletID:          "wallet_fixture",
		Chain:             domain.ChainEVM,
		Address:           "0x1234567890abcdef1234567890abcdef12345678",
		ObservedAt:        "2026-03-20T09:10:11Z",
		BridgeTransfers:   1,
		CEXProximityCount: 0,
		FanOutCount:       1,
	})
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}

	if !hasFindingType(findings.entries, domain.FindingTypeTreasuryRedistribution) {
		t.Fatalf("expected treasury_redistribution finding, got %#v", findings.entries)
	}
}

func TestShadowExitSnapshotServiceSkipsLabelOnlyInterpretationFindingWithoutFlowPattern(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := ShadowExitSnapshotService{
		Signals:  &fakeSignalEventStore{},
		Labels:   &fakeWalletLabelReader{labels: map[string]domain.WalletLabelSet{"solana|so11111111111111111111111111111111111111112": {Inferred: []domain.WalletLabel{{Key: "inferred:market_maker:wintermute", Name: "Wintermute", Class: domain.WalletLabelClassInferred, EntityType: "market_maker"}}}}},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshot(context.Background(), intelligence.ShadowExitSignal{
		WalletID:          "wallet_fixture",
		Chain:             domain.ChainSolana,
		Address:           "So11111111111111111111111111111111111111112",
		ObservedAt:        "2026-03-20T09:10:11Z",
		BridgeTransfers:   0,
		CEXProximityCount: 0,
		FanOutCount:       0,
		OutflowRatio:      0.02,
	})
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}

	if hasFindingType(findings.entries, domain.FindingTypeSuspectedMMHandoff) {
		t.Fatalf("expected suspected_mm_handoff to be gated by flow pattern, got %#v", findings.entries)
	}
}

func TestClusterScoreSnapshotServiceAddsFundAdjacentFinding(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := ClusterScoreSnapshotService{
		Wallets: &fakeWalletStore{},
		Graphs: &fakeWalletGraphLoader{
			graph: domain.WalletGraph{
				Chain:          domain.ChainEVM,
				Address:        "0x1234567890abcdef1234567890abcdef12345678",
				DepthRequested: 1,
				DepthResolved:  1,
				Nodes: []domain.WalletGraphNode{
					{ID: "wallet_seed", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678", Label: "Seed"},
					{ID: "counterparty_1", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", Label: "Counterparty 1"},
					{ID: "counterparty_2", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xbcdefabcdefabcdefabcdefabcdefabcdefabcde", Label: "Counterparty 2"},
				},
				Edges: []domain.WalletGraphEdge{
					{SourceID: "wallet_seed", TargetID: "counterparty_1", Kind: domain.WalletGraphEdgeInteractedWith, ObservedAt: "2026-03-20T00:00:00Z", Weight: 2, CounterpartyCount: 2},
					{SourceID: "wallet_seed", TargetID: "counterparty_2", Kind: domain.WalletGraphEdgeInteractedWith, ObservedAt: "2026-03-20T00:00:00Z", Weight: 2, CounterpartyCount: 2},
				},
			},
		},
		Signals:  &fakeSignalEventStore{},
		Labels:   &fakeWalletLabelReader{labels: map[string]domain.WalletLabelSet{"evm|0x1234567890abcdef1234567890abcdef12345678": {Inferred: []domain.WalletLabel{{Key: "inferred:fund:multicoin", Name: "Fund", Class: domain.WalletLabelClassInferred, EntityType: "fund"}}}}},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshot(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, 1, "2026-03-20T01:02:03Z")
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}

	if !hasFindingType(findings.entries, domain.FindingTypeFundAdjacentActivity) {
		t.Fatalf("expected fund_adjacent_activity finding, got %#v", findings.entries)
	}

	entry := firstFindingByType(findings.entries, domain.FindingTypeFundAdjacentActivity)
	if entry == nil {
		t.Fatalf("expected fund_adjacent_activity entry")
	}
	if got := evidenceTypesFromBundle(entry.Bundle); !containsExact(got, "graph_neighborhood") {
		t.Fatalf("expected graph_neighborhood evidence, got %#v", got)
	}
	if got := nextWatchLabelsFromBundle(entry.Bundle); len(got) == 0 {
		t.Fatalf("expected next_watch wallet targets, got %#v", entry.Bundle)
	}
}

func TestFirstConnectionSnapshotServiceAddsHighConvictionEntryFinding(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := FirstConnectionSnapshotService{
		Signals:  &fakeSignalEventStore{},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshot(context.Background(), intelligence.FirstConnectionSignal{
		WalletID:                "wallet_first_connection",
		Chain:                   domain.ChainEVM,
		Address:                 "0x1234567890abcdef1234567890abcdef12345678",
		ObservedAt:              "2026-03-20T09:10:11Z",
		NewCommonEntries:        2,
		FirstSeenCounterparties: 3,
		HotFeedMentions:         1,
	})
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}

	if !hasFindingType(findings.entries, domain.FindingTypeHighConvictionEntry) {
		t.Fatalf("expected high_conviction_entry finding, got %#v", findings.entries)
	}

	entry := firstFindingByType(findings.entries, domain.FindingTypeHighConvictionEntry)
	if entry == nil {
		t.Fatalf("expected high_conviction_entry entry")
	}
	if got := evidenceTypesFromBundle(entry.Bundle); !containsExact(got, "new_common_entries") || !containsExact(got, "first_seen_counterparties") {
		t.Fatalf("expected convergence evidence in bundle, got %#v", got)
	}
}

func TestFirstConnectionSnapshotServiceSkipsHighConvictionEntryWithoutConvergenceEvidence(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := FirstConnectionSnapshotService{
		Signals:  &fakeSignalEventStore{},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshot(context.Background(), intelligence.FirstConnectionSignal{
		WalletID:                "wallet_first_connection",
		Chain:                   domain.ChainEVM,
		Address:                 "0x1234567890abcdef1234567890abcdef12345678",
		ObservedAt:              "2026-03-20T09:10:11Z",
		NewCommonEntries:        0,
		FirstSeenCounterparties: 0,
		HotFeedMentions:         0,
	})
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}

	if hasFindingType(findings.entries, domain.FindingTypeHighConvictionEntry) {
		t.Fatalf("expected high_conviction_entry to be gated by convergence evidence, got %#v", findings.entries)
	}
}

func hasFindingType(entries []db.FindingEntry, findingType domain.FindingType) bool {
	for _, entry := range entries {
		if entry.FindingType == findingType {
			return true
		}
	}
	return false
}

func firstFindingByType(entries []db.FindingEntry, findingType domain.FindingType) *db.FindingEntry {
	for i := range entries {
		if entries[i].FindingType == findingType {
			return &entries[i]
		}
	}
	return nil
}

func stringSliceFromBundle(bundle map[string]any, key string) []string {
	raw, ok := bundle[key].([]string)
	if ok {
		return raw
	}
	items, ok := bundle[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		value, ok := item.(string)
		if ok {
			out = append(out, value)
		}
	}
	return out
}

func evidenceTypesFromBundle(bundle map[string]any) []string {
	items, ok := bundle["evidence"].([]map[string]any)
	if ok {
		out := make([]string, 0, len(items))
		for _, item := range items {
			value, _ := item["type"].(string)
			if value != "" {
				out = append(out, value)
			}
		}
		return out
	}
	rawItems, ok := bundle["evidence"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		value, _ := item["type"].(string)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func nextWatchLabelsFromBundle(bundle map[string]any) []string {
	rawItems, ok := bundle["next_watch"].([]any)
	if !ok {
		if items, ok := bundle["next_watch"].([]map[string]any); ok {
			out := make([]string, 0, len(items))
			for _, item := range items {
				value, _ := item["label"].(string)
				if value != "" {
					out = append(out, value)
				}
			}
			return out
		}
		return nil
	}
	out := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		value, _ := item["label"].(string)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func containsSubstring(items []string, needle string) bool {
	for _, item := range items {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}

func containsExact(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}
