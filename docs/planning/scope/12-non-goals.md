---
type: Decision
title: "Non-goals"
description: "What is explicitly out of v1?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 12
slug: non-goals
status: decided
verdict: "The full inherited list, each with its reasoning and any reopening condition"
decided_via: triage
depends_on: [mvp-feature-cut]
---

# Question

What is explicitly out, especially the tempting-but-deferred? Written non-goals are
what keep scope from creeping back in one issue at a time.

# Options

- **Full list, inherited from the concept** — carry every rejected design forward as a
  named non-goal, with the condition that would reopen it where one exists.
- **Short list** — name only the gateway. Everything else is "just not built yet",
  which reads to a future issue-writer as an omission rather than a decision.

# Recommendation

**Full list, inherited from the concept**, since the expensive thinking is already
done and a non-goal without its reasoning gets re-litigated. For v1:

- **A gateway or proxy in front of the session.** Rejected in
  [CONCEPT](/CONCEPT.md) with one named reopening condition: foreign models used as
  genuine *subagents* — implementers writing code, not reviewers reading it.
- **Any write to the repository or to GitHub.** No edits, no commits, no comments, no
  issue or PR state. The calling skill owns the report file.
- **Sharing the calling session's context.** No conversation, no history handoff, no
  memory between runs.
- **A model catalogue shipped inside the skills.** Facts about models live in
  `kern-link`; the assignment lives on the machine.
- **Task types other than reviewing a repository.** No implementing, no refactoring, no
  question-answering CLI.
- **Onboarding for users other than the author** — signed release binaries, install
  docs, config wizards. Follows from [target users](/scope/02-target-users.md).
- **A stale-catalogue signalling mechanism.** Parked by the concept, still parked.
- **Interactive UI of any kind.** No TUI, no prompts, no confirmations: the caller is a
  script.

# Verdict

**Accepted as recommended:** the full list above stands as the v1 non-goals, each
carrying its reasoning so it does not get re-litigated one issue at a time, and the
gateway carrying its single reopening condition (foreign models as genuine subagents
writing code, not reviewers reading it).
