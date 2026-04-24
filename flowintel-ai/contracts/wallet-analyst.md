# Wallet Analyst Contract

This contract defines the first structured AI agent for Qorvi.

## Purpose

Given a wallet and its indexed evidence bundle, produce a concise,
evidence-backed analyst brief for the product UI.

The wallet analyst does not invent facts or assign hard labels without evidence.

## Input contract

### Top-level fields

- `request_id`
- `generated_at`
- `wallet`
- `coverage`
- `scores`
- `latest_signals`
- `top_counterparties`
- `graph`
- `entities`
- `holdings`
- `recent_flow`

### `wallet`

- `wallet_id`
- `chain`
- `address`
- `label`
- `cluster_id`
- `status`

### `coverage`

- `status`
  - `ready`
  - `indexing`
- `coverage_start_at`
- `coverage_end_at`
- `coverage_window_days`
- `last_indexed_at`

### `scores`

- `cluster_score`
- `shadow_exit_risk`
- `smart_money_score`
- `score_confidence`

### `latest_signals`

Array of:

- `signal_type`
- `label`
- `rating`
- `value`
- `observed_at`
- `source`

### `top_counterparties`

Array of:

- `chain`
- `address`
- `entity_key`
- `entity_label`
- `interaction_count`
- `direction_label`
- `inbound_amount`
- `outbound_amount`
- `primary_token`
- `first_seen_at`
- `latest_activity_at`

### `graph`

- `node_count`
- `edge_count`
- `selected_node`
- `selected_relationship`
- `relationships`

Each relationship should include:

- `kind`
- `family`
- `directionality`
- `confidence`
- `evidence_summary`
- `last_tx_hash`
- `primary_token`

### `entities`

Array of:

- `entity_key`
- `entity_label`
- `entity_type`
- `source`
- `confidence`

### `holdings`

Array of:

- `symbol`
- `balance_formatted`
- `value_usd`
- `portfolio_percentage`

### `recent_flow`

- `incoming_tx_count_7d`
- `outgoing_tx_count_7d`
- `incoming_tx_count_30d`
- `outgoing_tx_count_30d`

## Output contract

### Top-level fields

- `headline`
- `summary`
- `why_it_matters`
- `next_steps`
- `confidence`
- `observed_facts`
- `inferred_interpretations`
- `evidence_refs`

### Field rules

#### `headline`

One short sentence for the wallet’s most important current state.

#### `summary`

Array of 2 to 4 concise bullets.

#### `why_it_matters`

Array of 1 to 3 bullets that explain relevance for traders, researchers, or
operators.

#### `next_steps`

Array of 2 to 4 actionable investigation suggestions.

#### `confidence`

One of:

- `low`
- `medium`
- `high`

#### `observed_facts`

Array of direct facts from the input bundle.

#### `inferred_interpretations`

Array of bounded inferences that are clearly labeled as interpretations.

#### `evidence_refs`

Array of pointers to source sections, for example:

- `scores.shadow_exit_risk`
- `latest_signals[0]`
- `top_counterparties[2]`
- `graph.relationships[1]`

## Guardrails

1. Never claim facts outside the provided coverage window.
2. Separate observed facts from inferred interpretations.
3. Never state an entity identity as confirmed unless the source confidence is
   explicit and evidence-backed.
4. If coverage is partial or indexing is active, mention that limitation.
5. Prefer omission over speculation when evidence is weak.

## UI usage

The first integration target is wallet detail:

- `AI Brief`
- `Why this matters`
- `Next steps`

This contract should remain model-agnostic. Prompting and model selection belong
in later serving code.
