---
type: Epic
title: "Reviewer selection and pipeline integration"
description: "Weight tiers, the machine-local assignment, binary-enforced family exclusion, caps sized from real measurements, and the review-interfaces.md swap that retires opencode."
tags: [epic]
timestamp: 2026-08-09T04:34:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/3
epic: 3
slug: reviewer-selection-integration
status: open
gh_issue: 3
milestone: 3
source: docs/planning/SCOPE.md#milestone-3-reviewer-selection-and-pipeline-integration
---

# Epic 3: Reviewer selection and pipeline integration

## Goal

What makes the pipeline call the binary instead of the author calling it. This is the
epic that retires `opencode`, and the first in which the binary runs with no human
watching.

It is also where the product's central safety property gets built, because `kern-link`
does not carry it: `ai.Model` says nothing about who made a model, and `amazon-bedrock`
and `openrouter` both serve Anthropic's. A provider-based filter would send Claude's
work back to Claude.

## Scope

- **Weight tiers as the request vocabulary**: `--tier light|standard|heavy`, reusing the
  grading `implement-epic` already applies to work. A skill asks for a kind of reviewer
  and never names a vendor.
- **The machine-local assignment**: one hand-written user-level TOML, never committed,
  resolved through `os.UserConfigDir()` (which implements the `%APPDATA%` /
  `$XDG_CONFIG_HOME` split in one stdlib call). `provider` and `model` are `kern-link`'s
  own identifiers verbatim — no aliasing layer. Unrecognised keys are **warned, not
  ignored**. The binary never writes config: no `init`, no wizard, no prompts.
- **Resolution order**, ending the run before any request is sent when it fails:
  `--model` → `EXTERNAL_REVIEWER_TIER_<TIER>` → `EXTERNAL_REVIEWER_CONFIG` →
  `os.UserConfigDir()/external-reviewer/config.toml`. Dynamic providers
  (`openrouter`, `vercel-ai-gateway`, `nvidia`, `github-copilot`) are refreshed
  **before** resolution — they hold no models until then, so skipping it makes a
  correctly configured tier fall back silently.
- **Family classification and exclusion enforced by the binary**: a data table over
  `(Provider, ID)` where a vendor provider yields its own family, a reseller or
  aggregator yields the id's vendor segment after stripping a regional qualifier, and
  anything else is **unknown, treated as excluded**. `--exclude-family anthropic` by
  default. A tier assigned an excluded-family model resolves to *no reviewer for that
  tier*, not an error.
- **Discovery as a query, not a document**: a `tiers` command reporting which tiers
  resolve to a reachable model on this machine right now, and a `models` command listing
  the intersection of the catalog, this machine's credentials and the family rule, with
  price and context window per row.
- **Loop caps, sized from Epic 2's measurements** — turns, cost and wall clock — plus
  the decision of which to expose as flags, made against real numbers. An exceeded bound
  stops the loop and returns what the run has, rather than failing.
- **The `review-interfaces.md` swap**: probe and call `external-reviewer`, remove the
  `opencode` branch, leave the native fallback unchanged and unconditional.

## Out of scope

- A `family` key or any rescue override in the config file. Proposed and dropped — it
  was the one mechanism able to weaken the fail-closed family rule, living on a user's
  disk where being wrong is silent.
- A model catalogue or list of recommended models shipped inside the skills; a stale-
  catalogue signalling mechanism; packaging, signed releases, install docs or config
  wizards. All named non-goals.
- A second delegation path kept beside `opencode` for compatibility. v1 has exactly one
  user and that user will have the new binary.

## Acceptance criteria

The three success criteria SCOPE sets for v1 land here:

- `_shared/review-interfaces.md` no longer mentions `opencode`, and the external pass
  runs through `external-reviewer`.
- A real `review-epics` or `review-issues` run on this project's own artifacts produced
  leads, at least some of which survived verification against the files.
- A standard-tier review's measured **wall-clock time and quota headroom** are recorded,
  and the author judges them low enough to reach for the tool again. (Token and notional
  cost figures are recorded too, as the volume proxy that becomes real money if a tier is
  ever assigned an API-key provider.)

Plus the mechanics they rest on:

- A tier assigned an excluded-family model resolves to no reviewer for that tier and
  exits `1` — not an error, and never a fallback to a same-family model.
- A model of unknown family is excluded, not admitted.
- A dynamic provider's correctly configured tier resolves, proving refresh precedes
  resolution.
- An absent credential exits `1`; a broken one exits `2`.
- `tiers` shows which tiers resolve, and an unrecognised config key produces a `warn`
  line rather than a silently missing tier.
- The whole embedded catalog passes through the family classifier in a table-driven test.

## Dependencies

[Epic 2: Read-only agentic loop](/epic-2-read-only-agentic-loop/EPIC_2.md) — and not
only for the code. The caps in this epic's scope cannot be sized until Epic 2's real-run
measurements exist.

## Context

- [Technical specs](../../planning/SPECS.md) — the tier schema, config resolution, the
  family classifier and the discovery commands.
- [Conventions](../../planning/CONVENTIONS.md) — repo standards, and the scoping rule
  for the cross-repository work below.

## Notes

- **⚠ This epic spans two repositories.** SCOPE carries a note addressed to
  `split-epics`: the `_shared/review-interfaces.md` work lands in the **lx skills
  repository** (`julienlegoux/skills`), not in this one. Every other item above is Go
  code here. `create-issues` will therefore have to cut this epic across two repos —
  the one place the pipeline's single-repo assumption breaks, so it is stated here
  rather than discovered.
- **Per [conventions decision 04](../../planning/conventions/04-cross-repo-conventions.md),
  the skills-repo issues follow *that* repository's own authoring contract, branch model
  and review gate.** This project's CONVENTIONS.md does not travel with them; what
  travels is the issue's acceptance criteria.
- **`create-issues` for this epic is deliberately deferred** and will be run separately,
  by the user's decision at split time. Epics 1 and 2 can be cut and implemented without
  it.
- **Project-wide, arriving for real here**: from this epic on the binary runs
  unsupervised inside a script, on the user's own credentials. On the current
  subscription that makes SCOPE's risk 4 **quota and rate-limit exhaustion** — shared
  with everything else on the account — rather than a bill, which is why turns and wall
  clock are the meaningful bounds and a cost cap is not.
- **Generic mechanism, one validated provider.** The binary names no provider anywhere
  except the family classifier's table, so all ~35 remain reachable; v1 is *validated*
  against `openai-codex` alone. That is an honest boundary, not an implied promise.
- **Assumed and unenforceable (SCOPE risk 6)**: everything downstream of a mediocre
  review is safe only because the calling skill verifies each lead against the files
  before it enters a report. This epic wires up that caller; it cannot make it verify.
