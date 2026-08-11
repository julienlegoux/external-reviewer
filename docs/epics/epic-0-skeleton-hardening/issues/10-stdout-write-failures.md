---
type: Issue
title: "Survive a failing or closed stdout without escaping the done-line seam"
description: "Stop SIGPIPE killing the process before the done line, and classify a mid-report write failure as an ordinary exit-2 termination that says the report is incomplete."
tags: [epic-0]
timestamp: 2026-08-11T08:49:37Z
epic: 0
issue: 10
slug: stdout-write-failures
size: M
status: done
gh_issue: 32
gh_pr: 42
resource: https://github.com/julienlegoux/external-reviewer/issues/32
depends_on: [7]
---

# Survive a failing or closed stdout without escaping the done-line seam

## Summary

`io.WriteString(stdout, report)` at `internal/cli/review.go:77` is treated as
all-or-nothing. It is not, and the hole has two manifestations.

**Error mid-write.** Redirect stdout to a filesystem that fills. `poll.FD.Write` loops, so a
silent short write is impossible, but an error after N bytes is not: the first N bytes are
already on stdout, `WriteString` returns `ENOSPC`, the run exits `2` — and the calling
skill's report file now holds a truncated report beside a failure code. The invariant fails
exactly where it is load-bearing.

**Broken pipe.** `external-reviewer review /repo --prompt "…" | head -20`. `head` exits and
closes the read end; the multi-KB write gets `EPIPE` on fd 1. The program registers
`os/signal` only for `os.Interrupt`, so the Go runtime's default rule applies — **SIGPIPE on
fd 1 kills the process by signal**. `defer diag.WriteDone` never runs; the shell sees `141`,
not `0`/`1`/`2`. Piping into a pager and quitting does the same. This is the second of the
two terminations that escape the `done`-line seam by not returning at all.

**A judgment this issue asks the implementer to make explicit, not discover.** Bytes already
on stdout cannot be unwritten, so "stdout is empty on any failure" cannot be discharged
after a partial write by retracting it. What *is* decidable: nothing is written until the
final message is complete (already true — `internal/cli/review.go:44-48` says so), the
process never dies by signal, and a partial write is announced on stderr so the caller can
tell a truncated report from a complete one. Land whichever wording is true of the shipped
code, and amend SPECS § Interfaces in the same PR if the sentence there has to change —
issue 03 of Epic 1 set that precedent. Do not leave the documents and the code disagreeing.

## Scope

- SIGPIPE on fd 1 no longer kills the process: the signal is handled (e.g. `signal.Notify`
  for `syscall.SIGPIPE`) so the write returns `EPIPE` to Go instead of terminating the
  process, and the run finishes through the ordinary seam with a `done` line and a real exit
  code.
- CONVENTIONS forbids `//go:build` platform divergence without a test that runs on both
  OSes, so whatever platform-conditional piece this needs carries a test that runs on
  ubuntu-latest and windows-latest and selects its expectation at runtime, not one that
  compiles away on Windows.
- A failed stdout write classified as an ordinary exit-`2` failure: one `error:` line, a
  `done` line, a recorded stop reason — never a signal death and never a silent success.
- The `error:` line names the report as incomplete, so a truncated report is distinguishable
  from a complete one by the caller reading the transcript.
- The invariant restated wherever it is written down (`internal/cli/review.go:44-48` and,
  if the wording needs it, `docs/planning/SPECS.md` § Interfaces) to say exactly what holds.

## Out of scope

- Buffering the report to a temporary file, or any output file at all. SPECS is explicit:
  the binary produces no output file and the calling skill keeps sole ownership of the
  report.
- The stdin half of the seam escapes —
  [issue 09](/epic-0-skeleton-hardening/issues/09-bounded-validated-prompt.md).
- Streaming the report to stdout as deltas arrive. Writing once, at the end, from the
  completed final message is what makes "empty on failure" true for the failures that stream
  a paragraph before falling over; that stays.
- Retrying a failed write.

## Acceptance criteria / Definition of done

- [ ] Written test-first, black-box through `Run`, with the failing stdout supplied as an
      `io.Writer` the test controls.
- [ ] A stdout writer returning `syscall.EPIPE` produces exit `2`, a `done` line and exactly
      one `error:` line — the process does not die by signal.
- [ ] `external-reviewer review <repo> --prompt "…" | head -20` exits `0`, `1` or `2` with a
      `done` line on stderr and **never** `141`, verified by hand on a Unix shell with the
      exit code recorded in the PR body.
- [ ] A stdout writer that accepts N bytes and then returns an error leaves the run at exit
      `2` with a `done` line and one `error:` line naming the report as incomplete; the test
      asserts the classification and the diagnostic.
- [ ] The invariant is stated in the code (and in SPECS § Interfaces if its wording had to
      change) in a form that is true of the shipped behaviour, and a test asserts what it
      claims. The PR body states which of the two documents changed and why.
- [ ] The success path is untouched: a complete report still reaches stdout byte-for-byte
      unchanged at exit `0`.
- [ ] No `//go:build`-divergent behaviour ships without a test that runs on both matrix
      OSes.
- [ ] `go test ./... -race` and `golangci-lint run` pass; CI green on ubuntu-latest and
      windows-latest.

## Relevant files / areas

- `internal/cli/review.go:77` (the write), `:44-48` (the doc comment stating the invariant)
- `internal/cli/run.go:34-40` (`Run`, where `os/signal` is registered today — only for
  `os.Interrupt`), `:47-49` (the inner seam and its `defer diag.WriteDone`)
- `internal/cli/roundtrip_test.go` — the round-trip tests these join
- `docs/planning/SPECS.md` § Interfaces (the "empty on any failure" sentence)

Governing decisions: [SPECS § Interfaces](../../../planning/SPECS.md),
[CONVENTIONS § Error handling](../../../planning/CONVENTIONS.md) L142-145 (stdout and stderr
are not interchangeable), [CONVENTIONS § Paths and platforms](../../../planning/CONVENTIONS.md)
L78-79 (no `//go:build` divergence without a two-OS test). Source finding:
[report 2](../../../REPORT_2.md) § "A failing stdout write leaves a partial report on stdout
with exit 2, or kills the process before the `done` line".

## Dependencies

- Blocked by: [07 — One prefixed `error:` line](/epic-0-skeleton-hardening/issues/07-single-error-line.md)
  — the failure this PR introduces a path for is reported through that writer.
- Blocks: nothing.

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
Sized **M**: ~250 lines — the signal handling and its two-OS test are the bulk; the write
classification is small.
