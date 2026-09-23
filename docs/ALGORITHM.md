# Generation algorithm and evaluation

Version 0.4.0 replaces eager combination and mutation materialization with a
consumer-driven pipeline. It improves resource usage and fixes a coverage bug;
it does not introduce a trained model or claim improved real-world guessing rates.

## Pipeline

1. Extract unique, sorted base tokens from the supplied profile.
2. Yield singles and ordered combinations of distinct token indices lazily.
3. For each token, explore unique mutation states in breadth-first order. Emit
   case seeds first, followed by leet, append, prepend and structural transforms.
4. Check each candidate against the policy; rejected candidates can still be
   explored, because a later transform may make them valid.
5. Deduplicate accepted candidates and write them until the output budget is met.
   Consumer stop and cancellation propagate into the producer.

The old algorithm expanded a complete token tree before checking the output
budget. Full leetspeak could materialize a combinatorial number of strings even
when only a few output lines were wanted. It now enumerates substitutions lazily
and stops inside the traversal. The position cap for full leet remains three
by default; this is distinct from the overall work limit.

Combination traversal visits exact-length paths in lexicographic input-index
order. This preserves the previous separator-first, length-first ordering
without retaining breadth levels or preallocating `len(level) * len(tokens)`
objects. Oversized joins are detected before allocating the joined string.

One worker preserves deterministic output order. Multiple workers send batches
of at most 64 accepted candidates through a bounded channel to a single writer.
Their output order, and the selected subset when an output cap applies, can
vary. Workers cancel ongoing search when the output cap is reached. Ctrl+C
cancels wordlist generation, and failed file generations remove their new output.

## Independent output and search budgets

Previously, an output budget of 1 also limited the combination stage to its
first input token. If that token could not satisfy the policy, later valid input
was never tried. Search now has its own limits, so a small output budget can
still examine later tokens.

Advanced generation YAML accepts:

```yaml
rules:
  max_combine: 2
  depth: 3
  combine_limit: 100000
  combine_attempts: 1000000
  max_variants: 10000
  max_attempts: 100000
  max_candidate_bytes: 4096
output:
  budget: 10000
```

| Setting | Scope and default |
| --- | --- |
| `combine_limit` | At most 100,000 unique combined tokens per run. |
| `combine_attempts` | At most 1,000,000 complete paths/singles attempted per run, including duplicates and oversized results. |
| `max_variants` | At most 10,000 unique retained mutation states per combined token, including seeds. |
| `max_attempts` | At most 100,000 attempted mutations/seeds per combined token, including duplicates and oversized results. |
| `max_candidate_bytes` | Maximum size of a combined token or mutation state: 4,096 UTF-8 bytes. Oversized states are skipped, never truncated to fit. |
| `max_combine` / `depth` | Supported range 1–8. Existing defaults remain 1 when omitted. |

Omitted or zero search-limit fields select defaults; negative values are
rejected. Search limits are not password-policy constraints. Skipping an
oversized intermediate can also omit descendants that would later shrink.
The CLI reports incomplete coverage when a search cap is encountered. The
browser exposes the same condition through `X-Search-Limited` and a partial
wordlist notice. A normal output-budget stop has its own message. If all search
limits and the output budget are avoided, the configured finite traversal is
exhausted; this is not exhaustion of all possible passwords.

The file CLI, console and browser use these default limits. For custom limits,
use the YAML `generate` workflow. `--workers` accepts 1–64. Memory is bounded
relative to the configured state limits, token lengths and worker count; raising
them increases resource use. Exact global dedup also scales with emitted output.
Bloom dedup remains approximate and can omit candidates through false positives.

Rules mode uses the combination limits, but emits mutation instructions rather
than executing the mutation tree, so per-token mutation limits do not constrain
an external rule consumer. Rules mode still cannot enforce a password policy.

## Compatibility and completeness tests

- Existing transform/combination order fixtures continue to pass.
- Small-tree mutation output is compared to an independent breadth-level reference.
- Full-leet streaming order is checked against collected reference results.
- Fuzz tests exercise ordering/coverage equivalence for bounded partial-leet trees.
- Cancellation, consumer stops, work/state/byte caps and incomplete-search notices
  have regression tests.
- Worker counts are checked for equal candidate sets when budgets are not reached.
- A regression test proves a low output cap no longer removes later valid inputs.
- A policy test verifies that rejected intermediates remain eligible for mutation.

## Measured performance

Local microbenchmarks on Apple M4, darwin/arm64, Go 1.27.1. Values below are medians
of three runs, compared with commit `0461dde` using identical benchmark fixtures.
Raw results: [before](benchmarks/algorithm-before.txt),
[after](benchmarks/algorithm-after.txt).

| Workload | Before | After | Allocated bytes before → after |
| --- | --- | --- | --- |
| Emit 50 policy-matching candidates, full leet and depth 3 | 41.92 ms | 0.260 ms | 59.04 MB → 0.281 MB |
| 500 input tokens, combination limit 510 | 0.236 ms | 0.0231 ms | 10.10 MB → 0.0738 MB |

The first fixture is approximately 161× faster with 99.5% fewer allocated bytes;
the second is approximately 10× faster with 99.3% fewer allocated bytes. These
are favorable small-budget workloads, not universal speedups, peak-memory
measurements, or password-recovery success rates. Larger budgets and restrictive
policies require more work. The new limits can deliberately reduce coverage on
large trees; compare uncapped fixtures when studying algorithm equivalence.

Reproduce:

```sh
go test ./cmd/pwprofiler ./internal/combine -run '^$' \
  -bench 'Benchmark(GenerationBudget|SmallLimit)$' -benchmem -benchtime=200ms -count=3
go test ./internal/mutate -run '^$' -fuzz '^FuzzWalkMatchesReference$' -fuzztime=3s
go test -race ./...
```

For a baseline comparison, use an isolated checkout of `0461dde` and copy only
`cmd/pwprofiler/generation_bench_test.go` and `internal/combine/bench_test.go`
into it before running the same benchmark command.

## Research and implementation decisions

Reviewed September 22–23, 2026:

- [Weir et al., Password Cracking Using Probabilistic Context-Free Grammars, IEEE S&P 2009](https://conferences.computer.org/sp/pdfs/sp/2009/oakland2009-23.pdf)
  learns structures from training data and evaluates guesses against separate
  test data. It supports treating candidate ordering as an empirical question,
  rather than assigning invented probability scores to handcrafted transforms.
- [Melicher et al., Fast, Lean, and Accurate: Modeling Password Guessability Using Neural Networks](https://www.usenix.org/conference/atc17/technical-sessions/presentation/melicher)
  explores learned guessability models. Such a model would add data, training,
  runtime and evaluation requirements that this self-contained tool does not have.
- [Hashcat rule-based attack documentation](https://hashcat.net/wiki/doku.php?id=rule_based_attack)
  describes composable transformations. The implementation retains explicit,
  testable rules and optimizes their execution instead of adding opaque scoring.

These sources inform the design; this release does not reproduce their models
or inherit their reported success rates. A future ranking experiment should
use permissioned training data, a disjoint held-out test set, and coverage at
fixed candidate budgets, alongside runtime and memory measurements. No password
corpus was downloaded or uploaded for this work.
