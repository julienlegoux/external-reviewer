---
type: Issue
title: "Read the task prompt under a context, a byte bound and real validation"
description: "Make the stdin read cancellable so SIGINT is observable before the model call, bound it explicitly, and reject whitespace-only and empty-flag prompts before they buy an API call."
tags: [epic-0]
timestamp: 2026-08-11T12:00:00Z
epic: 0
issue: 09
slug: bounded-validated-prompt
size: M
status: pr-open
gh_issue: 31
gh_pr: 46
resource: https://github.com/julienlegoux/external-reviewer/issues/31
depends_on: [8]
---

# Read the task prompt under a context, a byte bound and real validation

## Summary

Three holes meet at one read, `io.ReadAll(stdin)` at `internal/cli/review.go:194`.

**It takes no context, and that makes the binary uninterruptible.**
`signal.NotifyContext(context.Background(), os.Interrupt)` at `internal/cli/run.go:35`
replaces Go's default terminate-on-SIGINT for the whole lifetime of `Run`, and cancellation
is then observed only where a `ctx` is actually consulted — `run`'s entry check, `Resolve`,
and the stream. Stdin is the *documented* invocation form (`internal/cli/usage.go:16-17`:
"the task prompt comes from `--prompt`, or from stdin when `--prompt` is not given"). Run
`external-reviewer review /path/to/repo` with stdin on a terminal: `io.ReadAll` blocks for
EOF, Ctrl-C is delivered to the notify goroutine, cancels a context nobody reads, and the
`read(2)` resumes. Every subsequent Ctrl-C does the same, since `stop()` is deferred to the
end of `Run`. The process hangs indefinitely, emits **no `done` line and returns no exit
code**; only Ctrl-D or `kill` escapes. Before this wiring existed the default handler would
have killed it. This is one of the two terminations that escape the `done`-line seam by not
returning at all — the seam is sound, the hole is beside it.

**It has no size bound.** `cat 8GB.bin | external-reviewer review /repo` buffers the whole
stream into a Go string before any validation; the process is OOM-killed, again with no
`done` line and no exit code. Short of OOM, the entire blob is transmitted verbatim to a
third party — the security surface CONVENTIONS § Dependencies calls out.

**And `if task == ""` is the only validation.** `echo | external-reviewer review /repo`
yields `"\n"`, which is non-empty: the run resolves credentials and sends a one-newline task
to a real model. Real money, real latency, a report generated from no instruction. Same for
`--prompt " "`. The neighbouring test row (`internal/cli/review_test.go:87`) covers only the
exact-empty case, and collapsing the `fs.Visit` presence check to `promptSet := *prompt !=
""` survives mutation, so `--prompt ""` silently changing from "usage error" to "read stdin"
is undetected.

## Scope

- A context-aware stdin read, so cancellation is observable before the model call: the read
  runs where `ctx.Done()` can win (e.g. a goroutine feeding a channel the caller selects
  on), and the run then terminates through the ordinary seam — exit `2`, one `error:` line,
  a `done` line, empty stdout. The `signal.NotifyContext` wiring stays exactly as it is;
  what changes is that something consults the context.
- `io.ReadAll(io.LimitReader(stdin, N))` with an explicit `N` stated in the code with its
  reason, and a **named** over-limit error the classification inspects with `errors.Is` —
  never by message text.
- `strings.TrimSpace(task) == ""` as the emptiness test, covering `--prompt " "` and
  `echo |` alongside the exact-empty case.
- The `--prompt ""` case pinned: the flag-presence check at `internal/cli/review.go:183-188`
  keeps its `fs.Visit` form and a test makes the collapse to `*prompt != ""` fail.
- The trim decides emptiness only. What is sent to the model is the prompt the caller gave,
  not a silently rewritten one — stated in the code and asserted.

## Out of scope

- The stdout half of the `done`-line escapes (a failing write, a closed pipe) —
  [issue 10](/epic-0-skeleton-hardening/issues/10-stdout-write-failures.md).
- Rejecting a non-text or binary prompt beyond the byte bound. The bound is the natural
  place for it, and it is deliberately not taken here.
- `--system`, and the JSON request object on stdin — Epic 3.
- Where the `error:` lines come from — [issue 07](/epic-0-skeleton-hardening/issues/07-single-error-line.md),
  already landed.

## Acceptance criteria / Definition of done

- [ ] Written test-first, black-box, table-driven over the prompt cases.
- [ ] A cancelled context reaching the stdin read abandons it: exit `2`, `stop=interrupted`,
      stdout empty, a `done` line and one `error:` line — asserted in the suite through the
      inner `run`, which takes the context explicitly.
- [ ] `external-reviewer review <repo>` with stdin on a terminal terminates on Ctrl-C with a
      `done` line and exit `2`, verified by hand with the transcript and exit code recorded
      in the PR body. (Delivering a real `SIGINT` is not portable to the Windows runner —
      the same constraint issue 03 of Epic 1 documented beside the wiring.)
- [ ] `--prompt " "`, `--prompt ""` and `echo | review <repo>` are each usage errors at exit
      `2` with stdout empty and exactly one `error:` line.
- [ ] A stdin body over the bound returns a named error, inspected with `errors.Is`, at exit
      `2` with one `error:` line naming the limit; the test drives a reader larger than `N`
      without allocating `N` bytes of fixture.
- [ ] Mutating the presence check to `promptSet := *prompt != ""` turns the suite red,
      recorded in the PR body.
- [ ] A prompt with leading and trailing whitespace around real text still reaches the model
      with the caller's own text — asserted from inside the scripted `faux` step, on issue
      06's pattern.
- [ ] `go test ./... -race` and `golangci-lint run` pass; CI green on ubuntu-latest and
      windows-latest.

## Relevant files / areas

- `internal/cli/review.go:183-206` (the presence check, the stdin read, the emptiness test)
- `internal/cli/run.go:35` (the `signal.NotifyContext` wiring — read, not changed)
- `internal/cli/usage.go:16-17` (the documented stdin form)
- `internal/cli/review_test.go:87` (the exact-empty row this table joins),
  `internal/cli/exit.go` (the typed errors the new one joins)

Governing decisions: [SPECS § Interfaces](../../../planning/SPECS.md) (the `done` line on
every termination path), [CONVENTIONS § Error handling](../../../planning/CONVENTIONS.md)
(typed classification), [CONVENTIONS § Dependencies](../../../planning/CONVENTIONS.md)
L155-156 (the supply-chain and exposure surface). Source findings:
[report 2](../../../REPORT_2.md) § "SIGINT is trapped but unobservable before the model
call", § "A whitespace-only prompt passes validation and buys a real API call", §
"`io.ReadAll(stdin)` has no size bound", and the P3 row on the `fs.Visit` collapse.

## Dependencies

- Blocked by: [08 — OS and wire paths](/epic-0-skeleton-hardening/issues/08-os-and-wire-paths.md)
  — the same function, `runReviewCommand`, and landing them in order keeps the two edits
  from colliding.
- Blocks: nothing.

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
Sized **M**: ~250 lines — the cancellable read is the only structural piece; the bound, the
trim and the presence check are small, and the tests carry the rest.
