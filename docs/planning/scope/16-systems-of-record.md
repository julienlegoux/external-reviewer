---
type: Decision
title: "Existing systems of record & integrations"
description: "What already holds truth this project must respect rather than restate?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 16
slug: systems-of-record
status: decided
verdict: "Three owners — kern-link, the lx skills repo, the user's machine; two integrations only"
decided_via: triage
depends_on: [catalogue-location-and-format]
---

# Question

What already owns facts this project needs? Anything on this list must be read from,
not copied — a duplicated fact is a fact that will diverge.

# Options

- **Three systems, each owning one thing** — `kern-link` owns model facts and
  credentials, the lx skills repo owns the delegation contract, the user's machine owns
  the tier assignment. This project owns none of them.
- **Own the model facts locally** — keep a small vendored model list for the models the
  user cares about. Explicitly rejected by the concept: it rots on a two-month cycle.

# Recommendation

**Three systems, each owning one thing:**

- **`kern-link`** owns which models exist, what they cost, which family and provider
  they belong to, and how credentials resolve (env keys and OAuth, via its
  cross-process credential store). This project queries it and never maintains a
  parallel list — the concept is explicit that catalogue staleness lands there, because
  it is versioned, tracks an upstream and has a sync procedure.
- **The lx skills repo** (`_shared/review-interfaces.md`) owns the delegation contract:
  when a review delegates, how the model is named in the report, and the
  leads-not-findings rule. This project satisfies that contract; it does not redefine it.
- **The user's machine** owns the tier assignment and the credentials themselves.

Two integrations follow and no others: `kern-link` as a Go dependency, and the git
repository under review, read through the read-only tools. No GitHub API, no issue
tracker, no telemetry endpoint.

One caveat worth recording now: `kern-link`'s embedded catalogue was already behind the
models the author intends to use at the time the concept was written. The fix is
upstream, not a local override — but it means "query kern-link" occasionally means
"update kern-link first".

# Verdict

**Accepted as recommended.** Three systems own facts this project reads and never
copies: `kern-link` (which models exist, their cost, family and provider, and how
credentials resolve), the lx skills repo (the delegation contract), and the user's
machine (the tier assignment and the credentials themselves).

Two integrations follow and no others: `kern-link` as a Go dependency, and the git
repository under review through the read-only tools. No GitHub API, no issue tracker,
no telemetry.

Recorded caveat: `kern-link`'s embedded catalogue was already behind the models the
author intends to use when the concept was written, so "query kern-link" occasionally
means "update kern-link first". The fix is upstream, never a local override.
