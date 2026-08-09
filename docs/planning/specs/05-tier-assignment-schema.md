---
type: Decision
title: "Tier assignment file schema"
description: "The TOML shape of the machine-local tier→model assignment, and what decodes it."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 05
slug: tier-assignment-schema
status: decided
verdict: "Per-tier [tiers.X] table with provider + model, BurntSushi/toml with Undecoded() reporting; no family key; a models subcommand for discovery"
decided_via: discussion
depends_on: [cli-surface-and-argv, model-family-classification]
---

# Question

Scope decided *that* the assignment is one user-level TOML file, never committed, and
*why* TOML — the file records a human judgment that deserves a comment explaining it
([decision](/scope/05-catalogue-location-and-format.md)). It did not decide the schema,
and the schema is the surface a human hand-edits, so renaming a key breaks every machine
that already has the file.

Also open: what decodes it. Go has no TOML in the standard library, so this is the
project's second and last non-stdlib dependency.

# Options

**Schema**

- **A table per tier: `[tiers.standard]` with `provider` / `model` keys.** Room for
  per-tier settings (thinking level, an explicit family override) without a second
  schema, and every entry has an obvious place for the comment TOML was chosen for.
- **A flat map: `standard = "google/gemini-3.1-pro-preview"`.** Shortest possible file,
  and it has nowhere to put anything but the model — the first per-tier setting forces a
  migration of a file that lives on user machines.

**Decoder**

- **`BurntSushi/toml` v1.6.0.** The long-stable one; `MetaData.Undecoded()` reports keys
  it did not recognise.
- **`pelletier/go-toml/v2` v2.4.3.** Faster, actively maintained, `DecodeError` carries
  line/column context; strict mode is opt-in per decoder.

# Recommendation

**A table per tier, decoded by `BurntSushi/toml` v1.6.0, with unrecognised keys reported
rather than ignored.**

```toml
# Reviewer assignment. Judgments about which model suits which weight of review.
# Never committed to a repository.

[tiers.light]
provider = "google"
model    = "gemini-3.1-flash-lite"
# Issue files against one epic — a cheap read, and volume matters more than depth.

[tiers.standard]
provider = "google"
model    = "gemini-3.1-pro-preview"

[tiers.heavy]
provider = "google"
model    = "gemini-3.5-flash"
# An entire epic's merged diff. Wants the largest context reachable on this machine.
```

`provider` and `model` are `kern-link`'s own `ProviderId` and `Model.ID` verbatim — no
translation layer, no vendor aliasing. A tier whose table is absent, or whose model is
not in the catalog, or whose family is excluded, resolves to *no reviewer for that
tier*, which scope already defines as a normal outcome rather than an error
([decision](/scope/06-family-exclusion-rule.md)).

`BurntSushi/toml` over `pelletier` on one argument: `MetaData.Undecoded()`. This file is
hand-written, rarely, by someone who will not remember the key names — and the failure
mode of a silently ignored typo (`models = ` instead of `model = `) is a tier that
mysteriously has no reviewer, with the diagnostic that would explain it thrown away at
parse time. Every unrecognised key gets a stderr warning naming it. `pelletier` is the
better library on speed and error position; neither matters for a twelve-line file read
once per run.

Absent config is **not** an error: `tiers` reports every tier as unassigned, and `review`
exits `1` (capability absent, caller falls back natively) — journey 2 in scope
([decision](/scope/10-core-user-journeys.md)).

# Verdict

Option A as recommended — `provider` and `model` explicit per tier, `BurntSushi/toml`
v1.6.0, unrecognised keys warned rather than ignored — with three amendments from the
discussion.

**No `family` key.** A rescue-only override (consulted only when inference returns
unknown, never able to overrule a confident answer) was proposed and dropped. It existed
solely for providers outside the validated path, and it is the only mechanism in the
design that can weaken the fail-closed guarantee of
[decision 04](/specs/04-model-family-classification.md). When a provider does resolve to
an unknown family, the fix is a line of data in the classifier table — shipped by the
person who owns the binary, covered by the catalog-wide test — rather than a config key
on a user's disk that can defeat a safety property by being wrong.

**A `models` subcommand.** `external-reviewer models [--provider X] [--refresh] [--all]`
lists the intersection of the catalog, what this machine's credentials reach, and what
the family rule allows, with price and context window per model. `kern-link` supports
this directly — `GetModels`, `Refresh`, `GetAuth`, and the catalog's `Cost`/
`ContextWindow`/`MaxTokens` — so discovery is a query rather than a document that rots.
`--refresh` is opt-in because it costs network calls.

**Dynamic providers are refreshed before resolution.** `openrouter`,
`vercel-ai-gateway`, `nvidia` and `github-copilot` start with an empty model list;
`GetModel` returns nil until `Refresh` runs. So resolution is: if
`provider.CanRefreshModels()`, `Refresh(ctx, provider)` first. Without it every dynamic
provider resolves to "model not found" and exits `1` — a silent native fallback for a
tier that was correctly configured, which is the worst shape a bug can take here.

**Validation target: `openai-codex`.** The mechanism stays fully generic — no provider is
named anywhere in the binary except the family classifier's data table — but v1 is
*validated* against OpenAI over OAuth alone. Everything else is "the mechanism is
generic, only OpenAI is tested", stated in the README so it reads as an honest boundary
rather than an implied promise. This supersedes the scope-era assumption that Google
would be the credentialed family ([risk 2](/scope/18-risks-and-assumptions.md)).
