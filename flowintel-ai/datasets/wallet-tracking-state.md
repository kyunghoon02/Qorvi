# Wallet Tracking State

This spec defines the state model for moving FlowIntel from user-triggered lazy
indexing to candidate discovery, tracked coverage, and AI-ready serving.

## Goal

Separate:

- wallet identity
- wallet discovery source
- tracking freshness
- labeling progress
- scoring progress

Do not replace the existing `wallets` table in the main product stack. Treat
`wallets` as the canonical wallet dimension and add state tables around it.

## Recommended entities

### 1. `wallet_tracking_state`

One row per canonical wallet in the product.

Suggested fields:

- `wallet_id`
- `chain`
- `address`
- `status`
  - `candidate`
  - `tracked`
  - `labeled`
  - `scored`
  - `stale`
  - `suppressed`
- `first_discovered_at`
- `last_activity_at`
- `last_backfill_at`
- `last_realtime_at`
- `stale_after_at`
- `source_type`
  - `seed_list`
  - `dune_candidate`
  - `user_search`
  - `hop_expansion`
  - `watchlist`
- `source_ref`
- `tracking_priority`
- `candidate_score`
- `label_confidence`
- `entity_confidence`
- `smart_money_confidence`
- `notes_json`
- `created_at`
- `updated_at`

Notes:

- `wallet_id` should reference the existing canonical wallet row
- `status` should be derived from deterministic pipeline progress, not manual UI state
- `candidate_score` should drive queue priority, not final user-facing scoring

### 2. `wallet_candidate_source_events`

Append-only evidence for how a wallet entered the system.

Suggested fields:

- `id`
- `wallet_id`
- `chain`
- `address`
- `source_type`
- `source_ref`
- `discovery_reason`
- `payload_json`
- `confidence`
- `observed_at`
- `created_at`

Examples:

- Dune query result row
- repeated counterparty expansion
- watchlist import
- high-frequency user search

### 3. `wallet_tracking_queue`

Queue state for work scheduling and retries.

Suggested fields:

- `id`
- `wallet_id`
- `chain`
- `address`
- `reason`
  - `new_candidate`
  - `user_search`
  - `stale_refresh`
  - `hop_expansion`
  - `watchlist_bootstrap`
  - `signal_backfill`
- `priority`
- `attempts`
- `max_attempts`
- `scheduled_at`
- `started_at`
- `completed_at`
- `last_error`
- `metadata_json`

Implementation note:

- Reuse the existing backfill queue where possible and extend its metadata model
- This spec is for the target state and scheduling semantics, not a requirement
  to create a second independent queue implementation

### 4. `wallet_tracking_subscriptions`

Source of truth for realtime webhook tracking.

Suggested fields:

- `wallet_id`
- `chain`
- `address`
- `provider`
  - `alchemy`
  - `helius`
- `subscription_key`
- `status`
  - `pending`
  - `active`
  - `errored`
  - `paused`
- `last_synced_at`
- `last_event_at`
- `metadata_json`
- `created_at`
- `updated_at`

## State transitions

### Candidate lifecycle

1. `candidate`
   - discovered but not yet backfilled
2. `tracked`
   - initial backfill succeeded and realtime subscription is active or pending
3. `labeled`
   - deterministic entity or behavior evidence exists
4. `scored`
   - intelligence scores are computed and serving rows are fresh
5. `stale`
   - historical or realtime freshness is outside policy

### Example transitions

- `user_search` -> queue -> initial backfill -> `tracked`
- `dune_candidate` -> candidate score threshold -> queue -> `tracked`
- tracked + heuristic entity assignment -> `labeled`
- labeled + signal refresh complete -> `scored`
- scored + no realtime updates beyond policy -> `stale`

## Freshness policy

Suggested defaults:

- recently active tracked wallets: incremental update every 5m to 60m
- quiet tracked wallets: daily refresh
- stale candidate without conversion: drop after 7d
- curated funds, MMs, treasuries: never drop automatically

## Existing product table relationship

Main stack remains authoritative for:

- `wallets`
- `transactions`
- `wallet_daily_stats`
- `wallet_graph_snapshots`
- `wallet_enrichment_snapshots`
- `signal_events`

AI workspace state should reference those tables, not duplicate them.
