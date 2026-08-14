---
type: Decision
title: "Branch model and integration trunk"
description: "Which branch do issue branches cut from and merge into, and what does main mean?"
tags: [decision, conventions]
timestamp: 2026-08-09T04:08:38Z
phase: conventions
decision: 03
slug: branch-model
status: decided
verdict: "A — develop is the integration trunk, main is the install surface"
decided_via: triage
depends_on: []
---

# Question

The baseline says "one branch per issue, named `issue-<n>-<slug>`; branches are
short-lived" and "no force-pushes to main" — it never says what branches exist above
that. This repo already has both `main` and `develop`, and all planning work so far has
landed on `develop`. `implement-epic` merges green PRs into an integration branch, so
the answer is load-bearing for every epic run, and leaving it implicit means each run
re-derives it.

# Options

- **A — `develop` is the integration trunk; `main` is the install surface.** Issue
  branches cut from and merge into `develop`; `develop` merges to `main` at a milestone
  boundary. Matches what the repo already does. It also has a concrete meaning here that
  it lacks on most projects: distribution is
  `go install github.com/julienlegoux/external-reviewer@latest`
  ([specs 17](/specs/17-distribution-and-ci.md)), and `@latest` on an untagged module
  resolves to the **default branch's** newest commit — so `main` is literally what gets
  installed, and a half-finished epic on it is a broken install.
- **B — Trunk-based on `main`.** Delete `develop`, everything merges to `main`. Simpler,
  one fewer merge — but then every intermediate commit of every epic is what
  `go install @latest` hands the user.
- **C — `develop` trunk, plus release tags on `main`.** Adds tags for traceability;
  SPECS explicitly rejected release machinery ("no releases, no tags, no changelog"), so
  this reopens a decided item for a single-user tool.

# Recommendation

**A.** It is the status quo, it is what `bundle-interfaces.md` assumes when it says doc
work lands on "the integration trunk (`develop`) when one is in play", and on this
project the `main`/`develop` split earns itself — it is the difference between
`@latest` meaning "the last milestone" and "whatever an agent merged twenty minutes
ago". `main` keeps the baseline's no-force-push rule; `develop` gets it too, since
agents branch from it.

# Verdict

**A — `develop` is the integration trunk; `main` is the install surface.** Accepted at
triage as recommended. Issue branches cut from `develop` and merge back into it; planning
and doc work lands on `develop`; `develop` merges to `main` at a milestone boundary.
Neither branch is ever force-pushed. On this project the split is not ceremony: `main` is
the default branch, so it is literally what
`go install github.com/julienlegoux/external-reviewer@latest` resolves to.

**Promotion candidate**: the `main`/`develop` split is likely a personal default rather
than a fact about this project, and `bundle-interfaces.md` already assumes an integration
trunk without naming one. Worth raising in the improve-skill loop — the *reason* stated
here (untagged `@latest` tracks the default branch) is project-specific and should stay
here even if the branch model itself moves up.
