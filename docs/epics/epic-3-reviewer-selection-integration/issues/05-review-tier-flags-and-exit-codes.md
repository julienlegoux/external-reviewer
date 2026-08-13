---
type: Issue
title: "Select the reviewer from the command line with --tier, --model and --exclude-family"
description: "Wire tier resolution into `review`, retire the hard-coded reviewer, and prove the exit-code contract end to end through Run."
tags: [epic-3]
timestamp: 2026-08-13T00:13:53Z
resource: https://github.com/julienlegoux/external-reviewer/issues/64
epic: 3
issue: 05
slug: review-tier-flags-and-exit-codes
size: M
status: open
gh_issue: 64
depends_on: [4]
---

# Select the reviewer from the command line with --tier, --model and --exclude-family

## Summary

The point at which a skill can ask for *a kind of reviewer* and never name a vendor. This
PR puts [issue 04](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md)'s
chain behind three flags on `review`, deletes the walking skeleton's hard-coded
`openai-codex/gpt-5.5`, and proves the exit-code contract end to end through `Run` — which
is where it actually matters, because the caller is a script reading exit codes, not a
human reading prose.

The command line is a contract with a script that lives in another repository
([specs 02](../../../planning/specs/02-cli-surface-and-argv.md)), so the flag names and
their semantics are a one-way door: [issue 10](/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md)
builds `review-interfaces.md`'s invocation template from exactly what lands here.

## Scope

- `review` gains, all long-form:
  - `--tier light|standard|heavy` — the request vocabulary, reusing the grading
    `implement-epic` already applies to work ([scope 04](../../../planning/scope/04-reviewer-selection-vocabulary.md));
  - `--model provider/id` — bypasses tiers entirely, journey 4's by-hand path;
  - `--exclude-family anthropic[,…]` — comma-separated, **defaulting to `anthropic`**.
- `--tier` and `--model` are mutually exclusive (the grammar spells them as alternatives);
  giving both is a usage error, exit 2.
- With neither given, `--tier standard` is the default. Stated here as the decision this
  PR makes: the caller that omits the flag is asking for an ordinary review, and the
  alternative — a usage error — would make every hand invocation longer for no safety
  gained, since the family rule protects the outcome either way.
- An empty `--exclude-family ""` is a **usage error**, not "exclude nothing": a flag value
  that silently disables the product's central safety property is exactly the shape
  [specs 04](../../../planning/specs/04-model-family-classification.md) refuses to ship.
- `reviewer.DefaultProviderID` and `DefaultModelID` are removed, along with their use in
  `resolveAndReview`. Nothing in the binary names a provider after this PR except the
  family classifier's data table.
- The exit-code contract, exercised through `Run(argv, stdin, stdout, stderr)`
  ([specs 13](../../../planning/specs/13-error-handling-and-failure-classification.md)):
  - never reached → **1**, stdout empty, **no `error:` line** (SPECS' silent-fallback
    case), `done` line `stop=no_reviewer`;
  - reached and broken credential → **2**, one `error:` line, `stop=failed`;
  - malformed invocation → **2**, `stop=usage`.
- The `model` stderr line keeps its shape — `provider/id` verbatim from `kern-link`'s own
  catalog plus `auth=<source>` — and gains nothing that could carry a credential value.
- `usageText` in `internal/cli/usage.go` is updated to list exactly the flags that now
  work, and no others: it currently documents only `--allow` and `--prompt` and says so
  deliberately.

## Out of scope

- The `tiers` and `models` commands — issues [06](/epic-3-reviewer-selection-integration/issues/06-tiers-command.md)
  and [07](/epic-3-reviewer-selection-integration/issues/07-models-command.md) — and
  `version`, which lands with [issue 08](/epic-3-reviewer-selection-integration/issues/08-request-object-and-version.md).
- The JSON request object and `--system`; this PR leaves the prompt channel exactly as it
  is ([issue 08](/epic-3-reviewer-selection-integration/issues/08-request-object-and-version.md)
  changes it, and is sequenced after this one for that reason).
- Loop caps and any cap-related flag — [issue 09](/epic-3-reviewer-selection-integration/issues/09-loop-caps-from-measurements.md).
- Short flags, `--flag=value` after positionals, and any CLI framework
  ([specs 02](../../../planning/specs/02-cli-surface-and-argv.md)).

## Acceptance criteria / Definition of done

Strict red-green, black-box in `package cli_test`, driven through `Run` (or the existing
`RunForTest` seam) against an offline registry, with `EXTERNAL_REVIEWER_CONFIG` pointing at
a `t.TempDir()` fixture.

- [ ] Written failing first: `review --allow . --tier standard <repo>` with a fixture
      config resolves, and the `model` line on stderr names that tier's `provider/id`
      verbatim with `auth=<source>`.
- [ ] With **no config file and no environment override**: exit `1`, stdout empty, stderr
      carries **no** `error:` line, and the `done` line reads `stop=no_reviewer`.
- [ ] `--tier heavy` assigned an Anthropic-family model: exit `1`, `stop=no_reviewer`, and
      stderr never names a substitute model — asserted on the full stderr text, since a
      silent fallback is the failure this epic exists to prevent.
- [ ] `--model <provider>/<id>` bypasses a config fixture that assigns a different model;
      the `model` line shows the flag's value.
- [ ] `--tier standard --model x/y` → exit `2`, `stop=usage`, one `error:` line.
- [ ] `--tier bogus` → exit `2`, `stop=usage`.
- [ ] `--exclude-family ""` → exit `2`, `stop=usage`.
- [ ] `--exclude-family anthropic,google` excludes both families: a Google-family
      assignment that resolves without the flag resolves to `ErrNoReviewer` with it.
- [ ] Omitting both `--tier` and `--model` behaves identically to `--tier standard`.
- [ ] A broken credential (`*ai.ModelsError`, code `oauth` or `auth`) → exit `2`, exactly
      one `error:` line, `stop=failed`.
- [ ] `grep -rn "openai-codex" --include=*.go internal/ main.go` matches only the family
      classifier's data table (and test fixtures) — the mechanism names no provider.
- [ ] `usageText` lists `--tier`, `--model` and `--exclude-family`, and promises no flag
      that is still a usage error.
- [ ] `go test -race -ldflags=-s ./... -count=1` green; `golangci-lint` clean.

## Relevant files / areas

- `internal/cli/review.go` — `runReviewCommand` (the `FlagSet`), `resolveAndReview`
- `internal/cli/usage.go` — `usageText`
- `internal/cli/run.go` — `run` / `runProcess`, where the registry and bounds are threaded
- `internal/reviewer/resolve.go` — `DefaultProviderID`, `DefaultModelID` removed
- `internal/cli/review_test.go`, `internal/cli/run_test.go`, `internal/cli/roundtrip_test.go`

Governing decisions:
[specs 02 — CLI surface & argv grammar](../../../planning/specs/02-cli-surface-and-argv.md),
[specs 13 — Error handling and failure classification](../../../planning/specs/13-error-handling-and-failure-classification.md),
[scope 04 — Reviewer selection vocabulary](../../../planning/scope/04-reviewer-selection-vocabulary.md),
[scope 06 — Family exclusion rule](../../../planning/scope/06-family-exclusion-rule.md).

## Dependencies

- Blocked by: [04 — tier resolution chain](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md).
- Blocks: [06 — tiers command](/epic-3-reviewer-selection-integration/issues/06-tiers-command.md),
  [08 — request object and version](/epic-3-reviewer-selection-integration/issues/08-request-object-and-version.md)
  (both rewrite parts of `runReviewCommand`; sequencing them avoids a conflict neither
  would notice),
  [10 — review-interfaces swap](/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md).

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
