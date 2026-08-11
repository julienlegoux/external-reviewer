---
type: Issue
title: "Drive the multi-turn loop with tool dispatch and the bounds seam"
description: "Turn the single round trip into an accumulating multi-turn loop that declares tools, dispatches tool calls, feeds results back, and terminates on bound then cancellation then stop reason."
tags: [epic-2]
timestamp: 2026-08-11T03:38:33Z
epic: 2
issue: 03
slug: multi-turn-loop-and-dispatch
size: L
status: open
gh_issue: 11
resource: https://github.com/julienlegoux/external-reviewer/issues/11
depends_on: [2]
---

# Drive the multi-turn loop with tool dispatch and the bounds seam

## Summary

This is the PR that makes the binary a reviewer rather than a prompt-pipe: the model
declares what it wants to read, the loop dispatches it, and the result comes back as
another turn. It lands the whole loop shape at once — accumulation, dispatch,
termination order, the bounds seam with its values unset, and the two failure levels —
against the one tool that exists so far, `list`. Issues 04–06 then add tools to a loop
that already works rather than integrating three tools and a loop in one PR.

The bounds seam is present here with **no numbers**. SCOPE defers real caps until
measurements exist, and having the seam means Epic 3 sets values rather than
restructuring the loop.

## Scope

- The loop: `models.StreamSimple` + `Stream.Result(ctx)` per turn over an accumulating
  `[]ai.Message`, exactly the shape `kern-link` documents, built on the `faux`-provider
  harness Epic 1 issue 05 leaves behind.
- Tool declarations from issue 02's registry passed on every request; `ai.ToolCall`
  arriving as a decoded `map[string]any`, dispatched by name, results appended as
  `ToolResultMessage` before the next turn.
- Termination checked **in this order**: bound exceeded, then context cancellation
  (`SIGINT`), then `StopReason != StopReasonToolUse`. The order is asserted, not
  incidental.
- `bounds.check(state)` at the top of every turn, watching turns, accumulated cost and
  elapsed wall clock, **with its values unset** so nothing bounds a run in this epic. An
  exceeded bound stops the loop and returns what the run has, rather than failing.
- **Where the accumulated state lives, and which side wins.** Today the turn count,
  accumulated tokens and cost `bounds.check(state)` needs at the top of each turn live in
  `internal/cli` — `recordTurn` (`internal/cli/review.go:88-105`) folds each
  `reviewer.Turn` into `*diag.State` one call frame above `Conversation.Next`
  (`internal/reviewer/run.go:61-98`), only after `Next` has already returned. This PR's
  loop lands in `internal/reviewer`, where the totals do not yet exist at the point
  `bounds.check` needs to read them. **`internal/reviewer` wins**: `recordTurn`'s
  accumulation logic moves into the loop itself, which takes the same `*diag.State` and
  `io.Writer` (stderr) that `cli.recordTurn` takes today and updates `State` in place
  after every `Next`, so `bounds.check(state)` reads the running totals in-process instead
  of a value threaded back up to `cli` and down again on every turn. `internal/cli` keeps
  creating the `*diag.State` via `diag.NewState` before the loop starts and rendering the
  final `done` line from it once the loop returns — that half of Epic 1's contract does
  not move.
- **How the `tool` line reaches stderr from inside the loop.** `diag.WriteTool`
  (`internal/diag/diag.go:40-43`) has exactly one caller today — its own test — because
  nothing in `internal/reviewer` imports `internal/diag` yet. This PR adds that import
  (one-directional: `internal/diag` imports nothing from `internal/reviewer`, so no
  cycle) and calls `diag.WriteTool` directly from the loop for the `tool` line described
  below, the same way the moved accumulation above calls `diag.WriteTurn`.
- The two failure levels kept distinct, which is the single place CONVENTIONS' "never
  swallow an error" rule inverts:
  - a failing **tool** is a turn — returned as `ToolResultMessage{IsError: true}` with
    text saying what was wrong and what is allowed, never propagated up the stack;
  - a failing **turn** ends the run at exit `2`.
- Run instrumentation as the run progresses, into Epic 1 issue 03's diagnostics writer:
  a `turn` line per turn (`turn 3  tools=2  in=48210 out=1104  $0.0231  42.8s`) and a
  `tool` line per dispatch naming the tool and its arguments in wire form.
- An unknown tool name, or arguments that do not match the declared schema, returned as
  a tool error rather than crashing the run.

## Out of scope

- Bound **values**. Deliberately deferred to Epic 3, to be sized from issue 07's
  measurements rather than guessed. The seam ships; the numbers do not.
- `read_file`, `search` and `git_read` — issues 04, 05 and 06. This PR wires `list` and
  the dispatch mechanism the others plug into.
- Concurrency. Tool execution is sequential, one review per process
  (SPECS § Distribution & operations).
- Context-overflow mitigation. Risk 3 stays unmitigated in v1; this epic makes it
  observable through the `turn` lines, nothing more.

## Acceptance criteria / Definition of done

- [ ] Written test-first and black-box through `Run`, against the `faux` provider in a
      `MutableModels` registry — offline, no network, no bill.
- [ ] A scripted three-turn conversation (tool call → tool result → tool call → tool
      result → final text) completes with the final assistant text on stdout verbatim,
      exit `0`, and a `done` line reading `turns=3`.
- [ ] The tool result the loop feeds back is asserted to contain the real `list` output
      for a `t.TempDir()` repository — the loop and the tool are exercised together, not
      through a stub.
- [ ] A scripted tool call naming an unknown tool yields `ToolResultMessage{IsError:
      true}`, **the run continues**, and a subsequent scripted final message still exits
      `0`.
- [ ] A scripted tool call whose path is outside the allow-list yields a tool error whose
      text names the rule that refused (issue 01), the run continues, and the model's
      next turn is dispatched.
- [ ] A scripted turn failure mid-loop ends the run at exit `2` with an empty stdout and
      a reason on stderr.
- [ ] Termination order is asserted: with a bound artificially set in a test **and** a
      `StopReasonToolUse` response pending, the bound wins; with cancellation and a
      pending tool-use response, cancellation wins.
- [ ] An exceeded bound returns what the run has — the last assistant text on stdout —
      rather than failing, and the `done` line records the stop reason.
- [ ] Every terminating path above still emits the `done` line from Epic 1 issue 03,
      including cancellation.
- [ ] One `turn` line per turn and one `tool` line per dispatch appear on stderr, with
      `/`-separated wire paths.
- [ ] `golangci-lint run` passes; CI green on ubuntu-latest and windows-latest.

## Relevant files / areas

No existing code for this yet. Expected paths:

- `internal/reviewer/loop.go` (the turn loop and dispatch),
  `internal/reviewer/loop_test.go` (`package reviewer_test`)
- `internal/reviewer/bounds.go`, `internal/reviewer/bounds_test.go`
- `internal/reviewer/run.go` — the single round trip from Epic 1 issue 05, generalised
- `internal/tools/registry.go` — declarations passed to the model

Governing decisions:
[specs 07 — Agent loop mechanics](../../../planning/specs/07-agent-loop-mechanics.md),
[SPECS § Architecture](../../../planning/SPECS.md),
[specs 13 — Error handling and failure classification](../../../planning/specs/13-error-handling-and-failure-classification.md),
[CONVENTIONS § Error handling](../../../planning/CONVENTIONS.md) (a failing tool is a
value; a failing turn is an error).

## Dependencies

- Blocked by: [02 — Enumeration and the list tool](/epic-2-read-only-agentic-loop/issues/02-enumeration-and-list-tool.md);
  and Epic 1's
  [05 — Call the model once and return its markdown on stdout](/epic-1-walking-skeleton/issues/05-single-turn-model-call.md),
  whose `faux` harness and single round trip this generalises.
- Blocks: [04 — read_file](/epic-2-read-only-agentic-loop/issues/04-read-file-tool.md),
  [05 — search](/epic-2-read-only-agentic-loop/issues/05-search-tool.md),
  [06 — git_read](/epic-2-read-only-agentic-loop/issues/06-git-read-tool.md) — each of
  which asserts its tool end-to-end through this loop.

## PR size note

Sized **L**: target ~650 changed lines — the accumulating loop, dispatch, the bounds
seam, the asserted termination order, the instrumentation lines, and the scripted `faux`
conversations that exercise each. It stays whole: the loop, its termination order and its
two failure levels are one behaviour, and a loop that dispatches but cannot terminate in
the specified order is not half of this issue, it is the wrong half. If it passes ~1000,
the scripted-conversation fixtures move, not the loop.
