---
type: Issue
title: "Bound the turn and fix its cost and cancellation precedence"
description: "Three repairs in Conversation.Next: report the adapter's service-tier-adjusted cost, arm a stream timeout so a silent provider cannot hang forever, and prefer a complete message over a late ctx.Err()."
tags: [epic-0]
timestamp: 2026-08-11T03:33:08Z
epic: 0
issue: 11
slug: turn-hardening
size: M
status: open
gh_issue: 33
resource: https://github.com/julienlegoux/external-reviewer/issues/33
depends_on: [6]
---

# Bound the turn and fix its cost and cancellation precedence

## Summary

Three defects in one 40-line function, `Conversation.Next`
(`internal/reviewer/run.go:61-98`), each with its own test.

**The cost is reported at 40 % of what a `priority` response actually cost.** The adapter
has already filled `message.Usage.Cost` and then scaled it —
`ai/apis/openairesponses/stream.go:319` calls `ai.CalculateCost` and `:334` calls
`applyServiceTierPricing`, which multiplies every field and recomputes `Total`; codex routes
through this path. `internal/reviewer/run.go:83` copies the usage and calls
`ai.CalculateCost` **again**, and `ai/cost.go:16-20` assigns all five fields unconditionally
from the base price sheet. `serviceTierCostMultiplier` returns **2.5** for `priority` on
`gpt-5.5` and **0.5** for `flex`, so a `priority` response is reported at 40 % of its real
cost and a `flex` one at 2×. The second call can only discard information the adapter
computed.

**Nothing bounds the round trip.** `StreamSimple(ctx, model, chat,
&ai.SimpleStreamOptions{})` leaves `StreamOptions.Timeout` at zero and kern-link v0.1.1 arms
no default on either transport: the SSE path builds `&http.Client{}` with no `Timeout`
(`ai/apis/internal/httpretry/httpretry.go:96`), and the WebSocket path — the default for
codex — sets `idleTimeout = opts.Timeout` = 0 (`ai/apis/codex/websocket.go:297-299`) and
arms a read deadline only `if idleTimeout > 0` (`:355`); the 15 s
`defaultWebSocketConnectTimeout` covers the handshake only. A backend that accepts the
connection and then stops sending frames blocks `for range stream.Events(ctx)` forever, with
no output, no `done` line and no exit code.

**A finished, already-billed report is thrown away by a late Ctrl-C.** `stream.Result(ctx)`
returns the complete message and `ctx.Err()` is checked after (`:91-93`). A 90-second turn
finishes; the user, who stopped watching, presses Ctrl-C at t=90.001 s; the report is on the
wire and paid for, and the user gets an empty stdout, `stop=interrupted` and exit `2`. The
window upstream is worse: with `resultCh` already closed, `Stream.Result`'s select
(`ai/stream.go:155-158`) has both cases ready and Go picks one at random. And the guard is
dead to the suite either way — deleting the three lines leaves everything green, because
`TestRun_CancelledMidStream_ExitsTwo` (`internal/cli/roundtrip_test.go:173`) always wins the
race the guard exists to lose.

## Scope

- The second `ai.CalculateCost(c.model, &usage)` (`internal/reviewer/run.go:83`) removed; the
  adapter's figure is what the `turn` and `done` lines report, with a fallback computation
  only where the adapter left `Cost` unset, and a comment saying which is which.
- A default stream timeout set on `ai.SimpleStreamOptions`, its value and reason stated in
  the code — not a magic number. A provider that accepts the connection and then sends
  nothing terminates within it, through the ordinary seam.
- Cancellation precedence at `:91-93` inverted for the case it costs: a **complete,
  non-error** message is returned rather than replaced by a late `ctx.Err()`. An interrupted
  run that has no complete message is still reported as interrupted.
- The guard made reachable from the suite: a scripted `StopReasonAborted` message plus a
  cancelled context enters it deterministically, which is the case it exists for and that no
  test currently constructs.

## Out of scope

- `resolve.go:93`'s `flock.TryLockContext(ctx, 20ms)`, which retries indefinitely if another
  process holds `~/.pi/agent/auth.json.lock`. Same shape, different call path, and outside
  this epic's acceptance criteria — recorded in [report 2](../../../REPORT_2.md) and left
  there deliberately.
- Loop bounds (`bounds.check(state)`) and their values — Epic 2 and Epic 3. This is a
  transport timeout, not a run bound.
- Retrying a timed-out turn.
- The `stop=` vocabulary — [issue 01](/epic-0-skeleton-hardening/issues/01-stop-reason-vocabulary.md),
  already landed.

## Acceptance criteria / Definition of done

- [ ] Written test-first, black-box (`package reviewer_test` / `package cli_test`), in the
      `internal/reviewer/run_test.go` issue 06 created, table-driven where the cases share a
      shape.
- [ ] The `done` line's cost equals the adapter's service-tier-adjusted figure for
      `priority` and `flex` alike: a scripted turn whose `Usage.Cost.Total` differs from the
      base-sheet figure produces that same total on the `done` line. Re-adding
      `ai.CalculateCost` turns the test red, recorded in the PR body.
- [ ] A scripted turn whose `Usage.Cost` is unset still reports a computed cost rather than
      `$0.0000`, so removing the second call does not silently zero the figure.
- [ ] A provider that accepts the stream and then sends nothing terminates within a bounded
      time with a `done` line and exit `2`: a scripted step that blocks makes the turn return
      within the configured timeout, and the test's own deadline is comfortably longer than
      that timeout.
- [ ] A stream that completes before cancellation returns its message rather than
      `ctx.Err()`: a scripted complete, non-error message plus a context cancelled
      immediately afterwards exits `0` with the report on stdout.
- [ ] A scripted `StopReasonAborted` message plus a cancelled context enters the interruption
      guard deterministically and reports `stop=interrupted` at exit `2`; deleting
      `if err := ctx.Err(); err != nil { … }` turns the suite red, recorded in the PR body.
- [ ] The timeout's value and the reason for it are stated in the code.
- [ ] `go test ./... -race` and `golangci-lint run` pass; CI green on ubuntu-latest and
      windows-latest.

## Relevant files / areas

- `internal/reviewer/run.go:61-98` — `:65` (`SimpleStreamOptions`), `:82-83` (the cost
  recomputation), `:91-93` (the cancellation guard), `:94-96` (the stop-reason check)
- `internal/reviewer/run_test.go` — created by issue 06
- `internal/cli/roundtrip_test.go:173` (`TestRun_CancelledMidStream_ExitsTwo`), `:110`
- `internal/fauxtest/` — issue 05's harness, extended with a blocking step

Governing decisions: [SPECS § Architecture](../../../planning/SPECS.md) (the loop and its
termination order), [SPECS § Interfaces](../../../planning/SPECS.md) (the `done` line on
every termination path, the `$` figure as a volume proxy),
[CONVENTIONS § Error handling](../../../planning/CONVENTIONS.md). Source findings:
[report 2](../../../REPORT_2.md) § "`ai.CalculateCost` clobbers the adapter's
service-tier-adjusted cost", § "No timeout anywhere: a silent provider hangs the CLI forever
with no `done` line", § "A completed, paid-for report is discarded when cancellation lands a
moment after it arrives", § "The cancellation-precedence guard is dead to the suite".

## Dependencies

- Blocked by: [06 — the four surviving mutants](/epic-0-skeleton-hardening/issues/06-request-side-mutants.md)
  — its three tests land in the `internal/reviewer/run_test.go` that issue creates.
- Blocks: nothing.

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
Sized **M**: ~250 lines. Three small production changes in one function and four scripted
scenarios; they stay together because all three are the same function's contract and each
one's test needs the same scripted-stream machinery.
