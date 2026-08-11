---
type: Drift
title: "External Reviewer — Drift"
description: "Standards this codebase has drifted from, and why"
tags: [planning, drift]
timestamp: 2026-08-11T18:42:03Z
---

# Drift

Standards [SPECS](/SPECS.md), [SCOPE](/SCOPE.md) or [CONVENTIONS](/CONVENTIONS.md)
already decided, that the implementation could not follow — with the verified reason and
what was decided about it. Read this alongside those three documents: a decided standard
plus its live drift is what the code actually looks like.

Append-only, newest epic first. An entry that stops being true becomes
`resolved (<date>)` rather than disappearing.

## Epic 0: Skeleton hardening

### 09, 10, 11 — the decided test command does not run on the development machine — resolved (2026-08-11)

- **Decided**: [CONVENTIONS § Testing](/CONVENTIONS.md) names the command without
  qualification — *"`go test ./... -race` is the command"* — resting on
  [specs 16](/specs/16-testing-infrastructure.md)'s standard-library-only testing
  stack. Strict red-green in the same section assumes the author can watch a test go
  red and green locally.
- **Actual**: the suite is cross-built `GOOS=linux GOARCH=amd64` on the Windows host
  and run under `wsl -d docker-desktop`, **without** `-race`, and CI is treated as the
  authority for both `-race` and `windows-latest`. Two acceptance criteria that need a
  real signal — issue 09's `SIGINT`, issue 10's `SIGPIPE` — were verified by hand the
  same way, against a Linux binary driven through a fifo, rather than by the suite.
- **Because**: two independent blockers, both verified during Epic 0. `go test` cannot
  execute its own temp binaries natively — a local Windows Application Control policy
  blocks them — and `-race` cannot be built at all, because it requires cgo and neither
  the Windows toolchain nor the ~136 MB `docker-desktop` distro has a C compiler. The
  cost is visible: issue 11 records four findings as *"argued rather than observed"*,
  among them all behaviour under `-race` and on `windows-latest`.
- **Disposition**: fix-now ([#48](https://github.com/julienlegoux/external-reviewer/issues/48))
  — a C toolchain on the host with the policy exempted, or a full WSL distro with
  `build-essential`. Epic 2 lands the multi-turn loop, which is the concurrency the
  race detector exists for; finding a data race in CI on an OS the author cannot
  reproduce on is the expensive version of this.
- **Resolved (2026-08-11)** by [#48](https://github.com/julienlegoux/external-reviewer/issues/48),
  through neither branch of the disposition above: the author's own Linux VPS already had
  gcc and a Go toolchain, so `scripts/test-remote.sh` syncs the working tree there over
  ssh and runs the command, and nothing was installed on the Windows host at all. The
  suite now completes with `-race` from the development machine, and the detector was
  observed firing — a deliberately racy probe test reported `WARNING: DATA RACE` and
  exit `1` — rather than assumed live. [CONVENTIONS § Testing](/CONVENTIONS.md) now says
  where the command runs.
- **Residual, deliberately not covered**: the Windows Application Control policy is
  untouched, so `go test` still cannot execute its own binaries natively on the host and
  `windows-latest` behaviour remains observable only in CI. That half needs administrator
  rights on the machine, which is why it outlived this entry; it is the smaller half,
  since the two-OS split this project actually cares about — `os.Root` semantics and path
  separators — is asserted by tests the CI matrix runs on both.
- **Evidence**: PR #45 (issue 11's verification log — the constraint stated in full, and
  the four findings it left argued rather than observed), PR #46 (issue 09's hand-run
  SIGINT verification, the WSL cross-build in practice), PR #42 (issue 10's SIGPIPE
  criterion, verified the same way). Epic 0's folder was retired at its close, so these
  cite the PRs the evidence merged in rather than the files.

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

### 05 — the no-files check compares file mtimes, not directory mtimes

- **Decided**: issue 05's criterion
  [*"the tree's file set **and modification times** are unchanged"*](../epics/epic-1-walking-skeleton/issues/05-single-turn-model-call.md),
  the assertable half of [SCOPE](/SCOPE.md)'s and [SPECS § Interfaces](/SPECS.md)'s
  guarantee that the binary writes nothing inside the repository under review.
- **Actual**: `TestRun_SuccessfulReview_TouchesNoFile`
  (`internal/cli/roundtrip_test.go:276-299`) walks the tree twice and compares the file
  set and every **file** mtime. Directory mtimes are collected and deliberately not
  compared, which the test documents in place.
- **Because**: on Windows two consecutive walks of an untouched tree already report moved
  directory mtimes, so asserting on them makes the test flaky against the platform rather
  than against the binary — re-verified during PR #20. The narrowing costs nothing the
  criterion was protecting: a file created, modified or deleted inside the repository
  changes the file set or a file mtime, and a directory whose only change is its own mtime
  contains no such file.
- **Disposition**: accepted — issue 05's criterion amended on 2026-08-10 to state what is
  actually asserted, so the narrowing stops reading as an unmet criterion.
- **Revisit when**: Epic 2's `os.Root` confinement lands. It makes the in-repository
  guarantee structural for reads as well as writes, at which point what this test still
  needs to cover should be re-derived rather than inherited.
- **Evidence**: [implementation review report 2](../REPORT_2.md) § P3, which is where the
  narrowing was first noticed as undocumented; `internal/cli/roundtrip_test.go:276-299`;
  PR #20.
