# Signal Explainer Contract

This contract defines the AI agent that explains why a Qorvi signal was
raised and how to investigate it further.

## Purpose

Given a single signal event and its supporting evidence bundle, produce an
explainable signal brief for the product UI and alerting surfaces.

The signal explainer must not re-score the signal. It explains deterministic
results from the main platform.

## Input contract

### Top-level fields

- `request_id`
- `generated_at`
- `signal`
- `subject`
- `coverage`
- `score_context`
- `evidence`
- `graph_context`
- `counterparty_context`
- `historical_comparables`

### `signal`

- `signal_event_id`
- `signal_type`
  - `cluster_score`
  - `shadow_exit`
  - `first_connection`
  - `rotation_candidate`
  - `token_adoption`
- `label`
- `rating`
- `value`
- `source`
- `observed_at`

### `subject`

- `subject_type`
  - `wallet`
  - `cluster`
  - `entity`
  - `token`
- `subject_key`
- `chain`
- `display_label`

### `coverage`

- `coverage_start_at`
- `coverage_end_at`
- `coverage_window_days`
- `last_indexed_at`
- `is_partial`

### `score_context`

- `cluster_score`
- `shadow_exit_risk`
- `smart_money_score`
- `entity_confidence`
- `label_confidence`

### `evidence`

Array of structured evidence rows:

- `evidence_type`
- `confidence`
- `source`
- `summary`
- `observed_at`
- `payload_json`

### `graph_context`

- `node_count`
- `edge_count`
- `focused_relationships`
- `entity_proximity`

### `counterparty_context`

- `top_counterparties`
- `shared_destinations`
- `shared_funders`
- `primary_tokens`

### `historical_comparables`

Array of:

- `comparable_id`
- `signal_type`
- `outcome_label`
- `outcome_score`
- `resolved_window`
- `summary`

## Output contract

### Top-level fields

- `headline`
- `what_happened`
- `why_it_triggered`
- `why_it_matters`
- `investigation_paths`
- `confidence`
- `observed_facts`
- `inferred_interpretations`
- `evidence_refs`

### Field rules

#### `headline`

One sentence summarizing the signal in operator-friendly language.

#### `what_happened`

Array of 2 to 4 bullets describing the observed event.

#### `why_it_triggered`

Array of 1 to 3 bullets that map directly to deterministic signal evidence.

#### `why_it_matters`

Array of 1 to 3 bullets focused on market, cohort, or operator relevance.

#### `investigation_paths`

Array of 2 to 4 concrete next checks.

#### `confidence`

One of:

- `low`
- `medium`
- `high`

#### `observed_facts`

Direct factual statements only.

#### `inferred_interpretations`

Bounded interpretations only.

#### `evidence_refs`

Pointers into the provided signal bundle.

## Guardrails

1. Never reinterpret the signal as a different signal type than the input.
2. Treat historical comparables as context, not proof of future outcome.
3. When coverage is partial or indexing is active, state that uncertainty.
4. Separate evidence-backed triggers from speculative market meaning.
5. Never claim intent, identity, or causality without explicit evidence.

## UI usage

Primary surfaces:

- signal feed cards
- signal detail panels
- analyst-side review workflows
