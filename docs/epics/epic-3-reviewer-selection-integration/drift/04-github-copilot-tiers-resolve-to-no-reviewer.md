---
type: Drift record
title: "A github-copilot tier resolves to no reviewer, so one of the four dynamic providers cannot serve a tier at all"
description: "Refresh-before-lookup works for github-copilot, and then the family gate refuses all 25 of its models because none carries a vendor segment — the epic names it as a provider whose correctly configured tier resolves, and it cannot. Reported rather than fixed: the fix is a specs 04 rule this issue does not own."
tags: [epic-3, drift]
timestamp: 2026-08-13T14:10:00Z
epic: 3
issue: 04
---

# A github-copilot tier resolves to no reviewer, so one of the four dynamic providers cannot serve a tier at all

## Decided

[EPIC_3](/epic-3-reviewer-selection-integration/EPIC_3.md),
[SPECS § Architecture](../../../planning/SPECS.md) and
[specs 05](../../../planning/specs/05-tier-assignment-schema.md) all name the same four
providers in the same sentence, and all three give the same reason:

> Dynamic providers (`openrouter`, `vercel-ai-gateway`, `nvidia`, `github-copilot`) are
> refreshed **before** resolution — they hold no models until then, so skipping it makes a
> correctly configured tier fall back silently.

[Issue 04](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md)
carries that into an acceptance criterion — *"a dynamic provider's correctly configured
tier resolves"* — and repeats the four-provider list verbatim in its Scope.

## Actual

The refresh ordering is implemented and verified by mutation, and it works for
`github-copilot` exactly as for the other three. The tier still resolves to **no reviewer**,
one step later: the family gate classifies every model that provider serves as
`family.Unknown`, and an unknown family is excluded by every rule.

Measured against kern-link v0.1.1's embedded catalog, per provider, this run:

| provider | models | classify `Unknown` |
| --- | --- | --- |
| `nvidia` | 20 | 0 |
| `vercel-ai-gateway` | 187 | 0 |
| `openrouter` | 258 | 4 |
| `github-copilot` | **25** | **25** |

So three of the four dynamic providers can serve a tier, and the fourth cannot serve one
under any configuration a human could write.

## Because

`github-copilot` is a reseller, so rule 2 applies and reads the id's vendor segment — and
none of its 25 ids has one. All 25, in full:

```
claude-fable-5, claude-haiku-4.5, claude-opus-4.5, claude-opus-4.6, claude-opus-4.7,
claude-opus-4.8, claude-sonnet-4, claude-sonnet-4.5, claude-sonnet-4.6, claude-sonnet-5,
gemini-2.5-pro, gemini-3-flash-preview, gemini-3.1-pro-preview, gemini-3.5-flash,
gpt-4.1, gpt-5-mini, gpt-5.2, gpt-5.2-codex, gpt-5.3-codex, gpt-5.4, gpt-5.4-mini,
gpt-5.4-nano, gpt-5.5, kimi-k2.7-code, mai-code-1-flash-picker
```

This is the gap [issue 02](/epic-3-reviewer-selection-integration/issues/02-model-family-classifier.md)
measured and deliberately left open — 160 of 1042 embedded models are `Unknown` for the
same reason, across eight providers — and judged to be a specs decision rather than a bug.
Issue 04 is where the consequence becomes visible, because it is the first issue in which a
tier is resolved end to end.

**Ten of those 25 ids are Anthropic's.** That is the half worth being careful about: this
provider is not a case where fail-closed costs nothing but a few good models. A careless fix
— treat `github-copilot` as a vendor provider, or take the id at face value — would admit
`claude-opus-4.8` as a reviewer of Claude's own work, which is the one failure this product
cannot detect after the fact. The correct fix has to classify `claude-` as Anthropic's, not
route around the classifier.

The same measurement makes `openrouter`'s four `Unknown` ids look right rather than
regrettable: `auto`, `openrouter/auto`, `openrouter/free` and `openrouter/fusion` are
routing meta-models whose actual vendor is decided per request by OpenRouter. There is no
family that is true of them, and admitting one would mean a tier that reviews through Claude
on some requests and not others. Refusing them is the rule working.

## Disposition

**Reported, not fixed, and made legible.** This PR does not widen `internal/family`'s data
tables; it makes the refusal say which rule produced it, so the user whose configured tier
resolves to nothing is told why rather than left with a silent native fallback:

- `reviewer.ReasonFamilyUnknown` is a distinct reason from `ReasonFamilyExcluded` and from
  `ReasonNotInCatalog`, precisely because the fix differs — an excluded family was excluded
  on purpose, an unknown one is a gap in the classifier's data.
- The message names the model and the rule: *"the vendor family of github-copilot/gpt-5.5
  could not be determined, and an unknown family is never allowed to review"*.
- `NoReviewerError` carries the reason as a field, so the `model` stderr line
  ([issue 05](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md))
  and the `tiers` command
  ([issue 06](/epic-3-reviewer-selection-integration/issues/06-tiers-command.md)) switch on
  it rather than parse a string.

The fix itself is the third table [drift record 02](/epic-3-reviewer-selection-integration/drift/02-azure-openai-responses-is-a-vendor-provider.md)
already proposed and deferred — bare model-name prefixes, `claude-` → anthropic,
`gpt-` → openai, `gemini-` → google, … — which is a real decision about substring-shaped
matching and belongs in [specs 04](../../../planning/specs/04-model-family-classification.md),
not in this PR. Two records now name it; it should be decided rather than deferred a third
time.

## Alternatives tried

- **Add the bare-name table here.** Rejected on scope: it changes the classifier's rule set,
  which issue 04's Scope does not name and issue 02's Out-of-scope explicitly deferred. It
  would also have to be validated against all 1042 embedded models, which is issue 02's
  catalog-wide test, not this issue's.
- **Let an unknown family through when the caller's exclusion list is empty.** Rejected:
  that is precisely the fail-open direction [specs 04](../../../planning/specs/04-model-family-classification.md)
  refuses, and the ten `claude-*` ids above show what it would admit on this very provider.
- **Drop `github-copilot` from the epic's dynamic-provider list.** Rejected as papering over
  it: the refresh sentence is correct and worth keeping — `github-copilot` really does hold
  no models until refreshed, which this PR's tests exercise. It is the *next* step that
  refuses it, and the list is not the place to say so.

## Revisit when

specs 04 is next touched, or when a user assigns a tier to `github-copilot` and reports it —
whichever comes first. Adding the bare-name table resolves this record and
[drift record 02](/epic-3-reviewer-selection-integration/drift/02-azure-openai-responses-is-a-vendor-provider.md)'s
follow-up together; nothing in `internal/reviewer` changes when it lands, because the gate
asks `family` and takes its answer.

## Evidence

- Measured this run against kern-link v0.1.1's embedded catalog through
  `family.Classify`: the per-provider table and the 25 ids above.
- `TestResolveTier_UnknownFamilyTier_SaysWhichRuleRefusedIt`
  (`internal/reviewer/chain_test.go`) pins the behaviour end to end, on a registry that
  reproduces the provider's shape — empty until refreshed, then serving a bare id. Its
  assertion that the reason is `family-unknown` rather than `not-in-catalog` is what proves
  the refresh ran and the model was found before the gate refused it.
- `TestResolve_UnknownFamily_IsNoReviewer` (`internal/reviewer/resolve_test.go`) covers the
  same rule at the pre-flight level, for a bare reseller id and for a provider in neither
  table.
