---
type: Issue
title: "Resolve the hard-coded reviewer model and preflight auth"
description: "Add the kern-link dependency and the pre-loop resolution step that turns a hard-coded provider/model into a reachable, credentialed model — or ends the run at exit 1 or 2 before any request is sent."
tags: [epic-1]
timestamp: 2026-08-09T05:24:00Z
epic: 1
issue: 04
slug: model-resolution-and-auth
size: M
status: open
gh_issue: 7
resource: https://github.com/julienlegoux/external-reviewer/issues/7
depends_on: [3]
---

# Resolve the hard-coded reviewer model and preflight auth

## Summary

Resolution runs before the loop and can end a run before a single request is sent, which
is where the load-bearing `1`-versus-`2` distinction actually gets decided: an *absent*
reviewer is a silent native fallback, a *broken* credential is something a human must
fix. This PR adds the `kern-link` dependency and that pre-flight for a single hard-coded
model, with no tier vocabulary and no config file.

## Scope

- `go.mod`: add `github.com/julienlegoux/kern-link` at the tagged **v0.2.0**. No
  `replace` directive is ever committed; `go.work` stays gitignored (issue 01).
- An internal resolution step taking a hard-coded `provider/id` constant and returning a
  reachable, credentialed model, in the order SPECS fixes:
  1. `models.Refresh(ctx, provider)` **first** when `provider.CanRefreshModels()` — these
     providers hold no models until refreshed, so skipping this makes a correctly
     configured provider resolve to "model not found" and fall back silently.
  2. `models.GetModel(provider, id)` → nil ends the run at exit `1`.
  3. `models.GetAuth(ctx, model)` → `(nil, nil)` ends the run at exit `1`; a typed
     `*ai.ModelsError` with code `oauth` or `auth` ends it at exit `2`.
- `AuthResult.Source` — a label such as `OAuth`, never a secret — printed on stderr as a
  diagnostics line. This binary reads no credential, stores none and refreshes none.
- The resolution failures returned as the typed errors issue 03 classifies, so the exit
  code is produced by one switch rather than by each call site.

## Out of scope

- Tier vocabulary, `--tier`, `--model`, `EXTERNAL_REVIEWER_TIER_*`, the TOML config file
  and the `models` / `tiers` subcommands — Epic 3.
- **Family classification** — Epic 3. The hard-coded model here is chosen outside the
  Anthropic family by hand; no classifier ships in this PR.
- Sending any request to the model — issue 05.

## Acceptance criteria / Definition of done

- [ ] Written test-first and black-box, against a `MutableModels` registry with
      `kern-link`'s `faux` provider registered — offline, no network, no bill.
- [ ] A resolvable, credentialed model resolves successfully and the run proceeds past
      pre-flight.
- [ ] A model absent from the catalog → exit `1`, stdout empty, `done` line present.
- [ ] `GetAuth` returning `(nil, nil)` (provider unconfigured, credential absent) →
      exit `1`, stdout empty, `done` line present.
- [ ] `GetAuth` returning `*ai.ModelsError` with code `oauth` → exit `2`; the same with
      code `auth` → exit `2`. Both leave stdout empty and write the reason to stderr.
- [ ] A provider whose `CanRefreshModels()` reports true has `Refresh` called before
      `GetModel`; asserted by a test that fails if the order is inverted.
- [ ] Nothing resembling a credential value reaches stdout or stderr — only
      `AuthResult.Source`.
- [ ] `go test ./... -race` passes offline on a clean checkout; `golangci-lint run`
      passes; CI green on both OSes.
- [ ] Any `kern-link` API that turns out to differ from the names SPECS uses is followed
      as it really is, and the divergence is recorded as a drift record under
      `docs/epics/epic-1-walking-skeleton/drift/` rather than worked around.

## Relevant files / areas

No existing code for this yet, and `kern-link` is a first import here. Expected paths:

- `go.mod`, `go.sum`
- `internal/reviewer/resolve.go`, `internal/reviewer/resolve_test.go`
  (`package reviewer_test`)
- `internal/cli/run.go` (calls resolution before the run proceeds)

The `kern-link` symbol names above (`models.Refresh`, `provider.CanRefreshModels`,
`models.GetModel`, `models.GetAuth`, `ai.ModelsError`, `AuthResult.Source`, the `faux`
provider) are quoted from SPECS, which was written against the upstream docs — verify
each against the v0.2.0 tag before use.

Governing decisions: [SPECS § Architecture](../../../planning/SPECS.md) (the resolution
order), [SPECS § Data, configuration & auth](../../../planning/SPECS.md),
[CONVENTIONS § Dependencies](../../../planning/CONVENTIONS.md) (the frozen dependency
set — this PR adds the first of the two, and no third).

## Dependencies

- Blocked by: [03 — Emit run diagnostics on stderr and classify exit codes](/epic-1-walking-skeleton/issues/03-diagnostics-and-exit-codes.md).
- Blocks: [05 — Call the model once and return its markdown on stdout](/epic-1-walking-skeleton/issues/05-single-turn-model-call.md).

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
