---
type: Decision
title: "The read-only tool set"
description: "Which tools does the internal agent loop expose to the reviewing model?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 07
slug: read-only-tool-set
status: decided
verdict: "Four native read-only tools — list, read_file, search, git_read — confined to the repo root"
decided_via: triage
depends_on: [target-users]
---

# Question

The loop lives inside the binary precisely so the reviewer can choose what to read
([CONCEPT](/CONCEPT.md) § "What it reads, and what it never writes"). Which tools does
it get, and how is "never writes" actually guaranteed rather than merely intended?

# Options

- **Four native read-only tools** — `list` (glob), `read_file` (with offset/limit),
  `search` (ripgrep-style), `git_read` (a fixed allowlist: `log`, `diff`, `show`,
  `status`). No general shell. Every tool is implemented in-process, so there is no
  path to a write.
- **A sandboxed shell** — expose one `run` tool restricted by a command allowlist.
  Fewer tools to build, far more surface to audit, and an allowlist on a shell string
  is a parsing problem that never fully closes.
- **Pre-assembled evidence, no tools** — the caller pastes the material in. Explicitly
  rejected by the concept: the party under review would choose what the reviewer sees.

# Recommendation

**Four native read-only tools.** They cover how a reviewer actually reads a repository
— locate, read, grep, and inspect history/diff — and each is a function call rather
than a command line, so "never writes, never commits, never touches GitHub state" is a
property of the code that exists rather than of a filter on a string. `kern-link`
already carries the tool-call protocol as first-class types, so the cost here is the
four implementations, not the plumbing.

`git_read` earns its place separately from `read_file`: a review of an epic's merged
diff is a history question, and without it the model reconstructs diffs by reading both
sides of every file, burning context on work git does instantly.

Constrain the loop to the repository root passed on the command line — path traversal
outside it is refused — so a bad tool call cannot read the rest of the machine.

# Verdict

**Accepted as recommended.** Four native, in-process read-only tools: `list` (glob),
`read_file` (with offset/limit), `search` (ripgrep-style), and `git_read` (a fixed
allowlist: `log`, `diff`, `show`, `status`). No general shell. Every tool call is
confined to the repository root passed on the command line; traversal outside it is
refused.

Because each tool is a function rather than a command line, "never writes, never
commits, never touches GitHub state" is a property of the code that exists — not of a
filter applied to a string. `git_read` earns its own slot: an epic's merged diff is a
history question, and without it the model burns context reconstructing diffs by
reading both sides of every file.
