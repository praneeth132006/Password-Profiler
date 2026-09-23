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

## Local UI and interactive console

Build once with `make build`, then choose a workflow:

```bash
./pwprofiler serve           # open http://127.0.0.1:8080 in your browser
./pwprofiler serve --port 9000
./pwprofiler console         # interactive workspace; also the default with no arguments
```

The embedded browser UI needs no Node.js, external assets, or cloud service.
Upload multiple business-email, name, company and keyword files, or paste
entries directly. Set minimum/maximum length, required character classes and a
candidate limit. Generate, preview, and download `passwords.txt`.
The server binds only to `127.0.0.1`, checks Host/Origin, processes one generation
at a time, and keeps uploads and generated results in memory. Stop with Ctrl+C.

The console supports a familiar command-driven workflow:

```text
pwprofiler > add emails /path/to/email.txt
[+] Business email found : /path/to/email.txt (12 lines)
pwprofiler > add names /path/to/names.txt
pwprofiler > add companies /path/to/companies.txt
pwprofiler > add keywords /path/to/keywords.txt
pwprofiler > set min 10
pwprofiler > set max 24
pwprofiler > set require upper,lower,digit,special
pwprofiler > set budget 10000
pwprofiler > set output /path/to/passwords.txt
pwprofiler > show options
pwprofiler > run
pwprofiler > exit
```

Use `help` for all commands, `reset` to clear the session, and `set require none`
to disable character-class requirements. Paths may contain spaces, with or
without surrounding quotes. Input categories are labels; each uses the same
local token extraction (including splitting email addresses into components).

For scripts, use repeatable file flags:

```bash
./pwprofiler files --emails email.txt --names names.txt \
  --input keywords.txt --input more-keywords.txt \
  --min-length 10 --max-length 24 --require upper,lower,digit,special \
  --budget 10000 --output passwords.txt
```

These workflows accept UTF-8 text, one entry per line (not structured CSV).
Blank lines and `#` comments are ignored. Limits: 1 MiB per file, 256 bytes per
entry, 2,000 entries per session, 4 MiB per browser request, and 100,000 output
candidates. Password lengths are measured in Unicode characters. Defaults are
8–24 characters, all four classes required, and 10,000 unique candidates.
Case transforms, partial leetspeak, structural changes and common suffixes are
applied with depth 2. Use the YAML workflow below for custom mutation rules and
cross-token combinations. A candidate limit is a ceiling, not a requested count;
restrictive policies may yield fewer or zero matches. Zero matches return an
error instead of leaving an empty file. File output refuses to overwrite
existing files and uses owner-only permissions on Unix.

## Streaming generation engine

The generator now produces combinations and mutations on demand, stopping work
as soon as the output budget is reached. Full leetspeak no longer materializes
all substitutions first. Search limits independently bound mutation states,
attempts and intermediate size, with explicit partial-coverage notices.
A small output budget can still search later input words that match the policy.

See [algorithm design, research and benchmarks](docs/ALGORITHM.md) for settings,
compatibility guarantees, performance measurements and their limitations.

## Reusable policies, sessions and verification

- **Advanced policies:** exact password blocklists, forbidden characters,
  maximum UTF-8 bytes and consecutive-repeat limits, available in YAML, CLI,
  console and the browser's Advanced policy section.
- **Streaming check:** validate an existing wordlist against a policy, producing
  aggregate JSON counts without including passwords.
- **Saved sessions:** `save session.json` and `load session.json` preserve loaded
  inputs and options in the console. These are private plaintext files.
- **Preflight:** `validate --config profile.yaml` checks configuration without
  generating passwords or touching output files.
- **Browser cancellation:** cancel an in-flight generation while keeping inputs.

```bash
./pwprofiler files -i words.txt --policy testdata/policy.yaml -o passwords.txt
./pwprofiler check -i passwords.txt --policy testdata/policy.yaml --fail-on-reject
./pwprofiler validate -c testdata/sample.yaml
```

See [policy and workflow documentation](docs/POLICIES.md) for exact semantics,
report fields, session privacy, examples, limitations and research references.
Policy compliance is not a password-strength guarantee.

## Debian / Ubuntu installation

On Debian/Ubuntu with Go and `dpkg-dev` installed, build a local package:

```bash
make deb                                  # amd64 by default
ARCH=arm64 make deb                       # optional ARM64 package
sudo apt install ./dist/pwprofiler_0.5.0_amd64.deb
pwprofiler console
pwprofiler serve
```

The package contains a self-contained binary, including the UI, and installs
`/usr/bin/pwprofiler`. Remove it with `sudo apt remove pwprofiler`.
The release workflow builds both architectures and checksums when a `v*` tag
is pushed. A release has not been published by this change.

**`sudo apt install pwprofiler` without a local file requires a configured APT
repository. This project is not currently published in the Debian/Ubuntu package
indexes.** Use the local `.deb` command above; it does not require adding a
third-party repository. See [packaging notes](packaging/README.md) for publishing.

For a conventional source installation on macOS/Linux:

```bash
make build
sudo make install                         # /usr/local/bin/pwprofiler
# Or: make install PREFIX="$HOME/.local"   # add ~/.local/bin to PATH
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
  -c, --config string    path to YAML config (required)
  -o, --output string    write candidates here instead of the config's output.file / stdout
      --mode string      override output mode: wordlist|rules
      --workers int      mutation worker goroutines (>1 is faster but unordered) (default 1)
      --dedup string     dedup strategy: auto|exact|bloom (default "auto")
```

For large runs, `--workers N` (1–64) parallelizes mutation across N goroutines (a single
consumer still serializes writes, so output is correct but unordered), and
`--dedup bloom` keeps memory bounded regardless of output size. `auto` (the
default) switches to a Bloom filter once the budget is large. Bloom filters can
omit unique candidates through false positives; use exact or exhaustive mode
when complete coverage of the configured search is required.

## Large and exhaustive wordlists

File, console and browser defaults now combine up to **two tokens** with four
separators and allow **100,000 outputs**. To write a larger finite search directly
to disk without count or work caps:

```bash
pwprofiler files -i words.txt --max-combine 3 --depth 2 --leet full \
  --repeat-tokens --exhaustive -o passwords.txt
```

Add `--leet-cap -1` to substitute every eligible position; the default is three
positions per leet step. This can multiply the search space dramatically.
`--max-combine` and `--depth` each accept 1–8. All generated results still pass the
password policy. Exhaustive mode forces exact deduplication, including when
`--dedup bloom` was requested, and ignores the output budget and resource caps.
It completes the **configured transformations**, not every conceivable password.
Exact deduplication and mutation frontiers can require substantial RAM, disk and
time. Ctrl+C cancels; cancelled/failed file runs remove their partial output.

The console supports `set exhaustive true`, `set max-combine 3`, `set depth 2`,
`set leet full`, `set leet-cap -1` and `set repeat-tokens true`.
The browser's **Generation coverage** controls configure the same search. For
large runs, click **Export exhaustive CLI configuration**, save `audit.yaml`,
and run `pwprofiler generate -c audit.yaml -o passwords.txt`. The download embeds
your source words and policy; keep it private. Browser wordlist downloads remain
limited to 100,000 candidates and 60 seconds. Neither limit applies to CLI
exhaustive runs. YAML users can set `rules.exhaustive: true` or pass `--exhaustive`.
See [the exact coverage contract](docs/ALGORITHM.md#exhaustive-configured-rule-mode).

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

policy:                              # emit only candidates that satisfy this
  min_len: 8
  max_len: 16
  require: [upper, lower, digit]     # upper | lower | digit | special

output:
  mode: wordlist                     # wordlist | rules
  file: ""                           # empty => stdout
  budget: 5000000                    # global hard cap on candidates
  dedupe: true
```

Unknown keys and malformed values (bad dates, invalid enums) are rejected at
load time with a clear error.

### What's implemented today (Phases 1–6)

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
- **Policy filter** (`internal/policy`): drop candidates that can't satisfy the
  target policy — min/max length and required character classes (upper, lower,
  digit, special) — shrinking the keyspace before it reaches the cracker. The
  stats line reports how many candidates the policy rejected.
- **Buffered output** (`internal/output`): `bufio.Writer`, pluggable dedup,
  global budget cap, and a stats summary (count, dedup mode, elapsed, rate).
- **Scale hardening** (`internal/dedup` + worker pool): choose exact
  (map-based) or **Bloom-filter** de-dup — the Bloom filter holds ~9 MB for 5M
  items at a 0.1% false-positive rate instead of hundreds of MB — and run
  mutation across a `--workers` pool (a single consumer serializes writes, so
  results stay correct; ~2.6× faster at 4 workers on the sample).

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
pwprofiler generate -c rules-profile.yaml --mode rules -o base.words
# -> writes base.words + base.rule, and prints the command to run:
hashcat -a 0 -m <hash-type> hashes.txt base.words -r base.rule
```

Use a configuration with `policy: {}` for rules mode. Rules mode rejects a
nonempty policy because it cannot enforce the final mutated passwords; use
wordlist mode when policy compliance is required.

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
│   ├── policy/          # filter by length + required character classes
│   ├── rules/           # hashcat .rule emitter + validating interpreter
│   ├── dedup/           # exact + Bloom-filter de-duplication
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
- [x] **Phase 5 — Policy filter.**
- [x] **Phase 6 — Scale hardening:** scalable dedup, worker pool, keyspace stats.
- [ ] **Phase 7 — (Optional) crawler:** CeWL-style keyword harvesting (isolated).

## Contributing

Contributions are welcome — please read [CONTRIBUTING.md](./CONTRIBUTING.md)
first. All participants are expected to follow our
[Code of Conduct](./CODE_OF_CONDUCT.md). Changes are validated by CI (gofmt,
`go vet`, build, `go test -race`, and a `go mod tidy` check).

## Security

Found a vulnerability in `pwprofiler` itself? Please report it privately per
[SECURITY.md](./SECURITY.md) rather than opening a public issue. And remember:
this tool is for **authorized testing only**.

## License

Licensed under the [Apache License 2.0](./LICENSE).

