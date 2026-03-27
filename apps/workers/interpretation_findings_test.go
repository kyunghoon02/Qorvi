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

func (s *fakeFindingStore) GetFindingByID(_ context.Context, id string) (domain.Finding, error) {
	for _, entry := range s.entries {
		if strings.TrimSpace(entry.DedupKey) == strings.TrimSpace(id) {
			return domain.Finding{ID: id}, nil
		}
	}
	return domain.Finding{ID: strings.TrimSpace(id)}, nil
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

func TestShadowExitSnapshotServiceSkipsSuspectedMMHandoffWithoutTreasuryMMEvidence(t *testing.T) {
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

	if hasFindingType(findings.entries, domain.FindingTypeSuspectedMMHandoff) {
		t.Fatalf("expected suspected_mm_handoff to require treasury/MM evidence report, got %#v", findings.entries)
	}
}

func TestShadowExitSnapshotServiceSkipsTreasuryRedistributionWithoutTreasuryMMEvidence(t *testing.T) {
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

	if hasFindingType(findings.entries, domain.FindingTypeTreasuryRedistribution) {
		t.Fatalf("expected treasury_redistribution to require treasury/MM evidence report, got %#v", findings.entries)
	}
}

func TestShadowExitSnapshotServiceRunSnapshotForWalletAddsEvidenceBackedTreasuryAndMMFindings(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := ShadowExitSnapshotService{
		Wallets: &fakeWalletStore{},
		Candidates: &fakeShadowExitCandidateReader{
			metrics: db.ShadowExitCandidateMetrics{
				WalletID:                "wallet_fixture",
				Chain:                   domain.ChainEVM,
				Address:                 "0x1234567890abcdef1234567890abcdef12345678",
				WindowEnd:               time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				InboundTxCount:          1,
				OutboundTxCount:         4,
				FanOutCounterpartyCount: 3,
			},
		},
		TreasuryMM: &fakeTreasuryMMEvidenceStore{
			report: db.WalletTreasuryMMEvidenceReport{
				WalletID:         "wallet_fixture",
				Chain:            domain.ChainEVM,
				Address:          "0x1234567890abcdef1234567890abcdef12345678",
				WindowStartAt:    time.Date(2026, time.March, 19, 9, 10, 11, 0, time.UTC),
				WindowEndAt:      time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				HasTreasuryLabel: true,
				TreasuryFeatures: db.WalletTreasuryFeatures{
					AnchorMatchCount:                1,
					FanoutSignatureCount:            3,
					OperationalDistributionCount:    1,
					TreasuryToMarketPathCount:       1,
					TreasuryToExchangePathCount:     1,
					DistinctMarketCounterpartyCount: 1,
				},
				TreasuryPaths: []db.WalletTreasuryPathObservation{
					{
						TxHash:                 "0xtreasury",
						ObservedAt:             time.Date(2026, time.March, 20, 8, 0, 0, 0, time.UTC),
						PathKind:               "treasury_to_exchange_path",
						CounterpartyChain:      domain.ChainEVM,
						CounterpartyAddress:    "0xaaaa",
						CounterpartyLabel:      "Treasury Ops",
						CounterpartyEntityKey:  "entity:treasury",
						CounterpartyEntityType: "treasury",
						DownstreamChain:        domain.ChainEVM,
						DownstreamAddress:      "0xbbbb",
						DownstreamLabel:        "Exchange Sink",
						DownstreamEntityKey:    "entity:exchange",
						DownstreamEntityType:   "exchange",
						DownstreamTxHash:       "0xtreasurydown",
						Amount:                 "1250000",
						TokenSymbol:            "ETH",
						Confidence:             0.82,
					},
				},
				MMFeatures: db.WalletMMFeatures{
					MMAnchorMatchCount:            1,
					ProjectToMMPathCount:          1,
					ProjectToMMContactCount:       0,
					PostHandoffDistributionCount:  1,
					PostHandoffExchangeTouchCount: 1,
					InventoryRotationCount:        1,
				},
				MMPaths: []db.WalletMMPathObservation{
					{
						TxHash:                 "0xmm",
						ObservedAt:             time.Date(2026, time.March, 20, 8, 30, 0, 0, time.UTC),
						PathKind:               "post_handoff_exchange_distribution",
						CounterpartyChain:      domain.ChainEVM,
						CounterpartyAddress:    "0xcccc",
						CounterpartyLabel:      "MM Desk",
						CounterpartyEntityKey:  "entity:mm",
						CounterpartyEntityType: "market_maker",
						DownstreamChain:        domain.ChainEVM,
						DownstreamAddress:      "0xdddd",
						DownstreamLabel:        "Venue Route",
						DownstreamEntityKey:    "entity:venue",
						DownstreamEntityType:   "exchange",
						DownstreamTxHash:       "0xmmdown",
						Amount:                 "880000",
						TokenSymbol:            "ETH",
						Confidence:             0.84,
					},
				},
			},
		},
		Signals: &fakeSignalEventStore{},
		Labels: &fakeWalletLabelReader{labels: map[string]domain.WalletLabelSet{
			"evm|0x1234567890abcdef1234567890abcdef12345678": {
				Inferred: []domain.WalletLabel{
					{Key: "inferred:treasury:treasury", Name: "Treasury", Class: domain.WalletLabelClassInferred, EntityType: "treasury"},
					{Key: "inferred:fund:fund", Name: "Fund", Class: domain.WalletLabelClassInferred, EntityType: "fund"},
				},
			},
		}},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "")
	if err != nil {
		t.Fatalf("RunSnapshotForWallet returned error: %v", err)
	}

	if !hasFindingType(findings.entries, domain.FindingTypeTreasuryRedistribution) {
		t.Fatalf("expected treasury_redistribution finding, got %#v", findings.entries)
	}
	if !hasFindingType(findings.entries, domain.FindingTypeSuspectedMMHandoff) {
		t.Fatalf("expected suspected_mm_handoff finding, got %#v", findings.entries)
	}
	treasuryEntry := lastFindingByType(findings.entries, domain.FindingTypeTreasuryRedistribution)
	if treasuryEntry == nil || !evidenceBundleHasMetadataKey(treasuryEntry.Bundle, "entityRef") || !evidenceBundleHasMetadataKey(treasuryEntry.Bundle, "downstreamRef") {
		t.Fatalf("expected treasury finding evidence to include entity/downstream refs, got %#v", treasuryEntry)
	}
	mmEntry := lastFindingByType(findings.entries, domain.FindingTypeSuspectedMMHandoff)
	if mmEntry == nil || !nextWatchHasMetadataKey(mmEntry.Bundle, "pathRef") {
		t.Fatalf("expected mm finding next_watch to include path refs, got %#v", mmEntry)
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

func TestShadowExitSnapshotServiceSkipsMMHandoffWithoutRootAnchor(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := ShadowExitSnapshotService{
		Wallets: &fakeWalletStore{},
		Candidates: &fakeShadowExitCandidateReader{
			metrics: db.ShadowExitCandidateMetrics{
				WalletID:                "wallet_fixture",
				Chain:                   domain.ChainEVM,
				Address:                 "0x1234567890abcdef1234567890abcdef12345678",
				WindowEnd:               time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				InboundTxCount:          1,
				OutboundTxCount:         4,
				FanOutCounterpartyCount: 2,
			},
		},
		TreasuryMM: &fakeTreasuryMMEvidenceStore{
			report: db.WalletTreasuryMMEvidenceReport{
				WalletID:      "wallet_fixture",
				Chain:         domain.ChainEVM,
				Address:       "0x1234567890abcdef1234567890abcdef12345678",
				WindowStartAt: time.Date(2026, time.March, 19, 9, 10, 11, 0, time.UTC),
				WindowEndAt:   time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				MMFeatures: db.WalletMMFeatures{
					MMAnchorMatchCount:           1,
					ProjectToMMPathCount:         1,
					PostHandoffDistributionCount: 1,
					InventoryRotationCount:       1,
				},
			},
		},
		Signals:  &fakeSignalEventStore{},
		Labels:   &fakeWalletLabelReader{},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "")
	if err != nil {
		t.Fatalf("RunSnapshotForWallet returned error: %v", err)
	}

	if hasFindingType(findings.entries, domain.FindingTypeSuspectedMMHandoff) {
		t.Fatalf("expected suspected_mm_handoff to require a root fund/treasury anchor, got %#v", findings.entries)
	}
}

func TestShadowExitSnapshotServiceSkipsTreasuryRedistributionWithoutOperationalFanoutAndStrongMarketPath(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := ShadowExitSnapshotService{
		Wallets: &fakeWalletStore{},
		Candidates: &fakeShadowExitCandidateReader{
			metrics: db.ShadowExitCandidateMetrics{
				WalletID:                "wallet_fixture",
				Chain:                   domain.ChainEVM,
				Address:                 "0x1234567890abcdef1234567890abcdef12345678",
				WindowEnd:               time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				InboundTxCount:          1,
				OutboundTxCount:         2,
				FanOutCounterpartyCount: 1,
			},
		},
		TreasuryMM: &fakeTreasuryMMEvidenceStore{
			report: db.WalletTreasuryMMEvidenceReport{
				WalletID:         "wallet_fixture",
				Chain:            domain.ChainEVM,
				Address:          "0x1234567890abcdef1234567890abcdef12345678",
				WindowStartAt:    time.Date(2026, time.March, 19, 9, 10, 11, 0, time.UTC),
				WindowEndAt:      time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				HasTreasuryLabel: true,
				TreasuryFeatures: db.WalletTreasuryFeatures{
					AnchorMatchCount:                 1,
					FanoutSignatureCount:             1,
					OperationalDistributionCount:     0,
					OperationalOnlyDistributionCount: 1,
					ExternalOpsDistributionCount:     1,
					RebalanceDiscountCount:           1,
					TreasuryToMarketPathCount:        1,
					TreasuryToBridgePathCount:        1,
					DistinctMarketCounterpartyCount:  1,
				},
			},
		},
		Signals:  &fakeSignalEventStore{},
		Labels:   &fakeWalletLabelReader{},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "")
	if err != nil {
		t.Fatalf("RunSnapshotForWallet returned error: %v", err)
	}

	if hasFindingType(findings.entries, domain.FindingTypeTreasuryRedistribution) {
		t.Fatalf("expected treasury_redistribution to require operational fanout plus stronger market path, got %#v", findings.entries)
	}
}

func TestShadowExitSnapshotServiceSkipsMMHandoffWithoutPostHandoffEvidence(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := ShadowExitSnapshotService{
		Wallets: &fakeWalletStore{},
		Candidates: &fakeShadowExitCandidateReader{
			metrics: db.ShadowExitCandidateMetrics{
				WalletID:                "wallet_fixture",
				Chain:                   domain.ChainEVM,
				Address:                 "0x1234567890abcdef1234567890abcdef12345678",
				WindowEnd:               time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				InboundTxCount:          1,
				OutboundTxCount:         4,
				FanOutCounterpartyCount: 2,
			},
		},
		TreasuryMM: &fakeTreasuryMMEvidenceStore{
			report: db.WalletTreasuryMMEvidenceReport{
				WalletID:      "wallet_fixture",
				Chain:         domain.ChainEVM,
				Address:       "0x1234567890abcdef1234567890abcdef12345678",
				WindowStartAt: time.Date(2026, time.March, 19, 9, 10, 11, 0, time.UTC),
				WindowEndAt:   time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				HasFundLabel:  true,
				MMFeatures: db.WalletMMFeatures{
					MMAnchorMatchCount:           1,
					ProjectToMMPathCount:         1,
					ProjectToMMContactCount:      1,
					ProjectToMMAdjacencyCount:    1,
					PostHandoffDistributionCount: 0,
					PostHandoffBridgeTouchCount:  0,
					InventoryRotationCount:       1,
					RepeatMMCounterpartyCount:    1,
				},
			},
		},
		Signals:  &fakeSignalEventStore{},
		Labels:   &fakeWalletLabelReader{},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "")
	if err != nil {
		t.Fatalf("RunSnapshotForWallet returned error: %v", err)
	}

	if hasFindingType(findings.entries, domain.FindingTypeSuspectedMMHandoff) {
		t.Fatalf("expected suspected_mm_handoff to require post-handoff evidence, got %#v", findings.entries)
	}
}

func TestShadowExitSnapshotServiceSkipsTreasuryRedistributionForBridgeOnlyWeakMarketPaths(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := ShadowExitSnapshotService{
		Wallets: &fakeWalletStore{},
		Candidates: &fakeShadowExitCandidateReader{
			metrics: db.ShadowExitCandidateMetrics{
				WalletID:                "wallet_fixture",
				Chain:                   domain.ChainEVM,
				Address:                 "0x1234567890abcdef1234567890abcdef12345678",
				WindowEnd:               time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				InboundTxCount:          1,
				OutboundTxCount:         4,
				FanOutCounterpartyCount: 2,
			},
		},
		TreasuryMM: &fakeTreasuryMMEvidenceStore{
			report: db.WalletTreasuryMMEvidenceReport{
				WalletID:         "wallet_fixture",
				Chain:            domain.ChainEVM,
				Address:          "0x1234567890abcdef1234567890abcdef12345678",
				WindowStartAt:    time.Date(2026, time.March, 19, 9, 10, 11, 0, time.UTC),
				WindowEndAt:      time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				HasTreasuryLabel: true,
				TreasuryFeatures: db.WalletTreasuryFeatures{
					AnchorMatchCount:                 1,
					FanoutSignatureCount:             2,
					OperationalDistributionCount:     1,
					OperationalOnlyDistributionCount: 1,
					ExternalOpsDistributionCount:     1,
					TreasuryToMarketPathCount:        1,
					TreasuryToBridgePathCount:        1,
					DistinctMarketCounterpartyCount:  1,
				},
			},
		},
		Signals:  &fakeSignalEventStore{},
		Labels:   &fakeWalletLabelReader{},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "")
	if err != nil {
		t.Fatalf("RunSnapshotForWallet returned error: %v", err)
	}

	if hasFindingType(findings.entries, domain.FindingTypeTreasuryRedistribution) {
		t.Fatalf("expected bridge-only weak treasury market path to be suppressed, got %#v", findings.entries)
	}
}

func TestShadowExitSnapshotServiceSkipsMMHandoffForBridgeOnlyPostHandoffWithoutDistributionEvidence(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := ShadowExitSnapshotService{
		Wallets: &fakeWalletStore{},
		Candidates: &fakeShadowExitCandidateReader{
			metrics: db.ShadowExitCandidateMetrics{
				WalletID:                "wallet_fixture",
				Chain:                   domain.ChainEVM,
				Address:                 "0x1234567890abcdef1234567890abcdef12345678",
				WindowEnd:               time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				InboundTxCount:          1,
				OutboundTxCount:         4,
				FanOutCounterpartyCount: 2,
			},
		},
		TreasuryMM: &fakeTreasuryMMEvidenceStore{
			report: db.WalletTreasuryMMEvidenceReport{
				WalletID:      "wallet_fixture",
				Chain:         domain.ChainEVM,
				Address:       "0x1234567890abcdef1234567890abcdef12345678",
				WindowStartAt: time.Date(2026, time.March, 19, 9, 10, 11, 0, time.UTC),
				WindowEndAt:   time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				HasFundLabel:  true,
				MMFeatures: db.WalletMMFeatures{
					MMAnchorMatchCount:           1,
					ProjectToMMPathCount:         0,
					ProjectToMMContactCount:      1,
					ProjectToMMAdjacencyCount:    1,
					PostHandoffDistributionCount: 1,
					PostHandoffBridgeTouchCount:  1,
					InventoryRotationCount:       0,
					RepeatMMCounterpartyCount:    1,
				},
			},
		},
		Signals:  &fakeSignalEventStore{},
		Labels:   &fakeWalletLabelReader{},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "")
	if err != nil {
		t.Fatalf("RunSnapshotForWallet returned error: %v", err)
	}

	if hasFindingType(findings.entries, domain.FindingTypeSuspectedMMHandoff) {
		t.Fatalf("expected bridge-only post-handoff without rotation/repeat to be suppressed, got %#v", findings.entries)
	}
}

func TestShadowExitSnapshotServiceSkipsMMHandoffForAdjacencyOnlyContact(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	service := ShadowExitSnapshotService{
		Wallets: &fakeWalletStore{},
		Candidates: &fakeShadowExitCandidateReader{
			metrics: db.ShadowExitCandidateMetrics{
				WalletID:                "wallet_fixture",
				Chain:                   domain.ChainEVM,
				Address:                 "0x1234567890abcdef1234567890abcdef12345678",
				WindowEnd:               time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				InboundTxCount:          1,
				OutboundTxCount:         4,
				FanOutCounterpartyCount: 2,
			},
		},
		TreasuryMM: &fakeTreasuryMMEvidenceStore{
			report: db.WalletTreasuryMMEvidenceReport{
				WalletID:      "wallet_fixture",
				Chain:         domain.ChainEVM,
				Address:       "0x1234567890abcdef1234567890abcdef12345678",
				WindowStartAt: time.Date(2026, time.March, 19, 9, 10, 11, 0, time.UTC),
				WindowEndAt:   time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
				HasFundLabel:  true,
				MMFeatures: db.WalletMMFeatures{
					MMAnchorMatchCount:              1,
					ProjectToMMPathCount:            0,
					ProjectToMMContactCount:         1,
					ProjectToMMRoutedCandidateCount: 0,
					ProjectToMMAdjacencyCount:       1,
					PostHandoffDistributionCount:    1,
					PostHandoffExchangeTouchCount:   1,
					InventoryRotationCount:          1,
					RepeatMMCounterpartyCount:       2,
				},
			},
		},
		Signals:  &fakeSignalEventStore{},
		Labels:   &fakeWalletLabelReader{},
		Findings: findings,
		Cache:    &fakeWalletSummaryCache{},
		JobRuns:  &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "")
	if err != nil {
		t.Fatalf("RunSnapshotForWallet returned error: %v", err)
	}

	if hasFindingType(findings.entries, domain.FindingTypeSuspectedMMHandoff) {
		t.Fatalf("expected adjacency-only MM contact to be suppressed, got %#v", findings.entries)
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
	wallets := &fakeWalletStore{}
	candidates := &fakeFirstConnectionCandidateReader{
		metrics: db.FirstConnectionCandidateMetrics{
			WalletID:                          "wallet_first_connection",
			Chain:                             domain.ChainEVM,
			Address:                           "0x1234567890abcdef1234567890abcdef12345678",
			WindowEnd:                         time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
			FirstSeenCounterparties:           3,
			NewCommonEntries:                  2,
			HotFeedMentions:                   1,
			QualityWalletOverlapCount:         1,
			SustainedOverlapCounterpartyCount: 1,
			StrongLeadCounterpartyCount:       1,
			FirstEntryBeforeCrowdingCount:     1,
			BestLeadHoursBeforePeers:          18,
			PersistenceAfterEntryProxyCount:   1,
			TopCounterparties: []db.FirstConnectionCandidateCounterparty{
				{
					Chain:                domain.ChainEVM,
					Address:              "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
					InteractionCount:     2,
					FirstActivityAt:      time.Date(2026, time.March, 20, 6, 10, 11, 0, time.UTC),
					LatestActivityAt:     time.Date(2026, time.March, 20, 9, 0, 0, 0, time.UTC),
					LeadHoursBeforePeers: 18,
					PeerWalletCount:      2,
					PeerTxCount:          3,
				},
			},
		},
	}
	service := FirstConnectionSnapshotService{
		Wallets:    wallets,
		Candidates: candidates,
		Signals:    &fakeSignalEventStore{},
		Findings:   findings,
		Cache:      &fakeWalletSummaryCache{},
		JobRuns:    &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "2026-03-20T09:10:11Z")
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
	if got := evidenceTypesFromBundle(entry.Bundle); !containsExact(got, "quality_wallet_overlap_count") || !containsExact(got, "first_entry_before_crowding_count") || !containsExact(got, "persistence_after_entry_proxy_count") {
		t.Fatalf("expected convergence evidence in bundle, got %#v", got)
	}
}

func TestFirstConnectionSnapshotServiceBoostsHighConvictionFindingWithSustainedOutcome(t *testing.T) {
	t.Parallel()

	findings := &fakeFindingStore{}
	wallets := &fakeWalletStore{}
	candidates := &fakeFirstConnectionCandidateReader{
		metrics: db.FirstConnectionCandidateMetrics{
			WalletID:                          "wallet_first_connection",
			Chain:                             domain.ChainEVM,
			Address:                           "0x1234567890abcdef1234567890abcdef12345678",
			WindowEnd:                         time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
			FirstSeenCounterparties:           3,
			NewCommonEntries:                  2,
			HotFeedMentions:                   1,
			QualityWalletOverlapCount:         1,
			SustainedOverlapCounterpartyCount: 1,
			StrongLeadCounterpartyCount:       1,
			FirstEntryBeforeCrowdingCount:     1,
			BestLeadHoursBeforePeers:          18,
			PersistenceAfterEntryProxyCount:   1,
			TopCounterparties: []db.FirstConnectionCandidateCounterparty{
				{
					Chain:                domain.ChainEVM,
					Address:              "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
					InteractionCount:     2,
					FirstActivityAt:      time.Date(2026, time.March, 20, 6, 10, 11, 0, time.UTC),
					LatestActivityAt:     time.Date(2026, time.March, 20, 9, 0, 0, 0, time.UTC),
					LeadHoursBeforePeers: 18,
					PeerWalletCount:      2,
					PeerTxCount:          3,
				},
			},
		},
	}
	entryFeatures := &fakeWalletEntryFeaturesStore{
		priorSnapshot: db.WalletEntryFeaturesSnapshot{
			WalletID:                "wallet_first_connection",
			Chain:                   domain.ChainEVM,
			Address:                 "0x1234567890abcdef1234567890abcdef12345678",
			WindowStartAt:           time.Date(2026, time.March, 19, 9, 10, 11, 0, time.UTC),
			WindowEndAt:             time.Date(2026, time.March, 20, 8, 10, 11, 0, time.UTC),
			TopCounterparties:       []db.WalletEntryFeatureCounterparty{{Chain: domain.ChainEVM, Address: "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed"}},
			HoldingPersistenceState: "",
		},
		followThrough: db.WalletEntryFeatureFollowThrough{
			PostWindowFollowThroughCount:  2,
			MaxPostWindowPersistenceHours: 36,
		},
	}
	service := FirstConnectionSnapshotService{
		Wallets:       wallets,
		Candidates:    candidates,
		EntryFeatures: entryFeatures,
		Signals:       &fakeSignalEventStore{},
		Findings:      findings,
		Cache:         &fakeWalletSummaryCache{},
		JobRuns:       &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 23, 9, 10, 11, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "2026-03-23T09:10:11Z")
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}

	entry := firstFindingByType(findings.entries, domain.FindingTypeHighConvictionEntry)
	if entry == nil {
		t.Fatalf("expected high_conviction_entry entry")
	}
	if entry.Confidence < 0.8 {
		t.Fatalf("expected sustained outcome to boost confidence, got %#v", entry)
	}
	labels := nextWatchLabelsFromBundle(entry.Bundle)
	if !containsSubstring(labels, "(sustained)") {
		t.Fatalf("expected sustained next_watch label, got %#v", labels)
	}
	if !nextWatchHasMetadataKey(entry.Bundle, "holdingPersistenceState") {
		t.Fatalf("expected sustained next_watch metadata, got %#v", entry.Bundle)
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

func lastFindingByType(entries []db.FindingEntry, findingType domain.FindingType) *db.FindingEntry {
	for i := len(entries) - 1; i >= 0; i-- {
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

func evidenceBundleHasMetadataKey(bundle map[string]any, key string) bool {
	rawItems, ok := bundle["evidence"].([]any)
	if !ok {
		if items, ok := bundle["evidence"].([]map[string]any); ok {
			for _, item := range items {
				if metadata, ok := item["metadata"].(map[string]any); ok {
					if _, exists := metadata[key]; exists {
						return true
					}
				}
			}
		}
		return false
	}
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		metadata, ok := item["metadata"].(map[string]any)
		if !ok {
			continue
		}
		if _, exists := metadata[key]; exists {
			return true
		}
	}
	return false
}

func nextWatchHasMetadataKey(bundle map[string]any, key string) bool {
	rawItems, ok := bundle["next_watch"].([]any)
	if !ok {
		if items, ok := bundle["next_watch"].([]map[string]any); ok {
			for _, item := range items {
				if metadata, ok := item["metadata"].(map[string]any); ok {
					if _, exists := metadata[key]; exists {
						return true
					}
				}
			}
		}
		return false
	}
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		metadata, ok := item["metadata"].(map[string]any)
		if !ok {
			continue
		}
		if _, exists := metadata[key]; exists {
			return true
		}
	}
	return false
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
