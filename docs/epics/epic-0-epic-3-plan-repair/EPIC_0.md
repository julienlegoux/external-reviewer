---
type: Epic
title: "Epic 3 plan repair"
description: "The eight findings of Epic 3's issue review, converted into the plan repairs that must land before its first issue is implemented."
tags: [epic, remediation]
timestamp: 2026-08-13T04:39:56Z
resource: https://github.com/julienlegoux/external-reviewer/issues/70
epic: 0
slug: epic-3-plan-repair
status: open
gh_issue: 70
milestone: 5
source: docs/reviews/2026-08-13-issues-epic-3.md
---

# Epic 0: Epic 3 plan repair

## Goal

Epic 3's eleven issues are cut, filed and linked, and the review that graded them found
the plan around them wrong in eight places. None of the eight is a defect in what the
issues ask for — the coverage is complete and the transcriptions are faithful. What is
wrong is the frame: the epic does not name scope two of its issues will ship, one issue
points its status mirror at a merged issue in the wrong repository, the dependency graph
lets a real quota-consuming review run before the thing it measures exists, and every code
issue carries the same size and the same boilerplate note.

So the goal is not "fix the review findings". It is that **Epic 3's plan says what Epic 3
will build, in an order that can actually run**, before `implement-epic` starts — because
every one of these costs more after implementation than before it. A wrong `gh_issue`
comments on somebody else's finished work. A missing `depends_on` burns a hand-supervised
model run that has to be done again. An epic that does not name the scope its issues ship
is the file `close-epic` and `review-implementation` grade the merged diff against.

## Scope

Seven repairs, all in planning artifacts. No product code is touched, with one exception
noted in repair G — a `README.md` that does not exist yet and that no issue creates.

### A — EPIC_3 absorbs the scope its own issues will ship

`EPIC_3.md`'s `## Scope` and `## Acceptance criteria` gain issue 01's `paths` wire
parameter on `git_read` and issue 08's JSON `{"system","task"}` request object, `--system`
and `version`. Both are legitimate and already traceable — `SPECS.md:189-196` specifies
`--system` and `version` in the `review` grammar, and `docs/epics/log.md:34-46` records why
each was pulled into the cut — but the epic itself was never amended, and the words appear
in it nowhere. The reasoning is written already; this moves it into the file that is graded
against.

The issues are not weakened. The epic is the file that is wrong.

Riding along, because issues 05 and 09 both rest on it: `SPECS.md:275` introduces the
non-model stop reasons as *"one of five fixed CLI words"* and then lists six (`usage`,
`help`, `no_reviewer`, `interrupted`, `failed`, `bounds`). One word.

### B — Issue 10's cross-repository `gh_issue` stops resolving to the wrong repository

`issues/10-review-interfaces-external-reviewer-swap.md:14` carries `gh_issue: 37`, an
unqualified integer, with the repository named only in `resource`. Verified: issue 37 in
*this* repository is "Settle the done line's stop-reason vocabulary and document it in
SPECS", state MERGED, Epic 0's own first issue from the previous remediation cycle.
`implement-issue` and `implement-epic` mirror every status transition onto `gh_issue` and
close it at `done`; run from this repository — which is where the bundle lives and where
those skills are invoked — they would comment on and attempt to close that merged issue
while `julienlegoux/skills#37` never advanced.

The field is qualified so no consumer can resolve it locally, and the ⚠ block at `:21-27`
repeats it.

Riding along, in the same file: issue 09 owns the decision whether any of the three
ceilings is exposed on the command line (`09:57-62`), and issue 10 — which writes the
copy-pasteable invocation into `review-interfaces.md` — does not depend on it. One sentence
stating that the template takes the defaults unless issue 09 adds a flag.

### C — The dependency graph is repaired to match what the issues actually need

Three edges, all currently absent:

- **Issue 11 → issues 1 and 9.** Its criteria ask whether any cap bit (`11:86-88`) and
  whether closing the `git_read` gap moved the turn count the way MEASUREMENTS predicted
  (`11:52-56`). Issue 09 is what sets `Bounds`; issue 01 is what closes the gap. With
  `depends_on: [10]`, `implement-epic` may start issue 11 against a binary whose `Bounds`
  is still the zero value at `internal/cli/run.go:54` — and issue 11 is a real,
  quota-consuming, hand-supervised review that would then have to be run again.
- **Issue 09's "Blocks" line** reads *"None strictly"* at `:132`, which is what makes the
  omission look deliberate. It names issue 11.
- **Issue 07 behind issue 06.** Issues 06, 07 and 08 all rewrite `internal/cli/run.go`'s
  subcommand switch and `internal/cli/usage.go`'s `usageText`. Issue 08 already applies
  exactly this reasoning against issue 05 (`08:126-129`); 07 is ordered against neither of
  the others.

### D — Sizes stop being uniform, and the boilerplate PR-size note is retired

Issues 01–10 are all `size: M`, each closing with the byte-identical sentence *"Target ~500
changed lines; if this grows past ~1000, split it before opening the PR"* — which targets
the top of M and authorises an L on every issue. Measured against Epic 2, where a new
package plus its tool and its tests was graded `L` five times out of seven, three of these
are L work (02, 04, 05) and one is S (09, which describes itself as *"a data change plus
one correctness gap, not a restructuring"*).

Re-grade, replace each note with one naming what that issue's bulk actually is and where it
would split, and update the size letters in `issues/index.md`. `docs/epics/log.md:6` records
*"all sized S or M — nothing needed the L ceiling"* as though that were measured; the log
entry for this epic says what it actually was.

### E — Issue 07 gets a criterion for the silent-empty catalog

`models` is the command journeys 3 and 4 exist around. For `openrouter`,
`vercel-ai-gateway`, `nvidia` and `github-copilot` it prints nothing without `--refresh`,
because `GetModel` returns nil until `Refresh` runs. Issue 07's eleven criteria assert that
no `Refresh` happens without the flag (`07:74`) and never that the emptiness explains
itself — so the exact silent-fallback failure `EPIC_3.md:41-44` names is reproduced in the
command built to prevent it.

One criterion: a credentialed provider reporting `CanRefreshModels()` whose catalog is empty
is named as needing `--refresh`, asserted with a registry holding one such provider.

### F — The outward-facing surfaces match the cut that happened

- `docs/epics/index.md:12` still reads *"spans two repositories; `create-issues`
  deliberately deferred"*. The issues exist; `log.md:5` records them. The cross-repository
  half stays, the deferral clause goes.
- GitHub `#60` is the only one of the eleven whose body carries no scope and no acceptance
  criteria — it was adopted at Epic 2's close rather than created by the cut, and adopting
  it skipped the body rewrite the other ten got (#61–#69 and `julienlegoux/skills#37` each
  verified to carry a full `## Acceptance criteria`). Rewritten to the same rendering,
  keeping the original discovery text as the Summary it already is.

### G — The README the validation boundary is supposed to live in

`docs/planning/specs/05-tier-assignment-schema.md:118-122` says v1's OpenAI-only validation
is *"stated in the README so it reads as an honest boundary rather than an implied
promise"*, and `EPIC_3.md:130-132` repeats the claim as a Note. The repository has no
`README.md` at all, and no issue creates one.

It lands here rather than in Epic 3 because it is a promise the plan already made and no
issue owns — the same shape as every other repair in this lane.

## Out of scope

- **Re-grading the findings.** The severities are the report's. Nothing here re-reviews
  Epic 3's issues.
- **Product code.** Every repair above edits planning artifacts; `README.md` is the one new
  file and it is documentation. No `.go` file is touched by this epic.
- **The `pipeline-interfaces.md` schema gap.** It has no way to express a cross-repository
  issue, which is what made repair B possible in the first place. That file lives in
  `julienlegoux/skills`; the gap is recorded in Notes and filed upstream separately.
- **Epic 3's own implementation.** This lane repairs the plan and stops.

## Acceptance criteria

- [ ] **A** — `EPIC_3.md`'s `## Scope` and `## Acceptance criteria` each name the `paths`
      parameter, the JSON request object, `--system` and `version`; `grep -c` for each of
      the four returns non-zero in both sections. `SPECS.md:275` says "six".
- [ ] **B** — No `gh_issue` value anywhere in `docs/epics/epic-3-*/issues/` resolves to an
      issue in `julienlegoux/external-reviewer` other than the one the file's `resource`
      names. Issue 10's ⚠ block repeats the qualification, and states what the invocation
      template assumes about issue 09's ceilings.
- [ ] **C** — Every issue's `depends_on` is a superset of the issues its own Acceptance
      criteria require to have landed; every "Blocked by" / "Blocks" line in prose agrees
      with the frontmatter of both ends; the graph is acyclic and strictly forward (no
      issue depends on a higher number).
- [ ] **D** — No two consecutive issues share a byte-identical `## PR size note`; each note
      names its own target and its natural split line. `size` is not the same letter on all
      ten code issues, and `issues/index.md`'s letters match the files.
- [ ] **E** — Issue 07 carries a criterion covering what a credentialed, refreshable
      provider with an empty catalog renders, asserted rather than described.
- [ ] **F** — `docs/epics/index.md`'s Epic 3 bullet no longer claims the issues are not
      cut; `gh issue view 60` returns a body carrying `## Acceptance criteria`, like
      #61–#69.
- [ ] **G** — `README.md` exists and states v1's OpenAI-only validation boundary in the
      terms `specs/05:118-122` requires.
- [ ] Every repair above is verifiable from the repository without re-reading the report.

## Dependencies

None. This epic precedes [Epic 3](/epic-3-reviewer-selection-integration/EPIC_3.md)'s
implementation, which is what makes it epic 0.

## Context

- [Issue review report, Epic 3 issues](../../reviews/2026-08-13-issues-epic-3.md) — the
  single report this epic converts.
- [SPECS](../../planning/SPECS.md), [CONVENTIONS](../../planning/CONVENTIONS.md),
  [DRIFT](../../planning/DRIFT.md) — the standards the findings are measured against.
- [Epic 3](/epic-3-reviewer-selection-integration/EPIC_3.md) and its eleven issues — the
  artifacts every repair edits.

## Notes

**Findings converted**: 8 extracted from
[`docs/reviews/2026-08-13-issues-epic-3.md`](../../reviews/2026-08-13-issues-epic-3.md)
(5 × P2, 3 × P3), **0 dropped as stale**, 8 grouped into 6 repairs, plus one open question
from the report promoted to a seventh (G). Nothing was recorded `won't-fix`: no finding in
this round was judged wrong, and none was already settled.

**Staleness was checked, not assumed.** No commit touched `docs/epics/` or
`docs/planning/` between the report and this conversion — the last is `72d9350 docs: add
Epic 3 issues`, which is what the report reviewed — and every cited `file:line` was
re-verified against the working tree. `gh issue list --state all` was read: no open or
closed issue covers any of the eight, and `docs/planning/DRIFT.md` has accepted none of
them.

**Two riders were folded in rather than filed.** `SPECS.md:275`'s "five fixed CLI words"
introducing six, and the sentence stating what issue 10's invocation template assumes about
issue 09's ceilings. Both are edits, not decisions, and both land in files repairs A and B
already open.

**Recorded, not scheduled — upstream.** `_shared/pipeline-interfaces.md`'s issue schema has
no way to express a cross-repository `gh_issue`; it defines the field as an unqualified
integer, which is what let issue 10 point at the wrong repository. Repair B works around it
locally with a qualified value. The schema itself lives in `julienlegoux/skills` and this
epic cannot reach it — to be filed there once this lane lands.

**Reports read**: `docs/reviews/2026-08-13-issues-epic-3.md` (converted). Skipped as already
consumed: `docs/REPORT_1.md`, applied 2026-08-09 as a revision to the twelve issues of Epics
1 and 2 and carrying its own Resolution table; `docs/REPORT_2.md`, consumed by the retired
Epic 0 "Skeleton hardening" (`docs/epics/log.md:253`). No lint reports exist in this
repository.
