---
type: Decision
title: "Where the local assignment lives"
description: "Where and in what format does the user record which model serves which tier?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 05
slug: catalogue-location-and-format
status: decided
verdict: "One user-level TOML at the platform config path, with env-var override"
decided_via: triage
depends_on: [reviewer-selection-vocabulary, target-users]
---

# Question

The assignment of models to tiers is written by the user, on the user's machine
([CONCEPT](/CONCEPT.md)). Where does that file live, what format is it, and does it
travel with a repository or with the machine?

# Options

- **One user-level config file** — e.g. `~/.config/external-reviewer/config.toml`
  (and the Windows equivalent). One place, applies to every repo, never committed.
- **Per-repo config committed to the repo** — `.external-reviewer.toml` at the repo
  root. Shareable with collaborators; also means one person's model choices and
  provider accounts get committed into a project.
- **Environment variables only** — `EXTERNAL_REVIEWER_TIER_HEAVY=google/gemini-...`.
  Zero file format to design; unpleasant to maintain three of, and invisible when
  debugging.

# Recommendation

**One user-level config file, with env-var override.** The assignment is a property of
*this machine's credentials*, not of any repository — the concept is explicit that it is
written "against the models their own credentials actually reach". A committed per-repo
file makes a private fact about someone's provider accounts part of a shared artifact and
goes stale for every collaborator but its author.

TOML over JSON or YAML: it is comment-friendly, and this file's whole purpose is to
record a human judgment that deserves a "why" next to it. Resolve the path with the
platform convention (`%APPDATA%` on Windows, `$XDG_CONFIG_HOME` then `~/.config`
elsewhere) so the author-first machine and any later one behave the same.

Env vars stay as a per-invocation override, which is what makes the config testable
without editing it.

# Verdict

**Accepted as recommended.** One user-level TOML file, resolved by platform convention
(`%APPDATA%` on Windows; `$XDG_CONFIG_HOME` then `~/.config` elsewhere), with
per-invocation environment-variable overrides.

The assignment is a property of *this machine's credentials*, not of any repository, so
it never lives in a repo and is never committed. TOML because the file's whole purpose
is to record a human judgment, and a judgment deserves a comment next to it explaining
why that model was chosen for that tier.
