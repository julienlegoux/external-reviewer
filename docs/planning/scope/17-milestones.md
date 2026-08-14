---
type: Decision
title: "Milestones & phasing"
description: "How does the work phase into independently shippable chunks?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 17
slug: milestones
status: decided
verdict: "Three milestones: walking skeleton, read-only agentic loop, selection + pipeline integration"
decided_via: triage
depends_on: [mvp-feature-cut, skills-integration-scope, core-user-journeys]
---

# Question

How does v1 phase into independently shippable milestones? These become the
`## Milestone N:` headings that `split-epics` cuts into epics — one milestone is one
epic, weeks-sized, not an afternoon.

# Options

- **Three milestones** — (1) walking skeleton: CLI + `kern-link` + one foreign call +
  markdown out; (2) the agentic read-only loop: tools, bounds, coverage; (3) reviewer
  selection and pipeline integration: tiers, config, family rule, `review-interfaces.md`.
- **Two milestones** — fold selection into the skeleton and integration into the loop.
  Fewer boundaries, but the first milestone then mixes "can we reach a foreign model at
  all" with "how do we choose one", which are unrelated risks.
- **Four milestones** — split integration into "wire the skills" and "prove it on a
  real review". Cleaner reporting, but the second half is a validation pass rather than
  a shippable increment.

# Recommendation

**Three milestones**, ordered so each retires the biggest remaining unknown:

1. **Walking skeleton** — a binary that takes a repo and a prompt, calls one hard-coded
   foreign model through `kern-link`, and prints markdown to stdout with diagnostics on
   stderr. Ships the output contract and proves the dependency works, which is the
   riskiest single assumption ([constraints](/scope/15-constraints.md): multi-turn is
   unproven).
2. **Read-only agentic loop** — the four tools, the repo-root boundary, the multi-turn
   loop, the three caps, and the coverage statement. This is where the product stops
   being a prompt-pipe and becomes a reviewer.
3. **Reviewer selection and pipeline integration** — tier vocabulary, the machine-local
   config, family exclusion, the tier-resolution command, and the
   `review-interfaces.md` swap that retires `opencode`.

Each is independently shippable in the honest sense: stop after 1 and you have a usable
foreign-model prompt runner; stop after 2 and you have a working external reviewer you
invoke by hand; 3 is what makes the pipeline call it for you.

Milestone 3 spans two repositories (see
[skills integration](/scope/13-skills-integration-scope.md)) — `split-epics` should
expect its issues to land in the lx skills repo, not this one.

# Verdict

**Accepted as recommended: three milestones**, each retiring the biggest remaining
unknown, and each independently shippable in the honest sense.

1. **Walking skeleton** — binary takes a repo and a prompt, calls one hard-coded foreign
   model through `kern-link`, prints markdown to stdout with diagnostics on stderr.
   Ships the output contract and proves the riskiest assumption.
2. **Read-only agentic loop** — the four tools, the repo-root boundary, the multi-turn
   loop, and run instrumentation.
3. **Reviewer selection and pipeline integration** — tier vocabulary, machine-local
   config, family exclusion, the tier-resolution command, loop caps sized from
   Milestone 2's measurements, and the `review-interfaces.md` swap that retires
   `opencode`.

**Refreshed after [loop bounds](/scope/08-loop-bounds-and-termination.md) was decided.**
Milestone 2 no longer carries caps or a coverage statement; it carries *measurement*
instead — turns, accumulated cost and elapsed time on stderr. Caps moved into Milestone
3, which is both when the numbers exist and when the pipeline starts calling the binary
with no human watching. The three-milestone shape is unchanged; the boundary between 2
and 3 shifted.

**Milestone 3 spans two repositories.** Its `review-interfaces.md` work lands in the lx
skills repo, not this one — see
[skills integration](/scope/13-skills-integration-scope.md). `split-epics` should expect
that rather than discover it.
