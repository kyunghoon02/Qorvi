# Signal Outcome Labels

This spec defines the outcome labels needed to train or evaluate the FlowIntel
AI analyst layer and any future predictive scoring models.

## Goal

Measure whether a detected signal led to a meaningful market or behavioral
outcome over fixed time windows.

This dataset is for:

- model training
- offline evaluation
- signal quality measurement
- explanation grounding

## Core principle

Store:

- the signal snapshot at detection time
- the observed outcomes after 1d / 3d / 7d / 30d

Do not overwrite the original signal state after later market movement happens.

## Recommended entities

### 1. `signal_event_snapshots`

Immutable snapshot of a signal at creation time.

Suggested fields:

- `signal_event_id`
- `signal_type`
  - `cluster_score`
  - `shadow_exit`
  - `first_connection`
  - `rotation_candidate`
  - `token_adoption`
- `subject_type`
  - `wallet`
  - `cluster`
  - `entity`
  - `token`
- `subject_key`
- `chain`
- `primary_token`
- `score_value`
- `score_rating`
- `confidence`
- `evidence_json`
- `related_wallet_ids`
- `related_entity_keys`
- `coverage_window_days`
- `observed_at`
- `created_at`

### 2. `signal_outcomes`

One row per signal snapshot and evaluation window.

Suggested fields:

- `signal_event_id`
- `window`
  - `1d`
  - `3d`
  - `7d`
  - `30d`
- `price_return_pct`
- `volume_change_pct`
- `holder_change_pct`
- `liquidity_change_pct`
- `net_inflow_usd`
- `follow_on_wallet_count`
- `follow_on_high_quality_wallet_count`
- `follow_on_entity_count`
- `outcome_label`
  - `positive`
  - `neutral`
  - `negative`
  - `unresolved`
- `outcome_score`
- `resolved_at`
- `metadata_json`

### 3. `token_adoption_event_snapshots`

Track first-time entries or repeated adoption by high-quality wallets.

Suggested fields:

- `id`
- `token_key`
- `chain`
- `wallet_id`
- `entity_key`
- `event_type`
  - `first_entry`
  - `repeat_entry`
  - `cohort_entry`
- `wallet_quality_score`
- `position_size_usd`
- `observed_at`
- `metadata_json`

### 4. `cluster_movement_snapshots`

Capture coordinated movement context for cluster-level signals.

Suggested fields:

- `id`
- `cluster_id`
- `signal_event_id`
- `member_count`
- `active_member_count`
- `shared_destination_count`
- `shared_funder_count`
- `shared_token_entry_count`
- `net_outflow_usd`
- `net_inflow_usd`
- `bridge_count`
- `observed_at`
- `metadata_json`

## Labeling windows

Recommended default windows:

- `1d`
- `3d`
- `7d`
- `30d`

These should remain fixed so historical evals stay comparable.

## Outcome examples

### First-connection

Positive outcome examples:

- token volume spike after multiple first-time entries
- additional high-quality wallets enter in the next window
- price reaction exceeds configured threshold

### Shadow exit

Positive outcome examples:

- continued outflow after initial alert
- destination convergence toward exchange or bridge rails
- reduced holding concentration in subsequent windows

### Cluster movement

Positive outcome examples:

- multiple cluster members repeat the same action
- related entities join the move
- downstream price or liquidity reaction follows

## Modeling note

The first predictive models should learn from these structured labels before any
LLM fine-tuning. LLMs should explain signals using this dataset, not replace it.
