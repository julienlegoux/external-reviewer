---
type: Issue
title: "Kill the four surviving mutants on the request side of the round trip"
description: "Add internal/reviewer/run_test.go and the cache-token assertion in cli, so the prompt reaching the model, FinalText's concatenation and filtering, the messages accumulator and the cache half of in= are all observed."
tags: [epic-0]
timestamp: 2026-08-11T03:33:08Z
epic: 0
issue: 06
slug: request-side-mutants
size: M
status: open
gh_issue: 28
resource: https://github.com/julienlegoux/external-reviewer/issues/28
depends_on: [5]
---

# Kill the four surviving mutants on the request side of the round trip

## Summary

Eighteen of report 2's twenty-four mutations were killed. The four that survived are all
on the request side of the round trip, and three of them live in a file that does not
exist: `internal/reviewer` has no `run_test.go`. Coverage does not show it —
with `-coverpkg=./...` the package reads `Next 93.8 % / FinalText 85.7 % / toolCalls
80.0 %` — because every test scripts `faux` with a fixed response and asserts only that the
response came back.

What that costs, concretely: mutating `ai.UserText(task)` to the empty string leaves all
three packages green, so a binary that silently discarded the user's prompt and reviewed
nothing would ship. So would one whose report is only its first text block. Epic 2's tool
loop is built directly on both properties, and on the `messages` accumulator whose
`append` can be deleted with the suite green.

## Scope

- A new `internal/reviewer/run_test.go` (`package reviewer_test`) on issue 05's harness.
- **The prompt reaches the model.** A `faux.StepFunc` receives the `ai.Context` — it is
  already used that way at `internal/cli/roundtrip_test.go:118` — so one assertion inside a
  scripted step observes the user message's text and compares it to the task verbatim.
- **`FinalText` concatenates.** A scripted response with two text blocks yields both, in
  order, with nothing inserted between them.
- **`FinalText` filters.** A scripted response carrying a thinking block and an
  `ai.ToolCall` alongside its text puts neither on stdout nor in the returned report —
  the doc comment's promise at `internal/reviewer/run.go:100-102`, currently unverified.
- **The `messages` accumulator.** After a turn, the conversation holds the user message and
  the assistant message in order. It matters only once Epic 2's loop lands on that
  accumulator, which is the reason to pin it now.
- **The cache-token half of `in=`.** The accumulation lives in `internal/cli`
  (`review.go:97`), so its test does too: a scripted turn reporting non-zero `CacheRead`
  and `CacheWrite` makes the `done` line's `in=` include them. The arithmetic is already
  correct — kern-link's `Input` excludes cache tokens and prices them as disjoint terms —
  what is missing is the assertion.

## Out of scope

- Cost, the stream timeout and cancellation precedence in `Next` —
  [issue 11](/epic-0-skeleton-hardening/issues/11-turn-hardening.md), which adds its tests
  to the file this PR creates.
- The multi-turn loop itself and anything that dispatches a tool call — Epic 2. This PR
  scripts a tool call only to prove `FinalText` drops it.
- Automating mutation testing. Mutations are applied by hand, one at a time, and recorded.
- `reviewer.DefaultModels` and `reviewModels`' real-registry branch — unreachable offline
  by construction, as report 2 records.

## Acceptance criteria / Definition of done

- [ ] Written test-first, black-box (`package reviewer_test` / `package cli_test`),
      table-driven where the cases share a shape.
- [ ] Each of these mutations turns the suite **red**, with the command and the failing
      test name recorded in the PR body:
      - `ai.UserText(task)` → `ai.UserText("")` at `internal/reviewer/run.go:46`
      - `break` inserted after `out.WriteString(text.Text)` at `internal/reviewer/run.go:110`
      - deleting `+ turn.Usage.CacheRead + turn.Usage.CacheWrite` at `internal/cli/review.go:97`
      - deleting `c.messages = append(c.messages, message)` at `internal/reviewer/run.go:81`
- [ ] A scripted response containing a thinking block and an `ai.ToolCall` alongside two
      text blocks produces exactly the two text blocks concatenated — asserted both on
      `FinalText`'s return value and on what reaches stdout through `Run`.
- [ ] The prompt assertion is made against what the provider actually received, from inside
      the scripted step — not against the value passed to `NewConversation`.
- [ ] The cache-token test scripts a turn with `CacheRead` and `CacheWrite` non-zero and
      asserts the parsed `in=` on the `done` line equals `Input + CacheRead + CacheWrite`.
- [ ] `internal/reviewer/run_test.go` exists and is `package reviewer_test`.
- [ ] `go test ./... -race` and `golangci-lint run` pass; CI green on ubuntu-latest and
      windows-latest.

## Relevant files / areas

- `internal/reviewer/run_test.go` (new)
- `internal/reviewer/run.go:41-50` (`NewConversation`, the `ai.UserText(task)` seeding),
  `:81` (the accumulator), `:100-114` (`FinalText`), `:116-124` (`toolCalls`)
- `internal/cli/review.go:97` (the cache-token sum), and the `cli` test that reads the
  `done` line
- `internal/cli/roundtrip_test.go:118` — the existing `StepFunc` that already receives the
  `ai.Context`, the pattern the prompt assertion copies
- `internal/fauxtest/` — issue 05's harness

Governing decisions: [CONVENTIONS § Testing](../../../planning/CONVENTIONS.md),
[SPECS § Testing](../../../planning/SPECS.md). Source findings:
[report 2](../../../REPORT_2.md) § "The task prompt is never asserted to reach the model"
(P0), § "`FinalText` is only ever exercised with a single text block" (P0), §
"`internal/reviewer/run.go` has no test file of its own", § "The cache-token half of the
`in=` figure is unasserted".

## Dependencies

- Blocked by: [05 — Shared `faux` harness](/epic-0-skeleton-hardening/issues/05-shared-faux-harness.md)
  — the new `internal/reviewer` test file builds on the importable harness rather than on
  `resolve_test.go`'s soon-to-be-deleted copy.
- Blocks: [11 — Turn hardening](/epic-0-skeleton-hardening/issues/11-turn-hardening.md),
  whose three tests land in the file this PR creates.

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
Sized **M**: ~300 lines, nearly all of it test, plus the scripted multi-block and
thinking/tool-call fixtures.
