# Security Policy

## Acceptable use

`pwprofiler` is a tool for **authorized** password auditing and penetration
testing. By using it you agree to use it only against systems and accounts you
own or for which you hold explicit, written permission to test. Using this
project to attack systems without authorization is illegal and is not
supported, endorsed, or assisted by this project or its maintainers.

## Reporting a vulnerability

If you discover a security vulnerability in `pwprofiler` itself (for example, a
crash on untrusted config input, a path-traversal in output handling, or a
supply-chain issue), please report it **privately** — do not open a public
issue for it.

Preferred: use GitHub's private vulnerability reporting on this repository —
open the **Security** tab and choose **"Report a vulnerability"**. (Maintainers:
enable this under *Settings → Code security and analysis → Private vulnerability
reporting*.)

When reporting, please include:

- a description of the issue and its impact,
- steps to reproduce (a minimal config or input is ideal),
- the version / commit you tested, and your OS and Go version.

Please give maintainers a reasonable window to investigate and release a fix
before any public disclosure.

## In scope

- Crashes, panics, or unbounded resource use triggered by untrusted **config,
  policy, or input files**.
- Path traversal, symlink, or overwrite issues in output handling.
- Weaknesses in the local web server's isolation (Host/Origin checks, CSP,
  same-origin enforcement) that allow another local origin or a remote page to
  drive it.
- Supply-chain issues (dependencies, build, or release artifacts).

## Out of scope

- The tool doing what it is designed to do (generating candidate passwords) —
  that is the feature, used under the authorized-use terms above.
- Findings that require the operator to supply already-hostile input to their
  own local run, with no crossing of a trust boundary.

## Hardening already in place

- The core generation pipeline makes **no network calls**.
- `pwprofiler serve` binds only to `127.0.0.1`, verifies `Host`/`Origin`, sets a
  strict Content-Security-Policy, requires a same-origin header for generation,
  processes one request at a time, and keeps uploads and results in memory.
- File output uses `O_EXCL` (refuses to overwrite) with owner-only (`0600`)
  permissions on Unix; failed/cancelled runs remove partial output.

## Supported versions

This project is pre-1.0 and under active development. Fixes are applied to the
`main` branch; there are no long-term support branches yet.

| Version | Supported |
| ------- | --------- |
| `main`  | ✅        |
| tagged pre-releases | best-effort |
