---
type: Issue
title: "Enumerate the repository through git ls-files and add the list tool"
description: "Add gitignore-exact enumeration via git ls-files with an fs.WalkDir fallback, re-resolved through the root, and the first tool — list — with its declaration and announced truncation."
tags: [epic-2]
timestamp: 2026-08-09T06:32:00Z
epic: 2
issue: 02
slug: enumeration-and-list-tool
size: M
status: open
gh_issue: 10
resource: https://github.com/julienlegoux/external-reviewer/issues/10
depends_on: [1]
---

# Enumerate the repository through git ls-files and add the list tool

## Summary

`list` and `search` must agree on what the repository contains, so enumeration is one
mechanism shared by both, and it is gitignore-exact because a walk that wanders into
`node_modules/` fills the reviewer's context with vendored code it will read and reason
about. This PR adds that enumeration — `git ls-files`, with an `fs.WalkDir` fallback
outside a git repository — and the first tool built on it, `list`.

It also lands the one `exec` helper this codebase is allowed to have: `git` invoked with
argv, never a shell, with machine-level git config neutralised. Issue 06 reuses it.

## Scope

- Enumeration: `git ls-files -z` for tracked files plus
  `git ls-files -z --others --exclude-standard` for untracked-but-not-ignored ones,
  merged and de-duplicated. `-z` because git quotes unusual filenames otherwise.
- A `git` exec helper: `exec.CommandContext(ctx, "git", "-C", repoPath, …)` with argv
  never a joined string, `GIT_CONFIG_GLOBAL` and `GIT_CONFIG_SYSTEM` neutralised so a
  machine-level alias, pager or `include` cannot change what a subcommand does. It
  carries the single `//nolint:forbidigo` this epic adds, with its reason
  (CONVENTIONS § Code style & formatting).
- Fallback when `git` is absent or the path is not a git repository: `fs.WalkDir` over
  `Root.FS()` with `.git/` skipped — degraded, not broken, and reported once as a `warn`
  line on stderr so a thin review is explained rather than inferred.
- **Every path git reports is re-resolved through the `*os.Root`** from issue 01 before
  it is used or returned; a path git lists but the root refuses is skipped, not trusted.
  Paths outside the allow-list and paths matching the floor are excluded from results.
- The `list` tool: parameter `pattern` (glob, repo-relative, default `**/*`), result is
  matching paths one per line, repo-relative and `/`-separated. Truncation announced —
  `[truncated: showing 200 of 431 paths]`.
- Its `ai.Tool` declaration — name, description and `ai.JSONSchema(...)` literal — lives
  beside the implementation, with `snake_case` names and schema fields because they are a
  wire contract with the model, not Go API. The description stands alone: what the tool
  does, what its paths look like, and what truncation means, since the caller owns the
  system prompt and this may be the model's only instruction.
- A tool registry the later tool issues append to and issue 03's loop declares from.

## Out of scope

- `read_file`, `search`, `git_read` — issues 04, 05 and 06.
- `git_read`'s subcommand allowlist and pathspec confinement — issue 06, even though the
  exec helper lands here.
- The loop and tool dispatch — issue 03. This PR's tests call the tool's handler
  directly.
- Caching enumeration between calls. One review per process, sequential tool execution,
  no caching (SPECS § Distribution & operations).

## Acceptance criteria / Definition of done

- [ ] Written test-first and black-box, against throwaway `git init` repositories built
      in `t.TempDir()`, per CONVENTIONS § Testing.
- [ ] In a repository with a `.gitignore` containing `node_modules/`, enumeration returns
      the tracked files and an untracked-but-not-ignored file, and **does not** return
      anything under `node_modules/`.
- [ ] A file whose name needs quoting under git's default `core.quotepath` (e.g. a
      non-ASCII name) is returned intact, proving `-z` is honoured.
- [ ] Tests that need `git` are `t.Skip`ped at runtime with a message naming `git` when
      it is absent from `PATH` — never a build tag.
- [ ] In a `t.TempDir()` that is **not** a git repository, enumeration falls back to
      `fs.WalkDir`, returns the files, omits `.git/`, and the run emits one `warn` line
      saying enumeration is degraded.
- [ ] `list` with the default pattern returns only paths inside the `--allow` subtrees;
      a file inside the root but outside them is absent from the result.
- [ ] `list` never returns a path matching the sensitive-file floor, including with
      `--allow .`.
- [ ] `list` with a pattern matching more than the cap ends with a
      `[truncated: showing N of M paths]` line carrying both real numbers.
- [ ] Every path in every `list` result is `/`-separated and repo-relative, asserted on
      both matrix OSes.
- [ ] `GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` neutralisation is asserted: a test writes a
      config file setting a hostile value, points the environment at it, and shows
      enumeration is unaffected.
- [ ] `golangci-lint run` passes; the one `//nolint:forbidigo` on the exec helper carries
      its reason, and no other forbidden call is introduced.
- [ ] CI green on ubuntu-latest and windows-latest.

## Relevant files / areas

No existing code for this yet. Expected paths:

- `internal/repo/enumerate.go`, `internal/repo/git.go` (the exec helper),
  `internal/repo/enumerate_test.go` (`package repo_test`)
- `internal/tools/list.go`, `internal/tools/list_test.go`, `internal/tools/registry.go`

Governing decisions:
[specs 11 — Tool implementation strategy](../../../planning/specs/11-tool-implementation-strategy.md),
[specs 09 — Read-only tool contract](../../../planning/specs/09-read-only-tool-contract.md),
[SPECS § Reading the repository](../../../planning/SPECS.md),
[CONVENTIONS § Naming](../../../planning/CONVENTIONS.md) (wire-contract `snake_case`),
[CONVENTIONS § Documentation & comments](../../../planning/CONVENTIONS.md) (tool
declarations live beside their implementation).

## Dependencies

- Blocked by: [01 — Confine the run with --allow and os.OpenRoot](/epic-2-read-only-agentic-loop/issues/01-allow-list-and-root-confinement.md).
- Blocks: [03 — Drive the multi-turn loop](/epic-2-read-only-agentic-loop/issues/03-multi-turn-loop-and-dispatch.md),
  [05 — search](/epic-2-read-only-agentic-loop/issues/05-search-tool.md) (enumeration),
  [06 — git_read](/epic-2-read-only-agentic-loop/issues/06-git-read-tool.md) (the exec
  helper).

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
