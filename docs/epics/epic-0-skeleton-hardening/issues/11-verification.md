---
type: Note
title: "Issue 11 verification log"
description: "Observed red/green runs and mutation checks behind the three repairs in Conversation.Next, including what could and could not be verified on this machine."
tags: [epic-0]
timestamp: 2026-08-11T10:30:00Z
epic: 0
issue: 11
---

# Issue 11 verification log

Everything below was run from the issue branch. `go test` cannot execute its own
temp binaries on this machine (a local Windows Application Control policy blocks
them), so tests are cross-built for `linux/amd64` and run under
`wsl -d docker-desktop`, which is a real Linux kernel — the same OS as CI's
`ubuntu-latest` job. `-race` could not be built at all: it needs cgo, and neither
the Windows toolchain nor the WSL distro has a C compiler. **CI is the authority
for `-race` and for `windows-latest`.**

## Red before green

The new tests were run against the unrepaired `Conversation.Next`, with only the
`StreamTimeout` field and the `ErrStreamTimeout` sentinel stubbed in so the
package would compile:

* `TestNext_ReportsTheAdaptersOwnCost` — `turn cost = 0.2, want the adapter's own
  service-tier-adjusted 0.5`. The second `ai.CalculateCost` reproduced the base
  price sheet exactly, as predicted.
* `TestNext_SilentProviderEndsAtTheStreamTimeout` — **hung**, and the package
  died on `panic: test timed out after 2m0s` with the stack parked in
  `Conversation.Next` at the `for range stream.Events(ctx)` loop. That is the
  reported defect observed rather than argued: nothing bounded the round trip.
  The panic stopped the run before the remaining cases executed.

## Mutation checks (green code, one repair removed at a time)

| Mutation | Result |
| --- | --- |
| `accounted` recomputes unconditionally (`ai.CalculateCost` restored) | `TestNext_ReportsTheAdaptersOwnCost` fails (0.2 vs 0.5); `TestRun_DoneLineCarriesTheAdaptersOwnCost` fails with `done cost = $0.2000, want $0.5000` |
| the `endedBecause` branch deleted from the stop-reason check | `TestNext_AbortedMessageUnderCancelledContextIsInterrupted` fails; `TestNext_AbortingProviderStillReportsTheStreamTimeout` fails; `TestRun_CancelledMidStream_ExitsTwo` fails **3 runs out of 3** with `done stop = "failed", want interrupted` — the exact signature of the intermittent `windows-latest` failure on issue 05's PR, now deterministic |
| the graced second `stream.Result` read deleted | `TestNext_CompleteMessageBeatsALateCancellation` fails at repeat 3 of 20 (the coin flip it exists to remove) |
| the local `context.WithTimeout` removed | the silent-provider case hangs forever — observed in the red run above |

## Green

`go build ./...`, `go vet ./...`, then each package's test binary run five times
in a row under WSL: `internal/reviewer` 21 tests, `internal/cli` 59 tests and
subtests (one skip, `TestRun_LiveModel_ReturnsMarkdown`, which needs
`EXTERNAL_REVIEWER_LIVE_MODEL`), `internal/diag` 9 tests. No failures in any of
the five repeats.

Formatting and lint were checked in a fresh clone (`git clone -c
core.autocrlf=false`) after `golangci-lint cache clean`, because this machine's
`core.autocrlf=true` makes `gofmt -l .` flag every file and golangci-lint's
absolute-path cache has reported a stale clean run before: `gofmt -l .` printed
nothing and `golangci-lint run` printed `0 issues.`

## Argued rather than observed

* Behaviour under `-race`, on either OS.
* Behaviour on `windows-latest`.
* That `StreamOptions.Timeout` makes kern-link's real transports arm their read
  deadlines. The test asserts only that the value reaches the adapter; what the
  SSE and codex WebSocket paths do with it is read from kern-link v0.1.1's
  source, not exercised. The local `context.WithTimeout` is what actually bounds
  the turn, and that is exercised.
* That ten minutes is the right default. It is reasoned from the shape of a
  reviewing turn, not measured against production traffic.
