package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type fakeRedisDedupClient struct {
	claims map[string]bool
	err    error
}

func (c *fakeRedisDedupClient) SetNX(_ context.Context, key string, _ any, _ time.Duration) *redis.BoolCmd {
	if c.err != nil {
		return redis.NewBoolResult(false, c.err)
	}
	if c.claims == nil {
		c.claims = map[string]bool{}
	}
	if c.claims[key] {
		return redis.NewBoolResult(false, nil)
	}
	c.claims[key] = true
	return redis.NewBoolResult(true, nil)
}

func (c *fakeRedisDedupClient) Del(_ context.Context, keys ...string) *redis.IntCmd {
	if c.err != nil {
		return redis.NewIntResult(0, c.err)
	}
	deleted := int64(0)
	for _, key := range keys {
		if c.claims != nil && c.claims[key] {
			delete(c.claims, key)
			deleted++
		}
	}
	return redis.NewIntResult(deleted, nil)
}

func TestRedisIngestDedupStoreClaim(t *testing.T) {
	t.Parallel()

	store := NewRedisIngestDedupStore(&fakeRedisDedupClient{})
	key := BuildIngestDedupKey("normalized-transaction", "evm:0xabc:wallet")

	claimed, err := store.Claim(context.Background(), key, time.Hour)
	if err != nil {
		t.Fatalf("expected first claim to succeed, got %v", err)
	}
	if !claimed {
		t.Fatal("expected first claim to return true")
	}

	claimed, err = store.Claim(context.Background(), key, time.Hour)
	if err != nil {
		t.Fatalf("expected duplicate claim to return false without error, got %v", err)
	}
	if claimed {
		t.Fatal("expected duplicate claim to return false")
	}
}

func TestRedisIngestDedupStoreRelease(t *testing.T) {
	t.Parallel()

	client := &fakeRedisDedupClient{}
	store := NewRedisIngestDedupStore(client)
	key := BuildIngestDedupKey("normalized-transaction", "evm:0xabc:wallet")

	claimed, err := store.Claim(context.Background(), key, time.Hour)
	if err != nil || !claimed {
		t.Fatalf("expected initial claim to succeed, got claimed=%v err=%v", claimed, err)
	}
	if err := store.Release(context.Background(), key); err != nil {
		t.Fatalf("expected release to succeed, got %v", err)
	}
	claimed, err = store.Claim(context.Background(), key, time.Hour)
	if err != nil {
		t.Fatalf("expected re-claim after release to succeed, got %v", err)
	}
	if !claimed {
		t.Fatal("expected key to be claimable again after release")
	}
}

func TestRedisIngestDedupStoreClaimRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	store := NewRedisIngestDedupStore(&fakeRedisDedupClient{})

	if _, err := store.Claim(context.Background(), " ", time.Hour); err == nil {
		t.Fatal("expected empty key error")
	}
	if _, err := store.Claim(context.Background(), "dedup:key", 0); err == nil {
		t.Fatal("expected ttl error")
	}
	if err := store.Release(context.Background(), " "); err == nil {
		t.Fatal("expected empty key release error")
	}
}

func TestRedisIngestDedupStoreClaimPropagatesClientError(t *testing.T) {
	t.Parallel()

	store := NewRedisIngestDedupStore(&fakeRedisDedupClient{err: errors.New("redis unavailable")})

	_, err := store.Claim(context.Background(), "dedup:key", time.Hour)
	if err == nil {
		t.Fatal("expected client error")
	}
}

func TestBuildIngestDedupKey(t *testing.T) {
	t.Parallel()

	key := BuildIngestDedupKey(" Normalized-Transaction ", " evm:0xabc ", "", " wallet:seed ")
	if key != "ingest-dedup:normalized-transaction:evm:0xabc:wallet:seed" {
		t.Fatalf("unexpected dedup key %q", key)
	}
}
