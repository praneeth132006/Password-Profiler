# Contributing to pwprofiler

Thanks for your interest in improving `pwprofiler`! This document explains how
to get set up, the standards the codebase holds to, and how to submit changes.

## Ground rule: authorized-use only

`pwprofiler` is a defensive/authorized-testing tool. Contributions must keep it
that way. We will not accept changes whose primary purpose is to facilitate
unauthorized access — for example, built-in targeting of third-party services,
credential exfiltration, or evasion tooling. Features should serve legitimate
password-audit and penetration-testing workflows.

Two specific lines we hold:

- **No real leaked credentials.** The tool generates candidate *guesses* and may
  use *public, breach-derived composition data* (common-password corpora,
  keyboard walks) to rank and seed. It must never ship, fetch, or emit real
  individuals' leaked `email:password` records. The bundled common-password
  corpus is public weak-password knowledge, not a credential dump.
- **Hashing stays audit-scoped.** `internal/hashes` covers only fast, unsalted
  algorithms used to match a captured hash against a candidate list
  (md5/sha1/sha256/sha512/ntlm). Do not add salted/slow schemes as a
  "precompute" feature — they can't be matched from a list, and such a request
  is a red flag.

## Getting started

Requirements: **Go 1.23+**.

```bash
git clone https://github.com/praneeth132006/Password-Profiler.git
cd Password-Profiler
make build        # or: go build ./...
make test         # or: go test ./...
```

Run it against the sample config:

```bash
make run          # ./pwprofiler generate --config testdata/sample.yaml
```

## Project architecture

The pipeline is one package per stage under `internal/`, orchestrated by the
CLI in `cmd/pwprofiler`:

```
Config → Profile → Tokens → Combinations → Mutations → (Policy) → Output → (Rank) → (Hash)
         profile   tokens    combine        mutate/rules  policy    output    rank     hashes
```

| Package            | Responsibility                                             |
| ------------------ | ---------------------------------------------------------- |
| `internal/profile` | Parse + validate YAML config (person/company/other schema) |
| `internal/tokens`  | Profile → base tokens (splitting, date decomposition, diacritic folding, keyboard-walk + common-password seeds) |
| `internal/combine` | Bounded cross-token combination                            |
| `internal/mutate`  | Composable transforms (case, leet, affix, structural)      |
| `internal/rules`   | hashcat / John `.rule` emitter + validating interpreter    |
| `internal/policy`  | Filter candidates against a target policy                  |
| `internal/dedup`   | Exact + Bloom-filter de-duplication                        |
| `internal/output`  | Buffered, deduped, budget-capped sink (optional line formatter) |
| `internal/rank`    | Likelihood scoring + best-first ordering (pure, reusable)  |
| `internal/hashes`  | Fast unsalted digests for match-against-capture (incl. vendored MD4/NTLM) |

Each stage has a small, testable interface so stages stay swappable. Please
keep new logic inside the stage it belongs to rather than in `cmd/`. Pure
library packages (`rank`, `policy`, `mutate`, …) must not import `cmd`.

## Standards (a change is not "done" until these pass)

- **Documented:** update README command examples, `docs/POLICIES.md` for policy/workflow semantics, `docs/ALGORITHM.md` for generation behavior, and CHANGELOG for every user-visible change. Keep limitations and research references explicit; do not claim policy compliance measures password strength.
- **Formatted:** `gofmt -l .` reports nothing. Run `make fmt`.
- **Vetted:** `go vet ./...` is clean.
- **Builds:** `go build ./...` succeeds.
- **Tested:** `go test ./...` passes. Every `internal/` package ships
  **table-driven tests**; new behavior needs new tests.
- **Idiomatic:** wrap errors with context (`fmt.Errorf("...: %w", err)`), no
  `panic` in library code, no unused dependencies (`go mod tidy`).
- **Explosion-safe:** any new generation stage must respect the budget/caps and
  never silently truncate output without reporting it.

CI runs all of the above on every pull request.

## Submitting changes

1. Fork and create a topic branch (`git checkout -b feat/my-change`).
2. Make your change with tests; keep commits focused.
3. Use clear commit messages (e.g. `feat(mutate): ...`, `fix(output): ...`).
4. Open a pull request describing the what and why, and how you tested it.

## Where to help

Open to ideas of all sizes. [`docs/IDEAS.md`](docs/IDEAS.md) lists concrete,
scoped directions (ranking models, a keyword crawler, a strength/coverage
report, richer rule export, and more) with rough effort estimates — a good place
to find something to pick up.

## Reporting bugs and requesting features

Use the issue templates under **Issues → New issue**. For security issues,
follow [SECURITY.md](./SECURITY.md) instead of opening a public issue.

## License

By contributing, you agree that your contributions are licensed under the
project's [Apache License 2.0](./LICENSE).
