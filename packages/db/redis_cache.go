package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisKVClient interface {
	Get(context.Context, string) *redis.StringCmd
	Set(context.Context, string, any, time.Duration) *redis.StatusCmd
	Del(context.Context, ...string) *redis.IntCmd
}

type RedisWalletSummaryCache struct {
	Client redisKVClient
}

func NewRedisWalletSummaryCache(client redisKVClient) *RedisWalletSummaryCache {
	return &RedisWalletSummaryCache{Client: client}
}

func (c *RedisWalletSummaryCache) GetWalletSummaryInputs(
	ctx context.Context,
	key string,
) (WalletSummaryInputs, bool, error) {
	if c == nil || c.Client == nil {
		return WalletSummaryInputs{}, false, fmt.Errorf("redis cache client is nil")
	}

	raw, err := c.Client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return WalletSummaryInputs{}, false, nil
		}

		return WalletSummaryInputs{}, false, fmt.Errorf("read wallet summary cache: %w", err)
	}

	var inputs WalletSummaryInputs
	if err := json.Unmarshal(raw, &inputs); err != nil {
		return WalletSummaryInputs{}, false, fmt.Errorf("decode wallet summary cache: %w", err)
	}

	return inputs, true, nil
}

func (c *RedisWalletSummaryCache) SetWalletSummaryInputs(
	ctx context.Context,
	key string,
	inputs WalletSummaryInputs,
	ttl time.Duration,
) error {
	if c == nil || c.Client == nil {
		return fmt.Errorf("redis cache client is nil")
	}

	raw, err := json.Marshal(inputs)
	if err != nil {
		return fmt.Errorf("encode wallet summary cache: %w", err)
	}

	if err := c.Client.Set(ctx, key, raw, ttl).Err(); err != nil {
		return fmt.Errorf("store wallet summary cache: %w", err)
	}

	return nil
}

func (c *RedisWalletSummaryCache) DeleteWalletSummaryInputs(
	ctx context.Context,
	key string,
) error {
	if c == nil || c.Client == nil {
		return fmt.Errorf("redis cache client is nil")
	}

	if err := c.Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete wallet summary cache: %w", err)
	}

	return nil
}
