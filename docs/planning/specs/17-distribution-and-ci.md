---
type: Decision
title: "Distribution & CI"
description: "How the binary ships and what runs on every push."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 17
slug: distribution-and-ci
status: decided
verdict: "go install only, no releases; one workflow with a test job (-race, ubuntu and windows) and a lint job"
decided_via: triage
depends_on: [language-module-toolchain, testing-infrastructure]
---

# Question

Scope settled distribution in one line — `go install github.com/julienlegoux/external-reviewer@latest`,
with no packaging, signed releases, install docs or config wizards, because those
requirements only exist for users who do not exist yet
([decision](/scope/02-target-users.md)). What is undecided is CI: what runs on every
push, and whether there is any release ceremony at all.

CI matters here beyond hygiene. `go.mod` names a tagged `kern-link` while the author
develops against a local `go.work` ([decision](/specs/03-kern-link-dependency-policy.md)),
and CI — which has no workspace — is the only thing that catches a commit that builds
only on the author's disk.

# Options

- **Two jobs, `test` + `lint`, on push and PR.** Mirrors `kern-link`; catches the
  workspace-drift case on every commit.
- **Test only.** Half the setup; the lint config would then exist and never run, which is
  worse than not having one.
- **Add a release workflow with cross-compiled binaries.** Precisely the packaging scope
  ruled out, for users who do not exist.

# Recommendation

**`go install` is the whole distribution story. One workflow, two jobs, no releases.**

`.github/workflows/test.yml`, on push and pull request, `go-version-file: go.mod` so the
toolchain is pinned by the same declaration the module makes
([decision](/specs/01-language-module-toolchain.md)):

- **`test`** — `go build ./...` then `go test ./... -race`. Runs on `ubuntu-latest` **and
  `windows-latest`**. The matrix is not thoroughness for its own sake: Windows is the
  development machine and nothing may assume POSIX
  ([scope constraint](/scope/15-constraints.md)), while every implementer agent and this
  project's own path handling will be exercised on Linux in CI. The two rows check
  opposite halves of that constraint, and path-separator and `os.Root` behaviour are
  where it breaks.
- **`lint`** — `golangci-lint-action`, pinned to an explicit version, against a
  `.golangci.yml` in the repo (defaults plus `gosec`, `errcheck`, `govet` — this is a
  binary whose safety properties are path handling and process execution).

No release workflow, no `CHANGELOG.md`, no version tags in v1. `@latest` resolves to the
default branch's newest commit for an untagged module, which is exactly what a
single-user tool wants. A `version` subcommand reports
`runtime/debug.ReadBuildInfo()` — VCS revision and dirty flag, stamped by the toolchain
for free — so a report can be traced to a build without any release machinery.

No Dockerfile, no goreleaser, no Homebrew tap, no signing.

**Amended 2026-08-14 (first beta):** v1 is tagged after all. `v0.1.0-beta.1` on `main`
carries a GitHub release, and the tag is the *only* thing added — no release workflow, no
cross-compiled binaries, no `CHANGELOG.md`, no signing, none of the packaging this
decision ruled out. The reason the original wording gave for staying untagged was that
`@latest` should follow the default branch; what changed is that `main` now exists as a
release branch behind `develop`, so "the newest commit on the default branch" and "the
state someone should install" are no longer the same thing, and the tag is what names the
second one. Note the mechanical consequence, since it is the sentence this amendment
invalidates: `go install …@latest` now resolves to the newest **tag**, not to the newest
commit — for a module whose only tags are pre-releases, Go's `@latest` picks the highest
pre-release. `version` still reports `runtime/debug.ReadBuildInfo()` and is still what
ties a report to a build; the tag only adds a name a human can ask for.

**Cross-repository note:** Milestone 3's `review-interfaces.md` change lands in the **lx
skills repository**, which has its own CI and none of this
([SCOPE](/SCOPE.md), [decision](/scope/13-skills-integration-scope.md)).

# Verdict

Accepted at triage as recommended. The two-OS matrix is the constraint check, not thoroughness: Windows is the development machine and Linux is where every implementer agent and CI runs, and path handling plus `os.Root` behaviour is exactly where that splits.
