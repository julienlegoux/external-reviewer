---
type: Issue
title: "Widen the write-API guard to every write operation CONVENTIONS names"
description: "Amend CONVENTIONS' forbidden write-API enumeration and mirror it in .golangci.yml so the eight operations that pass the repo's own config today are rejected."
tags: [epic-0]
timestamp: 2026-08-11T03:33:08Z
epic: 0
issue: 02
slug: write-api-guard-coverage
size: S
status: open
gh_issue: 24
resource: https://github.com/julienlegoux/external-reviewer/issues/24
depends_on: []
---

# Widen the write-API guard to every write operation CONVENTIONS names

## Summary

`.golangci.yml` matches CONVENTIONS' enumeration exactly — 20/20 patterns verified firing,
the `_test.go` exemption verified — and the enumeration itself is the gap. Eight write
operations outside it pass the repository's own config with zero findings: `os.Chmod`,
`os.Chtimes`, `os.Truncate`, `os.Link`, `os.Root.Chmod`, and the material ones,
`(*os.File).Write`, `(*os.File).WriteString` and `(*os.File).Truncate`.

That last group is why this lands before Epic 2 rather than after. `os.Create` and
`os.OpenFile` are blocked, but Epic 2 opens files for reading through `os.Root.Open`, and
nothing then stops a `.Write` on the returned handle. SPECS states the product's central
guarantee as a property of the source — *"it cannot write to the repository because the
write half of the filesystem API is not in the codebase"* — and that is currently true by
author discipline rather than by the guard.

Decided at triage: **CONVENTIONS is the source of truth for the list and `.golangci.yml`
mirrors it.** That is how every other convention in this repository is enforced, and it
makes the criterion checkable rather than aspirational.

## Scope

- `docs/planning/CONVENTIONS.md` § Code style & formatting's enumeration (L30-35) gains
  `os.Chmod`, `os.Chtimes`, `os.Truncate`, `os.Link`, `os.Root.Chmod`, `(*os.File).Write`,
  `(*os.File).WriteString`, `(*os.File).Truncate` — and `os.Root.Link`, which
  `.golangci.yml:54` already forbids while CONVENTIONS does not name it.
- `.golangci.yml`'s `forbidigo.forbid` list mirrors the amended enumeration exactly: one
  pattern per operation, the same `msg` the existing entries carry.
- The method patterns are verified to actually fire rather than assumed: `analyze-types:
  true` is already set (`.golangci.yml:16`), but the receiver form `forbidigo` expects for
  a method on `*os.File` is not the same shape as the package-function patterns already in
  the file. The implementer confirms the pattern syntax against a probe before landing it.
- A sentence in CONVENTIONS naming itself the source of truth and the config the mirror,
  so the next person amending one knows to amend the other.

## Out of scope

- The `exec.Command` / `exec.CommandContext` entries and the `git_read` carve-out —
  unchanged.
- The `_test.go` exemption (`.golangci.yml:60-66`) — unchanged; fixtures are built with
  `t.TempDir()` and `git init`.
- The `formatters:` block, the `lint` job's OS matrix and `.gitattributes` — issue 04,
  which edits the same file next.
- Epic 2's read-half `os.Root` calls, which this guard must not catch.

## Acceptance criteria / Definition of done

- [ ] Every operation named in CONVENTIONS § Code style & formatting is rejected by the
      repository's own `.golangci.yml`, verified by running `golangci-lint run` over a
      probe file that calls each one. The PR body records the command and one finding per
      operation, including `(*os.File).Write`, `(*os.File).WriteString`,
      `(*os.File).Truncate`, `os.Chmod`, `os.Chtimes`, `os.Truncate`, `os.Link` and
      `os.Root.Chmod`.
- [ ] The probe file is **not** committed anywhere CI lints — a committed probe would fail
      the repository's own gate. If it is committed at all it goes under `testdata/`
      (which `golangci-lint` skips) with a comment saying how to run the probe by hand.
- [ ] CONVENTIONS' list and `.golangci.yml`'s `forbid:` list name the same operations,
      with no entry in one and not the other — `os.Root.Link` included.
- [ ] `golangci-lint run` over the repository as it stands reports zero findings: no
      existing non-test call is caught, and no `//nolint:forbidigo` is added.
- [ ] CI green on ubuntu-latest and windows-latest.

## Relevant files / areas

- `.golangci.yml:15-59` (the `forbidigo.forbid` list), `:60-66` (the `_test.go` exclusion)
- `docs/planning/CONVENTIONS.md` § Code style & formatting, L30-43
- `.github/workflows/ci.yml:34-36` (the pinned `golangci-lint` version the probe is run
  against, v2.12.2)

Governing decisions:
[conventions 01 — the write-API guard](../../../planning/conventions/01-write-api-guard.md),
[SPECS § Reading the repository](../../../planning/SPECS.md). Source finding:
[report 2](../../../REPORT_2.md) § "The write-API guard is under-inclusive against its own
stated purpose" and Open questions 3.

## Dependencies

- Blocked by: none.
- Blocks: [04 — CI formatting gate](/epic-0-skeleton-hardening/issues/04-ci-formatting-gate.md),
  which edits the same `.golangci.yml`.

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
Sized **S**: nine patterns, a CONVENTIONS paragraph, and a probe run recorded in the PR
body — roughly 60 changed lines.
