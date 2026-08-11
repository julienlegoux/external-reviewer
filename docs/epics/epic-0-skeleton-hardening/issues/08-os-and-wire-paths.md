---
type: Issue
title: "Handle the repository path as an OS path and emit wire paths on stderr"
description: "Establish the OS-path/wire-path boundary CONVENTIONS decided and no product file implements, so Epic 2's --allow arguments arrive with a precedent to copy."
tags: [epic-0]
timestamp: 2026-08-11T03:33:08Z
epic: 0
issue: 08
slug: os-and-wire-paths
size: M
status: open
gh_issue: 30
resource: https://github.com/julienlegoux/external-reviewer/issues/30
depends_on: [7]
---

# Handle the repository path as an OS path and emit wire paths on stderr

## Summary

CONVENTIONS § Paths and platforms legislates two vocabularies: OS paths use `path/filepath`
and never cross the model boundary; wire paths — tool input and output, `--allow`
arguments, **stderr diagnostics** — are `/`-separated on every platform, converted at the
boundary with `filepath.ToSlash` / `filepath.FromSlash`. `path/filepath` appears in **zero**
product files. The positional repository argument goes raw into `os.Stat` and raw back out
to stderr through `%q`.

On Windows that produces:

```
error: repository path "C:\\no\\such\\repo": GetFileAttributesEx C:\no\such\repo: The system cannot find the path specified.
```

Three breaches in one line: backslashes on a stderr diagnostic, `%q` double-escaping them,
and the OS's capitalised, full-stopped sentence spliced into a message CONVENTIONS requires
lowercase and unpunctuated. The same text differs on Linux. Issue 02 of Epic 1 ticked the
criterion — *"Repository paths are handled with `path/filepath`/`os.Stat` as OS paths"* —
and the two-OS matrix did not catch it because no test asserts on a path, exactly the case
CONVENTIONS legislates for: *"A test that asserts on a path asserts the wire form."*

Epic 2's `--allow` arguments and `os.Root` confinement are *the* wire-path boundary. They
arrive with no precedent to copy and a stderr formatter demonstrating the wrong habit.

## Scope

- The positional repository path handled as an OS path through `path/filepath`
  (`internal/cli/review.go:169-181`), so the one path this binary owns goes through the
  package CONVENTIONS names.
- Conversion at the boundary: every path that reaches stderr goes out in wire form —
  `/`-separated via `filepath.ToSlash`, quoted in a way that does not double-escape
  separators.
- The message text around it lowercase, unpunctuated, and stating what was attempted. The
  OS's own sentence is no longer spliced into it; classification stays on the wrapped
  error, inspected with `errors.Is` / `errors.As` at the `run()` boundary, never by
  matching message text.
- The wrapped error at `internal/cli/review.go:175` gets the same treatment as the stderr
  line it accompanies.
- One test that asserts on the wire form and runs on **both** matrix OSes — the assertion
  CONVENTIONS requires and the epic has never had.

## Out of scope

- `--allow`, `os.OpenRoot` and tool-output paths — Epic 2 issues 01 and 02. This PR
  establishes the habit; it does not add the flag.
- Replacing the `os.Stat` check itself with a root-based one. That is Epic 2's, and
  [issue 03](/epic-0-skeleton-hardening/issues/03-epic-2-plan-amendments.md) of this epic
  is what warns it not to build on the symlink-following `Stat`.
- Making the `error:` line's prefix and routing uniform —
  [issue 07](/epic-0-skeleton-hardening/issues/07-single-error-line.md), already landed;
  this PR changes what those lines contain, not where they come from.

## Acceptance criteria / Definition of done

- [ ] Written test-first, black-box, table-driven over the path cases.
- [ ] `path/filepath` is imported by at least one non-test product file and the repository
      path flows through it: `grep -rn "path/filepath" internal --include='*.go'` filtered
      to non-`_test.go` files is non-empty.
- [ ] A test that runs on **both** matrix OSes asserts the stderr line for a non-existent
      repository path: the path is `/`-separated, no `\` appears anywhere in the line, and
      no `\\` (no double-escaped separator).
- [ ] The wording around the path is lowercase and carries no trailing full stop, and the
      OS's own error sentence does not appear on stderr.
- [ ] The same holds for the non-directory case (`review <path-to-a-file>`).
- [ ] Classification is unchanged: both paths still exit `2` with an empty stdout, a
      `done` line, and exactly one `error:` line, and the wrapped error is still inspected
      by type rather than by text.
- [ ] The assertion is written once and passes unchanged on ubuntu-latest and
      windows-latest — no `//go:build` divergence and no `runtime.GOOS` branch in the
      expectation.
- [ ] `go test ./... -race` and `golangci-lint run` pass; CI green on both OSes.

## Relevant files / areas

- `internal/cli/review.go:169-181` (the positional path, `os.Stat`, and the two stderr
  lines and two wrapped errors around it)
- `internal/cli/review_test.go` — the parsing table these cases join
- `internal/diag/diag.go` — `WriteError` from issue 07, the writer these lines go through

Governing decisions:
[CONVENTIONS § Paths and platforms](../../../planning/CONVENTIONS.md) L72-79,
[CONVENTIONS § Error handling](../../../planning/CONVENTIONS.md) L134-135,
[conventions 02 — cross-platform path rules](../../../planning/conventions/02-cross-platform-path-rules.md).
Source finding: [report 2](../../../REPORT_2.md) § "The OS-path/wire-path split was never
established, and issue 02's criterion was ticked without being met".

## Dependencies

- Blocked by: [07 — One prefixed `error:` line](/epic-0-skeleton-hardening/issues/07-single-error-line.md)
  — it rewrites the same stderr sites, and this PR changes their content on top.
- Blocks: [09 — Bounded, validated prompt](/epic-0-skeleton-hardening/issues/09-bounded-validated-prompt.md),
  which edits the same function.

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
Sized **M**: ~200 lines — a small production change and the two-OS wire-form test that is
the reason the issue exists.
