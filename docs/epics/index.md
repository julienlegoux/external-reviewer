---
okf_version: "0.1"
---

# Epics

Epics for External Reviewer, cut from [SCOPE](../planning/SCOPE.md)'s three milestones
and kept in its order — each depends on the one before it. Epic 0 is a slot, not a fixed
identity: `triage-reports` reuses it for whatever remediation lane the latest review
findings need, and each occupant is retired once its issues are done, freeing the slot for
the next cycle. No epic 0 is currently open.

* [Epic 1: Walking skeleton](/epic-1-walking-skeleton/EPIC_1.md) - done, [#1](https://github.com/julienlegoux/external-reviewer/issues/1), milestone closed
* [Epic 2: Read-only agentic loop](/epic-2-read-only-agentic-loop/EPIC_2.md) - done, [#2](https://github.com/julienlegoux/external-reviewer/issues/2), milestone closed
* [Epic 3: Reviewer selection and pipeline integration](/epic-3-reviewer-selection-integration/EPIC_3.md) - open, [#3](https://github.com/julienlegoux/external-reviewer/issues/3) — spans two repositories

Reference documents produced by an epic, rather than epics themselves:

* [Epic 2 — Hand-run measurements](/epic-2-read-only-agentic-loop/MEASUREMENTS.md) - what four hand runs on `openai-codex/gpt-5.5` (three completed) cost in turns, tokens, context and wall clock; Epic 3's input for the loop caps
