---
type: Drift
title: "External Reviewer — Drift"
description: "Standards this codebase has drifted from, and why"
tags: [planning, drift]
timestamp: 2026-08-09T14:30:00Z
---

# Drift

Standards [SPECS](/SPECS.md), [SCOPE](/SCOPE.md) or [CONVENTIONS](/CONVENTIONS.md)
already decided, that the implementation could not follow — with the verified reason and
what was decided about it. Read this alongside those three documents: a decided standard
plus its live drift is what the code actually looks like.

Append-only, newest epic first. An entry that stops being true becomes
`resolved (<date>)` rather than disappearing.

## Epic 1: Walking skeleton

### 04 — `kern-link` is pinned at v0.1.1, not the specified v0.2.0

- **Decided**: [SPECS § Stack](/SPECS.md)'s dependency table pins
  `github.com/julienlegoux/kern-link` at **v0.2.0**, and
  [CONVENTIONS § Dependencies](/CONVENTIONS.md) requires everything pinnable to be
  pinned. Issue 04's Scope repeats the version verbatim.
- **Actual**: `go.mod` requires `github.com/julienlegoux/kern-link v0.1.1`. No `replace`
  directive is committed and `go.work` stays gitignored, so the rest of the dependency
  policy holds. The embedded model catalog is v0.1.1's, which is why the hard-coded
  reviewer is `openai-codex/gpt-5.5` rather than SPECS' illustrative `gpt-5.1-codex`.
- **Because**: the tag does not exist. `go get …@v0.2.0` fails with
  `invalid version: unknown revision v0.2.0`; `go list -m -versions` returns
  `v0.1.0 v0.1.1`; `gh api repos/julienlegoux/kern-link/tags` agrees. A forward pin to a
  release that has not been cut, not a yanked version. Every symbol the resolution
  sequence needs is present in v0.1.1 under the exact names SPECS uses — there is **no
  API-name drift**, only a version one.
- **Disposition**: accepted — [SPECS § Stack](/SPECS.md) amended to v0.1.1 on
  2026-08-09, so the table stops naming a version that cannot be installed.
- **Revisit when**: `kern-link` v0.2.0 (or later) is tagged. From here that is an
  ordinary version-bump PR, which CONVENTIONS already classes as ordinary — not the
  discharge of drift.
- **Evidence**:
  [drift record 04](../epics/epic-1-walking-skeleton/drift/04-kern-link-version-pin.md),
  PR #19.

### 05 — "the binary writes no files" excludes `kern-link`'s credential store

- **Decided**: [SCOPE § Report output contract](/SCOPE.md) and
  [SPECS § Interfaces](/SPECS.md) stated it without qualification — **the binary writes
  no files** — so the calling skill keeps sole ownership of the report.
  [CONVENTIONS § Code style & formatting](/CONVENTIONS.md) enforces it structurally: the
  write half of the filesystem API is absent from the codebase and `.golangci.yml`'s
  `forbidigo` block rejects it.
- **Actual**: both halves of the enforcement hold — no write-API call appears in
  non-test source, and a full successful review leaves the reviewed repository
  byte-identical (`TestRun_SuccessfulReview_TouchesNoFile`). The dependency nonetheless
  writes on the binary's behalf, outside the repository, in `~/.pi/agent/`:
  `auth.json` is rewritten when a stored OAuth credential is expired
  (`models.GetAuth` → `resolveStoredOAuth` → `credentials.Modify` → `os.WriteFile`), and
  `auth.json.lock` is created whenever an existing store is read.
- **Because**: observed on the manual verification run recorded in PR #20, on a machine
  whose `openai-codex` credential is OAuth — the repository tree was identical before and
  after, while `~/.pi/agent/auth.json` moved from 2621 to 2305 bytes with its mtime
  updated. The path is unreachable from the test suite, which injects an offline registry
  over an in-memory credential store. No wording keeps the literal claim true while the
  binary still authenticates: the alternative is for this binary to read the credential
  file itself, which contradicts [SPECS § Data, configuration & auth](/SPECS.md) and is
  strictly worse for security.
- **Disposition**: accepted — [SCOPE](/SCOPE.md),
  [scope/09](/scope/09-report-output-contract.md) and [SPECS](/SPECS.md) amended on
  2026-08-09 to state what is actually guaranteed: the binary writes nothing inside the
  repository under review and produces no output file; credential storage belongs to
  `kern-link` and lives in `~/.pi/agent/`.
- **Revisit when**: Epic 2's `os.Root` confinement lands — it makes the in-repository
  half structural for reads as well as writes, at which point the amended sentence should
  be re-checked against what the code enforces.
- **Evidence**:
  [drift record 05](../epics/epic-1-walking-skeleton/drift/05-credential-store-writes.md),
  PR #20.
