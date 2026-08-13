---
type: Issue
title: "Add the models command"
description: "List the intersection of the catalog, this machine's credentials and the family rule, with price and context window per row."
tags: [epic-3]
timestamp: 2026-08-13T07:10:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/66
epic: 3
issue: 07
slug: models-command
size: M
status: open
gh_issue: 66
depends_on: [2, 4, 6]
---

# Add the models command

## Summary

The other half of discovery-as-a-query: `tiers` answers *what is configured*, `models`
answers *what could be*. It is what makes journey 3 (first-time setup) and journey 4
(judging a model by hand before trusting it with a tier) something other than trial and
error ([scope 10](../../../planning/scope/10-core-user-journeys.md)).

A shipped list of recommended models was considered and refused — it would start rotting
the day it was written ([specs 05](../../../planning/specs/05-tier-assignment-schema.md)).
`kern-link` supports the query directly through `GetModels`, `Refresh`, `GetAuth` and the
catalog's own `Cost` / `ContextWindow` / `MaxTokens`, so this command reads numbers that
already exist rather than building an accounting layer.

## Scope

- `external-reviewer models [--provider <id>] [--refresh] [--all]`, its own `FlagSet`,
  long-form flags only, output on stdout.
- Default rows are the **intersection** of three filters: the catalog, what this machine's
  credentials reach, and what the family rule allows.
- `--all` drops the credential and family filters and shows the whole catalog — the escape
  hatch for "what exists at all", and the one place an excluded family may legitimately be
  printed, which the row must then say plainly.
- `--provider <id>` restricts to one provider; an id no provider matches is a usage error
  (exit `2`), because a typo silently printing nothing is indistinguishable from a
  provider with no models.
- `--refresh` is opt-in, because it costs network calls, and refreshes only providers
  reporting `CanRefreshModels()`.
- Each row carries `provider/id`, price (from the catalog's `Cost`, per million tokens,
  input and output) and context window. Rows are sorted deterministically (provider, then
  id) so two runs of an unchanged machine produce identical output.
- Everything that is not a row — warnings, refresh failures — goes to stderr, so the table
  stays cuttable by a script.

## Out of scope

- Recommending a model, ranking models, or annotating any row with a judgment. The file
  holds the human judgment; this command holds the facts.
- Writing anything, including a cache of a refreshed catalog.
- Assigning a model to a tier: the binary never writes config.
- The `tiers` command — [issue 06](/epic-3-reviewer-selection-integration/issues/06-tiers-command.md).

## Acceptance criteria / Definition of done

Strict red-green, black-box in `package cli_test` through `Run`, against an offline
registry ([CONVENTIONS § Testing](../../../planning/CONVENTIONS.md)); no network.

- [ ] Written failing first: with a registry holding two providers, one credentialed and
      one not, `models` lists only the credentialed provider's models, exit `0`.
- [ ] `--all` lists both providers' models, and marks the rows the default view filters
      out (uncredentialed, excluded family) rather than printing them indistinguishably.
- [ ] An Anthropic-family model never appears without `--all` — including a reseller id
      such as `openrouter` + `anthropic/claude-…`, asserted on the full stdout.
- [ ] `--refresh` calls `Refresh` exactly once per provider reporting
      `CanRefreshModels()`, and never for one that does not — asserted with a counting
      registry.
- [ ] Without `--refresh`, no `Refresh` call is made at all.
- [ ] A refresh that fails emits a `warn` line and still prints the rows the last-known
      catalog holds; exit stays `0`.
- [ ] A credentialed provider reporting `CanRefreshModels()` whose catalog is empty is named
      on stdout as **needing `--refresh`**, rather than rendering as an absent row — asserted
      with a registry holding exactly one such provider.
- [ ] `--provider <unknown>` → exit `2`, `stop=usage`, one `error:` line.
- [ ] Rows are sorted by provider then id; running the same command twice against an
      unchanged registry produces byte-identical stdout.
- [ ] Price and context window come from the catalog's own fields; a model whose catalog
      entry carries no price renders a placeholder rather than `0`, so free and unknown are
      not confused.
- [ ] No credential value reaches either stream.
- [ ] `usageText` documents the command; `go test -race -ldflags=-s ./... -count=1` green;
      `golangci-lint` clean.

## Relevant files / areas

- `internal/cli/run.go` — the subcommand switch in `run`
- `internal/cli/usage.go` — `usageText`
- `internal/cli/models.go` (new), `internal/cli/models_test.go` (new)
- `internal/family/` (from [issue 02](/epic-3-reviewer-selection-integration/issues/02-model-family-classifier.md)),
  `internal/reviewer/resolve.go` — `DefaultModels`, the registry builder
- `internal/fauxtest/` — the offline registry helpers

Governing decisions:
[specs 05 — Tier assignment file schema](../../../planning/specs/05-tier-assignment-schema.md)
(the `models` subcommand and why discovery is a query),
[specs 02 — CLI surface & argv grammar](../../../planning/specs/02-cli-surface-and-argv.md),
[SPECS § Data, configuration & auth](../../../planning/SPECS.md).

## Dependencies

- Blocked by: [02 — family classifier](/epic-3-reviewer-selection-integration/issues/02-model-family-classifier.md),
  [04 — tier resolution chain](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md)
  (for the registry and refresh behaviour it reuses),
  [06 — tiers command](/epic-3-reviewer-selection-integration/issues/06-tiers-command.md).
  Not a functional dependency on 06 — a sequencing one: both rewrite `internal/cli/run.go`'s
  subcommand switch and `internal/cli/usage.go`'s `usageText`, and landing them in parallel
  produces a conflict neither PR's tests would catch.
- Blocks: None.

## PR size note

The three-filter intersection (catalog × credentials × family) plus deterministic sort is
the core; `--all`, `--refresh` and `--provider` and their own usage-error and warning paths
are additive on top of it and the natural split if this grows.
