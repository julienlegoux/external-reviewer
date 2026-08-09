---
type: Decision
title: "Output & diagnostics format"
description: "What exactly lands on stdout and stderr, and in what shape."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 12
slug: output-and-diagnostics-format
status: decided
verdict: "stdout is the final assistant text verbatim and empty on failure; stderr is prefixed human-readable lines"
decided_via: triage
depends_on: [cli-surface-and-argv]
---

# Question

Scope fixed the channels and the exit codes: report as markdown on **stdout**, run
diagnostics on **stderr**, exit `0` usable / `1` no reviewer available / `2` ran but
unusable, and the binary writes no files ([decision](/scope/09-report-output-contract.md)).
What is open is the *shape* of each stream — and stdout in particular is parsed by a
skill, so its shape is a contract with the skills repository.

# Options

- **stdout: the model's final message verbatim. stderr: human-readable lines.** Nothing
  between the reviewer and the caller; diagnostics stay for a human.
- **stdout: the report wrapped in a binary-generated envelope** (run metadata header,
  cost footer). Useful provenance, and it means the caller must strip a wrapper before
  the text is quotable in a report.
- **stderr as NDJSON.** Machine-parseable diagnostics, for a consumer that does not exist
  — scope's caller reads stdout and the exit code, nothing else.

# Recommendation

**stdout is the model's final assistant text, verbatim, and nothing else. stderr is
human-readable prefixed lines.**

Stdout carries no header, no footer, no cost line, no separator. The scope decision that
"the binary writes no files" exists so the calling skill keeps sole ownership of the
report ([concept](/CONCEPT.md)); an envelope on stdout would put the binary back in the
business of formatting someone else's document. Anything the caller needs about the run
belongs on stderr or in the exit code. On exit `1` and `2`, **stdout is empty** — a
caller must never have to distinguish a report from an error message by reading it.

Stderr, one line per event, prefixed and stable enough to eyeball during an unbounded
run:

```
turn 3  tools=2  in=48210 out=1104  $0.0231  42.8s
tool    read_file docs/epics/epic-2/EPIC_2.md:1-2000
warn    config: unrecognised key "models" in [tiers.standard]
error   turn 4 failed: rate limit (provider google)
done    turns=7  $0.0918  2m14s  stop=end_turn
```

The `done` line is the one that matters beyond debugging: turns, total cost, wall clock.
Milestone 2 exists to produce those numbers, Milestone 3's caps are sized from them, and
success criterion 3 is the author reading them and judging the tool cheap enough to reach
for again ([decision](/scope/14-success-criteria.md)). They are emitted on **every**
termination path, including failure and interruption — a run that died after spending
money is exactly the run whose cost needs recording.

Streaming assistant text is **not** echoed to stderr during the run. It would bury the
diagnostics in the reviewer's own prose, and it arrives on stdout at the end anyway.

No colour, no ANSI, no spinner, no progress bar: the caller is a script and the stream is
captured, not watched ([scope](/scope/12-non-goals.md)).

# Verdict

Accepted at triage as recommended. No envelope on stdout, no colour, and a `done` line carrying turns/cost/elapsed on every termination path including failure and interruption.

**Refreshed after [decision 09](/specs/09-read-only-tool-contract.md).** The `$` figure on
the `turn` and `done` lines is **notional** on the current credentials: `openai-codex` is
a subscription, and `ai.CalculateCost` multiplies catalog prices by tokens regardless. It
stays on the line — it is the volume proxy, it costs nothing to compute, and it becomes
real the day a tier names an API-key provider — but the line's load-bearing fields are now
**turns, tokens and elapsed time**. Nothing about the format changes; what a reader should
take from it does.
