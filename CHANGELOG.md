# Changelog

## 0.5.0 — 2026-09-23

- Added exhaustive configured-rule wordlist traversal with exact dedup and no count, work, or intermediate byte caps.
- Exposed token combination depth, mutation depth, full-leet position limits, and token reuse in file CLI, console and UI.
- Increased session defaults to 100,000 outputs and two joined tokens with four separators. Removed the CLI session budget ceiling.
- Added browser export of exhaustive YAML configurations for large direct-to-disk CLI runs.
- Added known-complete search-space tests and documented precise completeness and resource limits.

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) once it
reaches 1.0.

## [Unreleased]

### Changed
- Redesigned the local web workspace (`pwprofiler serve`): refined dark theme
  with a light-mode variant, gradient hero and logo, numbered step badges,
  chip-style required-character toggles, sticky policy panel, a loading spinner
  and richer result card. All within the existing strict CSP (system fonts, no
  inline or external assets).

### Added
- Web UI shows per-category file-selection counts and a "Copy preview" button;
  the result card reports candidate count and limit/partial-search notices.

## [0.4.0] - 2026-09-23

### Changed
- Stream combinations and breadth-first mutations; stop immediately when output is sufficient.
- Enumerate full-leet substitutions lazily with cancellation inside traversal.
- Replace quadratic combination preallocation with exact-length path traversal.
- Stream bounded worker batches and propagate cancellation upstream.

### Added
- Separate configurable search limits for combinations, attempts, states and intermediate bytes.
- CLI/browser notices when search limits reduce coverage.
- Reference equivalence tests, fuzzing, concurrency regression tests and reproducible benchmarks.
- Algorithm research and evaluation guide with measured performance and explicit limitations.

### Fixed
- Small output budgets no longer prevent examination of later policy-matching inputs.
- Ctrl+C now cancels wordlist search and removes its partial output file.

### Compatibility
- Depth and combination width now support 1–8; worker count supports 1–64.
- Large searches may stop at the new default work limits; use documented YAML settings to adjust them.


## [0.3.0] - 2026-09-22

### Added
- Shared advanced policies: exact blocklists, forbidden characters, UTF-8 byte limits and consecutive repeat limits.
- Standalone policy YAML accepted by `files --policy` and the console's `policy` command.
- Streaming `check` command with aggregate JSON reports and optional failing exit status.
- `validate` command for generation-config preflight without output side effects.
- Versioned, validated console session save/load with private file permissions.
- Browser advanced-policy controls and generation cancellation.
- Policy/workflow reference documentation with NIST and OWASP research sources.

### Fixed
- Reject multiple YAML documents rather than silently ignoring trailing documents.
- Apply every policy constraint consistently, including rules-mode incompatibility checks.

## [0.2.0] - 2026-09-22

### Added
- Localhost browser workspace with multi-file upload, policy controls, preview and text download.
- Interactive `console` (also the default command), categorized file inputs, editable policy and session review.
- Scriptable `files` command with repeated inputs and password-policy flags.
- Debian/Ubuntu package builder, source installation target and tagged release workflow.
- Integration tests for file/console generation, browser downloads, policy compliance and invalid requests.

### Fixed
- File destinations no longer silently overwrite existing data; failed/empty wordlist runs remove their new output.
- Rules mode rejects policies that it cannot enforce. Negative generation budgets are rejected.

### Previously added
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
