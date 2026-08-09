---
type: Issue
title: "Confine the run with --allow and os.OpenRoot"
description: "Add the required repeatable --allow flag and the single *os.Root every later read goes through, with allow-list membership, the sensitive-file floor and refusals that name the rule that refused."
tags: [epic-2]
timestamp: 2026-08-09T06:32:00Z
epic: 2
issue: 01
slug: allow-list-and-root-confinement
size: M
status: open
gh_issue: 9
resource: https://github.com/julienlegoux/external-reviewer/issues/9
depends_on: []
---

# Confine the run with --allow and os.OpenRoot

## Summary

Every tool in this epic reads through one boundary, so the boundary is built before any
of them. This PR adds `--allow`, required and repeatable, and opens the single
`*os.Root` that every later read goes through — the mechanism that makes "cannot read
outside the granted paths" a kernel refusal rather than a string comparison. It also
lands the non-configurable sensitive-file floor and the refusal vocabulary the model
sees, so "outside the allow-list" and "denied filename" never read the same.

No tool exists yet after this PR. What exists is a resolver the four tool PRs call and
cannot bypass.

## Scope

- `--allow <relpath>` on the `review` `FlagSet` from issue 02 of Epic 1: repeatable
  (`flag.Var` over a slice type), repo-relative, `/`-separated wire paths. Zero
  occurrences is a **usage error at exit `2`** with an empty stdout — never an implicit
  run over the whole tree. `--allow .` grants the entire repository and has to be said.
- `os.OpenRoot(repoPath)` once, at startup, closed on every termination path. The
  `*os.Root` and the resolved allow-list travel together as one value the tools take.
- A resolution helper — the only way a wire path becomes a readable file — that in order:
  rejects the path if it escapes the root (delegated to `*os.Root`, not re-implemented),
  rejects it if it is not inside the union of allowed subtrees, rejects it if it matches
  the floor, and otherwise opens it through the root.
- The floor, non-configurable and hard-coded: `.env*`, `*.pem`, `*.key`, `id_rsa*`,
  `*.p12`, `*.pfx`, `.npmrc`, `.netrc`, `credentials*`, and anything under `.git/`.
  It is a floor **inside** allowed subtrees, not the boundary — the allow-list is.
- Refusal messages name which rule refused, in three distinguishable forms: outside the
  root, outside the allow-list, denied filename. Lowercase, unpunctuated, stating what
  was attempted, per CONVENTIONS § Error handling.
- Wire/OS path conversion at this boundary and nowhere else — `filepath.FromSlash` in,
  `filepath.ToSlash` out — so no path a tool emits depends on which OS ran the binary.
- An `--allow` value that does not resolve to an existing directory or file inside the
  root is a usage error at exit `2`, reported before any model request is sent.

## Out of scope

- All four tools — issues 02, 04, 05 and 06. This PR ships the resolver they call, and
  its tests drive it directly.
- `git_read`'s confinement (pathspecs, `show <rev>:<path>` validation) — issue 06, which
  reuses this PR's allow-list value rather than re-deriving one.
- Repository enumeration and `.gitignore` semantics — issue 02.
- The loop, tool dispatch and bounds — issue 03.

## Acceptance criteria / Definition of done

- [ ] Written test-first and black-box (`package … _test`), per CONVENTIONS § Testing;
      table-driven where the cases are a list.
- [ ] `review --prompt "x" <tempdir>` with **no** `--allow` returns exit `2`, stdout
      empty, and a stderr line naming `--allow` as missing.
- [ ] `review --allow docs --allow internal --prompt "x" <tempdir>` parses both values
      and reaches the same point in `Run` that Epic 1 issue 02's success case reaches.
- [ ] Against a `t.TempDir()` repository, resolution **refuses** each of: `../outside`,
      `docs/../../outside`, an absolute path (`/etc/passwd` and `C:\Windows\win.ini`),
      and a path inside the root but outside every `--allow` subtree. Each refusal's
      message identifies which of the three rules refused.
- [ ] A symlink inside the root pointing outside it is refused, and an absolute symlink
      is refused. Creating the symlink is attempted with `os.Symlink` in the test and the
      case is `t.Skip`ped with a reason when the platform refuses to create it — never
      guarded by a build tag.
- [ ] On Windows, a Windows reserved device name (`NUL`, `COM1`) is refused. Asserted by
      a test that runs on both matrix OSes and asserts refusal on both, rather than a
      `//go:build windows` file (CONVENTIONS § Paths and platforms).
- [ ] With `--allow .`, each of `.env`, `.env.local`, `server.pem`, `id_rsa`,
      `deploy.key`, `.npmrc`, `credentials.json` and `.git/config` is refused with the
      **denied-filename** message, not the allow-list one.
- [ ] A path inside an allowed subtree that matches nothing on the floor resolves and its
      contents are readable through the root.
- [ ] Every path in a refusal message and in a successful result is `/`-separated and
      repo-relative on both OSes — asserted on the wire form, not the OS form.
- [ ] `golangci-lint run` passes with no new `//nolint` directives; only read-half
      filesystem calls are introduced, so the issue-01 forbidigo guard stays clean.
- [ ] CI green on ubuntu-latest and windows-latest.

## Relevant files / areas

Epic 1 lands `main.go`, `internal/cli/` and the diagnostics/exit-code plumbing; this PR
adds a package beside them. No existing code for this epic yet — paths below follow the
layout SPECS fixes (`package main` at the root, everything else under `internal/`), not
verified against existing code:

- `internal/confine/root.go`, `internal/confine/allow.go`, `internal/confine/floor.go`
- `internal/confine/root_test.go`, `internal/confine/allow_test.go`
  (`package confine_test`)
- `internal/cli/run.go` — the `--allow` flag and the startup `os.OpenRoot`

Governing decisions:
[SPECS § Reading the repository](../../../planning/SPECS.md),
[specs 10 — Repository confinement](../../../planning/specs/10-repository-confinement.md),
[specs 14 — Security and data exposure](../../../planning/specs/14-security-and-data-exposure.md),
[CONVENTIONS § Paths and platforms](../../../planning/CONVENTIONS.md),
[CONVENTIONS § Error handling](../../../planning/CONVENTIONS.md).

## Dependencies

- Blocked by: Epic 1 issues 02 (the `review` grammar this flag joins) and 03 (the
  exit-code classification a usage error returns through) — see
  [Epic 1](/epic-1-walking-skeleton/EPIC_1.md).
- Blocks: every other issue in this epic.

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
