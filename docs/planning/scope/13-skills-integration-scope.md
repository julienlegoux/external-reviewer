---
type: Decision
title: "Skills-side integration"
description: "Does v1 include changing the lx skills to call this binary, and does opencode stay?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 13
slug: skills-integration-scope
status: decided
verdict: "In scope, replacing opencode; the native fallback stays unchanged"
decided_via: triage
depends_on: [report-output-contract, reviewer-selection-vocabulary, mvp-feature-cut]
---

# Question

The binary is only useful once something calls it. Today
`_shared/review-interfaces.md` in the lx skills repo probes `command -v opencode` and
shells out to it. Is changing that part of this project's v1 — and does the `opencode`
path survive alongside?

This crosses a repository boundary: the edit lands in the skills repo, not this one.

# Options

- **In scope, replacing opencode** — v1 ends with `review-interfaces.md` probing and
  calling `external-reviewer`, and the `opencode` branch removed. One documented way to
  get a second opinion.
- **In scope, alongside opencode** — probe `external-reviewer` first, fall back to
  `opencode`, then native. Nothing breaks for anyone still on `opencode`; two delegation
  paths must now be kept correct forever.
- **Out of scope** — ship the binary; wire the skills in a separate effort. Keeps this
  project to one repo, and leaves v1 unable to demonstrate that it works.

# Recommendation

**In scope, replacing opencode.** The stated problem is the `opencode` dependency, so a
v1 that leaves it in place has not solved anything — it has added a second tool beside
the one it meant to remove. It is also the only way the success criteria can be
checked: the proof this works is a real `review-epics` run producing verified leads
through the new binary.

Against keeping both: the fallback chain has three branches and the middle one is the
dependency being eliminated. Per [target users](/scope/02-target-users.md), v1 has one
user, who will have the new binary — so the compatibility this buys is compatibility
with nobody, at the price of a second delegation contract that must stay correct as
`review-interfaces.md` evolves. The native fallback stays, unchanged and unconditional;
that is the branch that actually protects users without the capability.

Note the deliverable: this milestone's work is an edit to the *skills* repo, so its
issues carry a different repo than the rest of the project. Worth flagging to
`split-epics` rather than discovering during implementation.

# Verdict

**Accepted as recommended: in scope, replacing `opencode`.** v1 ends with
`_shared/review-interfaces.md` probing and calling `external-reviewer`, and the
`opencode` branch removed. The native fallback stays unchanged and unconditional — that
is the branch that actually protects users without the capability.

A v1 that leaves `opencode` in place has not solved the stated problem; it has added a
second tool beside the one it meant to remove. Keeping both would also mean maintaining
two delegation contracts forever, for compatibility with nobody — v1 has one user
([target users](/scope/02-target-users.md)), and that user will have the new binary.

**Cross-repository note for `split-epics`:** this milestone's work is an edit to the
**lx skills repository**, not to this one. Its issues carry a different repo than every
other milestone, and that needs to be visible at epic-splitting time rather than
discovered during implementation.
