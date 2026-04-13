package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type redisQueueDepthClient interface {
	LLen(context.Context, string) *redis.IntCmd
}

type AdminQueueDepthRecord struct {
	DefaultDepth  int
	PriorityDepth int
}

type RedisAdminQueueStatsStore struct {
	Client redisQueueDepthClient
}

func NewRedisAdminQueueStatsStore(client redisQueueDepthClient) *RedisAdminQueueStatsStore {
	return &RedisAdminQueueStatsStore{Client: client}
}

func (s *RedisAdminQueueStatsStore) ReadWalletBackfillQueueDepth(
	ctx context.Context,
) (AdminQueueDepthRecord, error) {
	if s == nil || s.Client == nil {
		return AdminQueueDepthRecord{}, nil
	}

	defaultDepth, err := s.readQueueLength(ctx, BuildWalletBackfillQueueKey(DefaultWalletBackfillQueueName))
	if err != nil {
		return AdminQueueDepthRecord{}, err
	}
	priorityDepth, err := s.readQueueLength(ctx, BuildWalletBackfillQueueKey(PriorityWalletBackfillQueueName))
	if err != nil {
		return AdminQueueDepthRecord{}, err
	}

	return AdminQueueDepthRecord{
		DefaultDepth:  defaultDepth,
		PriorityDepth: priorityDepth,
	}, nil
}

func (s *RedisAdminQueueStatsStore) readQueueLength(ctx context.Context, key string) (int, error) {
	value, err := s.Client.LLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("read queue length %s: %w", key, err)
	}
	return int(value), nil
}
