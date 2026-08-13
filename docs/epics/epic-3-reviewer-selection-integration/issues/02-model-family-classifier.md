---
type: Issue
title: "Add the fail-closed model family classifier"
description: "A data table over (Provider, ID) that yields a model's vendor family, treats unknown as excluded, and is proved against the whole embedded catalog."
tags: [epic-3]
timestamp: 2026-08-13T00:13:53Z
resource: https://github.com/julienlegoux/external-reviewer/issues/61
epic: 3
issue: 02
slug: model-family-classifier
size: M
status: open
gh_issue: 61
depends_on: []
---

# Add the fail-closed model family classifier

## Summary

This is the product's central safety property, and `kern-link` does not carry it:
`ai.Model` says nothing about who *made* a model, and `amazon-bedrock` and `openrouter`
both serve Anthropic's. A provider-based filter would send Claude's work back to Claude
([specs 04](../../../planning/specs/04-model-family-classification.md)).

This PR ships the classifier as a self-contained package with no dependency on config,
flags or resolution — a pure function over `(Provider, ID)` plus the rule that decides
whether a family is excluded. [Issue 04](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md)
wires it into resolution; [issue 05](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md)
gives it a flag.

The asymmetry is the whole design: a false exclusion costs a native review — which SCOPE
already calls a normal outcome — while a false inclusion is Claude reviewing Claude while
reporting that it did not, the one failure this product cannot detect after the fact.

## Scope

- A new package (suggested `internal/family`) owning three things: the vendor-provider
  table, the reseller/aggregator table, and the classification function.
- Classification in the order [specs 04](../../../planning/specs/04-model-family-classification.md)
  fixes:
  1. Provider is itself a vendor (`anthropic`, `google`, `google-vertex`, `openai`,
     `openai-codex`, `mistral`, `deepseek`, `xai`, `moonshotai`, `zai`, …) → that vendor
     is the family.
  2. Provider is a reseller or aggregator (`amazon-bedrock`, `openrouter`,
     `vercel-ai-gateway`, `github-copilot`, `azure-openai-responses`, `groq`, `together`,
     `fireworks`, `nvidia`, `cloudflare-*`, …) → the family is the id's vendor segment:
     everything before the first `.` or `/`, after stripping a leading regional qualifier
     (`us.`, `eu.`, `apac.`, `global.`).
  3. Neither rule answers confidently → the family is **unknown**, and unknown is
     **excluded**.
- Matching is case-insensitive and **segment-exact, never substring**: `anthropic.claude-…`
  matches on the segment `anthropic`, so a model called `not-anthropic-clone` is not
  silently reclassified.
- The exclusion rule as a small value the caller builds from a list of family names
  (default `anthropic`) and asks a model about — so "is this model allowed?" is one call
  with one answer, not a predicate every call site re-assembles.
- A table-driven test that runs **the whole embedded `kern-link` catalog** through the
  classifier, plus fixtures for the dynamic-provider id shapes that are absent from the
  embedded files until `Refresh` runs.

## Out of scope

- Any `family` key, config value, environment variable or flag that can *supply* a
  family. Proposed and dropped at [specs 05](../../../planning/specs/05-tier-assignment-schema.md):
  it is the one mechanism able to weaken the fail-closed rule, from a file where being
  wrong is silent. An unknown family is fixed by adding a line to this table, in the
  binary, covered by the catalog-wide test.
- Wiring into resolution or the CLI — [issue 04](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md)
  and [issue 05](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md).
- Any change to `kern-link`'s catalog, or to `go.mod`.
- Listing or printing models — [issue 07](/epic-3-reviewer-selection-integration/issues/07-models-command.md).

## Acceptance criteria / Definition of done

Strict red-green, black-box (`package family_test`), table-driven
([CONVENTIONS § Testing](../../../planning/CONVENTIONS.md)).

- [ ] Written failing first: a table over every model in `kern-link`'s embedded catalog
      asserts that **no** `amazon-bedrock` id beginning `anthropic.` or `<region>.anthropic.`
      classifies as anything but `anthropic`. This test is the executable form of the
      concept's central warning.
- [ ] The same catalog-wide table asserts every entry classifies to a **known** family or
      is reported unknown — and the test names the unknown ones in its failure output, so
      a `kern-link` sync that adds a new id shape fails loudly rather than silently
      degrading to "excluded".
- [ ] Fixture cases for dynamic providers: `openrouter` + `anthropic/claude-opus-4-5` →
      `anthropic`; `openrouter` + `google/gemini-3.1-pro` → `google`;
      `vercel-ai-gateway` + `anthropic/claude-…` → `anthropic`.
- [ ] Regional qualifiers strip: `amazon-bedrock` + `us.anthropic.claude-…` and
      `global.anthropic.claude-…` both → `anthropic`.
- [ ] Substring safety: a vendor-provider model id `not-anthropic-clone` classifies to the
      provider's own family, never to `anthropic`; a reseller id `anthropicx.foo` does not
      classify as `anthropic`.
- [ ] Case-insensitivity: `Anthropic.Claude-…` classifies identically to the lowercase
      form.
- [ ] An unrecognised provider with a bare id classifies as unknown, and the exclusion
      rule **refuses** it — asserted directly, since this is the fail-closed default the
      whole design rests on.
- [ ] The default exclusion list is `anthropic` and is expressed once, in this package.
- [ ] The package carries a package comment stating what it owns
      ([CONVENTIONS § Documentation & comments](../../../planning/CONVENTIONS.md)).
- [ ] `go test -race -ldflags=-s ./... -count=1` green; `golangci-lint` clean.

## Relevant files / areas

- `internal/family/` (new) — no code for this exists yet; the path follows this repo's
  `internal/<thing>` layout ([CONVENTIONS § Repository layout](../../../planning/CONVENTIONS.md))
  rather than being a verified path.
- `internal/reviewer/resolve.go` — the consumer in [issue 04](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md);
  read here for the `ai.Models`/`ai.Model` shapes, not changed.

Governing decisions:
[specs 04 — Model family classification](../../../planning/specs/04-model-family-classification.md),
[scope 06 — Family exclusion rule](../../../planning/scope/06-family-exclusion-rule.md),
[SPECS § Architecture](../../../planning/SPECS.md).

## Dependencies

- Blocked by: None.
- Blocks: [04 — tier resolution chain](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md),
  [07 — models command](/epic-3-reviewer-selection-integration/issues/07-models-command.md).

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
