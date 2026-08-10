---
type: Epic
title: "Skeleton hardening"
description: "Close the holes Epic 1's implementation review found — four surviving criterion-bearing mutants, two terminations that escape the done-line seam, and three contracts Epic 2 must extend but that are written nowhere."
tags: [epic, remediation]
timestamp: 2026-08-10T14:22:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/22
epic: 0
slug: skeleton-hardening
status: open
gh_issue: 22
milestone: 4
source: docs/REPORT_2.md
---

# Epic 0: Skeleton hardening

## Goal

Epic 1 built the right shapes — the `run(ctx, …)` inner seam with its single
`defer diag.WriteDone`, the `1`-versus-`2` classification, the credential isolation by
type — and [report 2](../../REPORT_2.md) confirms them under mutation. What it did not
do is make them true on every path, or write down the contracts Epic 2 has to extend.

Three things must land before Epic 2 resumes.

**The suite does not observe the request side of the round trip.** Mutating
`ai.UserText(task)` to the empty string leaves all three packages green: a binary that
silently discarded the user's prompt and reviewed nothing would ship. So would one whose
report is only its first text block. Epic 2's tool loop is built directly on both
properties.

**Two terminations escape the seam by not returning at all.** `review <repo>` with stdin
on a terminal traps `SIGINT` into a context that `io.ReadAll` never consults — the
process hangs with no `done` line, no exit code, and no working Ctrl-C, where the default
handler would have killed it. Piping into `head` kills the process by `SIGPIPE` before the
deferred `done` line runs. The seam is sound; these are holes beside it.

**Three contracts Epic 2 extends are undecided or unenforced.** The `stop=` field has two
competing vocabularies and SPECS documents neither's values. The `forbidigo` write-API
guard lets eight write operations through, including `(*os.File).Write` — and Epic 2 is
the epic that starts opening file handles. The OS-path/wire-path split exists in
CONVENTIONS and in **zero** product files, so Epic 2's `--allow` arguments arrive with no
precedent to copy and a stderr formatter demonstrating the wrong habit.

This epic is temporary by design. It exists to put the project back on its rails, and is
retired once every issue is `done` and its milestone closed.

## Scope

Plan and standard repairs land first: each one changes the contract the code work is
judged against, and doing it the other way round means fixing the code twice.

**Standards and plan**

- **The `done` line's stop-reason vocabulary, decided and documented.** Two concepts
  compete for one field: `ai.AssistantMessage.StopReason` is read at
  `reviewer/run.go:94` only to detect `error`/`aborted` and then discarded, while
  `diag.State.StopReason` gets a hand-written CLI vocabulary. They are answers to
  different questions — *why the model stopped* and *how the run terminated* — and a
  usage error can never carry `end_turn`. The success path renders the model's own
  reason; every termination the model never reaches keeps the CLI word. SPECS §
  Interfaces then documents the closed set, and the `model … auth=` line that ships
  outside its vocabulary today.
- **The write-API guard, made as wide as its stated purpose.** `.golangci.yml` matches
  CONVENTIONS' enumeration exactly — and the enumeration is the gap: `os.Chmod`,
  `os.Chtimes`, `os.Truncate`, `os.Link`, `os.Root.Chmod`, `(*os.File).Write`,
  `(*os.File).WriteString` and `(*os.File).Truncate` all pass the repo's own config with
  zero findings. CONVENTIONS is the source of truth and the config mirrors it, so the
  criterion is checkable rather than aspirational.
- **Epic 1's structural warnings carried into Epic 2's plan.** Epic 1 put the per-turn
  round trip in `internal/reviewer` and the accumulation of its numbers in
  `internal/cli`; Epic 2's `bounds.check(state)` seam needs those totals at the *top* of
  each turn, and `diag.WriteTool` is reachable only from `cli`. Issue #11 must say so
  rather than discover it. And issue #9's `os.Root` confinement must not build on
  `review.go:171`'s symlink-following `os.Stat`.

**Code**

- **CI formatting gate first**, since it protects every PR after it: golangci-lint v2
  moved formatters out of `linters`, this config has no `formatters:` block, and the
  `test` job runs only `go build` and `go test` — so nothing in the repository checks
  formatting. Landing it needs `.gitattributes` in the same PR (the dev machine has
  `core.autocrlf=true`, so a naive gate reports all 16 files), and the `lint` job joins
  `test` on the two-OS matrix.
- **The four surviving criterion-bearing mutants**, in a `run_test.go` that
  `internal/reviewer` does not have: the prompt reaching the model, `FinalText`'s
  concatenation and its filtering of thinking blocks and tool calls, the cache-token half
  of the `in=` figure, and the `messages` accumulator Epic 2's loop lands on.
- **The two escapes from the done-line seam**: a context-aware stdin read so `SIGINT` is
  observable before the model call, and stdout writes that survive a mid-write error or a
  closed pipe without leaving a truncated report beside a failure code or dying by signal.
- **One `error:` line, written once.** One concept has four renderings across eight
  hand-rolled `fmt.Fprint*` sites in two files, the empty-argv path emits none at all
  (its `errors.New("no command given")` is created, classified and never printed), and
  `review --help` exits `2`. `diag` owns the prefix everywhere else; it should own this
  too. The same pass collapses the three independent "was this interrupted?" decisions to
  one, and asserts the exit-1 path's silence — the criterion nothing currently tests.
- **Cancellation precedence at `reviewer/run.go:91-93`**: prefer a complete, non-error
  message over a late `ctx.Err()`, so a finished and already-billed report is not thrown
  away by a Ctrl-C that lands a moment after it arrives — and construct the case the guard
  exists for, which no test currently enters.
- **The OS-path/wire-path boundary, actually established.** `path/filepath` appears in no
  product file; the positional argument goes raw into `os.Stat` and raw back out to stderr
  through `%q`, which on Windows puts escaped backslashes and a capitalised, full-stopped
  OS sentence into a diagnostic CONVENTIONS requires lowercase, unpunctuated and
  `/`-separated.
- **A prompt that is validated and bounded**: `strings.TrimSpace` so a single newline from
  `echo |` cannot buy a real API call, an explicit byte bound on `io.ReadAll(stdin)`, and
  the `--prompt ""` case the flag-presence check silently allows.
- **Cost reported at its real figure.** The adapter has already computed and service-tier-
  scaled `message.Usage.Cost`; calling `ai.CalculateCost` again overwrites all five fields
  from the base sheet, reporting a `priority` response at 40 % of what it cost.
- **A bounded round trip.** `StreamOptions.Timeout` is zero and kern-link arms no default
  on either transport, so a backend that accepts the connection and then stops sending
  frames blocks forever with no output, no `done` line and no exit code.
- **Provider-controlled text escaped on stderr**, so an error body carrying a newline
  cannot forge a `done … stop=ok` line ahead of the real one in a transcript SPECS makes
  parseable — and the comment claiming kern-link redacts credentials upstream, which is
  true of no version of kern-link, removed rather than left telling the next reviewer not
  to look.
- **One importable `faux` harness**, replacing ~75 byte-identical lines copied across two
  test packages whose registry builders have already diverged, and retiring the mutable
  package-level seams issue 04's sanctioned mechanism superseded.
- **`diag` polish**: table-driven tests as CONVENTIONS defaults to, and a `formatElapsed`
  that has an hour unit and a boundary its tests pin.

## Out of scope

- Epic 2's own work. The plan repairs above amend Epic 2's issues; they do not implement
  them.
- Re-grading report 2. Its severities and its `won't-fix` stand as recorded; this epic
  converts them, it does not review them again.
- Anything report 2 recorded as deliberate: `diag.WriteTool` and `reviewRequest.RepoPath`
  are dead by design with callers due in Epic 2, and `DefaultModels`' real-registry branch
  is unreachable offline by construction.

## Acceptance criteria

- The `done` line's `stop=` renders the model's own reason on the success path and the CLI
  vocabulary on every termination the model never reaches; SPECS § Interfaces documents the
  closed set and the `model … auth=` line; no test in `cli` and `diag` asserts two different
  vocabularies for the same field.
- Every write operation named in CONVENTIONS § Code style is rejected by the repository's
  own `.golangci.yml`, verified by a probe file — including `(*os.File).Write`,
  `(*os.File).WriteString`, `(*os.File).Truncate`, `os.Chmod`, `os.Chtimes`, `os.Truncate`,
  `os.Link` and `os.Root.Chmod`.
- Epic 2's issues state where accumulated turn/cost/elapsed state and `diag.WriteTool` must
  live for the `bounds.check(state)` seam, and that `os.Root` confinement does not build on
  the existing symlink-following `os.Stat`.
- CI fails on an unformatted Go file; `.gitattributes` normalises the tree to LF; the `lint`
  job runs on ubuntu-latest and windows-latest.
- Each of these mutations turns the suite red: `ai.UserText(task)` → `ai.UserText("")`;
  `break` after `FinalText`'s first text block; dropping `+ turn.Usage.CacheRead +
  turn.Usage.CacheWrite`; deleting `c.messages = append(…)`. A thinking block and a tool
  call in a scripted response reach neither stdout nor the report.
- `review <repo>` with stdin on a terminal terminates on Ctrl-C with a `done` line and exit
  `2`; `review … | head -20` exits `0`, `1` or `2` with a `done` line and never `141`; a
  stdout write that fails mid-report leaves the run at exit `2` with the invariant that
  stdout is empty on failure intact.
- Every exit-`2` path writes exactly one prefixed `error:` line naming what was wrong —
  empty argv and bad flags included — no unprefixed line precedes the `done` line,
  `review --help` exits `0`, and a test asserts that the exit-`1` path writes no `error:`
  line at all.
- A stream that completes before cancellation returns its message rather than
  `ctx.Err()`; a scripted `StopReasonAborted` message plus a cancelled context enters the
  interruption guard deterministically, and deleting the guard turns the suite red.
- The repository path is handled as an OS path through `path/filepath`, and every path
  reaching stderr is `/`-separated, lowercase and unpunctuated — asserted by a test that
  runs on both OSes.
- `--prompt " "`, `echo | review <repo>` and `--prompt ""` are usage errors at exit `2`
  with stdout empty; stdin is read under an explicit byte bound whose over-limit case has
  its own named error.
- The `done` line's cost equals the adapter's service-tier-adjusted figure, for `priority`
  and `flex` alike.
- A provider that accepts the connection and then sends nothing terminates within a bounded
  time with a `done` line and exit `2`.
- An error body containing a newline and a well-formed `done` line cannot put a second
  `done` line on stderr; no comment in the tree claims kern-link redacts credentials.
- One importable test harness serves both `internal/cli` and `internal/reviewer`, and no
  mutable package-level seam remains in non-test source.
- `internal/diag`'s tests are table-driven; `formatElapsed` renders a 95-minute run with an
  hour unit, and its sub-minute boundary survives no mutation.

## Dependencies

None. Epic 0 runs before [Epic 2](/epic-2-read-only-agentic-loop/EPIC_2.md) resumes — the
contracts it settles are the ones Epic 2 extends, and three of its issues amend Epic 2's
own plan.

## Context

- [Implementation review report 2](../../REPORT_2.md) — the source of every group here,
  with the evidence and the mutation results behind each.
- [Technical specs](../../planning/SPECS.md) — § Interfaces is amended by the first group.
- [Conventions](../../planning/CONVENTIONS.md) — § Code style is amended by the second.
- [Drift](../../planning/DRIFT.md) — three entries; the two Epic 1 entries were excluded
  from report 2 by construction and are not re-litigated here.

## Notes

**Consumed**: [report 2](../../REPORT_2.md) in full — 36 findings extracted, 0 dropped as
stale (no non-docs commit exists between the reviewed range and this epic, and every cited
`file:line` was re-verified), 34 grouped into the 15 groups above, 2 recorded as
`won't-fix` below. [Report 1](../../REPORT_1.md) was consumed earlier, by the revision
recorded in [log](/log.md) on 2026-08-09.

**Three questions were decided at triage**, not left to the implementer:

- The `stop=` field carries both vocabularies rather than one, because the two answer
  different questions and neither document has to be contradicted for it to be true.
- SPECS documents the closed stop-reason set. It is a union that grows per epic — Epic 2
  adds at least `bounds` — and a calling skill cannot parse a set that is written nowhere.
- CONVENTIONS is the source of truth for the forbidden write-API list; `.golangci.yml`
  mirrors it. That is how every other convention in this repository is enforced, and it
  makes the criterion checkable.

**`won't-fix`, with reasons:**

- *No red phase is visible in any of Epic 1's five PRs* (report 2, P2). In every PR the
  tests and the implementation they exercise landed in the same commit, and merges are
  unsquashed so this is the real branch history — but it does not *prove* the red phase was
  skipped, no code change would discharge it, and it cannot be verified retroactively. It
  is real feedback for the pipeline rather than work: if the convention is to be
  enforceable, `implement-issue` has to commit the red phase separately.
- *`diag.WriteTool` and `reviewRequest.RepoPath` are dead code* (report 2, P3). Deliberate,
  scoped by issues 03 and 02 respectively, with callers due in Epic 2.

**One finding became drift rather than work.** The no-files test compares file mtimes but
not directory mtimes — defensible, because two consecutive walks on Windows already report
moved directory mtimes, and documented in the test — but it narrows issue 05's literal
criterion. It is recorded in [DRIFT](../../planning/DRIFT.md) and the criterion corrected,
so the next reviewer finds the decision instead of the divergence.
