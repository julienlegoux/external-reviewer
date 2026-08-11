---
type: Issue
title: "Extract one importable faux harness and retire the mutable package-level seams"
description: "Replace the ~75 duplicated harness lines across two test packages with one importable fixture package, and remove performReview and models from product code."
tags: [epic-0]
timestamp: 2026-08-11T10:00:00Z
epic: 0
issue: 05
slug: shared-faux-harness
size: M
status: done
gh_issue: 27
gh_pr: 40
resource: https://github.com/julienlegoux/external-reviewer/issues/27
depends_on: [4]
---

# Extract one importable faux harness and retire the mutable package-level seams

## Summary

Issue 05 of Epic 1 scoped *"a reusable `faux`-provider test harness the later epics build
their loop tests on."* What exists is a per-package copy, duplicated twice:
`internal/reviewer/resolve_test.go:25-121` and `internal/cli/preflight_test.go:17-143` are
~75 byte-identical lines modulo one constant rename — `authProvider`, `credentialedAuth`,
`unconfiguredAuth`, `brokenAPIKeyAuth`, `expiredOAuthAuth`, and the credential literal
itself (`secret` / `preflightSecret`, same value). Neither copy is importable: both are
test-only files in `package cli_test` / `package reviewer_test`.

The builders have already **diverged** rather than converged — `fauxRegistry`
(`resolve_test.go:31`, takes a `providerID`, no price sheet), `registryWith` / `registry`
(`preflight_test.go:43,67`, hard-codes `DefaultProviderID`, adds `ModelCost`), `scripted`
(`roundtrip_test.go:50`). Every remaining issue in this epic scripts a `faux` response,
issue 06 adds a whole test file in `internal/reviewer` that cannot build on the
`internal/cli` copy, and Epic 2 adds four tools each needing the same offline registry. A
third copy is otherwise certain. CONVENTIONS bans assertion *frameworks*, not a shared
fixture package.

The same PR retires the mutable package-level seams. Issue 03 of Epic 1 needed to reach the
exit-1 and exit-2 terminations before a real reviewer existed, so it put
`var performReview = resolveAndReview` in production code plus an adapter that flattens
`reviewRequest` back into `(repoPath, task string)`. Issue 04 then arrived with the
sanctioned mechanism, and issues 04/05 prove it drives every one of those terminations for
real: `preflight_test.go:184` (`stop=no_reviewer`), `preflight_test.go:225` and
`roundtrip_test.go:110` (`stop=failed`) cover exactly what `exit_test.go:44` and
`exit_test.go:73` stub out.

## Scope

- One importable harness package — e.g. `internal/fauxtest` — carrying, once each: the
  auth-provider stub, the four auth shapes (`credentialedAuth`, `unconfiguredAuth`,
  `brokenAPIKeyAuth`, `expiredOAuthAuth`), **one** registry builder with options rather
  than three divergent ones, and the scripted-response helper. It is a non-`_test.go`
  package so both `cli_test` and `reviewer_test` can import it; nothing in the product
  build imports it.
- `internal/reviewer/resolve_test.go`, `internal/cli/preflight_test.go` and
  `internal/cli/roundtrip_test.go` rewritten onto it, with the divergence between
  `fauxRegistry`, `registry`/`registryWith` and `scripted` resolved into the one builder.
- `var performReview = resolveAndReview` (`internal/cli/review.go:31`) removed from product
  code, along with `internal/cli/export_test.go:21-38`'s override of it.
- `var models ai.Models` (`internal/cli/review.go:38`) removed as a package-level mutable:
  the registry reaches `resolveAndReview` as an explicit dependency threaded from `Run`'s
  inner seam, with `export_test.go` exposing a constructor rather than a setter. No mutable
  package-level seam is left in non-test source.
- `exit_test.go:44,73`'s stubbed terminations re-homed onto the faux-driven ones that
  already exist; the one assertion that survives uniquely — the two-level
  `fmt.Errorf("…: %w", …)` wrap at `exit_test.go:45`, issue 03 of Epic 1's explicit
  criterion — asserted against `classify` directly.

## Out of scope

- New behaviour of any kind. This PR removes duplication and a mutable global; the diff
  must not weaken a single assertion.
- The transcript-parsing helpers (two `done`-line regexes, two `turn`-line encodings) —
  they move into this harness in
  [issue 12](/epic-0-skeleton-hardening/issues/12-diag-escaping-and-polish.md), which is
  where their own defect is fixed.
- The tests issue 06 adds in `internal/reviewer` — this PR ships the harness they import,
  not the tests.
- `t.Parallel()`. No test uses it and every current override uses `defer restore()`, so
  there is no data race today; adding parallelism is a separate decision.

## Acceptance criteria / Definition of done

- [ ] One importable harness serves both `internal/cli`'s and `internal/reviewer`'s tests:
      `authProvider`, `credentialedAuth`, `unconfiguredAuth`, `brokenAPIKeyAuth`,
      `expiredOAuthAuth` and the registry builder each appear exactly once in the tree.
- [ ] No mutable package-level seam remains in non-test source: `performReview` and
      `models` are both gone from `internal/cli/review.go`, and the registry reaches
      `resolveAndReview` as an explicit dependency.
- [ ] Every termination `exit_test.go` stubbed is still asserted, now through the
      sanctioned seam: exit `1` with `stop=no_reviewer`, exit `2` with `stop=failed`, and
      the two-level wrap asserted against `classify` directly.
- [ ] The credential-leak assertions survive the move intact: the tests still plant a
      credential-shaped secret in **every** `AuthResult` field and assert it appears on
      neither stream (`preflight_test.go:272`, `resolve_test.go:201`).
- [ ] No product behaviour changes: every test that passed before this PR passes after it,
      and the PR body names any assertion that moved and where it went.
- [ ] `go test ./... -race` and `golangci-lint run` pass; CI green on ubuntu-latest and
      windows-latest.

## Relevant files / areas

- `internal/fauxtest/` (new — package name and layout are the implementer's call, provided
  it is importable from both test packages and imported by no product file)
- `internal/reviewer/resolve_test.go:25-121` (`fauxRegistry` at `:31`)
- `internal/cli/preflight_test.go:17-143` (`registryWith`/`registry` at `:43,67`),
  `internal/cli/roundtrip_test.go:50-55` (`scripted`)
- `internal/cli/review.go:31,38` (the two vars), `internal/cli/export_test.go:21-38`
- `internal/cli/exit_test.go:19-45,73`

Governing decisions: [CONVENTIONS § Testing](../../../planning/CONVENTIONS.md) (black-box
by default, mock only true externals, no assertion frameworks),
[SPECS § Testing](../../../planning/SPECS.md) (`faux` as the one mock). Source findings:
[report 2](../../../REPORT_2.md) § "The 'reusable `faux` harness' is a per-package copy"
and § "Two package-level mutable test seams in product code".

## Dependencies

- Blocked by: [04 — CI formatting gate](/epic-0-skeleton-hardening/issues/04-ci-formatting-gate.md)
  — the gate goes in before the epic's large test refactors.
- Blocks: [06 — the four surviving mutants](/epic-0-skeleton-hardening/issues/06-request-side-mutants.md),
  [11 — Turn hardening](/epic-0-skeleton-hardening/issues/11-turn-hardening.md) and
  [12 — diag escaping and polish](/epic-0-skeleton-hardening/issues/12-diag-escaping-and-polish.md),
  each of which scripts `faux` through this harness.

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
Sized **M**: ~150 lines deleted from two test packages, ~150 added as one harness, plus
the seam retirement and the re-homed `exit_test.go` assertions. Its split line, if one is
needed, is the harness extraction first and the `performReview`/`models` retirement second
— but the retirement is what makes the harness the *only* seam, so prefer to keep it whole.
