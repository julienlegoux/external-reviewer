---
type: Decision
title: "Agent loop mechanics"
description: "How the multi-turn loop drives kern-link: streaming or not, how messages accumulate, when it stops, and where bounds plug in."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 07
slug: agent-loop-mechanics
status: decided
verdict: "StreamSimple + Result per turn; bounds.check as a seam with values unset; failing tool is a turn, failing turn ends the run"
decided_via: discussion
depends_on: [kern-link-dependency-policy, tier-assignment-schema]
---

# Question

The loop is the product — it is why this is a binary rather than a prompt-pipe, and why
the reviewer chooses what to read instead of being handed evidence the reviewed party
selected ([concept](/CONCEPT.md)).

These mechanics are the shape of the finished v1, not of one milestone. Scope phases
*when* each part lands — a single round trip first, the loop second, bounds third — but
a spec that describes only the middle phase leaves the last one to be improvised. Two
things in particular are expensive to change once the tools and the prompt are built on
them: whether a turn streams or blocks, and what the loop does when something goes wrong
inside it.

# Options

- **`models.StreamSimple` + `Stream.Result(ctx)` per turn.** Events arrive live, so
  progress and cost are visible while a turn is still running; `Result` still yields the
  same `*ai.AssistantMessage` a blocking call would. Costs an event loop per turn.
- **`models.CompleteSimple` per turn.** The exact shape `kern-link`'s `docs/usage.md`
  documents — least code, and nothing is observable until a turn ends.
- **`models.Stream` (non-simple).** Full per-provider knobs, at the price of resolving
  thinking levels and options per provider by hand, which is what `StreamSimple` exists
  to do.

# Recommendation

**`StreamSimple` + `Result(ctx)` per turn**, over `[]ai.Message` accumulated exactly as
`kern-link` documents:

```
resolve tier → provider+model; Refresh(ctx, provider) if dynamic   // decision 05
messages := []ai.Message{&ai.UserMessage{…task…}}
for {
    if err := bounds.check(state); err != nil { break }
    chat := ai.Context{SystemPrompt: sys, Messages: messages, Tools: tools}
    stream := models.StreamSimple(ctx, model, chat, opts)
    for ev := range stream.Events(ctx) { … live diagnostics … }
    msg, err := stream.Result(ctx)
    messages = append(messages, msg)
    state.add(ai.CalculateCost(model, &msg.Usage))
    if msg.StopReason != ai.StopReasonToolUse { break }
    for each ai.ToolCall in msg.Content:
        messages = append(messages, &ai.ToolResultMessage{…})
}
```

Streaming wins on one argument: the run is observable while it happens. That matters for
a hand-run the operator can interrupt, for an unsupervised run whose stderr is captured,
and for the measurements that size the bounds — all three, not one phase of the project.

**Bounds are one seam, present from the first commit.** `bounds.check(state)` is
consulted at the top of every turn against turns taken, accumulated cost and elapsed
wall clock. Its *values* start unset — unbounded, which is what
[scope decided](/scope/08-loop-bounds-and-termination.md) for the phase before anything
has been measured, on the reasoning that a cap guessed early silently truncates
legitimate reviews. Setting real numbers later is then a change to data and to
[the CLI](/specs/02-cli-surface-and-argv.md), never a change to the loop's structure.
An exceeded bound is not a crash: the loop stops, the accumulated report is returned,
and stderr says which bound stopped it.

**Termination**, in the order the loop checks it:

1. A bound is exceeded → stop, report what exists.
2. `ctx` cancelled (`SIGINT`) → stop. This must work: while the loop is unbounded, a
   human at the keyboard is the *only* thing standing between a runaway run and real
   money ([scope risk 5](/scope/18-risks-and-assumptions.md)).
3. `StopReason != StopReasonToolUse` → the model is done. This is the normal exit.

**Failure handling, at two different levels — the distinction is the point:**

- **A failing tool is a turn, not an exit.** Every tool error returns as a
  `ToolResultMessage` with `IsError: true` and text saying what went wrong. A reviewer
  that globs a path which does not exist should try another path, not kill the review.
- **A failing *turn* ends the run.** `Result` returning an error, or a `StopReason` of
  `error`/`aborted`, stops the loop and classifies as exit `2`
  ([decision](/specs/13-error-handling-and-failure-classification.md)).

**Cost is accumulated per turn**, not estimated at the end:
`ai.CalculateCost(model, &msg.Usage)` after every `Result`. The running total is what
stderr reports live, what the `done` line carries on every termination path
([decision](/specs/12-output-and-diagnostics-format.md)), what bounds are checked
against, and what success criterion 3 is judged on
([scope](/scope/14-success-criteria.md)).

**Reasoning is left at the provider default** (`SimpleStreamOptions{}` with no
`Reasoning`). Thinking level is a plausible per-tier knob and the config schema has room
for one ([decision](/specs/05-tier-assignment-schema.md)); nothing measured yet justifies
choosing a value, and `StreamSimple` already clamps whatever is set to what the model
supports.

# Verdict

**Refreshed after [decision 09](/specs/09-read-only-tool-contract.md), before triage.**
The provider is a subscription, so of the three quantities `bounds.check(state)` watches,
**cost is no longer a meaningful bound** — turns and wall clock are. The seam keeps all
three (a tier assigned an API-key provider makes cost real again, and the check is one
comparison), but when Milestone 3 sets numbers, the ones that matter are a turn count and
a deadline. The dominant *hard* limit inside a run is the model's context window, which
`bounds` cannot see and which result caps address instead.

**Accepted as rewritten.** `StreamSimple` + `Result(ctx)` per turn; messages accumulated
as `[]ai.Message` exactly as `kern-link` documents; termination checked in the order
bound → cancellation → `StopReason`; a failing tool is a turn and a failing turn ends the
run; cost accumulated per turn; reasoning left at the provider default.

The two amendments above stand: bounds are a seam present from the first commit with
their values unset, so Milestone 3 sets numbers rather than restructuring the loop; and
of the three quantities that seam watches, turns and wall clock are the meaningful ones
on a subscription, while the binding limit inside a run is the context window, addressed
by result caps in [decision 09](/specs/09-read-only-tool-contract.md) rather than by
bounds.
