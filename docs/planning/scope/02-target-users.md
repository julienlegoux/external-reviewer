---
type: Decision
title: "Target users"
description: "Who is v1 for, specifically — the author only, or anyone running the lx pipeline?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 02
slug: target-users
status: decided
verdict: "Author-first — one machine, go install, hand-written config, no packaging"
decided_via: triage
depends_on: []
---

# Question

Who is v1 built for? This is not a formality: it decides whether v1 must carry an
installation path, credential onboarding, cross-platform packaging and documentation
for strangers — or whether it may assume a machine that already has Go, `kern-link`
and a foreign API key. [CONCEPT](/CONCEPT.md) parks "the installation path" as an open
question precisely because this had not been settled.

# Options

- **Author-first** — v1 targets one machine (the author's). `go install` is the install
  story, config is hand-written, no packaging. Cheapest; the first real user finds
  every rough edge.
- **lx pipeline users** — v1 targets anyone who runs the lx skills. Needs released
  binaries for at least Windows/macOS/Linux, a first-run config path, and user-facing
  docs. Multiplies v1 by a milestone or two.
- **Public developer tool** — v1 targets any developer wanting a foreign-model repo
  review, pipeline or not. Adds a general CLI surface, examples, discoverability.
  Contradicts the confirmed premise.

# Recommendation

**Author-first.** The capability has exactly one confirmed consumer today — the lx
review skills on the author's machine — and the whole point of the concept's
"judgment lives on the user's machine" rule is that the artifact does not pretend to
know other people's credentials. Every requirement that only exists for strangers
(signed releases, install docs, config wizards) is real work that cannot be validated
until someone else actually wants it, and shipping it unvalidated is how v1 slips.

Author-first is not a dead end: `go install github.com/julienlegoux/external-reviewer@latest`
is a legitimate install path for a Go tool, and widening later costs packaging and
docs, not a redesign.

# Verdict

**Accepted as recommended: author-first.** v1 targets one machine — the author's.
`go install github.com/julienlegoux/external-reviewer@latest` is the install story,
the tier assignment is hand-written, and nothing is packaged, signed or documented for
strangers. Widening later costs packaging and docs, not a redesign.

This is what makes the concept's parked "installation path" question answerable in v1
at its smallest honest size: not packaging, just how *this* user arrives at a working
assignment.
