---
type: Issue
title: "Add the README stating v1's OpenAI-only validation boundary"
description: "Create the README the plan twice promises, whose load-bearing sentence is that the mechanism is generic while v1 is validated against openai-codex alone."
tags: [epic-0]
timestamp: 2026-08-13T05:50:38Z
resource: https://github.com/julienlegoux/external-reviewer/issues/74
epic: 0
issue: 04
slug: readme-validation-boundary
size: S
status: open
gh_issue: 74
depends_on: []
---

# Add the README stating v1's OpenAI-only validation boundary

## Summary

The plan promises a README twice, and the repository has none.
`docs/planning/specs/05-tier-assignment-schema.md:116-121` decides the validation target
and says the boundary is *"stated in the README so it reads as an honest boundary rather
than an implied promise"*; `EPIC_3.md:130-132` repeats the claim in its Notes — *"The binary
names no provider anywhere except the family classifier's table, so all ~35 remain
reachable; v1 is validated against `openai-codex` alone. That is an honest boundary, not an
implied promise."* `ls README.md` returns nothing, and none of Epic 3's eleven issues
creates it.

This lands in the remediation lane rather than in Epic 3 because it is a promise the plan
already made and no issue owns — the same shape as every other repair here.

The point is not documentation for its own sake. A binary that reaches ~35 providers and is
tested against one is either honest about that or it is making a promise it has not
checked, and the difference is a single paragraph a human sees when they land on the
repository.

## Scope

- **`README.md` at the repository root**, written for a human landing on the repository —
  not a knowledge dump, per
  [CONVENTIONS § Repository layout](../../../planning/CONVENTIONS.md). Short. The knowledge
  lives in the `docs/` bundles and the README points at them.
- **The load-bearing paragraph**, in the terms `specs 05` requires: the mechanism is fully
  generic — no provider is named anywhere in the binary except the family classifier's data
  table — and **v1 is validated against `openai-codex` over OAuth alone**. Everything else
  is "the mechanism is generic, only OpenAI is tested". Stated as a boundary the reader can
  act on, not as a disclaimer at the bottom.
- **What the tool is**, in a few lines: a CLI that hands a repository to a second model for
  review, which reads through four read-only tools and returns leads, so a reviewing skill
  can verify them against the files.
- **The read-only guarantee as it is actually enforced** — the binary writes nothing inside
  the repository under review and produces no output file; the write half of the filesystem
  API is not in the codebase and CI rejects it
  ([CONVENTIONS § Code style](../../../planning/CONVENTIONS.md)). Credential storage belongs to
  `kern-link` and lives in `~/.pi/agent/`, which
  [DRIFT § Epic 1 / 05](../../../planning/DRIFT.md) already forced the plan to say out loud —
  the README says the same thing rather than the older unqualified claim.
- **Pointers, not copies**: `docs/planning/SCOPE.md`, `SPECS.md`, `CONVENTIONS.md`, and
  `docs/epics/` for where the work stands.

## Out of scope

- **Installation, packaging and release documentation.** Named non-goals in SCOPE and
  repeated in `EPIC_3.md`'s Out of scope. If the module path appears at all it is one line
  naming what `go install` resolves to, never a setup section.
- **A usage reference.** `external-reviewer help` and `SPECS.md` § Interfaces are the
  grammar's home; a second copy in the README would be the one that drifts.
- **A model catalogue or a list of recommended models** — an explicitly refused artifact
  ([specs 05](../../../planning/specs/05-tier-assignment-schema.md)); it would start rotting the
  day it was written.
- **Badges, contribution guidelines, a licence choice.** None is decided anywhere in the
  plan, and deciding one here would be this issue inventing scope.
- **Amending `specs 05` or `EPIC_3.md`.** Both already say the right thing; this PR makes
  them true.
- Any `.go` file.

## Acceptance criteria / Definition of done

- [ ] `README.md` exists at the repository root.
- [ ] It states the validation boundary in the terms `specs 05:116-121` requires — the
      mechanism is generic, `openai-codex` is what v1 is validated against, everything else
      is untested. `grep -i 'openai-codex' README.md` matches, and the sentence around the
      match says *validated against*, not *supports*.
- [ ] The boundary is visible without scrolling past the description of what the tool is —
      it is a boundary, not a footnote.
- [ ] `grep -i 'no provider is named' README.md` (or an equivalent phrasing) shows the
      generic half stated alongside the tested half, so the reader is not left thinking the
      binary is OpenAI-specific.
- [ ] The read-only guarantee is stated as enforced, with the credential-store exception
      `DRIFT` records — no unqualified "the binary writes no files".
- [ ] `grep -c 'install' README.md` shows at most one line, and no `## Install` heading
      exists.
- [ ] The README links `docs/planning/SCOPE.md`, `docs/planning/SPECS.md` and
      `docs/planning/CONVENTIONS.md` with working relative paths; every link resolves
      against the working tree.
- [ ] No `.go` file is changed, and nothing under `docs/` is changed by this PR.

## Relevant files / areas

- `README.md` — **new**, repository root. This is the only file created.
- Read, not changed:
  - `docs/planning/specs/05-tier-assignment-schema.md:116-121` — the requirement, and the
    exact terms it asks for
  - `docs/epics/epic-3-reviewer-selection-integration/EPIC_3.md:130-132` — the same claim
    in the epic's Notes
  - `docs/planning/CONVENTIONS.md` § Repository layout (what a README is for here) and
    § Code style (the write-API guard)
  - `docs/planning/DRIFT.md` § Epic 1 / 05 — why "the binary writes no files" is stated with
    its credential-store exception
  - `docs/planning/SCOPE.md`, `docs/planning/SPECS.md` — what the tool is, in the words the
    project already uses

Governing decisions:
[specs 05 — Tier assignment file schema](../../../planning/specs/05-tier-assignment-schema.md),
[CONVENTIONS § Repository layout](../../../planning/CONVENTIONS.md),
[SCOPE § Report output contract](../../../planning/SCOPE.md).

## Dependencies

- Blocked by: None. `README.md` does not exist, so nothing in this epic can conflict with it.
- Blocks: None.

## PR size note

One new file, and the whole point is that it stays short — a landing page, not a manual.
Sixty lines is generous. If it runs past that the cause will be a usage or install section
this issue's Out of scope forbids, and the fix is deletion rather than a split.
