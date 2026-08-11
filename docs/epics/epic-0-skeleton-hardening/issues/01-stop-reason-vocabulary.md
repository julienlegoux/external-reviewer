---
type: Issue
title: "Settle the done line's stop-reason vocabulary and document it in SPECS"
description: "Render the model's own stop reason on the success path and the CLI word on every termination the model never reaches, and document the closed set in SPECS § Interfaces."
tags: [epic-0]
timestamp: 2026-08-11T04:00:00Z
epic: 0
issue: 01
slug: stop-reason-vocabulary
size: S
status: done
gh_issue: 23
gh_pr: 37
resource: https://github.com/julienlegoux/external-reviewer/issues/23
depends_on: []
---

# Settle the done line's stop-reason vocabulary and document it in SPECS

## Summary

Two concepts compete for one field. `ai.AssistantMessage.StopReason` is read at
`internal/reviewer/run.go:94` only to detect `error`/`aborted` and is then discarded —
`reviewer.Turn` does not expose it, though it carries `Message`. `diag.State.StopReason`
gets a hand-written CLI vocabulary instead (`ok`, `failed`, `usage`, `help`,
`no_reviewer`, `interrupted`), so the binary prints `stop=ok` where SPECS' own worked
example prints `stop=end_turn`, and the two packages' tests encode the disagreement:
`internal/diag/diag_test.go:44,73` asserts on `"end_turn"` while every `internal/cli`
test asserts on `"ok"`.

This was decided at triage rather than left to the implementer: **the field carries both
vocabularies**, because they answer different questions — *why the model stopped* and
*how the run terminated* — and a usage error can never carry `end_turn`. The success path
renders the model's own reason; every termination the model never reaches keeps the CLI
word. That makes SPECS' example literally true and Epic 2's first acceptance criterion
("the run ending on the model's own stop reason") satisfiable without amending either
document.

It lands first because it changes the contract three later issues in this epic are judged
against.

## Scope

- `reviewer.Turn` exposes the assistant message's stop reason. It already carries
  `Message`, so this surfaces a value rather than tracking new state
  (`internal/reviewer/run.go:16-25,84`).
- `internal/cli/review.go:219` — `state.StopReason = "ok"` becomes the model's own reason,
  spelled as `kern-link` spells it, with no translation table between the two.
- The five terminations the model never reaches keep their CLI word unchanged: `usage`,
  `help`, `no_reviewer`, `interrupted`, `failed`.
- A named fallback for a successful turn whose `StopReason` is empty, so `stop=` can never
  render with nothing after it.
- `docs/planning/SPECS.md` § Interfaces documents the **closed set** — both halves — and
  states that it is a union that grows per epic (Epic 2 adds at least `bounds`), because a
  calling skill cannot parse a set that is written nowhere.
- The same SPECS pass documents the `model   <provider>/<id>  auth=<source>` line
  (`internal/diag/diag.go:54-56`), which ships today outside the documented stderr
  vocabulary — issue 03 of Epic 1 set the precedent that a change to the transcript shape
  is amended into SPECS by the same PR, and issue 04 did not follow it.
- The `internal/cli` tests that assert `stop=ok` on the success path move to the model's
  reason, so no two packages assert two vocabularies for one field.

## Out of scope

- Epic 2's `bounds` stop reason. SPECS says the set grows; this PR does not add a value
  no code can produce.
- Renaming `diag.State.StopReason` or changing the `done` line's shape — only what the
  field carries changes.
- `EPIC_2.md`'s acceptance criterion 1. It is satisfied by this decision as written; no
  amendment is needed and none is made here.
- The `error:` line and the scattered "was this interrupted?" decisions that assign this
  same field — issue 07.

## Acceptance criteria / Definition of done

- [ ] Written test-first, black-box (`package cli_test` / `package diag_test`),
      table-driven where the cases share a shape.
- [ ] A successful scripted `faux` review's `done` line reads `stop=end_turn` (the
      scripted message's own reason), asserted through `Run` — not `stop=ok`.
- [ ] Each of these still renders its CLI word, asserted through `Run`: a usage error →
      `stop=usage`; `help` → `stop=help`; `ErrNoReviewer` → `stop=no_reviewer`; a cancelled
      context → `stop=interrupted`; a failed turn → `stop=failed`.
- [ ] `grep -rn 'stop=ok' internal docs` returns nothing — the two packages no longer
      assert two vocabularies for the same field.
- [ ] A scripted successful message with an empty `StopReason` renders the named fallback
      rather than `stop=` followed by nothing; a test pins it.
- [ ] `docs/planning/SPECS.md` § Interfaces lists the closed stop-reason set, says it is a
      union that grows per epic, and shows the `model … auth=` line beside the `turn`,
      `tool`, `warn` and `done` lines.
- [ ] `go test ./... -race` and `golangci-lint run` pass; CI green on ubuntu-latest and
      windows-latest.

## Relevant files / areas

- `internal/reviewer/run.go:16-25` (`Turn`), `:84` (turn construction), `:94` (where the
  reason is read and discarded today)
- `internal/cli/review.go:215-230` (`runReview`'s switch), `:219` (`stop=ok`)
- `internal/diag/diag.go:54-56` (`WriteModel`), `:73-76` (`WriteDone`)
- `internal/diag/diag_test.go:44,73`; `internal/cli/exit_test.go`,
  `internal/cli/preflight_test.go`, `internal/cli/roundtrip_test.go`
- `docs/planning/SPECS.md` § Interfaces

Governing decisions: [SPECS § Interfaces](../../../planning/SPECS.md),
[CONVENTIONS § Error handling](../../../planning/CONVENTIONS.md). Source finding:
[report 2](../../../REPORT_2.md) § "`stop=ok` is Epic 1's invention" and Open questions 1
and 2.

## Dependencies

- Blocked by: none.
- Blocks: [07 — One prefixed `error:` line](/epic-0-skeleton-hardening/issues/07-single-error-line.md)
  (which consolidates the sites that assign this field) and
  [11 — Turn hardening](/epic-0-skeleton-hardening/issues/11-turn-hardening.md).

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
Sized **S**: a field on `Turn`, one assignment in `cli`, a SPECS section, and the test
churn of moving the success-path assertions from one vocabulary to the other.
