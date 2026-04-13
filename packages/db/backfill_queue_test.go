package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeRedisWalletBackfillQueueClient struct {
	queues map[string][][]byte
	err    error
}

func (c *fakeRedisWalletBackfillQueueClient) RPush(_ context.Context, key string, values ...interface{}) *redis.IntCmd {
	if c.err != nil {
		return redis.NewIntResult(0, c.err)
	}
	if c.queues == nil {
		c.queues = map[string][][]byte{}
	}

	for _, value := range values {
		switch typed := value.(type) {
		case []byte:
			c.queues[key] = append(c.queues[key], append([]byte(nil), typed...))
		case string:
			c.queues[key] = append(c.queues[key], []byte(typed))
		}
	}

	return redis.NewIntResult(int64(len(c.queues[key])), nil)
}

func (c *fakeRedisWalletBackfillQueueClient) LPop(_ context.Context, key string) *redis.StringCmd {
	if c.err != nil {
		return redis.NewStringResult("", c.err)
	}
	queue := c.queues[key]
	if len(queue) == 0 {
		return redis.NewStringResult("", redis.Nil)
	}

	next := string(queue[0])
	c.queues[key] = queue[1:]
	return redis.NewStringResult(next, nil)
}

func TestRedisWalletBackfillQueueStoreRoundTrip(t *testing.T) {
	t.Parallel()

	store := NewRedisWalletBackfillQueueStore(&fakeRedisWalletBackfillQueueClient{})
	job := WalletBackfillJob{
		Chain:       domain.ChainEVM,
		Address:     "0x1234567890abcdef1234567890abcdef12345678",
		Source:      "search_lookup_miss",
		RequestedAt: time.Date(2026, time.March, 20, 4, 5, 6, 0, time.UTC),
		Metadata:    map[string]any{"input_kind": "evm_address"},
	}

	if err := store.EnqueueWalletBackfill(context.Background(), job); err != nil {
		t.Fatalf("EnqueueWalletBackfill returned error: %v", err)
	}

	dequeued, ok, err := store.DequeueWalletBackfill(context.Background(), DefaultWalletBackfillQueueName)
	if err != nil {
		t.Fatalf("DequeueWalletBackfill returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected queued wallet backfill job")
	}
	if dequeued.Chain != domain.ChainEVM {
		t.Fatalf("unexpected chain %q", dequeued.Chain)
	}
	if dequeued.Address != job.Address {
		t.Fatalf("unexpected address %q", dequeued.Address)
	}
	if dequeued.Source != job.Source {
		t.Fatalf("unexpected source %q", dequeued.Source)
	}
	if dequeued.Metadata["input_kind"] != "evm_address" {
		t.Fatalf("unexpected metadata %#v", dequeued.Metadata)
	}
}

func TestRedisWalletBackfillQueueStorePrioritizesSearchJobs(t *testing.T) {
	t.Parallel()

	client := &fakeRedisWalletBackfillQueueClient{}
	store := NewRedisWalletBackfillQueueStore(client)
	defaultJob := WalletBackfillJob{
		Chain:       domain.ChainSolana,
		Address:     "So11111111111111111111111111111111111111112",
		Source:      "wallet_backfill_expansion",
		RequestedAt: time.Date(2026, time.March, 20, 4, 5, 6, 0, time.UTC),
	}
	priorityJob := WalletBackfillJob{
		Chain:       domain.ChainEVM,
		Address:     "0x1234567890abcdef1234567890abcdef12345678",
		Source:      "search_lookup_miss",
		RequestedAt: time.Date(2026, time.March, 20, 4, 5, 7, 0, time.UTC),
	}

	if err := store.EnqueueWalletBackfill(context.Background(), defaultJob); err != nil {
		t.Fatalf("enqueue default job: %v", err)
	}
	if err := store.EnqueueWalletBackfill(context.Background(), priorityJob); err != nil {
		t.Fatalf("enqueue priority job: %v", err)
	}

	first, ok, err := store.DequeueWalletBackfill(context.Background(), DefaultWalletBackfillQueueName)
	if err != nil || !ok {
		t.Fatalf("expected first dequeue, ok=%v err=%v", ok, err)
	}
	if first.Source != priorityJob.Source {
		t.Fatalf("expected priority source %q, got %q", priorityJob.Source, first.Source)
	}

	second, ok, err := store.DequeueWalletBackfill(context.Background(), DefaultWalletBackfillQueueName)
	if err != nil || !ok {
		t.Fatalf("expected second dequeue, ok=%v err=%v", ok, err)
	}
	if second.Source != defaultJob.Source {
		t.Fatalf("expected default source %q, got %q", defaultJob.Source, second.Source)
	}
}

func TestRedisWalletBackfillQueueStorePrioritizesSeedListJobs(t *testing.T) {
	t.Parallel()

	client := &fakeRedisWalletBackfillQueueClient{}
	store := NewRedisWalletBackfillQueueStore(client)
	defaultJob := WalletBackfillJob{
		Chain:       domain.ChainSolana,
		Address:     "So11111111111111111111111111111111111111112",
		Source:      "wallet_backfill_expansion",
		RequestedAt: time.Date(2026, time.March, 20, 4, 5, 6, 0, time.UTC),
	}
	priorityJob := WalletBackfillJob{
		Chain:       domain.ChainEVM,
		Address:     "0x1234567890abcdef1234567890abcdef12345678",
		Source:      "curated_wallet_seed",
		RequestedAt: time.Date(2026, time.March, 20, 4, 5, 7, 0, time.UTC),
		Metadata: map[string]any{
			"source_type": WalletTrackingSourceTypeSeedList,
		},
	}

	if err := store.EnqueueWalletBackfill(context.Background(), defaultJob); err != nil {
		t.Fatalf("enqueue default job: %v", err)
	}
	if err := store.EnqueueWalletBackfill(context.Background(), priorityJob); err != nil {
		t.Fatalf("enqueue priority job: %v", err)
	}

	first, ok, err := store.DequeueWalletBackfill(context.Background(), DefaultWalletBackfillQueueName)
	if err != nil || !ok {
		t.Fatalf("expected first dequeue, ok=%v err=%v", ok, err)
	}
	if first.Source != priorityJob.Source {
		t.Fatalf("expected priority source %q, got %q", priorityJob.Source, first.Source)
	}
}

func TestRedisWalletBackfillQueueStoreReturnsEmptyOnMissingQueueItem(t *testing.T) {
	t.Parallel()

	store := NewRedisWalletBackfillQueueStore(&fakeRedisWalletBackfillQueueClient{})
	_, ok, err := store.DequeueWalletBackfill(context.Background(), DefaultWalletBackfillQueueName)
	if err != nil {
		t.Fatalf("DequeueWalletBackfill returned error: %v", err)
	}
	if ok {
		t.Fatal("expected empty queue pop to report missing item")
	}
}

func TestRedisWalletBackfillQueueStoreRejectsInvalidJob(t *testing.T) {
	t.Parallel()

	store := NewRedisWalletBackfillQueueStore(&fakeRedisWalletBackfillQueueClient{})
	err := store.EnqueueWalletBackfill(context.Background(), WalletBackfillJob{})
	if err == nil {
		t.Fatal("expected invalid job to fail validation")
	}
}

func TestRedisWalletBackfillQueueStorePropagatesRedisErrors(t *testing.T) {
	t.Parallel()

	expected := errors.New("redis unavailable")
	store := NewRedisWalletBackfillQueueStore(&fakeRedisWalletBackfillQueueClient{err: expected})
	err := store.EnqueueWalletBackfill(context.Background(), WalletBackfillJob{
		Chain:       domain.ChainSolana,
		Address:     "So11111111111111111111111111111111111111112",
		Source:      "search_lookup_miss",
		RequestedAt: time.Date(2026, time.March, 20, 4, 5, 6, 0, time.UTC),
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected redis error, got %v", err)
	}
}

func TestBuildWalletBackfillQueueKeyDefaultsWhenQueueNameIsBlank(t *testing.T) {
	t.Parallel()

	if got := BuildWalletBackfillQueueKey(""); got != "wallet-backfill-queue:default" {
		t.Fatalf("unexpected queue key %q", got)
	}
}
