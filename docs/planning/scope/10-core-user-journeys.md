---
type: Decision
title: "Core user journeys"
description: "Which flows must work end to end for v1 to mean anything?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 10
slug: core-user-journeys
status: decided
verdict: "Four journeys: external review, clean absence, first-time tier setup, manual sanity-check"
decided_via: triage
depends_on: [target-users, reviewer-selection-vocabulary, report-output-contract]
---

# Question

Which end-to-end flows define v1? These become the acceptance-criteria seeds for every
epic downstream, so a missing journey is a missing epic.

# Options

- **Four journeys** — (1) a skill runs an external review and gets usable leads;
  (2) the capability is absent and the caller falls back cleanly; (3) the user sets up
  their tier assignment for the first time; (4) the user runs the binary by hand to
  sanity-check a model before trusting it.
- **Two journeys** — the happy path and the absent path only. Setup and manual
  inspection get folded into documentation.
- **Six or more** — add per-skill journeys (review-epics, review-issues,
  review-implementation) as separate flows.

# Recommendation

**Four journeys.** The first two are the product; the second two are what make the
first two possible on a real machine.

Journey 2 is not a footnote — "capability absent → review runs natively and says so" is
the single rule every failure mode in the concept resolves to, and it is the flow most
likely to be shipped broken because nobody tests the path where their tool does
nothing. Journey 3 is the concept's parked "installation path" question in its
smallest honest form: not packaging, just *how a user arrives at a working assignment*.
Journey 4 is how the user forms the judgment the concept insists only they can make —
without it, the tier config is guesswork.

Per-skill journeys are the same flow with a different prompt; splitting them would
triple the acceptance criteria without testing anything new.

# Verdict

**Accepted as recommended: four journeys**, which become the acceptance-criteria seeds
downstream.

1. A review skill runs an external pass and gets usable leads back.
2. The capability is absent (no binary, no credential, no assignment for the tier) and
   the caller falls back to a native review, recording that no external pass happened.
3. The user sets up their tier assignment for the first time and can see which tiers
   resolve.
4. The user runs the binary by hand to judge a model before trusting it with a tier.

Journey 2 is not a footnote: "capability absent → review runs natively and says so" is
where every failure mode in the concept resolves, and it is the flow most likely to ship
broken because nobody tests the path where their tool does nothing. Journey 4 is how the
user forms the judgment the concept insists only they can make — without it, the tier
assignment is guesswork.

Per-skill journeys were rejected: `review-epics` and `review-issues` are the same flow
with a different prompt.
