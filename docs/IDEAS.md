# Ideas & direction

Concrete, scoped ways to make `pwprofiler` a best-in-class **authorized**
password-audit tool. Everything here must keep the tool defensive and
authorized-use-only (see [SECURITY.md](../SECURITY.md)). Effort is a rough
t-shirt size; impact is on real audit effectiveness.

## Tier 1 — highest impact

### 1. A trained likelihood model for ranking  ·  impact: high · effort: M
The current `internal/rank` scorer is a transparent hand-tuned heuristic. Add an
optional **statistical model** so ordering reflects real frequencies:
- A frequency-weighted table of suffixes, years, and structural patterns derived
  from public breach-composition studies (shipped as data, not raw passwords).
- An **OMEN/Markov-style** per-position probability model for the word core, so
  `rank` scores by estimated guess number rather than shape alone.
- `pwprofiler rank --model freq.json` to let auditors supply their own
  frequency corpus (e.g. from a target's sector) without recompiling.
Keep the heuristic as the zero-config default; the model is opt-in.

### 2. Coverage & effort report  ·  impact: high · effort: M
`pwprofiler estimate -c profile.yaml [--hashrate 1e9]` → keyspace size, unique
candidates after policy, and expected wall-clock to exhaust at a given hashrate.
Answers the auditor's real question: *"is this list worth running?"* Extend
`check` to compare a generated list against an already-cracked potfile and report
net-new coverage.

### 3. Exclude-known & potfile awareness  ·  impact: high · effort: S
`--exclude cracked.txt` / `--exclude already-tried.txt` to skip candidates you've
already run, so repeated audits only spend time on new guesses. Pairs naturally
with `rank`.

## Tier 2 — strong additions

### 4. Richer rule export  ·  impact: med · effort: M
- Emit **John the Ripper** rules alongside hashcat.
- Import and optimize existing `.rule` files (dedupe, reorder by expected yield).
- Optionally rank rule lines the same way `rank` orders candidates.

### 5. Smarter token extraction  ·  impact: med · effort: M
- Transliteration / diacritic folding (`José → jose`).
- Locale-aware date formats and keyboard-walk seeds (`qwerty`, `1q2w3e`).
- Frequency-weighted provenance: name/DOB tokens rank above generic keywords, so
  combination and ranking both prefer the strongest signals.

### 6. Mask-guided generation  ·  impact: med · effort: M
Intersect token mutation with a hashcat-style **mask** (`?u?l?l?l?d?d?d?d`) so a
run targets exactly the structures a policy or prior analysis implies — a middle
ground between pure wordlist and pure brute force.

### 7. Phase 7 — isolated keyword crawler  ·  impact: med · effort: M
A CeWL-style, opt-in crawler (`colly`) that harvests keywords from an authorized
domain and feeds them in as `keywords`. Must stay fully isolated from the core,
respect `robots.txt`, and enforce scope/depth/rate limits and an explicit
authorization flag.

## Tier 3 — polish, trust & reach

### 8. Distribution & supply-chain trust  ·  effort: S–M
Homebrew tap, GitHub Releases with checksums, **cosign** signatures, an SBOM,
and reproducible builds. A container image for CI pipelines.

### 9. UX  ·  effort: S–M
Progress bar with ETA (CLI) and a live candidate counter (web UI); saved
profiles and **policy presets** (NIST 800-63B, common AD defaults); a run-diff
view; a one-click "generate → rank → download top N" flow in the browser.

### 10. Quality engineering  ·  effort: S
`go test -fuzz` for the profile/policy parsers and the rank scorer; golden-file
tests for generated lists and rule files; a benchmark gate in CI to catch
throughput regressions; race and coverage reporting.

### 11. Pipeline ergonomics  ·  effort: S
Structured JSON stats (`--stats-json`) for automation; stream straight to
`hashcat --stdin`; shardable output for distributed cracking.

---

**How to pick something up:** open an issue describing the slice you want to
build, keep it inside the package it belongs to, and bring tests. See
[CONTRIBUTING.md](../CONTRIBUTING.md).
