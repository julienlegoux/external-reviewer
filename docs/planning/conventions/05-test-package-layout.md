---
type: Decision
title: "Test package layout and style"
description: "Black-box `package foo_test` or in-package tests, and what shape does a test take here?"
tags: [decision, conventions]
timestamp: 2026-08-09T04:08:38Z
phase: conventions
decision: 05
slug: test-package-layout
status: decided
verdict: "A — black-box package foo_test by default, in-package by named exception"
decided_via: triage
depends_on: []
---

# Question

The baseline says tests live beside the source, name the behavior asserted, and prefer
integration-level over heavy mocking. Go then forces a choice the baseline does not
make: `package foo` (in-package, sees unexported identifiers) or `package foo_test`
(black-box, only the exported API). SPECS fixes the tooling — stdlib `testing`, no
`testify`, `-race`, `kern-link`'s `faux` provider, `t.TempDir()` repos, live tests gated
by `os.Getenv` + `t.Skip` — but not the layout.

It matters more here than usual: almost everything is under `internal/`, where "exported"
means "exported to the rest of this binary", and the highest-value tests
(confinement refusals, the family classifier over the whole catalog, the CLI through
`run()`) are naturally black-box.

# Options

- **A — Black-box by default, in-package by exception.** `package foo_test` unless the
  test targets unexported logic that has no reachable path through the package's API, in
  which case `package foo` in a file named `<thing>_internal_test.go`. Table-driven is
  the default shape; a table with one row is a sign the case belongs in another table.
- **B — In-package by default.** Least friction, no export dance — and it quietly
  encourages testing helpers instead of behavior, which is how a suite ends up green
  while the confinement boundary it was supposed to protect is untested.
- **C — No rule.** Let each package choose. Go tolerates this well; the cost is that the
  same behavior gets tested twice at two altitudes across an agent-implemented codebase.

# Recommendation

**A.** The security properties this project sells are properties of the API surface —
`os.Root` refuses an escaping name, an excluded family ends the run at exit `1`, a bad
path in a `read_file` batch is reported inline while the rest return. A black-box test
asserts exactly that; an in-package test can accidentally assert the implementation that
was supposed to be free to change. The named exception keeps the family classifier's
table and the argv parser testable without exporting them for the test's benefit.

# Verdict

**A — black-box by default, in-package by exception.** Accepted at triage as recommended.
Tests are `package foo_test` unless they target unexported logic with no reachable path
through the package's API, in which case `package foo` in a file named
`<thing>_internal_test.go` — the filename makes the exception visible in a diff.
Table-driven is the default shape.

**Promotion candidate**: nothing here is specific to this project — it is the Go variant
the baseline's *Testing* section is missing. Worth promoting to `assets/baseline.md` as a
*(per-stack)* Go entry via the improve-skill loop.
