package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisSetNXClient interface {
	SetNX(context.Context, string, any, time.Duration) *redis.BoolCmd
	Del(context.Context, ...string) *redis.IntCmd
}

type IngestDedupStore interface {
	Claim(context.Context, string, time.Duration) (bool, error)
	Release(context.Context, string) error
}

type RedisIngestDedupStore struct {
	Client redisSetNXClient
}

func NewRedisIngestDedupStore(client redisSetNXClient) *RedisIngestDedupStore {
	return &RedisIngestDedupStore{Client: client}
}

func (s *RedisIngestDedupStore) Claim(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if s == nil || s.Client == nil {
		return false, fmt.Errorf("redis ingest dedup client is nil")
	}

	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" {
		return false, fmt.Errorf("dedup key is required")
	}
	if ttl <= 0 {
		return false, fmt.Errorf("dedup ttl must be positive")
	}

	claimed, err := s.Client.SetNX(ctx, normalizedKey, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("claim ingest dedup key: %w", err)
	}

	return claimed, nil
}

func (s *RedisIngestDedupStore) Release(ctx context.Context, key string) error {
	if s == nil || s.Client == nil {
		return fmt.Errorf("redis ingest dedup client is nil")
	}

	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" {
		return fmt.Errorf("dedup key is required")
	}

	if err := s.Client.Del(ctx, normalizedKey).Err(); err != nil {
		return fmt.Errorf("release ingest dedup key: %w", err)
	}

	return nil
}

func BuildIngestDedupKey(namespace string, parts ...string) string {
	normalizedNamespace := strings.ToLower(strings.TrimSpace(namespace))
	normalizedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		normalizedParts = append(normalizedParts, part)
	}

	segments := []string{"ingest-dedup"}
	if normalizedNamespace != "" {
		segments = append(segments, normalizedNamespace)
	}
	segments = append(segments, normalizedParts...)

	return strings.Join(segments, ":")
}
