# Wallet Tracking State Migration Outline

This document describes how the AI dataset state model should map into the main
FlowIntel product schema.

It is not an executable migration yet. It is the bridge from AI workspace
design to product implementation.

## Goal

Introduce tracked-wallet lifecycle metadata without breaking the current
search, backfill, graph, and summary paths.

## Migration strategy

### Principle

- keep `wallets` as the canonical wallet dimension
- add lifecycle and discovery metadata in adjacent tables
- reuse the existing queue implementation where possible

## Proposed schema changes

### 1. New table: `wallet_tracking_state`

Suggested columns:

- `wallet_id UUID PRIMARY KEY REFERENCES wallets(wallet_id)`
- `status TEXT NOT NULL`
- `source_type TEXT NOT NULL`
- `source_ref TEXT NOT NULL DEFAULT ''`
- `tracking_priority INTEGER NOT NULL DEFAULT 0`
- `candidate_score NUMERIC NOT NULL DEFAULT 0`
- `label_confidence NUMERIC NOT NULL DEFAULT 0`
- `entity_confidence NUMERIC NOT NULL DEFAULT 0`
- `smart_money_confidence NUMERIC NOT NULL DEFAULT 0`
- `first_discovered_at TIMESTAMPTZ NOT NULL`
- `last_activity_at TIMESTAMPTZ`
- `last_backfill_at TIMESTAMPTZ`
- `last_realtime_at TIMESTAMPTZ`
- `stale_after_at TIMESTAMPTZ`
- `notes JSONB NOT NULL DEFAULT '{}'::jsonb`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`

Indexes:

- `(status, tracking_priority DESC, updated_at DESC)`
- `(source_type, updated_at DESC)`
- `(stale_after_at)`

### 2. New table: `wallet_candidate_source_events`

Suggested columns:

- `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- `wallet_id UUID NOT NULL REFERENCES wallets(wallet_id)`
- `source_type TEXT NOT NULL`
- `source_ref TEXT NOT NULL DEFAULT ''`
- `discovery_reason TEXT NOT NULL`
- `confidence NUMERIC NOT NULL DEFAULT 0`
- `payload JSONB NOT NULL DEFAULT '{}'::jsonb`
- `observed_at TIMESTAMPTZ NOT NULL`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`

Indexes:

- `(wallet_id, observed_at DESC)`
- `(source_type, observed_at DESC)`

### 3. New table: `wallet_tracking_subscriptions`

Suggested columns:

- `wallet_id UUID NOT NULL REFERENCES wallets(wallet_id)`
- `provider TEXT NOT NULL`
- `subscription_key TEXT NOT NULL`
- `status TEXT NOT NULL`
- `last_synced_at TIMESTAMPTZ`
- `last_event_at TIMESTAMPTZ`
- `metadata JSONB NOT NULL DEFAULT '{}'::jsonb`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`

Primary key suggestion:

- `(wallet_id, provider, subscription_key)`

Indexes:

- `(provider, status, updated_at DESC)`
- `(last_event_at)`

## Queue integration

Do not add a second queue system first.

Instead, extend the current backfill queue payload metadata to include:

- `reason`
- `priority`
- `source_type`
- `source_ref`
- `candidate_score`
- `tracking_status_target`

Only create a dedicated `wallet_tracking_queue` table later if Redis queue
metadata becomes insufficient for observability or auditability.

## Backfill migration path

### Phase 1

- backfill search-discovered wallets into `wallet_tracking_state`
- mark as `tracked` once initial backfill succeeds
- populate `last_backfill_at`

### Phase 2

- register realtime subscription rows for tracked wallets
- populate `last_realtime_at`
- mark stale wallets with scheduled refresh windows

### Phase 3

- add candidate collectors
- Dune collector
- hop expansion collector
- watchlist collector

## Serving integration

The following product paths should continue using existing tables:

- wallet summary
- wallet graph
- signal events
- enrichment snapshots

The new tracking tables should first support:

- freshness policy
- candidate prioritization
- tracked wallet observability
- AI-ready context assembly

## Rollout criteria

Ready for product implementation when:

1. status lifecycle is agreed
2. backfill queue metadata mapping is agreed
3. provider subscription ownership model is agreed
4. stale refresh policy is agreed
