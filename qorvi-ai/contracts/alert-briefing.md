# Alert Briefing Contract

This contract defines the AI agent that turns an alert event into a short,
actionable operator briefing.

## Purpose

Given an alert event and its associated signal context, produce a brief that
helps the user understand:

- what happened
- why it is important
- what to check next

The alert briefing should be optimized for fast triage rather than deep
investigation.

## Input contract

### Top-level fields

- `request_id`
- `generated_at`
- `alert`
- `signal`
- `subject`
- `coverage`
- `rule_context`
- `evidence`
- `recommended_links`

### `alert`

- `alert_event_id`
- `alert_rule_id`
- `severity`
- `title`
- `created_at`
- `observed_at`
- `is_read`

### `signal`

- `signal_type`
- `label`
- `rating`
- `value`
- `source`

### `subject`

- `subject_type`
- `chain`
- `address`
- `label`
- `cluster_id`
- `entity_keys`

### `coverage`

- `coverage_window_days`
- `last_indexed_at`
- `is_partial`

### `rule_context`

- `rule_name`
- `rule_type`
- `minimum_severity`
- `watchlist_name`
- `cooldown_seconds`

### `evidence`

Array of:

- `summary`
- `source`
- `confidence`
- `observed_at`
- `payload_json`

### `recommended_links`

Array of:

- `label`
- `href`
- `kind`

## Output contract

### Top-level fields

- `headline`
- `brief`
- `why_now`
- `next_actions`
- `confidence`
- `evidence_refs`

### Field rules

#### `headline`

One short triage line.

#### `brief`

Array of 2 to 3 short bullets for inbox display.

#### `why_now`

Array of 1 to 2 bullets that explain urgency or timing.

#### `next_actions`

Array of 2 to 4 actions with concrete pivots.

#### `confidence`

One of:

- `low`
- `medium`
- `high`

#### `evidence_refs`

Pointers to the evidence sections used for the briefing.

## Guardrails

1. Optimize for operational clarity over narrative detail.
2. Do not overstate severity beyond the supplied alert severity.
3. Mention partial coverage or indexing-in-progress when relevant.
4. Avoid identity claims unless evidence is explicit.
5. Keep every recommended action grounded in available product surfaces.

## UI usage

Primary surfaces:

- alert center inbox
- alert detail drawer
- digest and notification summarization
