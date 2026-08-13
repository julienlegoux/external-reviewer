---
type: Issue
title: "Amend EPIC_3 with the scope its issues will ship"
description: "Name the git_read paths parameter, the JSON request object, --system and version in EPIC_3's Scope and Acceptance criteria, and correct SPECS' count of the non-model stop reasons."
tags: [epic-0]
timestamp: 2026-08-13T05:50:38Z
resource: https://github.com/julienlegoux/external-reviewer/issues/71
epic: 0
issue: 01
slug: epic-3-scope-amendment
size: S
status: open
gh_issue: 71
depends_on: []
---

# Amend EPIC_3 with the scope its issues will ship

## Summary

`EPIC_3.md` is the file `close-epic` and `review-implementation` grade the merged diff
against, and two of the eleven issues cut from it ship work the epic never names. Issue 01
adds a **`paths`** wire parameter to `git_read`
(`issues/01-git-read-path-scoped-diff.md:49`); issue 08 replaces the stdin contract with a
JSON `{"system","task"}` request object and adds `--system` and a `version` subcommand
(`issues/08-request-object-and-version.md`). Neither word appears anywhere in `EPIC_3.md`.

Both are legitimate and already traceable — `SPECS.md:185-196`'s `review` grammar block
carries `[--system <text> --prompt <text>]`, the stdin request object and `version`, and
`docs/epics/log.md:34-46` records why each was pulled into the cut. What is missing is the
amendment to the epic itself. **The issues are not weakened by this PR; the epic is the
file that is wrong.**

Riding along, because issues 05 and 09 both rest on it: `SPECS.md:275` introduces the
non-model stop reasons as *"one of five fixed CLI words"* and then lists six — `usage`,
`help`, `no_reviewer`, `interrupted`, `failed`, `bounds`. One word.

## Scope

- `EPIC_3.md` § **Scope** gains what issues 01 and 08 will ship, in the register of the
  bullets already there (a capability and why it belongs to this epic, not a restatement of
  the issue):
  - The `paths` parameter on `git_read` — a path-scoped diff made reachable, which is the
    single largest distortion in the turn counts this epic sizes its caps from
    ([MEASUREMENTS](/epic-2-read-only-agentic-loop/MEASUREMENTS.md)).
  - The JSON request object on stdin, `--system`, and `version` — the request channel the
    calling skill actually uses, with the caller owning the system prompt
    ([specs 08](../../../planning/specs/08-system-prompt-and-task-assembly.md)), plus the
    build identity a report traces to
    ([specs 17](../../../planning/specs/17-distribution-and-ci.md)).
- `EPIC_3.md` § **Acceptance criteria**, under *"Plus the mechanics they rest on"*, gains a
  criterion per addition, in the assertable form the section already uses — a path-scoped
  `diff` through `git_read` reaches one subtree and is refused outside the granted set; a
  `review` invocation whose stdin carries `{"system","task"}` reaches the model with the
  caller's prompt verbatim and no substituted default; `version` prints build information
  and exits `0`.
- `SPECS.md:275`: *"one of five fixed CLI words"* → **six**. The list under it is already
  correct and is not touched.

## Out of scope

- **Editing the issues.** Issues 01 and 08 are faithful transcriptions of decisions that
  were made; nothing in their Scope, Out of scope or Acceptance criteria changes here.
- **Widening Epic 3.** This PR names work already cut into its issues. Anything not in one
  of the eleven issue files does not enter the epic through this door.
- The rest of `EPIC_3.md` — Goal, Out of scope, Dependencies, Context, Notes. Note's
  README claim is [issue 04](/epic-0-epic-3-plan-repair/issues/04-readme-validation-boundary.md);
  the `create-issues`-deferred clause is
  [issue 03](/epic-0-epic-3-plan-repair/issues/03-epic-3-sizes-and-surfaces.md).
- `SPECS.md` beyond the single miscount. The stop-reason vocabulary itself was settled by
  the previous remediation lane (#37, merged) and is not reopened.

## Acceptance criteria / Definition of done

Every criterion below is checkable with `grep` against the working tree, from the
repository root.

- [ ] `grep -c 'paths' docs/epics/epic-3-reviewer-selection-integration/EPIC_3.md` is
      non-zero, and the match is inside `## Scope`.
- [ ] The same holds for the request object, `--system` and `version`: each of the four
      additions appears in `## Scope` **and** in `## Acceptance criteria`, so a reader who
      grades the merged diff against either section finds them.
- [ ] `sed -n '/^## Acceptance criteria/,/^## Dependencies/p'` over `EPIC_3.md` shows one
      new criterion per addition, each stating an observable outcome (a command and what it
      does) rather than naming a feature.
- [ ] `grep -n 'five fixed CLI words' docs/planning/SPECS.md` returns nothing;
      `grep -n 'six fixed CLI words' docs/planning/SPECS.md` returns line 275, and the six
      words listed after it are unchanged.
- [ ] `EPIC_3.md`'s `timestamp` is refreshed; `SPECS.md`'s is refreshed.
- [ ] No file under `docs/epics/epic-3-reviewer-selection-integration/issues/` is changed by
      this PR, and no `.go` file is changed.

## Relevant files / areas

- `docs/epics/epic-3-reviewer-selection-integration/EPIC_3.md` — `## Scope` (lines 29-59),
  `## Acceptance criteria` (lines 72-95), frontmatter `timestamp`
- `docs/planning/SPECS.md` — line 275, and the grammar block at lines 185-196 that is the
  evidence both additions were already decided
- Read, not changed:
  `docs/epics/epic-3-reviewer-selection-integration/issues/01-git-read-path-scoped-diff.md`
  (the `paths` parameter, lines 49-63),
  `issues/08-request-object-and-version.md` (the request object, `--system`, `version`),
  `docs/epics/log.md:34-46` (why each was pulled into the cut)

Governing decisions:
[specs 02 — CLI surface & argv grammar](../../../planning/specs/02-cli-surface-and-argv.md),
[specs 08 — System prompt & task assembly](../../../planning/specs/08-system-prompt-and-task-assembly.md),
[specs 10 — Repository confinement](../../../planning/specs/10-repository-confinement.md),
[SPECS § Interfaces](../../../planning/SPECS.md).

## Dependencies

- Blocked by: None. This PR touches `EPIC_3.md` and `SPECS.md`, which no other issue in
  this epic opens.
- Blocks: None.

## PR size note

Two files, roughly twenty lines of prose and one word. The natural split line, if it were
ever needed, is between the epic amendment and the `SPECS.md` miscount — but the miscount is
one word in a file this PR is already the only one to open, and filing it separately would
cost more than it saves.
