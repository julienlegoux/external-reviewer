---
type: Issue
title: "Add the confined git_read tool"
description: "Add git_read over the fixed log/diff/show/status allowlist with args as an array, allowed subtrees passed as pathspecs and show's <rev>:<path> form validated before the command is built."
tags: [epic-2]
timestamp: 2026-08-12T08:15:33Z
epic: 2
issue: 06
slug: git-read-tool
size: L
status: done
gh_issue: 14
gh_pr: 55
resource: https://github.com/julienlegoux/external-reviewer/issues/14
depends_on: [3]
---

# Add the confined git_read tool

## Summary

History is reachable only through this tool, which is why `.git/` is refused to the
other three. It closes a hole that would otherwise make every file-level control
decorative: `git show HEAD:.env` reads any file in the repository's history through the
one tool that never touches `*os.Root`. So `show`'s `<rev>:<path>` form is validated
against the allow-list and the floor **before the command is built**, and `log`/`diff`
take the allowed subtrees as pathspecs.

## Scope

- Tool `git_read`, parameters `command` (∈ `log`, `diff`, `show`, `status`) and `args`
  (**array of strings, never a string** — a string implies a shell, and there is no
  shell). The array is passed as argv.
- The subcommand is validated against the four-item allowlist *before* the command is
  built; a refused subcommand never reaches `git`, and the refusal names what is allowed.
- Built on issue 02's exec helper: `exec.CommandContext(ctx, "git", "-C", repoPath, sub,
  args...)`, never a shell, never a joined string parsed back into arguments — which
  matters most on Windows, where argument splitting is a `CommandLineToArgvW` problem.
  `GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` stay neutralised.
- Confinement per subcommand:
  - `log`, `diff` — the allowed subtrees appended as pathspecs (`-- <paths>`), so history
    is scoped the way the working tree is;
  - `show` — the `<rev>:<path>` form parsed and its path validated against the allow-list
    and the floor before the command is built; a bare `<rev>` (a commit) is allowed, and
    its diff is pathspec-scoped the same way;
  - `status` — the allowed subtrees appended as pathspecs (`git status -- <paths>`),
    exactly as `log` and `diff` are. An unscoped `status` reports the working-tree state
    of the *whole* repository, which hands the model the names and modification state of
    every path outside the allow-list — including the ones the sensitive-file floor
    exists to hide.
- Result is the command's stdout verbatim, **capped at 2000 lines**, with truncation
  always announced — `[truncated: showing 2000 of 6431 lines]`.
- A non-zero `git` exit, or `git` absent from `PATH`, returns a tool error carrying
  git's stderr — a message the reviewer can act on, not a run failure.
- An `args` entry that would be interpreted as an option overriding the confinement
  (e.g. an injected `--` or a pathspec outside the allowed subtrees) is refused with a
  rule-naming message.
- Its `ai.Tool` declaration beside the implementation, `snake_case` schema fields, with a
  standalone description naming the four subcommands and stating that `args` is an array.

## Out of scope

- Any subcommand beyond the four, including read-shaped ones like `blame` or `ls-tree`.
  The allowlist is fixed; widening it is a SPECS change, not a PR decision.
- Any write-side git operation. `exec.Command` is forbidden outside this implementation
  and issue 02's enumeration helper, and CI enforces it.
- Parsing or reformatting git's output. It is returned verbatim.
- Enumeration via `git ls-files` — issue 02 owns that, and this PR reuses its helper.

## Acceptance criteria / Definition of done

- [ ] Written test-first and black-box against throwaway `git init` repositories with
      scripted commits in `t.TempDir()`; `t.Skip`ped at runtime with a message naming
      `git` when it is absent — never a build tag.
- [ ] `status`, `log`, `diff` and `show` each return git's stdout verbatim for a fixture
      repository.
- [ ] **`git show HEAD:.env` is refused**, with a message naming the denied-filename
      rule, and a test asserts `git` was never invoked for it.
- [ ] `show HEAD:<path>` for a path inside the root but outside the `--allow` subtrees is
      refused with a message naming the allow-list rule — a *different* message from the
      floor refusal above.
- [ ] `show HEAD:<path>` for an allowed path returns the file's contents at that
      revision.
- [ ] `log` and `diff` over a repository whose history touches both an allowed and a
      disallowed subtree return **no** entries from the disallowed one, proving the
      pathspecs are applied.
- [ ] `status` in a working tree with an uncommitted change in an allowed subtree **and**
      one in a disallowed subtree reports the first and **not** the second — the same
      pathspec scoping as `log` and `diff`, asserted separately because an unscoped
      `status` is the one form that leaks path names without reading a file.
- [ ] A subcommand outside the allowlist (`commit`, `push`, `blame`) is refused with a
      message listing the four allowed subcommands, and never reaches `git`.
- [ ] `args` supplied as a string rather than an array is a schema violation returned as
      a tool error.
- [ ] An `args` entry attempting to widen the pathspec scope is refused with a
      rule-naming message.
- [ ] A non-zero `git` exit returns a tool error carrying git's stderr, and — asserted
      end-to-end through issue 03's loop — leaves the run alive.
- [ ] Output exceeding the 2000-line cap ends with a
      `[truncated: showing 2000 of N lines]` line carrying both real numbers; output
      under the cap carries no truncation line.
- [ ] `golangci-lint run` passes; the `exec` call is the one already carrying its
      `//nolint:forbidigo` reason from issue 02, or a second one carrying its own. CI
      green on ubuntu-latest and windows-latest.

## Relevant files / areas

No existing code for this yet. Expected paths:

- `internal/tools/git_read.go`, `internal/tools/git_read_test.go`
  (`package tools_test`)
- `internal/repo/git.go` — the exec helper from issue 02, called, not re-implemented
- `internal/tools/registry.go` — registration

Governing decisions:
[specs 10 — Repository confinement](../../../planning/specs/10-repository-confinement.md)
(the `git show HEAD:.env` hole),
[specs 11 — Tool implementation strategy](../../../planning/specs/11-tool-implementation-strategy.md),
[specs 09 — Read-only tool contract](../../../planning/specs/09-read-only-tool-contract.md),
[CONVENTIONS § Code style & formatting](../../../planning/CONVENTIONS.md) (the
`exec.Command` prohibition and its one exception).

## Dependencies

- Blocked by: [03 — Drive the multi-turn loop](/epic-2-read-only-agentic-loop/issues/03-multi-turn-loop-and-dispatch.md)
  (and through it [02](/epic-2-read-only-agentic-loop/issues/02-enumeration-and-list-tool.md),
  whose exec helper this reuses).
- Blocks: [07 — Run a real review by hand](/epic-2-read-only-agentic-loop/issues/07-hand-run-measurements.md).

## PR size note

Sized **L**: target ~700 changed lines — the subcommand allowlist, per-subcommand
pathspec scoping for all four, the `<rev>:<path>` parser and their tests against scripted
`git init` fixtures. It has no natural split line: the `show` validation and the pathspec
scoping are the same confinement decision expressed twice, and splitting them ships a
tool whose history hole is open for one PR. If it passes ~1000, the tests are what move —
into a second PR of fixtures, never the confinement.
