---
type: Issue
title: "Enforce gofmt in CI, normalise line endings, and lint on both OSes"
description: "Add the golangci-lint v2 formatters block, .gitattributes for LF normalisation, and put the lint job on the two-OS matrix, so nothing unformatted can land after this."
tags: [epic-0]
timestamp: 2026-08-11T03:33:08Z
epic: 0
issue: 04
slug: ci-formatting-gate
size: S
status: open
gh_issue: 26
resource: https://github.com/julienlegoux/external-reviewer/issues/26
depends_on: [2]
---

# Enforce gofmt in CI, normalise line endings, and lint on both OSes

## Summary

CONVENTIONS opens with *"`gofmt` is law: it runs in CI, and style is never debated in
review."* It does not run in CI. `golangci-lint` **v2 moved formatters out of `linters`** —
`gofmt` runs only when listed under a top-level `formatters:` block, which `.golangci.yml`
does not have — and the `test` job runs `go build` and `go test` and nothing else. Verified
empirically: a deliberately malformed file run against the repository's own config reports
**0 issues**, while `golangci-lint fmt --diff` reports it. The enabled linters are
`errcheck forbidigo gosec govet ineffassign staticcheck unused`, with no formatter among
them.

This lands first among the code repairs because it protects every PR after it, and the
epic's remaining issues are the largest test churn the repository has seen.

`.gitattributes` ships in the same PR, not a follow-up: the development machine has
`core.autocrlf=true`, so the working tree is CRLF while the index is LF, and a gate added
without `* text=auto eol=lf` reports all 16 files as false positives on the first local
run.

The `lint` job joins `test` on the two-OS matrix in the same pass. It is harmless today
— Epic 1's rules are platform-independent — and stops being harmless the moment Epic 2
ships `os.Root` code whose `gosec`/`govet` findings are GOOS-conditional.

## Scope

- A top-level `formatters:` block in `.golangci.yml` enabling `gofmt`, per golangci-lint
  v2's schema (the pinned version is v2.12.2, `.github/workflows/ci.yml:36`).
- `.gitattributes` at the repository root with `* text=auto eol=lf`, and the tree
  renormalised (`git add --renormalize .`). The index is already LF, so this is expected
  to stage nothing; if it does stage something, that renormalisation is part of this PR.
- `.github/workflows/ci.yml:25-26` — the `lint` job gains the same
  `[ubuntu-latest, windows-latest]` matrix the `test` job already carries.

## Out of scope

- The `forbidigo` enumeration — [issue 02](/epic-0-skeleton-hardening/issues/02-write-api-guard-coverage.md),
  which edits the same file first.
- Enabling any linter beyond a formatter. The enabled set is a SPECS decision, not this
  issue's.
- Reformatting Go source: it is currently `gofmt`-clean when checked against the committed
  blobs rather than the CRLF working tree.
- `golangci-lint fmt` as a fixer in CI — the gate fails, it does not rewrite.

## Acceptance criteria / Definition of done

- [ ] A deliberately unformatted Go file (e.g. `func   BadlyFormatted( ) int {` with mixed
      indentation) makes `golangci-lint run` **fail**. The PR body records the command and
      its output; the file itself is not committed.
- [ ] `golangci-lint run` over the repository as it stands reports zero findings — the
      gate is green on the current tree, not a red CI left for the next PR.
- [ ] `.gitattributes` exists at the repository root with `* text=auto eol=lf`, and
      `git add --renormalize .` produces no staged change once this PR is applied.
- [ ] `.github/workflows/ci.yml`'s `lint` job runs on ubuntu-latest **and**
      windows-latest, and both are green on the PR.
- [ ] No Go source file's committed content changes in this PR — the diff is
      `.golangci.yml`, `.gitattributes` and `ci.yml` only.

## Relevant files / areas

- `.golangci.yml` (new top-level `formatters:` block; the file currently has `version`,
  `linters` and `issues` sections only)
- `.gitattributes` (new, repository root)
- `.github/workflows/ci.yml:25-36` (the `lint` job)
- `docs/planning/CONVENTIONS.md` § Code style & formatting L25 — the rule this makes true;
  `docs/planning/CONVENTIONS.md` § Git & PRs L93-94 — "lint + tests, both OSes"

Source finding: [report 2](../../../REPORT_2.md) § "`gofmt` is enforced by nothing", plus
its P3 rows on the single-OS `lint` job and the missing `.gitattributes`.

## Dependencies

- Blocked by: [02 — Widen the write-API guard](/epic-0-skeleton-hardening/issues/02-write-api-guard-coverage.md)
  — both edit `.golangci.yml`, and serialising them keeps the two edits from colliding.
- Blocks: [05 — Shared `faux` harness](/epic-0-skeleton-hardening/issues/05-shared-faux-harness.md)
  and everything after it, by intent rather than by compilation: the gate protects the
  epic's remaining PRs.

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
Sized **S**: roughly 40 changed lines across three config files, assuming the
renormalisation is the no-op the LF index predicts.
