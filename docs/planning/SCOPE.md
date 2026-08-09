---
type: Scope
title: "External Reviewer — Scope"
description: "v1 of a Go CLI that runs a read-only agent loop over a repository on a model outside the Anthropic family and returns its report to the calling Claude Code session."
tags: [planning, scope]
timestamp: 2026-08-09T01:20:00Z
status: final
---

# External Reviewer — Scope

## Problem

The lx skills pipeline wants a reviewer that did not write the thing under review — a
self-review is weakest exactly where it re-reads its own reasoning. Today that
capability arrives through `opencode` on `PATH`: a second agent harness, separately
installed and separately authenticated, just to get a second opinion.

The capability is right; the dependency is what it costs. v1 replaces it with a single
binary that runs its own loop against a model outside the Anthropic family — no second
harness, and no proxy in front of the session
([concept](/CONCEPT.md), [decision](/scope/01-problem-and-job.md)).

## Users

v1 targets **one machine: the author's** ([decision](/scope/02-target-users.md)).

The direct consumer is a skill, not a person: `review-epics` and `review-issues` invoke
the binary over Bash and parse its output. The human appears in two places — writing
the tier assignment, and running the binary by hand to judge a model before trusting
it.

`go install github.com/julienlegoux/external-reviewer@latest` is the install story.
No packaging, signed releases, install docs or config wizards: those requirements only
exist for users who do not exist yet, and widening later costs packaging, not a
redesign.

## Goals & success criteria

v1 succeeded when all three hold ([decision](/scope/14-success-criteria.md)):

1. `_shared/review-interfaces.md` no longer mentions `opencode`, and the external pass
   runs through `external-reviewer`.
2. A real `review-epics` or `review-issues` run on this project's own artifacts produced
   leads, at least some of which survived verification against the files.
3. A standard-tier review's measured cost and wall-clock time are recorded, and the
   author judges them low enough to reach for the tool again.

Explicitly **not** a criterion: that the external reviewer finds things the native
review missed. A second opinion that merely confirms is worth having, and making
novelty the bar would push the reviewer toward inventing findings.

## Non-goals (out of scope for v1)

Each carries its reasoning, so it does not get re-litigated one issue at a time
([decision](/scope/12-non-goals.md)):

- **A gateway or proxy in front of the session.** Rejected in the concept. One named
  reopening condition: foreign models used as genuine *subagents* — implementers
  writing code, not reviewers reading it.
- **Any write to the repository or to GitHub.** No edits, no commits, no comments, no
  issue or PR state. The calling skill keeps sole ownership of the report file.
- **Sharing the calling session's context.** No conversation, no history handoff, no
  memory between runs.
- **A model catalogue shipped inside the skills.** Model facts live in `kern-link`; the
  assignment lives on the machine.
- **Task types other than reviewing a repository.** No implementing, no refactoring, no
  general question-answering.
- **Onboarding for anyone but the author** — release binaries, install docs, config
  wizards.
- **A stale-catalogue signalling mechanism.** Parked by the concept, still parked.
- **Loop caps and coverage statements.** Deferred to Milestone 3, sized from measurement
  rather than guessed ([decision](/scope/08-loop-bounds-and-termination.md)).
- **Interactive UI of any kind.** No TUI, no prompts, no confirmations: the caller is a
  script.

## Constraints

Four bind, and `define-specs` must choose within them
([decision](/scope/15-constraints.md)):

- **Go.** `kern-link` (`module github.com/julienlegoux/kern-link`, `go 1.25.0`) carries
  the provider coverage, credential resolution, streaming, tool-call types and cost
  accounting this project refuses to rebuild. Building on it is a concept-level
  decision, so the language follows from it.
- **Windows is the development machine, and nothing may assume POSIX.** Path handling,
  config location and process invocation work on Windows first and still work
  elsewhere.
- **Runs on the user's own credentials, billed per call.** No service budget, no shared
  key. From Milestone 3 on, it also runs unsupervised inside a script.
- **`kern-link`'s multi-turn support is unproven here.** Its documented example covers a
  single round trip; the loop is this project's addition. Fixes land upstream in a repo
  the author also owns.

No deadline, and no compliance regime: the tool reads local repositories the user
already has open.

### Systems of record

Three systems own facts this project reads and never copies
([decision](/scope/16-systems-of-record.md)): **`kern-link`** (which models exist, their
cost, family and provider, how credentials resolve), the **lx skills repo** (the
delegation contract in `_shared/review-interfaces.md`), and the **user's machine** (the
tier assignment and the credentials).

Two integrations follow and no others: `kern-link` as a Go dependency, and the git
repository under review through read-only tools. No GitHub API, no telemetry.

### Core user journeys

The four flows v1 must support end to end
([decision](/scope/10-core-user-journeys.md)):

1. A review skill runs an external pass and gets usable leads back.
2. The capability is absent — no binary, no credential, or no model assigned to the
   requested tier — and the caller falls back to a native review, recording that no
   external pass happened.
3. The user sets up their tier assignment for the first time and can see which tiers
   resolve.
4. The user runs the binary by hand to judge a model before trusting it with a tier.

## Milestone 1: Walking skeleton

A binary that takes a repository path and a prompt, calls one hard-coded foreign model
through `kern-link`, and returns markdown. It retires the riskiest single assumption in
the project — that `kern-link` carries this workload at all.

- CLI entry point accepting a repository path and a task prompt (argument or stdin), so
  an invocation is self-contained and reproducible from a transcript.
- A single round-trip call to a foreign model through `kern-link`, using its existing
  credential resolution (env keys and OAuth) — no credential handling of its own.
- The output contract in full ([decision](/scope/09-report-output-contract.md)): the
  report as markdown on **stdout**; run diagnostics on **stderr**; exit codes `0` usable,
  `1` no reviewer available, `2` ran but unusable. The binary writes no files.
- The distinction between exit `1` and exit `2` is load-bearing: the first is a normal
  silent fallback for the caller, the second is worth a line in the report.

## Milestone 2: Read-only agentic loop

The multi-turn loop that turns a prompt-pipe into a reviewer that chooses what to read
— which is the reason the loop lives inside the binary rather than being replaced by
evidence the reviewed party pre-selected.

- Four native, in-process read-only tools
  ([decision](/scope/07-read-only-tool-set.md)): `list` (glob), `read_file` (with
  offset/limit), `search` (ripgrep-style), and `git_read` over a fixed allowlist
  (`log`, `diff`, `show`, `status`). No general shell.
- Every tool call confined to the repository root passed on the command line; traversal
  outside it refused. "Never writes, never commits, never touches GitHub state" is a
  property of the code that exists, not of a filter on a command string.
- The multi-turn loop itself, driving `kern-link`'s tool-call protocol until the model
  stops.
- **Run instrumentation, not caps** ([decision](/scope/08-loop-bounds-and-termination.md)):
  turns taken, accumulated cost and elapsed time emitted on stderr as the run
  progresses. Nobody knows yet how long a review takes, so a cap chosen now would be a
  guess that silently truncates legitimate reviews. Through this milestone the author
  launches by hand and can interrupt.
- These measurements are the deliverable that Milestone 3's caps and the third success
  criterion both depend on.

## Milestone 3: Reviewer selection and pipeline integration

What makes the pipeline call the binary instead of the author calling it. This is the
milestone that retires `opencode`, and the first in which the binary runs with no human
watching.

- **Weight tiers as the request vocabulary**
  ([decision](/scope/04-reviewer-selection-vocabulary.md)): `--tier light|standard|heavy`,
  reusing the grading `implement-epic` already applies to work. A skill asks for a kind
  of reviewer and never names a vendor.
- **The machine-local assignment** ([decision](/scope/05-catalogue-location-and-format.md)):
  one user-level TOML resolved by platform convention (`%APPDATA%` on Windows,
  `$XDG_CONFIG_HOME` then `~/.config` elsewhere), with per-invocation environment
  overrides. Never committed to a repository. TOML because the file records a human
  judgment that deserves a comment explaining it.
- **Family exclusion enforced by the binary**
  ([decision](/scope/06-family-exclusion-rule.md)): `--exclude-family anthropic` by
  default. Selection is by *family*, never by provider — `amazon-bedrock` serves
  Anthropic's models too, so a provider filter would send Claude's work back to Claude.
  A tier assigned an excluded-family model resolves to *no reviewer for that tier*, not
  an error.
- **A tier-resolution command** reporting which tiers resolve to a reachable model on
  this machine right now — without it, setup is trial and error against a model that
  silently is not there.
- **Loop caps, sized from Milestone 2's measurements.** Turns, cost and wall clock, plus
  the decision of which to expose as flags — made against real numbers.
- **The `review-interfaces.md` swap** ([decision](/scope/13-skills-integration-scope.md)):
  probe and call `external-reviewer`; remove the `opencode` branch; leave the native
  fallback unchanged and unconditional.

> **Cross-repository note for `split-epics`:** the `review-interfaces.md` work lands in
> the **lx skills repository**, not in this one. This milestone's issues carry a
> different repo than every other milestone.

## Risks & assumptions

Six risks, each tied to the milestone that retires it
([decision](/scope/18-risks-and-assumptions.md)):

1. **`kern-link`'s multi-turn tool loop is unproven.** Retired by Milestones 1–2;
   mitigated by the author owning the upstream.
2. **Foreign tool-calling fidelity varies by model.** Only Google is credentialed on this
   machine today, so v1 is validated against one family. Retired by Milestone 2;
   mitigated by the tier config making the model swappable without code changes.
3. **Context overflow on large surfaces.** An epic's merged diff is the heaviest read in
   the pipeline and nobody has measured it. **Unmitigated in v1** — the coverage
   statement that would have made a truncated read visible was dropped along with the
   caps. Milestone 2's measurements make it observable; the mitigation is deferred.
4. **Unsupervised spend.** Arrives at Milestone 3, when the pipeline invokes the binary
   with no human present — which is also when the numbers to size a cap exist.
5. **A runaway run during hand-run measurement.** The direct cost of running unbounded:
   one bad Milestone-2 invocation can spend real money before it is noticed. Accepted
   deliberately, since a cap guessed before any measurement would truncate legitimate
   reviews. Mitigated only by live stderr diagnostics and a human at the keyboard.
6. **Assumed: leads-not-findings holds.** Everything downstream of a mediocre review is
   safe *only* because the calling skill verifies each lead against the files before it
   enters a report. This is an assumption about the *caller*, and the one thing this
   project cannot enforce.

Assumed and not re-tested, both inherited from [CONCEPT](/CONCEPT.md): that a
same-family reviewer is worth avoiding, and that the difference in family is what makes
the second opinion valuable.

Parked by the concept and still parked: how a stale catalogue signals it needs
revisiting, whether `review-implementation` joins, and how a report states its own
coverage.
