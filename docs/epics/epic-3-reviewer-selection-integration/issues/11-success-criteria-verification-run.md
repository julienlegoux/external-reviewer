---
type: Issue
title: "Run a real review through the pipeline and record the success criteria"
description: "Drive review-epics or review-issues through external-reviewer on this project's own artifacts, verify the leads, and record wall clock and quota headroom against SCOPE's three v1 success criteria."
tags: [epic-3]
timestamp: 2026-08-13T12:25:00Z
resource: https://github.com/julienlegoux/external-reviewer/issues/69
epic: 3
issue: 11
slug: success-criteria-verification-run
size: S
status: pr-open
gh_issue: 69
gh_pr: 92
depends_on: [1, 9, 10]
---

# Run a real review through the pipeline and record the success criteria

## Summary

The epic's last deliverable is not code: it is the evidence that the three success criteria
SCOPE sets for v1 are actually met. Two of them can only be answered by running the real
thing — a real `review-epics` or `review-issues` pass, through the swapped contract, on
this project's own artifacts, with leads verified against the files. The third is a
judgment the author makes from measured numbers.

This is the same shape as Epic 2's [issue 07](/epic-2-read-only-agentic-loop/issues/07-hand-run-measurements.md):
a document, no `.go` file touched, run by hand with a human watching. What differs is that
the binary is now invoked **by a skill rather than by the author**, which is the property
under test.

## Scope

- At least one real `review-epics` or `review-issues` run on this repository's own
  artifacts, launched the way the pipeline launches it — through the skill, not by
  hand-composing the command line.
- `docs/epics/epic-3-reviewer-selection-integration/VERIFICATION.md`, an OKF document in the
  epics bundle (`type: Reference`, frontmatter per the bundle rules), recording per run:
  the skill invoked, the tier requested and the `provider/model` it resolved to, the
  `--allow` subtrees, turns, in/out tokens, cost figure, wall clock, stop reason, and
  whether any cap bit.
- **Leads and their verification**: how many leads came back, how many survived
  verification against the files, and at least one example of each — a lead that held and a
  lead that did not. Volume is explicitly not the measure; SCOPE refuses to make novelty a
  success criterion.
- **Quota headroom**: what the run consumed against the account's limits, in whatever terms
  the provider exposes, and whether anything else on the account was affected. On the
  current subscription this — not spend — is what "unsupervised" costs
  ([SPECS § Departures from SCOPE](../../../planning/SPECS.md)).
- **The author's judgment on criterion 3**, stated plainly: is the wall-clock time low
  enough to reach for this tool again? A "no" is a real answer and belongs in the document.
- A short comparison against [MEASUREMENTS](/epic-2-read-only-agentic-loop/MEASUREMENTS.md):
  did the caps set in [issue 09](/epic-3-reviewer-selection-integration/issues/09-loop-caps-from-measurements.md)
  leave enough headroom, and did closing the `git_read` gap
  ([issue 01](/epic-3-reviewer-selection-integration/issues/01-git-read-path-scoped-diff.md))
  move the turn count the way MEASUREMENTS predicted (12 → 6–8)?
- `EPIC_3.md` links the document from its Notes; `docs/epics/index.md` gains a bullet in the
  mechanical form the bundle rules require; `docs/epics/log.md` records its creation.

## Out of scope

- Any `.go` change. A defect this run exposes is its own issue — a verification PR that also
  changes behaviour has verified something that no longer exists.
- Re-tuning the caps. If the run says they are wrong, that is a finding recorded here and a
  follow-up issue, not an edit folded in.
- Grading the *quality* of the external model's review beyond the criteria above. Whether
  the leads were good is answered by "how many survived verification", not by an essay.
- Publishing or sharing the report the external model produced; the transcripts belong in
  the PR description.

## Acceptance criteria / Definition of done

- [ ] At least one real skill-launched run completed against a real configured provider,
      with its full stderr transcript captured.
- [ ] `VERIFICATION.md` exists in the epic folder with valid OKF frontmatter and records,
      per run: the skill, the tier, the resolved `provider/model`, `--allow` subtrees,
      turns, in/out tokens, cost figure, wall clock, stop reason.
- [ ] **Criterion 1** answered: `_shared/review-interfaces.md` no longer mentions
      `opencode`, and the external pass demonstrably ran through `external-reviewer` — the
      transcript's `model` line is the evidence.
- [ ] **Criterion 2** answered: the run produced leads, and the document names how many
      survived verification against the files, with at least one surviving and one
      discarded example quoted.
- [ ] **Criterion 3** answered: wall-clock time and quota headroom recorded, with the
      author's own judgment stated. Token and cost figures recorded alongside, as the volume
      proxy that becomes real money if a tier ever names an API-key provider.
- [ ] The document states whether any cap bit, and if none did, how close the run came to
      each — so the next reader knows the tested margin rather than an untested one.
- [ ] The exit-code path the caller actually took is recorded — including a `1` or `2` if
      one occurred, and what the skill did about it.
- [ ] `docs/epics/index.md`, `docs/epics/log.md` and `EPIC_3.md` are updated.
      (`VERIFICATION.md` is a reference document, not an issue, so it does **not** go in
      `issues/index.md`.)
- [ ] No `.go` file is changed by this PR.

## Relevant files / areas

- `docs/epics/epic-3-reviewer-selection-integration/VERIFICATION.md` (new)
- `docs/epics/epic-3-reviewer-selection-integration/EPIC_3.md` — Notes gains the link
- `docs/epics/index.md`, `docs/epics/log.md`
- `D:/Project/skills` — the swapped contract being exercised, read but not changed here

Governing decisions:
[EPIC_3 § Acceptance criteria](/epic-3-reviewer-selection-integration/EPIC_3.md),
[scope 14 — Success criteria](../../../planning/scope/14-success-criteria.md),
[SPECS § Departures from SCOPE](../../../planning/SPECS.md) (criterion 3 turns on wall
clock and quota headroom, not spend),
[MEASUREMENTS](/epic-2-read-only-agentic-loop/MEASUREMENTS.md) (the numbers to compare
against).

## Dependencies

- Blocked by: [01 — git_read path-scoped diff](/epic-3-reviewer-selection-integration/issues/01-git-read-path-scoped-diff.md)
  (closes the `git_read` gap whose effect on the turn count this run measures against
  MEASUREMENTS),
  [09 — loop caps from measurements](/epic-3-reviewer-selection-integration/issues/09-loop-caps-from-measurements.md)
  (sets the `Bounds` this run reports cap bits and headroom against), and
  [10 — review-interfaces swap](/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md).
  There is nothing to verify until the pipeline calls the binary.
- Blocks: the epic's close.

## PR size note

Documentation only; well under the ~500-line target. If it grows past ~1000, the extra is
transcripts that belong in the PR description rather than the bundle.
