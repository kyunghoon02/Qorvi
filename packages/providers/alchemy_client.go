package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
)

type AlchemyClient struct {
	baseURL       string
	solanaBaseURL string
	apiKey        string
	http          jsonHTTPClient
}

func NewAlchemyClient(credentials ProviderCredentials, client *http.Client) *AlchemyClient {
	return &AlchemyClient{
		baseURL:       strings.TrimRight(credentials.BaseURL, "/"),
		solanaBaseURL: strings.TrimRight(credentials.SolanaBaseURL, "/"),
		apiKey:        strings.TrimSpace(credentials.APIKey),
		http:          newJSONHTTPClient(client),
	}
}

func (c *AlchemyClient) FetchHistoricalWalletActivity(batch HistoricalBackfillBatch) ([]ProviderWalletActivity, error) {
	if c == nil {
		return nil, fmt.Errorf("alchemy client is nil")
	}
	if err := batch.Validate(); err != nil {
		return nil, err
	}
	if batch.Request.Chain == domain.ChainSolana {
		return c.fetchHistoricalSolanaWalletActivity(batch)
	}
	endpoint, err := c.endpoint()
	if err != nil {
		return nil, err
	}

	activities := make([]ProviderWalletActivity, 0, batch.Limit)
	seenTransfers := map[string]struct{}{}
	directionLimits := splitAlchemyDirectionLimit(batch.Limit)
	queryPlans := []alchemyAssetTransfersParams{
		{
			FromBlock:        "0x0",
			ToBlock:          "latest",
			ToAddress:        batch.Request.WalletAddress,
			Category:         []string{"external", "erc20", "erc721", "erc1155"},
			WithMetadata:     true,
			ExcludeZeroValue: true,
			MaxCount:         formatAlchemyQuantity(minInt(directionLimits[0], 1000)),
			Order:            "desc",
		},
		{
			FromBlock:        "0x0",
			ToBlock:          "latest",
			FromAddress:      batch.Request.WalletAddress,
			Category:         []string{"external", "erc20", "erc721", "erc1155"},
			WithMetadata:     true,
			ExcludeZeroValue: true,
			MaxCount:         formatAlchemyQuantity(minInt(directionLimits[1], 1000)),
			Order:            "desc",
		},
	}

	for _, queryPlan := range queryPlans {
		if len(activities) >= batch.Limit {
			break
		}

		directionalActivities, err := c.fetchHistoricalTransferActivities(
			endpoint,
			batch,
			queryPlan,
			len(activities),
			seenTransfers,
		)
		if err != nil {
			return nil, err
		}

		activities = append(activities, directionalActivities...)
	}

	return activities, nil
}

func (c *AlchemyClient) fetchHistoricalSolanaWalletActivity(batch HistoricalBackfillBatch) ([]ProviderWalletActivity, error) {
	endpoint, err := c.solanaEndpoint()
	if err != nil {
		return nil, err
	}

	signatures, err := c.fetchSolanaSignatures(endpoint, batch)
	if err != nil {
		return nil, err
	}
	if len(signatures) == 0 {
		return []ProviderWalletActivity{}, nil
	}

	activities := make([]ProviderWalletActivity, 0, len(signatures))
	for index, signature := range signatures {
		if len(activities) >= batch.Limit {
			break
		}

		tx, metadata, err := c.fetchSolanaTransaction(endpoint, batch, signature)
		if err != nil {
			return nil, err
		}
		if tx == nil {
			continue
		}

		activities = append(activities, alchemySolanaTransactionToActivity(batch, *tx, index, metadata))
	}

	return activities, nil
}

func (c *AlchemyClient) fetchHistoricalTransferActivities(
	endpoint string,
	batch HistoricalBackfillBatch,
	queryPlan alchemyAssetTransfersParams,
	startIndex int,
	seenTransfers map[string]struct{},
) ([]ProviderWalletActivity, error) {
	targetLimit := parseAlchemyQuantity(queryPlan.MaxCount)
	if targetLimit <= 0 {
		return nil, nil
	}

	activities := make([]ProviderWalletActivity, 0, targetLimit)
	pageKey := ""

	for len(activities) < targetLimit {
		nextQuery := queryPlan
		nextQuery.PageKey = pageKey
		nextQuery.MaxCount = formatAlchemyQuantity(minInt(targetLimit-len(activities), 1000))

		requestBody := alchemyAssetTransfersRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "alchemy_getAssetTransfers",
			Params:  []alchemyAssetTransfersParams{nextQuery},
		}

		req, err := newJSONRequest(http.MethodPost, endpoint, requestBody)
		if err != nil {
			return nil, err
		}

		response := alchemyAssetTransfersResponse{}
		rawBody, err := c.http.doJSONRequestWithRaw(req, &response)
		if err != nil {
			return nil, err
		}
		if response.Error != nil {
			return nil, fmt.Errorf("alchemy transfers api error: %s", response.Error.Message)
		}

		pageMetadata := capturePagePayloadMetadata(
			ProviderAlchemy,
			"alchemy_getAssetTransfers",
			batch.WindowEnd,
			response.Result.PageKey,
			rawBody,
			map[string]any{
				"response_page_key": response.Result.PageKey,
				"response_count":    len(response.Result.Transfers),
			},
		)

		for _, transfer := range response.Result.Transfers {
			transferKey := buildAlchemyTransferKey(transfer)
			if _, seen := seenTransfers[transferKey]; seen {
				continue
			}

			seenTransfers[transferKey] = struct{}{}
			activities = append(
				activities,
				alchemyTransferToActivity(batch, transfer, startIndex+len(activities), pageMetadata),
			)
			if len(activities) >= targetLimit {
				break
			}
		}

		pageKey = response.Result.PageKey
		if pageKey == "" {
			break
		}
	}

	return activities, nil
}

func (c *AlchemyClient) fetchSolanaSignatures(endpoint string, batch HistoricalBackfillBatch) ([]solanaSignatureInfo, error) {
	signatures := make([]solanaSignatureInfo, 0, batch.Limit)
	before := ""

	for len(signatures) < batch.Limit {
		requestBody := solanaRPCRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "getSignaturesForAddress",
			Params: []any{
				batch.Request.WalletAddress,
				map[string]any{
					"limit":  minInt(batch.Limit-len(signatures), 1000),
					"before": before,
				},
			},
		}
		if before == "" {
			requestBody.Params[1] = map[string]any{
				"limit": minInt(batch.Limit-len(signatures), 1000),
			}
		}

		req, err := newJSONRequest(http.MethodPost, endpoint, requestBody)
		if err != nil {
			return nil, err
		}

		var response solanaSignaturesResponse
		if _, err := c.http.doJSONRequestWithRaw(req, &response); err != nil {
			return nil, err
		}
		if response.Error != nil {
			return nil, fmt.Errorf("alchemy solana signatures api error: %s", response.Error.Message)
		}
		if len(response.Result) == 0 {
			break
		}

		stop := false
		for _, item := range response.Result {
			if item.BlockTime > 0 {
				observedAt := time.Unix(item.BlockTime, 0).UTC()
				if observedAt.Before(batch.WindowStart) {
					stop = true
					break
				}
				if observedAt.After(batch.WindowEnd) {
					continue
				}
			}
			signatures = append(signatures, item)
			if len(signatures) >= batch.Limit {
				break
			}
		}
		if stop || len(signatures) >= batch.Limit {
			break
		}

		before = response.Result[len(response.Result)-1].Signature
		if before == "" {
			break
		}
	}

	return signatures, nil
}

func (c *AlchemyClient) fetchSolanaTransaction(endpoint string, batch HistoricalBackfillBatch, signature solanaSignatureInfo) (*solanaTransactionResult, map[string]any, error) {
	requestBody := solanaRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "getTransaction",
		Params: []any{
			signature.Signature,
			map[string]any{
				"encoding":                       "jsonParsed",
				"maxSupportedTransactionVersion": 0,
			},
		},
	}

	req, err := newJSONRequest(http.MethodPost, endpoint, requestBody)
	if err != nil {
		return nil, nil, err
	}

	var response solanaTransactionResponse
	rawBody, err := c.http.doJSONRequestWithRaw(req, &response)
	if err != nil {
		return nil, nil, err
	}
	if response.Error != nil {
		return nil, nil, fmt.Errorf("alchemy solana transaction api error: %s", response.Error.Message)
	}
	if response.Result == nil {
		return nil, nil, nil
	}

	metadata := capturePagePayloadMetadata(
		ProviderAlchemy,
		"solana_getTransaction",
		batch.WindowEnd,
		signature.Signature,
		rawBody,
		map[string]any{
			"response_slot": response.Result.Slot,
		},
	)

	return response.Result, metadata, nil
}

func (c *AlchemyClient) endpoint() (string, error) {
	return c.alchemyEndpoint(c.baseURL)
}

func (c *AlchemyClient) solanaEndpoint() (string, error) {
	baseURL := c.solanaBaseURL
	if strings.TrimSpace(baseURL) == "" {
		baseURL = c.baseURL
	}
	return c.alchemyEndpoint(baseURL)
}

func (c *AlchemyClient) alchemyEndpoint(baseURL string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse alchemy base url: %w", err)
	}
	trimmedPath := strings.TrimRight(parsed.Path, "/")
	if strings.Contains(trimmedPath, "/v2/") && !strings.HasSuffix(trimmedPath, "/v2") {
		return parsed.String(), nil
	}
	parsed.Path = trimmedPath + "/v2/" + url.PathEscape(c.apiKey)
	parsed.RawQuery = ""
	return parsed.String(), nil
}

type alchemyAssetTransfersRequest struct {
	JSONRPC string                        `json:"jsonrpc"`
	ID      int                           `json:"id"`
	Method  string                        `json:"method"`
	Params  []alchemyAssetTransfersParams `json:"params"`
}

type alchemyAssetTransfersParams struct {
	FromBlock        string   `json:"fromBlock"`
	ToBlock          string   `json:"toBlock"`
	FromAddress      string   `json:"fromAddress,omitempty"`
	ToAddress        string   `json:"toAddress,omitempty"`
	Category         []string `json:"category,omitempty"`
	WithMetadata     bool     `json:"withMetadata"`
	ExcludeZeroValue bool     `json:"excludeZeroValue"`
	MaxCount         string   `json:"maxCount"`
	PageKey          string   `json:"pageKey,omitempty"`
	Order            string   `json:"order,omitempty"`
}

type alchemyAssetTransfersResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Transfers []alchemyAssetTransfer `json:"transfers"`
		PageKey   string                 `json:"pageKey"`
	} `json:"result"`
	Error *alchemyRPCError `json:"error,omitempty"`
}

type alchemyRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type alchemyAssetTransfer struct {
	BlockNum    string `json:"blockNum"`
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       any    `json:"value"`
	Asset       string `json:"asset"`
	Category    string `json:"category"`
	RawContract struct {
		Value   string `json:"value"`
		Address string `json:"address"`
		Decimal string `json:"decimal"`
	} `json:"rawContract"`
}

func alchemyTransferToActivity(batch HistoricalBackfillBatch, transfer alchemyAssetTransfer, index int, pageMetadata map[string]any) ProviderWalletActivity {
	amount := normalizeAlchemyTransferValue(transfer.Value)
	metadata := mergeMetadata(pageMetadata, map[string]any{
		"tx_hash":              transfer.Hash,
		"raw_payload_path":     fmt.Sprintf("alchemy://transfers/%s", transfer.Hash),
		"direction":            alchemyTransferDirection(batch.Request.WalletAddress, transfer),
		"amount":               amount,
		"block_number":         parseHexInt64(transfer.BlockNum),
		"transaction_index":    index,
		"kind":                 transfer.Category,
		"token_symbol":         transfer.Asset,
		"token_address":        transfer.RawContract.Address,
		"token_decimals":       parseHexInt(transfer.RawContract.Decimal),
		"counterparty_address": alchemyCounterparty(batch.Request.WalletAddress, transfer),
	})

	return CreateProviderActivityFixture(ProviderActivityFixtureInput{
		Provider:      ProviderAlchemy,
		Chain:         batch.Request.Chain,
		WalletAddress: batch.Request.WalletAddress,
		SourceID:      "alchemy_getAssetTransfers",
		Kind:          "transfer",
		Confidence:    0.91,
		ObservedAt:    batch.WindowEnd.Add(-time.Duration(index) * time.Minute),
		Metadata:      metadata,
	})
}

func alchemyTransferDirection(walletAddress string, transfer alchemyAssetTransfer) string {
	switch {
	case strings.EqualFold(transfer.From, walletAddress) && strings.EqualFold(transfer.To, walletAddress):
		return string(domain.TransactionDirectionSelf)
	case strings.EqualFold(transfer.From, walletAddress):
		return string(domain.TransactionDirectionOutbound)
	case strings.EqualFold(transfer.To, walletAddress):
		return string(domain.TransactionDirectionInbound)
	default:
		return string(domain.TransactionDirectionUnknown)
	}
}

func alchemyCounterparty(walletAddress string, transfer alchemyAssetTransfer) string {
	switch {
	case strings.EqualFold(transfer.From, walletAddress):
		return transfer.To
	case strings.EqualFold(transfer.To, walletAddress):
		return transfer.From
	default:
		return ""
	}
}

func buildAlchemyTransferKey(transfer alchemyAssetTransfer) string {
	return strings.Join([]string{
		strings.TrimSpace(transfer.Hash),
		strings.TrimSpace(transfer.BlockNum),
		strings.ToLower(strings.TrimSpace(transfer.From)),
		strings.ToLower(strings.TrimSpace(transfer.To)),
		strings.TrimSpace(transfer.Asset),
		normalizeAlchemyTransferValue(transfer.Value),
		strings.TrimSpace(transfer.Category),
	}, "|")
}

func normalizeAlchemyTransferValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return normalizeNilLikeAmountString(typed)
	default:
		return normalizeNilLikeAmountString(fmt.Sprint(typed))
	}
}

func normalizeNilLikeAmountString(raw string) string {
	trimmed := strings.TrimSpace(raw)
	switch strings.ToLower(trimmed) {
	case "", "<nil>", "nil", "null", "<null>":
		return ""
	default:
		return trimmed
	}
}

func parseHexInt64(raw string) int64 {
	value, _ := strconv.ParseInt(strings.TrimPrefix(strings.TrimSpace(raw), "0x"), 16, 64)
	return value
}

func parseHexInt(raw string) int {
	value, _ := strconv.ParseInt(strings.TrimPrefix(strings.TrimSpace(raw), "0x"), 16, 64)
	return int(value)
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func formatAlchemyQuantity(value int) string {
	if value < 0 {
		value = 0
	}

	return fmt.Sprintf("0x%x", value)
}

func parseAlchemyQuantity(raw string) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}

	parsed, err := strconv.ParseInt(strings.TrimPrefix(trimmed, "0x"), 16, 64)
	if err != nil {
		return 0
	}

	return int(parsed)
}

func splitAlchemyDirectionLimit(limit int) [2]int {
	if limit <= 1 {
		return [2]int{limit, limit}
	}

	inbound := (limit + 1) / 2
	outbound := limit / 2
	return [2]int{inbound, outbound}
}

type solanaRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type solanaSignaturesResponse struct {
	JSONRPC string                `json:"jsonrpc"`
	ID      int                   `json:"id"`
	Result  []solanaSignatureInfo `json:"result"`
	Error   *alchemyRPCError      `json:"error,omitempty"`
}

type solanaSignatureInfo struct {
	Signature string `json:"signature"`
	Slot      int64  `json:"slot"`
	BlockTime int64  `json:"blockTime"`
}

type solanaTransactionResponse struct {
	JSONRPC string                   `json:"jsonrpc"`
	ID      int                      `json:"id"`
	Result  *solanaTransactionResult `json:"result"`
	Error   *alchemyRPCError         `json:"error,omitempty"`
}

type solanaTransactionResult struct {
	Slot        int64                   `json:"slot"`
	BlockTime   int64                   `json:"blockTime"`
	Transaction solanaParsedTransaction `json:"transaction"`
	Meta        solanaTransactionMeta   `json:"meta"`
}

type solanaParsedTransaction struct {
	Signatures []string            `json:"signatures"`
	Message    solanaParsedMessage `json:"message"`
}

type solanaParsedMessage struct {
	AccountKeys []solanaAccountKey `json:"accountKeys"`
}

type solanaAccountKey struct {
	Pubkey string
}

func (k *solanaAccountKey) UnmarshalJSON(data []byte) error {
	var rawString string
	if err := json.Unmarshal(data, &rawString); err == nil {
		k.Pubkey = rawString
		return nil
	}

	var rawObject struct {
		Pubkey string `json:"pubkey"`
	}
	if err := json.Unmarshal(data, &rawObject); err != nil {
		return err
	}
	k.Pubkey = rawObject.Pubkey
	return nil
}

type solanaTransactionMeta struct {
	PreBalances       []int64              `json:"preBalances"`
	PostBalances      []int64              `json:"postBalances"`
	PreTokenBalances  []solanaTokenBalance `json:"preTokenBalances"`
	PostTokenBalances []solanaTokenBalance `json:"postTokenBalances"`
}

type solanaTokenBalance struct {
	AccountIndex  int                 `json:"accountIndex"`
	Mint          string              `json:"mint"`
	Owner         string              `json:"owner"`
	UITokenAmount solanaUITokenAmount `json:"uiTokenAmount"`
}

type solanaUITokenAmount struct {
	Amount         string `json:"amount"`
	Decimals       int    `json:"decimals"`
	UIAmountString string `json:"uiAmountString"`
}

func alchemySolanaTransactionToActivity(
	batch HistoricalBackfillBatch,
	tx solanaTransactionResult,
	index int,
	pageMetadata map[string]any,
) ProviderWalletActivity {
	observedAt := batch.WindowEnd.Add(-time.Duration(index) * time.Minute)
	if tx.BlockTime > 0 {
		observedAt = time.Unix(tx.BlockTime, 0).UTC()
	}

	metadata := mergeMetadata(pageMetadata, buildSolanaHistoricalTransferMetadata(batch.Request.WalletAddress, tx))
	metadata = mergeMetadata(metadata, map[string]any{
		"tx_hash":           firstString(tx.Transaction.Signatures),
		"raw_payload_path":  fmt.Sprintf("alchemy://solana/transactions/%s", firstString(tx.Transaction.Signatures)),
		"block_number":      tx.Slot,
		"transaction_index": index,
	})
	if _, ok := metadata["direction"]; !ok {
		metadata["direction"] = string(domain.TransactionDirectionUnknown)
	}

	return CreateProviderActivityFixture(ProviderActivityFixtureInput{
		Provider:      ProviderAlchemy,
		Chain:         batch.Request.Chain,
		WalletAddress: batch.Request.WalletAddress,
		SourceID:      "solana_getTransaction",
		Kind:          "transaction",
		Confidence:    0.82,
		ObservedAt:    observedAt,
		Metadata:      metadata,
	})
}

func buildSolanaHistoricalTransferMetadata(walletAddress string, tx solanaTransactionResult) map[string]any {
	tokenSeed := deriveSolanaTokenTransferSeed(walletAddress, tx.Meta.PreTokenBalances, tx.Meta.PostTokenBalances)
	if tokenSeed.amount != "" {
		return tokenSeed.toMetadata()
	}

	nativeSeed := deriveSolanaNativeTransferSeed(walletAddress, tx.Transaction.Message.AccountKeys, tx.Meta.PreBalances, tx.Meta.PostBalances)
	return nativeSeed.toMetadata()
}

type solanaTransferSeed struct {
	direction     domain.TransactionDirection
	counterparty  string
	amount        string
	tokenAddress  string
	tokenSymbol   string
	tokenDecimals int
	funderAddress string
}

func (s solanaTransferSeed) toMetadata() map[string]any {
	metadata := map[string]any{}
	if s.direction != "" {
		metadata["direction"] = string(s.direction)
	}
	if s.counterparty != "" {
		metadata["counterparty_address"] = s.counterparty
		metadata["counterparty_chain"] = string(domain.ChainSolana)
	}
	if s.amount != "" {
		metadata["amount"] = s.amount
	}
	if s.tokenAddress != "" {
		metadata["token_address"] = s.tokenAddress
		metadata["token_chain"] = string(domain.ChainSolana)
	}
	if s.tokenSymbol != "" {
		metadata["token_symbol"] = s.tokenSymbol
	}
	if s.tokenDecimals > 0 {
		metadata["token_decimals"] = s.tokenDecimals
	}
	if s.funderAddress != "" {
		metadata["funder_address"] = s.funderAddress
	}
	return metadata
}

func deriveSolanaNativeTransferSeed(walletAddress string, accountKeys []solanaAccountKey, preBalances []int64, postBalances []int64) solanaTransferSeed {
	walletIndex := -1
	for index, key := range accountKeys {
		if strings.EqualFold(strings.TrimSpace(key.Pubkey), strings.TrimSpace(walletAddress)) {
			walletIndex = index
			break
		}
	}
	if walletIndex < 0 || walletIndex >= len(preBalances) || walletIndex >= len(postBalances) {
		return solanaTransferSeed{direction: domain.TransactionDirectionUnknown}
	}

	walletDelta := postBalances[walletIndex] - preBalances[walletIndex]
	if walletDelta == 0 {
		return solanaTransferSeed{direction: domain.TransactionDirectionUnknown}
	}

	seed := solanaTransferSeed{
		direction: directionForSignedDelta(walletDelta),
		amount:    strconv.FormatInt(absInt64(walletDelta), 10),
	}

	var bestCounterparty string
	var bestCounterpartyMagnitude int64
	for index, key := range accountKeys {
		if index >= len(preBalances) || index >= len(postBalances) || index == walletIndex {
			continue
		}
		delta := postBalances[index] - preBalances[index]
		if !isOppositeSignedDelta(walletDelta, delta) {
			continue
		}
		magnitude := absInt64(delta)
		if magnitude > bestCounterpartyMagnitude {
			bestCounterpartyMagnitude = magnitude
			bestCounterparty = strings.TrimSpace(key.Pubkey)
		}
	}

	seed.counterparty = bestCounterparty
	if seed.direction == domain.TransactionDirectionInbound && bestCounterparty != "" {
		seed.funderAddress = bestCounterparty
	}

	return seed
}

func deriveSolanaTokenTransferSeed(walletAddress string, pre []solanaTokenBalance, post []solanaTokenBalance) solanaTransferSeed {
	type tokenOwnerState struct {
		amount   int64
		decimals int
	}
	type mintState struct {
		owners   map[string]tokenOwnerState
		decimals int
	}

	mints := map[string]*mintState{}
	accumulate := func(items []solanaTokenBalance, multiplier int64) {
		for _, item := range items {
			mint := strings.TrimSpace(item.Mint)
			owner := strings.TrimSpace(item.Owner)
			if mint == "" || owner == "" {
				continue
			}
			amount, err := strconv.ParseInt(strings.TrimSpace(item.UITokenAmount.Amount), 10, 64)
			if err != nil {
				continue
			}
			state := mints[mint]
			if state == nil {
				state = &mintState{owners: map[string]tokenOwnerState{}, decimals: item.UITokenAmount.Decimals}
				mints[mint] = state
			}
			existing := state.owners[owner]
			existing.amount += amount * multiplier
			if item.UITokenAmount.Decimals > 0 {
				existing.decimals = item.UITokenAmount.Decimals
			}
			state.owners[owner] = existing
		}
	}

	accumulate(pre, -1)
	accumulate(post, 1)

	type candidate struct {
		mint         string
		delta        int64
		counterparty string
		decimals     int
	}

	candidates := make([]candidate, 0, len(mints))
	walletAddress = strings.TrimSpace(walletAddress)
	for mint, state := range mints {
		walletDelta := state.owners[walletAddress].amount
		if walletDelta == 0 {
			continue
		}

		bestCounterparty := ""
		var bestMagnitude int64
		for owner, ownerState := range state.owners {
			if strings.EqualFold(owner, walletAddress) || ownerState.amount == 0 {
				continue
			}
			if !isOppositeSignedDelta(walletDelta, ownerState.amount) {
				continue
			}
			if magnitude := absInt64(ownerState.amount); magnitude > bestMagnitude {
				bestMagnitude = magnitude
				bestCounterparty = owner
			}
		}

		candidates = append(candidates, candidate{
			mint:         mint,
			delta:        walletDelta,
			counterparty: bestCounterparty,
			decimals:     state.decimals,
		})
	}

	if len(candidates) == 0 {
		return solanaTransferSeed{direction: domain.TransactionDirectionUnknown}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return absInt64(candidates[i].delta) > absInt64(candidates[j].delta)
	})
	best := candidates[0]

	seed := solanaTransferSeed{
		direction:     directionForSignedDelta(best.delta),
		counterparty:  best.counterparty,
		amount:        strconv.FormatInt(absInt64(best.delta), 10),
		tokenAddress:  best.mint,
		tokenDecimals: best.decimals,
	}
	if seed.direction == domain.TransactionDirectionInbound && best.counterparty != "" {
		seed.funderAddress = best.counterparty
	}

	return seed
}

func directionForSignedDelta(delta int64) domain.TransactionDirection {
	switch {
	case delta > 0:
		return domain.TransactionDirectionInbound
	case delta < 0:
		return domain.TransactionDirectionOutbound
	default:
		return domain.TransactionDirectionUnknown
	}
}

func isOppositeSignedDelta(walletDelta int64, otherDelta int64) bool {
	return (walletDelta > 0 && otherDelta < 0) || (walletDelta < 0 && otherDelta > 0)
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}
