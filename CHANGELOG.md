# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) once it
reaches 1.0.

## [Unreleased]

### Added
- **Phase 6 — Scale hardening**: pluggable de-duplication (`internal/dedup`)
  with an exact hash set and a memory-bounded Bloom filter (≈9 MB for 5M items
  at a 0.1% false-positive rate), selectable via `--dedup auto|exact|bloom`; a
  `--workers N` mutation pool that parallelizes generation while a single
  consumer serializes writes; and richer stats (dedup mode, elapsed, rate).
- **Phase 5 — Policy filter** (`internal/policy`): drop candidates that can't
  satisfy the target policy (min/max length, required character classes) before
  they reach the output; the stats line reports how many were rejected.
- **Phase 4 — Rule mode** (`internal/rules`): emit a small base wordlist plus a
  hashcat `.rule` file encoding the mutation set, with a self-contained rule
  interpreter that validates every emitted rule produces its intended
  candidate. `--mode rules` reports artifacts and estimated keyspace.
- **Phase 3 — Full mutation engine** (`internal/mutate`): leetspeak
  (`off|partial|full`, capped), year affixes derived from DOB ± range + current
  year + extras (appended and prepended), structural mutations
  (reverse/duplicate/truncate), and depth-bounded chaining of composable
  transforms.
- **Phase 2 — Combination engine** (`internal/combine`): singles plus bounded
  ordered concatenations of distinct tokens across a separator set, capped by
  `max_combine` and guarded by the budget.
- **Phase 1 — MVP**: YAML config parsing/validation (`internal/profile`), token
  extraction with multi-word splitting and date decomposition
  (`internal/tokens`), case + suffix mutations, and a buffered, de-duplicated,
  budget-capped output sink (`internal/output`) behind a cobra CLI.
- Open-source scaffolding: README with authorized-use notice, `LICENSE`
  (Apache-2.0), `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, issue
  and pull-request templates, and a GitHub Actions CI workflow.

### Notes
- Phases 1–6 complete. The optional CeWL-style crawler (Phase 7) is not yet
  implemented.
