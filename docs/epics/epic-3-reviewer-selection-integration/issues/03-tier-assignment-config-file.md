---
type: Issue
title: "Read the machine-local tier assignment TOML"
description: "Locate, decode and warn about the user-level config file that assigns a provider and model to each weight tier — never writing it."
tags: [epic-3]
timestamp: 2026-08-13T08:30:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/62
epic: 3
issue: 03
slug: tier-assignment-config-file
size: M
status: pr-open
gh_issue: 62
gh_pr: 80
depends_on: []
---

# Read the machine-local tier assignment TOML

## Summary

The assignment of a weight tier to an actual model is **one hand-written user-level TOML,
never committed** ([specs 05](../../../planning/specs/05-tier-assignment-schema.md)). This
PR ships the half that reads it: locating the file on either platform, decoding it,
reporting every unrecognised key, and returning what each tier was assigned — as data,
with no opinion yet about whether that model exists, is credentialed, or is allowed by
family.

Two properties make this more than a config loader. **The binary never writes config** —
no `init`, no wizard, no prompts — so setup is `tiers` printing what resolved and a human
editing the file. And **unrecognised keys are warned, not ignored**: the failure mode of a
silently dropped `models = ` typo is a tier that mysteriously has no reviewer, with the
one diagnostic that would explain it thrown away at parse time.

This also adds the project's second and last non-stdlib dependency,
`github.com/BurntSushi/toml v1.6.0`, which was decided at specs 05 and named in
[SPECS § Stack](../../../planning/SPECS.md) — it is not a reopening of the dependency
freeze.

## Scope

- `github.com/BurntSushi/toml v1.6.0` added to `go.mod` (pinned, no `replace`), and
  nothing else.
- A new package (suggested `internal/config`) that:
  - **locates** the file: `EXTERNAL_REVIEWER_CONFIG` when set, else
    `os.UserConfigDir()/external-reviewer/config.toml`. `os.UserConfigDir` is the one
    stdlib call that implements the `%APPDATA%` / `$XDG_CONFIG_HOME` split — a
    hand-rolled `runtime.GOOS` switch is where a POSIX assumption would first appear
    ([specs 06](../../../planning/specs/06-config-resolution-and-env-overrides.md)).
  - **decodes** `[tiers.<name>]` tables with `provider` and `model` string keys, both
    `kern-link`'s own identifiers verbatim — no aliasing, no translation layer.
  - **reports** the path it looked at, so callers can print it rather than re-deriving it.
  - **reports every unrecognised key** from `MetaData.Undecoded()` as a warning in SPECS'
    exact wording: `config: unrecognised key "models" in [tiers.standard]`.
- An **absent file is not an error**: it yields zero assignments plus the path that was
  looked at. Malformed TOML *is* an error (something is wrong that a human must fix).
- Every path built with `filepath.Join`; a path printed to a diagnostic is printed as the
  OS renders it, which is the one place a native path legitimately reaches stderr
  ([CONVENTIONS § Paths and platforms](../../../planning/CONVENTIONS.md) — a file path on
  the user's disk is an OS path, not a wire path).
- Nothing in this package opens a file for writing, and no write-API call appears in it —
  the `forbidigo` block in CI enforces that
  ([CONVENTIONS § Code style & formatting](../../../planning/CONVENTIONS.md)).

## Out of scope

- The `--model` flag and the `EXTERNAL_REVIEWER_TIER_<TIER>` environment overrides — the
  layers above this file in the precedence chain, and
  [issue 04](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md)'s
  job.
- Anything that asks whether an assignment is *reachable*: catalog lookup, refresh,
  credentials, family. This package returns what the file said, nothing more.
- A `family` key. It was proposed and dropped, so here it is simply an unrecognised key
  like any other — warned, never honoured.
- Creating, templating or migrating the file. There is no `init` and no migration story
  ([specs 20](../../../planning/specs/20-data-migrations.md)).
- The `tiers` command — [issue 06](/epic-3-reviewer-selection-integration/issues/06-tiers-command.md).

## Acceptance criteria / Definition of done

Strict red-green, black-box, with fixtures written into `t.TempDir()` and reached through
`EXTERNAL_REVIEWER_CONFIG` — **no test ever reads or writes the developer's real profile**.

- [ ] Written failing first: a fixture holding the three-tier example from
      [specs 05](../../../planning/specs/05-tier-assignment-schema.md) decodes to three
      assignments carrying `provider` and `model` verbatim.
- [ ] An absent file returns zero assignments, no error, and the path that was looked at.
- [ ] `EXTERNAL_REVIEWER_CONFIG` takes precedence over platform discovery; with it unset,
      the located path is `os.UserConfigDir()` joined with `external-reviewer` and
      `config.toml` (asserted against `os.UserConfigDir()` itself, so the test states the
      rule rather than a per-OS literal).
- [ ] `models = "gpt-5.5"` in `[tiers.standard]` produces exactly one warning reading
      `config: unrecognised key "models" in [tiers.standard]`, and that tier's assignment
      is reported as having no model — not silently half-filled.
- [ ] `family = "openai"` produces the same unrecognised-key warning and is not honoured
      anywhere.
- [ ] A tier table missing `provider` or `model` reports as unassigned with a warning
      naming which key is missing, rather than producing a half-assignment.
- [ ] Malformed TOML returns an error (classified as exit 2 by
      [issue 05](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md)),
      and the error names the file being read.
- [ ] A tier name outside `light`/`standard`/`heavy` is warned and ignored.
- [ ] `go.mod` gains exactly one `require`, at `v1.6.0`; `go.sum` is updated; no `replace`
      directive is committed; `go build ./...` clean.
- [ ] `go test -race -ldflags=-s ./... -count=1` green; `golangci-lint` clean.

## Relevant files / areas

- `internal/config/` (new) — no code for this exists yet; the path follows this repo's
  `internal/<thing>` layout rather than being a verified path.
- `go.mod`, `go.sum` — the second and last dependency
- `docs/planning/SPECS.md` § Stack already names `BurntSushi/toml v1.6.0`; no amendment
  should be needed. If the installable version differs from v1.6.0, that is drift — record
  it as a drift record rather than silently pinning something else
  ([DRIFT](../../../planning/DRIFT.md) has the precedent in Epic 1's entry 04).

Governing decisions:
[specs 05 — Tier assignment file schema](../../../planning/specs/05-tier-assignment-schema.md),
[specs 06 — Config resolution & environment overrides](../../../planning/specs/06-config-resolution-and-env-overrides.md),
[CONVENTIONS § Dependencies](../../../planning/CONVENTIONS.md).

## Dependencies

- Blocked by: None.
- Blocks: [04 — tier resolution chain](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md),
  [06 — tiers command](/epic-3-reviewer-selection-integration/issues/06-tiers-command.md).

## PR size note

The locate-and-decode core (`os.UserConfigDir`, `[tiers.<name>]` decoding into
`provider`/`model`) is the small half; the warning taxonomy is where the bulk is — an
unrecognised key, a missing `provider` or `model`, malformed TOML, and an out-of-vocabulary
tier name each get their own acceptance criterion and their own `t.TempDir()` fixture. If
this grows, that taxonomy is the natural split, since the locate-and-decode core has no
reason to change once it passes.
