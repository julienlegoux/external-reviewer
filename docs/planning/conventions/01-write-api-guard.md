---
type: Decision
title: "Write-API prohibition as an enforced rule"
description: "Is 'the write half of the filesystem API is not in the codebase' a habit, or something CI fails on?"
tags: [decision, conventions]
timestamp: 2026-08-09T04:08:38Z
phase: conventions
decision: 01
slug: write-api-guard
status: decided
verdict: "A — enforced by lint (forbidigo in the pinned golangci-lint)"
decided_via: triage
depends_on: []
---

# Question

SPECS states the product's central guarantee as a property of the source: *"it cannot
write to the repository because the write half of the filesystem API is not in the
codebase"*, and *"Only the read half of the API appears anywhere in the codebase."* That
is a claim about every future commit, not just the first one. The baseline's linting
convention is "the stack's community-standard ruleset; warnings are errors" — and no
community-standard Go ruleset forbids `os.WriteFile`. So today the guarantee rests on
whoever writes the next PR remembering it, including agent implementers who did not read
SPECS.

The question is whether this repo adds a project-specific enforcement rule to the
baseline's lint convention, and what it covers.

# Options

- **A — Enforced by lint.** `golangci-lint` gains a `forbidigo` block listing the write
  half (`os.Create`, `os.OpenFile`, `os.WriteFile`, `os.Remove*`, `os.Mkdir*`,
  `os.Rename`, `os.Symlink`, `(*os.Root).Create|OpenFile|Remove|Mkdir`, `io.Copy` to a
  file) plus `exec.Command`/`exec.CommandContext` outside the one `git_read` file.
  Non-test files only; tests are exempt (they build fixture repos with `t.TempDir()`).
  Cost: one config block, and an occasional `//nolint` with a reason when the list is
  wrong.
- **B — Enforced by a test.** A Go test walks the non-test source and fails on the same
  symbol list. Same coverage, no linter config, but it is hand-rolled AST or grep work
  the linter already does well.
- **C — Convention only.** Write it in CONVENTIONS.md and rely on review. Zero
  machinery; the guarantee degrades to a promise, which is exactly the distinction SPECS
  spends a paragraph making.

# Recommendation

**A.** The linter is already pinned in CI with `gosec`/`errcheck`/`govet`
([specs 17](/specs/17-distribution-and-ci.md)), so this is a config block in a file that
exists rather than new machinery. It is also the honest form of the SPECS claim: a
structural guarantee that nothing checks is a stylistic preference. `exec.Command`
belongs in the list for the same reason ripgrep was rejected — a subprocess reads on its
own authority, outside the root handle ([specs 11](/specs/11-tool-implementation-strategy.md)),
so the second one to appear should have to argue for itself in a `//nolint` comment.

# Verdict

**A — enforced by lint.** Accepted at triage as recommended. The write half of the
filesystem API and `exec.Command`/`exec.CommandContext` outside `git_read` are a
`forbidigo` block in the already-pinned `golangci-lint` config; `_test.go` files are
exempt, since fixtures are built with `t.TempDir()` and `git init`. An exception needs a
`//nolint:forbidigo` carrying a reason. This makes SPECS' central claim — "the write half
of the filesystem API is not in the codebase" — a property CI verifies on every commit
rather than one each implementer has to remember.
