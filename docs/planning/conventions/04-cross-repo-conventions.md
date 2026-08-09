---
type: Decision
title: "Which conventions govern the cross-repository work"
description: "Milestone 3's issues land in the lx skills repo — do this project's conventions follow them there?"
tags: [decision, conventions]
timestamp: 2026-08-09T04:08:38Z
phase: conventions
decision: 04
slug: cross-repo-conventions
status: decided
verdict: "A — conventions are repo-scoped; the lx skills repo governs its own work"
decided_via: triage
depends_on: [branch-model]
---

# Question

SCOPE carries an explicit cross-repository note: Milestone 3's `review-interfaces.md`
swap lands in the **lx skills repository**, not this one, so "this milestone's issues
carry a different repo than every other milestone"
([scope 13](/scope/13-skills-integration-scope.md)). Those issues will be implemented by
an agent reading *this* project's CONVENTIONS.md, in a repo that is markdown and skill
authoring rather than Go, and that has its own authoring contract, its own branch
model, and no `go test`.

Nothing in the baseline anticipates a conventions doc whose scope is one repo but whose
consumers work in two.

# Options

- **A — Conventions are repo-scoped; the skills repo governs itself.** CONVENTIONS.md
  states up front that it governs `external-reviewer` only, and that Milestone 3 issues
  touching the skills repo follow that repo's own authoring contract and branch model.
  What *does* travel is the issue's acceptance criteria, nothing else.
- **B — Conventions follow the work.** This doc's rules (branch naming, conventional
  commits, PR-closes-one-issue) apply wherever an issue from this project lands. Uniform
  for the implementer; imposes a foreign standard on a repo that already has one, and
  the Go-specific half is meaningless there anyway.
- **C — Silence.** Say nothing and let each implementer work it out. Cheapest to write,
  and it produces the one failure this project can least afford — an agent applying Go
  test-first rules to a markdown edit, or force-pushing a skills branch under this
  repo's rules.

# Recommendation

**A**, with the boundary stated in one sentence at the top of CONVENTIONS.md rather than
buried. The generic half (conventional commits, one PR per issue, short-lived branches)
is near-universal and will likely match the skills repo anyway; the parts that would
actually collide are exactly the parts that should not travel. An implementer who lands
in the other repo needs one instruction — *read that repo's contract* — and this is
where they will look for it.

# Verdict

**A — conventions are repo-scoped; the skills repo governs itself.** Accepted at triage as
recommended. CONVENTIONS.md opens by stating that it governs the `external-reviewer`
repository, and that Milestone 3 issues touching `_shared/review-interfaces.md` in the lx
skills repository follow that repository's own authoring contract, branch model and
review gate. What travels with the issue is its acceptance criteria and nothing else.
