---
type: Decision
title: "System prompt & task assembly"
description: "Who owns the reviewers instructions — the binary or the calling skill — and how the prompt reaches the binary."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 08
slug: system-prompt-and-task-assembly
status: decided
verdict: "The caller owns the system prompt; a JSON request object on stdin carries system + task, with --system/--prompt as by-hand shorthand"
decided_via: discussion
depends_on: [agent-loop-mechanics]
---

# Question

Something has to tell the foreign model that it is a reviewer, that it has four tools,
that it must not invent findings, and what to emit at the end. That text can live in the
binary or come from the caller, and the choice is hard to reverse: once
`review-interfaces.md` builds a prompt containing tool instructions, the binary can
never change its tool set without editing the skills repository
([scope](/scope/13-skills-integration-scope.md)).

It also bears directly on the project's one unenforceable assumption — that the caller
treats what comes back as **leads, not findings**, verified against the files before
anything enters a report ([risk 6](/scope/18-risks-and-assumptions.md)). The binary
cannot enforce that, but it can stop *encouraging* the opposite.

# Options

- **The binary owns the system prompt; the caller supplies only the task.** Tool
  instructions live next to the tools; the skills stay ignorant of a surface they do not
  control.
- **The caller supplies the whole prompt; the binary pipes it.** Maximum flexibility, and
  it couples the skills to the tool set and re-opens "the party under review decides what
  the reviewer is told" — a milder form of the failure the concept rejected pre-assembled
  evidence over.
- **Binary prompt with a caller-supplied `--system-append`.** Both, plus an extension
  point nobody has asked for and which becomes a compatibility surface the day it is
  used.

# Recommendation

**The binary owns the system prompt. The caller supplies the task and nothing else.**

The system prompt is a `const` in the binary (not a file, not a template, not
configurable), and it states:

1. The role: an external reviewer of a repository it did not write.
2. The tools available and that they are the only way to see the repository — plus that
   the repository has not been pre-summarised, so nothing has been chosen on its behalf.
3. **Leads, not findings.** Report what looks wrong and where to check it; do not assert
   a defect that has not been read in the file; say so when uncertain. Volume is not the
   goal, and a review that confirms is a legitimate outcome — scope explicitly refuses to
   make novelty a success criterion, precisely because that pressure makes a reviewer
   invent ([decision](/scope/14-success-criteria.md)).
4. Read-only: no edits, no commits, no GitHub. Not as a warning it could disobey — it
   physically cannot — but so it stops proposing to.
5. The output: a markdown report, as the final message, once no more reading is needed.

The task text is inserted verbatim as the first `ai.UserMessage`, with the repository's
absolute path stated alongside it. Nothing else is injected — no file listing, no README,
no git log. The reviewer chooses what to read; that choice is the reason the loop exists
at all ([concept](/CONCEPT.md)).

The prompt is asserted in tests for the presence of the leads-not-findings clause. It is
the one part of this codebase that carries a product guarantee, and a silent edit to it
would be invisible in every other test.

# Verdict

**Rejected.** The recommendation above — the binary owning a `const` system prompt — was
turned down, and the reasoning replaces it: an adjustment to how the reviewer is
instructed must be a change to a *skill*, not a new build of the binary. The skills are
where review judgment is authored and versioned; a prompt compiled into a Go binary
would put the fastest-moving text in the project behind the slowest release path.

**The caller owns the system prompt.** The binary supplies none, has no default, and
never substitutes one — an absent or empty system prompt is exit `2`, because a silent
fallback would be the binary owning the prompt again through the back door.

**Mechanism: a JSON request object on stdin.**

```json
{
  "system": "…who the reviewer is, what to look for, how to report…",
  "task":   "…what to review, this time…"
}
```

- Decoded with `DisallowUnknownFields`. A mistyped key is an error rather than a silently
  missing system prompt.
- Either field absent or empty ⇒ exit `2`.
- The object is extensible: something that needs passing later gets a field, not a flag.

`--system <text>` and `--prompt <text>` remain as the by-hand shorthand for journey 4
([scope](/scope/10-core-user-journeys.md)), mutually exclusive with stdin.

Two rejected mechanisms, for the record. **`--system-file <path>`** was recommended first
and fails on the case that matters: a system prompt generated in the calling turn — an
ad-hoc "use a foreign model for this" — has no file, and writing a temp file per
invocation to work around argument passing is machinery in place of an argument. It also
assumes the caller can locate its own prompt file, and skills live variously in a
project's `skills/` or under the user's `.claude/`. **`--system <text>` as the only
channel** fails on Windows argument quoting with a multi-kilobyte prompt.

**What the binary still owns: the tools.** `ai.Tool` names, descriptions and JSON schemas
are declared in Go beside the implementations
([decision](/specs/09-read-only-tool-contract.md)). `ai.Context.Tools` is a Go value, and
a description that drifts from its implementation is a reviewer told it may pass an
argument that does not exist. Tool descriptions are not overridable from the request.

**What moves to the caller with the prompt:** the leads-not-findings instruction, and the
test that asserted it. It stops being a guarantee the binary makes and becomes a property
of each caller's prompt — which is what SCOPE already says it is: *"an assumption about
the caller, and the one thing this project cannot enforce"*
([risk 6](/scope/18-risks-and-assumptions.md)).

**Note for implementers — do not add a task-type guard.** A caller-supplied system prompt
means the mechanism accepts tasks other than reviewing, while SCOPE's non-goals still say
v1 does not do task types other than reviewing a repository
([decision](/scope/12-non-goals.md)). That is not a contradiction to repair: the non-goal
governs what v1 *ships, documents, supports and tests*, not what the mechanism physically
permits. The flexibility is deliberate and deliberately unadvertised — no README section,
no help text, no examples. Nothing validates or classifies the prompt, and nothing should
be added that does. The real containment is elsewhere and unaffected: read-only tools,
repository confinement, no writes, no GitHub, no shared session context, one synchronous
run.
