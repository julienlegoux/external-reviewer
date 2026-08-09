---
type: Issue
title: "Scaffold the Go module, the run() seam and CI"
description: "Create the module, the thin main.go over an in-process Run() seam, and the CI workflow whose lint job forbids the write half of the filesystem API."
tags: [epic-1]
timestamp: 2026-08-09T09:58:55Z
epic: 1
issue: 01
slug: scaffold-module-and-ci
size: S
status: pr-open
gh_issue: 4
gh_pr: 16
resource: https://github.com/julienlegoux/external-reviewer/issues/4
depends_on: []
---

# Scaffold the Go module, the run() seam and CI

## Summary

Nothing else in this epic can be written until the repository is a Go module with the
seam that makes the CLI testable in-process and the CI that enforces the project's one
structural guarantee. This PR is that foundation: `go.mod`, a two-line `main.go` over
`Run(argv, stdin, stdout, stderr) int`, and a workflow whose `lint` job rejects the write
half of the filesystem API so the read-only guarantee is a property of the source rather
than a promise.

## Scope

- `go.mod`: module `github.com/julienlegoux/external-reviewer`, `go 1.26`. The
  `kern-link` require line lands with its first real import (issue 04) — adding it now
  would be removed again by `go mod tidy`.
- `main.go` at the module root, `package main`, whose entire body is
  `os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))`.
- `internal/cli` owning
  `func Run(argv []string, stdin io.Reader, stdout, stderr io.Writer) int` plus its
  package comment. SPECS writes this seam as `run(...)`; it is exported here only
  because it lives one package down, and `internal/` keeps it unpublished either way.
- `.gitignore` covering `go.work`, `go.work.sum` and the built binary.
- `.golangci.yml`, pinned, enabling `gosec`, `errcheck` and `govet`, with a `forbidigo`
  block rejecting `os.Create`, `os.OpenFile`, `os.WriteFile`, `os.Remove*`, `os.Mkdir*`,
  `os.Rename`, `os.Symlink`, the corresponding `*os.Root` write methods, and
  `exec.Command` / `exec.CommandContext`. `_test.go` files are exempt
  (`tests-exclude` / equivalent), since fixtures are built with `t.TempDir()`.
- `.github/workflows/ci.yml` on push and pull request: a `test` job
  (`go build ./...`, `go test ./... -race`) on a `[ubuntu-latest, windows-latest]`
  matrix with `go-version-file: go.mod`, and a `lint` job running the pinned
  `golangci-lint`.

## Out of scope

- Any argument parsing beyond "no arguments is a usage error" — the `review` grammar is
  issue 02.
- The `kern-link` dependency, model resolution and any network call — issues 04 and 05.
- A `README.md`. The repository's documentation lives in the `docs/` OKF bundle; a human
  landing page is not part of this epic.

## Acceptance criteria / Definition of done

- [ ] `go build ./...` succeeds on a clean checkout with no network access beyond the
      module proxy.
- [ ] `go test ./... -race` passes offline on a clean checkout.
- [ ] `internal/cli/run_test.go` (`package cli_test`) asserts, through `Run`, that an
      empty `argv` returns exit code `2`, writes nothing to stdout, and writes a
      non-empty usage message to stderr. Written failing first, per the strict red-green
      rule in CONVENTIONS.
- [ ] `main.go` contains no logic beyond the `os.Exit(cli.Run(...))` call, so every
      behavior in this epic is reachable from a test.
- [ ] `golangci-lint run` passes on the committed tree.
- [ ] The forbidigo guard is demonstrated to work: the PR description records that a
      scratch file calling `os.WriteFile` made `golangci-lint run` exit non-zero with a
      forbidigo message naming the rule. The scratch file is not committed.
- [ ] Both CI jobs run and are green on the PR, on ubuntu-latest and windows-latest.
- [ ] `gofmt` clean; commit messages follow the conventional prefixes in CONVENTIONS.

## Relevant files / areas

No existing code for this yet — the repository currently contains only `docs/` and
`.git`. The paths below follow the layout SPECS fixes (`package main` at the root,
everything else under `internal/`), not verified against existing code:

- `go.mod`, `main.go`, `.gitignore`
- `internal/cli/run.go`, `internal/cli/run_test.go`
- `.golangci.yml`, `.github/workflows/ci.yml`

Governing decisions: [SPECS § Stack](../../../planning/SPECS.md),
[SPECS § Distribution & operations](../../../planning/SPECS.md),
[CONVENTIONS § Code style & formatting](../../../planning/CONVENTIONS.md) (the write-API
guard), [CONVENTIONS § Repository layout](../../../planning/CONVENTIONS.md).

## Dependencies

- Blocked by: none.
- Blocks: [02 — Parse the review invocation](/epic-1-walking-skeleton/issues/02-review-invocation-parsing.md),
  and transitively everything else in this epic.

## PR size note

Sized **S**: target ~150 changed lines — a `go.mod`, an eight-line `main.go`, the `Run`
seam, two YAML files and one test. Most of the work is deciding the `forbidigo` list, not
typing it. If this passes ~300, scope has leaked in from issue 02.
