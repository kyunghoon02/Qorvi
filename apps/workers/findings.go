package main

import (
	"context"
	"fmt"
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

func shadowExitFindingEntries(report ShadowExitSnapshotReport, score domain.Score) []db.FindingEntry {
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

	commonBundle := map[string]any{
		"evidence": buildFindingEvidence(score.Evidence),
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
			},
			"inferred_interpretations": []string{
				"Observed flow looks more like preparation to distribute than passive treasury movement.",
			},
		}),
	}, base)

	if report.BridgeEscapeCount > 0 {
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
				},
				"inferred_interpretations": []string{
					"The wallet may be repositioning cross-chain rather than exiting outright.",
				},
			}),
		}, base)
	} else if report.CEXProximityCount > 0 {
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
				},
				"inferred_interpretations": []string{
					"Some recent flow resembles deposit-like paths into exchange-adjacent addresses.",
				},
			}),
		}, base)
	}

	return out
}

func firstConnectionFindingEntry(report FirstConnectionSnapshotReport, score domain.Score) db.FindingEntry {
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
			},
			"observed_facts": []string{
				fmt.Sprintf("First-connection score rated %s at %d.", score.Rating, score.Value),
				fmt.Sprintf("New common entries: %d. First-seen counterparties: %d.", report.NewCommonEntries, report.FirstSeenCounterparties),
			},
			"inferred_interpretations": []string{
				"Activity suggests early convergence rather than repeated legacy flow.",
			},
			"evidence": buildFindingEvidence(score.Evidence),
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
	ImportanceReason        []string
	ObservedFacts           []string
	InferredInterpretations []string
	Evidence                []map[string]any
	NextWatch               []map[string]any
	AllowTreasuryRedistribution bool
	AllowMMHandoff             bool
	AllowFundAdjacentActivity  bool
	AllowHighConvictionEntry   bool
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
	if context.AllowTreasuryRedistribution && hasWalletLabel(labels, domain.WalletLabelClassInferred, "treasury") {
		findings = append(findings, mergeFindingEntry(base, newEntry(
			domain.FindingTypeTreasuryRedistribution,
			fmt.Sprintf("Treasury-like redistribution behavior is increasing around %s.", trimmedAddress),
			[]string{
				"Treasury attribution changes how recent fanout and transfer clustering should be interpreted.",
			},
			append([]string{
				"Wallet carries inferred treasury labeling in the current indexed coverage.",
				fmt.Sprintf("%s scored %s at %d.", score.Name, score.Rating, score.Value),
			}, context.ObservedFacts...),
			append([]string{
				"Recent flow may reflect treasury redistribution rather than opportunistic trading.",
			}, context.InferredInterpretations...),
			maxFloat(baseConfidence, 0.68),
			maxFloat(baseImportance, 0.62),
		)))
	}
	if context.AllowMMHandoff && hasWalletLabel(labels, domain.WalletLabelClassInferred, "market_maker") {
		findings = append(findings, mergeFindingEntry(base, newEntry(
			domain.FindingTypeSuspectedMMHandoff,
			fmt.Sprintf("A market-maker-like handoff pattern is forming around %s.", trimmedAddress),
			[]string{
				"Distribution-style flow attached to a market-maker-like wallet can precede inventory rotation.",
			},
			append([]string{
				"Wallet carries inferred market maker labeling in the current indexed coverage.",
				fmt.Sprintf("%s scored %s at %d.", score.Name, score.Rating, score.Value),
			}, context.ObservedFacts...),
			append([]string{
				"Observed flow resembles a handoff into market-making or distribution infrastructure.",
			}, context.InferredInterpretations...),
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
			maxFloat(baseConfidence, 0.72),
			maxFloat(baseImportance, 0.7),
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

func shadowExitInterpretationContext(report ShadowExitSnapshotReport, score domain.Score) interpretationFindingContext {
	nextWatch := make([]domain.NextWatchTarget, 0, 2)
	if report.BridgeEscapeCount > 0 {
		nextWatch = append(nextWatch, domain.NextWatchTarget{
			SubjectType: domain.FindingSubjectEntity,
			Label:       "Bridge-linked destination wallets",
		})
	}
	if report.CEXProximityCount > 0 {
		nextWatch = append(nextWatch, domain.NextWatchTarget{
			SubjectType: domain.FindingSubjectEntity,
			Label:       "Exchange-adjacent counterparties",
		})
	}

	return interpretationFindingContext{
		AllowTreasuryRedistribution: report.FanOutCount > 0 || report.FanOutCandidateCount24h > 0,
		AllowMMHandoff:             report.BridgeEscapeCount > 0 || report.CEXProximityCount > 0 || report.FanOutCount > 0,
		ObservedFacts: []string{
			fmt.Sprintf("Bridge escape count: %d. CEX proximity count: %d. Fanout count: %d.", report.BridgeEscapeCount, report.CEXProximityCount, report.FanOutCount),
			fmt.Sprintf("Outflow ratio reached %.2f within the indexed coverage window.", report.OutflowRatio),
		},
		InferredInterpretations: []string{
			"Recent flow looks more like redistribution or handoff behavior than passive holding.",
		},
		Evidence: []map[string]any{
			buildFindingEvidenceItem("bridge_escape_count", fmt.Sprintf("%d", report.BridgeEscapeCount), findingConfidenceFromScore(score), nil),
			buildFindingEvidenceItem("cex_proximity_count", fmt.Sprintf("%d", report.CEXProximityCount), findingConfidenceFromScore(score), nil),
			buildFindingEvidenceItem("fan_out_count", fmt.Sprintf("%d", report.FanOutCount), findingConfidenceFromScore(score), nil),
			buildFindingEvidenceItem("outflow_ratio", fmt.Sprintf("%.2f", report.OutflowRatio), findingConfidenceFromScore(score), nil),
		},
		NextWatch: buildNextWatchTargets(nextWatch),
	}
}

func firstConnectionInterpretationContext(report FirstConnectionSnapshotReport, score domain.Score) interpretationFindingContext {
	nextWatch := make([]domain.NextWatchTarget, 0, 2)
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

	return interpretationFindingContext{
		AllowHighConvictionEntry: report.NewCommonEntries > 0 && (report.FirstSeenCounterparties > 0 || report.HotFeedMentions > 0),
		ObservedFacts: []string{
			fmt.Sprintf("New common entries: %d. First-seen counterparties: %d. Hot feed mentions: %d.", report.NewCommonEntries, report.FirstSeenCounterparties, report.HotFeedMentions),
		},
		InferredInterpretations: []string{
			"These overlaps look more like early convergence than repeated legacy flow.",
		},
		Evidence: []map[string]any{
			buildFindingEvidenceItem("new_common_entries", fmt.Sprintf("%d", report.NewCommonEntries), findingConfidenceFromScore(score), nil),
			buildFindingEvidenceItem("first_seen_counterparties", fmt.Sprintf("%d", report.FirstSeenCounterparties), findingConfidenceFromScore(score), nil),
			buildFindingEvidenceItem("hot_feed_mentions", fmt.Sprintf("%d", report.HotFeedMentions), findingConfidenceFromScore(score), nil),
		},
		NextWatch: buildNextWatchTargets(nextWatch),
	}
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

func maxFloat(left float64, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
