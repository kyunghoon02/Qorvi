ALTER TABLE transactions
  ADD COLUMN IF NOT EXISTS amount text,
  ADD COLUMN IF NOT EXISTS amount_numeric numeric,
  ADD COLUMN IF NOT EXISTS token_chain text,
  ADD COLUMN IF NOT EXISTS token_address text,
  ADD COLUMN IF NOT EXISTS token_symbol text,
  ADD COLUMN IF NOT EXISTS token_decimals integer;

CREATE INDEX IF NOT EXISTS idx_transactions_wallet_counterparty_observed_at
  ON transactions (wallet_id, counterparty_address, observed_at DESC);
