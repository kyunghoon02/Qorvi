package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

func recordWalletFinding(
	ctx context.Context,
	store db.FindingStore,
	entry db.FindingEntry,
) error {
	if store == nil {
		return nil
	}
	return store.UpsertFinding(ctx, entry)
}

func buildFindingEvidence(evidence []domain.Evidence) []map[string]any {
	out := make([]map[string]any, 0, len(evidence))
	for _, item := range evidence {
		out = append(out, map[string]any{
			"type":        string(item.Kind),
			"value":       item.Label,
			"confidence":  item.Confidence,
			"observed_at": item.ObservedAt,
			"metadata":    item.Metadata,
		})
	}
	return out
}

func buildFindingEvidenceItem(
	kind string,
	value string,
	confidence float64,
	metadata map[string]any,
) map[string]any {
	item := map[string]any{
		"type":       kind,
		"value":      value,
		"confidence": confidence,
	}
	if len(metadata) > 0 {
		item["metadata"] = metadata
	}
	return item
}

func buildNextWatchTargets(targets []domain.NextWatchTarget) []map[string]any {
	out := make([]map[string]any, 0, len(targets))
	for _, item := range targets {
		target := map[string]any{
			"subject_type": string(item.SubjectType),
		}
		if item.Chain != "" {
			target["chain"] = string(item.Chain)
		}
		if strings.TrimSpace(item.Address) != "" {
			target["address"] = strings.TrimSpace(item.Address)
		}
		if strings.TrimSpace(item.Token) != "" {
			target["token"] = strings.TrimSpace(item.Token)
		}
		if strings.TrimSpace(item.Label) != "" {
			target["label"] = strings.TrimSpace(item.Label)
		}
		if len(item.Metadata) > 0 {
			target["metadata"] = item.Metadata
		}
		out = append(out, target)
	}
	return out
}

func bundleCoverage(observedAt string, windowDays int) (time.Time, *time.Time, *time.Time) {
	end := time.Now().UTC()
	if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(observedAt)); err == nil {
		end = parsed.UTC()
	}
	start := end.Add(-time.Duration(windowDays) * 24 * time.Hour)
	return end, &start, &end
}

func clusterScoreFindingEntry(report ClusterScoreSnapshotReport, score domain.Score) db.FindingEntry {
	observedAt, coverageStartAt, coverageEndAt := bundleCoverage(report.ObservedAt, 30)
	return db.FindingEntry{
		FindingType:        domain.FindingTypeSmartMoneyConvergence,
		WalletID:           strings.TrimSpace(report.WalletID),
		SubjectType:        domain.FindingSubjectWallet,
		SubjectChain:       domain.Chain(strings.TrimSpace(report.Chain)),
		SubjectAddress:     strings.TrimSpace(report.Address),
		SubjectKey:         strings.ToLower(strings.TrimSpace(report.Chain) + ":" + strings.TrimSpace(report.Address)),
		SubjectLabel:       "Wallet",
		Confidence:         findingConfidenceFromScore(score),
		ImportanceScore:    float64(score.Value) / 100,
		Summary:            fmt.Sprintf("Behavior cohort overlap and repeated counterparties suggest coordinated smart money activity around %s.", strings.TrimSpace(report.Address)),
		DedupKey:           fmt.Sprintf("finding:%s:%s:%s", domain.FindingTypeSmartMoneyConvergence, strings.TrimSpace(report.WalletID), report.ObservedAt),
		Status:             "active",
		ObservedAt:         observedAt,
		CoverageStartAt:    coverageStartAt,
		CoverageEndAt:      coverageEndAt,
		CoverageWindowDays: 30,
		Bundle: map[string]any{
			"importance_reason": []string{
				"Repeated overlap with counterparties and neighborhood density increased cluster conviction.",
			},
			"observed_facts": []string{
				fmt.Sprintf("Cluster score snapshot rated %s at %d.", score.Rating, score.Value),
			},
			"inferred_interpretations": []string{
				"This wallet is moving with a behavior cohort rather than acting in isolation.",
			},
			"evidence": buildFindingEvidence(score.Evidence),
		},
	}
}

func shadowExitFindingEntries(
	report ShadowExitSnapshotReport,
	score domain.Score,
	evidenceReport *db.WalletBridgeExchangeEvidenceReport,
) []db.FindingEntry {
	observedAt, coverageStartAt, coverageEndAt := bundleCoverage(report.ObservedAt, 30)
	base := db.FindingEntry{
		WalletID:           strings.TrimSpace(report.WalletID),
		SubjectType:        domain.FindingSubjectWallet,
		SubjectChain:       domain.Chain(strings.TrimSpace(report.Chain)),
		SubjectAddress:     strings.TrimSpace(report.Address),
		SubjectKey:         strings.ToLower(strings.TrimSpace(report.Chain) + ":" + strings.TrimSpace(report.Address)),
		SubjectLabel:       "Wallet",
		Confidence:         findingConfidenceFromScore(score),
		ImportanceScore:    float64(score.Value) / 100,
		Status:             "active",
		ObservedAt:         observedAt,
		CoverageStartAt:    coverageStartAt,
		CoverageEndAt:      coverageEndAt,
		CoverageWindowDays: 30,
	}

	commonEvidence := buildFindingEvidence(score.Evidence)
	if evidenceReport != nil {
		commonEvidence = append(commonEvidence, buildBridgeExchangeFindingEvidence(*evidenceReport)...)
	}
	commonBundle := map[string]any{
		"evidence": commonEvidence,
	}

	out := make([]db.FindingEntry, 0, 2)
	out = append(out, db.FindingEntry{
		FindingType: domain.FindingTypeExitPreparation,
		Summary:     fmt.Sprintf("Distribution and exit risk is rising for %s based on bridge, fanout, and exchange-proximity signals.", strings.TrimSpace(report.Address)),
		DedupKey:    fmt.Sprintf("finding:%s:%s:%s", domain.FindingTypeExitPreparation, strings.TrimSpace(report.WalletID), report.ObservedAt),
		Bundle: mergeBundle(commonBundle, map[string]any{
			"importance_reason": []string{
				"Recent outflow behavior increased the probability of distribution or exit preparation.",
			},
			"observed_facts": []string{
				fmt.Sprintf("Shadow exit risk rated %s at %d.", score.Rating, score.Value),
				fmt.Sprintf("Bridge escape count: %d. CEX proximity count: %d.", report.BridgeEscapeCount, report.CEXProximityCount),
				fmt.Sprintf("Bridge share %.2f, exchange share %.2f, deposit-like paths %d.", report.BridgeOutflowShare, report.ExchangeOutflowShare, report.DepositLikePathCount),
			},
			"inferred_interpretations": []string{
				"Observed flow looks more like preparation to distribute than passive treasury movement.",
			},
		}),
	}, base)

	if shouldEmitCrossChainRotation(report, evidenceReport) {
		nextWatch := bridgeFindingNextWatch(evidenceReport)
		out = append(out, db.FindingEntry{
			FindingType: domain.FindingTypeCrossChainRotation,
			Summary:     fmt.Sprintf("Cross-chain rotation is likely underway for %s after bridge-linked outflows.", strings.TrimSpace(report.Address)),
			DedupKey:    fmt.Sprintf("finding:%s:%s:%s", domain.FindingTypeCrossChainRotation, strings.TrimSpace(report.WalletID), report.ObservedAt),
			Bundle: mergeBundle(commonBundle, map[string]any{
				"importance_reason": []string{
					"Bridge-linked outflows often precede rotation into new venues or assets.",
				},
				"observed_facts": []string{
					fmt.Sprintf("Bridge escape count reached %d within the indexed window.", report.BridgeEscapeCount),
					fmt.Sprintf("Confirmed bridge destinations: %d. Bridge recurrence days: %d.", report.BridgeConfirmedDestinationCount, report.BridgeRecurrenceDays),
				},
				"inferred_interpretations": []string{
					"The wallet may be repositioning cross-chain rather than exiting outright.",
				},
				"next_watch": buildNextWatchTargets(nextWatch),
			}),
		}, base)
	}
	if shouldEmitExchangePressure(report, evidenceReport) {
		nextWatch := exchangeFindingNextWatch(evidenceReport)
		out = append(out, db.FindingEntry{
			FindingType: domain.FindingTypeCEXDepositPressure,
			Summary:     fmt.Sprintf("Exchange deposit pressure is increasing for %s.", strings.TrimSpace(report.Address)),
			DedupKey:    fmt.Sprintf("finding:%s:%s:%s", domain.FindingTypeCEXDepositPressure, strings.TrimSpace(report.WalletID), report.ObservedAt),
			Bundle: mergeBundle(commonBundle, map[string]any{
				"importance_reason": []string{
					"Rising exchange proximity can precede inventory distribution and market pressure.",
				},
				"observed_facts": []string{
					fmt.Sprintf("CEX proximity count reached %d within the indexed window.", report.CEXProximityCount),
					fmt.Sprintf("Deposit-like paths: %d. Exchange recurrence days: %d.", report.DepositLikePathCount, report.ExchangeRecurrenceDays),
				},
				"inferred_interpretations": []string{
					"Some recent flow resembles deposit-like paths into exchange-adjacent addresses.",
				},
				"next_watch": buildNextWatchTargets(nextWatch),
			}),
		}, base)
	}

	return out
}

func firstConnectionFindingEntry(report FirstConnectionSnapshotReport, score domain.Score) db.FindingEntry {
	observedAt, coverageStartAt, coverageEndAt := bundleCoverage(report.ObservedAt, 30)
	evidence := buildFindingEvidence(score.Evidence)
	evidence = append(evidence,
		buildFindingEvidenceItem(
			"quality_wallet_overlap_count",
			fmt.Sprintf("%d", report.QualityWalletOverlapCount),
			findingConfidenceFromScore(score),
			map[string]any{
				"repeatEarlyEntrySuccess": report.RepeatEarlyEntrySuccess,
			},
		),
		buildFindingEvidenceItem(
			"repeat_early_entry_success",
			fmt.Sprintf("%t", report.RepeatEarlyEntrySuccess),
			findingConfidenceFromScore(score),
			map[string]any{
				"newCommonEntries":            report.NewCommonEntries,
				"firstSeenCounterparties":     report.FirstSeenCounterparties,
				"hotFeedMentions":             report.HotFeedMentions,
				"historicalSustainedOutcomes": report.HistoricalSustainedOutcomeCount,
			},
		),
		buildFindingEvidenceItem(
			"first_entry_before_crowding_count",
			fmt.Sprintf("%d", report.FirstEntryBeforeCrowdingCount),
			findingConfidenceFromScore(score),
			map[string]any{
				"bestLeadHoursBeforePeers": report.BestLeadHoursBeforePeers,
			},
		),
		buildFindingEvidenceItem(
			"persistence_after_entry_proxy_count",
			fmt.Sprintf("%d", report.PersistenceAfterEntryProxyCount),
			findingConfidenceFromScore(score),
			map[string]any{
				"repeatEarlyEntrySuccess": report.RepeatEarlyEntrySuccess,
			},
		),
	)
	for _, item := range report.TopCounterparties {
		evidence = append(evidence, buildFindingEvidenceItem(
			"top_counterparty_overlap",
			firstNonEmpty(item.Address, "counterparty"),
			findingConfidenceFromScore(score),
			map[string]any{
				"chain":                item.Chain,
				"interactionCount":     item.InteractionCount,
				"peerWalletCount":      item.PeerWalletCount,
				"peerTxCount":          item.PeerTxCount,
				"firstActivityAt":      item.FirstActivityAt,
				"latestActivityAt":     item.LatestActivityAt,
				"leadHoursBeforePeers": item.LeadHoursBeforePeers,
			},
		))
	}
	return db.FindingEntry{
		FindingType:        domain.FindingTypeSmartMoneyConvergence,
		WalletID:           strings.TrimSpace(report.WalletID),
		SubjectType:        domain.FindingSubjectWallet,
		SubjectChain:       domain.Chain(strings.TrimSpace(report.Chain)),
		SubjectAddress:     strings.TrimSpace(report.Address),
		SubjectKey:         strings.ToLower(strings.TrimSpace(report.Chain) + ":" + strings.TrimSpace(report.Address)),
		SubjectLabel:       "Wallet",
		Confidence:         findingConfidenceFromScore(score),
		ImportanceScore:    float64(score.Value) / 100,
		Summary:            fmt.Sprintf("Early convergence is forming around %s through newly shared counterparties.", strings.TrimSpace(report.Address)),
		DedupKey:           fmt.Sprintf("finding:%s:%s:%s", domain.FindingTypeSmartMoneyConvergence, strings.TrimSpace(report.WalletID), report.ObservedAt),
		Status:             "active",
		ObservedAt:         observedAt,
		CoverageStartAt:    coverageStartAt,
		CoverageEndAt:      coverageEndAt,
		CoverageWindowDays: 30,
		Bundle: map[string]any{
			"importance_reason": []string{
				"First-time overlap can surface wallets converging on the same opportunity before it becomes obvious.",
				"Quality-wallet overlap matters more than raw novelty when judging early entry quality.",
			},
			"observed_facts": []string{
				fmt.Sprintf("First-connection score rated %s at %d.", score.Rating, score.Value),
				fmt.Sprintf("New common entries: %d. First-seen counterparties: %d.", report.NewCommonEntries, report.FirstSeenCounterparties),
				fmt.Sprintf("Quality overlap count: %d. First entry before crowding count: %d.", report.QualityWalletOverlapCount, report.FirstEntryBeforeCrowdingCount),
				fmt.Sprintf("Best lead before peers: %dh. Persistence-after-entry proxy count: %d.", report.BestLeadHoursBeforePeers, report.PersistenceAfterEntryProxyCount),
				fmt.Sprintf("Repeat early-entry success proxy: %t.", report.RepeatEarlyEntrySuccess),
			},
			"inferred_interpretations": []string{
				"Activity suggests early convergence rather than repeated legacy flow.",
			},
			"evidence":   evidence,
			"next_watch": buildNextWatchTargets(firstConnectionNextWatchTargets(report, nil)),
		},
	}
}

func readWalletLabelSet(
	ctx context.Context,
	reader db.WalletLabelReader,
	ref db.WalletRef,
) (domain.WalletLabelSet, error) {
	if reader == nil {
		return domain.WalletLabelSet{}, nil
	}
	labelsByWallet, err := reader.ReadWalletLabels(ctx, []db.WalletRef{ref})
	if err != nil {
		return domain.WalletLabelSet{}, err
	}
	return labelsByWallet[walletLabelLookupKey(ref)], nil
}

type interpretationFindingContext struct {
	ImportanceReason                         []string
	ObservedFacts                            []string
	InferredInterpretations                  []string
	Evidence                                 []map[string]any
	NextWatch                                []map[string]any
	AllowTreasuryRedistribution              bool
	AllowMMHandoff                           bool
	AllowFundAdjacentActivity                bool
	AllowHighConvictionEntry                 bool
	HasTreasuryAnchorEvidence                bool
	HasMMCounterpartyEvidence                bool
	TreasuryAnchorMatchCount                 int
	TreasuryFanoutCount                      int
	TreasuryOperationalCount                 int
	TreasuryRebalanceDiscount                int
	TreasuryToMarketPathCount                int
	TreasuryToExchangePathCount              int
	TreasuryToBridgePathCount                int
	TreasuryToMMPathCount                    int
	TreasuryDistinctMarketCounterpartyCount  int
	TreasuryOperationalOnlyDistributionCount int
	TreasuryInternalOpsDistributionCount     int
	TreasuryExternalOpsDistributionCount     int
	TreasuryExternalMarketAdjacentCount      int
	TreasuryExternalNonMarketCount           int
	MMAnchorMatchCount                       int
	MMInventoryRotationCount                 int
	MMProjectToMMPathCount                   int
	MMProjectToMMContactCount                int
	MMProjectToMMRoutedCandidateCount        int
	MMProjectToMMAdjacencyCount              int
	MMPostHandoffCount                       int
	MMPostHandoffExchangeCount               int
	MMPostHandoffBridgeCount                 int
	MMRepeatCounterpartyCount                int
	EntryHoldingPersistenceState             string
	EntryPostWindowFollowThroughCount        int
	EntryMaxPostWindowPersistenceHours       int
	HighConvictionConfidence                 float64
	HighConvictionImportance                 float64
}

func interpretationFindingsFromLabels(
	ref db.WalletRef,
	walletID string,
	observedAt string,
	baseConfidence float64,
	baseImportance float64,
	coverageWindowDays int,
	labels domain.WalletLabelSet,
	score domain.Score,
	context interpretationFindingContext,
) []db.FindingEntry {
	trimmedWalletID := strings.TrimSpace(walletID)
	trimmedAddress := strings.TrimSpace(ref.Address)
	trimmedChain := strings.TrimSpace(string(ref.Chain))
	if trimmedWalletID == "" || trimmedAddress == "" || trimmedChain == "" {
		return nil
	}

	observedAtValue, coverageStartAt, coverageEndAt := bundleCoverage(observedAt, coverageWindowDays)
	base := db.FindingEntry{
		WalletID:           trimmedWalletID,
		SubjectType:        domain.FindingSubjectWallet,
		SubjectChain:       ref.Chain,
		SubjectAddress:     trimmedAddress,
		SubjectKey:         strings.ToLower(trimmedChain + ":" + trimmedAddress),
		SubjectLabel:       "Wallet",
		Status:             "active",
		ObservedAt:         observedAtValue,
		CoverageStartAt:    coverageStartAt,
		CoverageEndAt:      coverageEndAt,
		CoverageWindowDays: coverageWindowDays,
	}

	newEntry := func(
		findingType domain.FindingType,
		summary string,
		importanceReason []string,
		observedFacts []string,
		inferredInterpretations []string,
		confidence float64,
		importance float64,
	) db.FindingEntry {
		return db.FindingEntry{
			FindingType:     findingType,
			Confidence:      maxFloat(0.35, confidence),
			ImportanceScore: maxFloat(0.35, importance),
			Summary:         summary,
			DedupKey:        fmt.Sprintf("finding:%s:%s:%s", findingType, trimmedWalletID, observedAt),
			Bundle: map[string]any{
				"importance_reason":        append([]string{}, importanceReason...),
				"observed_facts":           append([]string{}, observedFacts...),
				"inferred_interpretations": append([]string{}, inferredInterpretations...),
				"evidence":                 append(buildFindingEvidence(score.Evidence), context.Evidence...),
				"next_watch":               append([]map[string]any{}, context.NextWatch...),
			},
		}
	}

	findings := make([]db.FindingEntry, 0, 4)
	if context.AllowTreasuryRedistribution && (hasWalletLabel(labels, domain.WalletLabelClassInferred, "treasury") || context.HasTreasuryAnchorEvidence) {
		treasuryFacts := append([]string{
			"Wallet carries treasury-like attribution in the current indexed coverage.",
			fmt.Sprintf(
				"Treasury anchor matches %d, fanout signatures %d, market paths %d, rebalance discounts %d.",
				context.TreasuryAnchorMatchCount,
				context.TreasuryFanoutCount,
				context.TreasuryToMarketPathCount,
				context.TreasuryRebalanceDiscount,
			),
			fmt.Sprintf("%s scored %s at %d.", score.Name, score.Rating, score.Value),
		}, context.ObservedFacts...)
		treasuryReasons := append([]string{
			"Treasury attribution only matters when matched with redistribution or market-path evidence.",
		}, context.ImportanceReason...)
		treasuryInterpretations := append([]string{
			"Recent flow resembles treasury redistribution more than passive internal wallet maintenance.",
		}, context.InferredInterpretations...)
		findings = append(findings, mergeFindingEntry(base, newEntry(
			domain.FindingTypeTreasuryRedistribution,
			fmt.Sprintf("Treasury-like redistribution behavior is increasing around %s.", trimmedAddress),
			treasuryReasons,
			treasuryFacts,
			treasuryInterpretations,
			maxFloat(baseConfidence, 0.68),
			maxFloat(baseImportance, 0.62),
		)))
	}
	if context.AllowMMHandoff && (hasWalletLabel(labels, domain.WalletLabelClassInferred, "market_maker") ||
		hasWalletLabel(labels, domain.WalletLabelClassInferred, "fund") ||
		hasWalletLabel(labels, domain.WalletLabelClassInferred, "treasury") ||
		context.HasMMCounterpartyEvidence) {
		mmFacts := append([]string{
			"Wallet carries MM-adjacent or capital-source attribution in the current indexed coverage.",
			fmt.Sprintf(
				"MM anchor matches %d, project-to-MM paths %d, post-handoff distributions %d, inventory rotations %d, repeat counterparties %d.",
				context.MMAnchorMatchCount,
				context.MMProjectToMMPathCount,
				context.MMPostHandoffCount,
				context.MMInventoryRotationCount,
				context.MMRepeatCounterpartyCount,
			),
			fmt.Sprintf("%s scored %s at %d.", score.Name, score.Rating, score.Value),
		}, context.ObservedFacts...)
		mmReasons := append([]string{
			"MM handoff should only fire when a transfer path into MM-like distribution infrastructure is visible.",
		}, context.ImportanceReason...)
		mmInterpretations := append([]string{
			"Observed flow resembles a project or treasury handoff into market-making or downstream distribution infrastructure.",
		}, context.InferredInterpretations...)
		findings = append(findings, mergeFindingEntry(base, newEntry(
			domain.FindingTypeSuspectedMMHandoff,
			fmt.Sprintf("A market-maker-like handoff pattern is forming around %s.", trimmedAddress),
			mmReasons,
			mmFacts,
			mmInterpretations,
			maxFloat(baseConfidence, 0.7),
			maxFloat(baseImportance, 0.66),
		)))
	}
	if context.AllowFundAdjacentActivity && hasWalletLabel(labels, domain.WalletLabelClassInferred, "fund") {
		findings = append(findings, mergeFindingEntry(base, newEntry(
			domain.FindingTypeFundAdjacentActivity,
			fmt.Sprintf("Fund-adjacent activity is increasing around %s.", trimmedAddress),
			[]string{
				"Fund-adjacent counterparties increase the likelihood that recent moves are strategic rather than retail noise.",
			},
			append([]string{
				"Wallet carries inferred fund labeling in the current indexed coverage.",
				fmt.Sprintf("%s scored %s at %d.", score.Name, score.Rating, score.Value),
			}, context.ObservedFacts...),
			append([]string{
				"The wallet appears to be operating close to fund-like capital or counterparties.",
			}, context.InferredInterpretations...),
			maxFloat(baseConfidence, 0.66),
			maxFloat(baseImportance, 0.6),
		)))
	}
	if context.AllowHighConvictionEntry && score.Name == domain.ScoreAlpha && score.Rating == domain.RatingHigh {
		findings = append(findings, mergeFindingEntry(base, newEntry(
			domain.FindingTypeHighConvictionEntry,
			fmt.Sprintf("High-conviction entry conditions are forming around %s.", trimmedAddress),
			[]string{
				"High alpha-style convergence can precede more visible smart money participation.",
			},
			append([]string{
				fmt.Sprintf("%s scored %s at %d.", score.Name, score.Rating, score.Value),
			}, context.ObservedFacts...),
			append([]string{
				"The wallet is behaving like an early, higher-conviction participant rather than a late follower.",
			}, context.InferredInterpretations...),
			maxFloat(baseConfidence, context.HighConvictionConfidence),
			maxFloat(baseImportance, context.HighConvictionImportance),
		)))
	}

	return findings
}

func clusterScoreInterpretationContext(graph domain.WalletGraph, score domain.Score) interpretationFindingContext {
	counterpartyCount := countClusterGraphCounterparties(graph)
	watchTargets := make([]domain.NextWatchTarget, 0, 2)
	for _, node := range graph.Nodes {
		if node.Kind != domain.WalletGraphNodeWallet || strings.TrimSpace(node.Address) == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(node.Address), strings.TrimSpace(graph.Address)) {
			continue
		}
		watchTargets = append(watchTargets, domain.NextWatchTarget{
			SubjectType: domain.FindingSubjectWallet,
			Chain:       node.Chain,
			Address:     strings.TrimSpace(node.Address),
			Label:       strings.TrimSpace(node.Label),
		})
		if len(watchTargets) == 2 {
			break
		}
	}

	return interpretationFindingContext{
		AllowFundAdjacentActivity: counterpartyCount >= 2,
		ObservedFacts: []string{
			fmt.Sprintf("Graph neighborhood contains %d nodes, %d edges, and %d wallet counterparties.", len(graph.Nodes), len(graph.Edges), counterpartyCount),
		},
		InferredInterpretations: []string{
			"Counterparty overlap and graph density raise the chance that this wallet is moving inside a coordinated behavior cohort.",
		},
		Evidence: []map[string]any{
			buildFindingEvidenceItem(
				"graph_neighborhood",
				fmt.Sprintf("%d wallet counterparties", counterpartyCount),
				findingConfidenceFromScore(score),
				map[string]any{
					"wallet_node_count": len(graph.Nodes),
					"edge_count":        len(graph.Edges),
					"density_capped":    graph.DensityCapped,
				},
			),
		},
		NextWatch: buildNextWatchTargets(watchTargets),
	}
}

func shadowExitInterpretationContext(
	report ShadowExitSnapshotReport,
	score domain.Score,
	evidenceReport *db.WalletBridgeExchangeEvidenceReport,
	treasuryMMReport *db.WalletTreasuryMMEvidenceReport,
) interpretationFindingContext {
	nextWatch := make([]domain.NextWatchTarget, 0, 2)
	nextWatch = append(nextWatch, bridgeFindingNextWatch(evidenceReport)...)
	nextWatch = append(nextWatch, exchangeFindingNextWatch(evidenceReport)...)
	nextWatch = append(nextWatch, treasuryMMFindingNextWatch(treasuryMMReport)...)
	if evidenceReport == nil {
		if report.BridgeEscapeCount > 0 {
			nextWatch = append(nextWatch, domain.NextWatchTarget{
				SubjectType: domain.FindingSubjectEntity,
				Label:       "Bridge-linked destination wallets",
				Metadata: map[string]any{
					"route": "cross-chain-rotation",
				},
			})
		}
		if report.CEXProximityCount > 0 {
			nextWatch = append(nextWatch, domain.NextWatchTarget{
				SubjectType: domain.FindingSubjectEntity,
				Label:       "Exchange-adjacent counterparties",
				Metadata: map[string]any{
					"route": "exchange-pressure",
				},
			})
		}
	}

	treasuryEvidence := buildTreasuryMMFindingEvidenceValue(treasuryMMReport)
	allowTreasury := treasuryMMReport != nil && shouldEmitTreasuryRedistribution(report, treasuryMMReport)
	allowMM := treasuryMMReport != nil && shouldEmitMMHandoff(report, treasuryMMReport)
	hasMMCounterpartyEvidence := false
	hasTreasuryAnchorEvidence := false
	if treasuryMMReport != nil {
		hasTreasuryAnchorEvidence = treasuryMMReport.HasTreasuryLabel || treasuryMMReport.TreasuryFeatures.AnchorMatchCount > 0
		hasMMCounterpartyEvidence = treasuryMMReport.MMFeatures.MMAnchorMatchCount > 0 || treasuryMMReport.MMFeatures.ProjectToMMPathCount > 0
	}

	return interpretationFindingContext{
		AllowTreasuryRedistribution: allowTreasury,
		AllowMMHandoff:              allowMM,
		ObservedFacts: []string{
			fmt.Sprintf("Bridge escape count: %d. CEX proximity count: %d. Fanout count: %d.", report.BridgeEscapeCount, report.CEXProximityCount, report.FanOutCount),
			fmt.Sprintf("Outflow ratio reached %.2f within the indexed coverage window.", report.OutflowRatio),
			fmt.Sprintf("Bridge confirmed destinations: %d. Deposit-like paths: %d.", report.BridgeConfirmedDestinationCount, report.DepositLikePathCount),
			fmt.Sprintf("Treasury anchor matches: %d. Fanout signatures: %d. Treasury market paths: %d.", report.TreasuryAnchorMatchCount, report.TreasuryFanoutSignatureCount, report.TreasuryToMarketPathCount),
			fmt.Sprintf("Treasury exchange paths: %d. Bridge paths: %d. MM paths: %d. Distinct market counterparties: %d.", report.TreasuryToExchangePathCount, report.TreasuryToBridgePathCount, report.TreasuryToMMPathCount, report.TreasuryDistinctMarketCounterpartyCount),
			fmt.Sprintf("Treasury operational distribution count: %d. Operational-only distribution count: %d. Internal ops: %d. External ops: %d. Market-adjacent external ops: %d. Non-market external ops: %d. Rebalance discount count: %d.", report.TreasuryOperationalDistributionCount, report.TreasuryOperationalOnlyDistributionCount, report.TreasuryInternalOpsDistributionCount, report.TreasuryExternalOpsDistributionCount, report.TreasuryExternalMarketAdjacentCount, report.TreasuryExternalNonMarketCount, report.TreasuryRebalanceDiscountCount),
			fmt.Sprintf("MM confirmed paths: %d. Contact-only paths: %d. Routed candidates: %d. Mere adjacency: %d. Post-handoff distribution: %d. Exchange touches: %d. Bridge touches: %d. Inventory rotation: %d. Repeat counterparties: %d.", report.ProjectToMMPathCount, report.ProjectToMMContactCount, report.ProjectToMMRoutedCandidateCount, report.ProjectToMMAdjacencyCount, report.PostHandoffDistributionCount, report.PostHandoffExchangeTouchCount, report.PostHandoffBridgeTouchCount, report.InventoryRotationCount, report.RepeatMMCounterpartyCount),
		},
		InferredInterpretations: []string{
			"Recent flow looks more like redistribution or handoff behavior than passive holding.",
		},
		Evidence: append([]map[string]any{
			buildFindingEvidenceItem("bridge_escape_count", fmt.Sprintf("%d", report.BridgeEscapeCount), findingConfidenceFromScore(score), nil),
			buildFindingEvidenceItem("cex_proximity_count", fmt.Sprintf("%d", report.CEXProximityCount), findingConfidenceFromScore(score), nil),
			buildFindingEvidenceItem("fan_out_count", fmt.Sprintf("%d", report.FanOutCount), findingConfidenceFromScore(score), nil),
			buildFindingEvidenceItem("outflow_ratio", fmt.Sprintf("%.2f", report.OutflowRatio), findingConfidenceFromScore(score), nil),
		}, append(buildBridgeExchangeFindingEvidenceValue(report, evidenceReport), treasuryEvidence...)...),
		NextWatch:                                buildNextWatchTargets(nextWatch),
		HasTreasuryAnchorEvidence:                hasTreasuryAnchorEvidence,
		HasMMCounterpartyEvidence:                hasMMCounterpartyEvidence,
		TreasuryAnchorMatchCount:                 report.TreasuryAnchorMatchCount,
		TreasuryFanoutCount:                      report.TreasuryFanoutSignatureCount,
		TreasuryOperationalCount:                 report.TreasuryOperationalDistributionCount,
		TreasuryRebalanceDiscount:                report.TreasuryRebalanceDiscountCount,
		TreasuryToMarketPathCount:                report.TreasuryToMarketPathCount,
		TreasuryToExchangePathCount:              report.TreasuryToExchangePathCount,
		TreasuryToBridgePathCount:                report.TreasuryToBridgePathCount,
		TreasuryToMMPathCount:                    report.TreasuryToMMPathCount,
		TreasuryDistinctMarketCounterpartyCount:  report.TreasuryDistinctMarketCounterpartyCount,
		TreasuryOperationalOnlyDistributionCount: report.TreasuryOperationalOnlyDistributionCount,
		TreasuryInternalOpsDistributionCount:     report.TreasuryInternalOpsDistributionCount,
		TreasuryExternalOpsDistributionCount:     report.TreasuryExternalOpsDistributionCount,
		TreasuryExternalMarketAdjacentCount:      report.TreasuryExternalMarketAdjacentCount,
		TreasuryExternalNonMarketCount:           report.TreasuryExternalNonMarketCount,
		MMAnchorMatchCount:                       report.MMAnchorMatchCount,
		MMInventoryRotationCount:                 report.InventoryRotationCount,
		MMProjectToMMPathCount:                   report.ProjectToMMPathCount,
		MMProjectToMMContactCount:                report.ProjectToMMContactCount,
		MMProjectToMMRoutedCandidateCount:        report.ProjectToMMRoutedCandidateCount,
		MMProjectToMMAdjacencyCount:              report.ProjectToMMAdjacencyCount,
		MMPostHandoffCount:                       report.PostHandoffDistributionCount,
		MMPostHandoffExchangeCount:               report.PostHandoffExchangeTouchCount,
		MMPostHandoffBridgeCount:                 report.PostHandoffBridgeTouchCount,
		MMRepeatCounterpartyCount:                report.RepeatMMCounterpartyCount,
	}
}

func shouldEmitTreasuryRedistribution(
	report ShadowExitSnapshotReport,
	treasuryMMReport *db.WalletTreasuryMMEvidenceReport,
) bool {
	if treasuryMMReport == nil {
		return false
	}

	hasAnchor := treasuryMMReport.HasTreasuryLabel || treasuryMMReport.TreasuryFeatures.AnchorMatchCount > 0
	hasOperationalFanout :=
		treasuryMMReport.TreasuryFeatures.FanoutSignatureCount >= 2 ||
			treasuryMMReport.TreasuryFeatures.OperationalDistributionCount > 0
	hasStrongMarketPath :=
		treasuryMMReport.TreasuryFeatures.TreasuryToExchangePathCount > 0 ||
			treasuryMMReport.TreasuryFeatures.TreasuryToMMPathCount > 0 ||
			(treasuryMMReport.TreasuryFeatures.TreasuryToBridgePathCount > 0 &&
				treasuryMMReport.TreasuryFeatures.FanoutSignatureCount >= 3)
	hasMarketBreadth :=
		treasuryMMReport.TreasuryFeatures.DistinctMarketCounterpartyCount > 0 ||
			treasuryMMReport.TreasuryFeatures.TreasuryToMarketPathCount > 1
	hasStrongOutflow :=
		report.OutflowRatio >= 0.35 ||
			report.FanOutCount >= 2 ||
			report.FanOutCandidateCount24h >= 2
	rebalanceHeavy :=
		treasuryMMReport.TreasuryFeatures.RebalanceDiscountCount > 0 &&
			treasuryMMReport.TreasuryFeatures.RebalanceDiscountCount >=
				treasuryMMReport.TreasuryFeatures.TreasuryToMarketPathCount &&
			treasuryMMReport.TreasuryFeatures.FanoutSignatureCount == 0

	if rebalanceHeavy {
		return false
	}
	if treasuryMMReport.TreasuryFeatures.OperationalOnlyDistributionCount > 0 &&
		treasuryMMReport.TreasuryFeatures.TreasuryToMarketPathCount == 0 {
		return false
	}
	if treasuryMMReport.TreasuryFeatures.ExternalOpsDistributionCount > 0 &&
		treasuryMMReport.TreasuryFeatures.TreasuryToExchangePathCount == 0 &&
		treasuryMMReport.TreasuryFeatures.TreasuryToMMPathCount == 0 {
		return false
	}
	if treasuryMMReport.TreasuryFeatures.ExternalNonMarketCount > 0 &&
		treasuryMMReport.TreasuryFeatures.TreasuryToExchangePathCount == 0 &&
		treasuryMMReport.TreasuryFeatures.TreasuryToMMPathCount == 0 {
		return false
	}

	return hasAnchor &&
		hasOperationalFanout &&
		hasStrongMarketPath &&
		hasMarketBreadth &&
		hasStrongOutflow
}

func shouldEmitMMHandoff(
	report ShadowExitSnapshotReport,
	treasuryMMReport *db.WalletTreasuryMMEvidenceReport,
) bool {
	if treasuryMMReport == nil {
		return false
	}

	hasProjectPath := treasuryMMReport.MMFeatures.ProjectToMMPathCount > 0
	hasContactOnly :=
		treasuryMMReport.MMFeatures.ProjectToMMContactCount > 0 &&
			treasuryMMReport.MMFeatures.ProjectToMMPathCount == 0
	hasAdjacencyOnly :=
		treasuryMMReport.MMFeatures.ProjectToMMAdjacencyCount > 0 &&
			treasuryMMReport.MMFeatures.ProjectToMMPathCount == 0 &&
			treasuryMMReport.MMFeatures.ProjectToMMRoutedCandidateCount == 0
	hasQualifiedPostHandoff :=
		treasuryMMReport.MMFeatures.PostHandoffExchangeTouchCount > 0 ||
			(treasuryMMReport.MMFeatures.PostHandoffBridgeTouchCount > 0 &&
				(treasuryMMReport.MMFeatures.InventoryRotationCount > 0 ||
					treasuryMMReport.MMFeatures.RepeatMMCounterpartyCount > 1))
	hasDistributionEvidence :=
		treasuryMMReport.MMFeatures.InventoryRotationCount > 0 ||
			treasuryMMReport.MMFeatures.RepeatMMCounterpartyCount > 1
	hasMMAnchor := treasuryMMReport.MMFeatures.MMAnchorMatchCount > 0
	hasRootAnchor := treasuryMMReport.HasFundLabel || treasuryMMReport.HasTreasuryLabel
	hasStrongOutflow :=
		report.OutflowRatio >= 0.25 ||
			report.FanOutCount >= 2 ||
			report.CEXProximityCount > 0

	return hasRootAnchor &&
		!hasContactOnly &&
		!hasAdjacencyOnly &&
		hasProjectPath &&
		hasQualifiedPostHandoff &&
		(hasDistributionEvidence || hasMMAnchor) &&
		hasStrongOutflow
}

func shouldEmitCrossChainRotation(
	report ShadowExitSnapshotReport,
	evidenceReport *db.WalletBridgeExchangeEvidenceReport,
) bool {
	if evidenceReport == nil {
		return report.BridgeEscapeCount > 0
	}
	return report.BridgeConfirmedDestinationCount > 0 ||
		report.BridgeRecurrenceDays >= 2 ||
		evidenceReport.BridgeFeatures.PostBridgeExchangeTouchCount > 0 ||
		evidenceReport.BridgeFeatures.PostBridgeProtocolEntryCount > 0
}

func shouldEmitExchangePressure(
	report ShadowExitSnapshotReport,
	evidenceReport *db.WalletBridgeExchangeEvidenceReport,
) bool {
	if evidenceReport == nil {
		return report.CEXProximityCount > 0
	}
	return report.DepositLikePathCount > 0 ||
		report.ExchangeRecurrenceDays >= 2 ||
		report.ExchangeOutflowShare >= 0.25 ||
		report.ExchangeOutboundCount >= 2
}

func buildBridgeExchangeFindingEvidence(
	report db.WalletBridgeExchangeEvidenceReport,
) []map[string]any {
	out := make([]map[string]any, 0, len(report.BridgeLinks)+len(report.ExchangePaths)+4)
	for _, item := range report.BridgeLinks {
		out = append(out, map[string]any{
			"type":        "bridge_link_confirmation",
			"value":       firstNonEmpty(item.BridgeLabel, item.BridgeAddress),
			"confidence":  item.Confidence,
			"observed_at": item.ObservedAt.UTC().Format(time.RFC3339),
			"metadata": map[string]any{
				"txRef": map[string]any{
					"chain":      string(report.Chain),
					"address":    report.Address,
					"txHash":     item.TxHash,
					"observedAt": item.ObservedAt.UTC().Format(time.RFC3339),
				},
				"pathRef": map[string]any{
					"kind":               "bridge_link_confirmation",
					"bridgeChain":        string(item.BridgeChain),
					"bridgeAddress":      item.BridgeAddress,
					"destinationChain":   string(item.DestinationChain),
					"destinationAddress": item.DestinationAddress,
					"destinationTxHash":  item.DestinationTxHash,
				},
				"entityRef": map[string]any{
					"entityKey":  item.BridgeEntityKey,
					"entityType": item.BridgeEntityType,
					"label":      item.BridgeLabel,
				},
				"counterpartyRef": map[string]any{
					"chain":      string(item.BridgeChain),
					"address":    item.BridgeAddress,
					"label":      item.BridgeLabel,
					"entityKey":  item.BridgeEntityKey,
					"entityType": item.BridgeEntityType,
				},
			},
		})
	}
	for _, item := range report.ExchangePaths {
		out = append(out, map[string]any{
			"type":        "deposit_like_path",
			"value":       firstNonEmpty(item.ExchangeLabel, item.ExchangeAddress),
			"confidence":  item.Confidence,
			"observed_at": item.ObservedAt.UTC().Format(time.RFC3339),
			"metadata": map[string]any{
				"txRef": map[string]any{
					"chain":      string(report.Chain),
					"address":    report.Address,
					"txHash":     item.TxHash,
					"observedAt": item.ObservedAt.UTC().Format(time.RFC3339),
				},
				"pathRef": map[string]any{
					"kind":                item.PathKind,
					"intermediaryChain":   string(item.IntermediaryChain),
					"intermediaryAddress": item.IntermediaryAddress,
					"exchangeChain":       string(item.ExchangeChain),
					"exchangeAddress":     item.ExchangeAddress,
					"exchangeTxHash":      item.ExchangeTxHash,
				},
				"entityRef": map[string]any{
					"entityKey":  item.ExchangeEntityKey,
					"entityType": item.ExchangeEntityType,
					"label":      item.ExchangeLabel,
				},
				"counterpartyRef": map[string]any{
					"chain":      string(item.ExchangeChain),
					"address":    item.ExchangeAddress,
					"label":      item.ExchangeLabel,
					"entityKey":  item.ExchangeEntityKey,
					"entityType": item.ExchangeEntityType,
				},
			},
		})
	}
	return out
}

func buildBridgeExchangeFindingEvidenceValue(
	report ShadowExitSnapshotReport,
	evidenceReport *db.WalletBridgeExchangeEvidenceReport,
) []map[string]any {
	if evidenceReport == nil {
		return nil
	}
	return append([]map[string]any{
		buildFindingEvidenceItem("bridge_confirmed_destination_count", fmt.Sprintf("%d", report.BridgeConfirmedDestinationCount), 0.78, map[string]any{
			"bridgeOutflowShare":   report.BridgeOutflowShare,
			"bridgeRecurrenceDays": report.BridgeRecurrenceDays,
		}),
		buildFindingEvidenceItem("deposit_like_path_count", fmt.Sprintf("%d", report.DepositLikePathCount), 0.8, map[string]any{
			"exchangeOutflowShare":   report.ExchangeOutflowShare,
			"exchangeRecurrenceDays": report.ExchangeRecurrenceDays,
			"exchangeOutboundCount":  report.ExchangeOutboundCount,
		}),
	}, buildBridgeExchangeFindingEvidence(*evidenceReport)...)
}

func bridgeFindingNextWatch(report *db.WalletBridgeExchangeEvidenceReport) []domain.NextWatchTarget {
	if report == nil {
		return nil
	}
	out := make([]domain.NextWatchTarget, 0, 2)
	for _, item := range report.BridgeLinks {
		if strings.TrimSpace(item.DestinationAddress) == "" {
			continue
		}
		out = append(out, domain.NextWatchTarget{
			SubjectType: domain.FindingSubjectWallet,
			Chain:       item.DestinationChain,
			Address:     item.DestinationAddress,
			Label:       firstNonEmpty(item.DestinationLabel, compactAddress(item.DestinationAddress)),
			Metadata: map[string]any{
				"route": "cross-chain-rotation",
				"pathRef": map[string]any{
					"kind":               "bridge_link_confirmation",
					"bridgeChain":        string(item.BridgeChain),
					"bridgeAddress":      item.BridgeAddress,
					"destinationChain":   string(item.DestinationChain),
					"destinationAddress": item.DestinationAddress,
					"destinationTxHash":  item.DestinationTxHash,
				},
			},
		})
		if len(out) == 2 {
			break
		}
	}
	return out
}

func exchangeFindingNextWatch(report *db.WalletBridgeExchangeEvidenceReport) []domain.NextWatchTarget {
	if report == nil {
		return nil
	}
	out := make([]domain.NextWatchTarget, 0, 2)
	for _, item := range report.ExchangePaths {
		out = append(out, domain.NextWatchTarget{
			SubjectType: domain.FindingSubjectEntity,
			Label:       firstNonEmpty(item.ExchangeLabel, "Exchange-adjacent counterparty"),
			Metadata: map[string]any{
				"route": "exchange-pressure",
				"entityRef": map[string]any{
					"entityKey":  item.ExchangeEntityKey,
					"entityType": item.ExchangeEntityType,
					"label":      item.ExchangeLabel,
				},
				"counterpartyRef": map[string]any{
					"chain":      string(item.ExchangeChain),
					"address":    item.ExchangeAddress,
					"label":      item.ExchangeLabel,
					"entityKey":  item.ExchangeEntityKey,
					"entityType": item.ExchangeEntityType,
				},
			},
		})
		if len(out) == 2 {
			break
		}
	}
	return out
}

func buildTreasuryMMFindingEvidenceValue(
	report *db.WalletTreasuryMMEvidenceReport,
) []map[string]any {
	if report == nil {
		return nil
	}
	out := []map[string]any{
		buildFindingEvidenceItem("treasury_anchor_match_count", fmt.Sprintf("%d", report.TreasuryFeatures.AnchorMatchCount), 0.76, map[string]any{
			"hasTreasuryLabel":                 report.HasTreasuryLabel,
			"fanoutSignatureCount":             report.TreasuryFeatures.FanoutSignatureCount,
			"operationalDistributionCount":     report.TreasuryFeatures.OperationalDistributionCount,
			"rebalanceDiscountCount":           report.TreasuryFeatures.RebalanceDiscountCount,
			"treasuryToMarketPathCount":        report.TreasuryFeatures.TreasuryToMarketPathCount,
			"treasuryToExchangePathCount":      report.TreasuryFeatures.TreasuryToExchangePathCount,
			"treasuryToBridgePathCount":        report.TreasuryFeatures.TreasuryToBridgePathCount,
			"treasuryToMMPathCount":            report.TreasuryFeatures.TreasuryToMMPathCount,
			"distinctMarketCounterpartyCount":  report.TreasuryFeatures.DistinctMarketCounterpartyCount,
			"operationalOnlyDistributionCount": report.TreasuryFeatures.OperationalOnlyDistributionCount,
			"internalOpsDistributionCount":     report.TreasuryFeatures.InternalOpsDistributionCount,
			"externalOpsDistributionCount":     report.TreasuryFeatures.ExternalOpsDistributionCount,
			"externalMarketAdjacentCount":      report.TreasuryFeatures.ExternalMarketAdjacentCount,
			"externalNonMarketCount":           report.TreasuryFeatures.ExternalNonMarketCount,
		}),
		buildFindingEvidenceItem("mm_project_path_count", fmt.Sprintf("%d", report.MMFeatures.ProjectToMMPathCount), 0.78, map[string]any{
			"hasFundLabel":                    report.HasFundLabel,
			"hasTreasuryLabel":                report.HasTreasuryLabel,
			"mmAnchorMatchCount":              report.MMFeatures.MMAnchorMatchCount,
			"postHandoffDistributionCount":    report.MMFeatures.PostHandoffDistributionCount,
			"postHandoffExchangeTouchCount":   report.MMFeatures.PostHandoffExchangeTouchCount,
			"postHandoffBridgeTouchCount":     report.MMFeatures.PostHandoffBridgeTouchCount,
			"projectToMMContactCount":         report.MMFeatures.ProjectToMMContactCount,
			"projectToMMRoutedCandidateCount": report.MMFeatures.ProjectToMMRoutedCandidateCount,
			"projectToMMAdjacencyCount":       report.MMFeatures.ProjectToMMAdjacencyCount,
			"inventoryRotationCount":          report.MMFeatures.InventoryRotationCount,
			"repeatMMCounterpartyCount":       report.MMFeatures.RepeatMMCounterpartyCount,
		}),
	}
	for _, item := range report.TreasuryPaths {
		out = append(out, map[string]any{
			"type":        item.PathKind,
			"value":       firstNonEmpty(item.CounterpartyLabel, item.CounterpartyAddress),
			"confidence":  item.Confidence,
			"observed_at": item.ObservedAt.UTC().Format(time.RFC3339),
			"metadata": map[string]any{
				"txRef": map[string]any{
					"chain":      string(report.Chain),
					"address":    report.Address,
					"txHash":     item.TxHash,
					"observedAt": item.ObservedAt.UTC().Format(time.RFC3339),
				},
				"pathRef": map[string]any{
					"kind":                item.PathKind,
					"counterpartyChain":   string(item.CounterpartyChain),
					"counterpartyAddress": item.CounterpartyAddress,
					"downstreamChain":     string(item.DownstreamChain),
					"downstreamAddress":   item.DownstreamAddress,
					"downstreamTxHash":    item.DownstreamTxHash,
				},
				"counterpartyRef": map[string]any{
					"chain":      string(item.CounterpartyChain),
					"address":    item.CounterpartyAddress,
					"label":      item.CounterpartyLabel,
					"entityKey":  item.CounterpartyEntityKey,
					"entityType": item.CounterpartyEntityType,
				},
				"entityRef": map[string]any{
					"entityKey":  item.CounterpartyEntityKey,
					"entityType": item.CounterpartyEntityType,
					"label":      item.CounterpartyLabel,
				},
				"downstreamRef": map[string]any{
					"chain":      string(item.DownstreamChain),
					"address":    item.DownstreamAddress,
					"label":      item.DownstreamLabel,
					"entityKey":  item.DownstreamEntityKey,
					"entityType": item.DownstreamEntityType,
					"txHash":     item.DownstreamTxHash,
				},
			},
		})
	}
	for _, item := range report.MMPaths {
		out = append(out, map[string]any{
			"type":        item.PathKind,
			"value":       firstNonEmpty(item.CounterpartyLabel, item.CounterpartyAddress),
			"confidence":  item.Confidence,
			"observed_at": item.ObservedAt.UTC().Format(time.RFC3339),
			"metadata": map[string]any{
				"txRef": map[string]any{
					"chain":      string(report.Chain),
					"address":    report.Address,
					"txHash":     item.TxHash,
					"observedAt": item.ObservedAt.UTC().Format(time.RFC3339),
				},
				"pathRef": map[string]any{
					"kind":                item.PathKind,
					"counterpartyChain":   string(item.CounterpartyChain),
					"counterpartyAddress": item.CounterpartyAddress,
					"downstreamChain":     string(item.DownstreamChain),
					"downstreamAddress":   item.DownstreamAddress,
					"downstreamTxHash":    item.DownstreamTxHash,
				},
				"counterpartyRef": map[string]any{
					"chain":      string(item.CounterpartyChain),
					"address":    item.CounterpartyAddress,
					"label":      item.CounterpartyLabel,
					"entityKey":  item.CounterpartyEntityKey,
					"entityType": item.CounterpartyEntityType,
				},
				"entityRef": map[string]any{
					"entityKey":  item.CounterpartyEntityKey,
					"entityType": item.CounterpartyEntityType,
					"label":      item.CounterpartyLabel,
				},
				"downstreamRef": map[string]any{
					"chain":      string(item.DownstreamChain),
					"address":    item.DownstreamAddress,
					"label":      item.DownstreamLabel,
					"entityKey":  item.DownstreamEntityKey,
					"entityType": item.DownstreamEntityType,
					"txHash":     item.DownstreamTxHash,
				},
			},
		})
	}
	return out
}

func treasuryMMFindingNextWatch(report *db.WalletTreasuryMMEvidenceReport) []domain.NextWatchTarget {
	if report == nil {
		return nil
	}
	out := make([]domain.NextWatchTarget, 0, 2)
	for _, item := range report.MMPaths {
		if strings.TrimSpace(item.CounterpartyAddress) == "" {
			continue
		}
		out = append(out, domain.NextWatchTarget{
			SubjectType: domain.FindingSubjectWallet,
			Chain:       item.CounterpartyChain,
			Address:     item.CounterpartyAddress,
			Label:       firstNonEmpty(item.CounterpartyLabel, compactAddress(item.CounterpartyAddress)),
			Metadata: map[string]any{
				"route": "mm-handoff",
				"txRef": map[string]any{
					"chain":      string(report.Chain),
					"address":    report.Address,
					"txHash":     item.TxHash,
					"observedAt": item.ObservedAt.UTC().Format(time.RFC3339),
				},
				"pathRef": map[string]any{
					"kind":                item.PathKind,
					"counterpartyChain":   string(item.CounterpartyChain),
					"counterpartyAddress": item.CounterpartyAddress,
					"downstreamChain":     string(item.DownstreamChain),
					"downstreamAddress":   item.DownstreamAddress,
					"downstreamTxHash":    item.DownstreamTxHash,
				},
			},
		})
		if len(out) == 2 {
			return out
		}
	}
	for _, item := range report.TreasuryPaths {
		if strings.TrimSpace(item.CounterpartyAddress) == "" {
			continue
		}
		out = append(out, domain.NextWatchTarget{
			SubjectType: domain.FindingSubjectWallet,
			Chain:       item.CounterpartyChain,
			Address:     item.CounterpartyAddress,
			Label:       firstNonEmpty(item.CounterpartyLabel, compactAddress(item.CounterpartyAddress)),
			Metadata: map[string]any{
				"route": "treasury-redistribution",
			},
		})
		if len(out) == 2 {
			break
		}
	}
	return out
}

func compactAddress(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 12 {
		return trimmed
	}
	return trimmed[:6] + "..." + trimmed[len(trimmed)-4:]
}

func firstConnectionInterpretationContext(
	report FirstConnectionSnapshotReport,
	score domain.Score,
	maturedPrior *db.WalletEntryFeaturesSnapshot,
) interpretationFindingContext {
	nextWatch := firstConnectionNextWatchTargets(report, maturedPrior)
	if report.NewCommonEntries > 0 {
		nextWatch = append(nextWatch, domain.NextWatchTarget{
			SubjectType: domain.FindingSubjectToken,
			Label:       "Newly shared token entries",
		})
	}
	if report.FirstSeenCounterparties > 0 {
		nextWatch = append(nextWatch, domain.NextWatchTarget{
			SubjectType: domain.FindingSubjectWallet,
			Label:       "First-seen counterparties",
		})
	}

	repeatEarlyEntrySuccess := report.RepeatEarlyEntrySuccess
	holdingPersistenceState := ""
	postWindowFollowThroughCount := 0
	maxPostWindowPersistenceHours := 0
	if maturedPrior != nil {
		holdingPersistenceState = strings.TrimSpace(maturedPrior.HoldingPersistenceState)
		postWindowFollowThroughCount = maturedPrior.PostWindowFollowThroughCount
		maxPostWindowPersistenceHours = maturedPrior.MaxPostWindowPersistenceHours
	}

	highConvictionConfidence := 0.72
	highConvictionImportance := 0.7
	switch holdingPersistenceState {
	case "sustained":
		highConvictionConfidence = 0.8
		highConvictionImportance = 0.76
	case "short_lived":
		highConvictionConfidence = 0.58
		highConvictionImportance = 0.52
	}

	return interpretationFindingContext{
		AllowHighConvictionEntry: shouldEmitHighConvictionEntry(report),
		HighConvictionConfidence: highConvictionConfidence,
		HighConvictionImportance: highConvictionImportance,
		ImportanceReason: []string{
			"High-conviction entry should require more than a generic alpha score. It needs repeated early overlap and crowding evidence.",
		},
		ObservedFacts: []string{
			fmt.Sprintf("New common entries: %d. First-seen counterparties: %d. Hot feed mentions: %d.", report.NewCommonEntries, report.FirstSeenCounterparties, report.HotFeedMentions),
			fmt.Sprintf("Quality wallet overlap count: %d. Sustained overlap count: %d. Strong lead count: %d.", report.QualityWalletOverlapCount, report.SustainedOverlapCounterpartyCount, report.StrongLeadCounterpartyCount),
			fmt.Sprintf("First entry before crowding count: %d. Best lead before peers: %dh.", report.FirstEntryBeforeCrowdingCount, report.BestLeadHoursBeforePeers),
			fmt.Sprintf("Persistence after entry proxy count: %d. Repeat early-entry success proxy: %t.", report.PersistenceAfterEntryProxyCount, repeatEarlyEntrySuccess),
			fmt.Sprintf("Historical sustained entry outcomes: %d.", report.HistoricalSustainedOutcomeCount),
			fmt.Sprintf("Post-window follow-through count: %d. Holding persistence state: %s. Max persistence after entry: %dh.", postWindowFollowThroughCount, firstNonEmpty(holdingPersistenceState, "monitoring"), maxPostWindowPersistenceHours),
		},
		InferredInterpretations: []string{
			"These overlaps look more like early convergence than repeated legacy flow when they appear before broader crowding.",
		},
		Evidence: []map[string]any{
			buildFindingEvidenceItem("new_common_entries", fmt.Sprintf("%d", report.NewCommonEntries), findingConfidenceFromScore(score), map[string]any{
				"crowdingProxy": maxFloat(float64(report.FirstSeenCounterparties), float64(report.HotFeedMentions)),
			}),
			buildFindingEvidenceItem("first_seen_counterparties", fmt.Sprintf("%d", report.FirstSeenCounterparties), findingConfidenceFromScore(score), map[string]any{
				"noveltyWindow": "current_snapshot",
			}),
			buildFindingEvidenceItem("hot_feed_mentions", fmt.Sprintf("%d", report.HotFeedMentions), findingConfidenceFromScore(score), map[string]any{
				"repeatEarlyEntrySuccess": repeatEarlyEntrySuccess,
			}),
			buildFindingEvidenceItem("quality_wallet_overlap_count", fmt.Sprintf("%d", report.QualityWalletOverlapCount), findingConfidenceFromScore(score), map[string]any{
				"sustainedOverlapCounterpartyCount": report.SustainedOverlapCounterpartyCount,
				"strongLeadCounterpartyCount":       report.StrongLeadCounterpartyCount,
				"topCounterpartyCount":              len(report.TopCounterparties),
			}),
			buildFindingEvidenceItem("first_entry_before_crowding_count", fmt.Sprintf("%d", report.FirstEntryBeforeCrowdingCount), findingConfidenceFromScore(score), map[string]any{
				"bestLeadHoursBeforePeers": report.BestLeadHoursBeforePeers,
			}),
			buildFindingEvidenceItem("persistence_after_entry_proxy_count", fmt.Sprintf("%d", report.PersistenceAfterEntryProxyCount), findingConfidenceFromScore(score), map[string]any{
				"repeatEarlyEntrySuccess": repeatEarlyEntrySuccess,
			}),
			buildFindingEvidenceItem("historical_sustained_outcome_count", fmt.Sprintf("%d", report.HistoricalSustainedOutcomeCount), highConvictionConfidence, map[string]any{
				"repeatEarlyEntrySuccess": repeatEarlyEntrySuccess,
			}),
			buildFindingEvidenceItem("holding_persistence_state", firstNonEmpty(holdingPersistenceState, "monitoring"), highConvictionConfidence, map[string]any{
				"postWindowFollowThroughCount":  postWindowFollowThroughCount,
				"maxPostWindowPersistenceHours": maxPostWindowPersistenceHours,
			}),
		},
		NextWatch:                          buildNextWatchTargets(nextWatch),
		EntryHoldingPersistenceState:       holdingPersistenceState,
		EntryPostWindowFollowThroughCount:  postWindowFollowThroughCount,
		EntryMaxPostWindowPersistenceHours: maxPostWindowPersistenceHours,
	}
}

func firstConnectionNextWatchTargets(
	report FirstConnectionSnapshotReport,
	maturedPrior *db.WalletEntryFeaturesSnapshot,
) []domain.NextWatchTarget {
	out := make([]domain.NextWatchTarget, 0, 3)
	route := "early-convergence"
	if maturedPrior != nil {
		switch strings.TrimSpace(maturedPrior.HoldingPersistenceState) {
		case "sustained":
			route = "early-convergence-sustained"
		case "short_lived":
			route = "early-convergence-recheck"
		}
	}
	followThroughCount := 0
	maxPersistenceHours := 0
	holdingState := ""
	if maturedPrior != nil {
		followThroughCount = maturedPrior.PostWindowFollowThroughCount
		maxPersistenceHours = maturedPrior.MaxPostWindowPersistenceHours
		holdingState = strings.TrimSpace(maturedPrior.HoldingPersistenceState)
	}
	for _, item := range report.TopCounterparties {
		if strings.TrimSpace(item.Address) == "" {
			continue
		}
		label := compactAddress(item.Address)
		if route == "early-convergence-sustained" {
			label = label + " (sustained)"
		} else if route == "early-convergence-recheck" {
			label = label + " (recheck)"
		}
		out = append(out, domain.NextWatchTarget{
			SubjectType: domain.FindingSubjectWallet,
			Chain:       domain.Chain(strings.TrimSpace(item.Chain)),
			Address:     strings.TrimSpace(item.Address),
			Label:       label,
			Metadata: map[string]any{
				"peerWalletCount":               item.PeerWalletCount,
				"interactionCount":              item.InteractionCount,
				"leadHoursBeforePeers":          item.LeadHoursBeforePeers,
				"route":                         route,
				"holdingPersistenceState":       holdingState,
				"postWindowFollowThroughCount":  followThroughCount,
				"maxPostWindowPersistenceHours": maxPersistenceHours,
				"rankScore":                     firstConnectionNextWatchScore(item, maturedPrior),
			},
		})
		if len(out) == 2 {
			break
		}
	}
	sort.SliceStable(out, func(left int, right int) bool {
		leftScore, _ := out[left].Metadata["rankScore"].(int)
		rightScore, _ := out[right].Metadata["rankScore"].(int)
		return leftScore > rightScore
	})
	return out
}

func firstConnectionNextWatchScore(
	item FirstConnectionSnapshotCounterparty,
	maturedPrior *db.WalletEntryFeaturesSnapshot,
) int {
	score := int(item.PeerWalletCount*4 + item.InteractionCount*2 + minInt64(item.LeadHoursBeforePeers, 24))
	if maturedPrior == nil {
		return score
	}
	switch strings.TrimSpace(maturedPrior.HoldingPersistenceState) {
	case "sustained":
		score += 20
	case "short_lived":
		score -= 15
		if maturedPrior.PostWindowFollowThroughCount == 0 {
			score -= 5
		}
	}
	return score
}

func minInt64(left int64, right int64) int64 {
	if left < right {
		return left
	}
	return right
}

func shouldEmitHighConvictionEntry(report FirstConnectionSnapshotReport) bool {
	if report.QualityWalletOverlapCount <= 0 || report.FirstEntryBeforeCrowdingCount <= 0 {
		return false
	}

	if report.SustainedOverlapCounterpartyCount <= 0 && report.StrongLeadCounterpartyCount <= 0 {
		return false
	}

	return report.RepeatEarlyEntrySuccess || (report.PersistenceAfterEntryProxyCount > 0 && report.StrongLeadCounterpartyCount > 0)
}

func hasWalletLabel(set domain.WalletLabelSet, class domain.WalletLabelClass, fragment string) bool {
	normalizedFragment := strings.ToLower(strings.TrimSpace(fragment))
	if normalizedFragment == "" {
		return false
	}
	labels := make([]domain.WalletLabel, 0, len(set.Verified)+len(set.Inferred)+len(set.Behavioral))
	switch class {
	case domain.WalletLabelClassVerified:
		labels = append(labels, set.Verified...)
	case domain.WalletLabelClassInferred:
		labels = append(labels, set.Inferred...)
	case domain.WalletLabelClassBehavioral:
		labels = append(labels, set.Behavioral...)
	default:
		labels = append(labels, set.Verified...)
		labels = append(labels, set.Inferred...)
		labels = append(labels, set.Behavioral...)
	}

	for _, label := range labels {
		if strings.Contains(strings.ToLower(strings.TrimSpace(label.Key)), normalizedFragment) {
			return true
		}
		if strings.Contains(strings.ToLower(strings.TrimSpace(label.Name)), normalizedFragment) {
			return true
		}
		if strings.Contains(strings.ToLower(strings.TrimSpace(label.EntityType)), normalizedFragment) {
			return true
		}
	}
	return false
}

func mergeFindingEntry(base db.FindingEntry, extra db.FindingEntry) db.FindingEntry {
	base.FindingType = extra.FindingType
	base.Confidence = extra.Confidence
	base.ImportanceScore = extra.ImportanceScore
	base.Summary = extra.Summary
	base.DedupKey = extra.DedupKey
	base.Bundle = extra.Bundle
	return base
}

func walletLabelLookupKey(ref db.WalletRef) string {
	return strings.ToLower(strings.TrimSpace(string(ref.Chain))) + "|" + strings.ToLower(strings.TrimSpace(ref.Address))
}

func findingConfidenceFromScore(score domain.Score) float64 {
	return maxFloat(0.35, float64(score.Value)/100)
}

func mergeBundle(base map[string]any, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}

func maxFloat(values ...float64) float64 {
	best := 0.0
	for _, value := range values {
		if value > best {
			best = value
		}
	}
	return best
}

func maxInt(values ...int) int {
	best := 0
	for _, value := range values {
		if value > best {
			best = value
		}
	}
	return best
}
