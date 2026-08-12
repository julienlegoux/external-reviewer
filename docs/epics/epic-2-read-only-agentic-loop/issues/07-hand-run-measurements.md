---
type: Issue
title: "Run a real review by hand and record the measurements"
description: "Run the completed loop against a real repository on a real provider and record turns, tokens, cost and wall clock — the epic's second deliverable and the input Epic 3's caps are sized from."
tags: [epic-2]
timestamp: 2026-08-12T18:10:00Z
epic: 2
issue: 07
slug: hand-run-measurements
size: S
status: in-progress
gh_issue: 15
resource: https://github.com/julienlegoux/external-reviewer/issues/15
depends_on: [4, 5, 6]
---

# Run a real review by hand and record the measurements

## Summary

This epic's second deliverable is not code: nobody knows how many turns a review takes
or how long it runs, and both Epic 3's loop caps and SCOPE's third success criterion
depend on numbers that only exist after this epic runs for real. No automated test can
stand in for it — the `faux` provider scripts responses, so it measures the harness, not
a reviewer. This PR runs the real thing by hand and writes the numbers down where Epic 3
will read them.

Risk 5 — a runaway run — is accepted deliberately here: the author launches by hand, the
stderr `turn` lines make the run observable while it happens, and there is a human at
the keyboard to interrupt.

## Scope

- At least three hand-run reviews against a real configured provider (`openai-codex`),
  chosen to span the range of surfaces this tool is pointed at:
  - a small one — a single directory of documentation;
  - a medium one — this repository's `docs/` and `internal/` together;
  - a large one — the heaviest read the pipeline has, an epic's merged diff, which is
    where risk 3 (context overflow) would first appear.
- For each run, recorded verbatim from the `done` and `turn` lines: turns taken,
  input/output tokens, accumulated cost figure, elapsed wall clock, the stop reason, the
  `--allow` subtrees, and roughly how much material was in reach.
- `docs/epics/epic-2-read-only-agentic-loop/MEASUREMENTS.md`, an OKF document in the
  epics bundle (`type: Reference`, frontmatter per the bundle rules), holding a table of
  the runs plus a short reading of what they imply for a turn cap and a wall-clock
  deadline — stated as observations, **not** as decided values.
- Whether any run hit context overflow, and at what surface size, recorded explicitly —
  that is the observation risk 3 was made observable for.
- `EPIC_2.md` updated to link the measurements from its Notes, and
  [Epic 3](/epic-3-reviewer-selection-integration/EPIC_3.md) left with a document to
  point at rather than a memory of a conversation.

## Out of scope

- **Setting the bound values.** They are Epic 3's decision, made from this data. This PR
  records numbers and observations; it does not change `bounds`' unset values.
- Any code change. If a hand run exposes a defect, it is a separate issue (or a fix in
  the issue that owns the behaviour), not folded in here — a measurement PR that also
  changes behaviour has measured something that no longer exists.
- Automated performance testing or benchmarks. The model's latency is not this project's
  to optimise.
- A coverage statement in the report — dropped along with the caps, and named in SCOPE as
  the reason risk 3 stays unmitigated in v1.

## Acceptance criteria / Definition of done

- [ ] Three or more real runs completed against a real provider, each one's full stderr
      transcript captured.
- [ ] `MEASUREMENTS.md` exists in the epic folder with valid OKF frontmatter and records,
      per run: the exact command line, `--allow` subtrees, turns, in/out tokens, cost
      figure, wall clock, stop reason, and the approximate size of the material in reach.
- [ ] At least one run exercises each of the four tools, and the transcript shows it —
      the tools are verified against a real model's tool-calling, not only against
      `faux`.
- [ ] The document states plainly whether context overflow was hit, and if so at what
      surface size; if it was not hit, it says the largest surface that did **not** hit
      it, so Epic 3 knows the tested ceiling rather than an untested one.
- [ ] The document ends with observations framed for Epic 3 — a plausible turn cap range
      and wall-clock deadline range, each with the run that suggests it — explicitly
      labelled as input to a decision, not as the decision.
- [ ] The bundle's root index `docs/epics/index.md` gains a bullet for the new document
      in the mechanical form the bundle rules require, `docs/epics/log.md` records its
      creation, and `EPIC_2.md` links it. (`MEASUREMENTS.md` is a reference document, not
      an issue, so it does **not** go in `issues/index.md`; there is no epic-level
      `index.md` in this bundle.)
- [ ] No `.go` file is changed by this PR.

## Relevant files / areas

- `docs/epics/epic-2-read-only-agentic-loop/MEASUREMENTS.md` (new)
- `docs/epics/epic-2-read-only-agentic-loop/EPIC_2.md` — Notes gains the link
- `docs/epics/index.md` (the bundle's root index gains the new document),
  `docs/epics/log.md`

Governing decisions:
[EPIC_2 § Goal](/epic-2-read-only-agentic-loop/EPIC_2.md) (measurements as the second
deliverable),
[specs 07 — Agent loop mechanics](../../../planning/specs/07-agent-loop-mechanics.md)
(turns and wall clock are the meaningful bounds on a subscription),
[SPECS § Departures from SCOPE](../../../planning/SPECS.md) (success criterion 3 turns on
wall clock and quota headroom, not spend).

## Dependencies

- Blocked by: [04 — read_file](/epic-2-read-only-agentic-loop/issues/04-read-file-tool.md),
  [05 — search](/epic-2-read-only-agentic-loop/issues/05-search-tool.md),
  [06 — git_read](/epic-2-read-only-agentic-loop/issues/06-git-read-tool.md) — all four
  tools must exist for the measurement to describe the tool this epic ships.
- Blocks: Epic 3's loop-cap issues, which have no input without this.

## PR size note

Documentation only; well under the ~500-line target. If it grows past ~1000, the extra
is transcripts that belong in the PR description rather than the bundle.
