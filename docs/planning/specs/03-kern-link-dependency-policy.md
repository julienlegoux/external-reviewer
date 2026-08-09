---
type: Decision
title: "kern-link dependency & version policy"
description: "How this repo depends on kern-link given that the author owns both and expects to fix the dependency mid-build."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 03
slug: kern-link-dependency-policy
status: decided
verdict: "require a tagged kern-link version, no committed replace, gitignored go.work for local iteration"
decided_via: triage
depends_on: [language-module-toolchain]
---

# Question

`kern-link` is at `v0.1.1` and its multi-turn tool loop is the project's top risk
([scope](/scope/18-risks-and-assumptions.md), risk 1) — SCOPE expects fixes to land
upstream *during* Milestones 1–2, in a repository the author also owns and has checked
out at `D:\Project\kern-link`. The question is how the dependency is declared so that
iterating on both at once does not commit a build only one machine can reproduce.

An intake correction worth recording: SCOPE says kern-link's "documented example covers
a single round trip". As of the checked-out revision, `docs/usage.md` § *Tool calls*
documents a full multi-turn loop — declare `[]ai.Tool` on the `ai.Context`, call
`models.CompleteSimple`, append a `ToolResultMessage` per `ai.ToolCall` while
`StopReason == StopReasonToolUse`. The loop is still this project's code to write, but
the pattern is upstream-blessed rather than unprecedented, which lowers risk 1 without
retiring it.

# Options

- **`require` a tagged version; `go.work` (gitignored) for local iteration.** The
  committed build is reproducible from the proxy by anyone; the author still edits both
  trees at once, and each upstream fix becomes a tag plus a `go get -u`.
- **`replace` directive to the local path, committed.** Simplest to iterate on, and it
  makes `go install github.com/julienlegoux/external-reviewer@latest` — the entire
  install story — fail for anyone whose disk does not look like the author's.
- **Vendor `kern-link` into this repo.** Removes the network from the build and freezes
  a fast-moving dependency the author is actively fixing. Wrong direction.

# Recommendation

**`require github.com/julienlegoux/kern-link v0.1.1`, no `replace` in `go.mod`, a
gitignored `go.work` for local iteration.**

A committed `replace` is the one option that breaks the install command SCOPE commits
to, and it breaks it silently — `go install` resolves the module, then fails on a path
that does not exist. `go.work` gives the same edit-both-trees workflow, is invisible to
`go install` (which ignores workspaces), and cannot be committed by accident once
gitignored.

The discipline that makes this work: an upstream fix is not "done" until it is tagged
and this repo's `go.mod` names the tag. Anything that only builds under the workspace is
an unfinished change, and CI — which has no workspace — is what enforces that.

`kern-link` is the only non-stdlib dependency this project takes on for its core
function; the only other is a TOML decoder
([decision](/specs/05-tier-assignment-schema.md)).

# Verdict

Accepted at triage as recommended, and the version is now known: **`kern-link v0.2.0`**, the tag carrying the upstream sync of issue #113 (a catalog refresh across 21+ providers plus adapter, retry and token-estimation changes). `go.mod` names `v0.2.0`; later bumps are ordinary `go get -u` work and not a re-decision.

The policy underneath is what was actually decided and does not move with the number: a tagged `require`, never a committed `replace`, and a gitignored `go.work` for iterating on both trees at once.
