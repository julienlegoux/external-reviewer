---
type: Decision
title: "Go error idiom"
description: "How errors are constructed, wrapped and inspected in a program whose exit code is a classification of them."
tags: [decision, conventions]
timestamp: 2026-08-09T04:08:38Z
phase: conventions
decision: 06
slug: go-error-idiom
status: decided
verdict: "A — wrap with %w for humans, typed/sentinel errors for classification; a tool error is a value"
decided_via: triage
depends_on: []
---

# Question

The baseline's error rules are language-neutral: fail fast, never swallow, add context,
and write user-facing and log-facing messages as different audiences. Go needs the
mechanics spelled out, and this program needs them more than most: its exit code *is* a
classification of its errors — `1` for "no reviewer was ever reachable", `2` for
"reached and then failed" — and the sharpest edge in the whole design is that an
*absent* credential is `1` while a *broken* one is `2`
([specs 13](/specs/13-error-handling-and-failure-classification.md)). Getting that right
in `run()` means the error has to still know what it was by the time it arrives there.

A second constraint: a failing *tool* is not a failing *turn*. A tool error becomes
`ToolResultMessage{IsError: true}` and the loop continues; only a failing turn ends the
run. So one of the two error paths must never propagate.

# Options

- **A — Wrap for humans, type for classification.** Context is added with
  `fmt.Errorf("reading %s: %w", p, err)`; classification travels on a small set of
  typed/sentinel errors inspected with `errors.Is`/`errors.As` at the `run()` boundary —
  never by matching message strings. Messages are lowercase, unpunctuated, and state
  what was attempted. Tool errors are a *value* returned to the loop, not an `error`
  returned up the stack. Nothing panics across the CLI boundary; `recover()` appears
  nowhere.
- **B — Sentinel errors only.** `errors.Is` against a flat list, no custom types. Fine
  until an error needs to carry a field (which rule refused a path, which provider
  failed auth) and the information goes into the message string, where the exit-code
  logic then has to parse it back out.
- **C — Return exit codes internally.** Functions return `(result, code int)`. Direct,
  and it makes every intermediate function a participant in the CLI contract while
  losing `%w` chains entirely.

# Recommendation

**A.** It is what SPECS already half-specifies — it names `*ai.ModelsError` with code
`oauth`/`auth` as the discriminator for the `1`-vs-`2` credential edge, which is
`errors.As` in everything but name. Stating the tool-error-is-a-value rule alongside it
is the part worth writing down: it is the one place where "never swallow an error"
inverts, and an implementer following the baseline literally would propagate a bad path
out of the loop and end a review that should have continued.

# Verdict

**A — wrap for humans, type for classification.** Accepted at triage as recommended.
Context is added with `fmt.Errorf("<what was attempted>: %w", err)`; exit-code
classification travels on typed or sentinel errors inspected with `errors.Is`/`errors.As`
at the `run()` boundary and never by matching message text. Messages are lowercase and
unpunctuated. A failing *tool* is a value returned into the loop as
`ToolResultMessage{IsError: true}`, not an `error` propagated up the stack — the single
place where the baseline's "never swallow" rule inverts, and the one an implementer
following the baseline literally would get wrong. Nothing panics across the CLI boundary
and `recover()` appears nowhere.

**Promotion candidate**: the first half (wrap with `%w`, classify with `errors.As`, no
string matching, no panics across a process boundary) is generic Go and belongs in
`assets/baseline.md` as a *(per-stack)* Go entry under *Error handling*. The
tool-error-is-a-value rule is specific to this project's loop and stays here.
