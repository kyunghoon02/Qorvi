package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type postgresTransactionExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type NormalizedTransactionWrite struct {
	WalletID    string
	Transaction domain.NormalizedTransaction
}

type NormalizedTransactionStore interface {
	UpsertNormalizedTransaction(context.Context, NormalizedTransactionWrite) error
	UpsertNormalizedTransactions(context.Context, []NormalizedTransactionWrite) error
}

type PostgresNormalizedTransactionStore struct {
	Execer postgresTransactionExecer
	Now    func() time.Time
}

const upsertNormalizedTransactionSQL = `
INSERT INTO transactions (
  chain,
  tx_hash,
  wallet_id,
  direction,
  counterparty_chain,
  counterparty_address,
  raw_payload_path,
  schema_version,
  observed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (chain, tx_hash, wallet_id) DO UPDATE SET
  direction = excluded.direction,
  counterparty_chain = excluded.counterparty_chain,
  counterparty_address = excluded.counterparty_address,
  wallet_id = excluded.wallet_id,
  raw_payload_path = excluded.raw_payload_path,
  schema_version = excluded.schema_version,
  observed_at = excluded.observed_at
`

func NewPostgresNormalizedTransactionStore(execer postgresTransactionExecer) *PostgresNormalizedTransactionStore {
	return &PostgresNormalizedTransactionStore{
		Execer: execer,
		Now:    time.Now,
	}
}

func NewPostgresNormalizedTransactionStoreFromPool(pool postgresTransactionExecer) *PostgresNormalizedTransactionStore {
	return NewPostgresNormalizedTransactionStore(pool)
}

func (s *PostgresNormalizedTransactionStore) UpsertNormalizedTransaction(
	ctx context.Context,
	write NormalizedTransactionWrite,
) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("normalized transaction store is nil")
	}

	record := domain.NormalizeNormalizedTransaction(write.Transaction)
	if err := domain.ValidateNormalizedTransaction(record); err != nil {
		return fmt.Errorf("validate normalized transaction: %w", err)
	}

	walletID := strings.TrimSpace(write.WalletID)
	if walletID == "" {
		return fmt.Errorf("wallet id is required")
	}

	if record.ObservedAt.IsZero() {
		record.ObservedAt = s.now().UTC()
	}

	if _, err := s.Execer.Exec(
		ctx,
		upsertNormalizedTransactionSQL,
		string(record.Chain),
		record.TxHash,
		walletID,
		string(record.Direction),
		counterpartyChain(record),
		counterpartyAddress(record),
		record.RawPayloadPath,
		record.SchemaVersion,
		record.ObservedAt.UTC(),
	); err != nil {
		return fmt.Errorf("upsert normalized transaction: %w", err)
	}

	return nil
}

func counterpartyChain(record domain.NormalizedTransaction) string {
	if record.Counterparty == nil {
		return ""
	}

	return string(record.Counterparty.Chain)
}

func counterpartyAddress(record domain.NormalizedTransaction) string {
	if record.Counterparty == nil {
		return ""
	}

	return record.Counterparty.Address
}

func (s *PostgresNormalizedTransactionStore) UpsertNormalizedTransactions(
	ctx context.Context,
	writes []NormalizedTransactionWrite,
) error {
	for _, write := range writes {
		if err := s.UpsertNormalizedTransaction(ctx, write); err != nil {
			return err
		}
	}

	return nil
}

func (s *PostgresNormalizedTransactionStore) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}

	return time.Now()
}
