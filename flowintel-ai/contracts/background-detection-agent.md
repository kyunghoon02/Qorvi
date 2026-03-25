# Background Detection Agent

This contract defines the non-interactive detection agent that produces
candidate findings from indexed data.

## Purpose

Generate candidate findings from deterministic backend evidence, enrich them
with ranked evidence bundles, and hand them off to downstream alerting and
AI-summary layers.

The background detection agent is primarily a scheduler and classifier around
existing scoring and rule engines. It should not replace deterministic
calculation paths.

## Role

- run on batch and realtime triggers
- collect precomputed evidence
- classify candidate findings
- merge duplicates
- assign priority
- request AI summaries after evidence is stabilized

## Non-goals

- direct RPC or provider ingestion
- direct transaction normalization
- graph traversal beyond existing backend tools
- raw score calculation inside the model

## Trigger sources

- new normalized transaction batches
- scoring snapshot updates
- wallet label updates
- entity interpretation updates
- watchlist-triggered reevaluations
- scheduled historical refresh windows

## Required tool classes

1. `finding candidate scan`
   - read score/materialization deltas
2. `behavior pattern detector`
   - detect exit preparation, convergence, redistribution, handoff
3. `entity interpretation`
   - enrich with verified/probable entity context
4. `bridge path resolver`
   - attach cross-chain evidence when applicable
5. `historical analog lookup`
   - attach comparable past cases
6. `finding writer`
   - persist canonical finding object
7. `alert bridge`
   - fan out eligible findings into alert evaluation

## Input contract

### Top-level fields

- `run_id`
- `generated_at`
- `trigger_type`
- `subject_batch`
- `score_deltas`
- `evidence_candidates`
- `existing_findings`

### `trigger_type`

One of:

- `realtime_ingest`
- `scheduled_batch`
- `score_refresh`
- `label_refresh`
- `manual_replay`

### `subject_batch`

Array of:

- `subject_type`
- `subject_key`
- `chain`
- `updated_at`

### `score_deltas`

Array of:

- `score_type`
- `previous_value`
- `current_value`
- `delta`
- `observed_at`

### `evidence_candidates`

Array of:

- `evidence_type`
- `subject_key`
- `payload_json`
- `confidence`
- `observed_at`

### `existing_findings`

Array of:

- `finding_id`
- `finding_type`
- `subject_key`
- `dedup_key`
- `status`
- `observed_at`

## Output contract

### Top-level fields

- `generated_findings`
- `suppressed_findings`
- `merged_findings`
- `alert_candidates`

### `generated_findings`

Array of finding objects that pass minimum evidence thresholds.

### `suppressed_findings`

Array of findings rejected for duplication, low confidence, or whitelist logic.

### `merged_findings`

Array of finding ids or dedup keys merged into an existing active finding.

### `alert_candidates`

Array of finding ids eligible for downstream alert evaluation.

## Baseline finding types

- `suspected_mm_handoff`
- `treasury_redistribution`
- `cross_chain_rotation`
- `coordinated_accumulation`
- `exit_preparation`
- `cex_deposit_pressure`
- `smart_money_convergence`
- `fund_adjacent_activity`
- `high_conviction_entry`

## Guardrails

1. Generate findings from evidence bundles, not raw text reasoning alone.
2. Merge duplicate findings before asking for AI summaries.
3. Keep `observed facts` and `interpreted meaning` separate in persisted data.
4. Do not emit alerts directly without passing through alert policy evaluation.
5. Persist why a finding was suppressed or merged.

## Product surfaces

- findings feed
- alert candidate queue
- analyst review workflows
