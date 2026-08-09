---
type: Decision
title: "Invocation and output contract"
description: "How is the binary called, and what exactly does it emit on stdout, stderr and exit code?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 09
slug: report-output-contract
status: decided
verdict: "Markdown on stdout, diagnostics on stderr, distinct exit codes; writes no file of its own"
decided_via: triage
depends_on: [loop-bounds-and-termination, reviewer-selection-vocabulary]
---

# Question

This binary's only user is a skill parsing its output over Bash. The command line and
the three streams *are* the product's interface — the equivalent of
`opencode run --dir <repo> --agent plan -m <model> "<prompt>"` in today's
`review-interfaces.md`. What is it?

# Options

- **Markdown on stdout, diagnostics on stderr, meaningful exit codes** — the report is
  the entire stdout; cost, model used, turns and coverage go to stderr; `0` usable,
  `1` no reviewer available, `2` ran but unusable. Trivially consumable by a skill and
  readable by a human.
- **A JSON envelope on stdout** — `{report, model, cost, coverage}`. Machine-precise;
  every caller must now parse JSON to read prose, and a human running it by hand gets
  an unreadable blob.
- **Write the report to a file** — hand back a path. Directly contradicts
  [CONCEPT](/CONCEPT.md): the calling skill keeps sole ownership of the report file,
  and the reviewer writes nothing into the repository.

# Recommendation

**Markdown on stdout, diagnostics on stderr, meaningful exit codes.** It matches the
shape the pipeline already consumes, keeps the binary a `$(...)`-able Unix citizen, and
keeps the "writes nothing" guarantee absolute — nothing to distinguish a report file
from a repository file once the binary can create either.

The stream split does real work: `review-interfaces.md` requires the report's Scope to
name the model that looked, and requires the skill to fall back silently when the
capability is absent. Both are metadata about the run, not part of the review, and
mixing them into stdout would put them inside the text the skill quotes as leads.

Distinct exit codes for "no reviewer available" and "ran but failed" matter for the same
reason: the first is a normal, silent fallback; the second is worth a line in the
report. A single non-zero code collapses them.

Prompt and task arrive as arguments/stdin rather than a config file, so an invocation is
self-contained and reproducible from the transcript.

# Verdict

**Accepted as recommended.** Markdown report on stdout, run diagnostics on stderr,
distinct exit codes (`0` usable, `1` no reviewer available, `2` ran but unusable), and
the binary writes no files.

**Refreshed after [loop bounds](/scope/08-loop-bounds-and-termination.md) was decided.**
That verdict removed the caps from v1 and put *measurement* in their place, which
promotes stderr from a convenience to a load-bearing part of this contract: turns taken,
accumulated cost and elapsed time are emitted there, and they are the numbers that
[success criteria](/scope/14-success-criteria.md) checks and that Milestone 3 uses to
size its caps. The stream split is therefore not just tidiness — it is what keeps those
measurements out of the prose a skill quotes as leads.

Also dropped from this contract by the same verdict: the appended coverage line. There
is no bound-driven truncation left for the binary to report.

**Narrowed at the close of [Epic 1](../../epics/epic-1-walking-skeleton/EPIC_1.md).** The
"writes no files" clause above was unqualified, and Epic 1's hand run showed it cannot be:
authenticating through `kern-link` lets the dependency rewrite `~/.pi/agent/auth.json`
when a stored OAuth credential is expired, and create an `auth.json.lock` sidecar when it
reads an existing store. What this contract actually guarantees — and what the code
enforces — is that **the binary writes nothing inside the repository under review and
produces no output file**; credential storage belongs to `kern-link`. The alternative,
this binary reading credentials itself, was rejected as strictly worse for security
([drift](/DRIFT.md)).
