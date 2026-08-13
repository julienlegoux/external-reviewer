---
type: Issue
title: "Accept the JSON request object on stdin, add --system and version"
description: "The caller-owned system prompt reaches the binary as {\"system\",\"task\"} on stdin with --system/--prompt as by-hand shorthand, and version reports the build."
tags: [epic-3]
timestamp: 2026-08-13T18:10:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/67
epic: 3
issue: 08
slug: request-object-and-version
size: M
status: in-progress
gh_issue: 67
depends_on: [5]
---

# Accept the JSON request object on stdin, add --system and version

## Summary

**The caller owns the system prompt** ([specs 08](../../../planning/specs/08-system-prompt-and-task-assembly.md)):
the binary supplies none, has no default, and never substitutes one — an adjustment to how
the reviewer is instructed must be a change to a *skill*, not a new build of the binary.
Epic 1 deliberately deferred the mechanism ([epic 1 issue 02](/epic-1-walking-skeleton/issues/02-review-invocation-parsing.md),
Out of scope: *"the structured form and the caller-supplied system prompt arrive with the
epics that need them"*), and this is that epic:
[issue 10](/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md)
cannot write an invocation template without a channel for the prompt, and the
leads-not-findings instruction — the product's one unenforceable assumption — moves into
that prompt.

`version` rides along because it is the last item of SPECS' grammar still unimplemented,
it touches the same `usageText` and the same subcommand switch, and it is three lines of
`runtime/debug.ReadBuildInfo()` — its own PR would be a PR nobody wants to review.

Note this **changes an existing behaviour**: today bare stdin is read as raw task text.
After this PR stdin is a JSON object. Nothing recorded depends on the old shape — Epic 2's
measured runs all used `--prompt` — and the only caller that will exist is written in
[issue 10](/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md).

## Scope

- Stdin for `review` is decoded as `{"system": "…", "task": "…"}` with
  `DisallowUnknownFields`. A mistyped key is an error rather than a silently missing system
  prompt.
- Either field absent, empty, or whitespace-only → exit `2`. **No default system prompt is
  ever substituted**: a silent fallback would be the binary owning the prompt again through
  the back door.
- `--system <text>` and `--prompt <text>` remain the by-hand shorthand, mutually exclusive
  with stdin: giving either flag means stdin is not read at all, and giving one without the
  other is a usage error.
- The system prompt reaches the model as the request's system message, and the task as the
  first user message, verbatim — nothing is injected alongside it (no file listing, no
  README, no git log). The repository's path is stated with the task, as it is today.
- The 1 MiB stdin bound (`maxPromptBytes`) and its cancellable read stay exactly as they
  are; only the decoding of what was read changes.
- `external-reviewer version` prints the version and revision from
  `runtime/debug.ReadBuildInfo()`, exit `0`, so a report traces to a build without release
  machinery ([specs 17](../../../planning/specs/17-distribution-and-ci.md)). A binary built
  outside a module context prints what `ReadBuildInfo` gives rather than failing.
- `usageText` documents the request object, both flags and `version`.

## Out of scope

- **A task-type guard, of any kind.** A caller-supplied system prompt means the mechanism
  accepts tasks other than reviewing; [specs 08](../../../planning/specs/08-system-prompt-and-task-assembly.md)
  states in terms that this is deliberate, deliberately unadvertised, and that nothing
  should be added that validates or classifies the prompt.
- `--system-file <path>` — considered and rejected: a prompt generated in the calling turn
  has no file.
- Tool declarations as request input. Tool names, descriptions and schemas stay in Go
  beside the implementations and are not overridable.
- Any new field on the request object beyond `system` and `task`. The object is extensible
  by design, but nothing needs a third field yet.
- Writing the leads-not-findings prompt itself — that text is
  [issue 10](/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md)'s,
  in the skills repository.

## Acceptance criteria / Definition of done

Strict red-green, black-box in `package cli_test` through `Run`, with the model behind
`kern-link`'s `faux` provider so the request that reached it can be asserted on.

- [ ] Written failing first: `{"system":"S","task":"T"}` on stdin reaches the model with
      `S` as the system message and `T` as the first user message — asserted against the
      recorded request, not against stderr.
- [ ] `{"systm":"S","task":"T"}` → exit `2`, one `error:` line naming the unknown field,
      `stop=usage`.
- [ ] `{"system":"","task":"T"}` and `{"system":"S","task":""}` and `{"system":"S"}` each
      → exit `2`; no request is ever sent (assert the registry recorded no call).
- [ ] Raw non-JSON text on stdin → exit `2` with a message saying a JSON request object was
      expected.
- [ ] `--system "S" --prompt "T"` is equivalent to the stdin form, and stdin is not read
      (assert with a stdin reader that fails the test if read).
- [ ] `--prompt "T"` without `--system`, and `--system "S"` without `--prompt`, are each
      usage errors.
- [ ] Stdin over `maxPromptBytes` still returns the existing `errPromptTooLarge` path, exit
      `2` — behaviour unchanged.
- [ ] Leading and trailing whitespace inside a non-empty `task` still reaches the model
      verbatim (the existing `TestRun_Review_PromptWhitespaceReachesModelVerbatim`
      guarantee, carried over to the new channel).
- [ ] `version` exits `0` and prints a line containing the module path and the revision or
      version `ReadBuildInfo` reports.
- [ ] `grep -rn "you are an? \(external \)\?reviewer" --include=*.go .` (case-insensitive)
      matches nothing outside `_test.go` — the binary carries no system prompt.
- [ ] `go test -race -ldflags=-s ./... -count=1` green; `golangci-lint` clean.

## Relevant files / areas

- `internal/cli/review.go` — `runReviewCommand`, `readPromptFromStdin`, `maxPromptBytes`,
  `errPromptTooLarge`, `reviewRequest`
- `internal/cli/run.go` — the subcommand switch (for `version`)
- `internal/cli/usage.go` — `usageText`
- `internal/reviewer/run.go` — `NewConversation`, which currently takes only the task
- `internal/cli/stdin_test.go`, `internal/cli/review_test.go`, `internal/cli/roundtrip_test.go`

Governing decisions:
[specs 08 — System prompt & task assembly](../../../planning/specs/08-system-prompt-and-task-assembly.md),
[specs 02 — CLI surface & argv grammar](../../../planning/specs/02-cli-surface-and-argv.md),
[specs 17 — Distribution and CI](../../../planning/specs/17-distribution-and-ci.md),
[scope 18 — Risks and assumptions](../../../planning/scope/18-risks-and-assumptions.md) (risk 6:
what the binary cannot enforce about its caller).

## Dependencies

- Blocked by: [05 — CLI tier flags](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md).
  Not a functional dependency — a sequencing one: both rewrite `runReviewCommand`'s flag
  and prompt handling, and landing them in parallel produces a conflict neither PR's tests
  would catch.
- Blocks: [10 — review-interfaces swap](/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md),
  which has no channel for its system prompt until this lands.

## PR size note

Two nearly unrelated pieces riding together, as the Summary already says: the stdin JSON
request-object contract (`DisallowUnknownFields`, the empty/whitespace checks, the
`--system`/`--prompt` shorthand) is the bulk; `version` is three lines of
`runtime/debug.ReadBuildInfo()` and the first thing to peel off if the PR runs long, since
it shares no code with the request-object change.
