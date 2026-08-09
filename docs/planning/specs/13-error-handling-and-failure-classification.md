---
type: Decision
title: "Error handling & failure classification"
description: "Which failures mean exit 1 (capability absent) and which mean exit 2 (ran but unusable)."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 13
slug: error-handling-and-failure-classification
status: decided
verdict: "Not reached means exit 1; reached then failed means exit 2; a broken credential is 2, an absent one is 1"
decided_via: triage
depends_on: [output-and-diagnostics-format]
---

# Question

Scope calls the `1`/`2` distinction load-bearing: exit `1` is a normal silent fallback
for the caller, exit `2` is worth a line in the report
([decision](/scope/09-report-output-contract.md)). That is a routing rule for the reader
of the final report — `1` means nothing happened and nobody needs to know, `2` means an
external pass was attempted and failed, which is information about the review's coverage.

So every failure the binary can hit must be classified into one of the two, and the
classification has to hold for failures that arrive from inside `kern-link` as typed
errors rather than as anything this code raised.

# Options

- **Classify by "did we get as far as sending a request?"** One crisp question with one
  answer per failure; some judgment calls sit right on the line.
- **Classify by error type from `kern-link`.** Precise where `*ai.ModelsError` codes
  exist, and undefined for the failures that never reach `kern-link` at all.
- **Everything that is not a clean report exits `2`.** Trivially correct and it destroys
  the distinction, making every unconfigured machine emit a line in every report about a
  capability nobody asked for.

# Recommendation

**Classify on whether an external reviewer was ever *reachable*. Not reached → `1`.
Reached and then failed → `2`.**

Exit **1 — capability absent** (silent native fallback, stdout empty):

- No config file, or the requested tier has no entry.
- The assigned model is not in `kern-link`'s catalog.
- The assigned model's family is excluded, or unknown and therefore excluded
  ([decision](/specs/04-model-family-classification.md)).
- `models.GetAuth(ctx, model)` returns `(nil, nil)` — the provider is unconfigured. This
  is checked *before* the first turn, precisely so an uncredentialed machine lands in `1`
  rather than discovering the problem mid-run.

Exit **2 — ran but unusable** (worth a line in the report, stdout empty):

- A turn failed: `Stream.Result` errored, or `StopReason` was `error`/`aborted`.
- Auth resolution failed with a typed `*ai.ModelsError` (`oauth` — a refresh failed;
  `auth` — key resolution or the credential store failed). A *broken* credential is not
  an *absent* one: something is wrong that a human should fix, and swallowing it silently
  is how a machine spends weeks doing native reviews while believing otherwise.
- Context overflow, rate limits, provider outages — `kern-link` exposes classifiers for
  these; all are `2`.
- The run completed but the final message carried no text.
- Interrupted by `SIGINT`.

Exit **2** also covers usage errors (unknown flag, missing repo path, unreadable repo),
which is the stdlib `flag` convention and keeps `1` meaning exactly one thing.

Two rules underneath:

- **Never panic across the CLI boundary.** `run()` returns an exit code; a panic in a
  tool or the loop is recovered, logged to stderr, and becomes exit `2`.
- **Diagnostics are already redacted.** `kern-link` attaches non-fatal problems as
  `AssistantMessageDiagnostic` values holding redacted `DiagnosticErrorInfo`. Those are
  printed to stderr as `warn` lines rather than being discarded, and no error message is
  ever re-formatted in a way that could re-introduce a credential
  ([decision](/specs/14-security-and-data-exposure.md)).

# Verdict

Accepted at triage as recommended. The credential split is the sharp edge: an unconfigured provider is a silent native fallback, a failing OAuth refresh or credential store is something a human must fix and must not be swallowed.
