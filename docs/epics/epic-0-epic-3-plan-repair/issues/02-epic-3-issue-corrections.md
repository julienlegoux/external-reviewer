---
type: Issue
title: "Repair Epic 3's cross-repository pointer, dependency edges and the models criterion"
description: "Qualify issue 10's gh_issue so no consumer resolves it locally, add the three missing dependency edges, and give issue 07 a criterion for the silently empty catalog."
tags: [epic-0]
timestamp: 2026-08-13T07:00:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/72
epic: 0
issue: 02
slug: epic-3-issue-corrections
size: S
status: done
gh_issue: 72
gh_pr: 77
depends_on: []
---

# Repair Epic 3's cross-repository pointer, dependency edges and the models criterion

## Summary

Three defects in Epic 3's issue files, grouped because they are all edits inside
`docs/epics/epic-3-reviewer-selection-integration/issues/` and each one costs more after
`implement-epic` starts than before it.

**The wrong repository.** `issues/10-review-interfaces-external-reviewer-swap.md:13` carries
`gh_issue: 37`, an unqualified integer, with the repository named only in `resource`
(`https://github.com/julienlegoux/skills/issues/37`). Verified: issue 37 in *this*
repository is *"Settle the done line's stop-reason vocabulary and document it in SPECS"*,
state MERGED — the previous remediation lane's own first issue. `implement-issue` and
`implement-epic` mirror every status transition onto `gh_issue` and close it at `done`; run
from this repository, which is where the bundle lives and where those skills are invoked,
they would comment on and attempt to close that merged issue while `julienlegoux/skills#37`
never advanced.

**Three absent dependency edges.** Issue 11 is a real, quota-consuming, hand-supervised
review run, and its criteria ask whether any cap bit (`11:87-88`) and whether closing the
`git_read` gap moved the turn count the way MEASUREMENTS predicted (`11:52-56`). Issue 09
is what sets `Bounds`; issue 01 is what closes the gap. With `depends_on: [10]`,
`implement-epic` may start issue 11 against a binary whose `Bounds` is still the zero value
at `internal/cli/run.go` — and the run would have to be done again, on quota.

**A silent empty catalog.** `models` is the command journeys 3 and 4 exist around. For
`openrouter`, `vercel-ai-gateway`, `nvidia` and `github-copilot` it prints nothing without
`--refresh`, because `GetModel` returns nil until `Refresh` runs
([specs 05](../../../planning/specs/05-tier-assignment-schema.md)). Issue 07's eleven
criteria assert that no `Refresh` happens without the flag (`07:74`) and never that the
emptiness explains itself — so the exact silent-fallback failure `EPIC_3.md:41-44` names is
reproduced in the command built to prevent it.

## Scope

### The cross-repository pointer

- `issues/10-…:13` — `gh_issue` is qualified so no consumer can resolve it against this
  repository. Write it as the `owner/repo#number` form (`julienlegoux/skills#37`), which is
  what `gh` itself accepts and what `resource` already agrees with.
- The ⚠ block at `10:21-27` repeats the qualification in prose: the GitHub issue for this
  work is `julienlegoux/skills#37`, and a bare `37` in this repository is a different,
  already-merged issue.
- Riding along in the same file: issue 09 owns the decision whether any of the three
  ceilings is exposed on the command line (`09:56-60`), and issue 10 — which writes the
  copy-pasteable invocation into `review-interfaces.md` — does not depend on it. One
  sentence in issue 10's Scope stating that the invocation template takes the defaults
  unless issue 09 adds a flag, and that if one is added the template gains it.

### The dependency edges

Three edges, all currently absent, and both ends of each are updated — frontmatter and
prose:

- **Issue 11 → issues 01 and 09.** `depends_on: [10]` becomes `[1, 9, 10]`, and the
  "Blocked by" line names both with the reason (issue 09 sets the caps the run reports
  against; issue 01 closes the gap whose effect the run measures).
- **Issue 09's "Blocks" line** reads *"None strictly"* at `09:132`, which is what makes the
  omission look deliberate. It names issue 11.
- **Issue 07 behind issue 06.** `depends_on: [2, 4]` becomes `[2, 4, 6]`. Issues 06, 07 and
  08 all rewrite `internal/cli/run.go`'s subcommand switch and `internal/cli/usage.go`'s
  `usageText`; issue 08 already applies exactly this reasoning against issue 05
  (`08:126-129`), and 07 is ordered against neither of the others. Issue 06's "Blocks" line
  gains issue 07 in return.

### The models criterion

- One criterion added to `issues/07-models-command.md` § *Acceptance criteria*: a
  credentialed provider reporting `CanRefreshModels()` whose catalog is empty is **named as
  needing `--refresh`** rather than rendering as an absent row, asserted with a registry
  holding one such provider. It sits with the other `--refresh` criteria (`07:71-76`), in
  the same written-failing-first, black-box register.

## Out of scope

- **`size` letters and `## PR size note` blocks**, in every file this PR opens. That sweep
  is [issue 03](/epic-0-epic-3-plan-repair/issues/03-epic-3-sizes-and-surfaces.md), which
  runs after this one precisely so the two do not fight over the same ten files.
- **`issues/index.md`** — its bullets carry sizes and statuses, both of which issue 03 owns.
  Nothing in this PR changes a bullet.
- **Amending `pipeline-interfaces.md`.** The schema defines `gh_issue` as an unqualified
  integer and has no way to express a cross-repository issue, which is what made this defect
  possible. That file lives in `julienlegoux/skills`; this PR works around it locally and the
  gap is filed upstream separately ([EPIC_0 § Out of scope](/epic-0-epic-3-plan-repair/EPIC_0.md)).
- **Re-cutting or re-scoping any Epic 3 issue.** Three named defects, nothing else.
- Any `.go` file.

## Acceptance criteria / Definition of done

- [ ] **No `gh_issue` in `docs/epics/epic-3-reviewer-selection-integration/issues/` resolves
      to the wrong repository**: for every issue file, the `gh_issue` value and the
      `resource` URL name the same repository and the same number. Mechanically —
      `grep -n '^gh_issue:' issues/*.md` shows issue 10's value carrying `julienlegoux/skills`,
      and every other value a bare integer whose `resource` points at
      `julienlegoux/external-reviewer`.
- [ ] `gh issue view 37 --json title,state` still returns the merged stop-reason issue,
      unchanged and untouched by this PR — the check that the qualification was the fix and
      not a renumbering.
- [ ] Issue 10's ⚠ block states the qualification, and its Scope states what the invocation
      template assumes about issue 09's ceilings.
- [ ] **The graph is a superset of what the criteria need**: issue 11's `depends_on` is
      `[1, 9, 10]`; issue 07's is `[2, 4, 6]`; every other file's is unchanged.
- [ ] **Both ends agree.** For every edge, the "Blocked by" line of the dependent and the
      "Blocks" line of the dependency name each other. Issue 09's "Blocks" no longer reads
      "None strictly"; issue 06's names issue 07.
- [ ] **The graph is acyclic and strictly forward**: no `depends_on` entry is greater than or
      equal to its own `issue` number, verifiable by reading the eleven frontmatter blocks.
- [ ] Issue 07 carries a criterion covering what a credentialed, refreshable provider with an
      empty catalog renders, written as an assertion over stdout with a named registry shape
      — not as a description of the behaviour.
- [ ] Every file this PR changes has its `timestamp` refreshed.
- [ ] No `.go` file is changed, and `issues/index.md` is untouched.

## Relevant files / areas

All under `docs/epics/epic-3-reviewer-selection-integration/issues/`:

- `10-review-interfaces-external-reviewer-swap.md` — frontmatter `gh_issue` (line 13), the
  ⚠ block (lines 21-27), `## Scope` (the invocation bullet, lines 52-54)
- `11-success-criteria-verification-run.md` — frontmatter `depends_on` (line 14), the
  Dependencies section
- `09-loop-caps-from-measurements.md` — the "Blocks" line (line 132)
- `07-models-command.md` — frontmatter `depends_on` (line 14), `## Acceptance criteria`
  (lines 71-76), the Dependencies section
- `06-tiers-command.md` — the "Blocks" line
- Read, not changed: `08-request-object-and-version.md:126-129` (the ordering argument this
  PR applies to issue 07), `01-git-read-path-scoped-diff.md`

Governing decisions:
[EPIC_0 § B and § C](/epic-0-epic-3-plan-repair/EPIC_0.md) (and its Notes, for why the
issue schema itself cannot be fixed from here),
[conventions 04 — Cross-repository conventions](../../../planning/conventions/04-cross-repo-conventions.md),
[specs 05 — Tier assignment file schema](../../../planning/specs/05-tier-assignment-schema.md).

## Dependencies

- Blocked by: None.
- Blocks: [03 — sizes and recorded surfaces](/epic-0-epic-3-plan-repair/issues/03-epic-3-sizes-and-surfaces.md),
  which sweeps the same ten files for `size` and the PR-size notes, and rewrites GitHub #60
  from issue 01's final `.md`.

## PR size note

Three defects across five files, none more than a paragraph — well under a hundred changed
lines. If it grows, the cause will be the dependency edges pulling in prose rewrites the
edge itself does not need; the split line is between the cross-repository pointer (one file)
and the graph (four files), and the models criterion rides with whichever half is left.
