---
type: Drift record
title: "kern-link is pinned at v0.1.1, not the specified v0.2.0"
description: "SPECS and issue 04 pin github.com/julienlegoux/kern-link at v0.2.0; no such tag exists, and the module resolves at v0.1.1 — which carries every API the resolution sequence needs, under the names SPECS uses."
tags: [epic-1, drift]
timestamp: 2026-08-09T11:45:00Z
epic: 1
issue: 04
---

# kern-link is pinned at v0.1.1, not the specified v0.2.0

## Decided

[SPECS § Stack](../../../planning/SPECS.md) fixes the dependency table at
`github.com/julienlegoux/kern-link` **v0.2.0**, and
[CONVENTIONS § Dependencies](../../../planning/CONVENTIONS.md) freezes the dependency set
at two, with everything the ecosystem allows to be pinned pinned. Issue 04's Scope repeats
the version verbatim: "add `github.com/julienlegoux/kern-link` at the tagged **v0.2.0**".

## Actual

`go.mod` requires `github.com/julienlegoux/kern-link v0.1.1`. No `replace` directive is
committed and `go.work` stays gitignored, so the rest of the dependency policy holds.

## Because

The tag does not exist. Both the module proxy and the repository agree:

```
$ go get github.com/julienlegoux/kern-link@v0.2.0
go: github.com/julienlegoux/kern-link@v0.2.0: invalid version: unknown revision v0.2.0

$ go list -m -versions github.com/julienlegoux/kern-link
github.com/julienlegoux/kern-link v0.1.0 v0.1.1

$ gh api repos/julienlegoux/kern-link/tags --jq '.[].name'
v0.1.1
v0.1.0
```

v0.1.1 is the newest tag, so this is a forward pin to a release that has not been cut
rather than a version that was yanked. The user owns the upstream, which is why the plan
could name a version ahead of the tag.

Nothing else in the plan is disturbed by the substitution. Every symbol issue 04 told this
run to verify against the tag is present in v0.1.1 under the exact name SPECS uses —
`ai.Models.Refresh`, `ai.Provider.CanRefreshModels`, `ai.Models.GetModel`,
`ai.Models.GetAuth`, `*ai.ModelsError` with codes `oauth` and `auth`, `AuthResult.Source`,
`ai.MutableModels`, and `ai/providers/faux`. GetAuth's documented contract matches SPECS
line for line, including "(nil, nil) when the provider is unknown or unconfigured". **There
is no API-name drift to record**: the resolution sequence is written against the real API
and the real API is the one the plan described.

One consequence does follow the version, and it is worth naming because a later issue
would otherwise trip on it: the embedded catalog is the catalog of *this* tag.
[SPECS § Data, configuration & auth](../../../planning/SPECS.md) illustrates the tier file
with `provider = "openai-codex"` / `model = "gpt-5.1-codex"`, and v0.1.1's
`openai-codex` catalog carries `gpt-5.3-codex-spark`, `gpt-5.4`, `gpt-5.4-mini` and
`gpt-5.5` — not `gpt-5.1-codex`. The walking skeleton's hard-coded reviewer is therefore
`openai-codex/gpt-5.5`, still outside the Anthropic family as the issue requires, and a
test asserts it is present in the embedded catalog so the constant cannot rot silently.

## Alternatives tried

- **`@v0.2.0` directly** — fails as above; there is nothing to fetch.
- **A `replace` directive onto a local checkout** — rejected outright by
  [CONVENTIONS § Dependencies](../../../planning/CONVENTIONS.md): a committed `replace`
  breaks `go install` for anyone whose disk does not match the author's, and it would have
  hidden the missing tag from CI, which is the one thing meant to catch it.
- **`@main` / a pseudo-version** — rejected: SPECS requires a *tagged* dependency, and a
  pseudo-version would swap a missing pin for an unpinnable one.

## Revisit when

`kern-link` v0.2.0 (or later) is tagged. At that point `go get
github.com/julienlegoux/kern-link@v0.2.0` is an ordinary version-bump PR — CONVENTIONS
explicitly classes bumping a pinned version as ordinary — and SPECS' dependency table stops
being ahead of reality. Until then, SPECS names a version that cannot be installed, so the
table should say v0.1.1 if the disposition is `accepted`.

## Evidence

- `go.mod`, `go.sum` on branch `issue-7-model-resolution-and-auth`.
- [Issue 04](../issues/04-model-resolution-and-auth.md).
