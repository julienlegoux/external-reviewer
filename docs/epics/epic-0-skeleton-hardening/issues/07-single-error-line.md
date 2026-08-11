---
type: Issue
title: "Write one prefixed error: line from diag on every exit-2 path"
description: "Give diag the error line, route all eight hand-rolled sites through it including the silent empty-argv path, make review --help exit 0, collapse the three interruption checks, and assert the exit-1 path's silence."
tags: [epic-0]
timestamp: 2026-08-11T06:30:00Z
epic: 0
issue: 07
slug: single-error-line
size: M
status: pr-open
gh_issue: 29
gh_pr: 39
resource: https://github.com/julienlegoux/external-reviewer/issues/29
depends_on: [1]
---

# Write one prefixed error: line from diag on every exit-2 path

## Summary

One concept has four renderings and one path with nothing at all:

| Path | What stderr gets |
|---|---|
| `run.go:58-63` empty argv | usage text only — **no reason line at all** |
| `run.go:74-80` unknown command | `error: unknown command "x"` **+** usage text |
| `review.go:152-156` bad flag | `flag`'s own message + `Usage of review:` — no `error:` prefix, and the *subcommand's* usage |
| `review.go:160-206` other usage errors | `error: …`, no usage text |

`errors.New("no command given")` is created, classified and never printed; the only test on
it (`run_test.go:23-25`) asserts `stderr.Len() != 0`, which the leftover usage text
satisfies. `internal/diag` claims to implement *"the run's stderr diagnostics"* and issue 03
of Epic 1 scoped the vocabulary as `turn`/`tool`/`warn`/`done`, yet the most important line
a failing run emits is hand-rolled `fmt.Fprint*` at **eight sites across two files**, with
no `diag.WriteError` and no padding to the 8-column prefix width every `diag.Write*` uses.
Epic 2 adds refusal reasons, tool failures and bound-exceeded terminations to this same
surface.

Three more defects share the pass because they are the same code. `review --help` exits `2`
(`flag.ContinueOnError` returns `flag.ErrHelp`, which the handler classifies as
`stop=usage`) while top-level `help` correctly exits `0`. "Was this interrupted?" is decided
in three independent places, two of them inspecting `ctx.Err()` and one the returned error.
And the exit-`1` path's silence — *"the silent-fallback case the caller needs no line
about"*, the criterion itself — is untested: adding an `error:` print to the
`errors.Is(err, ErrNoReviewer)` branch leaves the whole suite green.

## Scope

- `diag.WriteError(w, message)` — the `error:` prefix padded to the same 8-column width
  every other `diag.Write*` uses.
- All eight hand-rolled sites routed through it: `internal/cli/run.go:53,75`;
  `internal/cli/review.go:160,165,173,178,196,203,224,227`.
- The empty-argv path (`internal/cli/run.go:58-63`) writes its reason instead of only
  usage text.
- `review --help` / `review -h` exit `0` with usage on stdout, matching top-level `help`:
  `flag.ErrHelp` is distinguished from a genuine flag parse error
  (`internal/cli/review.go:148,152`).
- No unprefixed line precedes the `done` line on a failing run — `flag`'s own output and
  `printUsage(stderr)` currently emit four (`internal/cli/review.go:148-153`,
  `internal/cli/run.go:59,76`).
- The three "was this interrupted?" decisions (`internal/cli/run.go:51`,
  `internal/cli/review.go:222`, `internal/reviewer/run.go:91`) collapsed to one, in
  `runReview`. The gap between them is reachable in principle: `Resolver.Resolve` makes
  network calls with `ctx` and kern-link wraps auth failures as `*ai.ModelsError`, so a
  `ModelsError` that did not wrap `context.Canceled` would print `stop=failed` for a run a
  human interrupted. (In v0.1.1 the cause *is* preserved, so it does not fire today.)
- A named constant for the `"usage"` stop reason — the literal appears at eleven sites.

## Out of scope

- Escaping provider-controlled text in the `error:` line —
  [issue 12](/epic-0-skeleton-hardening/issues/12-diag-escaping-and-polish.md), which
  lands on the writer this PR creates.
- The `stop=` vocabulary itself — [issue 01](/epic-0-skeleton-hardening/issues/01-stop-reason-vocabulary.md),
  already landed.
- How paths are formatted inside those messages —
  [issue 08](/epic-0-skeleton-hardening/issues/08-os-and-wire-paths.md).
- The prompt-validation messages' *content* —
  [issue 09](/epic-0-skeleton-hardening/issues/09-bounded-validated-prompt.md) changes when
  they fire, not how they are written.

## Acceptance criteria / Definition of done

- [ ] Written test-first, black-box, table-driven over the exit-`2` paths.
- [ ] Every exit-`2` path writes exactly **one** prefixed `error:` line naming what was
      wrong: empty argv, unknown command, bad flag, missing repository path, extra
      arguments, an unstattable path, a non-directory path, a stdin read failure, an empty
      prompt, a failed review, and an interrupted run.
- [ ] `Run(nil, …)` exits `2` and writes an `error:` line whose **text** the test asserts —
      not `stderr.Len() != 0`.
- [ ] No unprefixed line precedes the `done` line on any exit-`2` path, asserted by a test
      that scans stderr and rejects any line not opening with a known 8-column prefix.
- [ ] `review --help` and `review -h` exit `0` with usage on stdout; a genuine bad flag
      (`review --nope`) still exits `2` with one `error:` line.
- [ ] A test asserts the exit-`1` path writes **no** `error:` line at all — the missing half
      of the pair `TestRun_BrokenCredential_ExitsTwo` already writes. Adding
      `fmt.Fprintln(stderr, "error:", err)` to the `errors.Is(err, ErrNoReviewer)` branch
      turns it red, recorded in the PR body.
- [ ] "Was this interrupted?" is decided in one place: deleting either of the other two
      checks changes no observable behaviour, and a `*ai.ModelsError` wrapping
      `context.Canceled` still reports `stop=interrupted`.
- [ ] `grep -n '"usage"' internal/cli/*.go` shows the named constant rather than eleven
      literals.
- [ ] `go test ./... -race` and `golangci-lint run` pass; CI green on ubuntu-latest and
      windows-latest.

## Relevant files / areas

- `internal/diag/diag.go` (new `WriteError`, beside `WriteTurn`/`WriteTool`/`WriteWarn`/
  `WriteDone` at `:35-76`)
- `internal/cli/run.go:51,53,58-63,74-80`
- `internal/cli/review.go:148-156,160,165,173,178,196,203,215-230`
- `internal/cli/run_test.go:23-25`, `internal/cli/exit_test.go`,
  `internal/cli/review_test.go`, `internal/diag/diag_test.go`

Governing decisions: [SPECS § Interfaces](../../../planning/SPECS.md) (stderr as prefixed
lines), [CONVENTIONS § Error handling](../../../planning/CONVENTIONS.md) (lowercase,
unpunctuated, stating what was attempted; classification by type, never message text).
Source findings: [report 2](../../../REPORT_2.md) § "The `error:` reason line has four
shapes", § "The exit-1 path's 'needs no line' property is unasserted", § "'Was this
interrupted?' is decided in three independent places", and the P3 rows on unprefixed
lines and `review --help`.

## Dependencies

- Blocked by: [01 — Stop-reason vocabulary](/epic-0-skeleton-hardening/issues/01-stop-reason-vocabulary.md)
  — this PR consolidates the sites that assign `state.StopReason`, so the field's contract
  is settled first.
- Blocks: [08](/epic-0-skeleton-hardening/issues/08-os-and-wire-paths.md),
  [10](/epic-0-skeleton-hardening/issues/10-stdout-write-failures.md) and
  [12](/epic-0-skeleton-hardening/issues/12-diag-escaping-and-polish.md), each of which
  writes through `diag.WriteError`.

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
Sized **M**: ~350 lines — one writer, eight call sites, the `flag.ErrHelp` split, the
interruption collapse, and the exit-`1`-silence and no-unprefixed-line tests that are the
point of the issue.
