---
type: Decision
title: "Config resolution & environment overrides"
description: "Where the config file is looked for on each platform, and which environment variables override it."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 06
slug: config-resolution-and-env-overrides
status: decided
verdict: "--model then EXTERNAL_REVIEWER_TIER_<TIER> then EXTERNAL_REVIEWER_CONFIG then os.UserConfigDir(); never written"
decided_via: triage
depends_on: [tier-assignment-schema]
---

# Question

Scope fixed the platform convention — `%APPDATA%` on Windows, `$XDG_CONFIG_HOME` then
`~/.config` elsewhere — and said there are per-invocation environment overrides
([decision](/scope/05-catalogue-location-and-format.md)). It did not name the file, the
directory, the variables, or the precedence. Every one of those is baked into a machine
the moment setup happens, and into the skills the moment they set an override.

# Options

- **Full precedence chain: flag → per-tier env → config-path env → config file.** Each
  layer has a distinct reason to exist; four layers to document and test.
- **Config file plus a single path override.** Minimal, and it makes "try this model
  once" mean "edit the file", which is journey 4 done badly.
- **Environment only, no file.** Nothing to locate, and it throws away the comments that
  were the entire reason TOML was chosen.

# Recommendation

**The full chain, four layers, highest first:**

1. `--model provider/id` on the command line ([decision](/specs/02-cli-surface-and-argv.md))
   — bypasses tiers entirely.
2. `EXTERNAL_REVIEWER_TIER_<TIER>` (`…_LIGHT`, `…_STANDARD`, `…_HEAVY`), each holding
   `provider/id` — overrides that one tier's table, leaving the others alone.
3. `EXTERNAL_REVIEWER_CONFIG` — an absolute path to the TOML, overriding platform
   discovery. This is what makes the config path testable without writing into the
   developer's real profile, and it is how CI reads a fixture.
4. Platform discovery:
   - Windows: `%APPDATA%\external-reviewer\config.toml`
   - otherwise: `$XDG_CONFIG_HOME/external-reviewer/config.toml`, then
     `$HOME/.config/external-reviewer/config.toml`

Resolution uses `os.UserConfigDir`, which already implements exactly that split — one
stdlib call rather than a hand-rolled `runtime.GOOS` switch that would be the first
place a POSIX assumption creeps in ([scope constraint](/scope/15-constraints.md)).

Every path is joined with `filepath.Join` and every path printed to stderr is printed as
the OS renders it. No path is ever built by string concatenation with `/`.

The binary reads config and **never writes it**. There is no `init`, no wizard, no
"shall I create this for you?" — scope rules out interactive UI of any kind
([decision](/scope/12-non-goals.md)). Setup is: the `tiers` command prints the path it
looked at and what resolved, and the human edits the file. Credentials are not in this
chain at all; they resolve through `kern-link`
([decision](/specs/18-auth-and-authorization.md)).

# Verdict

Accepted at triage as recommended. `os.UserConfigDir` implements the platform split in one stdlib call rather than a hand-rolled GOOS switch, which is where a POSIX assumption would first creep in.
