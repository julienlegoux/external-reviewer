---
type: Decision
title: "Reviewer selection vocabulary"
description: "How does a caller ask for a reviewer without naming a vendor?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 04
slug: reviewer-selection-vocabulary
status: decided
verdict: "Weight tiers — light / standard / heavy, reusing implement-epic's grading"
decided_via: triage
depends_on: [target-users]
---

# Question

[CONCEPT](/CONCEPT.md) § "Where the judgment about models lives" fixes the split: the
skills ship the *procedure* and the *vocabulary*; the machine holds the *assignment* of
models to that vocabulary. It also observes that the vocabulary does not have to be
invented, since `implement-epic` already grades work into tiers. What it does not fix is
what the vocabulary actually is — the terms a skill passes on the command line.

# Options

- **Weight tiers (`light` / `standard` / `heavy`)** — the caller states how heavy the
  read is; the local config maps each tier to a model. Mirrors `implement-epic`'s
  size-to-horsepower table.
- **Named roles (`plan-reviewer`, `code-reviewer`, `security-reviewer`)** — the caller
  states what kind of review; config maps roles to models. More expressive, but the
  role list becomes a shipped taxonomy that has to be kept in sync across skills.
- **Capability predicates (`needs: long-context, tool-use`)** — the caller states
  requirements; the binary picks. Most future-proof, most machinery, and it re-imports
  the judgment the concept deliberately pushed onto the user.

# Recommendation

**Weight tiers.** It is the vocabulary the pipeline already speaks — `implement-epic`
grades issues S/M/L and picks horsepower from that grading, and `review-interfaces.md`
already tells the reviewer to "match the model to the weight of the review, not to the
largest number available". Reusing it means no new taxonomy to teach, and a skill
asking for `--tier heavy` is making exactly the judgment the existing prose asks it to
make.

Roles sound richer but push a vendor-shaped judgment back into the shipped artifact,
which is the thing the concept rejected: "security-reviewer" implies somebody decided
which models are good at security. A tier only claims the read is big, which is a fact
the caller genuinely knows.

Three tiers, not five. An unassigned tier simply has no reviewer and the review runs
natively — the concept's rule, and it makes a partial config legitimate rather than
broken.

# Verdict

**Accepted as recommended: weight tiers.** Three of them — `light`, `standard`,
`heavy` — passed by the caller as `--tier <name>`. The vocabulary is the one the
pipeline already speaks: `implement-epic` grades issues by size and picks horsepower
from that grading, and `review-interfaces.md` already instructs a reviewer to match
the model to the weight of the review rather than to the largest number available.

A tier with nothing assigned to it simply has no reviewer, and the review runs
natively — the concept's existing fallback, which makes a partially filled config
legitimate rather than broken.

No role or capability vocabulary: both would push a vendor-shaped judgment back into
the shipped artifact, which is exactly what the concept moved onto the user's machine.
