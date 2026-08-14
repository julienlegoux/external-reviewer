---
type: Decision
title: "Dependency additions"
description: "Is a third module a PR-note justification, or a reopened specs decision?"
tags: [decision, conventions]
timestamp: 2026-08-09T04:08:38Z
phase: conventions
decision: 07
slug: dependency-freeze
status: decided
verdict: "A — frozen: a third module reopens specs decision 03"
decided_via: triage
depends_on: []
---

# Question

The baseline treats a new dependency as a lightweight event: "prefer the standard
library; a new dependency needs a one-line justification in the PR that adds it." SPECS
is considerably harder — it lists `kern-link` and `BurntSushi/toml` under the heading
"Two non-stdlib dependencies, **and no others**", and separately rules out a CLI
framework, a test framework, a logging library and a config library by name. The
supply-chain surface is also part of the product: this binary reads a user's private
repository and sends it to a third party, on the user's own credentials.

A one-line PR note is a weaker gate than what SPECS assumes, and an agent implementer
that reads only CONVENTIONS.md would clear it.

# Options

- **A — Frozen: a third module reopens a specs decision.** Adding any non-stdlib import
  is a change to [specs 03](/specs/03-kern-link-dependency-policy.md), not a PR note —
  the decision doc gets reopened and the user decides. Dropping a dependency, and
  bumping a pinned version, stay ordinary PRs. Also stated: no `replace` directive is
  ever committed (it breaks `go install` for anyone whose disk differs from the
  author's), and `go.work` stays gitignored.
- **B — Baseline as written.** One-line justification in the PR. Fast, and it makes the
  "and no others" in SPECS advisory the first time an implementer wants a helper.
- **C — Frozen for the binary, free for tests.** Test-only dependencies allowed under
  the baseline rule. Tempting, but SPECS ruled out `testify` on evidence (zero
  occurrences across `kern-link`'s 102 test files), so the carve-out would immediately
  contradict a decided item.

# Recommendation

**A.** SPECS did not say "few dependencies", it said two and named them; a conventions
doc that softens that to a PR note creates the second source of truth this bundle exists
to prevent. The rule is cheap to obey — it only bites when someone is about to make a
decision that was already made — and it puts the `replace`/`go.work` rules where the
person adding a module will actually read them.

# Verdict

**A — frozen; a third module reopens a specs decision.** Accepted at triage as
recommended. Adding any non-stdlib import beyond `kern-link` and `BurntSushi/toml` is a
reopening of [specs 03](/specs/03-kern-link-dependency-policy.md) with the user
deciding, not a one-line justification in a PR. Removing a dependency and bumping a
pinned version stay ordinary PRs. A `replace` directive is never committed — it breaks
`go install` for anyone whose disk does not match the author's — and `go.work` stays
gitignored.
