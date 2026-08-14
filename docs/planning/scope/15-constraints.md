---
type: Decision
title: "Constraints"
description: "What externally imposed limits bound this project?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 15
slug: constraints
status: decided
verdict: "Four binding constraints: Go, Windows-first, user-billed unsupervised spend, unproven kern-link multi-turn"
decided_via: triage
depends_on: []
---

# Question

What limits are imposed from outside and bound what `define-specs` may later choose —
deadline, budget, mandated platform or technology, compliance?

# Options

Constraints are found, not chosen; the option here is how tightly to state them.

- **State the four that actually bind** — language, host platform, cost model, and the
  dependency's own maturity.
- **State none** — treat it as unconstrained. Leaves `define-specs` free to pick a
  stack that cannot use `kern-link`, which would be a rediscovery, not a choice.

# Recommendation

**State the four that bind:**

- **Go.** `kern-link` is a Go library (`module github.com/julienlegoux/kern-link`,
  `go 1.25.0`) and carries the provider coverage, credential resolution, streaming,
  tool-call types and cost accounting this project refuses to rebuild. Building on it
  is a concept-level decision, so the language follows from it rather than being open.
- **Windows is the development machine, and the binary must not assume POSIX.** Path
  handling, config location and any process invocation have to work on Windows first
  and still work elsewhere — the pipeline's own skills run here.
- **Runs on the user's own credentials, billed per call, unsupervised.** There is no
  service budget and no shared key; every design choice that spends tokens spends the
  user's money inside a script nobody is watching. This is what makes the caps in
  [loop bounds](/scope/08-loop-bounds-and-termination.md) a constraint rather than a
  nicety.
- **`kern-link`'s multi-turn support is unproven here.** Its documented example covers
  a single round trip; the loop is this project's addition. Fixes land upstream in a
  repo the author also owns — an advantage, but the coupling is real and belongs in
  planning.

No deadline, and no compliance regime: the tool reads local repositories the user
already has open.

# Verdict

**Accepted as recommended:** the four constraints above bind, and `define-specs` must
choose within them. No deadline and no compliance regime.

**Refreshed after [loop bounds](/scope/08-loop-bounds-and-termination.md) was decided.**
The third constraint — unsupervised, user-billed spend — no longer implies caps in v1.
It arrives with Milestone 3, when the pipeline starts invoking the binary with no human
present; through Milestones 1 and 2 the author launches it by hand and can interrupt.
The constraint is unchanged; what it demands, and when, has moved.
