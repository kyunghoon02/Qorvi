package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/qorvi/qorvi/packages/domain"
)

type WalletGraphCache interface {
	GetWalletGraph(context.Context, string) (domain.WalletGraph, bool, error)
	SetWalletGraph(context.Context, string, domain.WalletGraph, time.Duration) error
	DeleteWalletGraph(context.Context, string) error
}

type RedisWalletGraphCache struct {
	Client redisKVClient
}

func NewRedisWalletGraphCache(client redisKVClient) *RedisWalletGraphCache {
	return &RedisWalletGraphCache{Client: client}
}

func (c *RedisWalletGraphCache) GetWalletGraph(
	ctx context.Context,
	key string,
) (domain.WalletGraph, bool, error) {
	if c == nil || c.Client == nil {
		return domain.WalletGraph{}, false, fmt.Errorf("redis cache client is nil")
	}

	raw, err := c.Client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return domain.WalletGraph{}, false, nil
		}

		return domain.WalletGraph{}, false, fmt.Errorf("read wallet graph cache: %w", err)
	}

	var graph domain.WalletGraph
	if err := json.Unmarshal(raw, &graph); err != nil {
		return domain.WalletGraph{}, false, fmt.Errorf("decode wallet graph cache: %w", err)
	}

	return graph, true, nil
}

func (c *RedisWalletGraphCache) SetWalletGraph(
	ctx context.Context,
	key string,
	graph domain.WalletGraph,
	ttl time.Duration,
) error {
	if c == nil || c.Client == nil {
		return fmt.Errorf("redis cache client is nil")
	}

	raw, err := json.Marshal(graph)
	if err != nil {
		return fmt.Errorf("encode wallet graph cache: %w", err)
	}

	if err := c.Client.Set(ctx, key, raw, ttl).Err(); err != nil {
		return fmt.Errorf("store wallet graph cache: %w", err)
	}

	return nil
}

func (c *RedisWalletGraphCache) DeleteWalletGraph(
	ctx context.Context,
	key string,
) error {
	if c == nil || c.Client == nil {
		return fmt.Errorf("redis cache client is nil")
	}

	if err := c.Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete wallet graph cache: %w", err)
	}

	return nil
}

type CachedWalletGraphReader struct {
	Loader        WalletGraphReader
	Cache         WalletGraphCache
	SnapshotStore WalletGraphSnapshotStore
	TTL           time.Duration
	Now           func() time.Time
}

func NewCachedWalletGraphReader(
	loader WalletGraphReader,
	cache WalletGraphCache,
	snapshots WalletGraphSnapshotStore,
	ttl time.Duration,
) *CachedWalletGraphReader {
	return &CachedWalletGraphReader{
		Loader:        loader,
		Cache:         cache,
		SnapshotStore: snapshots,
		TTL:           ttl,
		Now:           time.Now,
	}
}

func BuildWalletGraphCacheKey(query WalletGraphQuery) string {
	address := strings.ToLower(strings.TrimSpace(query.Ref.Address))
	return fmt.Sprintf(
		"wallet-graph:%s:%s:depth:%d:max:%d",
		query.Ref.Chain,
		address,
		query.DepthRequested,
		query.MaxCounterparties,
	)
}

func (r *CachedWalletGraphReader) ReadWalletGraph(
	ctx context.Context,
	query WalletGraphQuery,
) (domain.WalletGraph, error) {
	if r == nil || r.Loader == nil {
		return domain.WalletGraph{}, fmt.Errorf("wallet graph cache reader is nil")
	}

	if r.Cache == nil || r.TTL <= 0 {
		return r.Loader.ReadWalletGraph(ctx, query)
	}

	key := BuildWalletGraphCacheKey(query)
	if cached, ok, err := r.Cache.GetWalletGraph(ctx, key); err != nil {
		return domain.WalletGraph{}, fmt.Errorf("load wallet graph cache: %w", err)
	} else if ok {
		return withWalletGraphSnapshot(cached, key, "graph-cache-hit", r.TTL, r.now()), nil
	}

	if r.SnapshotStore != nil && IsCanonicalWalletGraphQuery(query) {
		if snapped, ok, err := r.SnapshotStore.ReadWalletGraphSnapshot(ctx, query); err != nil {
			return domain.WalletGraph{}, fmt.Errorf("load wallet graph snapshot: %w", err)
		} else if ok {
			snapshotGraph := withWalletGraphSnapshot(snapped, key, "graph-snapshot-hit", r.TTL, r.now())
			if err := r.Cache.SetWalletGraph(ctx, key, snapshotGraph, r.TTL); err != nil {
				return domain.WalletGraph{}, fmt.Errorf("store wallet graph cache from snapshot: %w", err)
			}
			return snapshotGraph, nil
		}
	}

	graph, err := r.Loader.ReadWalletGraph(ctx, query)
	if err != nil {
		return domain.WalletGraph{}, err
	}

	if r.SnapshotStore != nil && IsCanonicalWalletGraphQuery(query) {
		if err := r.SnapshotStore.UpsertWalletGraphSnapshot(ctx, query, graph); err != nil {
			return domain.WalletGraph{}, fmt.Errorf("store wallet graph snapshot: %w", err)
		}
	}

	cacheSource := "graph-cache"
	fillSource := "graph-cache-fill"
	if r.SnapshotStore != nil && IsCanonicalWalletGraphQuery(query) {
		cacheSource = "graph-snapshot"
		fillSource = "graph-snapshot-fill"
	}
	cachedGraph := withWalletGraphSnapshot(graph, key, cacheSource, r.TTL, r.now())
	if err := r.Cache.SetWalletGraph(ctx, key, cachedGraph, r.TTL); err != nil {
		return domain.WalletGraph{}, fmt.Errorf("store wallet graph cache: %w", err)
	}

	return withWalletGraphSnapshot(cachedGraph, key, fillSource, r.TTL, r.now()), nil
}

func withWalletGraphSnapshot(
	graph domain.WalletGraph,
	key string,
	source string,
	ttl time.Duration,
	now time.Time,
) domain.WalletGraph {
	next := graph
	generatedAt := ""
	if next.Snapshot != nil {
		generatedAt = next.Snapshot.GeneratedAt
	}
	if generatedAt == "" {
		generatedAt = now.UTC().Format(time.RFC3339)
	}
	next.Snapshot = &domain.WalletGraphSnapshot{
		Key:           key,
		Source:        source,
		GeneratedAt:   generatedAt,
		MaxAgeSeconds: int(ttl.Seconds()),
	}
	return next
}

func (r *CachedWalletGraphReader) now() time.Time {
	if r != nil && r.Now != nil {
		return r.Now()
	}

	return time.Now()
}

func InvalidateWalletGraphSnapshot(
	ctx context.Context,
	cache WalletGraphCache,
	snapshots WalletGraphSnapshotStore,
	ref WalletRef,
) error {
	query, err := BuildWalletGraphSnapshotQuery(ref)
	if err != nil {
		return err
	}

	if cache != nil {
		if err := cache.DeleteWalletGraph(ctx, BuildWalletGraphCacheKey(query)); err != nil {
			return err
		}
	}
	if snapshots != nil {
		if err := snapshots.DeleteWalletGraphSnapshot(ctx, query); err != nil {
			return err
		}
	}

	return nil
}
