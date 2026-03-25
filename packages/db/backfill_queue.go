package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/flowintel/flowintel/packages/domain"
)

const DefaultWalletBackfillQueueName = "default"

type redisWalletBackfillQueueClient interface {
	RPush(context.Context, string, ...interface{}) *redis.IntCmd
	LPop(context.Context, string) *redis.StringCmd
}

type WalletBackfillJob struct {
	Chain       domain.Chain   `json:"chain"`
	Address     string         `json:"address"`
	Source      string         `json:"source"`
	RequestedAt time.Time      `json:"requested_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type WalletBackfillQueueStore interface {
	EnqueueWalletBackfill(context.Context, WalletBackfillJob) error
	DequeueWalletBackfill(context.Context, string) (WalletBackfillJob, bool, error)
}

type RedisWalletBackfillQueueStore struct {
	Client redisWalletBackfillQueueClient
}

func NewRedisWalletBackfillQueueStore(client redisWalletBackfillQueueClient) *RedisWalletBackfillQueueStore {
	return &RedisWalletBackfillQueueStore{Client: client}
}

func (j WalletBackfillJob) Validate() error {
	if !domain.IsSupportedChain(j.Chain) {
		return fmt.Errorf("unsupported chain %q", j.Chain)
	}
	if strings.TrimSpace(j.Address) == "" {
		return fmt.Errorf("wallet address is required")
	}
	if strings.TrimSpace(j.Source) == "" {
		return fmt.Errorf("wallet backfill source is required")
	}
	if j.RequestedAt.IsZero() {
		return fmt.Errorf("requested_at is required")
	}

	return nil
}

func NormalizeWalletBackfillJob(job WalletBackfillJob) WalletBackfillJob {
	job.Chain = domain.Chain(strings.ToLower(strings.TrimSpace(string(job.Chain))))
	job.Address = strings.TrimSpace(job.Address)
	job.Source = strings.TrimSpace(job.Source)
	job.RequestedAt = job.RequestedAt.UTC()

	if job.Metadata == nil {
		job.Metadata = map[string]any{}
		return job
	}

	cloned := make(map[string]any, len(job.Metadata))
	for key, value := range job.Metadata {
		cloned[key] = value
	}
	job.Metadata = cloned

	return job
}

func (s *RedisWalletBackfillQueueStore) EnqueueWalletBackfill(ctx context.Context, job WalletBackfillJob) error {
	if s == nil || s.Client == nil {
		return fmt.Errorf("redis wallet backfill queue client is nil")
	}

	normalizedJob := NormalizeWalletBackfillJob(job)
	if err := normalizedJob.Validate(); err != nil {
		return err
	}

	raw, err := json.Marshal(normalizedJob)
	if err != nil {
		return fmt.Errorf("encode wallet backfill queue job: %w", err)
	}

	if err := s.Client.RPush(ctx, BuildWalletBackfillQueueKey(DefaultWalletBackfillQueueName), raw).Err(); err != nil {
		return fmt.Errorf("enqueue wallet backfill job: %w", err)
	}

	return nil
}

func (s *RedisWalletBackfillQueueStore) DequeueWalletBackfill(
	ctx context.Context,
	queueName string,
) (WalletBackfillJob, bool, error) {
	if s == nil || s.Client == nil {
		return WalletBackfillJob{}, false, fmt.Errorf("redis wallet backfill queue client is nil")
	}

	key := BuildWalletBackfillQueueKey(queueName)
	raw, err := s.Client.LPop(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return WalletBackfillJob{}, false, nil
		}

		return WalletBackfillJob{}, false, fmt.Errorf("dequeue wallet backfill job: %w", err)
	}

	var job WalletBackfillJob
	if err := json.Unmarshal(raw, &job); err != nil {
		return WalletBackfillJob{}, false, fmt.Errorf("decode wallet backfill job: %w", err)
	}

	job = NormalizeWalletBackfillJob(job)
	if err := job.Validate(); err != nil {
		return WalletBackfillJob{}, false, err
	}

	return job, true, nil
}

func BuildWalletBackfillQueueKey(queueName string) string {
	normalizedQueueName := strings.ToLower(strings.TrimSpace(queueName))
	if normalizedQueueName == "" {
		normalizedQueueName = DefaultWalletBackfillQueueName
	}

	return strings.Join([]string{"wallet-backfill-queue", normalizedQueueName}, ":")
}
