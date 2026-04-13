package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

type fakeFirstConnectionFeedLoader struct {
	page   domain.FirstConnectionFeedPage
	err    error
	called int
}

func (l *fakeFirstConnectionFeedLoader) LoadFirstConnectionFeed(
	_ context.Context,
	_ FirstConnectionFeedQuery,
) (domain.FirstConnectionFeedPage, error) {
	l.called++
	if l.err != nil {
		return domain.FirstConnectionFeedPage{}, l.err
	}

	return l.page, nil
}

func TestRedisFirstConnectionFeedCacheRoundTrip(t *testing.T) {
	t.Parallel()

	client := &fakeRedisClient{}
	cache := NewRedisFirstConnectionFeedCache(client)

	page := domain.FirstConnectionFeedPage{
		Items: []domain.FirstConnectionFeedItem{
			{
				WalletID:       "wallet_1",
				Chain:          domain.ChainEVM,
				Address:        "0x123",
				Label:          "Seed Whale",
				WalletRoute:    "/wallets/evm/0x123",
				Recommendation: "fresh connection",
				ObservedAt:     "2026-03-20T09:10:11Z",
			},
		},
	}

	if err := cache.SetFirstConnectionFeedPage(context.Background(), "first-connection-feed:latest:limit:20", page, time.Minute); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if len(client.set) == 0 {
		t.Fatal("expected cache to write serialized data")
	}

	client.value = client.set

	loaded, ok, err := cache.GetFirstConnectionFeedPage(context.Background(), "first-connection-feed:latest:limit:20")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit after round trip")
	}
	if len(loaded.Items) != 1 || loaded.Items[0].WalletID != "wallet_1" {
		t.Fatalf("unexpected loaded page %#v", loaded)
	}
}

func TestCachedFirstConnectionFeedReaderUsesCacheForLatestPage(t *testing.T) {
	t.Parallel()

	client := &fakeRedisClient{}
	cache := NewRedisFirstConnectionFeedCache(client)
	loader := &fakeFirstConnectionFeedLoader{
		page: domain.FirstConnectionFeedPage{
			Items: []domain.FirstConnectionFeedItem{
				{WalletID: "wallet_1", Chain: domain.ChainEVM, Address: "0x123"},
			},
		},
	}

	reader := NewCachedFirstConnectionFeedReader(loader, cache, time.Minute)

	first, err := reader.LoadFirstConnectionFeed(context.Background(), FirstConnectionFeedQuery{Limit: 20, Sort: FirstConnectionFeedSortLatest})
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}
	if loader.called != 1 {
		t.Fatalf("expected loader to be called once, got %d", loader.called)
	}
	if len(first.Items) != 1 {
		t.Fatalf("unexpected first page %#v", first)
	}
	if len(client.set) == 0 {
		t.Fatal("expected first page to be cached")
	}

	client.value = client.set

	second, err := reader.LoadFirstConnectionFeed(context.Background(), FirstConnectionFeedQuery{Limit: 20, Sort: FirstConnectionFeedSortLatest})
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}
	if loader.called != 1 {
		t.Fatalf("expected cached load to bypass loader, got %d calls", loader.called)
	}
	if len(second.Items) != 1 || second.Items[0].WalletID != "wallet_1" {
		t.Fatalf("unexpected cached page %#v", second)
	}
}

func TestCachedFirstConnectionFeedReaderUsesCacheForScorePage(t *testing.T) {
	t.Parallel()

	client := &fakeRedisClient{}
	cache := NewRedisFirstConnectionFeedCache(client)
	loader := &fakeFirstConnectionFeedLoader{
		page: domain.FirstConnectionFeedPage{
			Items: []domain.FirstConnectionFeedItem{
				{WalletID: "wallet_score_1", Chain: domain.ChainEVM, Address: "0x456"},
			},
		},
	}

	reader := NewCachedFirstConnectionFeedReader(loader, cache, time.Minute)

	first, err := reader.LoadFirstConnectionFeed(context.Background(), FirstConnectionFeedQuery{
		Limit: 20,
		Sort:  FirstConnectionFeedSortScore,
	})
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}
	if loader.called != 1 {
		t.Fatalf("expected loader to be called once, got %d", loader.called)
	}
	if len(first.Items) != 1 {
		t.Fatalf("unexpected first page %#v", first)
	}

	client.value = client.set

	second, err := reader.LoadFirstConnectionFeed(context.Background(), FirstConnectionFeedQuery{
		Limit: 20,
		Sort:  FirstConnectionFeedSortScore,
	})
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}
	if loader.called != 1 {
		t.Fatalf("expected score cache to bypass loader, got %d calls", loader.called)
	}
	if len(second.Items) != 1 || second.Items[0].WalletID != "wallet_score_1" {
		t.Fatalf("unexpected score cached page %#v", second)
	}
}

func TestCachedFirstConnectionFeedReaderBypassesCacheForCursorQueries(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
	loader := &fakeFirstConnectionFeedLoader{
		page: domain.FirstConnectionFeedPage{
			Items: []domain.FirstConnectionFeedItem{
				{WalletID: "wallet_2", Chain: domain.ChainSolana, Address: "So111"},
			},
		},
	}

	reader := NewCachedFirstConnectionFeedReader(loader, NewRedisFirstConnectionFeedCache(&fakeRedisClient{}), time.Minute)

	page, err := reader.LoadFirstConnectionFeed(context.Background(), FirstConnectionFeedQuery{
		Limit:            20,
		Sort:             FirstConnectionFeedSortLatest,
		CursorObservedAt: &observedAt,
		CursorWalletID:   "wallet_1",
	})
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loader.called != 1 {
		t.Fatalf("expected cursor query to bypass cache and call loader once, got %d", loader.called)
	}
	if len(page.Items) != 1 || page.Items[0].WalletID != "wallet_2" {
		t.Fatalf("unexpected page %#v", page)
	}
}

func TestCachedFirstConnectionFeedReaderPropagatesLoaderError(t *testing.T) {
	t.Parallel()

	reader := NewCachedFirstConnectionFeedReader(&fakeFirstConnectionFeedLoader{
		err: errors.New("backend unavailable"),
	}, NewRedisFirstConnectionFeedCache(&fakeRedisClient{}), time.Minute)

	if _, err := reader.LoadFirstConnectionFeed(context.Background(), FirstConnectionFeedQuery{Limit: 20, Sort: FirstConnectionFeedSortLatest}); err == nil {
		t.Fatal("expected loader error to propagate")
	}
}

func TestBuildFirstConnectionFeedCacheKeyForQueryIncludesSort(t *testing.T) {
	t.Parallel()

	latestKey := BuildFirstConnectionFeedCacheKeyForQuery(FirstConnectionFeedQuery{Limit: 20, Sort: FirstConnectionFeedSortLatest})
	if latestKey != "first-connection-feed:latest:limit:20" {
		t.Fatalf("unexpected latest cache key %q", latestKey)
	}

	scoreKey := BuildFirstConnectionFeedCacheKeyForQuery(FirstConnectionFeedQuery{Limit: 20, Sort: FirstConnectionFeedSortScore})
	if scoreKey != "first-connection-feed:score:limit:20" {
		t.Fatalf("unexpected score cache key %q", scoreKey)
	}
}
