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

## Supported versions

This project is pre-1.0 and under active development. Fixes are applied to the
`main` branch; there are no long-term support branches yet.

| Version | Supported |
| ------- | --------- |
| `main`  | ✅        |
| tagged pre-releases | best-effort |
