# FlowIntel AI Workspace

This directory is the dedicated workspace for the FlowIntel AI analyst layer.
It is intentionally separated from the main product code so dataset design,
evaluation, prompt contracts, and model-serving experiments can move without
polluting the core API, worker, and web application paths.

## Scope

- Candidate discovery and tracking schema for AI-ready analysis
- Evidence and signal outcome dataset design
- Agent input and output contracts
- Evaluation sets and replay cases
- Prompt, rubric, and serving experiments

## Principles

- Keep deterministic ingestion, labeling, and scoring in the main product stack
- Use AI primarily as an explanation and investigation layer first
- Store structured context and structured outputs before adding model-specific code
- Treat datasets and evals as first-class artifacts

## Initial workstreams

1. `datasets/`
   - Signal outcome labels
   - Wallet and cluster event snapshots
   - Candidate discovery sources
2. `contracts/`
   - Wallet analyst payload
   - Signal explainer payload
   - Alert briefing payload
3. `evals/`
   - Golden cases
   - Replay prompts
   - Scoring rubrics
4. `engine-hardening-roadmap.md`
   - Bridge / exchange / treasury / MM / early-conviction engine upgrade plan

## Relationship to the main repo

- Product code remains under `apps/` and `packages/`
- AI-specific design and experiments start here
- Once contracts stabilize, serving code can be promoted into `apps/api` and
  `apps/web`
