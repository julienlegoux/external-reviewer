---
type: Issue
title: "Add the tiers command"
description: "Report which tiers resolve to a reachable model on this machine right now, the config path that was read, and why a tier does not resolve."
tags: [epic-3]
timestamp: 2026-08-13T16:00:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/65
epic: 3
issue: 06
slug: tiers-command
size: M
status: done
gh_issue: 65
gh_pr: 87
depends_on: [4, 5]
---

# Add the tiers command

## Summary

Setup for this binary is not a wizard — it is `tiers` printing what resolved and a human
editing the file ([specs 06](../../../planning/specs/06-config-resolution-and-env-overrides.md)).
That makes this command the whole of the setup story, and the only place a misconfigured
machine explains itself.

**Discovery is a query, not a document**: the answer is what this machine's config,
catalog, credentials and family rule produce *right now*, never a shipped list that starts
rotting the day it is written.

## Scope

- `external-reviewer tiers [--exclude-family anthropic[,…]]`, its own `FlagSet`, long-form
  flags only.
- Output on **stdout** — for `tiers` the report *is* the command's answer, unlike `review`
  where stdout belongs exclusively to the model's final text
  ([specs 12](../../../planning/specs/12-output-and-diagnostics-format.md)).
- One row per tier (`light`, `standard`, `heavy`), each carrying:
  - the assignment and **where it came from** — the config file, an
    `EXTERNAL_REVIEWER_TIER_<TIER>` variable, or nothing;
  - whether it resolves to a reachable model, and when it does not, **which rule refused**:
    unassigned, model absent from the catalog, family excluded (naming the family),
    family unknown, provider unconfigured, credential broken.
- The **config path that was looked at**, printed whether or not the file exists — a
  machine with no config must be able to learn where to put one.
- Warnings from config decoding (unrecognised keys, missing keys) go to **stderr** as
  `warn` lines, so the stdout table stays parseable while the typo that broke a tier is
  still visible. This is the epic's acceptance criterion that an unrecognised key produces
  a `warn` line rather than a silently missing tier.
- Dynamic providers are refreshed before resolution here too, for the same reason as in
  `review`: without it a correctly configured `openrouter` tier reports as unreachable.
- Exit `0` whenever the query was answered — including when **no** tier resolves, since
  that is a true answer rather than a failure. Exit `2` only when the query could not be
  answered at all (malformed TOML, a credential store that cannot be located).

## Out of scope

- Writing, creating or templating the config file. There is no `init`, no wizard, no
  "shall I create this for you?" ([scope 12](../../../planning/scope/12-non-goals.md)).
- Listing models — [issue 07](/epic-3-reviewer-selection-integration/issues/07-models-command.md).
- Any credential value on any stream; the only credential-adjacent value that ever reaches
  a stream is `AuthResult.Source`, a label like `OAuth`
  ([specs 18](../../../planning/specs/18-auth-and-authorization.md)).
- Colour, spinners or interactive output — the caller is a script.

## Acceptance criteria / Definition of done

Strict red-green, black-box in `package cli_test` through `Run`, against an offline
registry and `t.TempDir()` config fixtures.

- [ ] Written failing first: `tiers` with a fixture assigning all three tiers to reachable
      models prints one row per tier, each naming `provider/id`, and exits `0`.
- [ ] With no config file at all: every tier reports unassigned, the path that was looked
      at is printed, and the exit code is `0`.
- [ ] With the `models = ` typo fixture: exactly one `warn` line on stderr naming the key
      and the tier, that tier reported unassigned on stdout, exit `0`.
- [ ] A tier whose model is Anthropic-family reports **excluded by family**, naming
      `anthropic` — not "model not found", which would send the reader looking for a
      catalog problem that does not exist.
- [ ] A tier whose model is of unknown family reports as excluded, not as reachable.
- [ ] A tier whose provider has no credential reports **provider unconfigured**, distinctly
      from a broken credential.
- [ ] `EXTERNAL_REVIEWER_TIER_STANDARD` set: the `standard` row names the environment
      variable as the source, and the other two rows still read from the config.
- [ ] `tiers --exclude-family google` flips a Google-family tier from resolving to
      excluded, in the same run shape as the default.
- [ ] Stdout carries the table and nothing else; every warning and diagnostic is on stderr.
- [ ] `tiers --nope` → exit `2`; `tiers extra-positional` → exit `2`.
- [ ] No credential value appears on either stream — asserted on the full captured output
      of a run whose fixture credential has a recognisable secret value.
- [ ] `usageText` documents the command; `go test -race -ldflags=-s ./... -count=1` green;
      `golangci-lint` clean.

## Relevant files / areas

- `internal/cli/run.go` — the subcommand switch in `run`
- `internal/cli/usage.go` — `usageText`
- `internal/cli/tiers.go` (new), `internal/cli/tiers_test.go` (new)
- `internal/config/`, `internal/family/`, `internal/reviewer/resolve.go` — all consumed,
  none redesigned here

Governing decisions:
[specs 05 — Tier assignment file schema](../../../planning/specs/05-tier-assignment-schema.md),
[specs 06 — Config resolution & environment overrides](../../../planning/specs/06-config-resolution-and-env-overrides.md),
[specs 12 — Output and diagnostics format](../../../planning/specs/12-output-and-diagnostics-format.md),
[scope 10 — Core user journeys](../../../planning/scope/10-core-user-journeys.md) (journey 3,
first-time setup).

## Dependencies

- Blocked by: [04 — tier resolution chain](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md),
  [05 — CLI tier flags](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md)
  (which establishes `--exclude-family`'s parsing and the usage-error shape this command
  reuses).
- Blocks: [07 — models command](/epic-3-reviewer-selection-integration/issues/07-models-command.md)
  (not a functional dependency — a sequencing one: both rewrite `internal/cli/run.go`'s
  subcommand switch and `internal/cli/usage.go`'s `usageText`, so landing them in parallel
  would produce a conflict neither PR's tests would catch), and
  [10 — review-interfaces swap](/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md),
  whose capability probe and setup instructions point at this command.

## PR size note

The per-tier resolution report (source, reachability, which rule refused) is the core; the
CLI wiring — the new `FlagSet`, `usageText`, the sequencing-only edit to `run.go`'s
subcommand switch shared with issues 07 and 08 — is the small remainder and the natural
split if it grows.
