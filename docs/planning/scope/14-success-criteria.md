---
type: Decision
title: "Success criteria"
description: "How would you know v1 worked?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 14
slug: success-criteria
status: decided
verdict: "opencode retired, leads that survive verification, and a measured cost/time envelope"
decided_via: triage
depends_on: [skills-integration-scope, mvp-feature-cut]
---

# Question

What checkable facts mean v1 succeeded? Must be verifiable, not aspirational.

# Options

- **Replacement + verified leads + a cost bound** — `opencode` is gone from the
  pipeline, a real review run on this project's own epics produces leads that survive
  verification against the files, and a standard-tier review costs and takes less than a
  stated bound.
- **Replacement only** — the binary is wired in and does not crash. Checkable, and
  compatible with a reviewer that returns nothing worth reading.
- **Quality metrics** — a measured findings-per-review or false-positive rate against
  a labelled corpus. Rigorous, and building the corpus is larger than the project.

# Recommendation

**Replacement + verified leads + a cost bound.** Each covers a distinct way v1 could
fail: not adopted, adopted but useless, or useful but too slow and expensive to reach
for.

"Leads that survive verification" is the right quality bar because it is the pipeline's
own rule — `review-interfaces.md` already treats external output as leads to be checked
against the files. It needs no new apparatus, only that a real run produced at least
some leads that turned out to be true.

The cost/time bound is the one that decides whether this gets used twice. It also gives
the concept's parked question about `review-implementation` and merged diffs its first
real measurement, rather than continuing to guess.

Explicitly not a criterion: that the external reviewer finds things the native review
missed. It might, but a second opinion that merely confirms is still worth having, and
making novelty the bar would push the reviewer toward inventing findings.

# Verdict

**Accepted as recommended**, with one wording change from the loop-bounds discussion.
v1 succeeded when all three hold:

1. `_shared/review-interfaces.md` no longer mentions `opencode`, and the external pass
   runs through `external-reviewer`.
2. A real `review-epics` or `review-issues` run on this project's own artifacts produced
   leads, at least some of which survived verification against the files.
3. A standard-tier review's **measured** cost and wall-clock time are recorded, and the
   author judges them low enough to reach for the tool again.

Each covers a distinct failure: not adopted, adopted but useless, useful but too
expensive to repeat.

**Refreshed after [loop bounds](/scope/08-loop-bounds-and-termination.md) was decided.**
Criterion 3 was originally "costs less than a stated bound". There is no bound to state
yet — that verdict deliberately deferred caps until real numbers exist. So the criterion
is now to *produce* the numbers and judge them, and those same numbers are what
Milestone 3 uses to size its caps. The measurement is the deliverable, not a threshold
invented in advance.

Explicitly not a criterion: that the external reviewer finds things the native review
missed. A second opinion that merely confirms is still worth having, and making novelty
the bar would push the reviewer toward inventing findings.
