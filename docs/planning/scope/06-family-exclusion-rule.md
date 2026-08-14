---
type: Decision
title: "Who enforces the family exclusion"
description: "The binary or the caller — who guarantees the reviewer is not from the Anthropic family?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 06
slug: family-exclusion-rule
status: decided
verdict: "The binary enforces it, defaulting to --exclude-family anthropic"
decided_via: triage
depends_on: [reviewer-selection-vocabulary, catalogue-location-and-format]
---

# Question

[CONCEPT](/CONCEPT.md) § "'External' means the family, not the model" settles the
*principle*: selection is by model family, never by provider, because `amazon-bedrock`
serves Anthropic's models too and a provider-based filter would send Claude's work back
to Claude. The mechanism is open: does the binary enforce this, or is it the caller's
responsibility to configure sensibly?

# Options

- **Binary enforces, with a default** — the binary knows the calling family (default
  `anthropic`) and refuses any configured model from that family, exiting as "no
  reviewer available". Wrong configs fail loudly and the guarantee holds for every
  caller.
- **Caller enforces** — the skill picks a model id and the binary runs whatever it is
  told. Simplest binary; the guarantee is re-implemented in every calling skill and
  quietly lost the first time one forgets.
- **Config-time only** — a `validate` subcommand warns about same-family entries but
  a run never blocks. Visible, but a warning printed months earlier is not a guarantee.

# Recommendation

**Binary enforces, with `--exclude-family anthropic` as the default.** The entire
product is the claim "this reviewer did not write the thing", and a claim that depends
on every caller remembering to check is not a claim. Enforcing in one place also means
the trap the concept identified is coded once, next to `kern-link`'s catalogue where
the family metadata actually lives — rather than restated in prose in each skill.

The flag rather than a hard-coded constant, because the family to exclude is a property
of the *calling* session, and the day a non-Claude harness calls this binary the rule
must still mean "not the family that wrote it". Same-family assignment is treated as
*no reviewer for that tier* — the concept's existing fallback — not as a crash.

# Verdict

**Accepted as recommended: the binary enforces it**, with `--exclude-family anthropic`
as the default. A tier assigned a model from the excluded family resolves to *no
reviewer for that tier* — the concept's existing fallback — rather than an error.

The product's entire claim is "this reviewer did not write the thing", and a claim that
depends on every caller remembering to check is not a claim. Enforcing in one place also
codes the `amazon-bedrock` trap once, next to `kern-link`'s catalogue where the family
metadata actually lives, instead of restating it as prose in every calling skill.

The flag exists rather than a hard-coded constant because the family to exclude is a
property of the *calling* session: the rule must still mean "not the family that wrote
it" the day a non-Claude harness invokes this binary.
