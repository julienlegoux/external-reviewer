---
type: Decision
title: "Testing infrastructure"
description: "The harness implement-issue's TDD loop actually runs, and how an agent loop gets tested without a network."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 16
slug: testing-infrastructure
status: decided
verdict: "stdlib testing only; kern-links faux provider drives the loop offline; live tests gated by env plus t.Skip"
decided_via: triage
depends_on: [language-module-toolchain, agent-loop-mechanics]
---

# Question

Every issue this project produces is implemented under a strict red-green TDD loop, so
the harness is decided before the first test rather than discovered by the first
implementer. The hard part is specific: **how is a multi-turn agent loop against a
foreign model tested without a network and without spending money?** A test that needs
`GEMINI_API_KEY` is a test CI cannot run and a test that costs money to re-run is a test
nobody re-runs.

# Options

- **Stdlib `testing` only, with `kern-link`'s `faux` provider driving the loop offline.**
  No dependencies, and the loop is exercised against the same event contract the real
  providers implement.
- **Stdlib `testing` plus `testify`.** Familiar assertions, one dependency, and it
  diverges from `kern-link` — where `testify` appears exactly zero times across 102 test
  files.
- **Mock `ai.Models` with a hand-written fake.** Full control, and it is a fake of an
  interface whose real implementations may drift from it, which is how a green suite
  starts lying.

# Recommendation

**Stdlib `testing` only. `kern-link`'s `ai/providers/faux` is the loop's test double.**

`faux` is an in-process provider that scripts canned responses and replays the full
streaming event protocol — `kern-link`'s own docs call it "the executable specification
of the event contract", and its package examples run against it. Registering it in a
`MutableModels` and scripting a tool-call sequence tests the real loop, the real tool
dispatch, the real cost accounting and the real stop conditions, offline and free. That
is the single highest-value thing this dependency provides beyond the API itself, and it
retires most of risk 1 ([scope](/scope/18-risks-and-assumptions.md)) at zero cost.

The rest, mirroring `kern-link`'s own posture:

- **Tests co-located with the code**, `_test.go` beside the source, table-driven where
  the case list is data (the family classifier of
  [decision 04](/specs/04-model-family-classification.md) is exactly that shape, tested
  against the whole embedded catalog).
- **`t.TempDir()` for repository fixtures.** Confinement, deny-lists and traversal
  refusals are tested against real directories — including symlink escapes, where
  supported by the platform.
- **`run()` takes its streams as parameters** and returns an exit code
  ([decision](/specs/01-language-module-toolchain.md)), so the whole CLI — argv parsing,
  exit codes, the stdout/stderr split of
  [decision 12](/specs/12-output-and-diagnostics-format.md) — is testable in-process
  without spawning anything. This is `cmd/pi-ai`'s pattern and the reason it is testable.
- **`git_read` tests build a throwaway repo** with `git init` in a `t.TempDir()` and
  `t.Skip` when `git` is absent.
- **Live model tests are gated at runtime, not by build tag** — `os.Getenv` then `t.Skip`
  with a message naming the missing variable, matching `kern-link` exactly. CI never sets
  those secrets, so they always skip there, and `go test ./...` is green offline on a
  clean checkout.
- **No golden files for model output.** A report's text is not deterministic and
  asserting on it produces a suite that fails for the wrong reasons.

`go test ./... -race` is the command. `-race` needs a C toolchain; when the machine lacks
one, `go test ./...` locally and CI enforces the race detector on every push — the same
split `kern-link`'s README documents.

# Verdict

Accepted at triage as recommended. `faux` replaying the full event protocol in-process is what makes a multi-turn agent loop testable without a network or a bill.
