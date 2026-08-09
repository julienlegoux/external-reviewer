---
type: Epic
title: "Walking skeleton"
description: "A binary that takes a repository path and a prompt, calls one hard-coded foreign model through kern-link, and returns markdown — retiring the riskiest assumption in the project."
tags: [epic]
timestamp: 2026-08-09T14:30:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/1
epic: 1
slug: walking-skeleton
status: done
gh_issue: 1
milestone: 1
source: docs/planning/SCOPE.md#milestone-1-walking-skeleton
---

# Epic 1: Walking skeleton

## Goal

Retire the riskiest single assumption in the project — that `kern-link` carries this
workload at all — by getting one real answer back from a foreign model through it.

What it unlocks is larger than the demo: this epic establishes the process shell every
later epic plugs into. The `run(argv, stdin, stdout, stderr) int` seam that makes the
whole CLI testable in-process, the stdout/stderr split, and the exit-code contract are
all decided here, and Epics 2 and 3 add capability inside them rather than reshaping
them.

## Scope

- A CLI entry point accepting a repository path and a task prompt (argument or stdin),
  so an invocation is self-contained and reproducible from a transcript.
- `main.go` as a thin `os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))`, with
  everything else under `internal/` — the layout the `go install` story requires.
- A single round-trip call to a foreign model through `kern-link`, using its existing
  credential resolution (env keys and OAuth). This binary reads no credential, stores
  none and refreshes none.
- **The output contract in full**, since everything downstream depends on it:
  - the report as markdown on **stdout**, verbatim and with no envelope, empty on any
    failure — the calling skill keeps sole ownership of the report file;
  - run diagnostics on **stderr** as prefixed human-readable lines, with a `done` line
    emitted on **every** termination path including failure and interruption;
  - exit codes `0` usable, `1` no reviewer available, `2` ran but unusable.
- The binary writes no files.
- The repository's baseline scaffolding: `go.mod` at Go 1.26 naming a tagged
  `kern-link` (never a committed `replace`), the CI workflow with `test` and `lint`
  jobs on ubuntu-latest and windows-latest, and the `forbidigo` guard that keeps the
  write half of the filesystem API out of the codebase.

## Out of scope

- The multi-turn loop and the read-only tools — Epic 2.
- Tier vocabulary, the config file, family exclusion and the discovery commands —
  Epic 3. The model is hard-coded here.
- Any write to the repository or to GitHub, any interactive UI, and any sharing of the
  calling session's context. Project-wide non-goals, not this epic's boundary.

## Acceptance criteria

- Invoked with a repository path and a prompt, the binary returns the model's markdown
  on stdout and exits `0`.
- With no reviewer reachable — provider unconfigured, credential absent — stdout is
  empty and the exit code is `1`, the silent-fallback case the caller needs no line
  about.
- Reached and then failed — a failed turn, an empty final message, a broken credential
  (`*ai.ModelsError` with code `oauth`/`auth`), `SIGINT`, or a usage error — stdout is
  empty and the exit code is `2`, with the reason on stderr.
- The `done` line appears on all three paths.
- The binary creates, modifies or deletes no file anywhere.
- `go test ./... -race` passes offline on a clean checkout, with the loop exercised
  against `kern-link`'s `faux` provider rather than the network.

## Dependencies

None — this is the first epic.

## Context

- [Technical specs](../../planning/SPECS.md) — the stack, the `run()` seam, the output
  and diagnostics format, and the exit-code classification.
- [Conventions](../../planning/CONVENTIONS.md) — repo standards, including the CI lint
  guard that this epic sets up.

## Notes

- **This epic ships a deliberate subset of the final CLI grammar.** SPECS specifies
  `review --allow <relpath> …` with a JSON request object (`system` + `task`) on stdin;
  the allow-list arrives with confinement in Epic 2 and the tier flags in Epic 3. The
  skeleton's simpler argument shape is a stage, not a contradiction of SPECS — but the
  stdout/stderr/exit-code contract above is final from here on.
- The `1`-versus-`2` distinction is load-bearing and easy to get subtly wrong: an
  *absent* credential is `1`, a *broken* one is `2`, because the second means something
  a human must fix.
- **Project-wide, relevant here**: Windows is the development machine and nothing may
  assume POSIX; the binary runs on the user's own credentials; distribution is
  `go install …@latest` with no release machinery, which is why the module root is
  `package main`.
- Retires the first half of SCOPE's risk 1 (`kern-link`'s multi-turn tool loop is
  unproven); Epic 2 retires the rest. Mitigated throughout by the author owning the
  upstream.
