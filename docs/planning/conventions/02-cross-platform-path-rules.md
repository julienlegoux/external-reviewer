---
type: Decision
title: "Cross-platform path and process rules"
description: "What concrete coding rules carry SCOPE's 'nothing may assume POSIX' constraint into day-to-day code?"
tags: [decision, conventions]
timestamp: 2026-08-09T04:08:38Z
phase: conventions
decision: 02
slug: cross-platform-path-rules
status: decided
verdict: "A — two named path vocabularies, OS paths and wire paths, converted at the boundary"
decided_via: triage
depends_on: []
---

# Question

SCOPE binds the project with *"Windows is the development machine, and nothing may
assume POSIX"*, and SPECS builds on it — `os.Root` for confinement, `os.UserConfigDir`
for config, `/`-separated repo-relative paths in every tool's input and output. The
baseline says nothing about platforms; it predates a project where the dev machine and
CI's primary consumer disagree.

Without a stated rule, the failure mode is specific and recurring: a `strings.Split(p,
"/")`, a `"dir/" + name`, a `filepath.Join` leaking `\` into a tool result the model
then quotes back, or a test asserting a hardcoded separator. CI's two-OS matrix catches
some of it, late.

# Options

- **A — Two path vocabularies, named and separated.** *OS paths* use `path/filepath` and
  never appear in tool input/output; *wire paths* (anything crossing the model boundary,
  the allow-list, and diagnostics) are `path`-package, `/`-separated, repo-relative, and
  converted at the boundary with `filepath.ToSlash`/`FromSlash`. Tests that assert on a
  path assert the wire form. No `//go:build windows` divergence without a test that runs
  on both.
- **B — Rely on CI.** State the constraint, let the windows-latest + ubuntu-latest matrix
  find violations. Cheaper to write; every violation costs a red build and a round trip,
  and only the ones a test happens to cover are found at all.
- **C — Normalise everything to `/` internally, convert only at syscalls.** Simplest
  mental model, but it fights the stdlib — `os.Root`, `filepath.Clean` and Windows
  reserved-name handling all want native paths, and hand-normalising is how the
  hand-rolled path validation SPECS rejected creeps back in.

# Recommendation

**A.** It is the rule the design already implies — SPECS commits to "repo-relative
`/`-separated paths in both directions on every platform" — stated so an implementer
who never opened SPECS still writes conforming code. The naming matters more than the
mechanism: once "OS path" and "wire path" are distinct words, a reviewer can see the
bug in a diff instead of waiting for a Linux runner to find it.

# Verdict

**A — two path vocabularies, named and separated.** Accepted at triage as recommended.
*OS paths* use `path/filepath` and never cross the model boundary; *wire paths* — tool
input and output, the allow-list, stderr diagnostics — are `path`-package,
`/`-separated and repo-relative, converted with `filepath.ToSlash`/`FromSlash` at the
boundary. Path assertions in tests assert the wire form, and no `//go:build` platform
divergence ships without a test that runs on both OSes in the CI matrix.

**Promotion candidate**: the rationale is not project-specific — Windows is the author's
machine on every project while CI and agent implementers run Linux. Worth promoting to
`assets/baseline.md` as a general section via the improve-skill loop, so future projects
inherit it instead of rediscovering it through a red Linux build.
