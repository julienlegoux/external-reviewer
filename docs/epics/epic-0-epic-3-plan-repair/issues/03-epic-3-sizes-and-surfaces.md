---
type: Issue
title: "Re-grade Epic 3's issue sizes and sync the recorded surfaces"
description: "Replace the uniform M and the boilerplate PR-size note with graded sizes and per-issue notes, then make issues/index.md, docs/epics/index.md, log.md and GitHub #60 say what was actually cut."
tags: [epic-0]
timestamp: 2026-08-13T05:50:38Z
resource: https://github.com/julienlegoux/external-reviewer/issues/73
epic: 0
issue: 03
slug: epic-3-sizes-and-surfaces
size: M
status: open
gh_issue: 73
depends_on: [2]
---

# Re-grade Epic 3's issue sizes and sync the recorded surfaces

## Summary

Epic 3's eleven issues carry one size letter and one sentence between them. Issues 01–10 are
all `size: M`, and eight of them close with the byte-identical *"Target ~500 changed lines;
if this grows past ~1000, split it before opening the PR"* — a note that targets the top of
M and authorises an L on every issue, which is the opposite of what a size is for. Only
issues 10 and 11 carry a note that says anything about their own bulk.

There is a measured yardstick to grade against, in this repository: Epic 2 shipped a new
package plus its tool and its tests seven times, and graded that shape **L five times out of
seven** (`epic-2-…/issues/*.md` — 01, 02, 03, 05, 06 are L; 04 is M; 07, documentation only,
is S). Against that, three of Epic 3's issues are L work (02, 04, 05) and one is S — issue
09, which describes itself as *"a data change plus one correctness gap, not a
restructuring"* (`09:30`).

The same sweep closes the recorded surfaces that still describe a cut that has not happened:
`docs/epics/index.md:14` claims `create-issues` was deliberately deferred for Epic 3, and
GitHub **#60** is the only one of the eleven whose body carries no scope and no acceptance
criteria — it was adopted at Epic 2's close rather than created by the cut, and adopting it
skipped the body rewrite the other ten got.

## Scope

### Sizes and PR-size notes

- **Re-grade all eleven `size` letters** against Epic 2's measured shapes, not against the
  target. The starting judgment, to be confirmed or revised per issue in the PR body:
  - **L** — 02 (new `internal/family` package, two data tables, whole-catalog table-driven
    test), 04 (the four-step resolution chain plus refresh ordering), 05 (three flags, the
    exit-code classification and the surfaces over it).
  - **S** — 09 (data change plus one correctness gap), 11 (documentation only, already S).
  - **M** — the rest, if and only if the grader can say what the ~200–500 lines are.
- **Replace every `## PR size note`** with one naming *that issue's* bulk and *its* natural
  split line — where the PR would be cut if it ran long, in that issue's own terms (a
  package boundary, a flag, a command). Issues 10 and 11 already have bespoke notes and
  keep them; the eight boilerplate copies go.
- **`issues/index.md`** — every bullet's size letter matches its file, and the bullets stay
  in the mechanical form the bundle rules require
  (`* [Title](path) - <size>, <status>, <link>`).

### The recorded surfaces

- **`docs/epics/index.md:14`** — the Epic 3 bullet's *"`create-issues` deliberately
  deferred"* clause goes; the issues exist and `log.md:43-69` records them. The
  cross-repository half of the sentence stays, because it is still true.
- **`docs/epics/log.md:44`** — the entry records *"all sized S or M — nothing needed the L
  ceiling"* as though that had been measured. This epic's log entry says what the grading
  actually was, with the Epic 2 comparison that produced it. The original entry is corrected
  in place rather than deleted: it is the claim this issue exists to disprove.
- **GitHub #60** — rewritten to the same rendering the other ten got (`#61`–`#69` and
  `julienlegoux/skills#37` each carry a full `## Acceptance criteria`): the issue file's body
  from `# <title>` down, with bundle-relative links rewritten to absolute
  `github.com/julienlegoux/external-reviewer/blob/develop/…` URLs, as `gh issue view 61`
  shows. The original discovery text — the reproduced `fatal: bad revision` transcript and
  the measured cost — is **kept**, as the Summary it already is.

## Out of scope

- **Re-cutting, merging or splitting any Epic 3 issue.** A size is a grade on the issue as
  written; if an issue turns out to need splitting, that is a finding for the PR body, not an
  edit here.
- **Rewriting the bodies of #61–#69.** They already carry the full rendering. Only #60 is
  behind.
- **The `depends_on` edges, the cross-repository `gh_issue`, and issue 07's criterion** —
  [issue 02](/epic-0-epic-3-plan-repair/issues/02-epic-3-issue-corrections.md), which this
  one runs behind.
- **`EPIC_3.md`** — [issue 01](/epic-0-epic-3-plan-repair/issues/01-epic-3-scope-amendment.md)
  owns its Scope and Acceptance criteria; nothing here opens it.
- Any `.go` file.

## Acceptance criteria / Definition of done

- [ ] **No two consecutive issues share a byte-identical `## PR size note`.** Mechanically:
      extracting the note from each of the eleven files yields eleven distinct blocks —
      `grep -c 'Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.'`
      across `issues/*.md` returns 0.
- [ ] Each note names what that issue's bulk actually is and where it would split, in terms
      specific to the issue (a package, a flag, a command) — not a line count alone.
- [ ] **`size` is not the same letter on all ten code issues**:
      `grep -h '^size:' issues/*.md | sort | uniq -c` shows at least three distinct letters,
      and every `L` is defended in the PR body against Epic 2's measured shapes.
- [ ] `issues/index.md`'s size letter for each bullet equals the `size:` in the file it
      links, checked pairwise for all eleven.
- [ ] `grep -n 'deliberately deferred' docs/epics/index.md` returns nothing; the Epic 3
      bullet still names the two-repository span.
- [ ] `docs/epics/log.md` no longer asserts unqualified that nothing needed the L ceiling,
      and this epic's entry records the re-grade with the Epic 2 comparison.
- [ ] `gh issue view 60 --json body --jq .body | grep -c '## Acceptance criteria'` returns
      non-zero, and the same command over `#61` and `#69` still does — the rendering matches,
      it was not invented for #60.
- [ ] `gh issue view 60` still contains the `fatal: bad revision 'internal/confine'`
      transcript: the discovery text was kept, not replaced.
- [ ] Every changed file's `timestamp` is refreshed, and no `.go` file is changed.

## Relevant files / areas

- `docs/epics/epic-3-reviewer-selection-integration/issues/01-…` through `11-…` —
  frontmatter `size` (line 11 in each) and the `## PR size note` section (the file's last
  block)
- `docs/epics/epic-3-reviewer-selection-integration/issues/index.md` — eleven bullets
- `docs/epics/index.md` — line 14, the Epic 3 bullet
- `docs/epics/log.md` — line 44 (the "all sized S or M" claim) and this epic's entry
- Read as the yardstick, not changed:
  `docs/epics/epic-2-read-only-agentic-loop/issues/*.md` (`size:` on line 10 of each)
- GitHub: issue #60 (body), with `#61` as the rendering reference

Governing decisions:
[EPIC_0 § D](/epic-0-epic-3-plan-repair/EPIC_0.md),
[CONVENTIONS § Documentation & comments](../../../planning/CONVENTIONS.md).

## Dependencies

- Blocked by: [02 — issue corrections](/epic-0-epic-3-plan-repair/issues/02-epic-3-issue-corrections.md).
  It edits five of the same ten files, and #60's body must be rendered from an issue file
  that is already correct — rewriting it first would publish a body that is wrong again a
  PR later.
- Blocks: None.

## PR size note

The bulk is thirteen files touched shallowly — one letter and one short block each — plus
one GitHub body rewritten from a file. Around a hundred changed lines, but spread wide, so
the review cost is per-file rather than per-line. If it runs long the split is clean and
already drawn: the ten issue files and `issues/index.md` are the sizing half, and
`docs/epics/index.md`, `log.md` and #60 are the recorded-surfaces half.
