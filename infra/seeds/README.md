# Seed Data

This directory contains repository-managed seed datasets used for local
development, bootstrapping, and deterministic worker flows.

## Rules

- Do not commit provider API keys, bearer tokens, passwords, or webhook secrets.
- Do not commit private customer exports or non-public operational dumps.
- Prefer public-source or synthetic seed material only.
- Keep files small enough to review in git diffs.

## Current files

- `qorvi_system_seed_wallets.json`
  - curated wallet seed labels used by backfill and seed-enqueue flows
  - intended for bootstrap and testing, not as a secret store

Runtime overrides can still be supplied through environment variables such as
`DUNE_SEED_EXPORT_PATH` or `DUNE_SEED_EXPORT_JSON`.
