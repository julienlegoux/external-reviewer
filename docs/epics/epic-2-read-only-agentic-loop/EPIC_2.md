---
type: Epic
title: "Read-only agentic loop"
description: "The multi-turn loop and four confined read-only tools that turn a prompt-pipe into a reviewer that chooses what to read, plus the instrumentation Epic 3's caps are sized from."
tags: [epic]
timestamp: 2026-08-12T19:05:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/2
epic: 2
slug: read-only-agentic-loop
status: done
gh_issue: 2
milestone: 2
source: docs/planning/SCOPE.md#milestone-2-read-only-agentic-loop
---

# Epic 2: Read-only agentic loop

## Goal

Turn a prompt-pipe into a reviewer that chooses what to read. This is the reason the
loop lives inside the binary rather than being replaced by evidence the reviewed party
pre-selected: a party under review that assembles the evidence copies its own blind
spots into the selection, and truncates silently once the material outgrows the context.

It also produces the epic's second deliverable, which is not code: **measurements**.
Nobody knows yet how many turns a review takes or how long it runs, and both Epic 3's
caps and SCOPE's third success criterion depend on numbers that only exist after this
epic runs for real.

## Scope

- **Four native, in-process read-only tools**, all scoped by the allow-list, all
  returning plain text with repo-relative `/`-separated paths in both directions on
  every platform: `list` (glob), `read_file` (batched, with `offset`/`limit`, a bad path
  in a batch reported inline while the rest still return), `search` (regex with
  surrounding context), and `git_read` over the fixed allowlist `log`/`diff`/`show`/
  `status` with `args` as an array, never a string.
- **Confinement as an allow-list passed per invocation.** `review` requires at least one
  `--allow <relpath>`; omitting it is a usage error, not an implicit run over the whole
  tree. `--allow .` grants everything but has to be said. Enforcement is
  `os.OpenRoot(repoPath)` at startup, and every read goes through that `*os.Root`.
- `git_read` confined too: subcommand allowlisted *before* the command is built, never a
  shell, `GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` neutralised, allowed subtrees passed as
  pathspecs to `log`/`diff`/`status`, and `show`'s `<rev>:<path>` form path-validated.
  No subcommand is exempt: an unscoped `status` reports the working-tree state of the
  whole repository, which leaks every path name outside the allow-list.
- The small non-configurable sensitive-file floor inside allowed subtrees (`.env*`,
  `*.pem`, `*.key`, `id_rsa*`, `*.p12`, `*.pfx`, `.npmrc`, `.netrc`, `credentials*`,
  `.git/`), with refusals naming which rule refused.
- Enumeration via `git ls-files -z` (plus `--others --exclude-standard`) for exact
  gitignore semantics, falling back to `fs.WalkDir` outside a git repository.
- **Truncation always announced** (`[truncated: showing 200 of 431 matches]`).
- **The multi-turn loop**: `StreamSimple` + `Stream.Result(ctx)` per turn over an
  accumulating `[]ai.Message`, with termination checked in order — bound exceeded, then
  context cancellation (`SIGINT`), then `StopReason != StopReasonToolUse`.
- The `bounds.check(state)` seam at the top of every turn, present **with its values
  unset**, watching turns, accumulated cost and elapsed wall clock.
- The two failure levels kept distinct: a failing **tool** is a turn, returned as
  `ToolResultMessage{IsError: true}` so the reviewer can try another path; a failing
  **turn** ends the run.
- **Run instrumentation on stderr**: turns taken, tokens, accumulated cost and elapsed
  time, emitted as the run progresses.

## Out of scope

- **Loop cap values.** Deliberately deferred to Epic 3, to be sized from this epic's
  measurements rather than guessed. The seam ships; the numbers don't.
- A coverage statement in the report — dropped along with the caps, and named in SCOPE
  as the reason risk 3 stays unmitigated in v1.
- A general shell, and any subprocess that reads the filesystem on its own authority.
  Ripgrep was reconsidered and rejected on exactly this point.
- Tier selection, config and family exclusion — Epic 3.

## Acceptance criteria

- A review completes across multiple turns with the model choosing its own reads, the
  tool results feeding back in, and the run ending on the model's own stop reason.
- `review` invoked without `--allow` is a usage error at exit `2`.
- A path outside the allowed subtrees is refused — including traversal, an absolute
  symlink, and a symlink pointing outside the root where the platform supports creating
  one — and the refusal names the rule.
- `git show HEAD:<path>` for a path outside the allow-list is refused; `log`, `diff` and
  `status` return nothing from outside the allowed subtrees.
- A file matching the sensitive-file floor inside an allowed subtree is refused.
- A truncated tool result says so, with both numbers.
- A failing tool call leaves the run alive and lets the model try another path; a
  failing turn ends the run at exit `2`.
- Every terminating path still emits the `done` line from Epic 1.
- Confinement, allow-list and tool-contract behaviour are covered by tests on both
  ubuntu-latest and windows-latest.
- **A real review has been run by hand and its turns, tokens, cost and wall clock
  recorded** — the input Epic 3 is blocked on.

## Dependencies

[Epic 1: Walking skeleton](/epic-1-walking-skeleton/EPIC_1.md) — the process shell, the
output contract and the exit codes this epic runs inside.

## Context

- [Technical specs](../../planning/SPECS.md) — the tool contract, confinement, the tool
  implementation strategy and the loop mechanics.
- [Conventions](../../planning/CONVENTIONS.md) — in particular the enforced write-API
  prohibition and the OS-path/wire-path rule, both of which this epic exercises hardest.

## Notes

- **"Never writes, never commits, never touches GitHub state" is a property of the code
  that exists, not of a filter on a command string.** Only the read half of the
  filesystem API appears anywhere in the codebase, and CI enforces that.
- **Project-wide, and load-bearing here**: Windows is the development machine and
  nothing may assume POSIX. `os.Root` carries the whole confinement requirement —
  escaping names, symlink rules, Windows reserved device names — precisely the list a
  hand-rolled `filepath.Clean` + prefix check would have to reproduce and would fail
  first on Windows.
- **Risk 3 (context overflow on large surfaces) is not mitigated by this epic** — an
  epic's merged diff is the heaviest read in the pipeline and nobody has measured it.
  This epic makes it *observable*; the mitigation is deferred. It has now been
  *observed*: [Hand-run measurements](/epic-2-read-only-agentic-loop/MEASUREMENTS.md)
  records a review of this epic's own merged diff peaking at 98k prompt tokens without
  overflowing, because the per-answer tool caps bind before the context does.
- **The measurements are written down**:
  [Hand-run measurements](/epic-2-read-only-agentic-loop/MEASUREMENTS.md) — four hand
  runs on `openai-codex/gpt-5.5`, their turns, tokens, context, cost and wall clock, and
  what they suggest (not decide) about Epic 3's caps. Epic 3 reads that document rather
  than a memory of this conversation.
- **Risk 5 (a runaway run during hand-run measurement) is accepted deliberately.**
  Through this epic the author launches by hand and can interrupt; the only mitigations
  are live stderr diagnostics and a human at the keyboard. On the current subscription
  credentials the cost of a runaway is quota, not money.
- Retires the remainder of risk 1, and risk 2 (foreign tool-calling fidelity) for the
  one credentialed family, OpenAI via `openai-codex`.
