---
type: Decision
title: "Read-only tool contract"
description: "The names, parameters and result shapes of the four tools the model sees."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 09
slug: read-only-tool-contract
status: decided
verdict: "Four tools; read_file takes a list of paths; search gains context lines; caps generous but truncation always announced"
decided_via: discussion
depends_on: [agent-loop-mechanics]
---

# Question

Scope decided the set: `list` (glob), `read_file` (with offset/limit), `search`
(ripgrep-style) and `git_read` over a fixed allowlist of `log`, `diff`, `show`, `status`,
with no general shell ([decision](/scope/07-read-only-tool-set.md)). What is open is
their exact JSON Schema and what their results look like — a contract with the model,
not with a programmer, and one every future prompt and every measurement is calibrated
against.

Result shape matters more than it looks. Tool output is the bulk of the tokens a run
spends: it is what pushes a large surface toward context overflow
([risk 3](/scope/18-risks-and-assumptions.md)), and it is most of the cost line
Milestone 2 is measuring.

# Options

- **Minimal parameters, plain-text results with a header line.** Cheap in tokens, reads
  naturally to a model, and any structure has to be parsed out of prose.
- **Rich parameters, JSON results.** Machine-precise, and it spends tokens on punctuation
  for a consumer that reads prose perfectly well.
- **Mirror an existing agent's tool schemas.** Familiar to the model from training, and
  it imports semantics this binary does not implement.

# Recommendation

**Four tools, minimal parameters, plain-text results, every result truncated with an
explicit marker.**

| Tool | Parameters | Result |
|---|---|---|
| `list` | `pattern` (glob, repo-relative; default `**/*`) | Matching paths, one per line, repo-relative with `/` separators. |
| `read_file` | `path`, `offset` (1-based line, default 1), `limit` (lines, default 2000) | The lines, each prefixed `<n>\t`, after a header naming the path and the range. |
| `search` | `pattern` (regex), `path` (subtree, default `.`), `glob` (file filter, optional), `max_results` (default 100) | `path:line:text` per match. |
| `git_read` | `command` (`log`/`diff`/`show`/`status`), `args` (array of strings) | The command's stdout verbatim. |

The details that are contract rather than implementation:

- **Paths are always repo-relative and always `/`-separated**, in both directions, on
  every platform. The model must never see `D:\Project\…`, and a path it emits must not
  depend on which OS the binary runs on ([scope constraint](/scope/15-constraints.md)).
  Conversion happens at the tool boundary, once.
- **`git_read.args` is an array, never a string.** A string implies a shell, and there is
  no shell ([decision](/specs/11-tool-implementation-strategy.md)). The array is passed
  as argv.
- **Truncation is always announced.** Every result that hit a limit ends with an explicit
  line — `[truncated: showing 100 of 431 matches]`, `[truncated: lines 1–2000 of 8104]`.
  The concept's objection to pre-assembled evidence is that it "truncates silently, and a
  report that saw sixty percent of the material reads exactly as confident as one that
  saw all of it". A tool that truncated silently would rebuild that flaw one call at a
  time. This does not resolve the parked question of how a *report* states its coverage
  ([SCOPE](/SCOPE.md)) — but it is the precondition for ever answering it, because the
  model cannot report a gap it was never told about.
- **Errors are messages, not failures.** A missing path, a bad regex, a disallowed git
  subcommand all return `IsError: true` with text saying what was wrong and what is
  allowed ([decision](/specs/07-agent-loop-mechanics.md)).
- Schemas are declared with `ai.JSONSchema(...)` literals, matching `kern-link`'s
  documented tool declaration; `ai.ToolCall.Arguments` arrives as a decoded
  `map[string]any`, so no JSON parsing happens on this side.

# Verdict

Recommended shape accepted, with two changes and one reframing.

**The set stays at four.** `list` for orientation, `read_file` for reading, `search` for
finding without knowing the layout, `git_read` for history. Nothing else earns a slot: a
`tree` tool is `list` with a broader pattern, and a `find_files` tool is `list` with a
narrower one. Four tools the model uses well beat six it chooses between badly.

**`read_file` takes a list of paths.** `paths: []string` rather than `path: string`, so
reading eight issue files is one turn instead of eight. Each file is returned under its
own header, and a failure on one path is reported inline beside the others rather than
failing the call — a batch where one path was a typo must still return the other seven.
This is the single largest reduction in round trips available, and it costs one loop.

**`search` gains `context`** (lines before and after each match, default 2). A match
without surrounding lines usually forces a follow-up `read_file` to judge it; two lines
of context usually does not. Same reasoning as the batch read: turns are the scarce
thing.

**Caps are reframed, not removed.** The original justification — token spend — does not
apply: this runs on a subscription, so tokens are not billed by the token. What remains
binding is the **context window**, which is a hard limit no billing model changes, and
[scope risk 3](/scope/18-risks-and-assumptions.md) (context overflow on large surfaces)
is unmitigated in v1. So the caps stay and get generous defaults — `read_file` 5000 lines
per file, `search` 200 matches — and **truncation is still always announced**. The
announcement was never about cost. It exists because the concept's objection to
pre-assembled evidence is that it *"truncates silently, and a report that saw sixty
percent of the material reads exactly as confident as one that saw all of it"* — a tool
that truncated silently would rebuild that flaw one call at a time.

**Tool descriptions carry more weight than originally planned.** With the caller owning
the system prompt ([decision 08](/specs/08-system-prompt-and-task-assembly.md)), the
`ai.Tool.Description` strings may be the model's *only* instructions on how to read this
repository. They are written to stand alone: what the tool does, what its paths look
like, what truncation means, and that a batch failure is partial rather than total.

**Refreshed after [decision 10](/specs/10-repository-confinement.md).** All four tools are
scoped by the run's `--allow` subtrees, not only by the root: `list` and `search`
enumerate within them, `read_file` refuses paths outside them (per path, in a batch —
the other paths still return), and `git_read` is pathspec-restricted with `show`'s
`<rev>:<path>` form validated before the command is built. A refusal names the rule that
refused, so "outside the allow-list" and "denied filename" read differently to the model.
