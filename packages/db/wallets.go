package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/whalegraph/whalegraph/packages/domain"
)

const upsertWalletSQL = `
INSERT INTO wallets (
  chain,
  address,
  display_name
) VALUES ($1, $2, $3)
ON CONFLICT (chain, address) DO UPDATE SET
  updated_at = now()
RETURNING
  id,
  chain,
  address,
  display_name,
  COALESCE(entity_key, '') AS entity_key,
  created_at,
  updated_at
`

type PostgresWalletStore struct {
	Querier postgresQuerier
}

func NewPostgresWalletStore(querier postgresQuerier) *PostgresWalletStore {
	return &PostgresWalletStore{Querier: querier}
}

func NewPostgresWalletStoreFromPool(pool postgresQuerier) *PostgresWalletStore {
	return NewPostgresWalletStore(pool)
}

func (s *PostgresWalletStore) EnsureWallet(ctx context.Context, ref WalletRef) (WalletSummaryIdentity, error) {
	if s == nil || s.Querier == nil {
		return WalletSummaryIdentity{}, fmt.Errorf("wallet store is nil")
	}

	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return WalletSummaryIdentity{}, err
	}

	var identity WalletSummaryIdentity
	if err := s.Querier.QueryRow(
		ctx,
		upsertWalletSQL,
		string(normalized.Chain),
		normalized.Address,
		defaultWalletDisplayName(normalized),
	).Scan(
		&identity.WalletID,
		&identity.Chain,
		&identity.Address,
		&identity.DisplayName,
		&identity.EntityKey,
		&identity.CreatedAt,
		&identity.UpdatedAt,
	); err != nil {
		return WalletSummaryIdentity{}, fmt.Errorf("ensure wallet: %w", err)
	}

	return identity, nil
}

func defaultWalletDisplayName(ref WalletRef) string {
	return fmt.Sprintf("%s wallet %s", strings.ToUpper(string(ref.Chain)), strings.TrimSpace(ref.Address))
}

func WalletRefFromTransaction(tx domain.NormalizedTransaction) WalletRef {
	return WalletRef{
		Chain:   tx.Wallet.Chain,
		Address: tx.Wallet.Address,
	}
}
