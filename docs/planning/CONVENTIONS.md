---
type: Conventions
title: "External Reviewer — Conventions"
description: "The personal conventions baseline filtered to Go, with eight project deviations covering enforcement of the read-only guarantee, cross-platform paths, the branch model and a frozen dependency set."
tags: [planning, conventions]
timestamp: 2026-08-11T03:38:28Z
status: final
baseline_version: 2026-07-20T13:00:00Z
---

# External Reviewer — Conventions

How code in this repository is written, named, tested, committed and reviewed. Most of
this is the standing personal baseline; the points that differ carry a link to the
decision that changed them.

**This document governs the `external-reviewer` repository only.** Milestone 3's issues
land in the **lx skills repository** instead ([SCOPE](/SCOPE.md), Milestone 3), and there
the skills repo's own authoring contract, branch model and review gate apply — what
travels with the issue is its acceptance criteria and nothing else
([decision](/conventions/04-cross-repo-conventions.md)).

## Code style & formatting

- `gofmt` is law: it runs in CI, and style is never debated in review.
- `golangci-lint`, pinned, runs in CI with `gosec`, `errcheck` and `govet` enabled —
  the safety properties here are path handling and process execution
  ([specs 17](/specs/17-distribution-and-ci.md)). Warnings are errors.
- **The write half of the filesystem API is forbidden, and CI enforces it**
  ([decision](/conventions/01-write-api-guard.md)). A `forbidigo` block rejects
  `os.Create`, `os.OpenFile`, `os.WriteFile`, `os.Remove*`, `os.Mkdir*`, `os.Rename`,
  `os.Symlink`, `os.Chmod`, `os.Chtimes`, `os.Truncate`, `os.Link`, the corresponding
  `*os.Root` methods (including `os.Root.Chmod` and `os.Root.Link`), and
  `(*os.File).Write`, `(*os.File).WriteString` and `(*os.File).Truncate` — the write
  methods reachable from a handle Epic 2's read-only tools open — plus `exec.Command` /
  `exec.CommandContext` outside the `git_read` implementation. `_test.go` files are
  exempt — fixtures are built with `t.TempDir()` and `git init`. An exception requires a
  `//nolint:forbidigo` carrying its reason. **This list is the source of truth;
  `.golangci.yml`'s `forbidigo.forbid` block mirrors it exactly, one pattern per
  operation — amend both together.**

  This is the one lint rule that is not about style. SPECS states the product's central
  guarantee as a property of the source — *"it cannot write to the repository because the
  write half of the filesystem API is not in the codebase"* — and a structural guarantee
  that nothing checks degrades into a promise by the third contributor, human or agent.
  `exec.Command` is on the list for the same reason ripgrep was rejected: a subprocess
  reads on its own authority, outside the root handle
  ([specs 11](/specs/11-tool-implementation-strategy.md)).
- No commented-out code in committed files.

## Naming

- Descriptive over short; no abbreviations that are not industry-standard.
- Go's community casing throughout — `PascalCase` for exported identifiers,
  `camelCase` for unexported. No house style.
- Files are named after the main thing they define.
- Identifiers that cross the model boundary are **not** Go-cased: tool names and their
  JSON schema fields are `snake_case` (`read_file`, `max_results`), because they are a
  wire contract with the model, not Go API ([specs 09](/specs/09-read-only-tool-contract.md)).

## Repository layout

- The top level stays small: `main.go`, `go.mod`, `docs/`, CI config. Everything else
  lives under `internal/` — the root is `package main` because
  `go install …@latest` only yields a binary of that name if it is
  ([specs 01](/specs/01-language-module-toolchain.md)).
- Knowledge-style docs (plans, references, runbooks) are OKF bundles under `docs/`.
  `README.md` is for a human landing on the repository, not a knowledge dump.

## Paths and platforms

Windows is the development machine; CI and every agent implementer run Linux. Two
vocabularies keep that from becoming a class of recurring bug
([decision](/conventions/02-cross-platform-path-rules.md)):

- **OS paths** use `path/filepath`, are native to the platform, and never cross the
  model boundary.
- **Wire paths** — tool input and output, `--allow` arguments, stderr diagnostics — are
  `path`-package, `/`-separated and repo-relative on every platform.
- Conversion happens at the boundary, with `filepath.ToSlash` / `filepath.FromSlash`.
  A test that asserts on a path asserts the wire form.
- No `//go:build` platform divergence ships without a test that runs on both OSes in the
  CI matrix. The two-OS matrix is the constraint check, not thoroughness: `os.Root`
  behaviour and path separators are exactly where the platforms split.

## Git & PRs

- Conventional commit messages: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`,
  `test:` — imperative mood, no trailing period in the subject.
- **`develop` is the integration trunk; `main` is the install surface**
  ([decision](/conventions/03-branch-model.md)). Issue branches cut from `develop` and
  merge back into it; planning and doc work lands on `develop`; `develop` merges to
  `main` at a milestone boundary. `main` is the default branch, so it is literally what
  `go install github.com/julienlegoux/external-reviewer@latest` resolves to — a
  half-finished epic on it is a broken install.
- One branch per issue, named `issue-<n>-<slug>`; branches are short-lived.
- A PR closes exactly one issue via `Closes #<n>`.
- The merge gate is green CI (lint + tests, both OSes). No second human approval is
  required.
- Neither `main` nor `develop` is ever force-pushed.

## Testing

- Strict red-green test-first — failing test, minimal pass, refactor — is mandatory
  wherever correctness is an assertable behavior. On this project that is nearly
  everything: tier and model resolution, the family classifier, confinement and
  allow-list refusals, each tool's contract, the loop's termination order, and the
  exit-code classification.
- Standard library `testing` only. No `testify`, no assertion helpers, no mocking
  framework ([specs 16](/specs/16-testing-infrastructure.md)). `go test ./... -race` is
  the command.
- **Black-box by default** ([decision](/conventions/05-test-package-layout.md)): tests
  are `package foo_test`. In-package tests are the exception, for unexported logic with
  no reachable path through the package's API, and go in a file named
  `<thing>_internal_test.go` so the exception is visible in a diff. Table-driven is the
  default shape.

  The properties this project sells are properties of the API surface — an escaping name
  is refused, an excluded family ends the run at exit `1`, a bad path in a `read_file`
  batch is reported inline while the rest return. A black-box test asserts that; an
  in-package test can accidentally assert the implementation that was meant to stay free
  to change.
- Test names state the behavior asserted, not the method called.
- Prefer integration-level tests where the real thing is cheap; mock only true externals.
  The model is the one true external, and `kern-link`'s `faux` provider is how it is
  mocked — in-process, replaying the real streaming event protocol.
- Live-model tests are gated at runtime by `os.Getenv` + `t.Skip` naming the missing
  variable, never by build tags, so `go test ./...` is green offline on a clean checkout.
  No golden files for model output.

## Error handling

- Fail fast; never swallow an error silently — handle it meaningfully or propagate it
  with context added.
- **Wrap for humans, type for classification** ([decision](/conventions/06-go-error-idiom.md)).
  Context is added with `fmt.Errorf("<what was attempted>: %w", err)`. Exit-code
  classification travels on typed or sentinel errors inspected with `errors.Is` /
  `errors.As` at the `run()` boundary — never by matching message text. Error messages
  are lowercase and unpunctuated, and state what was being attempted rather than only
  what broke.
- **A failing tool is a value; a failing turn is an error.** A tool error is returned
  into the loop as `ToolResultMessage{IsError: true}` so the reviewer can try another
  path — it is never propagated up the stack. This is the single place where the
  "never swallow" rule inverts, and getting it wrong ends reviews that should have
  continued ([specs 13](/specs/13-error-handling-and-failure-classification.md)).
- Nothing panics across the CLI boundary; `recover()` appears nowhere.
- The two audiences are stdout and stderr, and they are not interchangeable: stdout is
  the model's final text verbatim and is empty on any failure; everything the human or
  the calling script needs to understand a run goes to stderr as prefixed lines
  ([specs 12](/specs/12-output-and-diagnostics-format.md)).

## Dependencies

- The dependency set is **frozen at two**: `kern-link` and `BurntSushi/toml`
  ([decision](/conventions/07-dependency-freeze.md)). Adding any third non-stdlib import
  is a reopening of [specs 03](/specs/03-kern-link-dependency-policy.md) with the user
  deciding — not a one-line justification in a PR. Removing a dependency, or bumping a
  pinned version, is an ordinary PR.

  The supply-chain surface is part of the product: this binary reads a private repository
  and sends it to a third party on the user's own credentials.
- Prefer the standard library. No CLI framework, no test framework, no logging library,
  no config library.
- Everything the ecosystem allows to be pinned is pinned. A `replace` directive is never
  committed — it breaks `go install` for anyone whose disk does not match the author's —
  and `go.work` stays gitignored.

## Documentation & comments

- Comments exist only for constraints the code cannot show: invariants, workarounds with
  links, non-obvious *why*. Never narration of what the next line does.
- **No blanket doc-comment requirement on exported identifiers**
  ([decision](/conventions/08-doc-comment-policy.md)). Everything but `main` lives under
  `internal/`, so there is no published API and no godoc reader; applying Go's convention
  literally would produce restated names at scale. A comment written on an exported
  identifier still follows Go form, starting with the identifier's name.
- Two exceptions always apply: every `internal/` package carries a package comment
  stating what it owns, and each tool's `ai.Tool` name, description and JSON schema live
  beside its implementation — a description that drifts from its implementation tells a
  reviewer it may pass an argument that does not exist
  ([specs 08](/specs/08-system-prompt-and-task-assembly.md)).
- Every substantive doc produced during work lands in the OKF bundle under `docs/`, not
  in ad-hoc scattered markdown.
