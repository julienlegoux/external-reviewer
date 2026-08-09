---
type: Decision
title: "Loop bounds and termination"
description: "What stops the agent loop, and what does it emit when it stops?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 08
slug: loop-bounds-and-termination
status: decided
verdict: "Unbounded and instrumented in Milestones 1–2; caps deferred to Milestone 3, sized from measured runs"
decided_via: discussion
depends_on: [read-only-tool-set]
---

# Question

An agentic reader that chooses its own path can also loop, stall, or spend without
limit — on the user's own API credit. [CONCEPT](/CONCEPT.md) parks a related question:
"What 'seen the whole thing' means" — how a report states its own coverage so a gap is
visible. What bounds the run, and what does the binary emit when it stops?

# Options

- **Three hard caps + a coverage statement** — max turns, max cost, max wall clock,
  overridable per invocation, plus a "coverage: incomplete" marker on exhaustion.
- **Unbounded, instrumented** — no caps at all; the binary reports turns, cost and
  elapsed time on stderr as it runs, and the numbers are the input to setting caps later.
- **Turn cap only** — one number to reason about; says nothing about a model that spends
  heavily in few turns.

# Recommendation

*(Original recommendation, superseded at triage — kept as history.)* Three hard caps
plus a coverage statement, on the argument that the tool spends the user's credit
unsupervised inside a script.

# Verdict

**Unbounded and instrumented in Milestones 1–2. Caps deferred to Milestone 3, sized
from what those runs actually measured.**

The user's objection, and it holds: nobody knows yet how many turns a review takes or
how long it runs, so any cap chosen now is a guess — and a guess set too low does not
protect anything, it silently truncates legitimate reviews and makes the tool look
worse than it is. Bounds are a thing to add *after* the numbers exist.

The counter-argument — unsupervised spend on the user's own keys — turns out not to
apply yet. Through Milestones 1 and 2 the binary is launched by hand by its author, who
is watching and can interrupt; there is no script calling it. That changes at
[Milestone 3](/scope/17-milestones.md), when the pipeline starts invoking it with no
human present. So the risk and the knowledge arrive in the right order: measure while
supervised, bound before unsupervised.

What Milestone 2 ships instead of caps is **measurement**: turns taken, accumulated
cost and elapsed time, emitted on stderr (`kern-link` already supplies per-call cost
and token accounting, so this is reading numbers that exist, not building an accounting
layer). Those numbers are also what
[success criteria](/scope/14-success-criteria.md) needs for its cost/time bound, and
what gives the concept's parked question about `review-implementation` and merged diffs
its first real answer.

Deliberately dropped with the caps: the model-authored coverage narrative. It was
mitigation for a truncated read, and without caps there is no bound-driven truncation
to signal. Context overflow remains a real way a read gets cut short — it is recorded
as an open risk in [risks & assumptions](/scope/18-risks-and-assumptions.md) rather
than mitigated in v1.

Milestone 3 inherits: **caps sized from measurement**, and the decision of which to
expose as flags, made against real numbers instead of imagination.
