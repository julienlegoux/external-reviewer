---
type: Issue
title: "Set the loop caps from Epic 2's measurements"
description: "Fill in the bounds seam with turn and wall-clock ceilings sized from real runs, decide which to expose as flags, and make a bounded run exit 0 with the report it had."
tags: [epic-3]
timestamp: 2026-08-13T07:10:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/68
epic: 3
issue: 09
slug: loop-caps-from-measurements
size: S
status: open
gh_issue: 68
depends_on: [1]
---

# Set the loop caps from Epic 2's measurements

## Summary

From this epic on the binary runs **unsupervised inside a script**, on the user's own
credentials, with nobody watching the `turn` lines. SCOPE deferred the caps until real
numbers existed precisely so they would not be guessed
([scope 08](../../../planning/scope/08-loop-bounds-and-termination.md)); the numbers now
exist ([MEASUREMENTS](/epic-2-read-only-agentic-loop/MEASUREMENTS.md)), and this PR sets
them.

`reviewer.Bounds` and its `check` already ship as a seam with every field unset, and the
loop consults it at the top of every turn. So this is a data change plus one correctness
gap, not a restructuring.

**The correctness gap, verified in the current code**: at a bounds stop `Loop.Run` returns
the last non-empty assistant text — which is `""` when the run never produced prose — and
`resolveAndReview` turns an empty report into
`errors.New("the reviewer finished but its final message carried no text")`, which
`runReview` classifies as `stop=failed`, exit `2`. SPECS says a bounded run is
**exit `0` with the report the run had**, and MEASUREMENTS shows why it matters: the report
is written in one final turn, so a run stopped before it has no prose at all. A bounded run
with nothing to show must still report as bounded.

## Scope

- `Bounds` values set on the path every real invocation takes (`cli.Run`), sized from
  MEASUREMENTS' observations and **stated with the run that suggests each**:
  - `MaxTurns` in the range **25–40**. Observed: 4, 6 and 12 turns, the 12 inflated by the
    `git_read` gap [issue 01](/epic-3-reviewer-selection-integration/issues/01-git-read-path-scoped-diff.md)
    closes; nothing observed justifies a single-digit cap.
  - `MaxElapsed` in the range **15–25 minutes**. Observed: 1m57s, 2m12s, 3m59s — but the
    final report-writing turn alone took 1m35s–2m5s in all three runs, so a deadline below
    ~5 minutes kills runs mid-report and discards everything.
  - `MaxCost` stays **unset (0)**, with the comment saying why: the credential is a
    subscription, nothing is billed per token, and a cost bound is the least informative of
    the three. The field stays in the struct because a tier naming an API-key provider
    makes it real again and the check is one comparison either way.
- The **flag decision**, which this issue owns and must record in the PR body: whether any
  of the three ceilings is exposed on the command line. If any flag is added, SPECS'
  `review` grammar and [specs 02](../../../planning/specs/02-cli-surface-and-argv.md) are
  amended in the same PR — the grammar is a contract with a script in another repository.
  If none is added, the PR says so and why, so the next reader finds a decision rather than
  an omission.
- The bounded-termination path corrected end to end: `stop=bounds`, exit `0`, the `warn`
  line naming which of the three bounds bit, and whatever report the run had — including
  none.
- One `turn`-line-visible consequence checked: the bound is consulted *before* the turn, so
  `MaxTurns: N` means N turns are taken, not N+1.

## Out of scope

- Changing the loop's termination *order* (bound, then cancellation, then stop reason) or
  any other loop mechanic — [specs 07](../../../planning/specs/07-agent-loop-mechanics.md)
  fixed it and Epic 2 shipped it.
- Per-tool caps (`MaxGitReadLines`, `MaxListPaths`, `MaxReadLimit`, `MaxSearchMatches`).
  MEASUREMENTS shows they are what keeps context off the binding path; they are not
  re-tuned here.
- A model-authored coverage statement. Dropped with the original caps and named in SCOPE as
  the reason risk 3 stays unmitigated in v1.
- Retry behaviour after a failed turn — MEASUREMENTS notes one hand run in four died on a
  transient provider error, and whether the *caller* retries is
  [issue 10](/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md)'s
  question, not the binary's.
- Re-measuring. [Issue 11](/epic-3-reviewer-selection-integration/issues/11-success-criteria-verification-run.md)
  produces the fresh numbers.

## Acceptance criteria / Definition of done

Strict red-green, black-box, against `kern-link`'s `faux` provider scripting a run that
keeps calling tools ([CONVENTIONS § Testing](../../../planning/CONVENTIONS.md)).

- [ ] `cli.Run`'s `Bounds` is no longer the zero value: `MaxTurns` ∈ [25, 40],
      `MaxElapsed` ∈ [15m, 25m], `MaxCost` == 0 — asserted directly, so a later edit that
      quietly unsets one fails a test.
- [ ] Written failing first: a scripted run that would take more turns than `MaxTurns`
      terminates with `stop=bounds`, exit **`0`**, one `warn` line naming the `turn` bound,
      and the last prose the run produced on stdout.
- [ ] **The empty-report case**: a bounded run that produced no assistant text at all still
      exits `0` with `stop=bounds` and an empty stdout — it must not be reclassified as
      `failed` by the "final message carried no text" check in `resolveAndReview`.
- [ ] A run bounded by elapsed time reports `stop=bounds` with the `elapsed` bound named
      (drive it with a deliberately tiny `MaxElapsed` through the existing test seam, not by
      sleeping).
- [ ] `MaxTurns: 1` yields exactly one turn, not two — the `>=` comparison in
      `Bounds.check` asserted through the loop rather than in isolation.
- [ ] A run that ends normally under the caps is unaffected: `stop` carries the model's own
      reason, exit `0`.
- [ ] The `done` line's totals are correct on the bounded path (turns, tokens, cost,
      elapsed all present, never absent).
- [ ] If a cap flag is added: it appears in `usageText`, in SPECS' grammar block and in
      specs 02, and a test drives it through `Run`. If none is added, the PR body records
      the decision and its reason.
- [ ] `go test -race -ldflags=-s ./... -count=1` green; `golangci-lint` clean.

## Relevant files / areas

- `internal/reviewer/bounds.go` — `Bounds`, `check`, `BoundsStopReason`
- `internal/reviewer/loop.go` — `Loop.Run`'s bound branch (returns the accumulated report)
- `internal/cli/run.go` — `Run`'s `reviewer.Bounds{}`, today deliberately the zero value
- `internal/cli/review.go` — `resolveAndReview`'s empty-report check, and `runReview`'s
  stop-reason switch
- `internal/cli/loop_test.go`, `internal/reviewer/loop_test.go`
- `docs/planning/SPECS.md` § Interfaces (the `stop=` closed set already reserves `bounds`)

Governing decisions:
[MEASUREMENTS § Observations for Epic 3](/epic-2-read-only-agentic-loop/MEASUREMENTS.md),
[scope 08 — Loop bounds and termination](../../../planning/scope/08-loop-bounds-and-termination.md),
[specs 07 — Agent loop mechanics](../../../planning/specs/07-agent-loop-mechanics.md),
[SPECS § Interfaces](../../../planning/SPECS.md).

## Dependencies

- Blocked by: [01 — git_read path-scoped diff](/epic-3-reviewer-selection-integration/issues/01-git-read-path-scoped-diff.md),
  the single largest distortion in the turn counts these caps are sized against.
- Blocks: [11 — success-criteria verification run](/epic-3-reviewer-selection-integration/issues/11-success-criteria-verification-run.md),
  which reports its cap bits and headroom against the `Bounds` this issue sets and cannot
  do so until they land.

## PR size note

A data change — three `Bounds` fields set on `cli.Run`'s existing seam — plus one
correctness-gap fix in `resolveAndReview`'s empty-report check, as the Summary already
says: not a restructuring. The one place this could grow is the flag-exposure decision the
issue owns, and the issue already permits answering it "no" and recording why, rather than
shipping a flag to keep the PR from feeling small.
