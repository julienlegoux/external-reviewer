---
type: Issue
title: "Carry Epic 1's structural warnings into Epic 2's issues"
description: "Amend Epic 2's loop and confinement issues so they state where the bounds seam's accumulated state must live, how diag.WriteTool is reached from the loop, and that os.Root confinement does not build on the symlink-following os.Stat."
tags: [epic-0]
timestamp: 2026-08-11T03:44:09Z
epic: 0
issue: 03
slug: epic-2-plan-amendments
size: S
status: pr-open
gh_issue: 25
gh_pr: 35
resource: https://github.com/julienlegoux/external-reviewer/issues/25
depends_on: []
---

# Carry Epic 1's structural warnings into Epic 2's issues

## Summary

Epic 1's review turned up two facts that are not defects in Epic 1's own terms but that
Epic 2 walks straight into. Epic 1 put the per-turn round trip in `internal/reviewer` and
the accumulation of its numbers in `internal/cli`; Epic 2's `bounds.check(state)` seam
needs those totals at the *top* of each turn, where they do not exist — and
`diag.WriteTool`, which Epic 2 must call from inside the loop, is reachable only from
`cli`. Separately, the `os.Stat` at `internal/cli/review.go:171` follows symlinks and is a
classic TOCTOU gap, inert today because `req.RepoPath` is never read again, but not inert
once `os.OpenRoot` confinement lands on top of it.

Both belong in the issues rather than in a report, so the implementer reads them instead
of discovering them. This is a docs-only PR: no product code changes.

## Scope

- `docs/epics/epic-2-read-only-agentic-loop/issues/03-multi-turn-loop-and-dispatch.md`
  (GitHub [#11](https://github.com/julienlegoux/external-reviewer/issues/11)) states:
  - the accumulated turn count, cost and elapsed wall clock that `bounds.check(state)`
    reads at the top of each turn live in `internal/cli` today — `recordTurn`,
    `internal/cli/review.go:88-105` — while the loop lands in `internal/reviewer`
    (`Conversation.Next`, `internal/reviewer/run.go:61-98`), so the issue names which side
    of the seam they move to rather than leaving it to be discovered mid-PR;
  - `diag.WriteTool` (`internal/diag/diag.go:40-43`) has exactly one caller today (its own
    test) and is reachable only from `cli`, while Epic 2 reads tool calls off the wire
    inside `Conversation.Next` — the empty `for range stream.Events(ctx)` at
    `internal/reviewer/run.go:73-74` is explicitly reserved for it — so the issue says how
    the `tool` line gets written from inside the loop.
- `docs/epics/epic-2-read-only-agentic-loop/issues/01-allow-list-and-root-confinement.md`
  (GitHub [#9](https://github.com/julienlegoux/external-reviewer/issues/9)) states that the
  `os.OpenRoot` work does not build on `internal/cli/review.go:171`'s `os.Stat`: it follows
  symlinks (a symlink to a directory passes `IsDir()`) and the check is a TOCTOU gap.
- Both GitHub issue bodies re-synced with `gh issue edit <n> --body-file <file>` so an
  implementer reading GitHub sees the amendment, not just the repository.
- Both `.md` files' `timestamp` refreshed; an entry appended to `docs/epics/log.md`.

## Out of scope

- Implementing anything in Epic 2.
- Re-sizing, re-splitting or re-ordering Epic 2's issues — both stay `L` and keep their
  numbers, `gh_issue` and `status: open`.
- Changing Epic 2's acceptance criteria beyond what these two warnings require.
- `EPIC_2.md` itself: its Scope already names the `bounds.check(state)` seam, and the
  amendments belong at issue granularity.
- The path-handling repair that gives Epic 2 a wire-path precedent to copy — that is
  [issue 08](/epic-0-skeleton-hardening/issues/08-os-and-wire-paths.md) of this epic,
  product code, not a plan amendment.

## Acceptance criteria / Definition of done

- [ ] Epic 2 issue 03 names where the accumulated turn/cost/elapsed state must live for
      `bounds.check(state)` to read it at the top of each turn, citing
      `internal/cli/review.go:88-105` and `internal/reviewer/run.go:61-98` as the two sides
      of the seam, and states which side wins.
- [ ] Epic 2 issue 03 names `diag.WriteTool` (`internal/diag/diag.go:40-43`) as reachable
      only from `cli` today and says how the `tool` line is written from inside the loop.
- [ ] Epic 2 issue 01 names `internal/cli/review.go:171`'s `os.Stat` as symlink-following
      and a TOCTOU gap, and states that the `os.OpenRoot` confinement does not build on it.
- [ ] `gh issue view 11` and `gh issue view 9` show the amended bodies — the GitHub copy
      and the `.md` do not disagree.
- [ ] Both `.md` files' `timestamp` is refreshed and no other frontmatter field changes:
      `status` stays `open`, `gh_issue` and `size` unchanged.
- [ ] `docs/epics/log.md` carries an entry for the amendment.
- [ ] No file under `internal/`, `main.go`, `.golangci.yml` or `.github/` is touched by
      this PR.

## Relevant files / areas

- `docs/epics/epic-2-read-only-agentic-loop/issues/03-multi-turn-loop-and-dispatch.md`
- `docs/epics/epic-2-read-only-agentic-loop/issues/01-allow-list-and-root-confinement.md`
- `docs/epics/log.md`
- Read-only, as the evidence being carried across: `internal/cli/review.go:88-105,171`,
  `internal/reviewer/run.go:61-98,73-74`, `internal/diag/diag.go:40-43`

Source findings: [report 2](../../../REPORT_2.md) § "Accounting lives on the wrong side of
the seam Epic 2's loop needs" and its P3 row on `os.Stat`.

## Dependencies

- Blocked by: none — docs only, and deliberately early, since Epic 2 resumes on these
  issues the moment this epic retires.
- Blocks: nothing in this epic.

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
Sized **S**: two issue bodies amended in place, two `gh issue edit` calls, one log entry.
