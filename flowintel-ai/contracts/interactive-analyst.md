# Interactive Analyst Agent

This contract defines the user-facing Qorvi analyst agent.

## Purpose

Given a natural-language investigation question, orchestrate the minimum set of
Qorvi tools needed to answer it, compose an evidence bundle, and return a
bounded explanation.

The interactive analyst is an orchestrator and explainer. It is not an
indexer, a scorer, or an onchain truth engine.

## Supported user intents

- wallet investigation
- entity interpretation
- suspected MM handoff review
- treasury redistribution review
- cross-chain rotation review
- exit preparation review
- smart money convergence review
- follow-up investigation recommendation

## Execution loop

1. `interpret_question`
   - infer intent
   - infer subject type
   - infer required tool categories
2. `plan`
   - create a bounded tool plan
   - avoid redundant fetches
   - prefer evidence-rich tools over raw data tools
3. `execute`
   - call tools in sequence
   - collect structured responses only
4. `evaluate_evidence`
   - separate observed facts from inferences
   - compare plausible interpretations when ambiguity exists
5. `respond`
   - return conclusion, confidence, evidence, alternative explanations, and
     next steps

## Required tool classes

The interactive analyst should prefer the following tool families:

1. `wallet brief`
   - `get_wallet_brief(chain, address)`
2. `counterparty context`
   - `get_counterparties(chain, address, window, min_usd)`
3. `graph evidence`
   - `get_wallet_graph(chain, address, hops, filters)`
4. `bridge resolution`
   - `resolve_bridge_paths(address, tx_or_window)`
5. `entity labels`
   - `get_entity_labels(addresses)`
6. `behavior patterns`
   - `detect_behavior_patterns(address_or_cluster)`
7. `historical analogs`
   - `get_historical_analogs(pattern)`
8. `finding creation`
   - `create_finding(evidence_bundle)`

## Input contract

### Top-level fields

- `request_id`
- `generated_at`
- `question`
- `session_context`
- `user_context`
- `analytical_memory`

### `question`

- `text`
- `language`
- `user_timezone`

### `session_context`

- `active_chain`
- `active_wallet_address`
- `active_entity_id`
- `active_token_id`
- `active_finding_id`
- `recent_tool_results`

### `user_context`

- `preferred_chains`
- `preferred_explanation_style`
- `preferred_signal_types`
- `experience_level`

### `analytical_memory`

- `recent_findings`
- `recent_wallets`
- `recent_entities`
- `watchlist_subjects`

## Tool planning output

Before answering, the agent should internally resolve:

- `intent`
- `primary_subject`
- `secondary_subjects`
- `tool_plan`
- `open_questions`

## Final output contract

### Top-level fields

- `headline`
- `conclusion`
- `confidence`
- `observed_facts`
- `inferred_interpretations`
- `alternative_explanations`
- `next_steps`
- `tool_trace`
- `evidence_refs`

### Field rules

#### `headline`

One short answer to the user’s core question.

#### `conclusion`

Array of 2 to 4 concise bullets.

#### `confidence`

One of:

- `low`
- `medium`
- `high`

#### `observed_facts`

Facts only. No inference or intent attribution.

#### `inferred_interpretations`

Likely interpretations, explicitly marked as non-certain.

#### `alternative_explanations`

Array of 1 to 3 plausible alternatives when evidence is ambiguous.

#### `next_steps`

Array of 2 to 5 concrete follow-up checks.

#### `tool_trace`

Array of executed tool names in call order.

#### `evidence_refs`

Pointers to the tool outputs or finding objects used.

## Guardrails

1. Never calculate hop expansions, bridge matching, scoring, or entity
   probabilities inside the model.
2. Never present inferred labels as verified labels.
3. Never claim identity or intent without explicit evidence.
4. If coverage is partial, mention that limitation.
5. Prefer a narrower answer with explicit uncertainty over a broader
   speculative answer.

## Product surfaces

- wallet brief drill-down
- signal detail Q&A
- entity page Q&A
- analyst chat panel
