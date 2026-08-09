---
type: Issue
title: "Call the model once and return its markdown on stdout"
description: "Drive one streamed round trip through kern-link and write the final assistant text verbatim to stdout, closing the walking skeleton and retiring the first half of the project's largest risk."
tags: [epic-1]
timestamp: 2026-08-09T07:50:00Z
epic: 1
issue: 05
slug: single-turn-model-call
size: M
status: open
gh_issue: 8
resource: https://github.com/julienlegoux/external-reviewer/issues/8
depends_on: [4]
---

# Call the model once and return its markdown on stdout

## Summary

This is the PR that makes the skeleton walk: one streamed round trip to a foreign model
through `kern-link`, with the final assistant text on stdout verbatim and no envelope,
because the calling skill keeps sole ownership of the report file. It also retires the
first half of the project's largest risk — whether `kern-link` carries this workload at
all — and leaves Epic 2 the multi-turn half.

## Scope

- One turn: `models.StreamSimple` + `Stream.Result(ctx)` over an accumulating
  `[]ai.Message` seeded with the task prompt from issue 02. Streaming rather than
  `CompleteSimple`, because the run must be observable while it happens.
- The final assistant text written to **stdout verbatim, with no envelope**, exit `0`.
- Failure paths, all exit `2` with an empty stdout and a reason on stderr: a failed turn,
  a `StopReason` of `error` or `aborted`, an empty final assistant message, and
  cancellation (`SIGINT`) mid-stream.
- Real turn/token/cost/elapsed values fed into the run state issue 03 renders, so the
  `turn` line and the `done` line carry actual numbers.
- `kern-link`'s non-fatal `AssistantMessageDiagnostic` values — already redacted upstream
  — printed as `warn` lines rather than discarded.
- A reusable `faux`-provider test harness the later epics build their loop tests on.

## Out of scope

- The multi-turn loop, tool declarations, tool dispatch and `StopReasonToolUse` handling
  — Epic 2. This PR sends no tools and stops after one assistant message.
- `os.OpenRoot` confinement and the `--allow` list — Epic 2. The repository path is
  validated as a directory (issue 02) and otherwise unused.
- Loop bounds and their caps — Epic 2.
- Any caller-supplied system prompt; the skeleton sends the task prompt only.

## Acceptance criteria / Definition of done

- [ ] Written test-first and black-box, driven through `Run` against the `faux` provider
      in a `MutableModels` registry — offline, no network, no bill.
- [ ] A scripted successful response puts the model's final text on stdout **byte-for-byte
      unchanged** (asserted on an answer containing markdown, a trailing newline and a
      non-ASCII character) and returns exit `0`.
- [ ] A scripted turn failure → exit `2`, stdout empty, reason on stderr.
- [ ] A scripted `StopReason` of `error`, and separately of `aborted` → exit `2`, stdout
      empty.
- [ ] A scripted response whose final assistant message is empty → exit `2`, stdout
      empty.
- [ ] Cancelling the context mid-stream → exit `2`, stdout empty, interruption reason on
      stderr.
- [ ] The `done` line is present on all of the above, and carries a non-zero elapsed time
      and a turn count of 1 on the success path.
- [ ] A scripted `AssistantMessageDiagnostic` appears on stderr as a `warn` line.
- [ ] **The binary creates, modifies or deletes no file anywhere**: a test runs a full
      successful review against a `t.TempDir()` repository and asserts the tree's file
      set and modification times are unchanged afterwards. This complements the forbidigo
      guard from issue 01 rather than replacing it.
- [ ] A live-model test, if added at all, is gated at runtime by `os.Getenv` + `t.Skip`
      naming the missing variable — never a build tag — so `go test ./... -race` is green
      offline on a clean checkout. No golden files for model output.
- [ ] Manual verification recorded in the PR description: the binary run by hand against
      a real configured provider returned markdown on stdout and exited `0`. This is the
      epic's actual risk-retirement, and no automated test can stand in for it.
- [ ] `golangci-lint run` passes; CI green on both OSes.

## Relevant files / areas

No existing code for this yet. Expected paths:

- `internal/reviewer/run.go` (the single round trip), `internal/reviewer/run_test.go`
  (`package reviewer_test`)
- `internal/reviewer/faux_test.go` or `internal/reviewer/testdata` — the scripted-provider
  harness
- `internal/cli/run.go`, `internal/cli/run_test.go` (end-to-end through `Run`)

Governing decisions: [SPECS § Architecture](../../../planning/SPECS.md) (loop mechanics
and termination order), [SPECS § Interfaces](../../../planning/SPECS.md) (output and exit
codes), [SPECS § Testing](../../../planning/SPECS.md) (the `faux` provider),
[CONVENTIONS § Testing](../../../planning/CONVENTIONS.md).

## Dependencies

- Blocked by: [04 — Resolve the hard-coded reviewer model and preflight auth](/epic-1-walking-skeleton/issues/04-model-resolution-and-auth.md).
- Blocks: nothing in this epic; Epic 2 builds its loop on this PR's harness.

## PR size note

Sized **M**: target ~450 changed lines — one round trip and its five failure paths, plus
the reusable `faux` harness the whole of Epic 2 tests against, which is the part worth
spending lines on. It does not split: the harness with no round trip tests nothing, and
the round trip with no harness cannot be tested offline.
