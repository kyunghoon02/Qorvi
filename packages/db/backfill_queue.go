package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
	"github.com/redis/go-redis/v9"
)

const DefaultWalletBackfillQueueName = "default"
const PriorityWalletBackfillQueueName = "priority"

type redisWalletBackfillQueueClient interface {
	RPush(context.Context, string, ...interface{}) *redis.IntCmd
	LPop(context.Context, string) *redis.StringCmd
	LLen(context.Context, string) *redis.IntCmd
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

	queueName := walletBackfillQueueNameForJob(normalizedJob)
	if err := s.Client.RPush(ctx, BuildWalletBackfillQueueKey(queueName), raw).Err(); err != nil {
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

	queueNames := walletBackfillQueueReadOrder(queueName)
	now := time.Now().UTC()

	for _, candidateQueueName := range queueNames {
		key := BuildWalletBackfillQueueKey(candidateQueueName)
		depth, err := s.Client.LLen(ctx, key).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return WalletBackfillJob{}, false, fmt.Errorf("read wallet backfill queue depth: %w", err)
		}
		for i := int64(0); i < depth; i++ {
			raw, popErr := s.Client.LPop(ctx, key).Bytes()
			if popErr != nil {
				if errors.Is(popErr, redis.Nil) {
					break
				}
				return WalletBackfillJob{}, false, fmt.Errorf("dequeue wallet backfill job: %w", popErr)
			}

			var job WalletBackfillJob
			if err := json.Unmarshal(raw, &job); err != nil {
				return WalletBackfillJob{}, false, fmt.Errorf("decode wallet backfill job: %w", err)
			}

			job = NormalizeWalletBackfillJob(job)
			if err := job.Validate(); err != nil {
				return WalletBackfillJob{}, false, err
			}
			if job.RequestedAt.After(now) {
				if pushErr := s.Client.RPush(ctx, key, raw).Err(); pushErr != nil {
					return WalletBackfillJob{}, false, fmt.Errorf("requeue delayed wallet backfill job: %w", pushErr)
				}
				continue
			}

			return job, true, nil
		}
	}

	return WalletBackfillJob{}, false, nil
}

func BuildWalletBackfillQueueKey(queueName string) string {
	normalizedQueueName := strings.ToLower(strings.TrimSpace(queueName))
	if normalizedQueueName == "" {
		normalizedQueueName = DefaultWalletBackfillQueueName
	}

	return strings.Join([]string{"wallet-backfill-queue", normalizedQueueName}, ":")
}

func walletBackfillQueueNameForJob(job WalletBackfillJob) string {
	source := strings.ToLower(strings.TrimSpace(job.Source))
	if strings.EqualFold(walletBackfillSourceType(job.Metadata), WalletTrackingSourceTypeSeedList) {
		return PriorityWalletBackfillQueueName
	}
	if strings.HasPrefix(source, "search_") {
		return PriorityWalletBackfillQueueName
	}
	return DefaultWalletBackfillQueueName
}

func walletBackfillSourceType(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	raw, ok := metadata["source_type"]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func walletBackfillQueueReadOrder(queueName string) []string {
	normalizedQueueName := strings.ToLower(strings.TrimSpace(queueName))
	if normalizedQueueName == "" || normalizedQueueName == DefaultWalletBackfillQueueName {
		return []string{PriorityWalletBackfillQueueName, DefaultWalletBackfillQueueName}
	}
	return []string{normalizedQueueName}
}
