---
type: Decision
title: "Language, module & toolchain"
description: "Which Go version, module path and top-level program shape the binary is built as."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 01
slug: language-module-toolchain
status: decided
verdict: "Go 1.26, module github.com/julienlegoux/external-reviewer, root package main plus internal/"
decided_via: triage
depends_on: []
---

# Question

Go is fixed by [scope](/scope/15-constraints.md) — `kern-link` carries the provider
coverage this project refuses to rebuild, so the language follows from the dependency.
What is still open is the *version*, the *module path*, and where `package main` lives.

The last part is load-bearing rather than cosmetic: SCOPE names
`go install github.com/julienlegoux/external-reviewer@latest` as the entire install
story ([decision](/scope/02-target-users.md)). That command only produces a binary
called `external-reviewer` if the module root itself is `package main`. Putting the
program under `cmd/external-reviewer/` changes the published install command, and the
install command is quoted in SCOPE and will be quoted again in `review-interfaces.md`.

# Options

- **Root `package main`, `internal/` for everything else, Go 1.26.** `go install
  <module>@latest` works verbatim; nothing importable escapes `internal/`.
- **`cmd/external-reviewer/` layout, Go 1.26.** Conventional for repos shipping several
  binaries or a public library; costs a longer install path this project has no use for.
- **Match `kern-link`'s floor, Go 1.25.0.** Maximum toolchain portability, at the price
  of not using anything added in 1.26 — including the `os.Root` maturity that
  [confinement](/specs/10-repository-confinement.md) leans on.

# Recommendation

**Root `package main` + `internal/`, `go 1.26` in `go.mod`, module
`github.com/julienlegoux/external-reviewer`.**

The root layout is chosen by the install story, not by taste — this repo ships exactly
one binary and no importable library, which is the case the root layout exists for.
`main.go` stays a thin `os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))`, the
same shape `kern-link`'s `cmd/pi-ai` uses to keep its CLI testable end to end without a
process.

Go 1.26 over 1.25.0 because the toolchain on the one machine that matters is already
`go1.26.3`, CI pins from `go-version-file: go.mod`, and `kern-link`'s `go 1.25.0` is a
*floor* a consumer is free to exceed. Nothing is gained by declaring a version older
than the only compiler that will build this.

# Verdict

Accepted at triage as recommended. The root layout is chosen by the install command SCOPE commits to; `main.go` stays a thin `os.Exit(run(...))` so the whole CLI is testable in-process.
