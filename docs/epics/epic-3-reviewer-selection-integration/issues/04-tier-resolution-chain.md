---
type: Issue
title: "Resolve a tier to a reachable, allowed model"
description: "The four-layer precedence chain, refresh-before-lookup for dynamic providers, the family gate, and the two failure classes — as a library, before any flag exists."
tags: [epic-3]
timestamp: 2026-08-13T13:20:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/63
epic: 3
issue: 04
slug: tier-resolution-chain
size: L
status: in-progress
gh_issue: 63
depends_on: [2, 3]
---

# Resolve a tier to a reachable, allowed model

## Summary

Everything upstream of a review: a tier name goes in, and either a reachable,
credentialed, allowed model comes out or the run ends before a single request is sent.
`internal/reviewer.Resolver` already runs the second half of that sequence for a
hard-coded `openai-codex/gpt-5.5`; this PR puts the assignment chain in front of it and
the family gate inside it.

The ordering is the load-bearing part. **Dynamic providers are refreshed before the
lookup** — `openrouter`, `vercel-ai-gateway`, `nvidia` and `github-copilot` hold no models
at all until then, so looking up first makes a correctly configured tier resolve to "model
not found" and fall back silently, which is the worst shape a bug can take here. And the
**family gate sits between the catalog lookup and the credential check**, so an excluded
model never causes a credential to be resolved for it.

This issue is deliberately library-only: no flags, no CLI, no user-visible surface.
[Issue 05](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md)
exposes it.

## Scope

- The assignment chain, highest layer first
  ([specs 06](../../../planning/specs/06-config-resolution-and-env-overrides.md)):
  1. an explicit `provider/id` supplied by the caller (what `--model` will carry) —
     bypasses tiers entirely;
  2. `EXTERNAL_REVIEWER_TIER_<TIER>` (`…_LIGHT`, `…_STANDARD`, `…_HEAVY`), each holding
     `provider/id`, overriding that one tier and leaving the others alone;
  3. the tier's table from [issue 03](/epic-3-reviewer-selection-integration/issues/03-tier-assignment-config-file.md)'s
     config (which itself resolves `EXTERNAL_REVIEWER_CONFIG` before platform discovery).
- The environment is read through an injected lookup function, not `os.Getenv` at the call
  site, so the chain is testable without `t.Setenv` ordering games.
- Parsing `provider/id`: split at the **first** `/`, both halves non-empty. A malformed
  value is a distinct error from "no assignment" — it is a mistake to report, not a tier
  to skip.
- `Resolver` (in `internal/reviewer`) grows the tier vocabulary and the family gate, and
  runs the sequence [SPECS § Architecture](../../../planning/SPECS.md) fixes:
  refresh-if-dynamic → `GetModel` → **family classification** → `GetAuth`.
- The three outcomes, as typed errors so a single switch at the CLI boundary picks the
  exit code:
  - **no reviewer** (`ErrNoReviewer`, exit 1): no assignment for the tier, model absent
    from the catalog, family excluded, family unknown, provider unconfigured
    (`GetAuth` → `(nil, nil)`);
  - **broken credential** (an error wrapping `*ai.ModelsError` with code `oauth` or
    `auth`, exit 2);
  - success: the model plus the credential's `Source` label, and nothing else.
- A tier assigned an excluded-family model resolves to **no reviewer for that tier** — it
  never falls back to another tier, another model, or the same family by another provider.
- `Resolution` gains whatever the `model` stderr line and the `tiers` command need to name
  the assignment's *source* (flag, environment variable, or config file), since "why did
  this resolve to that?" is the question setup actually asks.

## Out of scope

- `--tier`, `--model`, `--exclude-family` and every other flag — [issue 05](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md).
- The `tiers` and `models` commands — issues [06](/epic-3-reviewer-selection-integration/issues/06-tiers-command.md)
  and [07](/epic-3-reviewer-selection-integration/issues/07-models-command.md).
- Removing `DefaultProviderID` / `DefaultModelID`; they stay until the CLI stops calling
  them in issue 05, so this PR does not have to change `internal/cli` at all.
- Anything credential-shaped beyond calling `GetAuth`: this binary reads no credential,
  stores none and refreshes none ([specs 18](../../../planning/specs/18-auth-and-authorization.md)).
- Any mechanism by which a config file, environment variable or flag can *supply* a family
  — dropped at [specs 05](../../../planning/specs/05-tier-assignment-schema.md).

## Acceptance criteria / Definition of done

Strict red-green, black-box, against `kern-link`'s `faux` provider in a `MutableModels`
registry ([CONVENTIONS § Testing](../../../planning/CONVENTIONS.md)); no network, no live
credential.

- [ ] Written failing first: with a config fixture assigning all three tiers, resolving
      `standard` yields that tier's model and no other.
- [ ] An explicit `provider/id` bypasses the config entirely — asserted with a config
      fixture that assigns something different.
- [ ] `EXTERNAL_REVIEWER_TIER_STANDARD=<provider>/<id>` overrides only `standard`; `light`
      and `heavy` still resolve from the config in the same run.
- [ ] A malformed assignment (`no-slash`, `/id`, `provider/`) returns a *malformed*
      error distinct from `ErrNoReviewer`, and names the value it could not parse.
- [ ] **Refresh precedes resolution**: a registry whose provider reports
      `CanRefreshModels() == true` and holds an empty model list until `Refresh` is called
      resolves successfully — and the same test with the refresh call removed fails.
      (This is the epic's acceptance criterion "a dynamic provider's correctly configured
      tier resolves"; verify by mutation, not by inspection.)
- [ ] A refresh that *fails* is not fatal on its own: the last-known list is used and a
      warning is emitted, matching the behaviour already in `Resolver.Resolve`.
- [ ] A tier assigned an Anthropic-family model — including through a reseller id such as
      `openrouter` + `anthropic/claude-…` — resolves to `ErrNoReviewer`, and the test
      asserts no other model is returned.
- [ ] A tier assigned a model of **unknown** family resolves to `ErrNoReviewer` (excluded,
      not admitted).
- [ ] `GetAuth` returning `(nil, nil)` → `ErrNoReviewer`; `GetAuth` returning an
      `*ai.ModelsError` with code `oauth` or `auth` → an error that is *not*
      `ErrNoReviewer` and unwraps to that `*ai.ModelsError` via `errors.As`.
- [ ] The family gate runs before `GetAuth`: a counting registry asserts `GetAuth` is never
      called for an excluded model.
- [ ] The successful `Resolution` carries the credential's `Source` label and no
      `AuthResult`, keeping "no credential value can reach stdout or stderr" a property of
      the type.
- [ ] `go test -race -ldflags=-s ./... -count=1` green; `golangci-lint` clean.

## Relevant files / areas

- `internal/reviewer/resolve.go`, `internal/reviewer/resolve_test.go` — `Resolver`,
  `Resolution`, `ErrNoReviewer`, `DefaultModels`
- `internal/family/` (from [issue 02](/epic-3-reviewer-selection-integration/issues/02-model-family-classifier.md))
  and `internal/config/` (from [issue 03](/epic-3-reviewer-selection-integration/issues/03-tier-assignment-config-file.md))
  — both new in this epic
- `internal/fauxtest/` — the existing offline registry helpers these tests build on

Governing decisions:
[specs 05 — Tier assignment file schema](../../../planning/specs/05-tier-assignment-schema.md),
[specs 06 — Config resolution & environment overrides](../../../planning/specs/06-config-resolution-and-env-overrides.md),
[specs 04 — Model family classification](../../../planning/specs/04-model-family-classification.md),
[specs 13 — Error handling and failure classification](../../../planning/specs/13-error-handling-and-failure-classification.md),
[SPECS § Architecture](../../../planning/SPECS.md).

## Dependencies

- Blocked by: [02 — family classifier](/epic-3-reviewer-selection-integration/issues/02-model-family-classifier.md),
  [03 — tier assignment config file](/epic-3-reviewer-selection-integration/issues/03-tier-assignment-config-file.md).
- Blocks: [05 — CLI tier flags](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md),
  [06 — tiers command](/epic-3-reviewer-selection-integration/issues/06-tiers-command.md),
  [07 — models command](/epic-3-reviewer-selection-integration/issues/07-models-command.md).

## PR size note

Three layers of precedence (flag, per-tier environment variable, config) plus
refresh-before-lookup ordering plus a family gate sequenced ahead of the credential check —
new behaviour layered onto an existing `Resolver`, the same shape Epic 2 graded L for its
multi-turn loop and dispatch. The precedence chain and its typed errors (`ErrNoReviewer`
versus the wrapped `*ai.ModelsError`) is one half; the refresh/family-gate/credential
sequencing and its mutation-tested ordering guarantees is the other, and where this splits
if it overruns.
