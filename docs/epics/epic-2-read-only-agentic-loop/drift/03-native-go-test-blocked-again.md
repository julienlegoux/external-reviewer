---
type: Drift record
title: "The native half of the test command stopped working again"
description: "CONVENTIONS says `go test ./...` runs natively once GOTMPDIR is moved; on 2026-08-12 it did not, so issue 05 ran entirely on the remote — root cause found the same day (Smart App Control keys its verdict to the binary's hash), so the drift is resolved rather than accepted."
tags: [epic-2, drift]
timestamp: 2026-08-12T10:20:00Z
epic: 2
issue: 05
---

# The native half of the test command stopped working again

## Decided

[CONVENTIONS § Testing](../../../planning/CONVENTIONS.md) splits the decided command in
two, and the split is documented as settled: *"`go test ./...` runs natively, and is the
fast inner loop — no ssh, no sync. It needs one machine-local setting,
`go env -w GOTMPDIR=C:\dev\gotmp` or any path outside `%LOCALAPPDATA%\Temp`."*
[DRIFT](../../../planning/DRIFT.md) records the same finding as **resolved** on
2026-08-11, and `scripts/test-remote.sh`'s own header repeats it.

## Actual

On 2026-08-12 the native half does not run at all. `go env GOTMPDIR` reports
`C:\dev\gotmp` — the setting the resolution prescribes, already in the machine's Go env —
and every package still fails before a test does:

```
fork/exec C:\dev\gotmp\go-build3702923759\b001\tools.test.exe: An Application Control policy has blocked this file.
FAIL	github.com/julienlegoux/external-reviewer/internal/tools	1.389s
```

So this issue's entire red-green cycle ran on the remote host through
`scripts/test-remote.sh`, which means every local run carried `-race` and every local run
paid an ssh round trip. Nothing about the code changed for it; what changed is that the
"fast inner loop" half of § Testing is currently not available at all.

## Because

The policy this whole entry is about refuses to *execute* the freshly linked test binary,
not merely to load unsigned modules into it. Relocating `GOTMPDIR` was only ever a
workaround for where that binary is written, so it stops helping the moment the block is
on the binary itself. Four combinations were verified within minutes of each other, all
identical:

- `GOTMPDIR=C:\dev\gotmp` (the prescribed value, from the machine's Go env) — blocked.
- `GOTMPDIR=D:\gotmp`, a freshly created directory on the other drive — blocked.
- Through the agent's Bash (Git Bash) tool — blocked.
- Through PowerShell 7 — blocked, same sentence, same exit code.

The remote path is unaffected: `scripts/test-remote.sh ./internal/diag/` was green in the
same session, so the toolchain, the module cache and the working tree are fine and the
refusal is entirely local policy.

## Alternatives tried

- **Both `GOTMPDIR` locations above**, which is the documented fix, applied exactly.
- **Two different shells**, to rule out the agent harness's sandbox as the cause.
- **A C toolchain** was not attempted: the earlier entry already establishes that cgo
  cannot load on this machine, and installing one would not help a block that lands on a
  pure-Go test binary.

Raising the policy itself is out of reach from a Claude Code session — no administrator
rights, and Smart App Control cannot be re-enabled once turned off.

## Resolved (2026-08-12) — the observations above are right, the cause named in them is wrong

Everything recorded above was observed accurately. The inference — that the policy had
started refusing to execute freshly linked test binaries as a class, making the native
half unavailable — was wrong, and so was the older `GOTMPDIR` explanation it argued
against.

**Smart App Control keys its verdict to the binary's exact SHA-256, and nothing else.**
Measured on `internal/confine` later the same day:

| Binary | SHA-256 (first 8) | Verdict |
|---|---|---|
| `go test -c` output | `04773FBE` | blocked |
| byte-identical copy, different name **and** directory | `04773FBE` | blocked |
| deterministic rebuild of unchanged source | `04773FBE` | blocked |
| same source built with `-ldflags=-s` | `E4DB25FC` | runs |

Not the package, not the path, not the filename, not `-race`, not the size —
`internal/diag`'s 10.8 MB race binary ran while `confine`'s 7.5 MB one was refused.

This explains the four identical failures recorded under *Because*: they were four runs
of **the same cached binary**, so they could only ever return the same verdict. It also
explains why a concurrent session on another Go project saw a *different subset* blocked
each run — its source was changing between runs, so every hash was new. One rule, two
workflows, two contradictory-sounding reports.

`-ldflags=-s` re-rolls the hash, and with it `go test -race -ldflags=-s ./... -count=1`
passes natively on Windows — whole suite. cgo is not blocked either: WinLibs MinGW-W64
UCRT `gcc` 16.1.0 compiles and its unsigned output executes.

So the standard was never unworkable; it was described wrongly. [CONVENTIONS §
Testing](../../../planning/CONVENTIONS.md) and [DRIFT](../../../planning/DRIFT.md) were
both amended on 2026-08-12 to state the hash rule and drop `GOTMPDIR`. **This record needs
no disposition at close beyond `resolved`.**

## Revisit when

A package starts failing again after a source change — that is a new hash drawing a
block, not a regression in the toolchain or a policy change. Re-roll with build flags
rather than debugging the code.

Two smaller consequences worth carrying: concurrent implementers must give the script
distinct `EXTERNAL_REVIEWER_TEST_DIR` values, since it wipes one fixed remote directory
(`ci/external-reviewer`) that all of them would otherwise share — that one still stands
and is an open follow-up; and the two-OS matrix claim in § Testing —
*"`windows-latest` behaviour is otherwise reproducible on the spot"* — which looked false
at the time but holds again under the resolution below.

## Evidence

- The four runs above, in this issue's session; the error text is quoted verbatim.
- [CONVENTIONS § Testing](../../../planning/CONVENTIONS.md),
  [DRIFT § 09, 10, 11](../../../planning/DRIFT.md), `scripts/test-remote.sh` (header).
- Every test result reported on [issue 05](../issues/05-search-tool.md)'s PR comes from
  the remote runner, with `-race`, for this reason.
