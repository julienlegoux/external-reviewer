---
type: Issue
title: "Parse the review invocation and exit 2 on usage errors"
description: "Add the review/help subcommand grammar over stdlib flag, taking the repository path positionally and the task prompt from --prompt or stdin, with every usage error exiting 2 and leaving stdout empty."
tags: [epic-1]
timestamp: 2026-08-09T07:50:00Z
epic: 1
issue: 02
slug: review-invocation-parsing
size: M
status: open
gh_issue: 5
resource: https://github.com/julienlegoux/external-reviewer/issues/5
depends_on: [1]
---

# Parse the review invocation and exit 2 on usage errors

## Summary

An invocation has to be self-contained and reproducible from a transcript, which makes
the command line a contract with a calling script rather than a convenience. This PR
puts the epic's subset of that grammar in place — `review` with a positional repository
path and a task prompt from `--prompt` or stdin — and makes every malformed invocation
exit `2` with an empty stdout, so the caller's failure branch is settled before anything
talks to a model.

## Scope

- One `flag.FlagSet` per subcommand, stdlib only, long-form flags only, as SPECS fixes.
- `review [--prompt <text>] <repo-path>`: repository path positional; the task prompt
  from `--prompt`, or from stdin when the flag is absent.
- `help`, `--help`, `-h`: usage text on **stdout**, exit `0`. Help is not a failure path,
  so the "stdout is empty on failure" rule does not apply to it.
- Unknown subcommand, unknown flag, missing repository path, extra positional arguments,
  a repository path that does not exist or is not a directory, and an empty prompt (no
  `--prompt` and empty stdin) are all usage errors: exit `2`, stdout empty, reason on
  stderr.
- Parsing errors from `FlagSet` are routed to the injected stderr writer, never to
  `os.Stderr` directly, so `Run` stays testable in-process (`SetOutput`, and
  `ContinueOnError` rather than `ExitOnError` — the latter would call `os.Exit` from
  inside a test).

## Out of scope

- `--allow` (Epic 2), `--tier` / `--model` / `--exclude-family`, `models`, `tiers` and
  `version` (Epic 3).
- The JSON request object on stdin (`{"system": …, "task": …}`) and `--system`. The
  skeleton reads stdin as the raw task prompt; the structured form and the
  caller-supplied system prompt arrive with the epics that need them.
- The exit-code taxonomy for runs that reach a model, the `done` line, and signal
  handling — issue 03. This PR only needs usage errors to return `2`.
- Any use of the repository path beyond checking that it is an existing directory;
  confinement via `os.OpenRoot` is Epic 2.

## Acceptance criteria / Definition of done

- [ ] Table-driven black-box tests in `package cli_test` drive every case below through
      `Run(argv, stdin, stdout, stderr)`, written failing first:
  - [ ] `review --prompt "review this" <tempdir>` parses successfully and reaches the
        (still stubbed) run path.
  - [ ] `review <tempdir>` with `"review this"` on stdin parses identically.
  - [ ] `review --prompt "x"` with no positional path → exit `2`, stdout empty.
  - [ ] `review --prompt "x" <tempdir> extra` → exit `2`, stdout empty.
  - [ ] `review --prompt "x" <path-that-does-not-exist>` → exit `2`, stdout empty.
  - [ ] `review --prompt "x" <path-to-a-regular-file>` → exit `2`, stdout empty.
  - [ ] `review <tempdir>` with empty stdin → exit `2`, stdout empty.
  - [ ] `review --nope <tempdir>` → exit `2`, stdout empty.
  - [ ] `frobnicate` → exit `2`, stdout empty.
  - [ ] `help`, `--help` and `-h` each → exit `0` with non-empty stdout.
- [ ] Every exit-`2` case writes a reason to stderr naming what was wrong.
- [ ] No test causes the process to exit (no `ExitOnError` anywhere).
- [ ] Repository paths are handled with `path/filepath` as OS paths, per the
      OS-path/wire-path split in CONVENTIONS; `t.TempDir()` fixtures make the tests pass
      unchanged on both CI platforms.
- [ ] `go test ./... -race` and `golangci-lint run` pass; CI green on both OSes.

## Relevant files / areas

No existing code for this yet — issue 01 creates `internal/cli`. Expected paths:

- `internal/cli/run.go` (subcommand dispatch), `internal/cli/review.go` (the `review`
  FlagSet and its parsed request struct), `internal/cli/usage.go`
- `internal/cli/review_test.go` (`package cli_test`)

Governing decisions: [SPECS § Interfaces](../../../planning/SPECS.md) (the command-line
contract and long-form-only flags),
[CONVENTIONS § Paths and platforms](../../../planning/CONVENTIONS.md),
[CONVENTIONS § Testing](../../../planning/CONVENTIONS.md) (black-box, table-driven).

## Dependencies

- Blocked by: [01 — Scaffold the Go module, the run() seam and CI](/epic-1-walking-skeleton/issues/01-scaffold-module-and-ci.md).
- Blocks: [03 — Emit run diagnostics on stderr and classify exit codes](/epic-1-walking-skeleton/issues/03-diagnostics-and-exit-codes.md).

## PR size note

Sized **M**: target ~350 changed lines, of which the table of usage cases is the larger
half. The grammar and its usage errors are one contract and do not split; if this passes
~700, the exit-code taxonomy from issue 03 has been pulled forward.
