---
type: Issue
title: "Emit run diagnostics on stderr and classify exit codes"
description: "Add the prefixed stderr diagnostics writer with a done line on every termination path, the typed-error exit-code classification at the Run boundary, and SIGINT cancellation."
tags: [epic-1]
timestamp: 2026-08-09T05:24:00Z
epic: 1
issue: 03
slug: diagnostics-and-exit-codes
size: M
status: open
gh_issue: 6
resource: https://github.com/julienlegoux/external-reviewer/issues/6
depends_on: [2]
---

# Emit run diagnostics on stderr and classify exit codes

## Summary

The output contract is what every later epic and the calling skill depend on, and it is
easier to build before there is a model call than to retrofit around one. This PR adds
the stderr diagnostics writer, guarantees the `done` line on **every** termination path
including failure and interruption, and puts the `0` / `1` / `2` classification behind
typed errors inspected at the `Run` boundary — never message matching.

## Scope

- An internal diagnostics writer producing the prefixed, human-readable lines SPECS
  fixes, no colour and no spinner: `turn`, `tool`, `warn`, `done`.
- The `done` line — `done    turns=7  $0.0918  2m14s  stop=end_turn` — emitted on every
  termination path: success, no reviewer available, failure, and interruption. Its
  load-bearing fields are turns, tokens and elapsed time.
- Run state (turn count, accumulated tokens and cost, start time, stop reason) carried in
  one struct that the `done` line is rendered from, so no path can terminate without one.
- The exit-code taxonomy as sentinel/typed errors in an internal package, classified with
  `errors.Is` / `errors.As` at the `Run` boundary: `0` usable, `1` no reviewer ever
  reached, `2` reached and unusable — including usage errors, which issue 02 already
  returns as `2`.
- `signal.NotifyContext` for `SIGINT` wired in `Run`, over an inner
  `run(ctx, …) (int, error)` that takes the context explicitly. Interruption terminates
  the run at exit `2` with a reason on stderr and an empty stdout.
- Nothing panics across the CLI boundary; `recover()` appears nowhere.

## Out of scope

- Anything that actually produces a turn, a tool line or a cost — issues 04 and 05 fill
  the state this PR renders. Tests here drive the writer directly and through stubbed
  terminations.
- Loop bounds (`bounds.check`) — Epic 2.
- Real token and cost accounting from `kern-link` — issue 05 wires it into the same
  struct.

## Acceptance criteria / Definition of done

- [ ] Written test-first, black-box (`package cli_test` / `package diag_test`),
      table-driven where the cases share a shape.
- [ ] A `done` line appears on stderr for each of: an exit-`0` termination, an exit-`1`
      termination, an exit-`2` failure, and a cancelled context — asserted through `Run`
      (or the inner `run`) for all four, not by calling the writer directly.
- [ ] The `done` line contains the turn count, the elapsed time and the stop reason, in
      the `key=value` shape SPECS shows; a test asserts on the parsed fields rather than
      on an exact string where timing is involved.
- [ ] stdout is empty on every non-zero exit path exercised here.
- [ ] Classification is by `errors.Is` / `errors.As`: a test wraps a sentinel with
      `fmt.Errorf("…: %w", err)` two levels deep and asserts the exit code is unchanged.
      No test and no production path compares error text.
- [ ] A cancelled context passed to the inner `run` returns exit `2` and writes an
      interruption reason to stderr. `signal.NotifyContext` wiring itself is exercised
      only through `Run`'s construction — delivering a real `SIGINT` is not portable to
      the Windows CI runner, and this is noted in a comment beside the wiring.
- [ ] Error messages are lowercase and unpunctuated and state what was being attempted,
      per CONVENTIONS.
- [ ] `go test ./... -race` and `golangci-lint run` pass; CI green on both OSes.

## Relevant files / areas

No existing code for this yet. Expected paths:

- `internal/diag/diag.go` (prefixed writer + run state + `done` rendering),
  `internal/diag/diag_test.go`
- `internal/cli/run.go` (signal wiring, inner `run(ctx, …)`, classification switch),
  `internal/cli/exit.go` (sentinel/typed errors), `internal/cli/exit_test.go`

Governing decisions: [SPECS § Interfaces](../../../planning/SPECS.md) (output,
diagnostics format, exit codes), [CONVENTIONS § Error handling](../../../planning/CONVENTIONS.md).

## Dependencies

- Blocked by: [02 — Parse the review invocation](/epic-1-walking-skeleton/issues/02-review-invocation-parsing.md).
- Blocks: [04 — Resolve the hard-coded reviewer model and preflight auth](/epic-1-walking-skeleton/issues/04-model-resolution-and-auth.md).

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
