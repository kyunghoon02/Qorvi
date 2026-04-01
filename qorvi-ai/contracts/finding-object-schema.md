# Finding Object Schema

This document defines the canonical finding object shared by deterministic
engines, background detection, and AI explanation layers.

## Purpose

Standardize how Qorvi represents interpreted onchain events before they are
rendered in feeds, alerts, wallet briefs, and entity pages.

The finding object is the boundary between:

- deterministic calculation
- evidence composition
- AI summarization
- UI serving

## Canonical schema

```json
{
  "finding_id": "uuid",
  "finding_type": "suspected_mm_handoff",
  "subject": {
    "subject_type": "wallet",
    "chain": "ethereum",
    "address": "0x..."
  },
  "confidence": 0.78,
  "importance_score": 0.84,
  "summary": "Large outbound transfer from a fund-adjacent wallet to a probable market-making counterparty.",
  "importance_reason": [
    "Counterparty behavior is consistent with repeated downstream distribution.",
    "Post-transfer path shows exchange proximity and fanout."
  ],
  "observed_facts": [
    "A large outbound transfer was observed within the indexed coverage window.",
    "The receiving wallet has probable market-maker characteristics."
  ],
  "inferred_interpretations": [
    "This may represent a market-maker handoff rather than a routine peer transfer."
  ],
  "evidence": [
    {
      "type": "entity_label",
      "value": "fund_adjacent",
      "confidence": 0.72
    },
    {
      "type": "transfer_size_usd",
      "value": 2100000
    },
    {
      "type": "counterparty_behavior",
      "value": "repeated distribution patterns"
    },
    {
      "type": "post_transfer_path",
      "value": "fanout + exchange proximity increase"
    }
  ],
  "next_watch": [
    {
      "subject_type": "wallet",
      "chain": "ethereum",
      "address": "0x..."
    },
    {
      "subject_type": "token",
      "chain": "solana",
      "symbol": "XYZ"
    }
  ],
  "dedup_key": "string",
  "observed_at": "timestamp",
  "coverage": {
    "coverage_start_at": "timestamp",
    "coverage_end_at": "timestamp",
    "coverage_window_days": 180
  }
}
```

## Required fields

- `finding_id`
- `finding_type`
- `subject`
- `confidence`
- `importance_score`
- `summary`
- `evidence`
- `observed_at`
- `coverage`

## Optional fields

- `importance_reason`
- `observed_facts`
- `inferred_interpretations`
- `next_watch`
- `dedup_key`

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

## Field semantics

### `confidence`

How strongly the current evidence supports the finding.

### `importance_score`

How important this finding is for ranking in feeds, alerts, and watchlists.

### `observed_facts`

Directly observed evidence-backed statements only.

### `inferred_interpretations`

Interpretations derived from the evidence bundle. Never present these as
confirmed facts.

### `next_watch`

Addresses, entities, or tokens worth monitoring next because they are directly
relevant to the current finding.

## Guardrails

1. A finding must be traceable back to structured evidence.
2. Coverage metadata must be preserved so the UI can state analysis bounds.
3. Inferred interpretations must never be the only support for a finding.
4. `finding_type` classification should be deterministic before AI phrasing.
5. AI summaries should decorate the finding object, not replace it.

## Primary uses

- findings feed items
- alert payloads
- wallet brief key findings
- entity interpretation panels
- historical analog comparisons
