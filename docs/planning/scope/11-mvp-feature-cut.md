---
type: Decision
title: "MVP feature cut"
description: "What is in v1?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 11
slug: mvp-feature-cut
status: decided
verdict: "Thin agentic reviewer with tier resolution, plus a tier-resolution check command"
decided_via: triage
depends_on: [core-user-journeys, read-only-tool-set, loop-bounds-and-termination, report-output-contract, family-exclusion-rule, catalogue-location-and-format]
---

# Question

What ships in v1? Everything listed here becomes real issues and pull requests
downstream, so the bias is small.

# Options

- **Thin agentic reviewer** — one `review` command: resolve a tier to a model, refuse
  same-family, run a bounded read-only loop over one repo, print markdown. Plus a
  `doctor`-style command that reports which tiers resolve, for the setup journey.
- **Thin reviewer, no config resolution** — take `--model provider/id` directly and
  leave tier resolution to the caller. Smaller, but drops the concept's central idea:
  skills ask for a kind of reviewer and never name a vendor.
- **Reviewer plus review-type presets** — ship prompt templates per review kind
  (epics, issues, implementation). Convenient, and it re-imports into the binary the
  pipeline knowledge that belongs in the skills.

# Recommendation

**Thin agentic reviewer, tier resolution included.** Tier resolution is not a
convenience feature — it is the mechanism that keeps vendor judgment on the user's
machine, which is the concept's load-bearing decision. Dropping it would ship a
correct binary that fails the thing the product is for.

Prompt templates go the other way: the task prompt is assembled by the calling skill,
which knows what it is reviewing and against what. Baking review kinds into the binary
would mean editing and re-releasing a Go binary to change review prose, and would leave
two sources of truth for what a good review asks.

The second command earns its place cheaply: without a way to ask "which tiers actually
resolve on this machine right now", the setup journey is trial and error against a
model that silently is not there.

# Verdict

**Accepted as recommended: thin agentic reviewer, tier resolution included.** v1 is:

- a `review` command that resolves a tier to a model, refuses the excluded family, runs
  a read-only agent loop over one repository, and prints markdown to stdout;
- a command that reports which tiers actually resolve on this machine, for the setup
  journey.

Tier resolution stays in even though it looks like a feature, because it is the
mechanism that keeps vendor judgment on the user's machine — the concept's load-bearing
decision. Prompt templates stay out: the task prompt is assembled by the calling skill,
and baking review kinds into the binary would mean re-releasing Go code to change
review prose.

**Refreshed after [loop bounds](/scope/08-loop-bounds-and-termination.md) was decided.**
The loop in v1 is *unbounded and instrumented*, not bounded: no turn, cost or
wall-clock caps, and no coverage statement. What v1 does ship is the measurement —
turns, accumulated cost and elapsed time on stderr. Caps move to Milestone 3, sized
from those numbers.
