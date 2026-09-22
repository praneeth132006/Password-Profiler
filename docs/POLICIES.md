# Password policies and verification

Password Profiler models an existing policy for an authorized audit. Passing a
policy is not a password-strength score or a recommendation to use a generated
password. Generated candidates are deliberately predictable.

## Reusable policy file

A standalone policy uses the fields directly, without a `policy:` wrapper:

```yaml
min_len: 8
max_len: 64
require: [upper, lower, digit, special]
max_bytes: 0
max_repeat: 3
forbidden: "<>"
blocklist:
  - "Password123!"
  - "Northstar123!"
```

Use the checked-in example:

```sh
pwprofiler files --input words.txt --policy testdata/policy.yaml --output passwords.txt
pwprofiler check --input passwords.txt --policy testdata/policy.yaml --fail-on-reject
```

`files --policy` replaces all policy defaults and cannot be combined with
individual policy flags. Omitted fields impose no constraint. YAML generation
configs use the same fields inside their `policy:` block. Unknown fields,
negative limits, reversed length bounds, and extra YAML documents are rejected.
`files`/console/browser sessions support a maximum length setting of 0–128;
0 disables that maximum. YAML generation and `check` support larger limits.

| Field | Meaning |
| --- | --- |
| `min_len` / `max_len` | Unicode code-point count, not byte count. 0 disables the bound. |
| `require` | Any of `upper`, `lower`, `digit`, `special`; an empty list requires none. |
| `max_bytes` | Maximum UTF-8 encoded bytes; 0 disables. Values are rejected, never truncated. |
| `max_repeat` | Maximum run of the same consecutive Unicode code point; 0 disables. |
| `forbidden` | Each character in this string is disallowed anywhere. |
| `blocklist` | Exact, case-sensitive whole-password matches, not substring matching. |

Upper/lower/digit classification uses Go's Unicode character tables. Special
characters are Unicode punctuation or symbols, excluding whitespace. The tool
does not normalize Unicode; canonically equivalent spellings remain distinct.
Input token generation can transform source text, whereas `check` examines each
wordlist line literally. Do not conflate the two workflows.

The browser exposes these controls under **Advanced policy**. Its blocklist
textarea preserves leading/trailing spaces, ignores empty lines, and uses one
blocked password per line. It accepts neither YAML nor special comment syntax.

## Stream an existing wordlist

```sh
pwprofiler check -i passwords.txt --policy policy.yaml --report report.json
cat passwords.txt | pwprofiler check --policy policy.yaml --fail-on-reject
```

`check` reads one line at a time and stores aggregate counters, not the entire
wordlist. It preserves spaces, `#`, and empty candidate lines. LF/CRLF line
endings are stripped. Invalid UTF-8 is rejected. Lines must be smaller than
1 MiB. An empty file is an error; an empty line is a candidate.

The JSON report contains only `total`, `accepted`, `rejected`, and
`first_failure_counts`. It does not contain source paths or password samples.
Every rejected candidate is counted under its first failure, in this order:
invalid UTF-8, blocklist, byte limit, forbidden character/repeat limit (encounter
order), character length, then missing upper/lower/digit/special classes.
Counts are not an exhaustive count of every failure on every candidate.

By default, violations are reported without a failing exit status. Add
`--fail-on-reject` for CI: a report is still written, followed by a nonzero exit
when any candidate fails. Parse/read/write failures always return nonzero.
Reports refuse to overwrite existing files and use owner-only Unix permissions.

## Validate before generating

```sh
pwprofiler validate --config testdata/sample.yaml
```

This checks schema, values, usable tokens, and rules-mode policy compatibility.
It does not generate candidates or open output files. It does not promise that
mutations will produce matches or that the output directory is writable.

## Save and restore console work

```text
pwprofiler > add keywords /path/words.txt
pwprofiler > policy /path/policy.yaml
pwprofiler > save /path/audit.session.json
pwprofiler > reset
pwprofiler > load /path/audit.session.json
pwprofiler > show options
pwprofiler > set output /path/new-passwords.txt
pwprofiler > run
```

Session schema version 1 stores the loaded input text, source labels, policy,
budget and output path. It does not need the original source files to restore.
The session is plaintext, not encrypted; keep it private. Saving refuses to
overwrite an existing file and uses mode 0600 on Unix. Loading validates the
schema, input limits and configuration before replacing the current session.
Session/policy files are capped at 4 MiB. Save does not run generation. The
built-in suffix containing the current year is evaluated at generation time.

## Research behind these choices

Reviewed September 22, 2026:

- [NIST SP 800-63B-4, Passwords](https://pages.nist.gov/800-63-4/sp800-63b.html#passwordver)
  discusses code-point length and checking the entire password against a
  blocklist. This informed the distinction between length limits and exact
  blocklist checks.
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html#implement-proper-password-strength-controls)
  recommends supporting long passwords and Unicode and avoiding silent
  truncation. Byte limits here are explicit compatibility constraints for an
  existing target, not default password-design advice.

Modern password guidance does not generally recommend mandatory composition
rules. Class, forbidden-character and repeat controls model existing systems;
they are not recommendations or claims of NIST/OWASP compliance. No online
breach lookup, account login, or external data submission is performed.
