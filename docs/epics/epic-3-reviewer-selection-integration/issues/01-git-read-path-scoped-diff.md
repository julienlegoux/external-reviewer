---
type: Issue
title: "Make a path-scoped diff reachable through git_read"
description: "Give git_read a paths parameter so a reviewer can scope a diff to a subtree, closing the gap that inflated Epic 2's turn counts before the caps are sized from them."
tags: [epic-3]
timestamp: 2026-08-13T09:45:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/60
epic: 3
issue: 01
slug: git-read-path-scoped-diff
size: M
status: done
gh_issue: 60
gh_pr: 81
depends_on: []
---

# Make a path-scoped diff reachable through git_read

## Summary

Epic 2's hand runs measured a real gap: **a path-scoped `diff` is unreachable through
`git_read`**, so the only diff a reviewer can obtain is the whole-repository one — which
is exactly the read that hits the 2,000-line cap
([MEASUREMENTS § One defect this measurement exposed](/epic-2-read-only-agentic-loop/MEASUREMENTS.md)).
Run C spent five of its twelve turns failing to scope a diff, then reconstructed the diff
file by file through fourteen `show` calls.

It lands first in this epic because [issue 09](/epic-3-reviewer-selection-integration/issues/09-loop-caps-from-measurements.md)
sizes the loop caps from those turn counts, and this is the single largest distortion in
them: MEASUREMENTS says the same review is *"plausibly 6–8"* turns with the gap closed
rather than 12. Sizing a cap against the tool as it should behave beats sizing it against
a tool that wastes turns.

Two things combine to close the door today (`internal/tools/git_read.go`):

- `allowedArgument` refuses a literal `--`, because `gitReadArgv` appends the granted
  subtrees as pathspecs itself. So the model cannot write `diff <range> -- <path>`.
- Without `--`, the natural fallback is `diff <range> <path>` — but git reads everything
  before a `--` as a revision, so the appended `-- <subtrees>` turns the path into a bad
  revision: `git diff --unified=60 a4e6db9^...develop internal/confine -- .` →
  `fatal: bad revision 'internal/confine'`.

This is **not drift**: nothing in SPECS or [specs 10](../../../planning/specs/10-repository-confinement.md)
ever decided a path-scoped diff must be reachable, and the pathspec appending is doing
exactly what that decision asked for. It is the gap the decision left.

## Scope

- `git_read` gains a **`paths`** wire parameter: an array of repo-relative,
  `/`-separated strings, declared in `gitReadParameters` beside the existing `command`
  and `args`.
- Every entry is resolved through `Scope.Resolve` **before the command is built**, under
  the same three rules and the same refusal wording as the `<rev>:<path>` object form
  (`objectRefusal`): outside the allow-list, denied by the sensitive-file floor, or not a
  repo-relative wire path at all.
- When `paths` is present and every entry resolves, those entries — not
  `scope.Allowed()` — are what follows the single `--` this tool owns. Because each one
  has already been proved inside a granted subtree, the narrower pathspec list is a
  narrowing of the grant, never a widening of it.
- When `paths` is absent, behaviour is exactly what it is today: `scope.Allowed()`
  appended after `--`, for all four subcommands.
- `paths` alongside a `<rev>:<path>` object argument is refused, for the reason that
  already refuses mixing an object with a plain revision — git takes pathspecs *or* an
  object, never both.
- `gitReadDescription` is updated to teach the parameter, since the tool's description
  lives beside its implementation and a description that drifts tells the reviewer it may
  pass an argument that does not exist
  ([CONVENTIONS § Documentation & comments](../../../planning/CONVENTIONS.md)).
- SPECS' `git_read` row in the four-tool table gains the parameter, and
  [specs 09](../../../planning/specs/09-read-only-tool-contract.md) is amended in the
  same PR — the wire contract is the one thing here a later reader will check against the
  documents.

## Out of scope

- Widening `gitReadCommands` beyond `log`, `diff`, `show`, `status`. That is a SPECS
  change, not a PR decision.
- Relaxing the literal `--` refusal or the `--output` refusal. `paths` exists precisely so
  neither has to move.
- Changing `MaxGitReadLines`, or any other tool's caps.
- Re-running Epic 2's measurements or editing MEASUREMENTS' recorded figures. The numbers
  stand as what was observed; [issue 09](/epic-3-reviewer-selection-integration/issues/09-loop-caps-from-measurements.md)
  reads them knowing this fix landed, and [issue 11](/epic-3-reviewer-selection-integration/issues/11-success-criteria-verification-run.md)
  produces fresh ones.

## Acceptance criteria / Definition of done

Strict red-green, black-box in `package tools_test`, against a throwaway `git init`
fixture skipped when `git` is absent
([CONVENTIONS § Testing](../../../planning/CONVENTIONS.md)).

- [ ] Written failing first: in a repository granted `--allow .`, a call
      `{"command":"diff","args":["--unified=0","<rev-range>"],"paths":["internal/confine"]}`
      returns only that subtree's diff — asserted by a path from another subtree being
      absent from the output while a path inside it is present.
- [ ] The same call with `paths` naming a subtree **outside** the granted set is refused
      with the `ErrOutsideAllowList` wording, and git is never run (assert on the refusal
      text naming the granted subtrees, not on git's own `fatal:`).
- [ ] `paths: [".env"]` inside a granted subtree is refused naming the sensitive-file
      floor, in the same words `objectRefusal` already uses.
- [ ] `paths: ["../outside"]`, `paths: ["/etc/passwd"]` and `paths: ["C:\\Windows\\win.ini"]`
      are each refused on both matrix OSes.
- [ ] `paths` together with a `<rev>:<path>` argument is refused, and the message says why
      (git takes pathspecs or an object, not both).
- [ ] Every existing `git_read` test passes unchanged with `paths` absent — in particular
      `TestGitRead_ScopesStatusToTheGrantedSubtrees` and
      `TestGitRead_RefusesABlobPairOnTheFloor`.
- [ ] Verified by mutation: deleting the `Scope.Resolve` call over `paths` makes the
      out-of-allow-list test fail with the ungranted subtree's content in the output.
- [ ] `go test -race -ldflags=-s ./... -count=1` is green, and `golangci-lint` is clean.
- [ ] SPECS' tool table row and [specs 09](../../../planning/specs/09-read-only-tool-contract.md)
      name the `paths` parameter; no drift record is written (nothing decided is being
      departed from).

## Relevant files / areas

- `internal/tools/git_read.go` — `gitReadParameters`, `gitReadDescription`,
  `gitReadArgv`, `allowedArgument`, `objectPath`, `objectRefusal`
- `internal/tools/git_read_test.go`
- `internal/confine/allow.go` — `Scope.Resolve`, `Scope.Allowed` (read, not changed)
- `docs/planning/SPECS.md` § Reading the repository (the four-tool table),
  `docs/planning/specs/09-read-only-tool-contract.md`

Governing decisions:
[specs 09 — Read-only tool contract](../../../planning/specs/09-read-only-tool-contract.md),
[specs 10 — Repository confinement](../../../planning/specs/10-repository-confinement.md),
[DRIFT § Epic 2 entry 06](../../../planning/DRIFT.md) (why `--` is owned by the tool and
why every object argument is validated per subcommand).

## Dependencies

- Blocked by: None.
- Blocks: [09 — loop caps](/epic-3-reviewer-selection-integration/issues/09-loop-caps-from-measurements.md),
  which sizes turn and wall-clock ceilings against turn counts this gap inflates, and
  [11 — success-criteria verification run](/epic-3-reviewer-selection-integration/issues/11-success-criteria-verification-run.md),
  which measures whether closing this gap moved the turn count the way MEASUREMENTS
  predicted.

## PR size note

The `paths` wire parameter and its `Scope.Resolve` wiring in `gitReadArgv` /
`allowedArgument` is small; the bulk is the refusal matrix — six of the nine acceptance
criteria are refusal cases, three of them run on both matrix OSes. If it grows, the
OS-matrix path-traversal tests (`../outside`, `/etc/passwd`, `C:\Windows\win.ini`) are the
natural first cut, since they exercise `Scope.Resolve` rather than anything new in
`git_read` itself.
