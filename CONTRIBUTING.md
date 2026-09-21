# Contributing to pwprofiler

Thanks for your interest in improving `pwprofiler`! This document explains how
to get set up, the standards the codebase holds to, and how to submit changes.

## Ground rule: authorized-use only

`pwprofiler` is a defensive/authorized-testing tool. Contributions must keep it
that way. We will not accept changes whose primary purpose is to facilitate
unauthorized access — for example, built-in targeting of third-party services,
credential exfiltration, or evasion tooling. Features should serve legitimate
password-audit and penetration-testing workflows.

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

The pipeline is one package per stage under `internal/`:

```
Config → Profile → Tokens → Combinations → Mutations → (Policy) → Output
         profile   tokens    combine        mutate/rules  policy    output
```

Each stage has a small, testable interface so stages stay swappable. Please
keep new logic inside the stage it belongs to rather than in `cmd/`.

## Standards (a change is not "done" until these pass)

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

## Reporting bugs and requesting features

Use the issue templates under **Issues → New issue**. For security issues,
follow [SECURITY.md](./SECURITY.md) instead of opening a public issue.

## License

By contributing, you agree that your contributions are licensed under the
project's [Apache License 2.0](./LICENSE).
