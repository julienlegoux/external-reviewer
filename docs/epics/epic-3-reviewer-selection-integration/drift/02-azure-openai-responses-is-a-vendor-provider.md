---
type: Drift record
title: "azure-openai-responses is classified by rule 1, not by the reseller rule specs 04 lists it under"
description: "specs 04's illustrative reseller list names azure-openai-responses, whose 42 embedded ids carry no vendor segment at all — so the reseller rule cannot classify a single one of them; the classifier puts it with the vendor providers instead, and strips OpenRouter's ~ alias marker."
tags: [epic-3, drift]
timestamp: 2026-08-13T10:30:00Z
epic: 3
issue: 02
---

# azure-openai-responses is classified by rule 1, not by the reseller rule specs 04 lists it under

## Decided

[specs 04 — Model family classification](../../../planning/specs/04-model-family-classification.md)
fixes three rules in order, and gives each an illustrative provider list. Rule 2 reads:

> Provider is a reseller or aggregator (`amazon-bedrock`, `openrouter`, `vercel-ai-gateway`,
> `github-copilot`, **`azure-openai-responses`**, `groq`, `together`, `fireworks`, `nvidia`,
> `cloudflare-*`, …) → the family is parsed from the id's vendor segment: everything before
> the first `.` or `/`, after stripping a leading regional qualifier.

[Issue 02](/epic-3-reviewer-selection-integration/issues/02-model-family-classifier.md)'s
Scope repeats both lists verbatim, so the placement travelled into this PR's instructions.

## Actual

`internal/family/vendors.go` puts `azure-openai-responses` in `vendorProviders`, mapped to
`openai` — rule 1, the id never read — beside `anthropic`, `google` and `openai-codex`.
Membership of that table is documented as a property rather than as corporate ownership:
*a provider is here when every model it can ever serve is one vendor's*.

## Because

The reseller rule cannot classify a single model this provider serves, because none of its
ids carry a vendor segment. All 42 entries in kern-link v0.1.1's
`data/models/azure-openai-responses.json` are bare OpenAI model names:

```
gpt-4, gpt-4-turbo, gpt-4o, gpt-4o-mini, gpt-5, gpt-5-chat-latest, gpt-5-codex,
gpt-5-mini, gpt-5-nano, gpt-5-pro, o1, o1-pro, o3, o3-deep-research, o3-mini,
o3-pro, o4-mini, o4-mini-deep-research, …
```

Applied literally, rule 2 takes everything before the first `.` or `/` and yields `gpt-4`
from `gpt-4.1` and the whole id from `o3` — neither is a vendor, so rule 3 fires and all 42
models are Unknown, which means excluded. The result would be safe (a false exclusion costs
a native review) and wrong in a way that costs the user a whole provider: Azure's OpenAI
service serves OpenAI's models and no one else's, so there is no id it can ever hand back
whose family is in doubt.

Rule 1's own list already contains a provider that is not a vendor by name —
`openai-codex`, OpenAI's Codex endpoint. Read against that, rule 1's membership test is
*serves exactly one vendor*, and `azure-openai-responses` meets it. The placement in rule 2
looks like an artifact of grouping providers by "is it a first-party endpoint" rather than
by "does the id carry a vendor", which is what the rules actually turn on.

The direction of the departure matters and is the safe one: it can never admit a model of a
different family than the one it names, because Azure OpenAI cannot serve Anthropic's
models. The dangerous direction — a reseller wrongly promoted to rule 1 — is not what
happened here.

## Also: OpenRouter's `~` alias marker is stripped with the regional qualifier

Same sentence in specs 04, smaller point. `vendorSegment` strips a leading `~` from the
vendor segment after the regional qualifier, which specs 04 does not mention. kern-link's
embedded `openrouter.json` carries nine floating aliases spelled that way —
`~anthropic/claude-opus-latest`, `~anthropic/claude-sonnet-latest`, `~google/gemini-pro-latest`,
`~openai/gpt-latest` and five more. Without the strip they classify Unknown, which is safe
but wrong: `~anthropic/claude-opus-latest` **is** Anthropic's, and the classifier that is
supposed to be the executable form of "no Claude reviewing Claude" should say so rather than
reach the same refusal by not knowing. The strip only ever resolves an alias to the vendor
already named in it.

## Alternatives tried

- **Follow the placement literally** — `azure-openai-responses` in `resellers`. Measured:
  42 of 42 models Unknown, the provider unusable for any tier, and the catalog-wide test
  records a whole provider as unknown for no reason a reader could reconstruct.
- **A third table of bare model-name prefixes** (`claude-` → anthropic, `gpt-` → openai, …)
  which would classify azure-openai-responses *and* the four other bare-id providers
  (`github-copilot`, `cloudflare-ai-gateway`, `opencode`, `cerebras`). Rejected as a rule
  specs 04 does not have and this issue's Scope does not name — it is a real decision, on
  substring-shaped matching, and it belongs to a decision doc rather than to this PR. Noted
  as a follow-up instead.

## Revisit when

specs 04 is next touched — the smallest amendment is to move `azure-openai-responses` from
rule 2's list to rule 1's, and to state rule 1's membership test as *serves exactly one
vendor's models* so the next provider is placed by the test rather than by the example.
Also revisit if kern-link ever gives this provider a vendor-qualified id, at which point
both rules would agree and the entry could move.

## Evidence

- kern-link v0.1.1 `ai/catalog/data/models/azure-openai-responses.json` — 42 ids, none with
  a `.`-or-`/` vendor segment; `openrouter.json` — nine `~vendor/...` ids.
- `internal/family/vendors.go` (`vendorProviders`, and `vendorSegment`'s `~` trim in
  `internal/family/family.go`).
- `TestClassify_AVendorProviderAnswersWithoutReadingTheId` covers the azure case;
  `TestClassify_AResellerYieldsTheVendorSegmentOfTheId` covers the two `~` cases;
  `TestClassify_EveryEmbeddedModelClassifiesToTheRecordedFamilies` records
  `azure-openai-responses` as classifying `openai` and nothing else.
