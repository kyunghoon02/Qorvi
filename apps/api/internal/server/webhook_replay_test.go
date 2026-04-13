package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
	"github.com/qorvi/qorvi/packages/intelligence"
)

type replayScoreSummary struct {
	ClusterValue         int
	ClusterRating        domain.ScoreRating
	ShadowExitValue      int
	ShadowExitRating     domain.ScoreRating
	FirstConnectionValue int
	FirstConnectionRate  domain.ScoreRating
}

func TestReplayAlchemyWebhookFixtureReproducesTransactionsAndScores(t *testing.T) {
	t.Parallel()

	raw := readReplayFixture(t, "alchemy_address_activity.json")

	firstWrites, firstScores := replayAlchemyFixture(t, raw)
	secondWrites, secondScores := replayAlchemyFixture(t, raw)

	if !reflect.DeepEqual(firstWrites, secondWrites) {
		t.Fatalf("expected replayed normalized transactions to match\nfirst=%#v\nsecond=%#v", firstWrites, secondWrites)
	}
	if !reflect.DeepEqual(firstScores, secondScores) {
		t.Fatalf("expected replayed score summaries to match\nfirst=%#v\nsecond=%#v", firstScores, secondScores)
	}
	if firstScores.ClusterValue == 0 || firstScores.ShadowExitValue == 0 || firstScores.FirstConnectionValue == 0 {
		t.Fatalf("expected non-zero score summaries, got %#v", firstScores)
	}
}

func TestReplayHeliusWebhookFixtureReproducesNormalizedTransactions(t *testing.T) {
	t.Parallel()

	raw := readReplayFixture(t, "helius_address_activity.json")

	firstWrites := replayWebhookTransactions(t, "helius", raw)
	secondWrites := replayWebhookTransactions(t, "helius", raw)

	if !reflect.DeepEqual(firstWrites, secondWrites) {
		t.Fatalf("expected replayed helius normalized transactions to match\nfirst=%#v\nsecond=%#v", firstWrites, secondWrites)
	}
	if len(firstWrites) == 0 {
		t.Fatal("expected replayed helius writes")
	}
}

func readReplayFixture(t *testing.T, fileName string) []byte {
	t.Helper()

	path := filepath.Join("testdata", "replay", fileName)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", path, err)
	}

	return body
}

func replayAlchemyFixture(t *testing.T, raw []byte) ([]db.NormalizedTransactionWrite, replayScoreSummary) {
	t.Helper()

	writes := replayWebhookTransactions(t, "alchemy", raw)
	scores := deriveReplayScoreSummary(
		"wallet:0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		domain.ChainEVM,
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		writes,
	)

	return writes, scores
}

func replayWebhookTransactions(t *testing.T, provider string, raw []byte) []db.NormalizedTransactionWrite {
	t.Helper()

	wallets := &fakeAPIWebhookWalletStore{}
	transactions := &fakeAPIWebhookTransactionStore{}
	dailyStats := &fakeAPIWebhookDailyStatsStore{}
	summaryCache := &fakeAPIWebhookSummaryCache{}
	graphCache := &fakeAPIWebhookGraphCache{}
	graphSnapshots := &fakeAPIWebhookGraphSnapshotStore{}
	graph := &fakeAPIWebhookGraphMaterializer{}
	dedup := &fakeAPIWebhookDedupStore{}
	rawPayloads := &fakeAPIWebhookRawPayloadStore{}
	providerUsage := &fakeAPIWebhookProviderUsageStore{}
	jobRuns := &fakeAPIWebhookJobRunStore{}
	entityAssignments := &fakeAPIWebhookEntityAssignmentStore{}
	labeling := &fakeAPIWebhookLabelingStore{}
	tracking := &fakeAPIWebhookTrackingStateStore{}

	service := NewWebhookIngestService(
		wallets,
		entityAssignments,
		labeling,
		transactions,
		dailyStats,
		graph,
		graphCache,
		graphSnapshots,
		summaryCache,
		dedup,
		rawPayloads,
		providerUsage,
		jobRuns,
		tracking,
	)
	persisting := service.(providerWebhookPersistingService)
	persisting.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
	}

	if provider == "alchemy" {
		var payload AlchemyAddressActivityWebhook
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("unmarshal alchemy replay payload: %v", err)
		}
		if _, err := persisting.IngestAlchemyAddressActivity(context.Background(), payload); err != nil {
			t.Fatalf("IngestAlchemyAddressActivity returned error: %v", err)
		}
	} else {
		if _, err := persisting.IngestProviderWebhook(context.Background(), provider, raw); err != nil {
			t.Fatalf("IngestProviderWebhook(%s) returned error: %v", provider, err)
		}
	}

	return flattenAndSortWrites(transactions.writes)
}

func flattenAndSortWrites(batches []db.NormalizedTransactionWrite) []db.NormalizedTransactionWrite {
	flattened := make([]db.NormalizedTransactionWrite, 0, len(batches))
	flattened = append(flattened, batches...)
	sort.Slice(flattened, func(i, j int) bool {
		left := flattened[i]
		right := flattened[j]
		if left.WalletID != right.WalletID {
			return left.WalletID < right.WalletID
		}
		if left.Transaction.Wallet.Address != right.Transaction.Wallet.Address {
			return left.Transaction.Wallet.Address < right.Transaction.Wallet.Address
		}
		if left.Transaction.TxHash != right.Transaction.TxHash {
			return left.Transaction.TxHash < right.Transaction.TxHash
		}
		return left.Transaction.Direction < right.Transaction.Direction
	})
	return flattened
}

func deriveReplayScoreSummary(
	rootWalletID string,
	rootChain domain.Chain,
	rootAddress string,
	writes []db.NormalizedTransactionWrite,
) replayScoreSummary {
	graph := buildReplayWalletGraph(rootWalletID, rootChain, rootAddress, writes)
	clusterScore := intelligence.BuildClusterScoreFromWalletGraph(graph, "2026-03-20T09:10:11Z")

	shadowSignal := intelligence.BuildShadowExitSignalFromInputs(buildReplayShadowExitInputs(rootWalletID, rootChain, rootAddress, writes))
	shadowScore := intelligence.BuildShadowExitRiskScore(shadowSignal)

	firstConnectionSignal := intelligence.BuildFirstConnectionSignalFromInputs(buildReplayFirstConnectionInputs(rootWalletID, rootChain, rootAddress, writes))
	firstConnectionScore := intelligence.BuildFirstConnectionScore(firstConnectionSignal)

	return replayScoreSummary{
		ClusterValue:         clusterScore.Value,
		ClusterRating:        clusterScore.Rating,
		ShadowExitValue:      shadowScore.Value,
		ShadowExitRating:     shadowScore.Rating,
		FirstConnectionValue: firstConnectionScore.Value,
		FirstConnectionRate:  firstConnectionScore.Rating,
	}
}

func buildReplayWalletGraph(
	rootWalletID string,
	rootChain domain.Chain,
	rootAddress string,
	writes []db.NormalizedTransactionWrite,
) domain.WalletGraph {
	type aggregate struct {
		first time.Time
		last  time.Time
		count int
	}

	counterparties := map[string]*aggregate{}
	for _, write := range writes {
		tx := write.Transaction
		if write.WalletID != rootWalletID || tx.Counterparty == nil {
			continue
		}
		key := strings.TrimSpace(tx.Counterparty.Address)
		if key == "" {
			continue
		}
		item := counterparties[key]
		if item == nil {
			item = &aggregate{first: tx.ObservedAt.UTC(), last: tx.ObservedAt.UTC()}
			counterparties[key] = item
		}
		if tx.ObservedAt.Before(item.first) {
			item.first = tx.ObservedAt.UTC()
		}
		if tx.ObservedAt.After(item.last) {
			item.last = tx.ObservedAt.UTC()
		}
		item.count++
	}

	edges := make([]domain.WalletGraphEdge, 0, len(counterparties))
	for address, item := range counterparties {
		edges = append(edges, domain.WalletGraphEdge{
			SourceID:          "wallet:" + rootAddress,
			TargetID:          "wallet:" + address,
			Kind:              domain.WalletGraphEdgeInteractedWith,
			FirstObservedAt:   item.first.Format(time.RFC3339),
			ObservedAt:        item.last.Format(time.RFC3339),
			CounterpartyCount: item.count,
			Weight:            item.count,
		})
	}
	sort.Slice(edges, func(i, j int) bool { return edges[i].TargetID < edges[j].TargetID })

	return domain.WalletGraph{
		Chain:   rootChain,
		Address: rootAddress,
		Edges:   edges,
	}
}

func buildReplayShadowExitInputs(
	rootWalletID string,
	rootChain domain.Chain,
	rootAddress string,
	writes []db.NormalizedTransactionWrite,
) intelligence.ShadowExitDetectorInputs {
	outboundCounterparties := map[string]struct{}{}
	inboundCount := 0
	outboundCount := 0
	bridgeCount := 0
	cexCount := 0

	for _, write := range writes {
		tx := write.Transaction
		if write.WalletID != rootWalletID || tx.Counterparty == nil {
			continue
		}
		address := strings.TrimSpace(tx.Counterparty.Address)
		if address == "" {
			continue
		}
		switch tx.Direction {
		case domain.TransactionDirectionInbound:
			inboundCount++
		case domain.TransactionDirectionOutbound:
			outboundCount++
			outboundCounterparties[address] = struct{}{}
			if address == "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
				bridgeCount++
			}
			if address == "0xcccccccccccccccccccccccccccccccccccccccc" {
				cexCount++
			}
		}
	}

	return intelligence.ShadowExitDetectorInputs{
		WalletID:                       rootWalletID,
		Chain:                          rootChain,
		Address:                        rootAddress,
		ObservedAt:                     "2026-03-20T09:10:11Z",
		BridgeTransfers:                bridgeCount,
		CEXProximityCount:              cexCount,
		FanOutCount:                    len(outboundCounterparties),
		FanOutCandidateCount24h:        len(outboundCounterparties),
		OutboundTransferCount24h:       outboundCount,
		InboundTransferCount24h:        inboundCount,
		BridgeEscapeCount:              bridgeCount,
		TreasuryWhitelistEvidenceCount: 0,
		InternalRebalanceEvidenceCount: 0,
	}
}

func buildReplayFirstConnectionInputs(
	rootWalletID string,
	rootChain domain.Chain,
	rootAddress string,
	writes []db.NormalizedTransactionWrite,
) intelligence.FirstConnectionDetectorInputs {
	rootCounterparties := map[string]struct{}{}
	peerHits := map[string]map[string]struct{}{}

	for _, write := range writes {
		tx := write.Transaction
		if tx.Counterparty == nil {
			continue
		}
		counterparty := strings.TrimSpace(tx.Counterparty.Address)
		if counterparty == "" {
			continue
		}
		if write.WalletID == rootWalletID {
			rootCounterparties[counterparty] = struct{}{}
			continue
		}
		if write.Transaction.Wallet.Address == counterparty {
			continue
		}
		if peerHits[counterparty] == nil {
			peerHits[counterparty] = map[string]struct{}{}
		}
		peerHits[counterparty][write.Transaction.Wallet.Address] = struct{}{}
	}

	newCommonEntries := 0
	hotFeedMentions := 0
	for counterparty := range rootCounterparties {
		peerWallets := peerHits[counterparty]
		if len(peerWallets) == 0 {
			continue
		}
		newCommonEntries++
		hotFeedMentions += len(peerWallets)
	}

	return intelligence.FirstConnectionDetectorInputs{
		WalletID:                rootWalletID,
		Chain:                   rootChain,
		Address:                 rootAddress,
		ObservedAt:              "2026-03-20T09:10:11Z",
		NewCommonEntries:        newCommonEntries,
		FirstSeenCounterparties: len(rootCounterparties),
		HotFeedMentions:         hotFeedMentions,
	}
}
