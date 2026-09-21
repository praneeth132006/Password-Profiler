# pwprofiler

A targeted **password-profiling wordlist generator** for **authorized** password
auditing and penetration testing. Given OSINT about a target (names, dates,
company, keywords), `pwprofiler` builds candidate password wordlists — and, from
Phase 4, hashcat `.rule` files — for offline cracking during engagements you are
authorized to perform.

Same category as [CUPP](https://github.com/Mebus/cupp),
[pydictor](https://github.com/LandGrey/pydictor), and
[psudohash](https://github.com/t3l3machus/psudohash) — with three deliberate
differentiators:

1. **Rule mode** — emit a small base wordlist + a generated hashcat `.rule` file
   instead of materializing giant wordlists to disk *(Phase 4)*.
2. **Policy-aware filtering** — given a target password policy, generate only
   candidates that satisfy it *(Phase 5)*.
3. **Explosion control** — hard budgets/caps at every pipeline stage so a naive
   run never produces terabytes.

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8.svg)](https://go.dev/)

---

## ⚠️ Authorized testing only

**This tool is for authorized security testing only.** Use it exclusively
against systems and accounts that you own or for which you hold **explicit,
written permission** to test. Generating or using password candidates to access
systems without authorization is illegal in most jurisdictions and unethical.
You are solely responsible for how you use this software. The sample profile
shipped in [`testdata/sample.yaml`](testdata/sample.yaml) is fictional.

---

## Install

**Build from source** (requires Go 1.23+):

```bash
git clone https://github.com/praneeth132006/Password-Profiler.git
cd Password-Profiler
go build -o pwprofiler ./cmd/pwprofiler
```

Or install the binary directly:

```bash
go install github.com/praneeth132006/Password-Profiler/cmd/pwprofiler@latest
```

## Quick start

```bash
# Generate candidates from a config to stdout
pwprofiler generate --config testdata/sample.yaml

# Write to a file instead
pwprofiler generate --config testdata/sample.yaml --output out.txt
```

Candidate lines go to **stdout**; the stats summary is written to **stderr**, so
you can pipe candidates straight into a cracker without polluting the stream.

## Usage

```
pwprofiler generate --config <file.yaml> [flags]

Flags:
  -c, --config string   path to YAML config (required)
  -o, --output string   write candidates here instead of the config's output.file / stdout
      --mode string     override output mode: wordlist|rules
```

## Config

`pwprofiler` is driven by a single YAML file:

```yaml
profile:
  first_name: John
  last_name: Doe
  nickname: JD
  dob: 1990-05-12          # YYYY-MM-DD
  partner: Jane
  pets: [rex, milo]
  company: Acme Corp
  domain: acme.com
  keywords: [falcons, chess]

rules:
  case: [lower, capitalize, upper]   # identity|lower|upper|capitalize|toggle|invert
  leet: partial                      # off | partial | full
  separators: ["", ".", "_"]         # joined between combined tokens
  affixes:
    years: {from_dob: true, extra: [2024, 2025]}   # appended + prepended
    suffixes: ["123", "!", "@", "01"]              # appended
  max_combine: 2                     # max tokens joined into one combination
  depth: 3                           # max chained mutations

policy:                              # (Phase 5)
  min_len: 8
  max_len: 16
  require: [upper, lower, digit]

output:
  mode: wordlist                     # wordlist | rules (rules => Phase 4)
  file: ""                           # empty => stdout
  budget: 5000000                    # global hard cap on candidates
  dedupe: true
```

Unknown keys and malformed values (bad dates, invalid enums) are rejected at
load time with a clear error.

### What's implemented today (Phases 1–3)

- **Config** parsing + validation (`internal/profile`).
- **Token extraction** (`internal/tokens`): multi-word splitting
  (`"Acme Corp"` → `acme`, `corp`, `acmecorp`, `acme_corp`), date decomposition
  (`1990-05-12` → `1990`, `90`, `05`, `12`, `0512`, `12051990`, …), domain
  labels, deterministic and de-duplicated output.
- **Combination engine** (`internal/combine`): singles plus bounded ordered
  concatenations of distinct tokens across the separator set
  (`john` + `doe` → `johndoe`, `john.doe`, `john_doe`, `doejohn`, …), capped by
  `max_combine` and guarded against explosion by the budget.
- **Full mutation engine** (`internal/mutate`) — composable transforms chained
  breadth-first up to `depth`:
  - **Case:** identity, lower, UPPER, Capitalize, tOGGLE, iNVERT.
  - **Leetspeak:** `off | partial | full`, e.g. `falcons` → `f@lc0n$`. `full`
    is combinatorial and capped per word so it cannot explode.
  - **Affixes:** year affixes derived from the DOB (±1 year) plus the current
    year and explicit extras, in 4- and 2-digit forms, appended *and* prepended
    (`john1990`, `1990john`); numeric/symbol suffixes appended.
  - **Structural:** reverse, duplicate, truncate (`john` → `nhoj`, `johnjohn`,
    `joh`).
- **Rule mode** (`internal/rules`): emit a small base wordlist + a hashcat
  `.rule` file encoding the mutation set (case, leet, affixes, structural), so
  hashcat expands `words × rules` on the fly instead of writing a giant list. A
  built-in rule interpreter validates that every emitted rule parses and
  produces its intended candidate.
- **Buffered output** (`internal/output`): `bufio.Writer`, exact dedup, global
  budget cap, and a stats summary.

Policy filtering and scale hardening land in later phases (see Roadmap).

## Cracking pipe examples

**hashcat** (straight wordlist attack, `-m 0` = MD5, adjust for your hash):

```bash
pwprofiler generate -c profile.yaml | hashcat -m 0 -a 0 hashes.txt
```

**John the Ripper**:

```bash
pwprofiler generate -c profile.yaml -o candidates.txt
john --wordlist=candidates.txt --format=raw-md5 hashes.txt
```

**Rule mode** (`--mode rules`) — the headline feature. Writes a small base
wordlist plus a hashcat `.rule` file instead of a giant materialized list:

```bash
pwprofiler generate -c profile.yaml --mode rules -o base.words
# -> writes base.words + base.rule, and prints the command to run:
hashcat -a 0 -m <hash-type> hashes.txt base.words -r base.rule
```

The `.rule` file uses standard hashcat functions (`:`, `l`, `u`, `c`, `sa@`,
`$1$2$3`, `^0^9^9^1`, `r`, `d`, `]`), each validated by the built-in interpreter
in [`internal/rules`](internal/rules/apply.go).

## Project layout

```
pwprofiler/
├── cmd/pwprofiler/      # CLI entrypoint (cobra)
├── internal/
│   ├── profile/         # YAML config parse + validate
│   ├── tokens/          # profile -> base tokens
│   ├── combine/         # join tokens (singles + bounded concatenations)
│   ├── mutate/          # composable transform engine
│   ├── rules/           # hashcat .rule emitter + validating interpreter
│   └── output/          # buffered, deduped, budget-capped sink
├── testdata/            # sample config
├── go.mod
└── README.md
```

## Development

```bash
make build     # build ./pwprofiler
make test      # go test ./...
make fmt       # gofmt -w .
make run       # run against testdata/sample.yaml
```

Every `internal/` package ships table-driven tests. `go build ./...` and
`go test ./...` must pass clean, and the code is kept `gofmt`-clean.

## Roadmap

- [x] **Phase 1 — MVP:** config → tokens → case + suffix mutations → wordlist.
- [x] **Phase 2 — Combination engine:** separators + `max_combine` cap.
- [x] **Phase 3 — Full mutation engine:** leet, affix years, structural, depth chaining.
- [x] **Phase 4 — Rule mode:** hashcat `.rule` emitter.
- [ ] **Phase 5 — Policy filter.**
- [ ] **Phase 6 — Scale hardening:** scalable dedup, worker pool, keyspace stats.
- [ ] **Phase 7 — (Optional) crawler:** CeWL-style keyword harvesting (isolated).

## License

Licensed under the [Apache License 2.0](./LICENSE).
