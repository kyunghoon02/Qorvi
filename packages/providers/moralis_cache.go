package providers

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

type moralisKVClient interface {
	Get(context.Context, string) *redis.StringCmd
	Set(context.Context, string, any, time.Duration) *redis.StatusCmd
}

type MoralisWalletEnrichmentCache interface {
	GetWalletEnrichment(context.Context, string) (domain.WalletEnrichment, bool, error)
	SetWalletEnrichment(context.Context, string, domain.WalletEnrichment, time.Duration) error
}

type RedisMoralisWalletEnrichmentCache struct {
	Client moralisKVClient
}

func NewRedisMoralisWalletEnrichmentCache(client moralisKVClient) *RedisMoralisWalletEnrichmentCache {
	return &RedisMoralisWalletEnrichmentCache{Client: client}
}

func BuildMoralisWalletEnrichmentCacheKey(chain domain.Chain, address string) string {
	return fmt.Sprintf(
		"moralis-wallet-enrichment:%s:%s",
		strings.ToLower(strings.TrimSpace(string(chain))),
		strings.ToLower(strings.TrimSpace(address)),
	)
}

func (c *RedisMoralisWalletEnrichmentCache) GetWalletEnrichment(
	ctx context.Context,
	key string,
) (domain.WalletEnrichment, bool, error) {
	if c == nil || c.Client == nil {
		return domain.WalletEnrichment{}, false, fmt.Errorf("redis cache client is nil")
	}

	raw, err := c.Client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return domain.WalletEnrichment{}, false, nil
		}
		return domain.WalletEnrichment{}, false, fmt.Errorf("read moralis enrichment cache: %w", err)
	}

	var enrichment domain.WalletEnrichment
	if err := json.Unmarshal(raw, &enrichment); err != nil {
		return domain.WalletEnrichment{}, false, fmt.Errorf("decode moralis enrichment cache: %w", err)
	}

	return enrichment, true, nil
}

func (c *RedisMoralisWalletEnrichmentCache) SetWalletEnrichment(
	ctx context.Context,
	key string,
	enrichment domain.WalletEnrichment,
	ttl time.Duration,
) error {
	if c == nil || c.Client == nil {
		return fmt.Errorf("redis cache client is nil")
	}

	raw, err := json.Marshal(enrichment)
	if err != nil {
		return fmt.Errorf("encode moralis enrichment cache: %w", err)
	}

	if err := c.Client.Set(ctx, key, raw, ttl).Err(); err != nil {
		return fmt.Errorf("store moralis enrichment cache: %w", err)
	}

	return nil
}
