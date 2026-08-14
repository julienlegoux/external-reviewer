---
type: Decision
title: "Doc comments on exported identifiers"
description: "Does an internal-only Go codebase owe godoc on everything it exports?"
tags: [decision, conventions]
timestamp: 2026-08-09T04:08:38Z
phase: conventions
decision: 08
slug: doc-comment-policy
status: decided
verdict: "A — baseline wins inside internal/; package comments and tool descriptions excepted"
decided_via: triage
depends_on: []
---

# Question

The baseline is strict and deliberately anti-narration: comments exist "only for
constraints the code can't show (invariants, workarounds with links, non-obvious 'why')
— never narration of what the next line does." Go's own convention pulls the other way:
every exported identifier gets a doc comment starting with its name, and `revive`/
`golint`-style rules enforce it.

They collide here in a specific way. Everything except `main` lives under `internal/`
([specs 01](/specs/01-language-module-toolchain.md)), so nothing is exported to anyone —
there is no published API and no godoc reader. Applying Go's convention literally
produces `// Read reads.` on dozens of identifiers, which is exactly the narration the
baseline bans, and it teaches an agent implementer that a comment is a box to fill.

# Options

- **A — Baseline wins inside `internal/`.** No blanket doc-comment requirement; a
  comment appears where it carries a constraint. Where one is written on an exported
  identifier it still follows Go form (starts with the name, full sentence), so nothing
  reads as foreign. Two places always get one: each `internal/` package gets a package
  comment saying what it owns, and every tool's `ai.Tool` description — which is
  model-facing contract, not commentary — stays beside its implementation, as SPECS
  requires.
- **B — Go convention wins.** Doc comment on every exported identifier, enforced by the
  linter. Uniform, familiar to any Go reader, and it fills the codebase with restated
  names.
- **C — Go convention on a subset.** Required on types and functions, optional on
  methods and fields. A line that has to be re-litigated at every review.

# Recommendation

**A.** The Go convention earns its keep by serving godoc readers of a public package,
and this module has none by construction — the exception it makes for package comments
and tool descriptions covers the two cases where a reader genuinely arrives cold. It
also matters for who writes this code: an agent implementer with a "document every
export" rule produces plausible noise at scale, while a "comment only what the code
can't show" rule produces nothing when there is nothing to say, which is correct.

# Verdict

**A — baseline wins inside `internal/`.** Accepted at triage as recommended. No blanket
doc-comment requirement and no linter rule demanding one; a comment appears where it
carries a constraint the code cannot show. Comments that *are* written on exported
identifiers still follow Go form, starting with the identifier's name. Two exceptions
always apply: every `internal/` package carries a package comment stating what it owns,
and each tool's `ai.Tool` description — model-facing contract rather than commentary —
lives beside its implementation, as SPECS requires.
