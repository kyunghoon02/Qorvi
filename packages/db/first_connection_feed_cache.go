package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/qorvi/qorvi/packages/domain"
)

type FirstConnectionFeedCache interface {
	GetFirstConnectionFeedPage(
		context.Context,
		string,
	) (domain.FirstConnectionFeedPage, bool, error)
	SetFirstConnectionFeedPage(
		context.Context,
		string,
		domain.FirstConnectionFeedPage,
		time.Duration,
	) error
}

type RedisFirstConnectionFeedCache struct {
	Client redisKVClient
}

func NewRedisFirstConnectionFeedCache(client redisKVClient) *RedisFirstConnectionFeedCache {
	return &RedisFirstConnectionFeedCache{Client: client}
}

func (c *RedisFirstConnectionFeedCache) GetFirstConnectionFeedPage(
	ctx context.Context,
	key string,
) (domain.FirstConnectionFeedPage, bool, error) {
	if c == nil || c.Client == nil {
		return domain.FirstConnectionFeedPage{}, false, fmt.Errorf("redis cache client is nil")
	}

	raw, err := c.Client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return domain.FirstConnectionFeedPage{}, false, nil
		}

		return domain.FirstConnectionFeedPage{}, false, fmt.Errorf("read first connection feed cache: %w", err)
	}

	var page domain.FirstConnectionFeedPage
	if err := json.Unmarshal(raw, &page); err != nil {
		return domain.FirstConnectionFeedPage{}, false, fmt.Errorf("decode first connection feed cache: %w", err)
	}

	return page, true, nil
}

func (c *RedisFirstConnectionFeedCache) SetFirstConnectionFeedPage(
	ctx context.Context,
	key string,
	page domain.FirstConnectionFeedPage,
	ttl time.Duration,
) error {
	if c == nil || c.Client == nil {
		return fmt.Errorf("redis cache client is nil")
	}

	raw, err := json.Marshal(page)
	if err != nil {
		return fmt.Errorf("encode first connection feed cache: %w", err)
	}

	if err := c.Client.Set(ctx, key, raw, ttl).Err(); err != nil {
		return fmt.Errorf("store first connection feed cache: %w", err)
	}

	return nil
}

type CachedFirstConnectionFeedReader struct {
	Loader FirstConnectionFeedLoader
	Cache  FirstConnectionFeedCache
	TTL    time.Duration
}

func NewCachedFirstConnectionFeedReader(
	loader FirstConnectionFeedLoader,
	cache FirstConnectionFeedCache,
	ttl time.Duration,
) *CachedFirstConnectionFeedReader {
	return &CachedFirstConnectionFeedReader{
		Loader: loader,
		Cache:  cache,
		TTL:    ttl,
	}
}

func BuildFirstConnectionFeedCacheKey(limit int) string {
	return BuildFirstConnectionFeedCacheKeyForQuery(FirstConnectionFeedQuery{
		Limit: limit,
		Sort:  FirstConnectionFeedSortLatest,
	})
}

func BuildFirstConnectionFeedCacheKeyForQuery(query FirstConnectionFeedQuery) string {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	sort := query.Sort
	if sort == "" {
		sort = FirstConnectionFeedSortLatest
	}

	return fmt.Sprintf("first-connection-feed:%s:limit:%d", sort, limit)
}

func (r *CachedFirstConnectionFeedReader) LoadFirstConnectionFeed(
	ctx context.Context,
	query FirstConnectionFeedQuery,
) (domain.FirstConnectionFeedPage, error) {
	if r == nil || r.Loader == nil {
		return domain.FirstConnectionFeedPage{}, fmt.Errorf("first connection feed cache reader is nil")
	}

	if r.Cache == nil || r.TTL <= 0 || query.CursorObservedAt != nil || query.CursorScoreValue != nil || strings.TrimSpace(query.CursorWalletID) != "" {
		return r.Loader.LoadFirstConnectionFeed(ctx, query)
	}

	key := BuildFirstConnectionFeedCacheKeyForQuery(query)
	if cached, ok, err := r.Cache.GetFirstConnectionFeedPage(ctx, key); err != nil {
		return domain.FirstConnectionFeedPage{}, fmt.Errorf("load first connection feed cache: %w", err)
	} else if ok {
		return cached, nil
	}

	page, err := r.Loader.LoadFirstConnectionFeed(ctx, query)
	if err != nil {
		return domain.FirstConnectionFeedPage{}, err
	}

	if err := r.Cache.SetFirstConnectionFeedPage(ctx, key, page, r.TTL); err != nil {
		return domain.FirstConnectionFeedPage{}, fmt.Errorf("store first connection feed cache: %w", err)
	}

	return page, nil
}
