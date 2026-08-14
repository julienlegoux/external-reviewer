---
type: Drift record
title: "The three gpt-5.6 models are defined in this repository, alongside kern-link's embedded catalog rather than in it"
description: "SPECS' dependency table and CONVENTIONS' pinning policy make kern-link's embedded catalog the source of model definitions; gpt-5.6-sol, -terra and -luna exist and the release carrying them does not, so they are registered locally onto the openai-codex provider through ai.MutableModels.SetProvider."
tags: [epic-3, drift]
timestamp: 2026-08-13T18:00:00Z
epic: 3
issue: null
---

# The three gpt-5.6 models are defined in this repository, alongside kern-link's embedded catalog rather than in it

## Decided

[SPECS § Stack](../../../planning/SPECS.md) fixes one dependency for everything to do with
models — `github.com/julienlegoux/kern-link`, at a tagged version — and
[specs 03 — kern-link dependency policy](../../../planning/specs/03-kern-link-dependency-policy.md)
makes its **embedded catalog** the source of model definitions: what a model costs, how
large its context window is and which wire protocol it speaks are facts this binary reads,
never facts it states. [CONVENTIONS § Dependencies](../../../planning/CONVENTIONS.md) pins
that dependency and treats a version bump as an ordinary PR, which is the intended way new
models arrive: they arrive with a release.

A second standard is departed from in the same change, smaller and worth naming.
[Issue 05](../issues/05-review-tier-flags-and-exit-codes.md) carries a grep criterion —
*the binary names no provider outside the family classifier's data table* — enforced by
`TestBinary_NamesNoProviderOutsideTheFamilyTables`, which until now allowed the literal
`openai-codex` in `internal/family/vendors.go` and nowhere else.

## Actual

`internal/reviewer/catalog.go` holds three `*ai.Model` literals — `gpt-5.6-sol`,
`gpt-5.6-terra`, `gpt-5.6-luna`, all on provider `openai-codex` — and
`RegisterLocalModels` adds them to the registry `DefaultModels` builds. The pin stays at
v0.1.1: `go.mod` is untouched.

The registration uses the one mechanism kern-link v0.1.1 offers for changing a registry's
contents, `ai.MutableModels.SetProvider`, which upserts a provider by id. A provider's
model list is not writable, so the built-in `openai-codex` provider is replaced *in place*
by a decorator embedding it: every method — id, name, base URL, headers, auth, refresh,
streaming — is the real provider's, and only `GetModels` differs, appending the local
models to whatever the provider returns. Delegating rather than rebuilding is what keeps
`oauth.CodexOAuth` and the `codex` wire adapter kern-link wired, and what would keep the
additions in place across a refresh on a dynamic provider.

Two properties make this an addition rather than a substitution, and both are tested:

- every model the embedded catalog already serves stays reachable, and the provider count
  is unchanged — the decorator replaces one entry, it does not add one;
- a local definition whose id the catalog already carries is **dropped**: the catalog's own
  definition is the one a lookup finds, always.

`TestBinary_NamesNoProviderOutsideTheFamilyTables` now allows the literal in
`internal/reviewer/catalog.go` as well. The criterion it enforces is unchanged in
substance — no resolution path, flag default or fallback names a provider, and the binary's
*mechanism* still knows no vendor — but the allow-list is now two data tables instead of
one.

## Because

The models exist; the release that carries them does not. kern-link v0.1.1's
`openai-codex` catalog holds `gpt-5.3-codex-spark`, `gpt-5.4`, `gpt-5.4-mini` and
`gpt-5.5` and nothing later, and v0.1.1 is still the newest tag — the same fact
[Epic 1's drift record 04](../../epic-1-walking-skeleton/drift/04-kern-link-version-pin.md)
records for v0.2.0, which does not exist either.

Without a local definition there is nothing to bump *to*: a tier naming
`openai-codex/gpt-5.6-sol` resolves through `Resolver.Resolve` to
`ReasonNotInCatalog`, which is `ErrNoReviewer`, which is exit 1 — the silent fallback. The
user would have current models on their account and a binary that cannot reach them.

The alternative the pinning policy prefers — bump the dependency — is unavailable, and the
alternatives that would fake it are worse than the departure (below).

Nothing about the family rule is weakened by the addition. `openai-codex` is a rule-1
vendor provider, so all three classify `openai` without the id being read, and the default
`anthropic` exclusion allows them. That is asserted rather than assumed, because the
failure it guards against is the quiet one: a model that classified `Unknown` would be
refused by every exclusion and unusable while looking perfectly well registered — the
fail-closed cost issue 02's design deliberately pays on shapes it does not recognise.

## Alternatives tried

- **Bump to a release carrying the models** — nothing to bump to; v0.1.1 is the newest tag
  and the drift is already on record from Epic 1.
- **A `replace` onto a local kern-link checkout with the catalog edited** — rejected
  outright by [CONVENTIONS § Dependencies](../../../planning/CONVENTIONS.md): a committed
  `replace` breaks `go install` for anyone whose disk does not match the author's.
- **Rebuild the `openai-codex` provider with `ai.CreateProvider`** and the full model list.
  Rejected: it means restating `oauth.CodexOAuth`, the `codex` stream functions and the
  base URL in this repository, so a change to any of them upstream would be silently
  overridden here — a much larger surface to keep in sync than three model literals.
- **Write the definitions into a tier config file** — not possible and not wanted: the tier
  schema names a provider and a model id, it does not carry prices or context windows, and
  a catalog that a file on a user's disk can extend is the mechanism
  [specs 05](../../../planning/specs/05-tier-assignment-schema.md) rules out for the family
  classifier for the same reason.

## Revisit when

**A kern-link release ships these ids — then delete the local registration rather than
keeping both.** `internal/reviewer/catalog.go` and its test go away in the same PR as the
version bump; `DefaultModels` drops the `RegisterLocalModels` call, and
`TestBinary_NamesNoProviderOutsideTheFamilyTables` goes back to a one-file allow-list.

Keeping both is the outcome to avoid. Two definitions of one model that disagree on price
or context window are worse than either alone, because the disagreement is invisible: the
`models` report would print one number, an invoice another, and nothing would fail. The
collision rule in `RegisterLocalModels` — the catalog wins — makes the intervening state
*inert* rather than wrong, so a release that lands before the cleanup PR does not produce a
wrong number; it does not make keeping both an acceptable resting state.

Also revisit if a fourth model is added here: three literals in one file is a stopgap, a
growing list of them is a private catalog, and that is a decision for
[specs 03](../../../planning/specs/03-kern-link-dependency-policy.md) rather than a PR.

## Evidence

- `internal/reviewer/catalog.go`, `internal/reviewer/resolve.go` (`DefaultModels`), and
  `internal/reviewer/catalog_test.go` on branch `register-gpt-5-6-sol`.
- kern-link v0.1.1 `ai/provider.go` — `MutableModels.SetProvider` is the only mutation the
  interface offers; `Provider.GetModels` has no writing counterpart.
- The built binary's own answer:

  ```
  $ external-reviewer models
  model                                      input/M  output/M  context  note
  …
  openai-codex/gpt-5.3-codex-spark           $1.75    $14.00    128000
  openai-codex/gpt-5.4                       $2.50    $15.00    272000
  openai-codex/gpt-5.4-mini                  $0.75    $4.50     272000
  openai-codex/gpt-5.5                       $5.00    $30.00    272000
  openai-codex/gpt-5.6-luna                  $0.20    $1.20     272000
  openai-codex/gpt-5.6-sol                   $5.00    $30.00    272000
  openai-codex/gpt-5.6-terra                 $2.00    $12.00    272000
  ```

- `TestLocalModels_ClassifyIntoTheOpenAIFamilyAndAreAllowed` — each of the three
  classifies `openai` and is allowed by `family.DefaultExclusion()`.
