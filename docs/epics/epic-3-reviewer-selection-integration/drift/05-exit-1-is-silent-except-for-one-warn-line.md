---
type: Drift record
title: "The exit-1 path is not silent: it writes one warn line naming the rule and the layer that assigned the model"
description: "SPECS calls the not-reached path 'a silent native fallback nobody needs told about'. The shipped code keeps stdout empty and writes no error: line, but does write one warn line — because with three assignment layers and five refusal reasons, exit 1 alone is undebuggable."
tags: [epic-3, drift]
timestamp: 2026-08-13T16:05:00Z
epic: 3
issue: 05
---

# The exit-1 path is not silent: it writes one warn line naming the rule and the layer that assigned the model

## Decided

[SPECS § Interfaces](../../../planning/SPECS.md) and
[specs 13](../../../planning/specs/13-error-handling-and-failure-classification.md) both
describe the not-reached path in the same words:

> Not reached → `1`, a silent native fallback nobody needs told about: no config, no tier
> entry, model absent from the catalog, family excluded, provider unconfigured.

[Issue 05](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md)
carries it into two acceptance criteria: exit `1` with *"stdout empty, stderr carries no
`error:` line"*, and — for the excluded-family case — *"stderr never names a substitute
model, asserted on the full stderr text"*.

## Actual

Both criteria hold as written: stdout is empty, `diag.WriteError` is never called on this
path, and no model other than the refused one appears anywhere on stderr. But the run is
not silent. `selectReviewer` (`internal/cli/review.go`) writes exactly one `warn` line
before returning, switching on `reviewer.Reason`:

```
warn    no reviewer: nothing assigns tier "heavy" - set [tiers.heavy] in <path>, or EXTERNAL_REVIEWER_TIER_HEAVY
warn    no reviewer: openrouter/anthropic/claude-sonnet-4.5 is a anthropic model, and the anthropic family is excluded from reviewing (assigned by <path>)
```

## Because

Epic 1 could call exit 1 silent honestly: there was one hard-coded reviewer, so exit 1
meant exactly one thing. After this epic there are three assignment layers (`--model`, a
per-tier environment variable, the config file) and five refusal reasons
(`unassigned`, `not-in-catalog`, `family-excluded`, `family-unknown`, `unconfigured`). A
correctly spelled tier that resolves to nothing is now the single hardest state this
binary can be in, and the exit code carries none of the four facts needed to fix it: which
tier, which model, which rule, which layer named it.

The live example is not hypothetical. All 25 `github-copilot` models classify as
unknown-family
([drift record 04](/epic-3-reviewer-selection-integration/drift/04-github-copilot-tiers-resolve-to-no-reviewer.md)),
so a user who assigns a tier to that provider gets a correct configuration, a successful
refresh, a found model — and exit 1. Without a line, the only way to learn why is to read
this repository's source.

The distinction that makes this safe is the one SPECS itself draws between the two writers.
`error:` is what a calling skill copies into its report — that is the channel
[scope 09](../../../planning/scope/09-report-output-contract.md) says exit 1 must stay off,
so a machine with no reviewer configured does not put a line in every report about a
capability nobody asked for. `warn` is transcript-only, and SPECS already uses it for
exactly this shape of fact: *"unrecognised keys are **warned**, not ignored, since a typo
would otherwise…"*. The silence the decision is protecting is silence *in the report*, and
that is preserved exactly.

## Alternatives tried

- **Put the reason on the `model` stderr line.** Impossible: there is no resolved model on
  this path, and the issue fixes that line's shape as `provider/id auth=<source>`.
- **Say nothing and let `tiers` (issue 06) explain it.** Rejected: `tiers` is a separate
  invocation a user has to know exists, and issue 10's template calls `review` from a
  script whose transcript is the only artifact a human later reads. The diagnosis has to be
  in the run that failed.
- **Use `error:` anyway and accept exit 1 with a line.** Rejected — it is the one thing
  both the criterion and the decision explicitly forbid, and it would put a line in every
  report produced on an unconfigured machine, which is precisely the outcome
  [specs 13](../../../planning/specs/13-error-handling-and-failure-classification.md)
  rejected the "everything that is not a clean report exits 2" option to avoid.

## Disposition

Shipped, and proposed as an amendment to SPECS' wording rather than a change to the code:
the sentence should say *a silent native fallback for the caller's report* rather than a
silent run. The property worth keeping — no `error:` line, empty stdout, no substitute
model named — is asserted directly by
`TestRun_Review_NoConfigAndNoEnvironment_ExitsOneSilently` and
`TestRun_Review_AnthropicTier_ResolvesToNothingAndNamesNoSubstitute`, so an amendment
loosens no guarantee anyone is relying on.

## Revisit when

Issue 06 lands `tiers`. If that command turns out to be where users actually diagnose
selection, this line could shrink to naming the tier and pointing at `tiers` — but not
before it exists. Also when issue 10 writes the invocation template: if the template
discards stderr on exit 1, this line buys nothing and the decision should be reopened
rather than left as dead output.

## Evidence

- `internal/cli/review.go` — `selectReviewer`, `noReviewerDiagnostic`, `assignedBy`.
- `TestRun_Review_NoConfigAndNoEnvironment_ExitsOneSilently` counts `error:` lines and
  requires zero; `TestRun_Review_UnassignedTier_SaysWhereItLooked` and
  `TestRun_Review_ExcludedTier_SaysWhichLayerAssignedIt` require the warn line's two facts
  (`internal/cli/selection_test.go`).
- `TestRun_Review_AnthropicTier_ResolvesToNothingAndNamesNoSubstitute` asserts on the full
  stderr text that none of the three other reachable, credentialed models is named.
